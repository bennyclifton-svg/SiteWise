package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

// Stage-sync routes (program plan 7.6, 8; WP-P10): procurement or the cost
// plan creates the program when it comes first; the RFP draft and the
// package offer Update or Keep after a stage changes in Program.

func stageOf(t *testing.T, v programView, catalogue string) store.ProgramActivity {
	t.Helper()
	for _, a := range v.Activities {
		if a.Kind == engine.KindStage && string(a.CatalogueStageID) == catalogue {
			return a
		}
	}
	t.Fatalf("no %s stage", catalogue)
	return store.ProgramActivity{}
}

func renameOp(t *testing.T, m *member, project string, v programView, id, label string) programView {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"program_id": v.Program.ID, "version": v.Program.Version, "ops": []map[string]any{{"op": "update", "id": id, "label": label}}})
	var out programView
	m.call(t, http.MethodPost, "/api/projects/"+project+"/program/ops", body, http.StatusOK, &out)
	return out
}

func syncBody(choice string, revision int64) []byte {
	raw, _ := json.Marshal(map[string]any{"choice": choice, "revision": revision})
	return raw
}

func TestStageSyncRoutesProcurementFirst(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Stage plan")
	base := "/api/projects/" + project
	jevBefore := a.jevHits.Load()
	m.call(t, http.MethodGet, base+"/program", nil, http.StatusNotFound, nil)

	var pkg procurement.Package
	m.call(t, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &pkg)
	var created struct {
		Report store.Report `json:"report"`
	}
	m.call(t, http.MethodPost, base+"/procurement/create", []byte(`{"package_id":"`+pkg.ID+`"}`), http.StatusCreated, &created)
	// Creating the RFP made the program; opening Program shows its stages.
	var v programView
	m.call(t, http.MethodGet, base+"/program", nil, http.StatusOK, &v)
	if v.Program.CreatedBySurface != "procurement" {
		t.Fatalf("program %+v", v.Program)
	}
	reportSync := "/api/reports/" + created.Report.ID + "/stage-sync"
	var sync store.StageSync
	m.call(t, http.MethodGet, reportSync, nil, http.StatusOK, &sync)
	if sync.Document != engine.DocDraftRFP || sync.BuiltFrom != v.Program.StageRevision || sync.Offer || len(sync.Rows) == 0 {
		t.Fatalf("draft sync %+v", sync)
	}
	concept := stageOf(t, v, "concept_design")
	if sync.Rows[0].ProgramStageID != concept.ID || sync.Rows[0].Label != concept.Label {
		t.Fatalf("rows %+v", sync.Rows)
	}

	// Rename in Program: the draft offers Update or Keep.
	v = renameOp(t, m, project, v, concept.ID, "Sketch design")
	m.call(t, http.MethodGet, reportSync, nil, http.StatusOK, &sync)
	if !sync.Offer || len(sync.Changes) != 1 || sync.Changes[0].Kind != engine.StageRenamed || sync.Changes[0].NewLabel != "Sketch design" {
		t.Fatalf("offer %+v", sync)
	}
	m.call(t, http.MethodPost, reportSync, []byte(`{"choice":"maybe","revision":1}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, reportSync, []byte(`{"choice":"update"}`), http.StatusUnprocessableEntity, nil)
	var conflict struct {
		Current store.StageSync `json:"current"`
	}
	m.call(t, http.MethodPost, reportSync, syncBody("update", v.Program.StageRevision-1), http.StatusConflict, &conflict)
	if !conflict.Current.Offer || conflict.Current.Revision != v.Program.StageRevision {
		t.Fatalf("conflict %+v", conflict.Current)
	}
	m.call(t, http.MethodPost, reportSync, syncBody("update", v.Program.StageRevision), http.StatusOK, &sync)
	if sync.Offer || sync.Rows[0].Label != "Sketch design" {
		t.Fatalf("update %+v", sync)
	}

	// The package: picker, tick, untick, and its own offer.
	pkgSync := base + "/packages/" + pkg.ID + "/stage-sync"
	m.call(t, http.MethodGet, pkgSync, nil, http.StatusOK, &sync)
	if !sync.Offer || sync.Changes[0].Action != engine.CopyLabel {
		t.Fatalf("package offer %+v", sync)
	}
	m.call(t, http.MethodPost, pkgSync, syncBody("keep", v.Program.StageRevision), http.StatusOK, &sync)
	if sync.Offer || sync.Acknowledged != v.Program.StageRevision {
		t.Fatalf("keep %+v", sync)
	}
	var defects store.PickerStage
	for _, s := range sync.Stages {
		if s.CatalogueStageID == "defects" {
			defects = s
		}
	}
	if defects.ProgramStageID == "" || defects.Ticked {
		t.Fatalf("picker %+v", sync.Stages)
	}
	var ticked procurement.Stage
	m.call(t, http.MethodPost, base+"/packages/"+pkg.ID+"/stages", []byte(`{"program_stage_id":"`+defects.ProgramStageID+`"}`), http.StatusCreated, &ticked)
	if ticked.ProgramStageID != defects.ProgramStageID || ticked.Label != defects.Label || ticked.StageID != "defects" {
		t.Fatalf("tick %+v", ticked)
	}
	m.call(t, http.MethodPatch, base+"/packages/"+pkg.ID+"/stages/"+ticked.ID, []byte(`{"version":1,"retired":true}`), http.StatusOK, nil)
	m.call(t, http.MethodGet, pkgSync, nil, http.StatusOK, &sync)
	for _, s := range sync.Stages {
		if s.ProgramStageID == defects.ProgramStageID && s.Ticked {
			t.Fatal("untick left the stage ticked")
		}
	}

	// stage_sync_write: Update on one RFP draft, recomputed document
	// returned, measured through the real HTTP server.
	var samples []time.Duration
	for i := 0; i < 21; i++ {
		v = renameOp(t, m, project, v, concept.ID, []string{"Concept", "Sketch design"}[i%2])
		start := time.Now()
		resp := m.do(t, http.MethodPost, reportSync, syncBody("update", v.Program.StageRevision))
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		samples = append(samples, time.Since(start))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update %d", resp.StatusCode)
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p50, p90 := samples[len(samples)/2], samples[len(samples)*9/10]
	t.Logf("stage_sync_write over HTTP: p50 %v, p90 %v (budget 100 ms / 250 ms)", p50, p90)
	if p50 > 100*time.Millisecond || p90 > 250*time.Millisecond {
		t.Fatalf("stage_sync_write over budget: p50 %v p90 %v", p50, p90)
	}
	if a.jevHits.Load() != jevBefore {
		t.Fatal("stage sync called Jev")
	}
}

func TestFeeLineByStageCreatesTheProgram(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Cost plan first")
	base := "/api/projects/" + project
	var pkg procurement.Package
	m.call(t, http.MethodPost, base+"/packages", []byte(`{"kind":"services","title":"Structural"}`), http.StatusCreated, &pkg)
	m.call(t, http.MethodGet, base+"/program", nil, http.StatusNotFound, nil)
	resp := m.do(t, http.MethodPost, base+"/cost-plan/prepare", []byte(`{}`))
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		t.Fatalf("prepare %d", resp.StatusCode)
	}
	var plan costs.Plan
	m.getJSON(t, base+"/cost-plan", &plan)
	body, _ := json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version, "label": "Structural concept", "line_kind": "fee",
		"package_id": pkg.ID, "package_stage_id": pkg.Stages[0].ID, "posting": true, "origin": "user", "meaning": "allowance"})
	m.call(t, http.MethodPost, base+"/cost-plan/items", body, http.StatusOK, nil)
	var v programView
	m.call(t, http.MethodGet, base+"/program", nil, http.StatusOK, &v)
	if v.Program.CreatedBySurface != "cost_plan" {
		t.Fatalf("program %+v", v.Program)
	}
	var sync store.StageSync
	m.call(t, http.MethodGet, base+"/packages/"+pkg.ID+"/stage-sync", nil, http.StatusOK, &sync)
	if !sync.ProgramExists || sync.BuiltFrom != v.Program.StageRevision {
		t.Fatalf("package sync %+v", sync)
	}
}

// Creating the first RFP now also prepares the program. Measured against
// the route's report_write budget (p50 100 ms, p90 250 ms) on fresh
// projects, so every sample includes the prepare.
func TestProcurementCreateWithPrepareTiming(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	var samples []time.Duration
	for i := 0; i < 11; i++ {
		project := m.createProject(t, "Create timing")
		base := "/api/projects/" + project
		var pkg procurement.Package
		m.call(t, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &pkg)
		start := time.Now()
		m.call(t, http.MethodPost, base+"/procurement/create", []byte(`{"package_id":"`+pkg.ID+`"}`), http.StatusCreated, nil)
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p50, p90 := samples[len(samples)/2], samples[len(samples)*9/10]
	t.Logf("procurement create with prepare: p50 %v, p90 %v (report_write 100 ms / 250 ms)", p50, p90)
	if p90 > 250*time.Millisecond {
		t.Fatalf("over budget: p50 %v p90 %v", p50, p90)
	}
}
