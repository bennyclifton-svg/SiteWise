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

func TestScopeProposalAcceptanceUndoAndRetries(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	for i := range b.Catalog.InterfaceConsequences() {
		r := &b.Catalog.InterfaceConsequences()[i]
		if r.ID == "ic.loads-investigate-supported" {
			r.Propose.Kind = "obligation"
			r.Propose.Label = "Coordinate structural review"
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
	var proposal store.ProposalView
	for _, p := range ps {
		if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" {
			proposal = p
			break
		}
	}
	if proposal.Key == "" {
		t.Fatal("missing obligation")
	}
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineering", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	choice := store.ScopeAcceptance{PackageID: pkg.ID, StageID: pkg.Stages[0].ID}
	if _, err := s.AcceptScopeProposal(ctx, orgB, projectA, proposal.Key, userB, proposal.InputsFingerprint, choice); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign %v", err)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := s.AcceptScopeProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint, choice)
			if err != nil {
				t.Error(err)
				return
			}
			if !item.Provisional || item.Origin != "calculation" || item.ReviewStatus != "accepted_for_planning" {
				t.Errorf("bad accepted scope %+v", item)
			}
			ids <- item.ID
		}()
	}
	wg.Wait()
	close(ids)
	id := ""
	count := 0
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate scope")
		}
		id = got
		count++
	}
	if count != 4 {
		t.Fatal("missing accepts", count)
	}
	changed := choice
	changed.StageID = ""
	if _, err := s.AcceptScopeProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint, changed); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("changed retry %v", err)
	}
	if err := s.UndoProposalDecision(ctx, orgA, projectA, proposal.Key, userA, 1); err != nil {
		t.Fatal(err)
	}
	items, err := s.ReadPackageScope(ctx, orgA, projectA, pkg.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("undo %+v %v", items, err)
	}
	restored, err := s.AcceptScopeProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint, choice)
	if err != nil || restored.ID != id || restored.Version != 3 {
		t.Fatalf("reaccept %+v %v", restored, err)
	}
	// A saved draft is use even when nobody has changed the accepted scope.
	report, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshReport(ctx, orgA, report.ID, userA, "test-build", true); err != nil {
		t.Fatal(err)
	}
	if err := s.UndoProposalDecision(ctx, orgA, projectA, proposal.Key, userA, 3); !errors.Is(err, store.ErrProposalUndoBlocked) {
		t.Fatalf("report-referenced undo %v", err)
	}
	// Remove the draft fixture so the next assertion independently checks edits.
	if _, err := rawPool(t).Exec(ctx, `DELETE FROM reports WHERE org_id=$1 AND id=$2`, orgA, report.ID); err != nil {
		t.Fatal(err)
	}
	text := "Edited obligation"
	if _, err := s.PatchPackageScope(ctx, orgA, projectA, pkg.ID, id, userA, store.ScopePatch{Version: 3, UserText: &text}); err != nil {
		t.Fatal(err)
	}
	if err := s.UndoProposalDecision(ctx, orgA, projectA, proposal.Key, userA, 3); !errors.Is(err, store.ErrProposalUndoBlocked) {
		t.Fatalf("edited undo %v", err)
	}
	// An accepted retry survives projection removal and subsequent user edits.
	if _, err := rawPool(t).Exec(ctx, `DELETE FROM proposals WHERE org_id=$1 AND project_id=$2`, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	retry, err := s.AcceptScopeProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint, choice)
	if err != nil || retry.UserText != text || retry.ID != id {
		t.Fatalf("durable retry %+v %v", retry, err)
	}
}
