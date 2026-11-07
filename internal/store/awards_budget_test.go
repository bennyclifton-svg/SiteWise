package store_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Owner direction, 7 Nov 2026: a package with no budget adopts the approved
// contract sum on award, and undo returns the budget to what it was.
func TestAwardAdoptsContractSumWhenBudgetIsBlank(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, orgA, store.Decision{ID: "11111111-1111-4111-8111-aaaaaaaaaa34", DocumentID: docA, Field: "discipline", Value: "consultant.town_planner", Band: "amber", DecidedBy: "jev"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	feeLine := func(p costs.Plan) *costs.Item {
		for i := range p.Items {
			if p.Items[i].LineKind == "fee" && p.Items[i].PackageID == pkg.ID && p.Items[i].ParentItemID == "" {
				return &p.Items[i]
			}
		}
		return nil
	}
	fee := feeLine(plan)
	var before costs.Value
	if fee != nil {
		before = fee.Values["budget"]
	}
	if before.Amount != nil {
		t.Fatalf("fixture expects a blank budget, got %+v", before)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, it := range table.Items {
		if it.Kind == "stage" && it.Base && len(keys) < 2 {
			keys = append(keys, it.Key)
		}
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, id, pageText(
		[]string{"Mecone Pty Ltd"}, []string{"ABN 88 071 762 537"}, []string{"1. FEE PROPOSAL"},
		[]string{"Advice", "$3,000"}, []string{"Report", "$9,000"},
	), answers{"price:" + keys[0]: "a1", "price:" + keys[1]: "a2", "firm_name": "c1"}, false)

	lines := []store.AwardLine{{Label: "Advice", Keys: keys[:1]}, {Label: "Report", Keys: keys[1:2]}}
	v, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines, Total: "12000.00"})
	if err != nil {
		t.Fatal(err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	var parent costs.Item
	for _, it := range plan.Items {
		if it.ID == v.ParentItemID {
			parent = it
		}
	}
	b := parent.Values["budget"]
	if b.ValueState != "known" || b.Amount == nil || *b.Amount != "12000.00" || b.Rationale == "" {
		t.Fatalf("budget did not adopt the contract sum: %+v", b)
	}
	totals, err := costs.Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if g := totals.Items[parent.ID]; g.BudgetVariance == nil || *g.BudgetVariance != "0.00" {
		t.Fatalf("variance %+v", g.BudgetVariance)
	}

	if err := s.UndoAward(ctx, orgA, projectA, pkg.ID, userA); err != nil {
		t.Fatal(err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	if fee := feeLine(plan); fee != nil {
		if got := fee.Values["budget"]; got.Amount != nil || got.ValueState != before.ValueState {
			t.Fatalf("undo left budget %+v, before %+v", got, before)
		}
	}
}
