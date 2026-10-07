package store_test

import (
	"context"
	"errors"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestWorkSplitRetireAtomicAndNested(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	parent, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Gas repairs"})
	if err != nil {
		t.Fatal(err)
	}
	request := works.SplitRequest{Version: parent.Version, Children: []works.SplitChild{{Title: "Pipework"}, {Title: "Valves"}}}
	bad := request
	bad.Children = []works.SplitChild{{Title: "Good"}, {Title: "Bad", PartID: "00000000-0000-4000-8000-000000000009"}}
	if _, err = s.SplitWorkItem(ctx, orgA, projectA, parent.ID, userA, bad); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign part: %v", err)
	}
	items, _ := s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].IsGroup {
		t.Fatal("failed split changed tree")
	}
	children, err := s.SplitWorkItem(ctx, orgA, projectA, parent.ID, userA, request)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		if child.ParentID != parent.ID || child.Action != parent.Action || child.SystemID != parent.SystemID || child.ReviewStatus != "accepted_for_planning" {
			t.Fatalf("inheritance %+v", child)
		}
	}
	if _, err = s.SplitWorkItem(ctx, orgA, projectA, parent.ID, userA, request); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale %v", err)
	}
	if _, err = s.SplitWorkItem(ctx, orgB, projectA, parent.ID, userA, request); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org %v", err)
	}
	var blocked *store.WorkReferencedError
	if err = s.RetireWorkItem(ctx, orgA, projectA, parent.ID, userA, parent.Version+1); !errors.As(err, &blocked) || len(blocked.Blockers) != 2 {
		t.Fatalf("parent blockers %v", err)
	}
	request.Version = children[0].Version
	nested, err := s.SplitWorkItem(ctx, orgA, projectA, children[0].ID, userA, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) != 2 || nested[0].ParentID != children[0].ID {
		t.Fatal("nested split")
	}
	if err = s.RetireWorkItem(ctx, orgA, projectA, children[1].ID, userA, children[1].Version); err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	for _, item := range items {
		if item.ID == children[1].ID {
			t.Fatal("rebuild revived retired child")
		}
	}
}

func TestWorkMovePreservesIdentityAndSurvivesEvidenceRebuild(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	parent, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Repair service"})
	if err != nil {
		t.Fatal(err)
	}
	system := "hydraulic.cold-water"
	moved, err := s.PatchWorkItem(ctx, orgA, projectA, parent.ID, userA, works.Patch{Version: parent.Version, SystemID: &system})
	if err != nil {
		t.Fatal(err)
	}
	facts := []store.StoredFact{}
	for _, sys := range []string{parent.SystemID, system} {
		facts = append(facts, store.StoredFact{PassageID: passageA, QuestionID: "sys." + sys + ".presence", Value: "included", Confidence: conf(.99), DecidedBy: "jev"})
	}
	if err = s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != moved.ID || items[0].SystemID != system || items[0].CoarseKey != part+"|"+system {
		t.Fatalf("moved identity %+v", items)
	}
	if err = s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope." + system: strPtr("out")}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope." + parent.SystemID: strPtr("in")}); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadWorks(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("scope picker overwrote moved item %+v", items)
	}
	for _, item := range items {
		if item.ID == moved.ID && (item.SystemID != system || item.Inclusion != "excluded") {
			t.Fatalf("moved scope corrupted %+v", item)
		}
	}
}

func TestWorkGroupScopeTogglePreservesOtherSystemsAndBlocksReset(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	parent, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Services"})
	if err != nil {
		t.Fatal(err)
	}
	children, err := s.SplitWorkItem(ctx, orgA, projectA, parent.ID, userA, works.SplitRequest{Version: 1, Children: []works.SplitChild{{Title: "Gas"}, {Title: "Water", SystemID: "hydraulic.cold-water"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": strPtr("out")}); err != nil {
		t.Fatal(err)
	}
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.ID == children[0].ID && item.Inclusion != "excluded" {
			t.Fatal("gas child ignored toggle")
		}
		if item.ID == children[1].ID && item.Inclusion != "included" {
			t.Fatal("unrelated child changed")
		}
	}
	if err = s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": nil}); !errors.Is(err, store.ErrInvalidWork) {
		t.Fatalf("reset orphaned children: %v", err)
	}
}
