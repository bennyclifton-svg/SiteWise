package httpapi_test

import (
	"net/http"
	"testing"
)

type headerCell struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	Band         string `json:"band"`
	Scope        string `json:"scope"`
	Origin       string `json:"origin"`
	ReviewStatus string `json:"review_status"`
	Version      int64  `json:"version"`
}

// WP-12: an edit from a stale version is 409 with the current version; the
// read model says where a value came from and which version to edit from;
// provenance is validated at the boundary.
func TestProfileEditVersionAndProvenance(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Fire upgrade")

	var p struct {
		Header []headerCell `json:"header"`
	}
	put(t, m, project, "hdr.work_type", map[string]any{"value": "refurb", "version": 0, "origin": "assumption"}, http.StatusOK, &p)
	var cell headerCell
	for _, h := range p.Header {
		if h.Key == "hdr.work_type" {
			cell = h
		}
	}
	if cell.Value != "refurb" || cell.Version != 1 || cell.Origin != "assumption" || cell.Scope != "project" || cell.ReviewStatus != "accepted_for_planning" {
		t.Fatalf("cell %+v", cell)
	}
	var conflict struct {
		Error          string `json:"error"`
		CurrentVersion int64  `json:"current_version"`
	}
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new", "version": 0}, http.StatusConflict, &conflict)
	if conflict.Error != "version_conflict" || conflict.CurrentVersion != 1 {
		t.Fatalf("conflict %+v", conflict)
	}
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new", "version": 1}, http.StatusOK, nil)
	// Older clients that send no version still work.
	put(t, m, project, "hdr.work_type", map[string]any{"value": "refurb"}, http.StatusOK, nil)

	// A stale reset reports the real current version (3 after the edits above).
	conflict.CurrentVersion = 0
	put(t, m, project, "hdr.work_type", map[string]any{"reset": true, "version": 1}, http.StatusConflict, &conflict)
	if conflict.CurrentVersion != 3 {
		t.Fatalf("stale reset reported version %d", conflict.CurrentVersion)
	}
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new", "origin": "guess"}, http.StatusUnprocessableEntity, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new", "meaning": "hope"}, http.StatusUnprocessableEntity, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new", "unknown": true}, http.StatusUnprocessableEntity, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": nil, "unknown": true}, http.StatusOK, nil)
}
