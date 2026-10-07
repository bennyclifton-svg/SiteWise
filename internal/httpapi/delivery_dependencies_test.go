package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/delivery"
	"testing"
)

func TestDeliveryDependenciesIsolationCyclesAndVersionsAPI(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	p := m.createProject(t, "Dependencies")
	base := "/api/projects/" + p + "/delivery"
	var first, second delivery.Item
	m.call(t, http.MethodPost, base, []byte(`{"kind":"activity","title":"First"}`), 201, &first)
	m.call(t, http.MethodPost, base, []byte(`{"kind":"activity","title":"Second"}`), 201, &second)
	put := func(from, to string, version int64, lag int) []byte {
		b, _ := json.Marshal(map[string]any{"predecessor_id": from, "successor_id": to, "version": version, "lag_days": lag})
		return b
	}
	m.call(t, http.MethodPost, base+"/dependencies", put(first.ID, second.ID, 1, 2), 200, nil)
	m.call(t, http.MethodPost, base+"/dependencies", put(first.ID, second.ID, 1, 3), 409, nil)
	m.call(t, http.MethodPost, base+"/dependencies", put(second.ID, first.ID, 1, 0), 422, nil)
	m.call(t, http.MethodPost, base+"/dependencies", put(first.ID, first.ID, 1, 0), 422, nil)
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodGet, base+"/dependencies", nil, 404, nil)
	foreign.call(t, http.MethodPost, base+"/dependencies", put(first.ID, second.ID, 2, 0), 404, nil)
	other := m.createProject(t, "Other")
	var outsider delivery.Item
	m.call(t, http.MethodPost, "/api/projects/"+other+"/delivery", []byte(`{"kind":"milestone","title":"Other"}`), 201, &outsider)
	m.call(t, http.MethodPost, base+"/dependencies", put(outsider.ID, second.ID, 2, 0), 404, nil)
	var edges struct {
		Items []delivery.Dependency `json:"items"`
	}
	m.getJSON(t, base+"/dependencies", &edges)
	if len(edges.Items) != 1 || edges.Items[0].LagDays != 2 {
		t.Fatal(edges)
	}
	m.call(t, http.MethodDelete, base+"/dependencies", put(first.ID, second.ID, 2, 0), 200, nil)
	m.getJSON(t, base+"/dependencies", &edges)
	if len(edges.Items) != 0 {
		t.Fatal(edges)
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("dependency called Jev")
	}
}
