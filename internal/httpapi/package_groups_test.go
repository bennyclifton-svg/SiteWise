package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"sitewise/internal/procurement"
)

func TestCombineUncombineAndRemoveRoutes(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Combined engagements")
	base := "/api/projects/" + project
	jevBefore := a.jevHits.Load()
	rows := []procurement.Package{}
	for _, d := range []string{"consultant.structural", "consultant.civil", "consultant.hydraulic"} {
		var p procurement.Package
		m.call(t, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"`+d+`"}`), http.StatusCreated, &p)
		rows = append(rows, p)
	}
	var trade procurement.Package
	m.call(t, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"trade.electrical"}`), http.StatusCreated, &trade)
	combine := base + "/packages/combine"

	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID}}), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, trade.ID}}), http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, "nope"}}), http.StatusBadRequest, nil)
	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, newUUID(t)}}), http.StatusNotFound, nil)

	var group procurement.Package
	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, rows[1].ID, rows[2].ID}, "title": " Structural, civil and hydraulic "}), http.StatusCreated, &group)
	if !group.IsGroup() || group.Title != "Structural, civil and hydraulic" || group.DisciplineID != "" {
		t.Fatalf("group %+v", group)
	}
	// The same request again finds the members already grouped.
	m.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, rows[1].ID}}), http.StatusConflict, nil)

	// Another org sees none of it.
	foreign := a.member(t, newUUID(t))
	foreign.call(t, http.MethodPost, combine, groupBody(t, map[string]any{"package_ids": []string{rows[0].ID, rows[1].ID}}), http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/packages/"+group.ID+"/uncombine", nil, http.StatusNotFound, nil)
	foreign.call(t, http.MethodPost, base+"/packages/"+rows[0].ID+"/ungroup", nil, http.StatusNotFound, nil)

	// Rename through the title patch.
	var renamed procurement.Package
	m.call(t, http.MethodPatch, base+"/packages/"+group.ID, groupBody(t, map[string]any{"version": group.Version, "title": "Civil and services"}), http.StatusOK, &renamed)
	if renamed.Title != "Civil and services" {
		t.Fatalf("rename %+v", renamed)
	}
	var overview struct {
		Items []procurement.Package `json:"items"`
	}
	m.getJSON(t, base+"/packages", &overview)
	members := 0
	for _, p := range overview.Items {
		if p.GroupID == group.ID {
			members++
		}
	}
	if members != 3 {
		t.Fatalf("members listed %d", members)
	}

	m.call(t, http.MethodPost, base+"/packages/"+rows[0].ID+"/uncombine", nil, http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, base+"/packages/"+rows[0].ID+"/ungroup", nil, http.StatusNoContent, nil)
	m.call(t, http.MethodPost, base+"/packages/"+rows[0].ID+"/ungroup", nil, http.StatusUnprocessableEntity, nil)
	m.call(t, http.MethodPost, base+"/packages/"+group.ID+"/uncombine", nil, http.StatusNoContent, nil)
	m.call(t, http.MethodPost, base+"/packages/"+group.ID+"/uncombine", nil, http.StatusNotFound, nil)
	var after struct {
		Items []procurement.Package `json:"items"`
	}
	m.getJSON(t, base+"/packages", &after)
	for _, p := range after.Items {
		if p.IsGroup() || p.GroupID != "" {
			t.Fatalf("still grouped: %+v", p)
		}
	}
	if a.jevHits.Load() != jevBefore {
		t.Fatal("combining called Jev")
	}
}

func groupBody(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
