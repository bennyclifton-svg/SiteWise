package store_test

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/store"
)

// The brief's contingency, owner-supplied items and budget exclusions land
// in the cost plan; an edited figure stays the PM's, and a repeat writes
// nothing (docs/plans/2026-10-21-source-to-cost-plan-design.md).
func TestBriefFiguresInTheCostPlan(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{
		"proj.contingency_stated":  "$120,000",
		"proj.owner_supply_stated": "pendant lights (allowance $8,000), appliances package (allowance $22,000)",
		"proj.exclusions_stated":   "landscaping, loose furniture, and authority fees",
	} {
		v := value
		if _, e := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: "project", Value: &v}); e != nil {
			t.Fatal(key, e)
		}
	}
	r, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	find := func(p costs.Plan, key string) costs.Item {
		for _, i := range p.Items {
			if i.SourceKey == key {
				return i
			}
		}
		t.Fatalf("no line %s", key)
		return costs.Item{}
	}
	amount := func(i costs.Item) string {
		if v := i.Values["budget"]; v.Amount != nil {
			return string(*v.Amount)
		}
		return ""
	}
	contingency := find(r.Plan, "pw:construction-contingency")
	if amount(contingency) != "120000.00" {
		t.Fatalf("contingency %+v", contingency.Values)
	}
	lights, appliances := find(r.Plan, "brief:owner-supply:pendant-lights"), find(r.Plan, "brief:owner-supply:appliances-package")
	if lights.Label != "Owner supply: pendant lights" || amount(lights) != "8000.00" || amount(appliances) != "22000.00" {
		t.Fatalf("owner supply %+v %+v", lights, appliances)
	}
	if r.Totals.SectionItems[lights.ID] == "construction" {
		t.Fatal("an owner supply counts against the construction budget")
	}
	if ex := find(r.Plan, "pw:exclusions"); !strings.Contains(ex.Label, "loose furniture") {
		t.Fatalf("exclusions label %q", ex.Label)
	}
	if again, e := s.PrepareCostPlan(ctx, orgA, projectA, userA); e != nil || again.Changed {
		t.Fatalf("a repeat rewrote the brief figures: %v", e)
	}
	// The PM's own contingency stays.
	mine := costs.Money("150000.00")
	p := r.Plan
	if p, e = s.PutCostValue(ctx, orgA, projectA, contingency.ID, userA, "budget", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: costs.Value{Amount: &mine, ValueState: "known", Origin: "user", Meaning: "allowance", Version: contingency.Values["budget"].Version}}); e != nil {
		t.Fatal(e)
	}
	r, e = s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil || amount(find(r.Plan, "pw:construction-contingency")) != "150000.00" {
		t.Fatalf("edited contingency overwritten: %v", e)
	}
}
