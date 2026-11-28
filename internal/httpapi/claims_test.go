package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// liveContract is a Main Works award with two contract lines, written
// through the store and, for the award itself, directly.
func liveContract(t *testing.T, m *member, project string) (string, []string, string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var user string
	if err := conn.QueryRow(ctx, `SELECT id::text FROM users WHERE org_id=$1 LIMIT 1`, m.orgID).Scan(&user); err != nil {
		t.Fatal(err)
	}
	s := m.app.store
	row, _ := procurement.DisciplinePackage("trade.main_works", "Main Works")
	pkg, err := s.CreateDisciplinePackage(ctx, m.orgID, project, user, row)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, m.orgID, project, "")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for i, a := range []string{"100000.00", "50000.00"} {
		before := map[string]bool{}
		for _, it := range plan.Items {
			before[it.ID] = true
		}
		plan, err = s.CreateCostItem(ctx, m.orgID, project, user, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: []string{"Substructure", "Superstructure"}[i], LineKind: "works", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}})
		if err != nil {
			t.Fatal(err)
		}
		var id string
		for _, it := range plan.Items {
			if !before[it.ID] {
				id = it.ID
			}
		}
		amount := costs.Money(a)
		if plan, err = s.PutCostValue(ctx, m.orgID, project, id, user, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "allowance"}}); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, id)
	}
	sub, award := newUUID(t), newUUID(t)
	if _, err := conn.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,$2,$3,$4,'tender.md','awarded','2026-11-01')`, m.orgID, sub, project, pkg.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,$2,$3,$4,$5,$6,false,$7)`, m.orgID, award, project, pkg.ID, sub, lines[0], user); err != nil {
		t.Fatal(err)
	}
	for i, l := range lines {
		if _, err := conn.Exec(ctx, `INSERT INTO cost_line_sources(org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal) VALUES($1,$2,$3,$4,$5,$6,'',$7)`, m.orgID, newUUID(t), project, award, l, "line:"+l[:8], i); err != nil {
			t.Fatal(err)
		}
	}
	return pkg.ID, lines, user
}

type tabLedger struct {
	Contracts []struct {
		PackageID string  `json:"package_id"`
		SumToDate *string `json:"contract_sum_to_date"`
		Certified *string `json:"certified_to_date"`
	} `json:"contracts"`
	Variations []struct {
		ID      string `json:"id"`
		Number  string `json:"number"`
		Status  string `json:"status"`
		Version int64  `json:"version"`
	} `json:"variations"`
	Claims []struct {
		InvoiceID  string `json:"invoice_id"`
		Unassessed int    `json:"unassessed"`
		Payable    *string
		Alert      string `json:"alert"`
		Rows       []struct {
			ID   string `json:"id"`
			Line string `json:"line"`
		} `json:"rows"`
		ScheduleDue struct {
			Words   string `json:"words"`
			ToCheck bool   `json:"to_check"`
		} `json:"schedule_due"`
		Issued []struct {
			File string `json:"file"`
		} `json:"issued"`
	} `json:"claims"`
	Invoices []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"invoices"`
	Summary struct {
		Invoiced *string `json:"invoiced"`
	} `json:"summary"`
}

func jsonBody(v any) []byte { b, _ := json.Marshal(v); return b }

// The routes: scoping to the org and project, missing rows, unexpected
// input, and done means 1 and 3 through the API.
func TestVariationsAndClaimsRoutes(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, foreign := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Claims routes")
	other := foreign.createProject(t, "Foreign claims")
	pkg, lines, _ := liveContract(t, m, project)
	base := "/api/projects/" + project

	// Variations: add, refuse, approve, undo.
	add := jsonBody(map[string]any{"package": pkg, "line": lines[0], "title": "Rock excavation", "claimed": "12400", "time_claimed_days": "3"})
	if got := foreign.status(t, http.MethodPost, "/api/projects/"+project+"/variations", add); got != http.StatusNotFound {
		t.Fatalf("foreign org add = %d", got)
	}
	if got := m.status(t, http.MethodPost, "/api/projects/"+other+"/variations", add); got != http.StatusNotFound {
		t.Fatalf("foreign project add = %d", got)
	}
	if got := m.status(t, http.MethodPost, base+"/variations", []byte(`{"package":"x"}`)); got != http.StatusBadRequest {
		t.Fatalf("bad package = %d", got)
	}
	var v tabLedger
	m.call(t, http.MethodPost, base+"/variations", add, http.StatusOK, &v)
	if len(v.Variations) != 1 || v.Variations[0].Number != "V01" {
		t.Fatalf("variations %+v", v.Variations)
	}
	vid := v.Variations[0].ID
	if got := m.status(t, http.MethodPost, base+"/variations", jsonBody(map[string]any{"package": pkg, "line": lines[0], "title": "x", "number": "V01"})); got != http.StatusConflict {
		t.Fatalf("duplicate number = %d", got)
	}
	if got := m.status(t, http.MethodPatch, base+"/variations/"+vid, []byte(`{"status":"approved"}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("approve without amount = %d", got)
	}
	if got := foreign.status(t, http.MethodPatch, base+"/variations/"+vid, []byte(`{"title":"x"}`)); got != http.StatusNotFound {
		t.Fatalf("foreign edit = %d", got)
	}
	if got := m.status(t, http.MethodPatch, base+"/variations/"+newUUID(t), []byte(`{"title":"x"}`)); got != http.StatusNotFound {
		t.Fatalf("missing variation = %d", got)
	}
	reply := m.ops(t, project, "variations", http.StatusOK, op("variations.decide", map[string]any{"variation": "V01", "decision": "approve", "approved": "11000"}))
	m.getJSON(t, base+"/invoices", &v)
	if v.Contracts[0].SumToDate == nil || *v.Contracts[0].SumToDate != "161000.00" {
		t.Fatalf("contract sum to date %v", v.Contracts[0].SumToDate)
	}
	m.undo(t, project, reply.ChangeID, http.StatusOK)
	m.getJSON(t, base+"/invoices", &v)
	if *v.Contracts[0].SumToDate != "150000.00" || v.Variations[0].Status != "submitted" {
		t.Fatalf("after undo: %v %s", *v.Contracts[0].SumToDate, v.Variations[0].Status)
	}

	// A claim dropped on its contract is read and its rows mapped by rule.
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.status(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+newUUID(t), raw); got != http.StatusNotFound {
		t.Fatalf("a claim on no live contract = %d", got)
	}
	if got := m.status(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=invoice&package="+pkg, raw); got != http.StatusBadRequest {
		t.Fatalf("an unknown kind = %d", got)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	m.getJSON(t, base+"/invoices", &v)
	if len(v.Claims) != 1 || len(v.Claims[0].Rows) != 3 || v.Claims[0].Rows[0].Line != "Substructure" || v.Claims[0].Unassessed != 3 {
		t.Fatalf("claim %+v", v.Claims)
	}
	if !v.Claims[0].ScheduleDue.ToCheck {
		t.Fatalf("a date from no state is to check: %+v", v.Claims[0].ScheduleDue)
	}
	claimPath := base + "/invoices/" + created.ID
	if got := foreign.status(t, http.MethodPut, claimPath+"/claim-lines", []byte(`{"accept_claimed":true}`)); got != http.StatusNotFound {
		t.Fatalf("foreign assess = %d", got)
	}
	if got := m.status(t, http.MethodPut, base+"/invoices/"+newUUID(t)+"/claim-lines", []byte(`{"accept_claimed":true}`)); got != http.StatusNotFound {
		t.Fatalf("missing claim = %d", got)
	}
	if got := m.status(t, http.MethodPut, claimPath+"/claim-lines", []byte(`{"rows":[{"id":"nope"}]}`)); got != http.StatusBadRequest {
		t.Fatalf("bad row = %d", got)
	}
	if got := m.status(t, http.MethodPost, claimPath+"/schedule", nil); got != http.StatusUnprocessableEntity {
		t.Fatalf("issue before assessing = %d", got)
	}
	// Accept claimed from the screen: one Undo.
	reply = m.ops(t, project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	if reply.ChangeID == "" {
		t.Fatal("accept claimed has no undo")
	}
	if got := foreign.status(t, http.MethodPost, claimPath+"/schedule", nil); got != http.StatusNotFound {
		t.Fatalf("foreign issue = %d", got)
	}
	var issued struct {
		ReportID  string `json:"report_id"`
		VersionID string `json:"version_id"`
	}
	m.call(t, http.MethodPost, claimPath+"/schedule", nil, http.StatusCreated, &issued)
	m.getJSON(t, base+"/invoices", &v)
	if len(v.Claims[0].Issued) != 1 || v.Summary.Invoiced == nil || *v.Summary.Invoiced != "86000.00" {
		t.Fatalf("issued: %+v invoiced %v", v.Claims[0].Issued, v.Summary.Invoiced)
	}
	if got := m.status(t, http.MethodGet, v.Claims[0].Issued[0].File, nil); got != http.StatusOK {
		t.Fatalf("schedule PDF = %d", got)
	}
	if got := foreign.status(t, http.MethodGet, v.Claims[0].Issued[0].File, nil); got != http.StatusNotFound {
		t.Fatalf("foreign PDF = %d", got)
	}
	if got := m.status(t, http.MethodPut, claimPath+"/claim-lines", []byte(`{"accept_claimed":true}`)); got != http.StatusConflict {
		t.Fatalf("assess after issue = %d", got)
	}
	// The contract's periods.
	periods := base + "/procurement/packages/" + pkg + "/award"
	if got := m.status(t, http.MethodPatch, periods, []byte(`{"schedule_days":5,"schedule_basis":"fortnights"}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("bad basis = %d", got)
	}
	if got := foreign.status(t, http.MethodPatch, periods, []byte(`{"schedule_days":5}`)); got != http.StatusNotFound {
		t.Fatalf("foreign periods = %d", got)
	}
	m.call(t, http.MethodPatch, periods, []byte(`{"schedule_days":5,"schedule_basis":"business"}`), http.StatusOK, nil)
}

func p50p90(d []time.Duration) (time.Duration, time.Duration) {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)/2], d[len(d)*9/10]
}

// TestVariationsBudget measures the tab's read, a variation write, an
// assessment and an issue against bench/budgets.json (100 / 250 ms each),
// with 200 variations, 24 claims of 40 rows and 300 invoices.
func TestVariationsBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("budget fixture")
	}
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Claims budget")
	pkg, lines, user := liveContract(t, m, project)
	ctx := context.Background()
	s := a.store
	now := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)
	for i := 0; i < 200; i++ {
		if _, err := s.AddVariation(ctx, m.orgID, project, user, store.VariationInput{Package: pkg, Line: &lines[i%2], Title: strp(fmt.Sprintf("Variation %d", i)), Claimed: strp("1000")}, now); err != nil {
			t.Fatal(err)
		}
	}
	upload := func(name string, claim bool) string {
		sum := []byte(fmt.Sprintf("%032s", name))
		up := store.InvoiceUpload{SHA256: sum[:32], Size: int64(len(name)), MediaType: "application/pdf", Filename: name + ".pdf", UserID: user}
		if claim {
			up.ClaimPackage = pkg
		}
		c, err := s.CreateInvoice(ctx, m.orgID, project, up, now)
		if err != nil {
			t.Fatal(err)
		}
		d := func(f, v string) invoices.Decision {
			return invoices.Decision{Field: f, Value: v, Band: "green", DecidedBy: "rule"}
		}
		out := invoices.Outcome{Asked: true, Reading: invoices.Reading{Decisions: []invoices.Decision{d(invoices.FieldNumber, name), d(invoices.FieldDate, "2026-10-14"),
			d(invoices.FieldTotal, "110000"), d(invoices.FieldGST, "10000"), d(invoices.FieldPeriod, "2026-10-01"), d(invoices.FieldCurrency, "AUD")}}}
		if claim {
			var rows []invoices.ClaimRow
			for r := 0; r < 40; r++ {
				to, prev, this := int64(100000*(r+1)), int64(0), int64(100000*(r+1))
				rows = append(rows, invoices.ClaimRow{Kind: invoices.RowContract, Label: fmt.Sprintf("Row %d", r), LineID: lines[r%2], MappedBy: "rule", Band: "green", ToDate: &to, Previous: &prev, ThisClaim: &this})
			}
			out.Claim = &invoices.ClaimReading{TableFound: true, Rows: rows}
		} else {
			out.Decisions = append(out.Decisions, d(invoices.FieldKind, "invoice"), d(invoices.FieldSupplier, pkg), d(invoices.FieldLine, lines[0]))
		}
		if _, err := s.CommitInvoiceRead(ctx, m.orgID, c.ID, out, false, now); err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	var claims []string
	for i := 0; i < 24; i++ {
		claims = append(claims, upload("claim-"+strconv.Itoa(i), true))
	}
	for i := 0; i < 300; i++ {
		upload("inv-"+strconv.Itoa(i), false)
	}
	base := "/api/projects/" + project
	var reads, writes, assesses, issues, sheets []time.Duration
	var v tabLedger
	m.getJSON(t, base+"/invoices", &v)
	if len(v.Claims) != 24 || len(v.Variations) != 200 || len(v.Invoices) != 324 {
		t.Fatalf("fixture: %d claims %d variations %d invoices", len(v.Claims), len(v.Variations), len(v.Invoices))
	}
	for i := 0; i < 20; i++ {
		began := time.Now()
		m.getJSON(t, base+"/invoices", &v)
		reads = append(reads, time.Since(began))
	}
	for i := 0; i < 20; i++ {
		began := time.Now()
		m.call(t, http.MethodPatch, base+"/variations/"+v.Variations[i].ID, jsonBody(map[string]any{"estimate": strconv.Itoa(900 + i)}), http.StatusOK, nil)
		writes = append(writes, time.Since(began))
	}
	for i := 0; i < 20; i++ {
		began := time.Now()
		m.call(t, http.MethodPut, base+"/invoices/"+claims[i]+"/claim-lines", []byte(`{"accept_claimed":true}`), http.StatusOK, nil)
		assesses = append(assesses, time.Since(began))
	}
	for i := 0; i < 20; i++ {
		began := time.Now()
		m.call(t, http.MethodPost, base+"/invoices/"+claims[i%4]+"/schedule", nil, http.StatusCreated, nil)
		issues = append(issues, time.Since(began))
	}
	// One claim's sheet, with four schedules issued before it.
	for i := 0; i < 20; i++ {
		began := time.Now()
		m.getJSON(t, base+"/invoices?sheet="+claims[4+i%16], &v)
		sheets = append(sheets, time.Since(began))
	}
	// One line of the breakdown added, then edited.
	var breakdowns []time.Duration
	var bl struct {
		Contracts []struct {
			Breakdown []struct {
				ID string `json:"id"`
			} `json:"breakdown"`
		} `json:"contracts"`
	}
	for i := 0; i < 10; i++ {
		began := time.Now()
		m.call(t, http.MethodPost, base+"/contracts/"+pkg+"/breakdown", jsonBody(map[string]any{"label": "Line " + strconv.Itoa(i), "amount": "100"}), http.StatusOK, &bl)
		breakdowns = append(breakdowns, time.Since(began))
		last := bl.Contracts[0].Breakdown[len(bl.Contracts[0].Breakdown)-1].ID
		began = time.Now()
		m.call(t, http.MethodPatch, base+"/contracts/"+pkg+"/breakdown/"+last, []byte(`{"note":"measured"}`), http.StatusOK, nil)
		breakdowns = append(breakdowns, time.Since(began))
	}
	for name, d := range map[string][]time.Duration{"variations_read": reads, "variation_write": writes, "claim_assess": assesses, "payment_schedule_issue": issues, "schedule_read": sheets, "breakdown_write": breakdowns} {
		p50, p90 := p50p90(d)
		t.Logf("%s p50=%s p90=%s", name, p50, p90)
		if p50 > 100*time.Millisecond || p90 > 250*time.Millisecond {
			t.Errorf("%s over budget: p50=%s p90=%s", name, p50, p90)
		}
	}
}

func strp(s string) *string { return &s }

// TestClaimIntakeBudget: a claim dropped on its contract, upload to its
// ledger row, with the test Jev (claim_intake 1,000 / 2,000 ms).
func TestClaimIntakeBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Claim intake")
	pkg, _, _ := liveContract(t, m, project)
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var times []time.Duration
	for i := 0; i < 20; i++ {
		body := append(append([]byte{}, raw...), []byte(fmt.Sprintf("\nCopy %d\n", i))...)
		began := time.Now()
		m.call(t, http.MethodPost, "/api/projects/"+project+"/invoices?name=pc"+strconv.Itoa(i)+".md&kind=progress_claim&package="+pkg, body, http.StatusCreated, nil)
		m.waitInvoices(t, project, i+1)
		times = append(times, time.Since(began))
	}
	p50, p90 := p50p90(times)
	t.Logf("claim_intake p50=%s p90=%s (test Jev)", p50, p90)
	if p50 > time.Second || p90 > 2*time.Second {
		t.Fatal("claim intake budget exceeded")
	}
}
