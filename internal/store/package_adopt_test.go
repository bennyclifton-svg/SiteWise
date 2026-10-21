package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Every proposed consultant is saved as a package (owner decision 3,
// 2026-10-21): a repeat adds none, a removed consultant is not brought
// back, and another org can do nothing.
func TestAdoptServiceSuggestions(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "industrial", "hdr.work_type": "new"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: map[string]string{"hdr.building_class": "site", "hdr.work_type": "project"}[key], Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	_, suggestions, err := s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(suggestions) == 0 {
		t.Fatalf("no proposals to adopt: %v %v", suggestions, err)
	}
	disciplines := []procurement.Discipline{{ID: "consultant.certifier", Label: "Certifier"}}
	if _, err := s.AdoptServiceSuggestions(ctx, orgB, projectA, userB, disciplines); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	n, err := s.AdoptServiceSuggestions(ctx, orgA, projectA, userA, disciplines)
	if err != nil || n != len(suggestions) {
		t.Fatalf("adopted %d of %d: %v", n, len(suggestions), err)
	}
	items, left, err := s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(left) != 0 || len(items) != n {
		t.Fatalf("after adoption: %d packages, %d proposals left, %v", len(items), len(left), err)
	}
	var certifier procurement.Package
	for _, p := range items {
		if p.Title == "Certifier" {
			certifier = p
		}
	}
	if certifier.DisciplineID != "consultant.certifier" || certifier.LifecycleStatus != "planned" {
		t.Fatalf("certifier saved as %+v", certifier)
	}
	if again, err := s.AdoptServiceSuggestions(ctx, orgA, projectA, userA, disciplines); err != nil || again != 0 {
		t.Fatalf("a repeat adopted %d: %v", again, err)
	}
	retired := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, certifier.ID, userA, procurement.PackagePatch{Retired: &retired, Version: certifier.Version}); err != nil {
		t.Fatal(err)
	}
	if again, err := s.AdoptServiceSuggestions(ctx, orgA, projectA, userA, disciplines); err != nil || again != 0 {
		t.Fatalf("a removed consultant came back: %d %v", again, err)
	}
}
