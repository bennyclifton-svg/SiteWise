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
	var totals costs.Totals
	m.getJSON(t, base+"/totals", &totals)
	line := totals.Items[p.Items[0].ID]
	if line.Basis != "budget" || *line.Forecast.Amount != "9007199254740993.01" || *line.BudgetVariance != "0.00" || line.Remaining != nil || *totals.Buckets["project_wide"].BudgetVariance != "0.00" {
		t.Fatal("grid figures missing or inexact", totals)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, base + "/prepare"}, {http.MethodPost, base + "/suggestions/accept"}, {http.MethodPost, base + "/suggestions/pw:escalation/accept"}, {http.MethodPost, base + "/suggestions/pw:escalation/dismiss"}, {http.MethodDelete, base + "/suggestions/pw:escalation/dismiss"},
		{http.MethodGet, base + "/totals?by=system"}, {http.MethodGet, base + "/items/" + p.Items[0].ID + "/benchmarks"},
		{http.MethodPut, base}, {http.MethodPost, base + "/items"}, {http.MethodPatch, base + "/items/" + p.Items[0].ID}, {http.MethodPut, base + "/items/" + p.Items[0].ID + "/values/budget"},
		{http.MethodPost, base + "/items/" + p.Items[0].ID + "/subdivide"}, {http.MethodPost, base + "/baseline"}, {http.MethodPut, base + "/links"}, {http.MethodPost, base + "/items/" + p.Items[0].ID + "/benchmarks"}, {http.MethodDelete, base + "/items/" + p.Items[0].ID},
	} {
		foreign.call(t, route.method, route.path, []byte(`{}`), http.StatusNotFound, nil)
	}
	raw, _ := json.Marshal(map[string]any{"plan_version_id": p.ID, "version": p.Version})
	lineID := p.Items[0].ID
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
	m.call(t, http.MethodDelete, base+"/items/"+lineID, raw, http.StatusOK, &p)
	for _, row := range p.Items {
		if row.ID == lineID {
			t.Fatal("deleted line remains", row)
		}
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("costs called Jev")
	}
}

func TestCostSuggestionAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Autofill")
	base := "/api/projects/" + project + "/cost-plan"
	var p costs.Plan
	m.getJSON(t, base, &p)
	if len(p.Suggestions) != 7 {
		t.Fatal("missing initial suggestions", p)
	}
	m.call(t, http.MethodPost, base+"/suggestions/pw:escalation/accept", []byte(`{"version":0,"budget":"120.00"}`), 200, &p)
	if len(p.Items) != 1 || *p.Items[0].Values["budget"].Amount != "120.00" {
		t.Fatal(p)
	}
	body, _ := json.Marshal(map[string]any{"plan_version_id": p.ID, "version": p.Version})
	m.call(t, http.MethodPost, base+"/suggestions/accept", body, 200, &p)
	if len(p.Suggestions) != 0 || len(p.Items) != 7 {
		t.Fatal(p)
	}
	m.call(t, http.MethodPost, base+"/suggestions/accept", body, 409, nil)
	if a.jevHits.Load() != 0 {
		t.Fatal("autofill called Jev")
	}
}

func TestCostPrepareAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Automatic cost plan")
	base := "/api/projects/" + project + "/cost-plan"
	var result struct {
		Plan   costs.Plan   `json:"plan"`
		Totals costs.Totals `json:"totals"`
	}
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{}`), 200, &result)
	if len(result.Plan.Items) != 7 || len(result.Plan.Suggestions) != 0 || result.Totals.Overall["budget"].Unknown != 6 {
		t.Fatal("missing saved defaults or totals", result)
	}
	version := result.Plan.Version
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{}`), 200, &result)
	if result.Plan.Version != version {
		t.Fatal("repeat changed version")
	}
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{"budget":"123"}`), 400, nil)
	if a.jevHits.Load() != 0 {
		t.Fatal("preparation called Jev")
	}
}
