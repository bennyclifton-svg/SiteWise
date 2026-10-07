package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/procurement"
	"testing"
)

func TestPackagesCreateAndListAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Packages")
	path := "/api/projects/" + project + "/packages"
	body := []byte(`{"kind":"services","title":"Structural engineering","novation":true}`)
	var p procurement.Package
	m.call(t, http.MethodPost, path, body, http.StatusCreated, &p)
	if p.ID == "" || p.LifecycleStatus != "planned" || p.ReviewStatus != "accepted_for_planning" || len(p.Stages) != 4 {
		t.Fatalf("package %+v", p)
	}
	var view struct {
		Items []procurement.Package `json:"items"`
	}
	m.getJSON(t, path, &view)
	if len(view.Items) != 1 || view.Items[0].ID != p.ID {
		t.Fatal(view)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, path, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, path, body, http.StatusNotFound, nil)
	for _, bad := range []string{
		`{"kind":"works","title":"Builder","works_scope":"head_contract","novation":true}`,
		`{"kind":"works","title":"Builder"}`,
		`{"kind":"services","title":"Design","stages":[{"stage_id":"unknown","label":"Unknown"}]}`,
		`{"kind":"services","title":"Design","stages":[{"stage_id":"concept_design","label":"Concept","novation_phase":"pre"}]}`,
	} {
		m.call(t, http.MethodPost, path, []byte(bad), http.StatusUnprocessableEntity, nil)
	}
	m.call(t, http.MethodPost, path, []byte(`{"kind":"services","title":"Design","review_status":"verified"}`), http.StatusBadRequest, nil)
	duplicates := map[string]any{"kind": "services", "title": "Duplicate stages", "stages": []any{map[string]any{"stage_id": "design", "label": "Design"}, map[string]any{"stage_id": "design", "label": "Design"}}}
	raw, _ := json.Marshal(duplicates)
	m.call(t, http.MethodPost, path, raw, http.StatusConflict, nil)
	m.getJSON(t, path, &view)
	if len(view.Items) != 1 {
		t.Fatal("failed creation left partial package", view)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("package CRUD called Jev")
	}
	editPath := path + "/" + p.ID
	foreign.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"title":"Other"}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"title":"Updated engineering"}`), http.StatusOK, &p)
	if p.Version != 2 || p.Title != "Updated engineering" {
		t.Fatal(p)
	}
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"title":"Stale"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":2,"novation":false}`), http.StatusUnprocessableEntity, nil)
	stagePath := editPath + "/stages/" + p.Stages[0].ID
	var stage procurement.Stage
	m.call(t, http.MethodPatch, stagePath, []byte(`{"version":1,"label":"Renamed concept"}`), http.StatusOK, &stage)
	if stage.Version != 2 || stage.Label != "Renamed concept" {
		t.Fatal(stage)
	}
	m.call(t, http.MethodPatch, stagePath, []byte(`{"version":1,"label":"Stale"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPost, editPath+"/stages", []byte(`{"stage_id":"completion","label":"Handover","ordinal":5,"novation_phase":"post"}`), http.StatusCreated, &stage)
	if stage.PackageID != p.ID || stage.Origin != "user" {
		t.Fatal(stage)
	}
	m.call(t, http.MethodPatch, editPath+"/stages/"+stage.ID, []byte(`{"version":1,"retired":true}`), http.StatusOK, &stage)
	if stage.RetiredAt == nil {
		t.Fatal("stage not retired")
	}
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":2,"retired":true}`), http.StatusOK, &p)
	if p.RetiredAt == nil || p.Version != 3 {
		t.Fatal("package not retired", p)
	}
	m.getJSON(t, path, &view)
	if len(view.Items) != 0 {
		t.Fatal("retired package remains active")
	}
	other := m.createProject(t, "Other project")
	m.call(t, http.MethodPatch, "/api/projects/"+other+"/packages/"+p.ID, []byte(`{"version":2,"title":"Wrong project"}`), http.StatusNotFound, nil)
}
