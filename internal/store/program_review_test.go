package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// Plan 11.3 "Review": differences matched by source key, nothing written
// to the program until Adopt, Adopt defaults only for unedited proposed
// rows, Adopt all is one undoable batch, a newer difference supersedes an
// older pending one, and Keep is not raised again until inputs change.

func pendingChanges(t *testing.T, s *store.Store) store.ProgramChangeList {
	t.Helper()
	list, err := s.ProgramChanges(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func changeFor(t *testing.T, list store.ProgramChangeList, kind, key string) store.ProgramChange {
	t.Helper()
	for _, c := range list.Changes {
		if c.Kind == kind && c.SourceKey == key {
			return c
		}
	}
	t.Fatalf("no %s change for %s in %+v", kind, key, list.Changes)
	return store.ProgramChange{}
}

func changeStatus(t *testing.T, id string) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var status string
	if err := conn.QueryRow(ctx, `SELECT status FROM program_changes WHERE org_id=$1 AND id=$2`, orgA, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestProgramReview(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	cat := b.Catalog
	review := func() store.ProgramReviewOutcome {
		t.Helper()
		out, err := s.ReviewProgram(ctx, orgA, projectA, cat)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	// No program: the review does nothing.
	if out := review(); out.ProgramID != "" || out.Changed {
		t.Fatalf("review without a program %+v", out)
	}
	// Work items drive the review: a single-activity fragment takes the
	// item's title, so a renamed item is a renamed row. Packages add no rows
	// (owner direction 25 October 2026).
	erosion, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "site.erosion-sediment-control", Action: "new", Title: "Erosion controls"})
	if err != nil {
		t.Fatal(err)
	}
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil {
		t.Fatal(err)
	}
	if out := review(); out.Changed || out.Pending != 0 || out.ProgramID != st.Program.ID {
		t.Fatalf("unchanged program reviewed %+v", out)
	}
	erosionRow := rowOf(t, st, "Erosion controls")

	// Changes elsewhere: a new work item, a renamed one, and a new package.
	// No path outside Program writes a program row, and the package raises
	// nothing.
	power, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "electrical.temporary-power", Action: "new", Title: "Builders power"})
	if err != nil {
		t.Fatal(err)
	}
	rename := func(title string) {
		t.Helper()
		if erosion, err = s.PatchWorkItem(ctx, orgA, projectA, erosion.ID, userA, works.Patch{Version: erosion.Version, Title: &title}); err != nil {
			t.Fatal(err)
		}
	}
	rename("Erosion and sediment controls")
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineer", LifecycleStatus: "planned",
		Stages: []procurement.Stage{{StageID: "concept_design", Label: "Concept", NovationPhase: "none", Origin: "user"}}}); err != nil {
		t.Fatal(err)
	}
	unchanged := func(when string) {
		t.Helper()
		now, err := s.LoadProgram(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		if now.Program.Version != st.Program.Version || labels(now) != labels(st) {
			t.Fatalf("%s wrote the program: v%d→v%d %s", when, st.Program.Version, now.Program.Version, labels(now))
		}
	}
	unchanged("a scope change")

	out := review()
	if !out.Changed || out.Pending != 2 {
		t.Fatalf("review %+v %+v", out, pendingChanges(t, s).Changes)
	}
	unchanged("the review")
	list := pendingChanges(t, s)
	powerKey := "fragment:fr.temporary-power:" + power.ID + ":connect"
	add := changeFor(t, list, engine.ChangeAdd, powerKey)
	ren := changeFor(t, list, engine.ChangeRename, string(erosionRow.SourceKey))
	if add.Source != engine.SourceWorkItems || add.DefaultChoice != engine.ChoiceAdopt || add.TargetActivityID != "" || !strings.Contains(add.Summary, "Builders power") {
		t.Fatalf("add %+v", add)
	}
	if string(ren.TargetActivityID) != erosionRow.ID || ren.DefaultChoice != engine.ChoiceAdopt || ren.EditedSince {
		t.Fatalf("rename %+v", ren)
	}
	if list.Version != st.Program.Version || list.Counts["add"] != 1 || list.Counts["rename"] != 1 || len(list.Changes) != 2 {
		t.Fatalf("list %+v", list)
	}
	// Running again raises nothing new.
	if out := review(); out.Changed || out.Pending != 2 {
		t.Fatalf("second review %+v", out)
	}

	// The user edits the erosion row after it was raised: it now defaults
	// to Keep and says so.
	label := "Silt fences"
	res := apply(t, s, st, store.ProgramOp{Op: "update", ID: erosionRow.ID, Label: &label})
	list = pendingChanges(t, s)
	ren = changeFor(t, list, engine.ChangeRename, string(erosionRow.SourceKey))
	if ren.DefaultChoice != engine.ChoiceKeep || !ren.EditedSince || !ren.Edited || !strings.HasSuffix(ren.Summary, engine.EditedSince) {
		t.Fatalf("edited since %+v", ren)
	}
	if out := review(); out.Changed {
		t.Fatalf("an edit re-raised the same differences %+v", out)
	}

	// A stale version is a conflict.
	if _, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, st.Program.Version, nil, true); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale decide %v", err)
	}
	// Adopt all: one batch, only the changes that default to Adopt.
	v := res.Program.Version
	adopted, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, v, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if adopted.Program.Version != v+1 || !hasRow(adopted.ProgramState, "Builders power") || !hasRow(adopted.ProgramState, label) ||
		hasRow(adopted.ProgramState, "Erosion and sediment controls") {
		t.Fatalf("adopt all v%d %s", adopted.Program.Version, labels(adopted.ProgramState))
	}
	added := rowOf(t, adopted.ProgramState, "Builders power")
	if string(added.SourceKey) != powerKey || added.Origin != "library" || added.ReviewStatus != "proposed" || string(added.WorkItemID) != power.ID {
		t.Fatalf("adopted row %+v", added)
	}
	if changeStatus(t, add.ID) != "adopted" || changeStatus(t, ren.ID) != "pending" {
		t.Fatal("statuses after adopt all")
	}
	if list = pendingChanges(t, s); len(list.Changes) != 1 || list.Changes[0].ID != ren.ID {
		t.Fatalf("pending after adopt all %+v", list.Changes)
	}
	view, err := s.ProgramView(ctx, orgA, projectA, adopted.ProgramState, cat)
	if err != nil || view.ChangesPending != 1 {
		t.Fatalf("changes_pending %d %v", view.ChangesPending, err)
	}

	// One undo reverses the whole adoption; the review then raises the
	// differences again.
	undone, err := s.UndoProgram(ctx, orgA, projectA, userA, adopted.Program.Version)
	if err != nil {
		t.Fatal(err)
	}
	if labels(undone.ProgramState) != labels(res.ProgramState) {
		t.Fatalf("undo %s, want %s", labels(undone.ProgramState), labels(res.ProgramState))
	}
	if out := review(); !out.Changed || out.Pending != 2 {
		t.Fatalf("review after undo %+v", out)
	}

	// Keep: marked kept, and not raised again while its inputs stand.
	list = pendingChanges(t, s)
	ren = changeFor(t, list, engine.ChangeRename, string(erosionRow.SourceKey))
	kept, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, undone.Program.Version, []store.ProgramDecision{{ID: ren.ID, Choice: "keep"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Program.Version != undone.Program.Version || changeStatus(t, ren.ID) != "kept" {
		t.Fatalf("keep moved the version or did not mark: v%d", kept.Program.Version)
	}
	if _, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, kept.Program.Version, []store.ProgramDecision{{ID: ren.ID, Choice: "keep"}}, false); err == nil {
		t.Fatal("a decided change was decided again")
	}
	if out := review(); out.Changed || out.Pending != 1 {
		t.Fatalf("kept change raised again %+v", out)
	}
	// Its input changes: the item is renamed again, so it is raised. The
	// pending add is the same difference and is not replaced.
	oldAdd := changeFor(t, pendingChanges(t, s), engine.ChangeAdd, powerKey)
	rename("Sediment fencing")
	if out := review(); !out.Changed || out.Pending != 2 {
		t.Fatalf("review after a new rename %+v", out)
	}
	list = pendingChanges(t, s)
	newRen := changeFor(t, list, engine.ChangeRename, string(erosionRow.SourceKey))
	if newRen.ID == ren.ID || !strings.Contains(newRen.Summary, "Sediment fencing") || newRen.DefaultChoice != engine.ChoiceKeep {
		t.Fatalf("re-raised keep %+v", newRen)
	}
	if changeFor(t, list, engine.ChangeAdd, powerKey).ID != oldAdd.ID {
		t.Fatal("an unchanged pending difference was replaced")
	}
	// A second rename while the first is pending supersedes it.
	rename("Sediment fences")
	review()
	if changeStatus(t, newRen.ID) != "superseded" {
		t.Fatal("a newer difference did not supersede the older pending one")
	}

	// The item leaves scope: the removal of its edited row defaults to Keep;
	// the pending rename of a removed row is superseded.
	if err := s.RetireWorkItem(ctx, orgA, projectA, erosion.ID, userA, erosion.Version); err != nil {
		t.Fatal(err)
	}
	review()
	list = pendingChanges(t, s)
	rm := changeFor(t, list, engine.ChangeRemove, string(erosionRow.SourceKey))
	if rm.DefaultChoice != engine.ChoiceKeep || !rm.Edited {
		t.Fatalf("removal %+v", rm)
	}
	for _, c := range list.Changes {
		if c.Kind == engine.ChangeRename {
			t.Fatalf("a rename of a removed row stayed pending: %+v", c)
		}
	}
	done, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, kept.Program.Version,
		[]store.ProgramDecision{{ID: rm.ID, Choice: "keep"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRow(done.ProgramState, label) {
		t.Fatalf("decide %s", labels(done.ProgramState))
	}

	// Bad decisions are refused before anything is written.
	var opErr *store.ProgramOpError
	for _, d := range [][]store.ProgramDecision{{{ID: "nope", Choice: "adopt"}}, {{ID: add.ID, Choice: "maybe"}}, {{ID: newUUIDString(), Choice: "adopt"}}} {
		if _, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, done.Program.Version, d, false); !errors.As(err, &opErr) {
			t.Fatalf("decision %+v: %v", d, err)
		}
	}
	if _, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, done.Program.Version, nil, false); !errors.As(err, &opErr) {
		t.Fatalf("empty decision %v", err)
	}
	// Another org sees no changes and decides nothing.
	if _, err := s.ProgramChanges(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	if _, err := s.DecideProgramChanges(ctx, orgB, projectA, userA, done.Program.Version, nil, true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign decide %v", err)
	}
}

func newUUIDString() string { return "0f0f0f0f-0f0f-4f0f-8f0f-0f0f0f0f0f0f" }
