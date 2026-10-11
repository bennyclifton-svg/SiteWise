package store_test

import (
	"context"
	"errors"
	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"sort"
	"testing"
	"time"
)

func TestCostAutofillLifecycle(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	pkg, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	for _, sys := range []string{"mechanical.air-conditioning", "access-egress.barriers", "access-egress.stairs-and-ramps"} {
		if _, e = s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: sys, Action: "new", Title: sys}); e != nil {
			t.Fatal(e)
		}
	}
	p, e := s.ReadCostPlan(ctx, orgA, projectA, "")
	if e != nil || p.Status != "unavailable" || len(p.Suggestions) < 10 {
		t.Fatal(p, e)
	}
	var count int
	if e = rawPool(t).QueryRow(ctx, `SELECT count(*) FROM cost_plan_versions WHERE org_id=$1`, orgA).Scan(&count); e != nil || count != 0 {
		t.Fatal("read wrote plan", count, e)
	}
	key := "fee:" + pkg.ID
	if _, e = s.AcceptCostSuggestions(ctx, orgB, projectA, userB, key, store.CostSuggestionInput{}); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("foreign accept", e)
	}
	if _, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, key, store.CostSuggestionInput{Budget: cm("invalid")}); e == nil {
		t.Fatal("invalid budget accepted")
	}
	p, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, key, store.CostSuggestionInput{Budget: cm("50000")})
	if e != nil || len(p.Items) != 1 || *p.Items[0].Values["budget"].Amount != "50000.00" {
		t.Fatal(p, e)
	}
	if _, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, key, store.CostSuggestionInput{}); !errors.Is(e, store.ErrVersionConflict) {
		t.Fatal("stale retry", e)
	}
	oldVersion := p.Version
	p, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, key, store.CostSuggestionInput{CostWrite: cw(p), Budget: cm("1")})
	if e != nil || p.Version != oldVersion || *p.Items[0].Values["budget"].Amount != "50000.00" {
		t.Fatal("retry altered budget", p, e)
	}
	p, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, "works:access-egress.barriers", store.CostSuggestionInput{CostWrite: cw(p)})
	if e != nil || len(p.Items) != 3 {
		t.Fatal("child first", p, e)
	}
	p, e = s.DismissCostSuggestion(ctx, orgA, projectA, userA, "pw:escalation", false, cw(p))
	if e != nil || len(p.Dismissed) != 1 {
		t.Fatal(p, e)
	}
	expected := len(p.Items) + len(p.Suggestions)
	p, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, "", store.CostSuggestionInput{CostWrite: cw(p)})
	if e != nil || len(p.Suggestions) != 0 || len(p.Items) != expected {
		t.Fatal("accept all", p, e)
	}
	base := p.ID
	p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if e != nil || len(p.Suggestions) != 0 {
		t.Fatal("clone lost source", p, e)
	}
	frozen, e := s.ReadCostPlan(ctx, orgA, projectA, base)
	if e != nil || len(frozen.Suggestions) != 0 {
		t.Fatal(frozen, e)
	}
	if _, e = s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "works", WorksScope: "head_contract", Title: "Main Works", LifecycleStatus: "planned"}); e != nil {
		t.Fatal(e)
	}
	live, e := s.ReadCostPlan(ctx, orgA, projectA, "")
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range live.Items {
		if i.LineKind == "works" && i.ResolvedCategory != "Main Works" {
			t.Fatal("live mapping did not update", i)
		}
	}
	frozen, e = s.ReadCostPlan(ctx, orgA, projectA, base)
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range frozen.Items {
		if i.LineKind == "works" && (i.ResolvedCategory != "Head contractor" || i.ResolvedPackageID != "" || len(i.ScopeWorkItems) == 0) {
			t.Fatal("frozen display not captured", i)
		}
	}
	if _, e = s.AcceptCostSuggestions(ctx, orgA, projectA, userA, "", store.CostSuggestionInput{CostWrite: cw(frozen)}); !errors.Is(e, store.ErrFrozenCostPlan) {
		t.Fatal("frozen", e)
	}
	var feeID string
	for _, i := range p.Items {
		if i.SourceKey == key {
			feeID = i.ID
		}
	}
	p, e = s.RemoveCostItem(ctx, orgA, projectA, feeID, userA, cw(p))
	if e != nil || len(p.Suggestions) != 0 {
		t.Fatal("removed line came back", p, e)
	}
	p, e = s.DismissCostSuggestion(ctx, orgA, projectA, userA, key, true, cw(p))
	if e != nil || len(p.Suggestions) != 1 {
		t.Fatal("restore", p, e)
	}
	in := ci("Measured")
	in.CostWrite = cw(p)
	in.Values = nil
	in.Quantity = cm("10")
	in.Rate = cm("120")
	in.Unit = "m2"
	in.RateBasis = "User supplied"
	in.CalculateBudget = true
	p, e = s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range p.Items {
		if i.Label == "Measured" && (i.Values["budget"].Amount == nil || *i.Values["budget"].Amount != "1200.00") {
			t.Fatal("atomic calculation", i)
		}
	}
	if e = s.DeleteOrg(ctx, orgA); e != nil {
		t.Fatal("tenant cascade", e)
	}
}

func TestCostAutofillLatency(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	for _, sys := range b.Catalog.BuildingOrderSystems() {
		if _, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: sys.ID, Action: "new", Title: sys.Label}); e != nil {
			t.Fatal(e)
		}
	}
	for n := 0; n < 8; n++ {
		if _, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: string(rune('A' + n)), LifecycleStatus: "planned"}); e != nil {
			t.Fatal(e)
		}
	}
	samples := map[string][]time.Duration{}
	for n := 0; n < 20; n++ {
		start := time.Now()
		p, e := s.ReadCostPlan(ctx, orgA, projectA, "")
		samples["costs_read"] = append(samples["costs_read"], time.Since(start))
		if e != nil {
			t.Fatal(e)
		}
		start = time.Now()
		prepared, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
		samples["costs_prepare"] = append(samples["costs_prepare"], time.Since(start))
		if e != nil {
			t.Fatal(e)
		}
		p = prepared.Plan
		start = time.Now()
		noop, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
		samples["costs_prepare_noop"] = append(samples["costs_prepare_noop"], time.Since(start))
		if e != nil || noop.Changed || noop.Plan.Version != p.Version {
			t.Fatal("no-op prepare", e)
		}
		start = time.Now()
		p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
		samples["costs_freeze"] = append(samples["costs_freeze"], time.Since(start))
		if e != nil {
			t.Fatal(e)
		}
		// Remove children before parents, then restore dismissals for another real run.
		sort.Slice(p.Items, func(i, j int) bool { return p.Items[i].ParentItemID > p.Items[j].ParentItemID })
		items := append([]costs.Item(nil), p.Items...)
		for _, i := range items {
			start = time.Now()
			p, e = s.RemoveCostItem(ctx, orgA, projectA, i.ID, userA, cw(p))
			samples["costs_delete"] = append(samples["costs_delete"], time.Since(start))
			if e != nil {
				t.Fatal(e)
			}
		}
		dismissed := append([]costs.Suggestion(nil), p.Dismissed...)
		for _, d := range dismissed {
			p, e = s.DismissCostSuggestion(ctx, orgA, projectA, userA, d.Key, true, cw(p))
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	for name, ds := range samples {
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		p50, p90 := ds[len(ds)/2], ds[len(ds)*9/10]
		a, b := 100*time.Millisecond, 250*time.Millisecond
		if name == "costs_prepare" || name == "costs_freeze" {
			a, b = 250*time.Millisecond, 500*time.Millisecond
		}
		t.Logf("%s p50=%s p90=%s budget=%s/%s", name, p50, p90, a, b)
		if p50 > a || p90 > b {
			t.Fatal("budget exceeded", name)
		}
	}
}
