package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
)

func TestReportDraftAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Report project")
	var p procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Engineering"}`), http.StatusCreated, &p)
	body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": p.ID})
	var r store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", body, http.StatusCreated, &r)
	base := "/api/reports/" + r.ID
	var listed []store.Report
	m.getJSON(t, "/api/projects/"+project+"/reports", &listed)
	if len(listed) != 1 || listed[0].ID != r.ID {
		t.Fatal("saved report missing from list", listed)
	}
	var view store.ReportView
	m.getJSON(t, base, &view)
	if view.Draft != nil {
		t.Fatal("unrequested assembly")
	}
	var d store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	if len(d.Sections) != 7 || d.SourceRevisions.AppBuild == "" {
		t.Fatal("missing draft content/build")
	}
	editURL := base + "/edits/" + url.PathEscape("package:"+p.ID)
	edit, _ := json.Marshal(map[string]any{"version": d.Version, "text": "Protected wording"})
	m.call(t, http.MethodPut, editURL, edit, http.StatusOK, &d)
	m.call(t, http.MethodPut, editURL, edit, http.StatusConflict, nil)
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, "/api/projects/"+project+"/reports", nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/draft", []byte(`{}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPut, editURL, edit, http.StatusNotFound, nil)
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true,"issue":true}`), http.StatusBadRequest, nil)
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	m.getJSON(t, base, &view)
	if len(view.Edits) != 1 || view.Edits[0].Text != "Protected wording" {
		t.Fatal("edit lost", view.Edits)
	}
	reset, _ := json.Marshal(map[string]int64{"version": d.Version})
	foreign.call(t, http.MethodDelete, editURL, reset, http.StatusNotFound, nil)
	stale, _ := json.Marshal(map[string]int64{"version": d.Version - 1})
	m.call(t, http.MethodDelete, editURL, stale, http.StatusConflict, nil)
	m.call(t, http.MethodDelete, editURL, reset, http.StatusOK, &d)
	m.getJSON(t, base, &view)
	if len(view.Edits) != 0 {
		t.Fatal("edit not reset")
	}
	for _, section := range d.Sections {
		for _, block := range section.Blocks {
			if block.ID == "package:"+p.ID && (block.Text != "Engineering — planned" || block.Edited) {
				t.Fatal("wrong restored wording", block)
			}
		}
	}
	for _, ref := range d.References {
		if ref.Basis.ProtectedEdit {
			t.Fatal("old user citation retained", ref)
		}
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("report called Jev")
	}
}
