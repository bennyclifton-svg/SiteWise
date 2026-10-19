package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

func rowLabels(rows []store.StageRow) string {
	var out []string
	for _, r := range rows {
		label := r.Label
		if r.RemovedFromProgram {
			label += " (removed from program)"
		}
		out = append(out, label)
	}
	return strings.Join(out, ",")
}

// TestRFPStageUpdateTable: a draft RFP rewrites its stage rows; an issued
// RFP gets a new draft revision and stays as issued.
func TestRFPStageUpdateTable(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	pool := rawPool(t)
	row, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	_, r, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, pkg.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureProgram(ctx, orgA, projectA, "procurement", b.Catalog); err != nil {
		t.Fatal(err)
	}
	draft, err := s.EnsureReportDraft(ctx, orgA, r.ID, userA, "test")
	if err != nil {
		t.Fatal(err)
	}
	st, _ := s.LoadProgram(ctx, orgA, projectA)
	sync, err := s.ReadReportStageSync(ctx, orgA, r.ID)
	if err != nil || sync.Document != engine.DocDraftRFP || sync.BuiltFrom != st.Program.StageRevision || sync.Offer {
		t.Fatalf("new draft %+v %v", sync, err)
	}
	// The package's catalogue defaults, as the program names their stages:
	// construction-stage inspections landed on Construction.
	if got := rowLabels(sync.Rows); got != "Concept design,Approval documentation,Construction documentation,Construction" {
		t.Fatalf("rows %s", got)
	}
	sections := func(id string) string {
		var raw string
		if err := pool.QueryRow(ctx, `SELECT sections::text FROM report_versions WHERE org_id=$1 AND id=$2`, orgA, id).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := sections(draft.ID)

	concept := programStageID(t, st, "concept_design")
	st = renameStage(t, s, concept, "Sketch design")
	sync, err = s.ReadReportStageSync(ctx, orgA, r.ID)
	if err != nil || !sync.Offer {
		t.Fatalf("rename not offered %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageRenamed, concept); c == nil || c.Action != engine.RewriteRow {
		t.Fatalf("changes %+v", sync.Changes)
	}
	if _, err := s.SyncReportStages(ctx, orgA, r.ID, userA, "test", "update", st.Program.StageRevision+1); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale revision %v", err)
	}
	if _, err := s.SyncReportStages(ctx, orgB, r.ID, userB, "test", "update", st.Program.StageRevision); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign %v", err)
	}
	if _, err := s.ReadReportStageSync(ctx, orgB, r.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	sync, err = s.SyncReportStages(ctx, orgA, r.ID, userA, "test", "update", st.Program.StageRevision)
	if err != nil || sync.Offer || sync.VersionID != draft.ID {
		t.Fatalf("update %+v %v", sync, err)
	}
	if got := rowLabels(sync.Rows); !strings.HasPrefix(got, "Sketch design,") {
		t.Fatalf("rows not rewritten: %s", got)
	}
	// The printed sections come from the discipline's stages: unchanged.
	if sections(draft.ID) != before {
		t.Fatal("update changed the draft's sections")
	}

	// Keep on the report.
	st = renameStage(t, s, concept, "Concept")
	sync, err = s.SyncReportStages(ctx, orgA, r.ID, userA, "test", "keep", st.Program.StageRevision)
	if err != nil || sync.Offer || sync.Acknowledged != st.Program.StageRevision || !strings.HasPrefix(rowLabels(sync.Rows), "Sketch design,") {
		t.Fatalf("keep %+v %v", sync, err)
	}

	// Issue the draft directly (the issue path is tested elsewhere; RFP
	// issue checks need answers this fixture does not give).
	if _, err := pool.Exec(ctx, `UPDATE report_versions SET status='issued',snapshot='{}'::jsonb,snapshot_sha256=repeat('a',64),export_file_sha256=decode(repeat('ab',32),'hex'),issued_at=now(),issued_by=$3 WHERE org_id=$1 AND id=$2`, orgA, draft.ID, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL WHERE org_id=$1 AND id=$2`, orgA, r.ID); err != nil {
		t.Fatal(err)
	}
	issued := func() string {
		var raw string
		if err := pool.QueryRow(ctx, `SELECT to_jsonb(v)::text FROM report_versions v WHERE org_id=$1 AND id=$2`, orgA, draft.ID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	issuedBefore := issued()
	approval := programStageID(t, st, "approval_documentation")
	st = renameStage(t, s, approval, "Planning approval")
	sync, err = s.ReadReportStageSync(ctx, orgA, r.ID)
	if err != nil || sync.Document != engine.DocIssuedRFP || !sync.Offer || sync.VersionID != draft.ID {
		t.Fatalf("issued %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageRenamed, approval); c == nil || c.Action != engine.NewDraftRevision {
		t.Fatalf("issued changes %+v", sync.Changes)
	}
	sync, err = s.SyncReportStages(ctx, orgA, r.ID, userA, "test", "update", st.Program.StageRevision)
	if err != nil || sync.Document != engine.DocDraftRFP || sync.VersionID == draft.ID || sync.Offer {
		t.Fatalf("new draft revision %+v %v", sync, err)
	}
	if got := rowLabels(sync.Rows); got != "Concept,Planning approval,Construction documentation,Construction" {
		t.Fatalf("new draft rows %s", got)
	}
	if issued() != issuedBefore {
		t.Fatal("the issued RFP changed")
	}
}

// TestAwardedStagesAndFrozenBaseline: on an awarded package a rename flows
// to the package stage, an added stage becomes a new unpriced line under
// the award, a removed stage is kept and marked, contract values stay as
// priced and the frozen baseline never changes.
func TestAwardedStagesAndFrozenBaseline(t *testing.T) {
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
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err = s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
		Content: costs.Content{Code: "F01", Label: "Architect fees", LineKind: "fee", PackageID: pkg.ID, Posting: false, Origin: "user", Meaning: "stated"}})
	if err != nil {
		t.Fatal(err)
	}
	group := ""
	for _, it := range plan.Items {
		if it.Label == "Architect fees" && !it.Posting {
			group = it.ID
		}
	}
	plan, err = s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
		Content: costs.Content{ParentItemID: group, Code: "F01.1", Label: "Design stages", LineKind: "fee", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "stated"}})
	if err != nil {
		t.Fatal(err)
	}
	child := ""
	for _, it := range plan.Items {
		if it.ParentItemID == group {
			child = it.ID
		}
	}
	amount := costs.Money("42000.00")
	if plan, err = s.PutCostValue(ctx, orgA, projectA, child, userA, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "stated"}}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,'99999999-9999-4999-8999-999999999991',$2,$3,'fee.pdf','awarded','2026-10-10')`, []any{orgA, projectA, pkg.ID}},
		{`INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,'99999999-9999-4999-8999-999999999992',$2,$3,'99999999-9999-4999-8999-999999999991',$4,true,$5)`, []any{orgA, projectA, pkg.ID, group, userA}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(q.sql, err)
		}
	}
	// Freeze a baseline; the draft that follows keeps the same lines.
	if _, err = s.BaselineCostPlan(ctx, orgA, projectA, userA, store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version}); err != nil {
		t.Fatal(err)
	}
	var baselineID string
	if err := pool.QueryRow(ctx, `SELECT baseline_version_id::text FROM cost_plans WHERE org_id=$1 AND project_id=$2`, orgA, projectA).Scan(&baselineID); err != nil || baselineID == "" {
		t.Fatalf("baseline %v", err)
	}
	frozen := func() string {
		var raw string
		if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY r.cost_item_id) FROM cost_item_revisions r WHERE org_id=$1 AND plan_version_id=$2),'[]')::text||
 COALESCE((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.cost_item_id,v.metric) FROM cost_values v WHERE org_id=$1 AND plan_version_id=$2),'[]')::text`, orgA, baselineID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	contract := func() string {
		var raw string
		if err := pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(v) ORDER BY v.plan_version_id,v.cost_item_id),'[]')::text FROM cost_values v WHERE org_id=$1 AND metric='commitment'`, orgA).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	frozenBefore, contractBefore := frozen(), contract()

	concept := programStageID(t, st, "concept_design")
	documentation := programStageID(t, st, "construction_documentation")
	renameStage(t, s, concept, "Sketch design")
	deleteStage(t, s, documentation)
	st, added := addStage(t, s, "Feasibility")
	sync, err := s.ReadPackageStageSync(ctx, orgA, projectA, pkg.ID)
	if err != nil || !sync.Awarded || sync.Document != engine.DocAwardedContract {
		t.Fatalf("awarded %+v %v", sync, err)
	}
	if c := changeOf(sync, engine.StageRenamed, concept); c == nil || c.Action != engine.CopyLabel {
		t.Fatalf("rename %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageAdded, added); c == nil || c.Action != engine.AddUnpricedLine {
		t.Fatalf("added %+v", sync.Changes)
	}
	if c := changeOf(sync, engine.StageRemoved, documentation); c == nil || c.Action != engine.MarkRemovedKeep || !c.Held {
		t.Fatalf("removed %+v", sync.Changes)
	}
	if _, err := s.SyncPackageStages(ctx, orgA, projectA, pkg.ID, userA, "update", st.Program.StageRevision); err != nil {
		t.Fatal(err)
	}
	p := readPkg(t, s, pkg.ID)
	if c := stageByProgram(p, concept); c == nil || c.Label != "Sketch design" {
		t.Fatalf("rename %+v", p.Stages)
	}
	if d := stageByProgram(p, documentation); d == nil || d.RemovedFromProgram == nil {
		t.Fatalf("awarded stage removed downstream: %+v", p.Stages)
	}
	draft, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	var line *costs.Item
	for i, it := range draft.Items {
		if it.ParentItemID == group && it.Label == "Feasibility" {
			line = &draft.Items[i]
		}
	}
	if line == nil || !line.Posting || line.Code != "F01.2" {
		t.Fatalf("no unpriced line for the added stage: %+v", draft.Items)
	}
	for metric, v := range line.Values {
		if v.Amount != nil {
			t.Fatalf("added line priced: %s %+v", metric, v)
		}
	}
	if contract() != contractBefore {
		t.Fatal("contract values changed")
	}
	if frozen() != frozenBefore {
		t.Fatal("the frozen baseline changed")
	}
}
