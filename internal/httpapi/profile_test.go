package httpapi_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"sitewise/internal/httpapi"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

type profileBody struct {
	ProjectID  string         `json:"project_id"`
	Thresholds map[string]any `json:"thresholds"`
	Parts      []struct {
		ID, Label, Kind string
	} `json:"parts"`
	Header []struct {
		Key, Value, Band string
		Options          []struct{ ID, Label string }
	} `json:"header"`
	Systems []struct {
		ID   string `json:"id"`
		Rows []struct {
			Leaf     string `json:"leaf"`
			Shown    bool   `json:"shown_by_default"`
			Presence struct {
				Value, Band string
			} `json:"presence"`
		} `json:"rows"`
	} `json:"systems"`
	Compliance []struct {
		Group string `json:"group"`
		Rows  []struct {
			Key      string   `json:"key"`
			StatedIn []string `json:"stated_in"`
			Derived  *struct {
				Reason string `json:"reason"`
			} `json:"derived"`
		} `json:"rows"`
	} `json:"compliance"`
}

func withProfile(t *testing.T) func(*httpapi.Options) {
	cat, err := knowledge.Load(filepath.Join("..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	return func(o *httpapi.Options) {
		o.Knowledge = cat
		o.ProfileThresholds = profile.Thresholds{Version: "profile-1",
			Amber: map[string]float64{"presence": 0.6}, Green: map[string]*float64{}}
	}
}

func put(t *testing.T, m *member, project, key string, body map[string]any, want int, out any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	m.call(t, http.MethodPut, "/api/projects/"+project+"/profile/"+key, raw, want, out)
}

func TestProfileReadEditAndSuggestions(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Warehouse")

	var p profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	if p.ProjectID != project || len(p.Parts) != 1 || p.Parts[0].Kind != "whole" || p.Thresholds["provisional"] != true {
		t.Fatalf("profile %+v", p)
	}
	if len(p.Compliance) != 4 {
		t.Fatalf("compliance groups %d", len(p.Compliance))
	}
	for _, g := range p.Compliance {
		for _, r := range g.Rows {
			if r.Key == "det.site_class" && (len(r.StatedIn) == 0 || r.StatedIn[0] != "Geotechnical report") {
				t.Fatalf("site class stated_in %v", r.StatedIn)
			}
			if r.Key == "det.type_of_construction" && (r.Derived == nil || r.Derived.Reason == "") {
				t.Fatalf("derived %+v", r.Derived)
			}
		}
	}

	put(t, m, project, "hdr.subclass", map[string]any{"value": "house"}, http.StatusOK, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new"}, http.StatusOK, &p)
	suggested := 0
	for _, g := range p.Systems {
		for _, r := range g.Rows {
			if r.Presence.Band == "suggested" && r.Shown {
				suggested++
			}
		}
	}
	if suggested == 0 {
		t.Fatal("choosing house x new must suggest systems")
	}

	put(t, m, project, "sys.hydraulic.gas.presence", map[string]any{"value": "not_included"}, http.StatusOK, &p)
	if got := presence(p, "hydraulic.gas"); got.Band != "user" || got.Value != "not_included" {
		t.Fatalf("user presence %+v", got)
	}
	put(t, m, project, "sys.hydraulic.gas.presence", map[string]any{"reset": true}, http.StatusOK, &p)
	if got := presence(p, "hydraulic.gas"); got.Band == "user" {
		t.Fatalf("reset presence %+v", got)
	}
}

func presence(p profileBody, leaf string) struct{ Value, Band string } {
	for _, g := range p.Systems {
		for _, r := range g.Rows {
			if r.Leaf == leaf {
				return struct{ Value, Band string }{r.Presence.Value, r.Presence.Band}
			}
		}
	}
	return struct{ Value, Band string }{}
}

func TestProfileValidatesAtBoundary(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "P")
	long := make([]byte, 121)
	for i := range long {
		long[i] = 'x'
	}
	cases := []struct {
		key  string
		body map[string]any
	}{
		{"hdr.nonsense", map[string]any{"value": "x"}},
		{"hdr.subclass", map[string]any{"value": "castle"}},
		{"sys.hydraulic.gas.presence", map[string]any{"value": "maybe"}},
		{"sys.hydraulic.presence", map[string]any{"value": "included"}},
		{"det.rise_in_storeys", map[string]any{"value": "five"}},
		{"det.heritage", map[string]any{"value": "stated_true"}},
		{"sys.hydraulic.gas.presence", map[string]any{"value": "included", "note": string(long)}},
	}
	for _, c := range cases {
		put(t, m, project, c.key, c.body, http.StatusUnprocessableEntity, nil)
	}
	other := a.member(t, newUUID(t))
	put(t, other, project, "hdr.subclass", map[string]any{"value": "house"}, http.StatusNotFound, nil)
	if got := other.status(t, http.MethodGet, "/api/projects/"+project+"/profile", nil); got != http.StatusNotFound {
		t.Fatalf("cross-org read = %d", got)
	}
}

func TestPartsCreateAndRename(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Petersham")
	var part struct{ ID, Label, Kind string }
	m.call(t, http.MethodPost, "/api/projects/"+project+"/parts", []byte(`{"label":"Building B","kind":"building","ncc_class":"2"}`), http.StatusCreated, &part)
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/parts/"+part.ID, []byte(`{"label":"Building B (mixed use)"}`), http.StatusOK, &part)
	if part.Label != "Building B (mixed use)" {
		t.Fatalf("rename %+v", part)
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/parts", []byte(`{"label":"Whole project","kind":"building"}`), http.StatusUnprocessableEntity, nil)
	if got := m.status(t, http.MethodDelete, "/api/projects/"+project+"/parts/"+part.ID, nil); got != http.StatusMethodNotAllowed && got != http.StatusNotFound {
		t.Fatalf("delete = %d", got)
	}
}
