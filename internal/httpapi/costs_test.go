package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/costs"
	"testing"
)

func TestCostPlanAPIContractAndIsolation(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Cost plan")
	base := "/api/projects/" + project + "/cost-plan"
	var p costs.Plan
	m.getJSON(t, base, &p)
	if p.Status != "unavailable" {
		t.Fatal(p)
	}
	m.call(t, http.MethodPost, base+"/items", []byte(`{"version":0,"label":"Allowance","line_kind":"project_wide","category":"contingency","posting":true,"origin":"user","meaning":"allowance","values":{"budget":{"amount":"9007199254740993.01","value_state":"known","origin":"user","meaning":"allowance"}}}`), http.StatusOK, &p)
	if *p.Items[0].Values["budget"].Amount != "9007199254740993.01" {
		t.Fatal("precision lost", p)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, base + "/totals?by=system"}, {http.MethodGet, base + "/items/" + p.Items[0].ID + "/benchmarks"},
		{http.MethodPut, base}, {http.MethodPost, base + "/items"}, {http.MethodPatch, base + "/items/" + p.Items[0].ID}, {http.MethodPut, base + "/items/" + p.Items[0].ID + "/values/budget"},
		{http.MethodPost, base + "/items/" + p.Items[0].ID + "/subdivide"}, {http.MethodPost, base + "/baseline"}, {http.MethodPut, base + "/links"}, {http.MethodPost, base + "/items/" + p.Items[0].ID + "/benchmarks"}, {http.MethodDelete, base + "/items/" + p.Items[0].ID},
	} {
		foreign.call(t, route.method, route.path, []byte(`{}`), http.StatusNotFound, nil)
	}
	raw, _ := json.Marshal(map[string]any{"plan_version_id": p.ID, "version": p.Version})
	old := p.ID
	m.call(t, http.MethodPost, base+"/baseline", raw, http.StatusOK, &p)
	m.call(t, http.MethodPost, base+"/baseline", raw, http.StatusConflict, nil)
	m.call(t, http.MethodGet, base+"?version=not-a-uuid", nil, http.StatusUnprocessableEntity, nil)
	var frozen costs.Plan
	m.getJSON(t, base+"?version="+old, &frozen)
	if frozen.Status != "baseline" {
		t.Fatal(frozen)
	}
	raw, _ = json.Marshal(map[string]any{"plan_version_id": p.ID, "version": p.Version})
	m.call(t, http.MethodDelete, base+"/items/"+p.Items[0].ID, raw, http.StatusOK, &p)
	if len(p.Items) != 0 {
		t.Fatal(p)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("costs called Jev")
	}
}
