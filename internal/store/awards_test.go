package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func TestAwardWritesTheBreakdownAndUndoRestores(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, orgA, store.Decision{ID: "11111111-1111-4111-8111-aaaaaaaaaa33", DocumentID: docA, Field: "discipline", Value: "consultant.town_planner", Band: "amber", DecidedBy: "jev"}); err != nil {
		t.Fatal(err)
	}
	// The package's fee line, with the budget the PM set.
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	var fee *costs.Item
	for i := range plan.Items {
		if plan.Items[i].LineKind == "fee" && plan.Items[i].PackageID == pkg.ID {
			fee = &plan.Items[i]
		}
	}
	if fee == nil {
		plan, err = s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: "Town planning", LineKind: "fee", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}})
		if err != nil {
			t.Fatal(err)
		}
		for i := range plan.Items {
			if plan.Items[i].LineKind == "fee" && plan.Items[i].PackageID == pkg.ID {
				fee = &plan.Items[i]
			}
		}
	}
	budget := costs.Money("16000.00")
	if plan, err = s.PutCostValue(ctx, orgA, projectA, fee.ID, userA, "budget", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: costs.Value{ValueState: "known", Amount: &budget, Origin: "user", Meaning: "allowance"}}); err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, it := range table.Items {
		if it.Kind == "stage" && it.Base && len(keys) < 3 {
			keys = append(keys, it.Key)
		}
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, id, pageText(
		[]string{"Mecone Pty Ltd"}, []string{"ABN 88 071 762 537"}, []string{"1. FEE PROPOSAL"},
		[]string{"Advice", "$3,000"}, []string{"Pre-lodgement", "$2,500"}, []string{"Report", "$9,000"},
	), answers{"price:" + keys[0]: "a1", "price:" + keys[1]: "a2", "price:" + keys[2]: "a3", "firm_name": "c1"}, false)

	lines := []store.AwardLine{{Label: "Preliminary advice", Keys: keys[:1]}, {Label: "Pre-lodgement and report", Keys: keys[1:3]}}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: lines, Total: "14000.00"}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("a running total that is not the sum must be refused: %v", err)
	}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Twice", Keys: []string{keys[0], keys[0]}}}}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("an item in two lines must be refused: %v", err)
	}
	preview, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: lines, Total: "14500.00"})
	if err != nil || preview.Total != 1450000 || preview.Lines[1].Cents != 1150000 || preview.Budget == nil || *preview.Budget != 1600000 {
		t.Fatalf("preview %+v %v", preview, err)
	}
	if _, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines, Total: "14500.00"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("second award: %v", err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	totals, err := costs.Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	var parent costs.Item
	var children []costs.Item
	for _, it := range plan.Items {
		if it.ID == fee.ID {
			parent = it
		}
		if it.ParentItemID == fee.ID {
			children = append(children, it)
		}
	}
	if parent.Posting || parent.Values["budget"].Amount == nil || *parent.Values["budget"].Amount != "16000.00" || len(children) != 2 {
		t.Fatalf("parent %+v children %d", parent, len(children))
	}
	if g := totals.Items[fee.ID]; g.Forecast.Amount == nil || *g.Forecast.Amount != "14500.00" || g.BudgetVariance == nil || *g.BudgetVariance != "1500.00" {
		t.Fatalf("group figures %+v", g)
	}
	award, err := s.ReadAward(ctx, orgA, projectA, pkg.ID)
	if err != nil || award == nil || len(award.Lines) != 2 || len(award.Lines[1].Keys) != 2 || award.Locked {
		t.Fatalf("award %+v %v", award, err)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if table.Columns[0].Submission.Status != store.SubmissionAwarded || table.Lifecycle != "appointed" {
		t.Fatalf("after award %+v %s", table.Columns[0].Submission, table.Lifecycle)
	}

	// An invoice on a child line locks the award.
	pool := rawPool(t)
	inv := "11111111-1111-4111-8111-aaaaaaaaab01"
	if _, err := pool.Exec(ctx, `INSERT INTO invoices (org_id,id,project_id,file_id,filename,status,period) VALUES ($1,$2,$3,$4,'inv.pdf','filed','2026-10-01')`, orgA, inv, projectA, fileA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO invoice_allocations (org_id,id,project_id,invoice_id,cost_item_id,decided_by,band) VALUES ($1,$2,$3,$4,$5,'user','green')`, orgA, "11111111-1111-4111-8111-aaaaaaaaab02", projectA, inv, children[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg.ID, userA); !errors.Is(err, store.ErrAwardLocked) {
		t.Fatalf("undo with an invoice: %v", err)
	}
	if award, _ := s.ReadAward(ctx, orgA, projectA, pkg.ID); award == nil || !award.Locked {
		t.Fatal("award not shown locked")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM invoices WHERE org_id=$1 AND id=$2`, orgA, inv); err != nil {
		t.Fatal(err)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg.ID, userA); err != nil {
		t.Fatal(err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	for _, it := range plan.Items {
		if it.ParentItemID == fee.ID {
			t.Fatalf("child left after undo: %+v", it)
		}
		if it.ID == fee.ID && (!it.Posting || it.Label != fee.Label || *it.Values["budget"].Amount != "16000.00") {
			t.Fatalf("parent not restored: %+v", it)
		}
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if table.Columns[0].Submission.Status != store.SubmissionReceived || table.Lifecycle == "appointed" {
		t.Fatalf("after undo %+v %s", table.Columns[0].Submission, table.Lifecycle)
	}
	if _, err := s.Award(ctx, orgB, projectA, pkg.ID, userB, store.AwardInput{SubmissionID: id, Lines: lines}); err == nil {
		t.Fatal("another org awarded")
	}
}
