package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

func TestPackageRemovalWithCostLinesIsAtomic(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove", true: "scope blocks"}[blocked], func(t *testing.T) {
			ctx := context.Background()
			s, _, _ := workStore(t)
			pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Structural Design", LifecycleStatus: "planned"})
			if err != nil {
				t.Fatal(err)
			}
			ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
			if err != nil {
				t.Fatal(err)
			}
			plan := ready.Plan
			var line string
			for _, i := range plan.Items {
				if i.PackageID == pkg.ID {
					line = i.ID
				}
			}
			if line == "" {
				t.Fatal("missing fee")
			}
			plan, err = s.PutCostValue(ctx, orgA, projectA, line, userA, "budget", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: cv("100.00")})
			if err != nil {
				t.Fatal(err)
			}
			frozenID := plan.ID
			plan, err = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(plan))
			if err != nil {
				t.Fatal(err)
			}
			retire := true
			patch := procurement.PackagePatch{Version: pkg.Version, Retired: &retire, RemoveCostLines: true, CostPlanID: plan.ID, CostPlanVersion: plan.Version}
			if _, err = s.PatchPackage(ctx, orgB, projectA, pkg.ID, userB, patch); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("foreign removal", err)
			}
			patch.CostPlanVersion--
			if _, err = s.PatchPackage(ctx, orgA, projectA, pkg.ID, userA, patch); !errors.Is(err, store.ErrVersionConflict) && !errors.Is(err, store.ErrInvalidPackage) {
				t.Fatal("stale removal", err)
			}
			patch.CostPlanVersion = plan.Version
			if blocked {
				_, err = s.CreatePackageScope(ctx, orgA, projectA, pkg.ID, userA, store.ScopeInput{ScopeContent: procurement.ScopeContent{ItemKind: "obligation", UserText: "Review structure", Inclusion: "included"}})
				if err != nil {
					t.Fatal(err)
				}
			}
			start := time.Now()
			_, err = s.PatchPackage(ctx, orgA, projectA, pkg.ID, userA, patch)
			elapsed := time.Since(start)
			t.Logf("combined removal: %s (p90 budget 250ms)", elapsed)
			if elapsed > 250*time.Millisecond {
				t.Fatal("removal exceeded budget", elapsed)
			}
			if blocked && !errors.Is(err, store.ErrInvalidPackage) {
				t.Fatal(err)
			}
			if !blocked && err != nil {
				t.Fatal(err)
			}
			current, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, i := range current.Plan.Items {
				if i.ID == line {
					found = true
				}
			}
			if found != blocked {
				t.Fatal("partial deletion or resurrected fee", found)
			}
			frozen, err := s.ReadCostPlan(ctx, orgA, projectA, frozenID)
			if err != nil {
				t.Fatal(err)
			}
			found = false
			for _, i := range frozen.Items {
				if i.ID == line {
					found = true
					if i.Values["budget"].Amount == nil || *i.Values["budget"].Amount != "100.00" {
						t.Fatal("baseline amount changed")
					}
				}
			}
			if !found {
				t.Fatal("baseline removed")
			}
		})
	}
}

func TestPackageCostRemovalLatency(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	samples := []time.Duration{}
	for n := 0; n < 10; n++ {
		pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Design", LifecycleStatus: "planned"})
		if err != nil {
			t.Fatal(err)
		}
		ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
		if err != nil {
			t.Fatal(err)
		}
		retire := true
		start := time.Now()
		_, err = s.PatchPackage(ctx, orgA, projectA, pkg.ID, userA, procurement.PackagePatch{Version: pkg.Version, Retired: &retire, RemoveCostLines: true, CostPlanID: ready.Plan.ID, CostPlanVersion: ready.Plan.Version})
		samples = append(samples, time.Since(start))
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	t.Logf("combined removal p50=%s p90=%s budget=100ms/250ms", samples[4], samples[8])
	if samples[4] > 100*time.Millisecond || samples[8] > 250*time.Millisecond {
		t.Fatal("combined removal exceeds budget")
	}
}
