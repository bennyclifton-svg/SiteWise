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
	s, b, _ := workStore(t)
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
	stage := func(id, label string) procurement.Stage {
		return procurement.Stage{StageID: id, Label: label, NovationPhase: "none", Origin: "user"}
	}
	arch, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned",
		Stages: []procurement.Stage{stage("concept_design", "Concept"), stage("construction", "Site visits")}})
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

	// Changes made in Procurement: a new package, a renamed one. No path
	// outside Program writes a program row.
	eng, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineer", LifecycleStatus: "planned",
		Stages: []procurement.Stage{stage("concept_design", "Concept")}})
	if err != nil {
		t.Fatal(err)
	}
	title := "Architect (lead)"
	if arch, err = s.PatchPackage(ctx, orgA, projectA, arch.ID, userA, procurement.PackagePatch{Version: arch.Version, Title: &title}); err != nil {
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
	unchanged("a package change")

	out := review()
	if !out.Changed || out.Pending != 3 {
		t.Fatalf("review %+v", out)
	}
	unchanged("the review")
	list := pendingChanges(t, s)
	engKey := "package:" + eng.ID + ":" + eng.Stages[0].ID
	add := changeFor(t, list, engine.ChangeAdd, engKey)
	concept := rowOf(t, st, "Architect: Concept")
	visits := rowOf(t, st, "Architect: Site visits")
	renConcept := changeFor(t, list, engine.ChangeRename, string(concept.SourceKey))
	renVisits := changeFor(t, list, engine.ChangeRename, string(visits.SourceKey))
	if add.Source != engine.SourceProcurement || add.DefaultChoice != engine.ChoiceAdopt || add.TargetActivityID != "" || !strings.Contains(add.Summary, "Engineer: Concept") {
		t.Fatalf("add %+v", add)
	}
	if string(renVisits.TargetActivityID) != visits.ID || renVisits.DefaultChoice != engine.ChoiceAdopt || renVisits.EditedSince {
		t.Fatalf("rename %+v", renVisits)
	}
	if list.Version != st.Program.Version || list.Counts["add"] != 1 || list.Counts["rename"] != 2 || len(list.Changes) != 3 {
		t.Fatalf("list %+v", list)
	}
	// Running again raises nothing new.
	if out := review(); out.Changed || out.Pending != 3 {
		t.Fatalf("second review %+v", out)
	}

	// The user edits Site visits after the item was raised: it now defaults
	// to Keep and says so.
	label := "Site visits (fortnightly)"
	res := apply(t, s, st, store.ProgramOp{Op: "update", ID: visits.ID, Label: &label})
	list = pendingChanges(t, s)
	renVisits = changeFor(t, list, engine.ChangeRename, string(visits.SourceKey))
	if renVisits.DefaultChoice != engine.ChoiceKeep || !renVisits.EditedSince || !renVisits.Edited || !strings.HasSuffix(renVisits.Summary, engine.EditedSince) {
		t.Fatalf("edited since %+v", renVisits)
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
	if adopted.Program.Version != v+1 || !hasRow(adopted.ProgramState, "Engineer: Concept") || !hasRow(adopted.ProgramState, "Architect (lead): Concept") ||
		!hasRow(adopted.ProgramState, label) || hasRow(adopted.ProgramState, "Architect (lead): Site visits") {
		t.Fatalf("adopt all v%d %s", adopted.Program.Version, labels(adopted.ProgramState))
	}
	added := rowOf(t, adopted.ProgramState, "Engineer: Concept")
	if string(added.SourceKey) != engKey || added.Origin != "library" || added.ReviewStatus != "proposed" || string(added.PackageID) != eng.ID {
		t.Fatalf("adopted row %+v", added)
	}
	if changeStatus(t, add.ID) != "adopted" || changeStatus(t, renConcept.ID) != "adopted" || changeStatus(t, renVisits.ID) != "pending" {
		t.Fatal("statuses after adopt all")
	}
	if list = pendingChanges(t, s); len(list.Changes) != 1 || list.Changes[0].ID != renVisits.ID {
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
	if out := review(); !out.Changed || out.Pending != 3 {
		t.Fatalf("review after undo %+v", out)
	}

	// Keep: marked kept, and not raised again while its inputs stand.
	list = pendingChanges(t, s)
	renVisits = changeFor(t, list, engine.ChangeRename, string(visits.SourceKey))
	kept, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, undone.Program.Version, []store.ProgramDecision{{ID: renVisits.ID, Choice: "keep"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Program.Version != undone.Program.Version || changeStatus(t, renVisits.ID) != "kept" {
		t.Fatalf("keep moved the version or did not mark: v%d", kept.Program.Version)
	}
	if _, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, kept.Program.Version, []store.ProgramDecision{{ID: renVisits.ID, Choice: "keep"}}, false); err == nil {
		t.Fatal("a decided change was decided again")
	}
	if out := review(); out.Changed || out.Pending != 2 {
		t.Fatalf("kept change raised again %+v", out)
	}
	// Its inputs change: the package is renamed again, so it is raised.
	title = "Architect (design)"
	if arch, err = s.PatchPackage(ctx, orgA, projectA, arch.ID, userA, procurement.PackagePatch{Version: arch.Version, Title: &title}); err != nil {
		t.Fatal(err)
	}
	// The older pending rename of Concept and the add are superseded only
	// when their difference changes: the add is the same, the rename is new.
	oldConcept := changeFor(t, pendingChanges(t, s), engine.ChangeRename, string(concept.SourceKey))
	oldAdd := changeFor(t, pendingChanges(t, s), engine.ChangeAdd, engKey)
	if out := review(); !out.Changed || out.Pending != 3 {
		t.Fatalf("review after a new rename %+v", out)
	}
	list = pendingChanges(t, s)
	newVisits := changeFor(t, list, engine.ChangeRename, string(visits.SourceKey))
	newConcept := changeFor(t, list, engine.ChangeRename, string(concept.SourceKey))
	if newVisits.ID == renVisits.ID || !strings.Contains(newVisits.Summary, "Architect (design): Site visits") || newVisits.DefaultChoice != engine.ChoiceKeep {
		t.Fatalf("re-raised keep %+v", newVisits)
	}
	if newConcept.ID == oldConcept.ID || changeStatus(t, oldConcept.ID) != "superseded" {
		t.Fatal("a newer difference did not supersede the older pending one")
	}
	if changeFor(t, list, engine.ChangeAdd, engKey).ID != oldAdd.ID {
		t.Fatal("an unchanged pending difference was replaced")
	}

	// Package removed in Procurement: the removal of its edited row defaults
	// to Keep, and the package_removed check names the row meanwhile.
	retired := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, arch.ID, userA, procurement.PackagePatch{Version: arch.Version, Retired: &retired}); err != nil {
		t.Fatal(err)
	}
	unchanged2, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged2.Program.Version != kept.Program.Version {
		t.Fatal("removing a package wrote the program")
	}
	review()
	list = pendingChanges(t, s)
	rmVisits := changeFor(t, list, engine.ChangeRemove, string(visits.SourceKey))
	rmConcept := changeFor(t, list, engine.ChangeRemove, string(concept.SourceKey))
	if rmVisits.DefaultChoice != engine.ChoiceKeep || !rmVisits.Edited || rmConcept.DefaultChoice != engine.ChoiceAdopt {
		t.Fatalf("removals %+v %+v", rmVisits, rmConcept)
	}
	if changeStatus(t, newConcept.ID) != "superseded" {
		t.Fatal("a rename of a removed package stayed pending")
	}
	view, err = s.ProgramView(ctx, orgA, projectA, unchanged2, cat)
	if err != nil {
		t.Fatal(err)
	}
	var removedCheck bool
	for _, c := range view.Checks {
		if c.Code == engine.CheckPackageRemoved && c.ActivityID == visits.ID {
			removedCheck = true
		}
	}
	if !removedCheck || view.ChangesPending != len(list.Changes) {
		t.Fatalf("checks %+v pending %d/%d", view.Checks, view.ChangesPending, len(list.Changes))
	}
	// Adopting the removal explicitly applies it, kept rows stay.
	done, err := s.DecideProgramChanges(ctx, orgA, projectA, userA, unchanged2.Program.Version,
		[]store.ProgramDecision{{ID: rmConcept.ID, Choice: "adopt"}, {ID: rmVisits.ID, Choice: "keep"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if hasRow(done.ProgramState, "Architect: Concept") || !hasRow(done.ProgramState, label) || done.Program.Version != unchanged2.Program.Version+1 {
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
