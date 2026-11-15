package store_test

import (
	"context"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/store"
)

// The first cost line of a project also creates its plan. Undoing that
// change removes both, and the project reads as having no plan again.
func TestUndoTheChangeThatCreatedThePlan(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	change := store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "chat", Area: "cost_plan"}
	in := store.CostItemInput{Content: costs.Content{Label: "DA fees", LineKind: "project_wide", Category: "authority_fees", Posting: true, Origin: "user", Meaning: "allowance"}}
	p, err := s.CreateCostItem(store.WithChange(ctx, change), orgA, projectA, userA, in)
	if err != nil || len(p.Items) != 1 {
		t.Fatalf("plan %+v err %v", p, err)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatalf("undo: %v", err)
	}
	after, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil || len(after.Items) != 0 {
		t.Fatalf("after undo: %d lines, err %v", len(after.Items), err)
	}
	// A new first line still works.
	if _, err := s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: after.ID, Version: after.Version}, Content: in.Content}); err != nil {
		t.Fatalf("a line after the undo: %v", err)
	}
}
