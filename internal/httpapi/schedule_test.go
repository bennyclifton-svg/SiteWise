package httpapi_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

type sheetLedger struct {
	Contracts []struct {
		SumToDate *string `json:"contract_sum_to_date"`
		Adjusted  *string `json:"adjusted"`
		Certified *string `json:"certified_to_date"`
		Retention *string `json:"retention_held"`
		Paid      *string `json:"paid_to_date"`
		Latest    *struct {
			ClaimID   string `json:"claim_id"`
			VersionID string `json:"version_id"`
			Inc       string `json:"inc_gst"`
		} `json:"latest_schedule"`
	} `json:"contracts"`
	Claims []struct {
		DisplayNumber string  `json:"display_number"`
		Period        string  `json:"period"`
		Claimed       *string `json:"claimed"`
	} `json:"claims"`
	Sheet *struct {
		ClaimID  string `json:"claim_id"`
		Sections []struct {
			Kind      string  `json:"kind"`
			TotalLet  *string `json:"total_let"`
			Completed *string `json:"completed"`
			Lines     []struct {
				Label         string  `json:"label"`
				RowID         string  `json:"row_id"`
				TotalLet      *string `json:"total_let"`
				Completed     *string `json:"completed"`
				Percent       string  `json:"percent"`
				Prior         string  `json:"prior"`
				ThisClaim     *string `json:"this_claim"`
				ClaimedToDate *string `json:"claimed_to_date"`
			} `json:"lines"`
		} `json:"sections"`
		ThisClaim *string `json:"this_claim"`
		Retention struct {
			Held       *string `json:"held"`
			HeldToDate *string `json:"held_to_date"`
		} `json:"retention"`
		Net      *string `json:"net"`
		Payable  *string `json:"payable"`
		Payments []struct {
			Number  string  `json:"number"`
			Period  string  `json:"period"`
			Ex      *string `json:"ex"`
			Current bool    `json:"current"`
		} `json:"payments"`
		Paid struct {
			Inc *string `json:"inc"`
		} `json:"paid"`
		PaidInFull bool `json:"paid_in_full"`
	} `json:"sheet"`
}

func is(p *string, want string) bool { return p != nil && *p == want }

// A claim's payment schedule as a sheet rides on the ledger read
// (docs/plans/2026-11-26-payment-schedules.md, slice 1): the contract's
// lines with what is complete, what was certified before and what this
// claim pays, and one summary per contract. Scoped to the org and project.
func TestClaimSheetRidesOnTheLedger(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, foreign := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Claim sheet")
	pkg, _, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)

	var v sheetLedger
	m.getJSON(t, base+"/invoices", &v)
	if v.Sheet != nil {
		t.Fatal("a read that asks for no sheet carries none")
	}
	sheet := base + "/invoices?sheet=" + created.ID
	if got := m.status(t, http.MethodGet, base+"/invoices?sheet=nope", nil); got != http.StatusBadRequest {
		t.Fatalf("a sheet id that is not an id = %d", got)
	}
	if got := m.status(t, http.MethodGet, base+"/invoices?sheet="+newUUID(t), nil); got != http.StatusNotFound {
		t.Fatalf("a sheet for no claim = %d", got)
	}
	if got := foreign.status(t, http.MethodGet, sheet, nil); got != http.StatusNotFound {
		t.Fatalf("foreign sheet = %d", got)
	}

	// Before it is assessed: the lines and what is claimed, nothing completed.
	m.getJSON(t, sheet, &v)
	if v.Sheet == nil || v.Sheet.ClaimID != created.ID || len(v.Sheet.Sections) != 1 || v.Sheet.Sections[0].Kind != "works" || len(v.Sheet.Sections[0].Lines) != 2 {
		t.Fatalf("sheet %+v", v.Sheet)
	}
	works := v.Sheet.Sections[0]
	if l := works.Lines[0]; l.Label != "Substructure" || !is(l.TotalLet, "100000.00") || l.Completed != nil || l.Percent != "" || !is(l.ClaimedToDate, "60000.00") || l.RowID == "" {
		t.Fatalf("an unassessed line: %+v", l)
	}
	if !is(works.TotalLet, "150000.00") || works.Completed != nil || v.Sheet.ThisClaim != nil || v.Sheet.PaidInFull {
		t.Fatalf("unassessed totals: %+v", works)
	}

	// The period is the claim's own, and the PM's to correct.
	m.call(t, http.MethodPatch, base+"/invoices/"+created.ID, []byte(`{"fields":{"period":"2026-09"}}`), http.StatusOK, nil)
	// Accept claimed: every figure is the ledger's.
	m.ops(t, project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	m.getJSON(t, sheet, &v)
	works = v.Sheet.Sections[0]
	if l := works.Lines[0]; !is(l.Completed, "60000.00") || l.Percent != "60.0" || l.Prior != "0.00" || !is(l.ThisClaim, "60000.00") {
		t.Fatalf("substructure: %+v", l)
	}
	if l := works.Lines[1]; !is(l.Completed, "26000.00") || l.Percent != "52.0" {
		t.Fatalf("superstructure: %+v", l)
	}
	s := v.Sheet
	if !is(s.ThisClaim, "86000.00") || !is(s.Retention.Held, "2300.00") || !is(s.Retention.HeldToDate, "2300.00") || !is(s.Net, "83700.00") || !is(s.Payable, "92070.00") || !s.PaidInFull {
		t.Fatalf("totals: this claim %v retention %+v net %v payable %v", s.ThisClaim, s.Retention, s.Net, s.Payable)
	}
	if len(s.Payments) != 1 || !s.Payments[0].Current || s.Payments[0].Number != "PC-04" || s.Payments[0].Period != "SEP26" || !is(s.Payments[0].Ex, "83700.00") || !is(s.Paid.Inc, "92070.00") {
		t.Fatalf("payment list: %+v", s.Payments)
	}
	// Not issued, so the contract has certified and paid nothing yet.
	if c := v.Contracts[0]; !is(c.SumToDate, "150000.00") || !is(c.Adjusted, "150000.00") || !is(c.Certified, "0.00") || !is(c.Paid, "0.00") || c.Latest != nil {
		t.Fatalf("contract before issue: %+v", c)
	}

	m.call(t, http.MethodPost, base+"/invoices/"+created.ID+"/schedule", nil, http.StatusCreated, nil)
	m.getJSON(t, sheet, &v)
	c := v.Contracts[0]
	if !is(c.Certified, "86000.00") || !is(c.Retention, "2300.00") || !is(c.Paid, "83700.00") || c.Latest == nil || c.Latest.ClaimID != created.ID || c.Latest.VersionID == "" || c.Latest.Inc != "92070.00" {
		t.Fatalf("contract after issue: %+v latest %+v", c, c.Latest)
	}
}

// Slice 2: the PM types what is complete as a percent, assesses a line the
// claim left out, and the list prints each claim as its schedule does.
func TestClaimSheetTakesAPercentAndALineTheClaimLeftOut(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Claim percent")
	pkg, lines, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	// V01, approved at 11,000, is on the contract and not on the claim; V02 is not approved.
	var tab tabLedger
	m.call(t, http.MethodPost, base+"/variations", jsonBody(map[string]any{"package": pkg, "line": lines[0], "title": "Rock excavation", "claimed": "12400"}), http.StatusOK, &tab)
	m.ops(t, project, "variations", http.StatusOK, op("variations.decide", map[string]any{"variation": "V01", "decision": "approve", "approved": "11000"}))
	m.call(t, http.MethodPost, base+"/variations", jsonBody(map[string]any{"package": pkg, "line": lines[0], "title": "Extra fence", "claimed": "900"}), http.StatusOK, &tab)
	approved, open := tab.Variations[0].ID, tab.Variations[1].ID
	if tab.Variations[0].Number != "V01" {
		approved, open = open, approved
	}
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	sheet, rows := base+"/invoices?sheet="+created.ID, base+"/invoices/"+created.ID+"/claim-lines"

	var v sheetLedger
	m.getJSON(t, sheet, &v)
	if len(v.Claims) != 1 || v.Claims[0].DisplayNumber != "PC-04" || v.Claims[0].Period == "" {
		t.Fatalf("the list's claim: %+v", v.Claims)
	}
	sub := v.Sheet.Sections[0].Lines[0]

	// 80% of the line's total let: the dollars are Go's.
	m.call(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"id": sub.RowID, "percent": "80"}}}), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if l := v.Sheet.Sections[0].Lines[0]; !is(l.Completed, "80000.00") || l.Percent != "80.0" {
		t.Fatalf("80%% of 100,000: %+v", l)
	}
	if got := m.status(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"id": sub.RowID, "percent": "most"}}})); got != http.StatusUnprocessableEntity {
		t.Fatalf("a percent that is not a number = %d", got)
	}
	if got := m.status(t, http.MethodPut, rows, []byte(`{"rows":[{"assessed_to_date":"1"}]}`)); got != http.StatusBadRequest {
		t.Fatalf("a row that names nothing = %d", got)
	}

	// A variation the claim left out: the first assessment adds its row.
	m.call(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"variation": approved, "percent": "50"}}}), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if len(v.Sheet.Sections) != 2 || v.Sheet.Sections[1].Kind != "variations" {
		t.Fatalf("sections: %+v", v.Sheet.Sections)
	}
	if l := v.Sheet.Sections[1].Lines[0]; !is(l.Completed, "5500.00") || l.Percent != "50.0" || l.RowID == "" || !is(l.ThisClaim, "5500.00") {
		t.Fatalf("the variation's added row: %+v", l)
	}
	if got := m.status(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"variation": approved, "assessed_to_date": "1"}}})); got != http.StatusUnprocessableEntity {
		t.Fatalf("a second row on the same variation = %d", got)
	}
	// A percent needs a sum to be a percent of.
	if got := m.status(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"variation": open, "percent": "50"}}})); got != http.StatusUnprocessableEntity {
		t.Fatalf("a percent of a variation with no approved sum = %d", got)
	}

	// Through an operation the row is named by its line; the variation's row is found, not added twice.
	m.ops(t, project, "claims", http.StatusOK, op("claims.assess", map[string]any{"claim": created.ID, "row": "Superstructure", "percent": "50"}))
	m.ops(t, project, "claims", http.StatusOK, op("claims.assess", map[string]any{"claim": created.ID, "row": "V01", "assessed_to_date": "11000"}))
	m.getJSON(t, sheet, &v)
	if l := v.Sheet.Sections[0].Lines[1]; !is(l.Completed, "25000.00") || l.Percent != "50.0" {
		t.Fatalf("superstructure at 50%%: %+v", l)
	}
	if ls := v.Sheet.Sections[1].Lines; len(ls) != 1 || !is(ls[0].Completed, "11000.00") {
		t.Fatalf("the variation after the operation: %+v", ls)
	}
	// The claimed total still reads (20,000 and 26,000 this claim, as
	// printed): an added row claims nothing itself.
	if !is(v.Claims[0].Claimed, "46000.00") {
		t.Fatalf("claimed %v", v.Claims[0].Claimed)
	}
}

type breakdownLedger struct {
	Contracts []struct {
		Adjusted    *string `json:"adjusted"`
		Provisional *string `json:"provisional_adjustment"`
		Breakdown   []struct {
			ID      string  `json:"id"`
			Section string  `json:"section"`
			Label   string  `json:"label"`
			Amount  *string `json:"amount"`
			Final   bool    `json:"final"`
			Retired bool    `json:"retired"`
			Source  string  `json:"source"`
			Stored  bool    `json:"stored"`
		} `json:"breakdown"`
		Gaps []struct {
			Label string `json:"label"`
			Diff  string `json:"diff"`
		} `json:"breakdown_gaps"`
	} `json:"contracts"`
	Sheet *struct {
		Sections []struct {
			Kind  string `json:"kind"`
			Lines []struct {
				Key       string  `json:"key"`
				Label     string  `json:"label"`
				RowID     string  `json:"row_id"`
				Completed *string `json:"completed"`
				Percent   string  `json:"percent"`
				Final     bool    `json:"final"`
				Retired   bool    `json:"retired"`
			} `json:"lines"`
		} `json:"sections"`
		Notes []string `json:"notes"`
	} `json:"sheet"`
}

// Slice 3: a contract's breakdown. It is the cost lines until the PM's
// first edit stores it; then lines are added, moved, marked final and
// retired, each with Undo, and a claim's rows sit on them.
func TestContractBreakdownRoutes(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, foreign := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Breakdown")
	pkg, _, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	lines := base + "/contracts/" + pkg + "/breakdown"

	var v breakdownLedger
	m.getJSON(t, base+"/invoices", &v)
	if b := v.Contracts[0].Breakdown; len(b) != 2 || b[0].Stored || b[0].Label != "Substructure" || !is(b[0].Amount, "100000.00") {
		t.Fatalf("before any edit the breakdown is the cost lines: %+v", b)
	}
	rock := jsonBody(map[string]any{"label": "Rock, 40 tonne @ $120", "section": "provisional", "amount": "4800"})
	if got := foreign.status(t, http.MethodPost, lines, rock); got != http.StatusNotFound {
		t.Fatalf("foreign add = %d", got)
	}
	if got := m.status(t, http.MethodPost, base+"/contracts/"+newUUID(t)+"/breakdown", rock); got != http.StatusNotFound {
		t.Fatalf("a contract that is not there = %d", got)
	}
	for name, body := range map[string]string{"no label": `{"section":"works"}`, "a section that is neither": `{"label":"x","section":"extras"}`, "a sum that is not one": `{"label":"x","amount":"lots"}`} {
		if got := m.status(t, http.MethodPost, lines, []byte(body)); got != http.StatusUnprocessableEntity {
			t.Fatalf("%s = %d", name, got)
		}
	}
	m.call(t, http.MethodPost, lines, rock, http.StatusOK, &v)
	b := v.Contracts[0].Breakdown
	if len(b) != 3 || !b[0].Stored || b[0].Source != "award" || b[2].Section != "provisional" || b[2].Source != "user" || !is(b[2].Amount, "4800.00") {
		t.Fatalf("after adding a provisional sum: %+v", b)
	}
	// The sum sits under the first cost line, which the breakdown now overshoots.
	if g := v.Contracts[0].Gaps; len(g) != 1 || g[0].Label != "Substructure" || g[0].Diff != "4800.00" {
		t.Fatalf("gaps: %+v", g)
	}
	sub, sup, ps := b[0].ID, b[1].ID, b[2].ID
	if got := m.status(t, http.MethodPatch, lines+"/"+newUUID(t), []byte(`{"label":"x"}`)); got != http.StatusNotFound {
		t.Fatalf("a line that is not there = %d", got)
	}
	if got := foreign.status(t, http.MethodPatch, lines+"/"+sub, []byte(`{"label":"x"}`)); got != http.StatusNotFound {
		t.Fatalf("foreign edit = %d", got)
	}
	if got := m.status(t, http.MethodPatch, lines+"/"+sub, []byte(`{"final":true}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("a works line marked final = %d", got)
	}
	m.call(t, http.MethodPatch, lines+"/"+sub, []byte(`{"amount":"95200","note":"less the rock allowance"}`), http.StatusOK, &v)
	if g := v.Contracts[0].Gaps; len(g) != 0 {
		t.Fatalf("the breakdown adds up again: %+v", g)
	}

	// A claim's rows sit on the breakdown's lines; the sum it left out shows nil.
	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	sheet, rows := base+"/invoices?sheet="+created.ID, base+"/invoices/"+created.ID+"/claim-lines"
	m.getJSON(t, sheet, &v)
	if s := v.Sheet.Sections; len(s) != 2 || s[0].Kind != "works" || s[1].Kind != "provisional" || s[0].Lines[0].Key != "b:"+sub || s[0].Lines[0].RowID == "" || s[0].Lines[1].Key != "b:"+sup {
		t.Fatalf("sections: %+v", s)
	}
	if l := v.Sheet.Sections[1].Lines[0]; l.RowID != "" || !is(l.Completed, "0.00") {
		t.Fatalf("the sum the claim left out: %+v", l)
	}
	// Half the rock allowance, by percent, on the sum itself.
	m.call(t, http.MethodPut, rows, jsonBody(map[string]any{"rows": []any{map[string]any{"line": ps, "percent": "50"}}}), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if l := v.Sheet.Sections[1].Lines[0]; l.RowID == "" || !is(l.Completed, "2400.00") || l.Percent != "50.0" {
		t.Fatalf("the provisional sum at 50%%: %+v", l)
	}

	// Final, through an operation, with Undo.
	reply := m.ops(t, project, "claims", http.StatusOK, op("claims.breakdown_edit", map[string]any{"package": pkg, "line": "Rock, 40 tonne @ $120", "final": "yes"}))
	m.getJSON(t, sheet, &v)
	if !v.Contracts[0].Breakdown[2].Final || !v.Sheet.Sections[1].Lines[0].Final {
		t.Fatalf("marked final: %+v", v.Contracts[0].Breakdown[2])
	}
	m.undo(t, project, reply.ChangeID, http.StatusOK)
	m.getJSON(t, sheet, &v)
	if v.Contracts[0].Breakdown[2].Final {
		t.Fatal("undo did not clear the final mark")
	}
	m.call(t, http.MethodPatch, lines+"/"+ps, []byte(`{"final":true}`), http.StatusOK, nil)
	m.ops(t, project, "claims", http.StatusOK, op("claims.breakdown_add", map[string]any{"package": pkg, "label": "Site sheds", "amount": "1500"}))

	// Issued: the sum counts at what was spent, and the breakdown can no longer be rebuilt.
	m.ops(t, project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	m.call(t, http.MethodPost, base+"/invoices/"+created.ID+"/schedule", nil, http.StatusCreated, nil)
	m.getJSON(t, sheet, &v)
	if c := v.Contracts[0]; !is(c.Provisional, "-2400.00") || !is(c.Adjusted, "147600.00") {
		t.Fatalf("a final sum half spent: provisional %v adjusted %v", c.Provisional, c.Adjusted)
	}
	if got := m.status(t, http.MethodPost, lines, []byte(`{"rebuild":true}`)); got != http.StatusConflict {
		t.Fatalf("rebuild after a schedule is issued = %d", got)
	}
	// Retired, not deleted: the issued schedule keeps the line.
	m.call(t, http.MethodPatch, lines+"/"+sup, []byte(`{"retired":true}`), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if !v.Contracts[0].Breakdown[1].Retired || len(v.Sheet.Sections[0].Lines) != 3 || !v.Sheet.Sections[0].Lines[1].Retired {
		t.Fatalf("retired: %+v %+v", v.Contracts[0].Breakdown[1], v.Sheet.Sections[0].Lines)
	}
	// Site sheds went under Superstructure, the last works line; with that
	// line retired its cost line is left 48,500 short, and the sheet says so.
	if n := v.Sheet.Notes; len(n) != 1 || n[0] != "Breakdown is $48,500.00 under the contract line Superstructure." {
		t.Fatalf("notes: %v", n)
	}
}

// Rebuild before any schedule is issued: with no tender behind the award
// the breakdown is one stored line per cost line.
func TestContractBreakdownRebuildsBeforeIssue(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Breakdown rebuild")
	pkg, _, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	var v breakdownLedger
	m.call(t, http.MethodPost, base+"/contracts/"+pkg+"/breakdown", []byte(`{"label":"Extra","amount":"10"}`), http.StatusOK, &v)
	reply := m.ops(t, project, "claims", http.StatusOK, op("claims.breakdown_rebuild", map[string]any{"package": pkg}))
	m.getJSON(t, base+"/invoices", &v)
	if b := v.Contracts[0].Breakdown; len(b) != 2 || !b[0].Stored || b[0].Source != "award" || b[1].Label != "Superstructure" {
		t.Fatalf("rebuilt: %+v", b)
	}
	m.undo(t, project, reply.ChangeID, http.StatusOK)
	m.getJSON(t, base+"/invoices", &v)
	if b := v.Contracts[0].Breakdown; len(b) != 3 || b[2].Label != "Extra" {
		t.Fatalf("rebuild undone: %+v", b)
	}
}

type termsLedger struct {
	Contracts []struct {
		ContractDate     string `json:"contract_date"`
		RetentionPercent string `json:"retention_percent"`
		RetentionBase    string `json:"retention_base"`
		ScheduleDays     *int   `json:"schedule_days"`
	} `json:"contracts"`
	Claims []struct {
		Note          string `json:"schedule_note"`
		RetentionOpen bool   `json:"retention_open"`
	} `json:"claims"`
	Sheet *struct {
		Retention struct {
			RatePercent string  `json:"rate_percent"`
			BaseKind    string  `json:"base_kind"`
			Base        *string `json:"base"`
			Open        bool    `json:"open"`
			Limit       *string `json:"limit"`
			Held        *string `json:"held"`
			Percent     string  `json:"percent"`
			HeldToDate  *string `json:"held_to_date"`
		} `json:"retention"`
		Header *struct {
			Principal         string            `json:"principal"`
			PrincipalABN      string            `json:"principal_abn"`
			PrincipalTyped    bool              `json:"principal_typed"`
			AssessorOrg       string            `json:"assessor_org"`
			AssessorCompanyNo string            `json:"assessor_company_no"`
			AssessorTyped     bool              `json:"assessor_typed"`
			Approver          string            `json:"approver"`
			ApproverTyped     bool              `json:"approver_typed"`
			ContractDate      string            `json:"contract_date"`
			Warnings          []string          `json:"warnings"`
			Typed             map[string]string `json:"typed"`
		} `json:"header"`
	} `json:"sheet"`
}

// Slice 4: the contract's terms (its date, retention as a percent of a
// base), what a claim holds or returns against them, the parties a
// schedule names, and the PM's note. Every party field is optional and
// none is invented.
func TestScheduleTermsRetentionAndParties(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, foreign := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Schedule terms")
	pkg, _, _ := liveContract(t, m, project)
	base := "/api/projects/" + project
	award := base + "/procurement/packages/" + pkg + "/award"

	// The contract's periods and its terms are separate groups of one route.
	m.call(t, http.MethodPatch, award, []byte(`{"schedule_days":5,"schedule_basis":"business"}`), http.StatusOK, nil)
	var v termsLedger
	m.call(t, http.MethodPatch, award, []byte(`{"contract_date":"2026-02-25","retention_percent":"5"}`), http.StatusOK, &v)
	if c := v.Contracts[0]; c.ContractDate != "2026-02-25" || c.RetentionPercent != "5" || c.RetentionBase != "works" || c.ScheduleDays == nil || *c.ScheduleDays != 5 {
		t.Fatalf("terms: %+v", c)
	}
	for name, body := range map[string]string{"a base that is none of the three": `{"retention_base":"gross"}`, "a rate over 100": `{"retention_percent":"120"}`, "a date that is not one": `{"contract_date":"25/02/2026"}`} {
		if got := m.status(t, http.MethodPatch, award, []byte(body)); got != http.StatusUnprocessableEntity {
			t.Fatalf("%s = %d", name, got)
		}
	}
	if got := m.status(t, http.MethodPatch, award, []byte(`{"retention_rate":"5"}`)); got != http.StatusBadRequest {
		t.Fatalf("a field that is not one = %d", got)
	}
	if got := foreign.status(t, http.MethodPatch, award, []byte(`{"contract_date":"2026-02-25"}`)); got != http.StatusNotFound {
		t.Fatalf("foreign terms = %d", got)
	}

	// The parties, typed once on the project.
	settings := base + "/delivery/settings"
	m.call(t, http.MethodPut, settings, []byte(`{"principal_name":"Harbour Trust","principal_abn":"51 824 753 556","assessor_org":"Example PM Pty Ltd","assessor_company_no":"605 0814 883"}`), http.StatusOK, nil)
	if got := m.status(t, http.MethodPut, settings, []byte(`{"principal_email":"not an address"}`)); got != http.StatusUnprocessableEntity && got != http.StatusBadRequest {
		t.Fatalf("an email that is not one = %d", got)
	}

	raw, err := os.ReadFile("../invoices/testdata/claim-pc04.md")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	m.call(t, http.MethodPost, base+"/invoices?name=pc04.md&kind=progress_claim&package="+pkg, raw, http.StatusCreated, &created)
	m.waitInvoices(t, project, 1)
	sheet, rows := base+"/invoices?sheet="+created.ID, base+"/invoices/"+created.ID+"/claim-lines"
	m.getJSON(t, sheet, &v)
	h := v.Sheet.Header
	if h == nil || h.Principal != "Harbour Trust" || !h.PrincipalTyped || h.PrincipalABN != "51 824 753 556" || h.AssessorOrg != "Example PM Pty Ltd" || !h.AssessorTyped || h.ContractDate != "2026-02-25" {
		t.Fatalf("header: %+v", h)
	}
	// The approver's name is not typed, so the signed-in user stands in.
	if h.ApproverTyped || h.Approver == "" || h.Typed["assessor_name"] != "" || h.Typed["principal_name"] != "Harbour Trust" {
		t.Fatalf("approver %q typed %v, typed fields %v", h.Approver, h.ApproverTyped, h.Typed)
	}
	// Ten digits is neither an ACN nor an ABN: printed as typed, and said.
	if len(h.Warnings) != 1 || !strings.Contains(h.Warnings[0], "605 0814 883") {
		t.Fatalf("warnings: %v", h.Warnings)
	}

	// Retention: 5% of the works sub total is the limit; the claim is open until it says.
	r := v.Sheet.Retention
	if r.RatePercent != "5" || r.BaseKind != "works" || !is(r.Base, "150000.00") || !is(r.Limit, "7500.00") || !r.Open || !v.Claims[0].RetentionOpen {
		t.Fatalf("retention before: %+v", r)
	}
	m.ops(t, project, "claims", http.StatusOK, op("claims.accept_claimed", map[string]any{"claim": created.ID}))
	if got := m.status(t, http.MethodPut, rows, []byte(`{"retention":{"hold_percent":"6"}}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("holding past the limit = %d", got)
	}
	if got := m.status(t, http.MethodPut, rows, []byte(`{"retention":{"return_percent":"1"}}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("returning what is not held = %d", got)
	}
	if got := m.status(t, http.MethodPut, rows, []byte(`{"retention":{"hold_percent":"2","hold":"100"}}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("two ways at once = %d", got)
	}
	m.call(t, http.MethodPut, rows, []byte(`{"retention":{"hold_percent":"2.5"}}`), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if r := v.Sheet.Retention; !is(r.Held, "3750.00") || r.Percent != "2.5" || !is(r.HeldToDate, "3750.00") || r.Open || v.Claims[0].RetentionOpen {
		t.Fatalf("2.5%% held: %+v", r)
	}
	// Through an operation, in dollars; nil is "no change this claim".
	m.ops(t, project, "claims", http.StatusOK, op("claims.assess", map[string]any{"claim": created.ID, "retention_hold": "1000"}))
	m.getJSON(t, sheet, &v)
	if r := v.Sheet.Retention; !is(r.HeldToDate, "1000.00") {
		t.Fatalf("1,000 held: %+v", r)
	}
	m.ops(t, project, "claims", http.StatusOK, op("claims.assess", map[string]any{"claim": created.ID, "retention_hold": "0"}))
	m.getJSON(t, sheet, &v)
	if r := v.Sheet.Retention; !is(r.Held, "0.00") || r.Open {
		t.Fatalf("no change this claim: %+v", r)
	}

	// The PM's own words on the schedule.
	m.call(t, http.MethodPatch, base+"/invoices/"+created.ID, []byte(`{"fields":{"schedule_note":"Retention is held on the works only."}}`), http.StatusOK, nil)
	m.getJSON(t, sheet, &v)
	if v.Claims[0].Note != "Retention is held on the works only." {
		t.Fatalf("note %q", v.Claims[0].Note)
	}
	// The terms and the parties, as chat and the screen send them.
	// A contract's terms are the PM's to confirm: the operation waits for the
	// click, and the screen's own route applies them.
	m.ops(t, project, "claims", http.StatusUnprocessableEntity, op("claims.contract_periods", map[string]any{"package": pkg, "retention_percent": "10", "retention_base": "to_date"}))
	m.call(t, http.MethodPatch, award, []byte(`{"retention_percent":"10","retention_base":"to_date"}`), http.StatusOK, nil)
	m.ops(t, project, "delivery", http.StatusOK, op("delivery.settings", map[string]any{"assessor_name": "A. Manager", "assessor_company_no": "605 081 488"}))
	m.getJSON(t, sheet, &v)
	if c := v.Contracts[0]; c.RetentionPercent != "10" || c.RetentionBase != "to_date" || c.ContractDate != "2026-02-25" || *c.ScheduleDays != 5 {
		t.Fatalf("terms after the operation: %+v", c)
	}
	if h := v.Sheet.Header; h.Approver != "A. Manager" || !h.ApproverTyped || len(h.Warnings) != 0 {
		t.Fatalf("header after the operation: %+v", h)
	}
}
