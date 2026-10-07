package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/delivery"
	"testing"
)

func TestDeliveryManualStatusDatesAndIsolationAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Delivery")
	base := "/api/projects/" + project + "/delivery"
	var item delivery.Item
	m.call(t, http.MethodPost, base, []byte(`{"kind":"approval","title":"Authority review","target_date":"2026-12-01","details":{"authority":"Authority"}}`), http.StatusCreated, &item)
	if item.Status != "not_submitted" || item.TargetDate == nil || item.ReviewStatus != "accepted_for_planning" {
		t.Fatal(item)
	}
	path := base + "/" + item.ID
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, base, nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPatch, path, []byte(`{"version":1,"status":"approved"}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPost, base, []byte(`{"kind":"approval","title":"Spoof","origin":"document"}`), http.StatusBadRequest, nil)
	m.call(t, http.MethodPost, base, []byte(`{"kind":"risk","title":"Risk","status":"approved"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"target_date":"tomorrow"}`), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"status":"submitted","as_of":"2026-10-05"}`), http.StatusOK, &item)
	if item.TargetDate == nil || item.Version != 2 {
		t.Fatal("omitted date changed", item)
	}
	m.call(t, http.MethodPatch, path, []byte(`{"version":1,"status":"approved"}`), http.StatusConflict, nil)
	m.call(t, http.MethodPatch, path, []byte(`{"version":2,"status":"approved","target_date":null,"details":{"authority":"Authority","submitted_on":"2026-10-01","determined_on":"2026-10-05"}}`), http.StatusOK, &item)
	if item.TargetDate != nil || item.Status != "approved" || item.ReviewStatus == "verified" {
		t.Fatal("approval/date clear", item)
	}
	var provenance struct {
		StatusHistory []struct {
			Actor  string `json:"actor"`
			Status string `json:"status"`
			At     string `json:"at"`
		} `json:"status_history"`
	}
	if err := json.Unmarshal(item.Provenance, &provenance); err != nil || len(provenance.StatusHistory) != 2 || provenance.StatusHistory[1].Actor == "" || provenance.StatusHistory[1].Status != "approved" || provenance.StatusHistory[1].At == "" {
		t.Fatalf("audit %+v %v", provenance, err)
	}
	var view struct {
		Items []delivery.Item `json:"items"`
	}
	m.getJSON(t, base, &view)
	if len(view.Items) != 1 {
		t.Fatal(view)
	}
	m.call(t, http.MethodPatch, path, []byte(`{"version":3,"retired":true}`), http.StatusOK, &item)
	if item.RetiredAt == nil {
		t.Fatal("not retired")
	}
	m.getJSON(t, base, &view)
	if len(view.Items) != 0 {
		t.Fatal("retired item visible")
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("delivery called Jev")
	}
}
