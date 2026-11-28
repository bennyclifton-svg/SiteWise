package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// seedContract gives a project a live Main Works award with two contract
// lines (Substructure $100,000, Superstructure $50,000), for the Variations
// and claims tab's browser test. The award row is written directly: the
// test is about what sits on a contract, not how it was awarded.
func seedContract(ctx context.Context, st *store.Store, dsn, project string) (string, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	var org, user string
	if err := conn.QueryRow(ctx, `SELECT p.org_id::text, u.id::text FROM projects p JOIN users u ON u.org_id=p.org_id WHERE p.id::text=$1 ORDER BY u.created_at LIMIT 1`, project).Scan(&org, &user); err != nil {
		return "", err
	}
	row, _ := procurement.DisciplinePackage("trade.main_works", "Main Works")
	pkg, err := st.CreateDisciplinePackage(ctx, org, project, user, row)
	if err != nil {
		return "", err
	}
	plan, err := st.ReadCostPlan(ctx, org, project, "")
	if err != nil {
		return "", err
	}
	var lines []string
	for i, label := range []string{"Substructure", "Superstructure"} {
		before := map[string]bool{}
		for _, it := range plan.Items {
			before[it.ID] = true
		}
		if plan, err = st.CreateCostItem(ctx, org, project, user, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: label, LineKind: "works", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}}); err != nil {
			return "", err
		}
		var id string
		for _, it := range plan.Items {
			if !before[it.ID] {
				id = it.ID
			}
		}
		amount := costs.Money([]string{"100000.00", "50000.00"}[i])
		if plan, err = st.PutCostValue(ctx, org, project, id, user, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version,
			Value: costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "allowance"}}); err != nil {
			return "", err
		}
		lines = append(lines, id)
	}
	sub, award := uuid(), uuid()
	if _, err := conn.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,$2,$3,$4,'tender.md','awarded','2026-09-01')`, org, sub, project, pkg.ID); err != nil {
		return "", err
	}
	if _, err := conn.Exec(ctx, `INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,$2,$3,$4,$5,$6,false,$7)`, org, award, project, pkg.ID, sub, lines[0], user); err != nil {
		return "", err
	}
	for i, l := range lines {
		if _, err := conn.Exec(ctx, `INSERT INTO cost_line_sources(org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal) VALUES($1,$2,$3,$4,$5,$6,'',$7)`, org, uuid(), project, award, l, fmt.Sprintf("line:%d", i), i); err != nil {
			return "", err
		}
	}
	return pkg.ID, nil
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// claimMarkdown is a synthetic progress claim on that contract.
const claimMarkdown = `HARBOURSIDE BUILDING PTY LTD

# PROGRESS CLAIM

Synthetic test claim; not a real document.

| Claim no | PC-04 |
|---|---|
| Date | 14/10/2026 |

| Item | Contract value | Claimed to date | Previous claims | This claim |
|---|---|---|---|---|
| Substructure | $100,000.00 | $60,000.00 | $0.00 | $60,000.00 |
| Superstructure | $50,000.00 | $26,000.00 | $0.00 | $26,000.00 |
| Retention | | -$2,300.00 | $0.00 | -$2,300.00 |
| Total | $150,000.00 | $83,700.00 | $0.00 | $83,700.00 |

| | |
|---|---|
| GST | $8,370.00 |
| Total payable | $92,070.00 |
`

func contractRoutes(mux *http.ServeMux, st *store.Store, dsn string) {
	mux.HandleFunc("POST /__e2e/contract", func(w http.ResponseWriter, r *http.Request) {
		pkg, err := seedContract(r.Context(), st, dsn, r.URL.Query().Get("project"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"package":%q}`, pkg)
	})
	// A tax invoice from the contract's firm, read as Jev would read it, so
	// the link rule and placing can be tested without a document:
	// ?project=&number=&total=&gst= (cents).
	mux.HandleFunc("POST /__e2e/contract-invoice", func(w http.ResponseWriter, r *http.Request) {
		id, err := seedContractInvoice(r.Context(), st, dsn, r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":%q}`, id)
	})
	mux.HandleFunc("GET /__e2e/claim.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte(claimMarkdown))
	})
}

func seedContractInvoice(ctx context.Context, st *store.Store, dsn string, q map[string][]string) (string, error) {
	get := func(k string) string {
		if v := q[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	var org, user, pkg, line string
	if err := conn.QueryRow(ctx, `SELECT a.org_id::text, a.created_by::text, a.package_id::text, s.cost_item_id::text FROM awards a
JOIN cost_line_sources s ON s.org_id=a.org_id AND s.award_id=a.id WHERE a.project_id::text=$1 ORDER BY s.ordinal LIMIT 1`, get("project")).Scan(&org, &user, &pkg, &line); err != nil {
		return "", err
	}
	total, err1 := strconv.ParseInt(get("total"), 10, 64)
	gst, err2 := strconv.ParseInt(get("gst"), 10, 64)
	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("total and gst are cents")
	}
	sum := sha256.Sum256([]byte(get("project") + get("number")))
	now := time.Now()
	created, err := st.CreateInvoice(ctx, org, get("project"), store.InvoiceUpload{SHA256: sum[:], Size: 1, MediaType: "application/pdf", Filename: get("number") + ".pdf", UserID: user}, now)
	if err != nil {
		return "", err
	}
	d := func(f, v string) invoices.Decision {
		return invoices.Decision{Field: f, Value: v, Band: "green", DecidedBy: "rule"}
	}
	out := invoices.Outcome{Asked: true, InvoiceCount: 1, Decisions: []invoices.Decision{d(invoices.FieldKind, "invoice"), d(invoices.FieldNumber, get("number")),
		d(invoices.FieldDate, now.Format("2006-01-02")), d(invoices.FieldTotal, strconv.FormatInt(total, 10)), d(invoices.FieldGST, strconv.FormatInt(gst, 10)),
		d(invoices.FieldSupplier, pkg), d(invoices.FieldLine, line), d(invoices.FieldPeriod, now.Format("2006-01")+"-01"), d(invoices.FieldCurrency, "AUD")}}
	if _, err := st.CommitInvoiceRead(ctx, org, created.ID, out, false, now); err != nil {
		return "", err
	}
	return created.ID, nil
}
