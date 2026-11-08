package store_test

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// The building type a user picks in Profile ("House (Class 1a)") names
// the class, so the RFP must not call the class not established.
func TestRFPReadsClassFromBuildingType(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "residential", "hdr.subclass": "house", "hdr.work_type": "new"} {
		v := value
		scope := b.Catalog.KeyScope(key)
		if scope == "" {
			scope = "project"
		}
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Value: &v, State: "set", Scope: scope}); err != nil {
			t.Fatal(key, err)
		}
	}
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", DisciplineID: "consultant.architect", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || view.RFP == nil {
		t.Fatal(err)
	}
	labels := []string{}
	for _, g := range view.RFP.Groups {
		for _, c := range g.Controls {
			if c.Key == "parts:affected" {
				for _, o := range c.Options {
					labels = append(labels, o.Label)
				}
			}
		}
	}
	if len(labels) != 1 || !strings.Contains(labels[0], "(Class 1a)") {
		t.Fatalf("a house reads Class 1a: %v", labels)
	}
}
