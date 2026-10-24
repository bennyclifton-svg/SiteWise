package store_test

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// The profile's construction budget is spread over the construction lines
// by the split its building type picks (owner decisions 1 and 2,
// docs/plans/2026-10-21-source-to-cost-plan-design.md): the shares add to
// the budget, an edited line keeps its figure and moves no other, a new
// budget refreshes only unedited shares, and a cleared line stays unknown.
func TestConstructionBudgetSpread(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	systems := []string{"site", "substructure", "structure", "envelope", "interiors", "mechanical", "hydraulic", "electrical"}
	for _, sys := range systems {
		if _, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: sys, Action: "new", Title: sys}); e != nil {
			t.Fatal(e)
		}
	}
	set := func(key, scope, value string) {
		t.Helper()
		if _, e := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: scope, Value: &value}); e != nil {
			t.Fatal(key, e)
		}
	}
	// No class or work type yet: the budget shows, nothing is spread.
	set("proj.budget_stated", "project", "$1,850,000")
	r, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || r.Plan.Budget == nil || !r.Plan.Budget.Known || r.Plan.Budget.NoSplit == "" {
		t.Fatalf("budget without a split: %+v %v", r.Plan.Budget, e)
	}
	for _, i := range construction(r.Plan) {
		if v, ok := i.Values["budget"]; ok && v.ValueState == "known" {
			t.Fatalf("spread with no building type: %s %+v", i.Label, v)
		}
	}
	set("hdr.building_class", "site", "residential")
	set("hdr.subclass", "site", "house")
	set("hdr.work_type", "project", "new")
	r, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || r.Plan.Budget.Split != "New house" {
		t.Fatalf("split: %+v %v", r.Plan.Budget, e)
	}
	lines := construction(r.Plan)
	if len(lines) < len(systems) || sum(lines) != "1850000.00" {
		t.Fatalf("%d lines sum to %s", len(lines), sum(lines))
	}
	for _, i := range lines {
		v := i.Values["budget"]
		if v.Origin != "assumption" || !strings.Contains(v.Rationale, "new house split") {
			t.Fatalf("%s: %+v", i.Label, v)
		}
	}
	if d := r.Totals.Budget; d == nil || d.Difference == nil || *d.Difference != "0.00" {
		t.Fatalf("budget check %+v", d)
	}
	if again, e := s.PrepareCostPlan(ctx, orgA, projectA, userA); e != nil || again.Changed {
		t.Fatalf("a second prepare rewrote the spread: %v", e)
	}

	// The PM bumps the envelope and clears the hydraulic line.
	envelope, hydraulic := line(lines, "works:envelope"), line(lines, "works:hydraulic")
	before := map[string]costs.Money{}
	for _, i := range lines {
		before[i.SourceKey] = *i.Values["budget"].Amount
	}
	p := r.Plan
	put := func(i costs.Item, v costs.Value) {
		t.Helper()
		v.Version = i.Values["budget"].Version
		var e error
		if p, e = s.PutCostValue(ctx, orgA, projectA, i.ID, userA, "budget", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: v}); e != nil {
			t.Fatal(e)
		}
	}
	bumped := costs.Money("500000.00")
	put(envelope, costs.Value{Amount: &bumped, ValueState: "known", Origin: "user", Meaning: "allowance"})
	put(hydraulic, costs.Value{ValueState: "unknown", Origin: "user", Meaning: "allowance"})
	r, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range construction(r.Plan) {
		v := i.Values["budget"]
		switch i.SourceKey {
		case "works:envelope":
			if *v.Amount != bumped {
				t.Fatal("the edited line was overwritten", v)
			}
		case "works:hydraulic":
			if v.ValueState != "unknown" {
				t.Fatal("a cleared line was refilled", v)
			}
		default:
			if *v.Amount != before[i.SourceKey] {
				t.Fatalf("%s moved from %s to %s", i.Label, before[i.SourceKey], *v.Amount)
			}
		}
	}

	// A new budget refreshes the unedited shares only.
	set("proj.budget_stated", "project", "$2 million excluding GST")
	r, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || r.Plan.Budget.Stated.GST != "ex" {
		t.Fatal(e, r.Plan.Budget)
	}
	for _, i := range construction(r.Plan) {
		v := i.Values["budget"]
		switch i.SourceKey {
		case "works:envelope":
			if *v.Amount != bumped {
				t.Fatal("a new budget overwrote the edited line", v)
			}
		case "works:hydraulic":
			if v.ValueState != "unknown" {
				t.Fatal("a new budget refilled a cleared line", v)
			}
		default:
			if costs.Compare(*v.Amount, before[i.SourceKey]) <= 0 {
				t.Fatalf("%s did not follow the new budget: %s", i.Label, *v.Amount)
			}
		}
	}
}

func construction(p costs.Plan) []costs.Item {
	var out []costs.Item
	for _, i := range p.Items {
		if i.LineKind == "works" && strings.HasPrefix(i.SourceKey, "works:") {
			out = append(out, i)
		}
	}
	return out
}

func line(items []costs.Item, key string) costs.Item {
	for _, i := range items {
		if i.SourceKey == key {
			return i
		}
	}
	return costs.Item{}
}

func sum(items []costs.Item) costs.Money {
	total := costs.Money("0.00")
	for _, i := range items {
		if v := i.Values["budget"]; v.ValueState == "known" && v.Amount != nil {
			total = costs.Add(total, *v.Amount)
		}
	}
	return total
}
