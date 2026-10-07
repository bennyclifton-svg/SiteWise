package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"sync"
	"testing"
)

func TestDisciplineProposalAcceptance(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	records := b.Catalog.InterfaceConsequences()
	for i := range records {
		if records[i].ID == "ic.loads-investigate-supported" {
			records[i].Propose.Kind = "discipline"
			records[i].Propose.Label = "Structural services"
		}
	}
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	e, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
		t.Fatal(err)
	}
	ps, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var p store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.loads-investigate-supported" && candidate.InterfaceID == "if.plant-loads-structure" {
			p = candidate
			break
		}
	}
	if p.Kind != "discipline" {
		t.Fatal("missing synthetic discipline proposal")
	}
	if _, err := s.AcceptPackageProposal(ctx, orgB, projectA, p.Key, userB, p.InputsFingerprint); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project %v", err)
	}
	if _, err := s.AcceptPackageProposal(ctx, orgA, projectA, p.Key, userB, p.InputsFingerprint); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor %v", err)
	}
	if _, err := s.AcceptPackageProposal(ctx, orgA, projectA, p.Key, userA, "stale"); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale %v", err)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := s.AcceptPackageProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint)
			if err != nil {
				t.Error(err)
				return
			}
			if got.Kind != "services" || got.ReviewStatus != "accepted_for_planning" || got.LifecycleStatus != "planned" || len(got.Stages) != 4 {
				t.Errorf("invalid package %+v", got)
			}
			ids <- got.ID
		}()
	}
	wg.Wait()
	close(ids)
	id, count := "", 0
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate package")
		}
		id = got
		count++
	}
	if count != 4 {
		t.Fatal("missing accepts", count)
	}
	packages, err := s.ReadPackages(ctx, orgA, projectA)
	if err != nil || len(packages) != 1 {
		t.Fatalf("packages %+v %v", packages, err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
		t.Fatal(err)
	}
	ps, err = s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range ps {
		if got.Key == p.Key && (got.Decision == nil || got.Decision.CreatedRecordType != "package" || got.Decision.CreatedRecordID != id || got.Decision.Version != 1) {
			t.Fatal("decision lost", got)
		}
	}
	if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 1); err != nil {
		t.Fatal(err)
	}
	packages, err = s.ReadPackages(ctx, orgA, projectA)
	if err != nil || len(packages) != 0 {
		t.Fatal("undo did not retire package", packages, err)
	}
	restored, err := s.AcceptPackageProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint)
	if err != nil || restored.ID != id || restored.Version != 3 || len(restored.Stages) != 4 {
		t.Fatalf("reaccept %+v %v", restored, err)
	}
	label := "Edited stage"
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, id, restored.Stages[0].ID, userA, procurement.StagePatch{Version: 1, Label: &label}); err != nil {
		t.Fatal(err)
	}
	if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 3); !errors.Is(err, store.ErrProposalUndoBlocked) {
		t.Fatalf("edited stage undo %v", err)
	}
}
