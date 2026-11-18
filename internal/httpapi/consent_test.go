package httpapi_test

import (
	"net/http"
	"testing"

	"sitewise/internal/delivery"
)

// A consent condition added by hand through POST /delivery
// (docs/plans/2026-11-17-consent-conditions.md, Task 1): its number and
// words are the instrument's, an edit cannot renumber it, and another org
// gets 404.
func TestConditionAddedByHandKeepsItsNumber(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Conditions by hand")
	base := "/api/projects/" + project + "/delivery"
	var item delivery.Item
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Condition 46 · Landscape plan","details":{"consent_ref":"DA/2025/0733","number":"46","heading":"Landscape plan","text":"A landscape plan is to be submitted.","stage":"before_cc"}}`), http.StatusCreated, &item)
	if item.Status != "open" || delivery.Condition(item.Content).Number != "46" || item.Number != nil {
		t.Fatalf("created %+v", item)
	}
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"No number","details":{"text":"x"}}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Bad status","status":"answered","details":{"number":"2"}}`), http.StatusUnprocessableEntity, nil)
	path := base + "/" + item.ID
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"details":{"number":"47","text":"Reworded","stage":"before_occupation","discipline":"consultant.landscape"}}`), http.StatusOK, &item)
	d := delivery.Condition(item.Content)
	if d.Number != "46" || d.Text != "A landscape plan is to be submitted." || d.Stage != "before_occupation" || d.Discipline != "consultant.landscape" {
		t.Fatalf("edit renumbered or reworded: %+v", d)
	}
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPost, base, []byte(`{"kind":"condition","title":"Condition 1","details":{"number":"1"}}`), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPatch, path, []byte(`{"version":2,"status":"satisfied"}`), http.StatusNotFound, nil)
	other := m.createProject(t, "Another project")
	m.call(t, http.MethodPatch, "/api/projects/"+other+"/delivery/"+item.ID, []byte(`{"version":2,"status":"satisfied"}`), http.StatusNotFound, nil)
}
