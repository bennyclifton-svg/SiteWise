package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// Combined engagements, slice 4 (docs/plans/2026-11-07-combined-engagements-slice4.md):
// a group has one set of stage lines, the union of its members' stages in
// program order, kept by the existing stage sync. Members' rows are never
// rewritten and are inert while grouped.

// stageRows is a package's live stages as (program stage, id, version,
// label), for comparing before and after.
func stageRows(t *testing.T, s *store.Store, id string) string {
	t.Helper()
	p := readPkg(t, s, id)
	raw, _ := json.Marshal(p.Stages)
	return string(raw)
}

func programStageIDs(p procurement.Package) []string {
	out := []string{}
	for _, st := range p.Stages {
		out = append(out, st.ProgramStageID)
	}
	return out
}

// retireStage unticks a member's stage pointing at program stage ps.
func retireStage(t *testing.T, s *store.Store, pkg, ps string) {
	t.Helper()
	st := stageByProgram(readPkg(t, s, pkg), ps)
	if st == nil {
		t.Fatalf("package %s has no stage %s", pkg, ps)
	}
	if _, err := s.PatchPackageStage(context.Background(), orgA, projectA, pkg, st.ID, userA, procurement.StagePatch{Version: st.Version, Retired: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
}

func programSnapshot(t *testing.T, s *store.Store) string {
	t.Helper()
	st, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"version": st.Program.Version, "revision": st.Program.StageRevision, "activities": st.Activities, "links": st.Links})
	return string(raw)
}

func consultantRows(t *testing.T, s *store.Store, disciplines ...string) []procurement.Package {
	t.Helper()
	out := []procurement.Package{}
	for _, d := range disciplines {
		row, ok := procurement.DisciplinePackage(d, memberLabels[d])
		if !ok {
			t.Fatal("unknown discipline", d)
		}
		p, err := s.CreateDisciplinePackage(context.Background(), orgA, projectA, userA, row)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestGroupStageLinesAreTheMembersUnion(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	// A user's edits to the program: a duration and a link's lag.
	var activity string
	for _, a := range st.Activities {
		if a.Kind != engine.KindStage && activity == "" {
			activity = a.ID
		}
	}
	lag := 3
	ops := []store.ProgramOp{{Op: "update", ID: activity, DurationDays: dur(17)}}
	if len(st.Links) > 0 {
		ops = append(ops, store.ProgramOp{Op: "link_update", ID: st.Links[0].ID, LagDays: &lag})
	}
	apply(t, s, st, ops...)

	pkgs := consultantRows(t, s, "consultant.structural", "consultant.civil", "consultant.hydraulic")
	structural, civil, hydraulic := pkgs[0], pkgs[1], pkgs[2]
	all := programStageIDs(structural)
	if len(all) < 3 {
		t.Fatalf("too few default stages %v", all)
	}
	// Nobody keeps the first stage; only civil keeps the third.
	for _, p := range pkgs {
		retireStage(t, s, p.ID, all[0])
	}
	retireStage(t, s, structural.ID, all[2])
	retireStage(t, s, hydraulic.ID, all[2])
	want := all[1:]
	before := map[string]string{}
	for _, p := range pkgs {
		before[p.ID] = stageRows(t, s, p.ID)
	}
	program := programSnapshot(t, s)

	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{structural.ID, civil.ID, hydraulic.ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	g := readPkg(t, s, group.ID)
	if got := programStageIDs(g); len(got) != len(want) {
		t.Fatalf("group stages %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("group stages %v, want %v (program order)", got, want)
			}
		}
	}
	st, err = s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	labels := map[string]string{}
	for _, a := range st.Activities {
		labels[a.ID] = a.Label
	}
	for i, line := range g.Stages {
		if line.Label != labels[line.ProgramStageID] || line.Ordinal != i || line.StageRevision != st.Program.StageRevision || line.NovationPhase != "none" {
			t.Fatalf("group line %+v", line)
		}
	}
	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, group.ID)
	if err != nil || !sync.ProgramExists || sync.BuiltFrom != st.Program.StageRevision || sync.Offer || sync.GroupID != "" {
		t.Fatalf("group sync %+v %v", sync, err)
	}
	ticked := 0
	for _, pick := range sync.Stages {
		if pick.Ticked {
			ticked++
		}
	}
	if ticked != len(want) {
		t.Fatalf("group picker %+v", sync.Stages)
	}
	// Members are untouched and contribute nothing while grouped.
	for _, p := range pkgs {
		if got := stageRows(t, s, p.ID); got != before[p.ID] {
			t.Fatalf("member %s stages rewritten:\n%s\n%s", p.Title, before[p.ID], got)
		}
		ms, err := s.ReadPackageStageSync(ctx, orgA, projectA, p.ID)
		if err != nil || ms.GroupID != group.ID || ms.Offer || len(ms.Stages) != 0 || len(ms.Changes) != 0 {
			t.Fatalf("member sync %+v %v", ms, err)
		}
	}
	for _, choice := range []string{store.StageSyncUpdate, store.StageSyncKeep} {
		if _, err := s.SyncPackageStages(ctx, orgA, projectA, civil.ID, userA, choice, st.Program.StageRevision); !errors.Is(err, store.ErrPackageGrouped) {
			t.Fatalf("member %s: %v", choice, err)
		}
	}
	if _, err := s.CreatePackageStage(ctx, orgA, projectA, civil.ID, userA, procurement.Stage{ProgramStageID: all[0]}); !errors.Is(err, store.ErrPackageGrouped) {
		t.Fatalf("member stage create: %v", err)
	}
	line := readPkg(t, s, civil.ID).Stages[0]
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, civil.ID, line.ID, userA, procurement.StagePatch{Version: line.Version, Retired: boolPtr(true)}); !errors.Is(err, store.ErrPackageGrouped) {
		t.Fatalf("member stage change: %v", err)
	}

	// Rename: the lines are the group's, so nothing is rewritten.
	lines := stageRows(t, s, group.ID)
	title := "Engineering"
	if _, err := s.PatchPackage(ctx, orgA, projectA, group.ID, userA, procurement.PackagePatch{Version: g.Version, Title: &title}); err != nil {
		t.Fatal(err)
	}
	if got := stageRows(t, s, group.ID); got != lines {
		t.Fatalf("rename rewrote lines:\n%s\n%s", lines, got)
	}
	if sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, group.ID); err != nil || sync.Offer {
		t.Fatalf("sync after rename %+v %v", sync, err)
	}

	// Remove the only member holding the third stage: the group's lines are
	// the remaining union, the kept lines keep their ids.
	keep := map[string]string{}
	for _, l := range readPkg(t, s, group.ID).Stages {
		keep[l.ProgramStageID] = l.ID
	}
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, civil.ID); err != nil {
		t.Fatal(err)
	}
	g = readPkg(t, s, group.ID)
	if got := programStageIDs(g); len(got) != len(want)-1 || stageByProgram(g, all[2]) != nil {
		t.Fatalf("after remove %v", got)
	}
	for _, l := range g.Stages {
		if keep[l.ProgramStageID] != l.ID {
			t.Fatalf("line %s recreated", l.Label)
		}
	}
	if got := stageRows(t, s, civil.ID); got != before[civil.ID] {
		t.Fatalf("removed member's stages changed")
	}
	if ms, err := s.ReadPackageStageSync(ctx, orgA, projectA, civil.ID); err != nil || ms.GroupID != "" || len(ms.Stages) == 0 {
		t.Fatalf("removed member sync %+v %v", ms, err)
	}

	// The program's dates and links are as the user left them.
	if got := programSnapshot(t, s); got != program {
		t.Fatalf("program changed by combine or remove")
	}

	// Another org or project sees nothing.
	if _, err := s.ReadPackageStageSync(ctx, orgB, projectA, group.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org read: %v", err)
	}
	if _, err := s.SyncPackageStages(ctx, orgB, projectA, group.ID, userB, store.StageSyncUpdate, st.Program.StageRevision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org write: %v", err)
	}
	const otherProject = "33333333-3333-4333-8333-555555555555"
	if err := s.CreateProject(ctx, orgA, otherProject, "Other project"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadPackageStageSync(ctx, orgA, otherProject, group.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong project read: %v", err)
	}
	if _, err := s.SyncPackageStages(ctx, orgA, otherProject, group.ID, userA, store.StageSyncKeep, st.Program.StageRevision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong project write: %v", err)
	}
}

func TestGroupStageSyncOfferedOnceAndRestoredOnUncombine(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	pkgs := consultantRows(t, s, "consultant.structural", "consultant.civil")
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[0].ID, pkgs[1].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	// The group's RFP reads the group's lines.
	if _, _, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, group.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	reports, err := s.ListReports(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	groupReport, memberReport := "", ""
	for _, r := range reports {
		switch r.PackageID {
		case group.ID:
			groupReport = r.ID
		case pkgs[0].ID:
			memberReport = r.ID
		}
	}
	if groupReport == "" {
		t.Fatal("no group RFP")
	}
	if _, err := s.EnsureReportDraft(ctx, orgA, groupReport, userA, "test"); err != nil {
		t.Fatal(err)
	}
	if reports, err = s.ListReports(ctx, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.PackageID == pkgs[0].ID {
			memberReport = r.ID
		}
	}
	rs, err := s.ReadReportStageSync(ctx, orgA, groupReport)
	if err != nil || len(rs.Rows) != len(readPkg(t, s, group.ID).Stages) || len(rs.Rows) == 0 {
		t.Fatalf("group RFP rows %+v %v", rs, err)
	}
	before := map[string]string{}
	for _, p := range pkgs {
		before[p.ID] = stageRows(t, s, p.ID)
	}

	concept := programStageID(t, st, "concept_design")
	documentation := programStageID(t, st, "construction_documentation")
	renameStage(t, s, concept, "Sketch design")
	deleteStage(t, s, documentation)
	st, added := addStage(t, s, "Feasibility")

	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, group.ID)
	if err != nil || !sync.Offer {
		t.Fatalf("group offer %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageRenamed, concept); c == nil || c.Action != engine.CopyLabel {
		t.Fatalf("rename %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageRemoved, documentation); c == nil || c.Action != engine.Retire {
		t.Fatalf("removed %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageAdded, added); c == nil || c.Action != engine.OfferUnticked {
		t.Fatalf("added %+v", sync.Changes)
	}
	if rs, err := s.ReadReportStageSync(ctx, orgA, groupReport); err != nil || !rs.Offer {
		t.Fatalf("group RFP offer %+v %v", rs, err)
	}
	// Offered once: not on the members or their own RFPs.
	for _, p := range pkgs {
		if ms, err := s.ReadPackageStageSync(ctx, orgA, projectA, p.ID); err != nil || ms.Offer {
			t.Fatalf("member offer %+v %v", ms, err)
		}
	}
	if memberReport == "" {
		t.Fatal("the group build made no member RFP")
	}
	{
		ms, err := s.ReadReportStageSync(ctx, orgA, memberReport)
		if err != nil || ms.Offer || ms.GroupID != group.ID {
			t.Fatalf("member RFP offer %+v %v", ms, err)
		}
		if _, err := s.SyncReportStages(ctx, orgA, memberReport, userA, "test", store.StageSyncUpdate, st.Program.StageRevision); !errors.Is(err, store.ErrPackageGrouped) {
			t.Fatalf("member RFP update: %v", err)
		}
	}

	sync, err = s.SyncPackageStages(ctx, orgA, projectA, group.ID, userA, store.StageSyncUpdate, st.Program.StageRevision)
	if err != nil || sync.Offer {
		t.Fatalf("group update %+v %v", sync, err)
	}
	g := readPkg(t, s, group.ID)
	if l := stageByProgram(g, concept); l == nil || l.Label != "Sketch design" {
		t.Fatalf("rename not copied %+v", g.Stages)
	}
	if stageByProgram(g, documentation) != nil || stageByProgram(g, added) != nil {
		t.Fatalf("removed stage kept or added stage ticked %+v", g.Stages)
	}
	if rs, err := s.SyncReportStages(ctx, orgA, groupReport, userA, "test", store.StageSyncUpdate, st.Program.StageRevision); err != nil || rs.Offer {
		t.Fatalf("group RFP update %+v %v", rs, err)
	}
	for _, p := range pkgs {
		if got := stageRows(t, s, p.ID); got != before[p.ID] {
			t.Fatalf("member rows rewritten by the group's update")
		}
	}

	// Uncombine: the members' own lines are live again and their next sync
	// offers what the program changed while they were grouped.
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, group.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		ms, err := s.ReadPackageStageSync(ctx, orgA, projectA, p.ID)
		if err != nil || ms.GroupID != "" || !ms.Offer || changeOf(ms, engine.StageRenamed, concept) == nil {
			t.Fatalf("released member sync %+v %v", ms, err)
		}
		if ms, err = s.SyncPackageStages(ctx, orgA, projectA, p.ID, userA, store.StageSyncUpdate, st.Program.StageRevision); err != nil || ms.Offer {
			t.Fatalf("released member update %+v %v", ms, err)
		}
		if l := stageByProgram(readPkg(t, s, p.ID), concept); l == nil || l.Label != "Sketch design" {
			t.Fatalf("released member rename not copied")
		}
	}
	if items, err := s.ReadPackages(ctx, orgA, projectA); err != nil {
		t.Fatal(err)
	} else {
		for _, p := range items {
			if p.ID == group.ID {
				t.Fatal("group still live")
			}
		}
	}
}

func TestGroupStageLinesWithoutAProgram(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	pkgs := consultantRows(t, s, "consultant.structural", "consultant.civil")
	// Civil drops its first catalogue stage; structural still has it.
	first := pkgs[1].Stages[0]
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, pkgs[1].ID, first.ID, userA, procurement.StagePatch{Version: first.Version, Retired: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[1].ID, pkgs[0].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	g := readPkg(t, s, group.ID)
	if len(g.Stages) != len(pkgs[0].Stages) {
		t.Fatalf("group stages %+v", g.Stages)
	}
	for i, l := range g.Stages {
		if l.StageID != pkgs[0].Stages[i].StageID || l.ProgramStageID != "" {
			t.Fatalf("catalogue order %d: %+v", i, l)
		}
	}
	// A program prepared later links the group's lines like any package's.
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range readPkg(t, s, group.ID).Stages {
		if l.ProgramStageID == "" || l.StageRevision != st.Program.StageRevision {
			t.Fatalf("group line not linked %+v", l)
		}
	}
	if sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, group.ID); err != nil || !sync.ProgramExists || sync.Offer || sync.BuiltFrom != st.Program.StageRevision {
		t.Fatalf("group sync %+v %v", sync, err)
	}
}

func TestAwardedGroupFollowsTheProgram(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	group, members := combinedGroup(t, s, "consultant.structural", "consultant.civil")
	table, err := s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	stKeys := groupKeys(t, table, memberKey(t, table, members[0].ID))
	if len(stKeys) == 0 {
		t.Fatal("no structural stage cells")
	}
	bid := pageText([]string{"Karrak Consulting Engineers Pty Ltd"}, []string{"ABN 77 650 369 205"}, []string{"FEE PROPOSAL"}, []string{"Structural first stage", "$5,000"})
	sub, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 9, ""), group.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, sub, bid, answers{"firm_name": "c1", "gst_basis": "ex", "price:" + stKeys[0]: "a1"}, false)
	award, err := s.Award(ctx, orgA, projectA, group.ID, userA, store.AwardInput{SubmissionID: sub, Lines: []store.AwardLine{{Label: "Structural first stage", Keys: stKeys[:1]}}})
	if err != nil {
		t.Fatal(err)
	}
	commitments := func() (sum int64, lines int) {
		t.Helper()
		plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range plan.Items {
			if v, ok := it.Values["commitment"]; ok && v.Amount != nil {
				c, _, err := invoices.ParseCents(string(*v.Amount))
				if err != nil {
					t.Fatal(err)
				}
				sum += c
			}
			if it.ParentItemID == award.ParentItemID {
				lines++
			}
		}
		return sum, lines
	}
	contract, under := commitments()
	if contract != 500000 {
		t.Fatalf("contract %d", contract)
	}

	concept := programStageID(t, st, "concept_design")
	renameStage(t, s, concept, "Sketch design")
	deleteStage(t, s, concept)
	st, added := addStage(t, s, "Feasibility")
	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, group.ID)
	if err != nil || !sync.Awarded || sync.Document != engine.DocAwardedContract {
		t.Fatalf("awarded group sync %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageAdded, added); c == nil || c.Action != engine.AddUnpricedLine {
		t.Fatalf("added %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageRemoved, concept); c == nil || c.Action != engine.MarkRemovedKeep || !c.Held {
		t.Fatalf("removed %+v", sync.Changes)
	}
	if _, err := s.SyncPackageStages(ctx, orgA, projectA, group.ID, userA, store.StageSyncUpdate, st.Program.StageRevision); err != nil {
		t.Fatal(err)
	}
	after, underAfter := commitments()
	if after != contract || underAfter != under+1 {
		t.Fatalf("contract %d -> %d, lines under the group %d -> %d", contract, after, under, underAfter)
	}
	if l := stageByProgram(readPkg(t, s, group.ID), concept); l == nil || l.RemovedFromProgram == nil {
		t.Fatalf("awarded stage not kept")
	}
}
