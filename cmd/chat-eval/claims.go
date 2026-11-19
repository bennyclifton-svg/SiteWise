package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// seedClaims gives the house a live Main Works contract (Greyhaven Builders)
// with two lines, variation V01 submitted at $12,400 claimed, and progress
// claim 4 read with two contract rows and a retention row, none assessed
// (claims group; docs/plans/2026-11-17-variations-progress-claims.md,
// "From chat"). Everything is invented. The award row is written directly:
// the cases are about what sits on a contract, not how it was awarded.
func (s *session) seedClaims(ctx context.Context) error {
	st := s.l.st
	conn, err := pgx.Connect(ctx, s.l.dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var user string
	if err := conn.QueryRow(ctx, `SELECT id::text FROM users WHERE org_id=$1 LIMIT 1`, s.org).Scan(&user); err != nil {
		return err
	}
	row, _ := procurement.DisciplinePackage("trade.main_works", "Main Works")
	pkg, err := st.CreateDisciplinePackage(ctx, s.org, s.project, user, row)
	if err != nil {
		return err
	}
	plan, err := st.ReadCostPlan(ctx, s.org, s.project, "")
	if err != nil {
		return err
	}
	var lines []string
	for i, label := range []string{"Substructure", "Superstructure"} {
		before := map[string]bool{}
		for _, it := range plan.Items {
			before[it.ID] = true
		}
		if plan, err = st.CreateCostItem(ctx, s.org, s.project, user, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: label, LineKind: "works", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}}); err != nil {
			return err
		}
		var id string
		for _, it := range plan.Items {
			if !before[it.ID] {
				id = it.ID
			}
		}
		amount := costs.Money([]string{"400000.00", "600000.00"}[i])
		if plan, err = st.PutCostValue(ctx, s.org, s.project, id, user, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version,
			Value: costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "allowance"}}); err != nil {
			return err
		}
		lines = append(lines, id)
	}
	sub, award := newID(), newID()
	if _, err := conn.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,$2,$3,$4,'greyhaven-tender.md','awarded','2026-09-01')`, s.org, sub, s.project, pkg.ID); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, `INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,$2,$3,$4,$5,$6,false,$7)`, s.org, award, s.project, pkg.ID, sub, lines[0], user); err != nil {
		return err
	}
	for i, l := range lines {
		if _, err := conn.Exec(ctx, `INSERT INTO cost_line_sources(org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal) VALUES($1,$2,$3,$4,$5,$6,'',$7)`, s.org, newID(), s.project, award, l, fmt.Sprintf("line:%d", i), i); err != nil {
			return err
		}
	}
	claimed, line := "12400", lines[0]
	if _, err := st.AddVariation(ctx, s.org, s.project, user, store.VariationInput{Package: pkg.ID, Line: &line, Title: ptr("Extra footing depth"), Claimed: &claimed}, time.Now()); err != nil {
		return err
	}
	up := store.InvoiceUpload{SHA256: make([]byte, 32), Size: 9, MediaType: "application/pdf", Filename: "greyhaven-pc4.pdf", UserID: user, ClaimPackage: pkg.ID}
	copy(up.SHA256, []byte("greyhaven progress claim four"))
	created, err := st.CreateInvoice(ctx, s.org, s.project, up, time.Now())
	if err != nil {
		return err
	}
	d := func(f, v string) invoices.Decision {
		return invoices.Decision{Field: f, Value: v, Band: "green", DecidedBy: "rule"}
	}
	n := func(c int64) *int64 { return &c }
	out := invoices.Outcome{Asked: true}
	out.Decisions = []invoices.Decision{d(invoices.FieldNumber, "4"), d(invoices.FieldDate, "2026-11-02"), d(invoices.FieldTotal, "5500000"),
		d(invoices.FieldGST, "500000"), d(invoices.FieldPeriod, "2026-11-01"), d(invoices.FieldCurrency, "AUD")}
	out.Claim = &invoices.ClaimReading{TableFound: true, Rows: []invoices.ClaimRow{
		{Kind: invoices.RowContract, Label: "Substructure", LineID: lines[0], MappedBy: "rule", Band: "green", ToDate: n(9000000), Previous: n(5000000), ThisClaim: n(4000000)},
		{Kind: invoices.RowContract, Label: "Superstructure", LineID: lines[1], MappedBy: "rule", Band: "green", ToDate: n(1000000), Previous: n(0), ThisClaim: n(1000000)},
		{Kind: invoices.RowRetention, Label: "Retention", MappedBy: "rule", Band: "green"},
	}}
	_, err = st.CommitInvoiceRead(ctx, s.org, created.ID, out, false, time.Now())
	return err
}

func ptr(s string) *string { return &s }

// newID is a random UUID for the rows written directly.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
