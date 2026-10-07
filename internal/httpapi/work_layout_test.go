package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/works"
	"testing"
)

func TestWorkLayoutChangeAPIBoundary(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Layout")
	base := "/api/projects/" + project + "/works"
	var profile struct {
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &profile)
	create, _ := json.Marshal(map[string]string{"part_id": profile.Parts[0].ID, "system_id": "interiors.walls-linings", "action": "alter", "title": "Partition layout"})
	var item works.Item
	m.call(t, http.MethodPost, base, create, http.StatusCreated, &item)
	if item.LayoutChange != "unknown" {
		t.Fatal(item)
	}
	path := base + "/" + item.ID
	other := a.member(t, newUUID(t))
	other.call(t, http.MethodPatch, path, []byte(`{"version":1,"layout_change":"yes"}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"layout_change":true}`), http.StatusBadRequest, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"layout_change":"sometimes"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"layout_change":"yes"}`), http.StatusOK, &item)
	if item.LayoutChange != "yes" {
		t.Fatal(item)
	}
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"layout_change":"no"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":2,"action":"repair"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":2,"action":"repair","layout_change":"unknown"}`), http.StatusOK, &item)
	if item.LayoutChange != "unknown" || item.Action != "repair" {
		t.Fatal(item)
	}
}

// POST has an explicit request DTO, unlike PATCH's works.Patch decoder. Cover
// authored create input independently so omission in that DTO cannot be hidden
// by tests that create unknown and only author the answer in a later PATCH.
func TestWorkLayoutChangeCreateAPIBoundary(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Authored layout")
	base := "/api/projects/" + project + "/works"
	var profile struct {
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &profile)
	body := map[string]any{"part_id": profile.Parts[0].ID, "system_id": "interiors.walls-linings", "action": "alter", "title": "Partition layout", "layout_change": "yes"}
	encode := func() []byte {
		raw, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	other := a.member(t, newUUID(t))
	other.call(t, http.MethodPost, base, encode(), http.StatusNotFound, nil)
	body["layout_change"] = true
	m.call(t, http.MethodPost, base, encode(), http.StatusBadRequest, nil)
	body["layout_change"] = "sometimes"
	m.call(t, http.MethodPost, base, encode(), http.StatusUnprocessableEntity, nil)
	body["layout_change"] = "yes"
	body["action"] = "repair"
	m.call(t, http.MethodPost, base, encode(), http.StatusUnprocessableEntity, nil)
	body["action"] = "alter"
	body["system_id"] = "hydraulic.gas"
	m.call(t, http.MethodPost, base, encode(), http.StatusUnprocessableEntity, nil)
	body["system_id"] = "interiors.walls-linings"
	var item works.Item
	m.call(t, http.MethodPost, base, encode(), http.StatusCreated, &item)
	if item.LayoutChange != "yes" {
		t.Fatal("POST dropped authored layout", item)
	}
	var saved struct {
		Items []works.Item `json:"items"`
	}
	m.getJSON(t, base, &saved)
	if len(saved.Items) != 1 || saved.Items[0].LayoutChange != "yes" {
		t.Fatal("POST layout not persisted", saved)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("authored layout called Jev")
	}
}
