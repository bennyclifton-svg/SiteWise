package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
	"testing"
)

func TestWorksAPIValidationIsolationAndPicker(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Works")
	base := "/api/projects/" + project + "/works"
	var p profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	// Part identity comes from the existing profile response.
	var raw map[string]json.RawMessage
	m.getJSON(t, "/api/projects/"+project+"/profile", &raw)
	var parts []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw["parts"], &parts); err != nil || len(parts) == 0 {
		t.Fatalf("parts %s %v", raw["parts"], err)
	}
	body := map[string]any{"part_id": parts[0].ID, "system_id": "structure", "action": "investigate", "quantity": 12.5, "unit": "m2", "existing_condition": "serviceable", "existing_condition_note": "Confirm at inspection", "target": map[string]any{"text": "Inspect existing structure", "values": []any{map[string]any{"key": "plan.gross_floor_area", "value": 1200, "unit": "m2"}}}}
	encoded, _ := json.Marshal(body)
	var item works.Item
	m.call(t, http.MethodPost, base, encoded, http.StatusCreated, &item)
	if item.ID == "" || !item.UserTouched || item.Origin != "user" || item.Quantity == nil || *item.Quantity != "12.5" {
		t.Fatalf("item %+v", item)
	}
	gapURL := "/api/projects/" + project + "/gaps"
	var gaps struct {
		Items []procurement.Gap `json:"items"`
	}
	m.getJSON(t, gapURL, &gaps)
	if len(gaps.Items) != 1 || gaps.Items[0].Role != "inspect_or_test" || gaps.Items[0].WorkItemID != item.ID {
		t.Fatalf("missing investigation gap %+v", gaps)
	}
	var pkg procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Survey"}`), http.StatusCreated, &pkg)
	assignment, _ := json.Marshal(map[string]string{"item_kind": "responsibility", "work_item_id": item.ID, "role": "inspect", "user_text": "Inspect structure", "inclusion": "included"})
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages/"+pkg.ID+"/scope", assignment, http.StatusCreated, nil)
	m.getJSON(t, gapURL, &gaps)
	if len(gaps.Items) != 0 {
		t.Fatalf("assigned gap persists %+v", gaps)
	}
	m.call(t, http.MethodPost, base, encoded, http.StatusConflict, nil)
	var view struct {
		Items []works.Item `json:"items"`
	}
	m.getJSON(t, base, &view)
	if len(view.Items) != 1 || view.Items[0].ID != item.ID {
		t.Fatalf("read %+v", view)
	}
	if view.Items[0].ExistingCondition != "serviceable" || view.Items[0].ExistingConditionNote != "Confirm at inspection" {
		t.Fatalf("condition read-through %+v", view.Items[0])
	}
	put(t, m, project, "sys.structure.condition", map[string]any{"value": "failed", "version": 1}, http.StatusOK, &p)
	m.getJSON(t, base, &view)
	if view.Items[0].ExistingCondition != "failed" {
		t.Fatal("work item copied site condition instead of reading through")
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, gapURL, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base, encoded, http.StatusNotFound, nil)
	editPath := base + "/" + item.ID
	foreign.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"title":"Foreign"}`), http.StatusNotFound, nil)
	for _, bad := range []string{`{"version":1}`, `{"version":1,"title":" "}`, `{"version":1,"action":"invented"}`, `{"version":1,"quantity":true}`, `{"version":1,"quantity":-1}`, `{"version":1,"quantity":null,"unit":"m"}`, `{"version":1,"target":{"clause_refs":[{"id":"invented","version":1}]}}`} {
		m.call(t, http.MethodPatch, editPath, []byte(bad), http.StatusUnprocessableEntity, nil)
	}
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"origin":"document","title":"Fake provenance"}`), http.StatusBadRequest, nil)
	item = works.Item{}
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"action":"repair","title":"Repair structure","quantity":null}`), http.StatusOK, &item)
	if item.Version != 2 || item.Action != "repair" || item.Quantity != nil || item.Unit != nil || item.Provenance.LastEditedBy == "" {
		t.Fatalf("patch %+v", item)
	}
	m.call(t, http.MethodPatch, editPath, []byte(`{"version":1,"title":"Stale"}`), http.StatusConflict, nil)
	for _, bad := range []map[string]any{
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "existing_condition": "invented"},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "target": map[string]any{"clause_refs": []any{map[string]any{"id": "invented", "version": 1}}}},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "target": map[string]any{"values": []any{map[string]any{"key": "plan.gross_floor_area", "value": "large"}}}},
		{"part_id": parts[0].ID, "system_id": "invented", "action": "new"},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "invented"},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "quantity": -1, "unit": "m"},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "quantity": 1},
		{"part_id": parts[0].ID, "system_id": "hydraulic", "action": "new", "target": map[string]any{"values": []any{map[string]any{"key": "x", "value": map[string]any{"unsafe": "shape"}}}}},
	} {
		encoded, _ := json.Marshal(bad)
		m.call(t, http.MethodPost, base, encoded, http.StatusUnprocessableEntity, nil)
	}
	otherProject := m.createProject(t, "Other")
	m.call(t, http.MethodPatch, "/api/projects/"+otherProject+"/works/"+item.ID, []byte(`{"version":2,"title":"Wrong project"}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPost, "/api/projects/"+otherProject+"/works", encoded, http.StatusNotFound, nil)
}
