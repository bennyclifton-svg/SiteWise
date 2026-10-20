package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// Plan section 2 rule 2, 7.4 and 13.1: the review raises what a change made
// elsewhere did, never the user's own Program edits. A rename the user made
// is not offered back; a generated row the user deleted is not offered
// again; a row the review itself removed comes back when its input does.

func execSQL(t *testing.T, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestProgramReviewLeavesTheUsersOwnEdits(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	cat := b.Catalog
	review := func(when string) {
		t.Helper()
		if _, err := s.ReviewProgram(ctx, orgA, projectA, cat); err != nil {
			t.Fatal(err)
		}
		if list := pendingChanges(t, s); len(list.Changes) != 0 {
			t.Fatalf("%s raised the user's own edit: %+v", when, list.Changes)
		}
	}
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil {
		t.Fatal(err)
	}
	review("a fresh program")

	concept := rowOf(t, st, "Concept design")
	label := "Sketch design"
	res := apply(t, s, st, store.ProgramOp{Op: "update", ID: concept.ID, Label: &label})
	review("a rename")

	def := rowOf(t, res.ProgramState, "Defects liability")
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{def.ID}})
	if hasRow(res.ProgramState, "Defects liability") {
		t.Fatal("delete did not retire the stage")
	}
	review("a delete")

	// Answering a question applies Adopt differences at once; it must not
	// bring the deleted stage back either.
	q := engine.Question{Kind: engine.QuestionWorkType, Key: "work_type", Label: "Work type"}
	ans, _, err := s.ApplyProgramAnswer(ctx, orgA, projectA, userA, res.Program.Version, cat, q, "new")
	if err != nil {
		t.Fatal(err)
	}
	if hasRow(ans.ProgramState, "Defects liability") || !hasRow(ans.ProgramState, label) {
		t.Fatalf("an answer undid the user's edits: %s", labels(ans.ProgramState))
	}
}

func TestProgramReviewRaisesAReviewRemovalWhoseInputComesBack(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	cat := b.Catalog
	stage := procurement.Stage{StageID: "concept_design", Label: "Concept", NovationPhase: "none", Origin: "user"}
	arch, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned", Stages: []procurement.Stage{stage}})
	if err != nil {
		t.Fatal(err)
	}
	eng, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineer", LifecycleStatus: "planned", Stages: []procurement.Stage{stage}})
	if err != nil {
		t.Fatal(err)
	}
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil {
		t.Fatal(err)
	}
	archRow, engRow := rowOf(t, st, "Architect: Concept"), rowOf(t, st, "Engineer: Concept")

	// The user deletes the engineer's row; the architect's package is
	// removed in Procurement and the review's removal is adopted.
	res := apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{engRow.ID}})
	retire := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, arch.ID, userA, procurement.PackagePatch{Version: arch.Version, Retired: &retire}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReviewProgram(ctx, orgA, projectA, cat); err != nil {
		t.Fatal(err)
	}
	rm := changeFor(t, pendingChanges(t, s), engine.ChangeRemove, string(archRow.SourceKey))
	done, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, res.Program.Version, []store.ProgramDecision{{ID: rm.ID, Choice: "adopt"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if hasRow(done.ProgramState, "Architect: Concept") {
		t.Fatal("adopted removal kept the row")
	}

	// The package comes back (its input returns): the review offers the row
	// again with Adopt. The engineer's row, which the user deleted, is not
	// offered.
	execSQL(t, `UPDATE packages SET retired_at=NULL WHERE org_id=$1 AND id=$2`, orgA, arch.ID)
	if _, err := s.ReviewProgram(ctx, orgA, projectA, cat); err != nil {
		t.Fatal(err)
	}
	list := pendingChanges(t, s)
	add := changeFor(t, list, engine.ChangeAdd, string(archRow.SourceKey))
	if add.DefaultChoice != engine.ChoiceAdopt {
		t.Fatalf("re-raised add %+v", add)
	}
	for _, c := range list.Changes {
		if c.SourceKey == string(engRow.SourceKey) {
			t.Fatalf("the user's delete was offered back: %+v (engineer %s)", c, eng.ID)
		}
	}
}

// Plan 7.7: a user-made stage (`other`) receives evidence only when the
// user assigns it, so evidence for a deleted one never moves to another.
func TestProgramReadingForADeletedOtherStageStaysUnassigned(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	_, done := readImport(t, s)
	if done.ReadingsPending == 0 {
		t.Fatal("no readings")
	}
	// Approvals is user stage A (`other`); add user stage B (`other`).
	kind, other, label := "stage", "other", "Authority works"
	res := apply(t, s, st, store.ProgramOp{Op: "add", TempID: "b", Kind: &kind, Label: &label, CatalogueStageID: &other})
	a, bRow := rowOf(t, res.ProgramState, "Approvals"), rowOf(t, res.ProgramState, label)
	// The Slab readings are re-pointed at A, as an assigned reading for a
	// user stage carries them.
	slab := rowID(t, res.ProgramState, "Slab")
	execSQL(t, `UPDATE program_readings SET target_activity_id=$3::uuid,catalogue_stage_id='other' WHERE org_id=$1 AND target_activity_id=$2::uuid`, orgA, slab, a.ID)
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{a.ID}})
	bRow = rowOf(t, res.ProgramState, label) // as the delete left it

	v, err := s.ProgramView(ctx, orgA, projectA, res.ProgramState, nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range v.Readings {
		if r.CatalogueStageID == "other" {
			ids = append(ids, r.ID)
		}
	}
	if len(ids) == 0 {
		t.Fatal("no readings for the deleted stage")
	}
	acc, err := s.AcceptProgram(ctx, orgA, projectA, userA, res.Program.Version, ids, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(acc.Toasts) != 1 || !strings.Contains(acc.Toasts[0], "no longer in the program") {
		t.Fatalf("accept toasts %v", acc.Toasts)
	}
	now, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	b2 := rowOf(t, now, label)
	if b2.Version != bRow.Version || b2.DurationOrigin != bRow.DurationOrigin || b2.ConstraintKind != bRow.ConstraintKind {
		t.Fatalf("evidence for a deleted user stage moved to another: %+v", b2)
	}
	v, err = s.ProgramView(ctx, orgA, projectA, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range v.Checks {
		found = found || c.Code == "evidence_without_stage"
	}
	if !found {
		t.Fatalf("no evidence_without_stage check: %+v", v.Checks)
	}
}

// Plan 13.1: a hold point deleted with a reason is retired, and a check
// naming it and the reason stays visible until undo restores it.
func TestProgramHoldPointDismissedCheck(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st, hold := holdProgram(t, s)
	check := func(st store.ProgramState) *engine.Check {
		t.Helper()
		v, err := s.ProgramView(ctx, orgA, projectA, st, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range v.Checks {
			if c.Code == "hold_point_dismissed" {
				return &v.Checks[i]
			}
		}
		return nil
	}
	if c := check(st); c != nil {
		t.Fatalf("live hold point has a check %+v", c)
	}
	reason := "Engineer waived the inspection"
	res := apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{hold}, Reason: &reason})
	c := check(res.ProgramState)
	if c == nil || c.Message != "Hold point removed: Hold point: frame inspection. Reason: Engineer waived the inspection." {
		t.Fatalf("dismissed check %+v", c)
	}
	undone, err := s.UndoProgram(ctx, orgA, projectA, userA, res.Program.Version)
	if err != nil {
		t.Fatal(err)
	}
	if c := check(undone.ProgramState); c != nil {
		t.Fatalf("restored hold point still has a check %+v", c)
	}
}
