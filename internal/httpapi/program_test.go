package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/httpapi"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// programView is a program route's response. Computed is read without its
// per-row times, which only marshal (the browser reads them by id).
type programView struct {
	store.ProgramView
	Computed struct {
		Status engine.Status `json:"status"`
		Finish engine.Finish `json:"finish"`
	} `json:"computed"`
	Created bool `json:"created"`
}

type programErrorBody struct {
	Error   string          `json:"error"`
	Field   string          `json:"field"`
	OpIndex int             `json:"op_index"`
	Cycle   []string        `json:"cycle"`
	Held    []string        `json:"held"`
	Current json.RawMessage `json:"current"`
}

func (m *member) programError(t *testing.T, method, path string, body []byte, want int) programErrorBody {
	t.Helper()
	resp := m.do(t, method, path, body)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	var out programErrorBody
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	return out
}

func programRow(t *testing.T, v programView, label string) store.ProgramActivity {
	t.Helper()
	for _, a := range v.Activities {
		if a.Label == label {
			return a
		}
	}
	t.Fatalf("no row %q", label)
	return store.ProgramActivity{}
}

func TestProgramRoutes(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program routes")
	base := "/api/projects/" + project + "/program"

	// No program yet: a 404 that says so, for this org's project only.
	resp := m.do(t, "GET", base, nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 404 || strings.TrimSpace(string(raw)) != `{"empty":true}` {
		t.Fatalf("empty program: %d %s", resp.StatusCode, raw)
	}
	var pkg procurement.Package
	m.call(t, "POST", "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect"}`), 201, &pkg)
	if len(pkg.Stages) == 0 {
		t.Fatal("services package has no stages")
	}

	if e := m.programError(t, "POST", base+"/prepare", []byte(`{"surface":"elsewhere"}`), 422); e.Field != "surface" {
		t.Fatalf("surface %+v", e)
	}
	var v programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"program"}`), 200, &v)
	if !v.Created || v.Program.CreatedBySurface != "program" || v.Program.Version != 1 {
		t.Fatalf("prepare %+v", v.Program)
	}
	var again programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"procurement"}`), 200, &again)
	if again.Created || again.Program.ID != v.Program.ID || again.Program.Version != 1 || again.Program.CreatedBySurface != "program" {
		t.Fatalf("second prepare %+v", again.Program)
	}
	// sk.generic stages and one activity per package stage, every duration
	// unknown and the status saying so (plan 13.1).
	var stages, packageRows int
	for _, act := range v.Activities {
		if act.Kind == engine.KindStage {
			stages++
		}
		if act.PackageStageID != "" {
			packageRows++
		}
		if act.DurationDays != nil {
			t.Fatalf("a duration came from nowhere: %+v", act)
		}
	}
	if stages == 0 || packageRows != len(pkg.Stages) || v.Skeleton.ID != "sk.generic" || v.Question != nil {
		t.Fatalf("stages %d package rows %d/%d skeleton %+v", stages, packageRows, len(pkg.Stages), v.Skeleton)
	}
	if want := fmt.Sprintf("Indicative: 0 of %d durations known", v.Computed.Status.Total); v.Computed.Status.Words != want || v.Computed.Status.Total == 0 {
		t.Fatalf("status %+v", v.Computed.Status)
	}
	var got programView
	m.getJSON(t, base, &got)
	if got.Program.ID != v.Program.ID || len(got.Activities) != len(v.Activities) || got.ProgramFeedback != nil || got.Checks == nil {
		t.Fatalf("GET %+v", got.Program)
	}
	// The 8.2 keys the frontend reads, with feedback only after a write.
	raw = m.raw(t, "GET", base)
	for _, key := range []string{`"program":{"id"`, `"activities":[`, `"links":[`, `"calendar":{"days":`, `"computed":{"status"`, `"checks":[`, `"question":null`, `"start_suggestion":null`, `"skeleton":{"id":"sk.generic"`, `"readings_pending":0`, `"changes_pending":0`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("GET has no %s: %.400s", key, raw)
		}
	}
	if strings.Contains(string(raw), `"toasts"`) {
		t.Fatal("a read carries op feedback")
	}

	construction := ""
	for _, act := range v.Activities {
		if act.Kind == engine.KindStage && act.CatalogueStageID == "construction" {
			construction = act.ID
		}
	}
	ops := func(version int64, list ...map[string]any) []byte {
		body, _ := json.Marshal(map[string]any{"program_id": v.Program.ID, "version": version, "ops": list})
		return body
	}
	var w programView
	m.call(t, "POST", base+"/ops", ops(1, map[string]any{"op": "add", "temp_id": "t1", "parent_id": construction, "kind": "activity", "label": "Slab", "duration_days": 10}), 200, &w)
	if w.Program.Version != 2 || w.ProgramFeedback == nil || w.TempIDs["t1"] == "" || programRow(t, w, "Slab").ID != w.TempIDs["t1"] {
		t.Fatalf("add %+v %+v", w.Program, w.ProgramFeedback)
	}
	slab := w.TempIDs["t1"]

	// 422s name the field and the op.
	for _, c := range []struct {
		name, field string
		op          map[string]any
	}{
		{"label", "label", map[string]any{"op": "update", "id": slab, "label": "   "}},
		{"lag", "lag_days", map[string]any{"op": "link_add", "pred_id": construction, "succ_id": slab, "type": "FS", "lag_days": 400}},
		{"date", "start_date", map[string]any{"op": "set_start", "start_date": "2027-02-30"}},
		{"foreign id", "id", map[string]any{"op": "update", "id": newUUID(t), "label": "Intruder"}},
	} {
		e := m.programError(t, "POST", base+"/ops", ops(2, c.op), 422)
		if e.Field != c.field || e.OpIndex != 0 || e.Error == "" {
			t.Fatalf("%s: %+v", c.name, e)
		}
	}
	many := make([]map[string]any, store.MaxProgramOps+1)
	for i := range many {
		many[i] = map[string]any{"op": "set_start", "start_date": nil}
	}
	if e := m.programError(t, "POST", base+"/ops", ops(2, many...), 422); e.Field != "ops" || e.OpIndex != -1 || e.Error != store.MsgTooManyOps {
		t.Fatalf("too many ops %+v", e)
	}
	// A stale version is 409 with the current state.
	e := m.programError(t, "POST", base+"/ops", ops(1, map[string]any{"op": "update", "id": slab, "label": "Late"}), 409)
	var current programView
	if err := json.Unmarshal(e.Current, &current); err != nil || current.Program.Version != 2 || programRow(t, current, "Slab").ID != slab {
		t.Fatalf("conflict %+v %v", e, err)
	}
	if e.Error != "Someone else changed the program; showing their version." {
		t.Fatal(e.Error)
	}
	// Another program's id is not this one's.
	other := "/api/projects/" + m.createProject(t, "Other program") + "/program"
	m.call(t, "POST", other+"/prepare", []byte(`{"surface":"cost_plan"}`), 200, nil)
	if resp := m.do(t, "POST", other+"/ops", ops(1, map[string]any{"op": "set_start", "start_date": "2027-03-01"})); resp.StatusCode != 404 {
		t.Fatalf("program id of another project: %d", resp.StatusCode)
	}

	// Undo, redo, accept all.
	m.call(t, "POST", base+"/undo", []byte(`{"version":2}`), 200, &w)
	if w.Program.Version != 3 || hasLabel(w, "Slab") {
		t.Fatalf("undo %+v", w.Program)
	}
	m.call(t, "POST", base+"/redo", []byte(`{"version":3}`), 200, &w)
	if w.Program.Version != 4 || programRow(t, w, "Slab").ID != slab {
		t.Fatalf("redo %+v", w.Program)
	}
	if e := m.programError(t, "POST", base+"/redo", []byte(`{"version":4}`), 422); e.Error != store.MsgNothingToRedo {
		t.Fatalf("nothing to redo %+v", e)
	}
	m.call(t, "POST", base+"/accept", []byte(`{"version":4,"all":true}`), 200, &w)
	for _, act := range w.Activities {
		if act.ReviewStatus != "accepted" {
			t.Fatalf("not accepted %+v", act)
		}
	}
	if e := m.programError(t, "POST", base+"/accept", []byte(`{"version":5,"ids":["not-an-id"]}`), 422); e.Field != "ids" {
		t.Fatalf("bad id %+v", e)
	}

	// Every committed change was announced with its version.
	events, err := a.store.EventsAfter(ctx, m.orgID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	var versions []int64
	for _, ev := range events {
		if ev.Kind != "program.changed" {
			continue
		}
		var p struct {
			ProjectID string `json:"project_id"`
			ProgramID string `json:"program_id"`
			Version   int64  `json:"version"`
		}
		if err := json.Unmarshal([]byte(ev.Payload), &p); err != nil {
			t.Fatal(err)
		}
		if p.ProjectID == project && p.ProgramID == v.Program.ID {
			versions = append(versions, p.Version)
		}
	}
	if fmt.Sprint(versions) != "[1 2 3 4 5]" {
		t.Fatalf("program events %v", versions)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("a program route called Jev")
	}
}

func hasLabel(v programView, label string) bool {
	for _, a := range v.Activities {
		if a.Label == label {
			return true
		}
	}
	return false
}

func (m *member) raw(t *testing.T, method, path string) []byte {
	t.Helper()
	resp := m.do(t, method, path, nil)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s = %d: %s", method, path, resp.StatusCode, raw)
	}
	return raw
}

// Test fixture only: an invented fragment and duration. An unanswered
// flood site omits the fragment, and the program asks no profile question.
const programTestSrc = `[{design: test-fixture-only, anchor: "# Test"}]`

var programQuestionFiles = map[string]string{
	"fragments.yaml": `version: 1
status: draft
sources: ` + programTestSrc + `
fragments:
  - id: fr.flood-check
    label: Test flood check
    stage: construction
    systems: [structure]
    applies_when: {det: flood_hazard, is: true}
    activities:
      - {key: check, label: Flood check, systems: [structure], duration: du.check}
    status: draft
    sources: ` + programTestSrc + `
`,
	"durations.yaml": `version: 1
status: draft
sources: ` + programTestSrc + `
durations:
  - id: du.check
    version: 1
    basis: fixed
    unit_days: working
    low: "3"
    typical: "5"
    high: "8"
    geography: Test
    limitations: []
    status: reviewed
    sources: ` + programTestSrc + `
`,
}

var (
	programCatOnce sync.Once
	programCat     *knowledge.Catalog
	programCatErr  error
)

func programQuestionCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	programCatOnce.Do(func() {
		repo := filepath.Join("..", "..", "knowledge")
		root, err := os.MkdirTemp("", "program-question-")
		if err != nil {
			programCatErr = err
			return
		}
		defer os.RemoveAll(root)
		err = filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repo, path)
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(root, rel), 0o755)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(root, rel), body, 0o600)
		})
		for name, body := range programQuestionFiles {
			if err == nil {
				err = os.WriteFile(filepath.Join(root, "program", name), []byte(body), 0o600)
			}
		}
		if err == nil {
			programCat, err = knowledge.Load(root)
		}
		programCatErr = err
	})
	if programCatErr != nil {
		t.Fatal(programCatErr)
	}
	return programCat
}

func TestProgramQuestionRoute(t *testing.T) {
	cat := programQuestionCatalog(t)
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.Knowledge = cat })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program question")
	var p profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	body, _ := json.Marshal(map[string]any{"title": "Frame", "part_id": p.Parts[0].ID, "system_id": "structure", "action": "new"})
	m.call(t, "POST", "/api/projects/"+project+"/works", body, 201, new(works.Item))
	base := "/api/projects/" + project + "/program"
	var v programView
	m.call(t, "POST", base+"/prepare", []byte(`{"surface":"program"}`), 200, &v)
	if v.Question != nil {
		t.Fatalf("program asked a profile question: %+v", v.Question)
	}
	if v.Program.StartDate == "" {
		t.Fatal("program opened with no start date")
	}
	for _, act := range v.Activities {
		if strings.Contains(act.Label, "flood") || strings.Contains(act.Label, "Flood") {
			t.Fatalf("unanswered flood hazard added %+v", act)
		}
	}
	ask, _ := json.Marshal(map[string]any{"version": v.Program.Version, "key": "flood_hazard", "value": "true"})
	if e := m.programError(t, "POST", base+"/question", ask, 422); e.Field != "key" {
		t.Fatalf("question still accepted %+v", e)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("building the program called Jev")
	}
}
