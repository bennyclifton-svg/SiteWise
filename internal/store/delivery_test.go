package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"sitewise/internal/delivery"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"sync"
	"testing"
)

func TestDeliveryCreateIsolationAndReferences(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineering", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if err != nil {
		t.Fatal(err)
	}
	date := "2026-11-01"
	input := store.DeliveryInput{Content: delivery.Content{Kind: "approval", Title: "Authority approval", PackageID: p.ID, StageID: p.Stages[0].ID, WorkItemID: w.ID, TargetDate: &date, Details: json.RawMessage(`{"authority":"Authority"}`)}, SourceRefs: []profile.Source{{DocumentID: docA, PassageID: passageA}}}
	if _, err := s.CreateDelivery(ctx, orgB, projectA, userB, input); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project %v", err)
	}
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userB, input); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor %v", err)
	}
	item, err := s.CreateDelivery(ctx, orgA, projectA, userA, input)
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != "not_submitted" || item.Version != 1 || item.Origin != "user" || item.TargetDate == nil || *item.TargetDate != date {
		t.Fatalf("bad record %+v", item)
	}
	items, err := s.ReadDelivery(ctx, orgA, projectA)
	if err != nil || len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("read %+v %v", items, err)
	}
	if _, err := s.ReadDelivery(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	retire := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("package reference %v", err)
	}
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, p.ID, p.Stages[0].ID, userA, procurement.StagePatch{Version: 1, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("stage reference %v", err)
	}
	if err := s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.mechanical.air-conditioning": nil}); !errors.Is(err, store.ErrInvalidWork) {
		t.Fatalf("work reference %v", err)
	}
	approved := "approved"
	item, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: 1, Status: &approved}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: 1, Retired: &retire}); err != nil {
		t.Fatal("closed delivery should allow retirement", err)
	}
	title := "Historical authority approval"
	item, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: 2, Title: &title}})
	if err != nil {
		t.Fatal("historical edit", err)
	}
	submitted := "submitted"
	if _, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: 3, Status: &submitted}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reopened retired reference: %v", err)
	}
}

func TestDeliveryConcurrentEditsRejectLostUpdate(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	item, err := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "milestone", Title: "Completion"}})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, title := range []string{"Practical completion", "Handover"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			_, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: 1, Title: &title}})
			results <- err
		}(title)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, store.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success %d conflicts %d", successes, conflicts)
	}
}
