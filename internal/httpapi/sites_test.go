package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"sitewise/internal/store"
)

// A project has a site; only its org can read or edit it; a stale edit is
// 409 with the current version; new part kinds roof and plant_area.
func TestProjectSiteReadEditAndIsolation(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "12 Smith St fire upgrade")

	var site store.Site
	m.getJSON(t, "/api/projects/"+project+"/site", &site)
	if site.ID == "" || site.Label != "12 Smith St fire upgrade" || site.Version != 1 {
		t.Fatalf("site %+v", site)
	}
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodGet, "/api/projects/"+project+"/site", nil); got != http.StatusNotFound {
		t.Fatalf("foreign site read %d", got)
	}
	edit, _ := json.Marshal(map[string]any{"version": 1, "address": "12 Smith Street, Newtown NSW"})
	if got := other.status(t, http.MethodPatch, "/api/sites/"+site.ID, edit); got != http.StatusNotFound {
		t.Fatalf("foreign site edit %d", got)
	}
	var updated store.Site
	m.call(t, http.MethodPatch, "/api/sites/"+site.ID, edit, http.StatusOK, &updated)
	if updated.Address != "12 Smith Street, Newtown NSW" || updated.Version != 2 {
		t.Fatalf("updated %+v", updated)
	}
	var conflict struct {
		Error          string `json:"error"`
		CurrentVersion int64  `json:"current_version"`
	}
	m.call(t, http.MethodPatch, "/api/sites/"+site.ID, edit, http.StatusConflict, &conflict)
	if conflict.Error != "version_conflict" || conflict.CurrentVersion != 2 {
		t.Fatalf("conflict %+v", conflict)
	}
	blank, _ := json.Marshal(map[string]any{"version": 2, "label": "  "})
	if got := m.status(t, http.MethodPatch, "/api/sites/"+site.ID, blank); got != http.StatusUnprocessableEntity {
		t.Fatalf("blank label %d", got)
	}
	noVersion, _ := json.Marshal(map[string]any{"label": "x"})
	if got := m.status(t, http.MethodPatch, "/api/sites/"+site.ID, noVersion); got != http.StatusBadRequest {
		t.Fatalf("missing version %d", got)
	}

	var proj struct {
		SiteID string `json:"site_id"`
	}
	m.getJSON(t, "/api/projects/"+project, &proj)
	if proj.SiteID != site.ID {
		t.Fatalf("project site_id %q, want %q", proj.SiteID, site.ID)
	}
	dup, _ := json.Marshal(map[string]any{"label": "Level 1", "kind": "storey"})
	m.call(t, http.MethodPost, "/api/projects/"+project+"/parts", dup, http.StatusCreated, nil)
	if got := m.status(t, http.MethodPost, "/api/projects/"+project+"/parts", dup); got != http.StatusConflict {
		t.Fatalf("duplicate part label %d", got)
	}

	for _, kind := range []string{"roof", "plant_area"} {
		body, _ := json.Marshal(map[string]any{"label": "Part " + kind, "kind": kind})
		if got := m.status(t, http.MethodPost, "/api/projects/"+project+"/parts", body); got != http.StatusCreated {
			t.Fatalf("kind %s: %d", kind, got)
		}
	}
}
