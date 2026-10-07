package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestPackageScopeIsolationEditsAndRetirement(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Design", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if err != nil {
		t.Fatal(err)
	}
	in := store.ScopeInput{ScopeContent: procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: w.ID, Role: "design", UserText: "Design the plant", StageID: p.Stages[0].ID, Inclusion: "included"}}
	if _, err := s.CreatePackageScope(ctx, orgB, projectA, p.ID, userB, in); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project: %v", err)
	}
	if _, err := s.CreatePackageScope(ctx, orgA, projectA, p.ID, userB, in); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor: %v", err)
	}
	item, err := s.CreatePackageScope(ctx, orgA, projectA, p.ID, userA, in)
	if err != nil {
		t.Fatal(err)
	}
	if item.Version != 1 || item.Origin != "user" || item.ReviewStatus != "accepted_for_planning" {
		t.Fatalf("bad provenance: %+v", item)
	}
	if _, err := s.CreatePackageScope(ctx, orgA, projectA, p.ID, userA, in); err == nil {
		t.Fatal("duplicate role accepted")
	}
	retire := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("package retire: %v", err)
	}
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, p.ID, p.Stages[0].ID, userA, procurement.StagePatch{Version: 1, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("stage retire: %v", err)
	}
	kind := "supply"
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Kind: &kind}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("supply design: %v", err)
	}
	text := "Updated design brief"
	updated, err := s.PatchPackageScope(ctx, orgA, projectA, p.ID, item.ID, userA, store.ScopePatch{Version: 1, UserText: &text})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.UserText != text {
		t.Fatalf("bad update: %+v", updated)
	}
	if _, err := s.PatchPackageScope(ctx, orgA, projectA, p.ID, item.ID, userA, store.ScopePatch{Version: 1, UserText: &text}); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale edit: %v", err)
	}
	items, err := s.ReadPackageScope(ctx, orgA, projectA, p.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("read %+v %v", items, err)
	}
	if _, err := s.PatchPackageScope(ctx, orgA, projectA, p.ID, item.ID, userA, store.ScopePatch{Version: 2, Retired: &retire}); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadPackageScope(ctx, orgA, projectA, p.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("retired read %+v %v", items, err)
	}
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, p.ID, p.Stages[0].ID, userA, procurement.StagePatch{Version: 1, Retired: &retire}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Retired: &retire}); err != nil {
		t.Fatal(err)
	}
}

func TestPackageScopeSourceSnapshots(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Evidence", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	in := store.ScopeInput{ScopeContent: procurement.ScopeContent{ItemKind: "obligation", UserText: "Review the evidence", Inclusion: "included"}, SourceRefs: []profile.Source{{DocumentID: docA, PassageID: passageA, Filename: "forged.pdf", FileSHA256: "forged", Excerpt: "forged"}}}
	item, err := s.CreatePackageScope(ctx, orgA, projectA, p.ID, userA, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.SourceRefs) != 1 || item.SourceRefs[0].Filename == "forged.pdf" || item.SourceRefs[0].FileSHA256 == "forged" || item.SourceRefs[0].Excerpt == "forged" {
		t.Fatalf("untrusted source used: %+v", item.SourceRefs)
	}
	in.SourceRefs[0].DocumentID = docB
	if _, err := s.CreatePackageScope(ctx, orgA, projectA, p.ID, userA, in); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign source: %v", err)
	}
	if _, err := rawPool(t).Exec(ctx, `DELETE FROM passages WHERE org_id=$1 AND id=$2`, orgA, passageA); err != nil {
		t.Fatal(err)
	}
	wording := "Review saved evidence"
	updated, err := s.PatchPackageScope(ctx, orgA, projectA, p.ID, item.ID, userA, store.ScopePatch{Version: 1, UserText: &wording})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.SourceRefs) != 1 || updated.SourceRefs[0] != item.SourceRefs[0] {
		t.Fatal("snapshot lost on unrelated edit")
	}
}
