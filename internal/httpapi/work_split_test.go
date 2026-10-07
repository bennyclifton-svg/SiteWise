package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/works"
	"testing"
)

func TestWorkTreeAPIIsolationVersionsAndBlockers(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Split work")
	base := "/api/projects/" + project + "/works"
	var profile struct {
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &profile)
	create, _ := json.Marshal(map[string]string{"part_id": profile.Parts[0].ID, "system_id": "hydraulic.gas", "action": "repair", "title": "Gas repairs"})
	var parent works.Item
	m.call(t, http.MethodPost, base, create, http.StatusCreated, &parent)
	split := base + "/" + parent.ID + "/split"
	body := []byte(`{"version":1,"children":[{"title":"Pipework"},{"title":"Valves"}]}`)
	other := a.member(t, newUUID(t))
	other.call(t, http.MethodPost, split, body, http.StatusNotFound, nil)
	m.call(t, http.MethodPost, split, []byte(`{"version":1,"children":[{"title":"Only"}]}`), http.StatusUnprocessableEntity, nil)
	var children []works.Item
	m.call(t, http.MethodPost, split, body, http.StatusOK, &children)
	if len(children) != 2 {
		t.Fatal("children missing")
	}
	m.call(t, http.MethodPost, split, body, http.StatusConflict, nil)
	var blocked struct {
		Error    string `json:"error"`
		Blockers []any  `json:"blockers"`
	}
	m.call(t, http.MethodPost, base+"/"+parent.ID+"/retire", []byte(`{"version":2}`), http.StatusConflict, &blocked)
	if blocked.Error != "referenced" || len(blocked.Blockers) != 2 {
		t.Fatalf("blockers %+v", blocked)
	}
	retire := base + "/" + children[0].ID + "/retire"
	other.call(t, http.MethodPost, retire, []byte(`{"version":1}`), http.StatusNotFound, nil)
	m.call(t, http.MethodPost, retire, []byte(`{"version":1}`), http.StatusOK, nil)
}
