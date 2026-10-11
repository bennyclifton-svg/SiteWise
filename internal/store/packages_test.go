package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func TestPackageComplexitySuggestionsUseWholeSiteAppliedFacts(t *testing.T) {
	ctx := context.Background()
	s, _, whole := workStore(t)
	for key, value := range map[string]string{"det.heritage_status": "local_item", "det.bal": "BAL-40"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, whole, userA, key, store.UserWrite{Scope: "site", Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	part, err := s.CreatePart(ctx, orgA, projectA, "Other building", "building", "")
	if err != nil {
		t.Fatal(err)
	}
	value := "protected_habitat"
	if _, err := s.SetUserValue(ctx, orgA, projectA, part.ID, userA, "hdr.cond.environmental_sensitivity", store.UserWrite{Scope: "site", Value: &value}); err != nil {
		t.Fatal(err)
	}
	_, suggestions, err := s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(suggestions) != 2 {
		t.Fatalf("whole-site suggestions: %+v %v", suggestions, err)
	}
	for _, p := range suggestions {
		if !p.Draft || len(p.MatchedFields) != 1 {
			t.Fatal(p)
		}
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE profile_rows SET value_state='unknown' WHERE org_id=$1::uuid AND project_id=$2::uuid AND key='det.heritage_status'`, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	_, suggestions, err = s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(suggestions) != 1 || suggestions[0].Title != "Bushfire" {
		t.Fatalf("unknown used: %+v %v", suggestions, err)
	}
	if _, _, err := s.ReadPackageOverview(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project: %v", err)
	}
}

func TestPackageCreateStagesAndIsolation(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p := procurement.Package{Kind: "services", Title: "Structural engineering", Novation: "option", LifecycleStatus: "planned"}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userB, p); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor: %v", err)
	}
	if _, err := s.CreatePackage(ctx, orgB, projectA, userB, p); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project: %v", err)
	}
	created, err := s.CreatePackage(ctx, orgA, projectA, userA, p)
	if err != nil {
		t.Fatal(err)
	}
	if created.ReviewStatus != "accepted_for_planning" || created.Version != 1 || len(created.Stages) != 4 {
		t.Fatalf("created %+v", created)
	}
	pre, post := 0, 0
	for _, stage := range created.Stages {
		if stage.NovationPhase == "pre" {
			pre++
		}
		if stage.NovationPhase == "post" {
			post++
		}
		if stage.PackageID != created.ID || stage.ID == "" {
			t.Fatal(stage)
		}
	}
	if pre != 2 || post != 2 {
		t.Fatalf("novation defaults %d/%d", pre, post)
	}
	got, err := s.ReadPackages(ctx, orgA, projectA)
	if err != nil || len(got) != 1 || got[0].ID != created.ID || len(got[0].Stages) != 4 {
		t.Fatalf("read %+v %v", got, err)
	}
	if _, err := s.ReadPackages(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	bad := p
	bad.Stages = []procurement.Stage{{StageID: "unknown", Label: "Unknown", NovationPhase: "none"}}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, bad); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("invalid stage %v", err)
	}
	got, err = s.ReadPackages(ctx, orgA, projectA)
	if err != nil || len(got) != 1 {
		t.Fatal("failed stage left package behind", got, err)
	}
	bad = p
	bad.Kind = "works"
	bad.WorksScope = "trade"
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, bad); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("novation on works %v", err)
	}
	p.Novation = "no"
	plain, err := s.CreatePackage(ctx, orgA, projectA, userA, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range plain.Stages {
		if stage.NovationPhase != "none" {
			t.Fatal("unrequested novation", stage)
		}
	}
	// Both clients read version one; the project lock permits exactly one edit.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, title := range []string{"Client A", "Client B"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			_, err := s.PatchPackage(ctx, orgA, projectA, plain.ID, userA, procurement.PackagePatch{Version: 1, Title: &title})
			results <- err
		}(title)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, store.ErrVersionConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent edits: %d successes, %d conflicts", success, conflict)
	}
}

func TestPackageSuggestionsUseEligibleProfileRows(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "industrial", "hdr.work_type": "new"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: map[string]string{"hdr.building_class": "site", "hdr.work_type": "project"}[key], Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	_, suggestions, err := s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(suggestions) != 14 {
		t.Fatalf("baseline %+v %v", suggestions, err)
	}
	for _, sug := range suggestions {
		if sug.Title == "Services Engineer" {
			t.Fatal("services suggested as one engineer")
		}
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE profile_rows SET value_state='unknown' WHERE org_id=$1::uuid AND project_id=$2::uuid AND key='hdr.work_type'`, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	// Without an applied work type only the scope still suggests consultants.
	_, suggestions, err = s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil || len(suggestions) == 0 {
		t.Fatalf("scope additions lost: %+v %v", suggestions, err)
	}
	for _, sug := range suggestions {
		if sug.MatchedFields["scope"] == "" || len(sug.WorkTypes) != 0 {
			t.Fatalf("unknown value used: %+v", sug)
		}
	}
}

// The PM removes a suggested consultant; it stays removed for that project,
// a second removal is harmless, and another org can neither see nor remove it.
func TestDismissPackageSuggestion(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "industrial", "hdr.work_type": "new"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: map[string]string{"hdr.building_class": "site", "hdr.work_type": "project"}[key], Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	const key = "default:services:Certifier"
	has := func() bool {
		_, suggestions, err := s.ReadPackageOverview(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		for _, sug := range suggestions {
			if sug.Key == key {
				return true
			}
		}
		return false
	}
	if !has() {
		t.Fatal("certifier not suggested")
	}
	if err := s.DismissPackageSuggestion(ctx, orgB, projectA, userB, key); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	if !has() {
		t.Fatal("wrong org removed a suggestion")
	}
	if err := s.DismissPackageSuggestion(ctx, orgA, projectA, userA, "default:services:Nobody"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown key: %v", err)
	}
	for range 2 {
		if err := s.DismissPackageSuggestion(ctx, orgA, projectA, userA, key); err != nil {
			t.Fatal(err)
		}
	}
	if has() {
		t.Fatal("dismissed suggestion came back")
	}
}
