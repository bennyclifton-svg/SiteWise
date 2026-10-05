package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type planningBody struct {
	Values []struct {
		Key          string   `json:"key"`
		Label        string   `json:"label"`
		Scope        string   `json:"scope"`
		ValueState   string   `json:"value_state"`
		Value        *string  `json:"value"`
		RangeLow     *float64 `json:"range_low"`
		Unit         string   `json:"unit"`
		Origin       string   `json:"origin"`
		ReviewStatus string   `json:"review_status"`
		Version      int64    `json:"version"`
	} `json:"values"`
	Unresolved []struct {
		Key string `json:"key"`
	} `json:"unresolved"`
	History []struct {
		Version int64 `json:"version"`
	} `json:"history"`
	Suggestions []any `json:"suggestions"`
}

func putPlanning(t *testing.T, m *member, project, key string, body map[string]any, want int, out any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	m.call(t, http.MethodPut, "/api/projects/"+project+"/planning/"+key, raw, want, out)
}

// WP-13, AT-07, AT-09: record an assumption with a range and limitations,
// an explicit unknown listed as unresolved, a stale edit refused, and a
// withdrawal kept as history. Favourable defaults and money keys are refused.
func TestPlanningValuesAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Warehouse")
	base := "/api/projects/" + project + "/planning"

	var p planningBody
	m.getJSON(t, base, &p)
	if len(p.Values) != 0 || p.Suggestions == nil || len(p.Suggestions) != 0 {
		t.Fatalf("empty planning %+v", p)
	}
	putPlanning(t, m, project, "gross_floor_area", map[string]any{"value": 1200, "range_low": 1100, "range_high": 1300,
		"unit": "m2", "rationale": "Brief area schedule", "limitations": "Excludes plant deck", "version": 0}, http.StatusOK, &p)
	if len(p.Values) != 1 || *p.Values[0].Value != "1200" || p.Values[0].Origin != "assumption" || p.Values[0].Scope != "site" ||
		p.Values[0].ReviewStatus != "accepted_for_planning" || p.Values[0].Version != 1 || *p.Values[0].RangeLow != 1100 {
		t.Fatalf("assumption %+v", p.Values)
	}
	putPlanning(t, m, project, "existing_structure_adequate", map[string]any{"value": nil, "version": 0}, http.StatusOK, &p)
	if len(p.Unresolved) != 1 || p.Unresolved[0].Key != "existing_structure_adequate" {
		t.Fatalf("unknown must be listed as unresolved: %+v", p.Unresolved)
	}

	var conflict struct {
		Error          string `json:"error"`
		CurrentVersion int64  `json:"current_version"`
	}
	putPlanning(t, m, project, "gross_floor_area", map[string]any{"value": 1250, "version": 0}, http.StatusConflict, &conflict)
	if conflict.Error != "version_conflict" || conflict.CurrentVersion != 1 {
		t.Fatalf("conflict %+v", conflict)
	}

	for name, c := range map[string]struct {
		key  string
		body map[string]any
	}{
		"unregistered":         {"roof_colour", map[string]any{"value": "red"}},
		"money total":          {"cost_total", map[string]any{"value": 1000000}},
		"favourable structure": {"existing_structure_adequate", map[string]any{"value": true, "version": 1}},
		"favourable ground":    {"site_classification", map[string]any{"value": "A"}},
		"favourable by user":   {"existing_compliance", map[string]any{"value": "complies", "origin": "user"}},
		"calculation sent":     {"site_area", map[string]any{"value": 500, "origin": "calculation"}},
		"not an option":        {"site_classification", map[string]any{"value": "Z"}},
		"wrong unit":           {"gross_floor_area", map[string]any{"value": 10, "unit": "ft2", "version": 1}},
		"outside range":        {"site_area", map[string]any{"value": 10, "range_low": 20}},
		"range on text":        {"target_completion", map[string]any{"value": "Q3", "range_low": 1}},
		"set without value":    {"site_area", map[string]any{"value_state": "set"}},
		"unknown with value":   {"site_area", map[string]any{"value": 5, "value_state": "unknown"}},
	} {
		raw, _ := json.Marshal(c.body)
		if got := m.status(t, http.MethodPut, base+"/"+c.key, raw); got != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d, want 422", name, got)
		}
	}
	// A poor site classification may be assumed; only A and S may not.
	putPlanning(t, m, project, "site_classification", map[string]any{"value": "M", "version": 0}, http.StatusOK, nil)

	// Withdraw: superseded, not deleted.
	if got := m.status(t, http.MethodDelete, base+"/gross_floor_area?version=0", nil); got != http.StatusUnprocessableEntity {
		t.Fatalf("withdraw without a version: %d", got)
	}
	m.call(t, http.MethodDelete, base+"/gross_floor_area?version=1", nil, http.StatusOK, &p)
	for _, v := range p.Values {
		if v.Key == "gross_floor_area" {
			t.Fatal("withdrawn value still live")
		}
	}
	m.getJSON(t, base+"?history=1", &p)
	if len(p.History) != 1 || p.History[0].Version != 1 {
		t.Fatalf("history %+v", p.History)
	}
}

// The user's stated word on a planning key outranks the assumption, and a
// favourable value cannot be recorded on the profile as an assumption either.
func TestPlanningKeyStatedOnTheProfile(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Fit-out")
	putPlanning(t, m, project, "construction_duration", map[string]any{"value": 40, "meaning": "forecast", "version": 0}, http.StatusOK, nil)
	put(t, m, project, "plan.construction_duration", map[string]any{"value": "52"}, http.StatusOK, nil)
	put(t, m, project, "plan.construction_duration", map[string]any{"value": "soon"}, http.StatusUnprocessableEntity, nil)
	put(t, m, project, "plan.existing_structure_adequate", map[string]any{"value": "true", "origin": "assumption"}, http.StatusUnprocessableEntity, nil)
	// An edit that omits origin keeps a stored assumption, so it cannot
	// turn that assumption favourable either (review finding, L276).
	put(t, m, project, "plan.existing_structure_adequate", map[string]any{"value": "false", "origin": "assumption", "version": 0}, http.StatusOK, nil)
	put(t, m, project, "plan.existing_structure_adequate", map[string]any{"value": "true", "version": 1}, http.StatusUnprocessableEntity, nil)
	put(t, m, project, "plan.existing_structure_adequate", map[string]any{"value": "true", "origin": "user", "version": 1}, http.StatusOK, nil)
}

// AT-22: another org cannot read or write a project's planning values.
func TestPlanningIsOrgScoped(t *testing.T) {
	a := newApp(t, withProfile(t))
	owner := a.member(t, newUUID(t))
	project := owner.createProject(t, "Private")
	stranger := a.member(t, newUUID(t))
	base := "/api/projects/" + project + "/planning"
	if got := stranger.status(t, http.MethodGet, base, nil); got != http.StatusNotFound {
		t.Fatalf("GET %d", got)
	}
	raw, _ := json.Marshal(map[string]any{"value": 100, "version": 0})
	if got := stranger.status(t, http.MethodPut, base+"/site_area", raw); got != http.StatusNotFound {
		t.Fatalf("PUT %d", got)
	}
	if got := stranger.status(t, http.MethodDelete, base+"/site_area?version=1", nil); got != http.StatusNotFound {
		t.Fatalf("DELETE %d", got)
	}
	// A part of another project's site is not found, by the API as well.
	other := owner.createProject(t, "Other building")
	var p struct {
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	owner.getJSON(t, "/api/projects/"+other+"/profile", &p)
	raw, _ = json.Marshal(map[string]any{"value": 100, "version": 0, "part_id": p.Parts[0].ID})
	if got := owner.status(t, http.MethodPut, base+"/site_area", raw); got != http.StatusNotFound {
		t.Fatalf("PUT with another site's part: %d", got)
	}
	// Large numbers keep every digit.
	var body planningBody
	putPlanning(t, owner, project, "site_area", map[string]any{"value": json.Number("9007199254740993"), "version": 0}, http.StatusOK, &body)
	if *body.Values[0].Value != "9007199254740993" {
		t.Fatalf("number rounded: %s", *body.Values[0].Value)
	}
}
