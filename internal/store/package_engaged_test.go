package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Who engages a consultant is the client's answer per package (owner
// decision, 13 October 2026): set, cleared back to not decided, never an
// invented value, never on a works package, never across orgs.
func TestPackageEngagedBy(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Structural", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if p.EngagedBy != "" {
		t.Fatalf("a new package is not decided: %q", p.EngagedBy)
	}
	set := func(version int64, v string) (procurement.Package, error) {
		return s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: version, EngagedBy: &v})
	}
	p, err = set(p.Version, procurement.EngagedByLead)
	if err != nil || p.EngagedBy != procurement.EngagedByLead {
		t.Fatalf("lead consultant engages: %+v %v", p, err)
	}
	if _, err := set(p.Version, "architect"); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("an unknown answer is refused: %v", err)
	}
	p, err = set(p.Version, "")
	if err != nil || p.EngagedBy != "" {
		t.Fatalf("clearing returns to not decided: %+v %v", p, err)
	}
	v := procurement.EngagedByClient
	if _, err := s.PatchPackage(ctx, orgB, projectA, p.ID, userB, procurement.PackagePatch{Version: p.Version, EngagedBy: &v}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org cannot answer: %v", err)
	}
	w, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "works", WorksScope: "trade", Title: "Roofing", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PatchPackage(ctx, orgA, projectA, w.ID, userA, procurement.PackagePatch{Version: w.Version, EngagedBy: &v}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("a works package is not engaged by a consultant: %v", err)
	}
}
