package store_test

import (
	"context"
	"errors"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"strings"
	"testing"
)

func TestPhysicalProposalRequiresExplicitActionAndKeepsChoice(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.local-exhaust", Action: "alter", Title: "Alter exhaust"}); err != nil {
		t.Fatal(err)
	}
	evaluator, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	ps, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var p store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.penetrates-make-good" && candidate.TargetSystemID != "" && candidate.TargetPartID != "" {
			p = candidate
			break
		}
	}
	if p.Key == "" {
		t.Fatal("missing make-good proposal")
	}
	if _, err = s.AcceptProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint); !errors.Is(err, store.ErrProposalUnavailable) {
		t.Fatal("action guessed", err)
	}
	if _, err = s.AcceptProposalWithAction(ctx, orgB, projectA, p.Key, userB, p.InputsFingerprint, "repair"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign accepted", err)
	}
	if _, err = s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, "stale", "repair"); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("stale accepted", err)
	}
	if _, err = s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "invented"); !errors.Is(err, store.ErrInvalidWork) {
		t.Fatal("unknown action accepted", err)
	}
	item, err := s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "repair")
	if err != nil {
		t.Fatal(err)
	}
	if item.Action != "repair" || item.SystemID != p.TargetSystemID || !strings.Contains(string(item.Provenance.Sources), `"user_selected_action":"repair"`) {
		t.Fatal("choice or target lost", item)
	}
	retry, err := s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "repair")
	if err != nil || retry.ID != item.ID {
		t.Fatal("retry duplicated", err)
	}
	if _, err = s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "replace"); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("retry changed action", err)
	}
	ps, err = s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var version int64
	for _, candidate := range ps {
		if candidate.Key == p.Key && candidate.Decision != nil {
			version = candidate.Decision.Version
		}
	}
	if version == 0 {
		t.Fatal("decision missing")
	}
	if err = s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, version); err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	ps, err = s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range ps {
		if candidate.Key == p.Key {
			p = candidate
		}
	}
	again, err := s.AcceptProposalWithAction(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "replace")
	if err != nil || again.ID != item.ID || again.Action != "replace" {
		t.Fatal("reaccept choice lost", err)
	}
}
