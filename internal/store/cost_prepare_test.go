package store_test

import (
	"context"
	"errors"
	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"sync"
	"testing"
)

func TestCostPrepareAndSubtreeDeletion(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for _, sys := range []string{"access-egress.barriers", "access-egress.stairs-and-ramps"} {
		if _, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: sys, Action: "new", Title: sys}); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.PrepareCostPlan(ctx, orgB, projectA, userB); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("foreign prepare", e)
	}
	// A first-open race must initialise and insert only once.
	results := make([]store.PreparedCostPlan, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for n := range results {
		wg.Add(1)
		go func(n int) { defer wg.Done(); results[n], errs[n] = s.PrepareCostPlan(ctx, orgA, projectA, userA) }(n)
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if results[0].Plan.ID != results[1].Plan.ID || results[0].Plan.Version != results[1].Plan.Version || results[0].Changed == results[1].Changed {
		t.Fatal("prepare not idempotent", results)
	}
	p := results[0].Plan
	var events int
	if e := rawPool(t).QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1::uuid AND kind='costs'`, orgA).Scan(&events); e != nil || events != 1 {
		t.Fatal("no-op preparation published another event", events, e)
	}
	if len(p.Suggestions) != 0 || len(p.Items) < 10 {
		t.Fatal("defaults not saved", p)
	}
	var parent, leaf string
	for _, i := range p.Items {
		if len(i.Values) > 0 {
			t.Fatal("invented price", i)
		}
		if i.SourceKey == "works:access-egress" {
			parent = i.ID
		}
		if i.SourceKey == "works:access-egress.barriers" {
			leaf = i.ID
		}
	}
	if parent == "" || leaf == "" {
		t.Fatal("missing access group")
	}
	stale := cw(p)
	p, e := s.RemoveCostItem(ctx, orgA, projectA, leaf, userA, cw(p))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.RemoveCostItem(ctx, orgA, projectA, parent, userA, stale); !errors.Is(e, store.ErrVersionConflict) {
		t.Fatal("stale delete", e)
	}
	r, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || r.Changed {
		t.Fatal("leaf resurrected", r, e)
	}
	for _, i := range r.Plan.Items {
		if i.ID == leaf {
			t.Fatal("deleted child returned")
		}
	}
	// Freeze preserves the remaining subtree even after its next draft is deleted.
	old := p.ID
	p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if e != nil {
		t.Fatal(e)
	}
	p, e = s.RemoveCostItem(ctx, orgA, projectA, parent, userA, cw(p))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "access-egress", Action: "new", Title: "Additional access scope"}); e != nil {
		t.Fatal(e)
	}
	r, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range r.Plan.Items {
		if i.SystemID == "access-egress" || i.ParentItemID == parent {
			t.Fatal("deleted group resurrected", i)
		}
	}
	frozen, e := s.ReadCostPlan(ctx, orgA, projectA, old)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, i := range frozen.Items {
		if i.ID == parent {
			found = true
		}
	}
	if !found {
		t.Fatal("baseline changed")
	}
	if _, e = s.RemoveCostItem(ctx, orgA, projectA, parent, userA, cw(frozen)); !errors.Is(e, store.ErrFrozenCostPlan) {
		t.Fatal("frozen delete", e)
	}
}

func TestCostBaselinePreparesMissingDefaults(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, e := s.BaselineCostPlan(ctx, orgA, projectA, userA, store.CostWrite{})
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Items) != 7 {
		t.Fatal("direct freeze omitted defaults", p)
	}
	totals, e := costs.Summarize(p, "overall")
	if e != nil || totals.Overall["budget"].Amount != nil {
		t.Fatal("unpriced defaults not unknown", totals, e)
	}
}

func TestDeletedManualFeeDoesNotReturnAsDefault(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	input := ci("My architect fee")
	input.LineKind = "fee"
	input.Category = ""
	input.PackageID = pkg.ID
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, input)
	if e != nil {
		t.Fatal(e)
	}
	id := p.Items[0].ID
	prepared, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	p, e = s.RemoveCostItem(ctx, orgA, projectA, id, userA, cw(prepared.Plan))
	if e != nil {
		t.Fatal(e)
	}
	prepared, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || prepared.Changed {
		t.Fatal("manual removal resurrected", e)
	}
	for _, i := range prepared.Plan.Items {
		if i.PackageID == pkg.ID {
			t.Fatal("default replaced deleted manual fee", i)
		}
	}
}

func TestUnpricedDefaultsPermitInitialTaxSettings(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	result, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	p := result.Plan
	settings := p.Settings
	settings.TaxBasis = "inc_tax"
	p, e = s.SaveCostSettings(ctx, orgA, projectA, userA, store.CostSettingsInput{CostWrite: cw(p), Settings: settings})
	if e != nil {
		t.Fatal("blank defaults locked settings", e)
	}
	p, e = s.PutCostValue(ctx, orgA, projectA, p.Items[0].ID, userA, "budget", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: cv("100.00")})
	if e != nil {
		t.Fatal(e)
	}
	settings.TaxBasis = "ex_tax"
	if _, e = s.SaveCostSettings(ctx, orgA, projectA, userA, store.CostSettingsInput{CostWrite: cw(p), Settings: settings}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("priced defaults allowed tax reinterpretation", e)
	}
}
