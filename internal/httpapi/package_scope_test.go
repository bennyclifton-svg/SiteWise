package httpapi_test

import (
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
)

func TestPackageScopeAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Responsibilities")
	base := "/api/projects/" + project + "/packages"
	var p procurement.Package
	m.call(t, http.MethodPost, base, []byte(`{"kind":"services","title":"Design"}`), http.StatusCreated, &p)
	path := base + "/" + p.ID + "/scope"
	var item store.ScopeItem
	body := []byte(`{"item_kind":"obligation","clause_id":"cl.rfp-draft-status","clause_version":1,"inclusion":"included"}`)
	m.call(t, http.MethodPost, path, body, http.StatusCreated, &item)
	if !item.Provisional || item.Version != 1 {
		t.Fatalf("draft lost: %+v", item)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, path, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, path, body, http.StatusNotFound, nil)
	m.call(t, http.MethodPost, path, []byte(`{"item_kind":"obligation","user_text":"Meet","inclusion":"included","review_status":"verified"}`), http.StatusBadRequest, nil)
	m.call(t, http.MethodPost, path, []byte(`{"item_kind":"obligation","user_text":"Meet","inclusion":"included","interface_ids":["invented"]}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, path, []byte(`{"item_kind":"responsibility","work_item_id":"bad","role":"design","user_text":"Design","inclusion":"included"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPatch, path+"/"+item.ID, []byte(`{"version":1,"deliverable":"Fee response"}`), http.StatusOK, &item)
	m.call(t, http.MethodPatch, path+"/"+item.ID, []byte(`{"version":1,"deliverable":"Stale"}`), http.StatusConflict, nil)
	var view struct {
		Items []store.ScopeItem `json:"items"`
	}
	m.getJSON(t, path, &view)
	if len(view.Items) != 1 || !view.Items[0].Provisional || view.Items[0].Deliverable != "Fee response" {
		t.Fatal(view)
	}
	m.call(t, http.MethodPatch, path+"/"+item.ID, []byte(`{"version":2,"retired":true}`), http.StatusOK, nil)
	m.getJSON(t, path, &view)
	if len(view.Items) != 0 {
		t.Fatal("retired scope visible")
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("scope CRUD called Jev")
	}
}
