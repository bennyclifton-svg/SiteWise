package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// Stage plan tests (program plan docs/plans/2026-10-18-program.md 11.3
// "Stage plan", 7.6): the surface that comes first creates the program,
// default stages read it, and each row of the Update table.

func programStageID(t *testing.T, st store.ProgramState, catalogue string) string {
	t.Helper()
	for _, a := range st.Activities {
		if a.Kind == engine.KindStage && string(a.CatalogueStageID) == catalogue {
			return a.ID
		}
	}
	t.Fatalf("no %s stage", catalogue)
	return ""
}

func renameStage(t *testing.T, s *store.Store, id, label string) store.ProgramState {
	t.Helper()
	st, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return apply(t, s, st, store.ProgramOp{Op: "update", ID: id, Label: &label}).ProgramState
}

func deleteStage(t *testing.T, s *store.Store, id string) store.ProgramState {
	t.Helper()
	st, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{id}}).ProgramState
}

func addStage(t *testing.T, s *store.Store, label string) (store.ProgramState, string) {
	t.Helper()
	st, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	kind := "stage"
	res := apply(t, s, st, store.ProgramOp{Op: "add", TempID: "new", Kind: &kind, Label: &label})
	return res.ProgramState, res.TempIDs["new"]
}

func changeOf(sync store.StageSync, kind, id string) *store.StageSyncChange {
	for i := range sync.Changes {
		if sync.Changes[i].Kind == kind && sync.Changes[i].ProgramStageID == id {
			return &sync.Changes[i]
		}
	}
	return nil
}

func stageByProgram(p procurement.Package, id string) *procurement.Stage {
	for i := range p.Stages {
		if p.Stages[i].ProgramStageID == id {
			return &p.Stages[i]
		}
	}
	return nil
}

func readPkg(t *testing.T, s *store.Store, id string) procurement.Package {
	t.Helper()
	items, err := s.ReadPackages(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range items {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("package %s not live", id)
	return procurement.Package{}
}

func TestStagePlanPreparedFromProcurementAndCostPlan(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	// Procurement first: an RFP on a project with no program.
	row, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Stages) == 0 || pkg.Stages[0].ProgramStageID != "" {
		t.Fatalf("with no program the catalogue defaults stay: %+v", pkg.Stages)
	}
	if _, _, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, pkg.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureProgram(ctx, orgA, projectA, "procurement", b.Catalog); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil || st.Program.CreatedBySurface != "procurement" {
		t.Fatalf("program %+v %v", st.Program, err)
	}
	// Opening Program later finds the same program, already linked.
	again, created, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil || created || again.Program.ID != st.Program.ID {
		t.Fatalf("prepare again %v %v", created, err)
	}
	pkg = readPkg(t, s, pkg.ID)
	for _, ps := range pkg.Stages {
		if ps.ProgramStageID == "" || ps.StageRevision != st.Program.StageRevision {
			t.Fatalf("stage not linked: %+v", ps)
		}
	}
	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, pkg.ID)
	if err != nil || !sync.ProgramExists || sync.BuiltFrom != st.Program.StageRevision || sync.Offer {
		t.Fatalf("sync after prepare %+v %v", sync, err)
	}
	ticked := 0
	for _, pick := range sync.Stages {
		if pick.Ticked {
			ticked++
		}
	}
	if len(sync.Stages) != len(b.Catalog.GenericStages()) || ticked == 0 {
		t.Fatalf("picker %+v", sync.Stages)
	}
	// A second EnsureProgram is a no-op.
	if err := s.EnsureProgram(ctx, orgA, projectA, "cost_plan", b.Catalog); err != nil {
		t.Fatal(err)
	}
	if st2, _ := s.LoadProgram(ctx, orgA, projectA); st2.Program.Version != st.Program.Version || st2.Program.CreatedBySurface != "procurement" {
		t.Fatal("ensure changed an existing program")
	}

	// Cost Plan first, on the other org's project.
	if err := s.EnsureProgram(ctx, orgB, projectB, "cost_plan", b.Catalog); err != nil {
		t.Fatal(err)
	}
	other, err := s.LoadProgram(ctx, orgB, projectB)
	if err != nil || other.Program.CreatedBySurface != "cost_plan" || other.Program.ID == st.Program.ID {
		t.Fatalf("cost plan program %+v %v", other.Program, err)
	}
	// Another org never creates one on this project.
	if err := s.EnsureProgram(ctx, orgB, projectA, "procurement", b.Catalog); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign ensure %v", err)
	}
}

func TestDefaultStagesReadTheProgram(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	concept := programStageID(t, st, "concept_design")
	st = renameStage(t, s, concept, "Sketch design")
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	// sk.generic has three of the catalogue's design sub-stages as program
	// stages; construction-stage inspections is not a program stage there.
	var got []string
	for _, ps := range pkg.Stages {
		got = append(got, ps.Label)
		if ps.ProgramStageID == "" || ps.StageRevision != st.Program.StageRevision || ps.Origin != "calculation" {
			t.Fatalf("stage %+v", ps)
		}
	}
	if strings.Join(got, ",") != "Sketch design,Approval documentation,Construction documentation" {
		t.Fatalf("labels %v", got)
	}
	if pkg.Stages[0].ProgramStageID != concept || pkg.Stages[0].StageID != "concept_design" {
		t.Fatalf("first stage %+v", pkg.Stages[0])
	}
	// A works package still has no stages.
	works, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "works", WorksScope: "trade", Title: "Roofing", LifecycleStatus: "planned"})
	if err != nil || len(works.Stages) != 0 {
		t.Fatalf("works %+v %v", works.Stages, err)
	}
	// Ticking a program stage in the picker takes its catalogue stage and
	// label from the program, whatever the caller sent.
	construction := programStageID(t, st, "construction")
	ticked, err := s.CreatePackageStage(ctx, orgA, projectA, pkg.ID, userA, procurement.Stage{StageID: "defects", Label: "Mine", NovationPhase: "none", ProgramStageID: construction})
	if err != nil || ticked.StageID != "construction" || ticked.Label != "Construction" || ticked.ProgramStageID != construction {
		t.Fatalf("tick %+v %v", ticked, err)
	}
	if _, err := s.CreatePackageStage(ctx, orgA, projectA, pkg.ID, userA, procurement.Stage{StageID: "construction", Label: "X", NovationPhase: "none", ProgramStageID: newStageUUID()}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown program stage %v", err)
	}
	// Unticking retires it through the existing path.
	if _, err := s.PatchPackageStage(ctx, orgA, projectA, pkg.ID, ticked.ID, userA, procurement.StagePatch{Version: ticked.Version, Retired: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if stageByProgram(readPkg(t, s, pkg.ID), construction) != nil {
		t.Fatal("untick left the stage")
	}
}

func newStageUUID() string { return "77777777-7777-4777-8777-777777777777" }

func boolPtr(v bool) *bool { return &v }

// TestPackageStageUpdateTable walks the package rows of the 7.6 table.
func TestPackageStageUpdateTable(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	pool := rawPool(t)
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	concept := programStageID(t, st, "concept_design")
	approval := programStageID(t, st, "approval_documentation")
	documentation := programStageID(t, st, "construction_documentation")
	// Money on Approval documentation: an invoice charged to its fee line.
	// The line itself is excluded, so only the invoice holds the stage.
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	approvalStage := stageByProgram(pkg, approval)
	plan, err = s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
		Content: costs.Content{Label: "Approval fee", LineKind: "fee", PackageID: pkg.ID, PackageStageID: approvalStage.ID, Posting: true, Origin: "user", Meaning: "allowance"}})
	if err != nil {
		t.Fatal(err)
	}
	var line string
	for _, it := range plan.Items {
		if it.PackageStageID == approvalStage.ID {
			line = it.ID
		}
	}
	file := "88888888-8888-4888-8888-888888888881"
	sum := sha256.Sum256([]byte(file))
	if err := s.CreateFile(ctx, orgA, store.File{ID: file, ProjectID: projectA, SHA256: sum[:], ByteSize: 1, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO invoices(org_id,id,project_id,file_id,filename,status,period) VALUES($1,'88888888-8888-4888-8888-888888888882',$2,$3,'inv.pdf','filed','2026-10-01')`, []any{orgA, projectA, file}},
		{`INSERT INTO invoice_allocations(org_id,id,project_id,invoice_id,cost_item_id,amount_cents,decided_by,band) VALUES($1,'88888888-8888-4888-8888-888888888883',$2,'88888888-8888-4888-8888-888888888882',$3,100000,'user','green')`, []any{orgA, projectA, line}},
		{`UPDATE cost_item_revisions SET excluded=true WHERE org_id=$1 AND cost_item_id=$2`, []any{orgA, line}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(q.sql, err)
		}
	}
	// A proposal received for the package.
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,'88888888-8888-4888-8888-888888888884',$2,$3,'fee.pdf','received','2026-10-10')`, orgA, projectA, pkg.ID); err != nil {
		t.Fatal(err)
	}
	baseline := proposalSnapshot(t, pool)

	// Program changes: rename Concept, delete Documentation (nothing on it)
	// and Approval documentation (invoiced), add a new stage.
	renameStage(t, s, concept, "Sketch design")
	deleteStage(t, s, documentation)
	deleteStage(t, s, approval)
	st, added := addStage(t, s, "Feasibility")

	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, pkg.ID)
	if err != nil || !sync.Offer || sync.Revision != st.Program.StageRevision {
		t.Fatalf("offer %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageRenamed, concept); c == nil || c.Action != engine.CopyLabel || c.OldLabel != "Concept design" || c.NewLabel != "Sketch design" {
		t.Fatalf("rename %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageRemoved, documentation); c == nil || c.Action != engine.Retire || c.Held {
		t.Fatalf("removed %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageRemoved, approval); c == nil || c.Action != engine.MarkRemovedKeep || !c.Held {
		t.Fatalf("invoiced %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageAdded, added); c == nil || c.Action != engine.OfferUnticked {
		t.Fatalf("added %+v", sync.Changes)
	}
	if sync.ProposalsFlagged != 1 {
		t.Fatalf("proposals flagged %d", sync.ProposalsFlagged)
	}
	// A rename is never a delete and an add.
	if changeOf(sync, engine.StageRemoved, concept) != nil || changeOf(sync, engine.StageAdded, concept) != nil {
		t.Fatal("rename read as delete and add")
	}

	// Version pin.
	if _, err := s.SyncPackageStages(ctx, orgA, projectA, pkg.ID, userA, "update", st.Program.StageRevision-1); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale revision %v", err)
	}
	// Another org.
	if _, err := s.SyncPackageStages(ctx, orgB, projectA, pkg.ID, userB, "update", st.Program.StageRevision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign %v", err)
	}
	if _, err := s.ReadPackageStageSync(ctx, orgB, projectA, pkg.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}

	after, err := s.SyncPackageStages(ctx, orgA, projectA, pkg.ID, userA, "update", st.Program.StageRevision)
	if err != nil {
		t.Fatal(err)
	}
	if after.Offer || len(after.Changes) != 0 || after.BuiltFrom != st.Program.StageRevision {
		t.Fatalf("after update %+v", after)
	}
	pkg = readPkg(t, s, pkg.ID)
	if c := stageByProgram(pkg, concept); c == nil || c.Label != "Sketch design" {
		t.Fatalf("rename not copied %+v", pkg.Stages)
	}
	if stageByProgram(pkg, documentation) != nil {
		t.Fatal("removed stage with nothing on it kept")
	}
	if a := stageByProgram(pkg, approval); a == nil || a.RemovedFromProgram == nil {
		t.Fatalf("invoiced stage not kept and marked: %+v", pkg.Stages)
	}
	if stageByProgram(pkg, added) != nil {
		t.Fatal("added stage ticked by Update")
	}
	var pick *store.PickerStage
	for i := range after.Stages {
		if after.Stages[i].ProgramStageID == added {
			pick = &after.Stages[i]
		}
	}
	if pick == nil || pick.Ticked || pick.Label != "Feasibility" {
		t.Fatalf("added stage not offered unticked: %+v", after.Stages)
	}
	if len(after.Kept) != 1 || after.Kept[0].ProgramStageID != approval {
		t.Fatalf("kept %+v", after.Kept)
	}
	// Proposals received never change.
	if proposalSnapshot(t, pool) != baseline {
		t.Fatal("a proposal received changed")
	}

	// Keep: acknowledged, not offered again until the stages change again.
	renameStage(t, s, concept, "Concept")
	st, _ = s.LoadProgram(ctx, orgA, projectA)
	if sync, _ = s.ReadPackageStageSync(ctx, orgA, projectA, pkg.ID); !sync.Offer {
		t.Fatal("second rename not offered")
	}
	kept, err := s.SyncPackageStages(ctx, orgA, projectA, pkg.ID, userA, "keep", st.Program.StageRevision)
	if err != nil || kept.Offer || kept.Acknowledged != st.Program.StageRevision {
		t.Fatalf("keep %+v %v", kept, err)
	}
	if c := stageByProgram(readPkg(t, s, pkg.ID), concept); c.Label != "Sketch design" {
		t.Fatal("keep changed a label")
	}
	renameStage(t, s, concept, "Concept design")
	if sync, _ = s.ReadPackageStageSync(ctx, orgA, projectA, pkg.ID); !sync.Offer {
		t.Fatal("a newer change after Keep is not offered")
	}
}

// proposalSnapshot is every proposal row and price cell of org A.
func proposalSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var raw string
	if err := pool.QueryRow(context.Background(), `SELECT COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY s.id) FROM submissions s WHERE org_id=$1),'[]')::text||
 COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id) FROM submission_prices p WHERE org_id=$1),'[]')::text`, orgA).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	return raw
}
