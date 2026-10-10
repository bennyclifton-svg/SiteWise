package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func TestConsultantRFPSettingsAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFP project")
	var p procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect","discipline_id":"consultant.architect"}`), http.StatusCreated, &p)
	var civil procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Civil"}`), http.StatusCreated, &civil)
	body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": p.ID})
	var r store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", body, http.StatusCreated, &r)
	base := "/api/reports/" + r.ID
	var d store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	if d.TemplateID != "tpl.rfp-consultant" {
		t.Fatalf("template %s", d.TemplateID)
	}
	var view store.ReportView
	m.getJSON(t, base, &view)
	if view.RFP == nil || view.RFP.Discipline != "architecture" || len(view.RFP.Groups) == 0 {
		t.Fatal("controls missing from the report view")
	}
	put := func(key string, value any, version int64, want int, out any) {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"value": value, "version": version})
		m.call(t, http.MethodPut, base+"/settings/"+key, raw, want, out)
	}
	put("position:bim_coordination", "X", d.Version, http.StatusOK, &view)
	if view.Draft == nil || view.RFP == nil || view.Draft.Version != d.Version+1 {
		t.Fatal("a setting returns the re-resolved view")
	}
	put("position:bim_coordination", "I", d.Version, http.StatusConflict, nil)
	put("position:bim_coordination", "Y", view.Draft.Version, http.StatusBadRequest, nil)
	put("position:no_such_item", "I", view.Draft.Version, http.StatusBadRequest, nil)
	put("Bad Key", "I", view.Draft.Version, http.StatusBadRequest, nil)
	raw, _ := json.Marshal(map[string]any{"version": view.Draft.Version})
	m.call(t, http.MethodPut, base+"/settings/scale", raw, http.StatusBadRequest, nil)
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPut, base+"/settings/scale", []byte(`{"value":"simple","version":1}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodDelete, base+"/settings/position:bim_coordination", []byte(`{"version":1}`), http.StatusNotFound, nil)
	m.call(t, http.MethodDelete, base+"/settings/position:bim_coordination", raw, http.StatusOK, &view)
	m.call(t, http.MethodDelete, base+"/settings/position:bim_coordination", []byte(`{"version":`+jsonNumber(view.Draft.Version)+`}`), http.StatusNotFound, nil)

	var disciplines struct {
		Items []struct {
			ID           string   `json:"id"`
			TitleAliases []string `json:"title_aliases"`
		} `json:"items"`
	}
	m.getJSON(t, "/api/rfp/disciplines", &disciplines)
	available := map[string]bool{}
	for _, item := range disciplines.Items {
		if available[item.ID] || len(item.TitleAliases) == 0 {
			t.Fatalf("duplicate or unidentifiable RFP discipline: %+v", item)
		}
		available[item.ID] = true
	}
	for _, id := range []string{"architecture", "civil", "structural", "town-planning", "traffic",
		"bca-certification", "geotechnical", "land-surveying", "cost-planning", "access", "waste-management",
		"arborist", "fire-engineering", "landscape-architecture", "fire-services", "mechanical-services",
		"electrical-services", "hydraulic-services"} {
		if !available[id] {
			t.Errorf("golden RFP %s missing from the API catalogue", id)
		}
	}
	body, _ = json.Marshal(map[string]string{"kind": "rfp", "package_id": civil.ID})
	var generic store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", body, http.StatusCreated, &generic)
	var genericView store.ReportView
	m.getJSON(t, "/api/reports/"+generic.ID, &genericView)
	if genericView.RFPSuggestion == nil || genericView.RFPSuggestion.Discipline != "civil" || genericView.RFPSuggestion.DisciplineID != "consultant.civil" {
		t.Fatalf("a Civil package with no discipline gets a suggestion: %+v", genericView.RFPSuggestion)
	}
	var gd store.ReportDraft
	m.call(t, http.MethodPost, "/api/reports/"+generic.ID+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &gd)
	if gd.TemplateID != "tpl.rfp-capex" {
		t.Fatal("a title alone never switches to the discipline RFP")
	}
	raw, _ = json.Marshal(map[string]any{"value": "complex", "version": gd.Version})
	m.call(t, http.MethodPut, "/api/reports/"+generic.ID+"/settings/scale", raw, http.StatusUnprocessableEntity, nil)
	if a.jevHits.Load() != 0 {
		t.Fatal("RFP settings called Jev")
	}
}

func jsonNumber(n int64) string {
	raw, _ := json.Marshal(n)
	return string(raw)
}

// A project fact edited in an RFP saves to Profile (owner decision D1).
func TestRFPFactAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFP facts")
	var p procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Structural","discipline_id":"consultant.structural"}`), http.StatusCreated, &p)
	body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": p.ID})
	var r store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", body, http.StatusCreated, &r)
	base := "/api/reports/" + r.ID
	var d store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	put := func(fact string, value any, version int64, want int, out any) {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"value": value, "version": version})
		m.call(t, http.MethodPut, base+"/facts/"+fact, raw, want, out)
	}
	var view store.ReportView
	put("zone", "R2", d.Version, http.StatusOK, &view)
	if view.Draft == nil || view.RFP == nil || view.Draft.Version <= d.Version || len(view.Stale) != 0 {
		t.Fatalf("a fact returns the refreshed, current view: %+v", view.Stale)
	}
	var profile detailsBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &profile)
	if v, _ := detailValue(profile, "proj.zone"); v != "R2" {
		t.Fatalf("Profile zone = %q", v)
	}
	put("zone", "R3", d.Version, http.StatusConflict, nil)
	put("budget", "100", view.Draft.Version, http.StatusBadRequest, nil)
	put("flag.basement", "maybe", view.Draft.Version, http.StatusBadRequest, nil)
	put("Bad Fact", "x", view.Draft.Version, http.StatusBadRequest, nil)
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPut, base+"/facts/zone", []byte(`{"value":"B4","version":1}`), http.StatusNotFound, nil)
	put("zone", nil, view.Draft.Version, http.StatusOK, &view)
	m.getJSON(t, "/api/projects/"+project+"/profile", &profile)
	if v, _ := detailValue(profile, "proj.zone"); v != "" {
		t.Fatalf("a null value clears the user's word: %q", v)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("saving a fact never calls Jev")
	}
}

// report_fact budget (bench/budgets.json): p50 150 ms, p90 400 ms.
func TestRFPFactLatencyBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "RFP fact timing")
	var p procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Structural","discipline_id":"consultant.structural"}`), http.StatusCreated, &p)
	body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": p.ID})
	var r store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", body, http.StatusCreated, &r)
	var d store.ReportDraft
	m.call(t, http.MethodPost, "/api/reports/"+r.ID+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	version := d.Version
	var durations []time.Duration
	for i := 0; i < 25; i++ {
		raw, _ := json.Marshal(map[string]any{"value": fmt.Sprintf("Zone %d", i), "version": version})
		var view store.ReportView
		start := time.Now()
		m.call(t, http.MethodPut, "/api/reports/"+r.ID+"/facts/zone", raw, http.StatusOK, &view)
		durations = append(durations, time.Since(start))
		version = view.Draft.Version
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	t.Logf("report_fact p50=%s p90=%s (budget 150ms/400ms)", durations[12], durations[22])
	if durations[12] > 150*time.Millisecond || durations[22] > 400*time.Millisecond {
		t.Fatal("report_fact exceeded budget")
	}
}
