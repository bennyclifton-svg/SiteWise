package store_test

import (
	"context"
	"testing"

	"sitewise/internal/store"
)

// Walsh builder email, 2026-10-27: the builder's preliminary estimate sits
// beside the budget in the cost plan and changes no line.
func TestPreliminaryEstimateBesideTheBudget(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	before, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	if before.Plan.Estimate != nil {
		t.Fatalf("estimate without a stated one: %+v", before.Plan.Estimate)
	}
	v := "$880,000 – $980,000"
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "proj.preliminary_estimate", store.UserWrite{Scope: "project", Value: &v}); err != nil {
		t.Fatal(err)
	}
	r, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	e := r.Plan.Estimate
	if e == nil || e.Low != "880000.00" || e.High != "980000.00" {
		t.Fatalf("estimate %+v", e)
	}
	if r.Changed {
		t.Fatal("an estimate changed a cost line")
	}
}
