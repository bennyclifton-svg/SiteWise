package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// contractFixture is a live Main Works award with two contract lines,
// written directly: these tests are about what sits on a contract, not how
// it was awarded. It returns the package and the two lines.
func contractFixture(t *testing.T, s *store.Store, amounts ...string) (string, []string) {
	t.Helper()
	ctx := context.Background()
	row, _ := procurement.DisciplinePackage("trade.main_works", "Main Works")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for i, a := range amounts {
		before := map[string]bool{}
		for _, it := range plan.Items {
			before[it.ID] = true
		}
		label := []string{"Substructure", "Superstructure", "Finishes"}[i]
		plan, err = s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: label, LineKind: "works", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}})
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
		if plan, err = s.PutCostValue(ctx, orgA, projectA, id, userA, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "allowance"}}); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, id)
	}
	sub, award := newUUID(t), newUUID(t)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,$2,$3,$4,'tender.md','awarded','2026-11-01')`, orgA, sub, projectA, pkg.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,$2,$3,$4,$5,$6,false,$7)`, orgA, award, projectA, pkg.ID, sub, lines[0], userA); err != nil {
		t.Fatal(err)
	}
	for i, l := range lines {
		if _, err := pool.Exec(ctx, `INSERT INTO cost_line_sources(org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, orgA, newUUID(t), projectA, award, l, "line:"+l[:8], "", i); err != nil {
			t.Fatal(err)
		}
	}
	return pkg.ID, lines
}


// lineFigures are a line's figures as the grid shows them: Approved and
// Forecast Variations and Forecast Final Cost from the totals, Remaining
// from the ledger.
func lineFigures(t *testing.T, s *store.Store, line string) (approved, forecastVar, forecast string, remaining *int64) {
	t.Helper()
	ctx := context.Background()
	totals, err := s.CostTotals(ctx, orgA, projectA, "", "overall")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range plan.Items {
		if it.ID == line {
			if v := it.Values[costs.MetricApprovedVariations]; v.Amount != nil {
				approved = string(*v.Amount)
			}
			if v := it.Values[costs.MetricForecastVariations]; v.Amount != nil {
				forecastVar = string(*v.Amount)
			}
		}
	}
	if f := totals.Items[line].Forecast; f.Amount != nil {
		forecast = string(*f.Amount)
	}
	v, err := s.InvoiceLedger(ctx, orgA, projectA, "2026-11-19")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range v.Summary.Lines {
		if l.ID == line {
			remaining = l.ContractRemaining
		}
	}
	return
}

func TestVariationRegisterAndTheFigures(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	now := time.Now()

	// Two lines: the PM picks one.
	if _, err := s.AddVariation(ctx, orgA, projectA, userA, store.VariationInput{Package: pkg, Title: sp("Rock excavation")}, now); !errors.Is(err, store.ErrInvalidVariation) {
		t.Fatalf("a variation on a two-line contract needs its line: %v", err)
	}
	v, err := s.AddVariation(ctx, orgA, projectA, userA, store.VariationInput{Package: pkg, Line: sp(lines[0]), Title: sp("Rock excavation"), Claimed: sp("12,400"), TimeDays: sp("3")}, now)
	if err != nil {
		t.Fatal(err)
	}
	if v.Number != "V01" || v.Status != invoices.VariationSubmitted {
		t.Fatalf("the first variation is V01, submitted: %+v", v)
	}
	if _, err := s.AddVariation(ctx, orgA, projectA, userA, store.VariationInput{Package: pkg, Line: sp(lines[0]), Number: sp("v01"), Title: sp("Again")}, now); !errors.Is(err, store.ErrVariationNumber) {
		t.Fatalf("a number used on the contract is refused: %v", err)
	}
	// Open: forecast at the claimed amount (no estimate yet), nothing approved.
	approved, fv, forecast, remaining := lineFigures(t, s, lines[0])
	if approved != "" || fv != "12400.00" || forecast != "112400.00" || remaining == nil || *remaining != 10000000 {
		t.Fatalf("submitted: approved %q forecast variations %q forecast %q remaining %v", approved, fv, forecast, remaining)
	}
	// Approving needs an amount.
	if _, err := s.EditVariation(ctx, orgA, projectA, userA, v.ID, store.VariationInput{Status: sp("approved")}, now); !errors.Is(err, store.ErrInvalidVariation) {
		t.Fatalf("approve without an amount: %v", err)
	}
	marked := store.WithChange(ctx, store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "variations"})
	if _, err := s.EditVariation(marked, orgA, projectA, userA, v.ID, store.VariationInput{Status: sp("approved"), Approved: sp("11000")}, now); err != nil {
		t.Fatal(err)
	}
	change, _ := store.ChangeOf(marked)
	// Done means 1: three columns move by the same 11,000 and the open
	// forecast goes.
	approved, fv, forecast, remaining = lineFigures(t, s, lines[0])
	if approved != "11000.00" || fv != "" || forecast != "111000.00" || remaining == nil || *remaining != 11100000 {
		t.Fatalf("approved: approved %q forecast variations %q forecast %q remaining %v", approved, fv, forecast, remaining)
	}
	cs, vs, err := s.VariationRegister(ctx, orgA, projectA)
	if err != nil || len(cs) != 1 || len(vs) != 1 {
		t.Fatalf("register: %v %d %d", err, len(cs), len(vs))
	}
	amounts := []*int64{cs[0].Lines[0].Amount, cs[0].Lines[1].Amount}
	sum, appr, toDate := invoices.ContractTotals(amounts, vs)
	if *sum != 15000000 || appr != 1100000 || *toDate != 16100000 {
		t.Fatalf("contract sum %d approved %d to date %d", *sum, appr, *toDate)
	}
	// Undo puts it all back.
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatal(err)
	}
	approved, fv, forecast, _ = lineFigures(t, s, lines[0])
	if approved != "" || fv != "12400.00" || forecast != "112400.00" {
		t.Fatalf("after undo: approved %q forecast variations %q forecast %q", approved, fv, forecast)
	}
	// An open variation with no figure leaves the forecast unknown.
	if _, err := s.AddVariation(ctx, orgA, projectA, userA, store.VariationInput{Package: pkg, Line: sp(lines[1]), Title: sp("Unknown")}, now); err != nil {
		t.Fatal(err)
	}
	if _, _, forecast, _ = lineFigures(t, s, lines[1]); forecast != "" {
		t.Fatalf("an unknown variation leaves the forecast unknown, got %q", forecast)
	}
	// Another org sees nothing and cannot write.
	if _, err := s.EditVariation(ctx, orgB, projectB, userB, v.ID, store.VariationInput{Title: sp("x")}, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org edit: %v", err)
	}
	if _, err := s.AddVariation(ctx, orgB, projectB, userB, store.VariationInput{Package: pkg, Title: sp("x")}, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org add: %v", err)
	}
	if _, err := s.AddVariation(ctx, orgA, projectA, userB, store.VariationInput{Package: pkg, Title: sp("x")}, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a foreign actor: %v", err)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg, userA); !errors.Is(err, store.ErrAwardHasVariations) {
		t.Fatalf("an award with variations cannot be undone: %v", err)
	}
}

// A frozen baseline keeps the forecast it froze; only the draft adds
// variations.
func TestVariationsLeaveTheBaseline(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00")
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.BaselineCostPlan(ctx, orgA, projectA, userA, store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version}); err != nil {
		t.Fatal(err)
	}
	draft, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.CostTotals(ctx, orgA, projectA, draft.BaselineVersionID, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddVariation(ctx, orgA, projectA, userA, store.VariationInput{Package: pkg, Title: sp("Extra"), Estimate: sp("5000")}, time.Now()); err != nil {
		t.Fatal(err)
	}
	after, err := s.CostTotals(ctx, orgA, projectA, draft.BaselineVersionID, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if b, a := before.Overall["forecast"].KnownSubtotal, after.Overall["forecast"].KnownSubtotal; b != a || b != "100000.00" {
		t.Fatalf("baseline forecast moved: %s to %s", b, a)
	}
	if _, _, forecast, _ := lineFigures(t, s, lines[0]); forecast != "105000.00" {
		t.Fatalf("the draft adds the estimate: %q", forecast)
	}
}
