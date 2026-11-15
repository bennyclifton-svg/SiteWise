package store_test

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/store"
	"sitewise/internal/works"
)

// Owner, 12 November 2026: Update program applies everything in one click.
// The Merricks path: a detailed program built and accepted (so every row
// reads as the user's), then the outline arrives through Update program.
// One click lands it all, nothing is left for a review to raise, and one
// undo puts the old program back.
func TestUpdateProgramAppliesEverythingInOneClick(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	cat := b.Catalog
	// Merricks: a new house on a DA, traditional, NCC class not stated.
	setValue(t, s, part, "hdr.cond.planning", store.UserWrite{Value: strPtr("da")})
	setValue(t, s, part, "hdr.cond.procurement_route", store.UserWrite{Value: strPtr("traditional")})
	setValue(t, s, part, "hdr.work_type", store.UserWrite{Value: strPtr("new")})
	setValue(t, s, part, "det.state", store.UserWrite{Value: strPtr("VIC"), Scope: cat.KeyScope("det.state")})
	for _, sys := range []string{"site.erosion-sediment-control", "substructure.footings", "structure.timber", "envelope.roof-coverings", "electrical.cabling-wiring"} {
		if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: sys, Action: "new", Title: sys}); err != nil {
			t.Fatal(err)
		}
	}

	store.SetProgramOutlineForTest(t, false)
	old, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := s.AcceptProgram(ctx, orgA, projectA, userA, old.Program.Version, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	store.SetProgramOutlineForTest(t, true)
	res, err := s.UpdateProgram(ctx, orgA, projectA, userA, accepted.Program.Version, cat)
	if err != nil {
		t.Fatal(err)
	}
	if res.Program.Version != accepted.Program.Version+1 {
		t.Fatalf("one batch: v%d → v%d", accepted.Program.Version, res.Program.Version)
	}
	if hasRow(res.ProgramState, "Erosion") || !hasRow(res.ProgramState, "Construction") {
		t.Fatalf("outline not applied: %s", labels(res.ProgramState))
	}
	for _, a := range res.Activities {
		if strings.HasPrefix(a.Label, "If ") || a.Kind == "hold_point" {
			t.Fatalf("left behind: %s", a.Label)
		}
	}
	if n := len(res.Activities); n < 20 || n > 30 {
		t.Fatalf("%d rows after update: %s", n, labels(res.ProgramState))
	}
	if out, err := s.ReviewProgram(ctx, orgA, projectA, cat); err != nil || out.Pending != 0 {
		t.Fatalf("review after update raised %d: %v %+v", out.Pending, err, pendingChanges(t, s).Changes)
	}

	// Nothing more to apply: the program is returned unchanged.
	again, err := s.UpdateProgram(ctx, orgA, projectA, userA, res.Program.Version, cat)
	if err != nil || again.Program.Version != res.Program.Version {
		t.Fatalf("second update v%d %v", again.Program.Version, err)
	}

	undone, err := s.UndoProgram(ctx, orgA, projectA, userA, res.Program.Version)
	if err != nil {
		t.Fatal(err)
	}
	if labels(undone.ProgramState) != labels(accepted.ProgramState) {
		t.Fatalf("undo %s, want %s", labels(undone.ProgramState), labels(accepted.ProgramState))
	}
}
