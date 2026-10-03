package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type scopeBody struct {
	Header []struct {
		Key, Label string
	} `json:"header"`
	Systems []struct {
		Rows []struct {
			Leaf        string `json:"leaf"`
			InScope     bool   `json:"in_scope"`
			ScopeOrigin string `json:"scope_origin"`
			Shown       bool   `json:"shown_by_default"`
		} `json:"rows"`
	} `json:"systems"`
	Compliance []struct {
		Rows []struct {
			Key      string `json:"key"`
			Label    string `json:"label"`
			Relevant *bool  `json:"relevant"`
		} `json:"rows"`
	} `json:"compliance"`
	Presets []struct {
		ID      string   `json:"id"`
		Systems []string `json:"systems"`
	} `json:"presets"`
}

func (p scopeBody) inScope() map[string]string {
	out := map[string]string{}
	for _, g := range p.Systems {
		for _, r := range g.Rows {
			if r.InScope {
				if !r.Shown {
					panic(r.Leaf + " in scope but hidden")
				}
				out[r.Leaf] = r.ScopeOrigin
			}
		}
	}
	return out
}

func (p scopeBody) relevant(key string) bool {
	for _, g := range p.Compliance {
		for _, r := range g.Rows {
			if r.Key == key {
				return r.Relevant != nil && *r.Relevant
			}
		}
	}
	return false
}

func TestScopeOfWorksDrivesTheProfile(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Sprinkler pumps")
	var p scopeBody
	put(t, m, project, "hdr.subclass", map[string]any{"value": "warehouse"}, http.StatusOK, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "refurb"}, http.StatusOK, &p)
	if got := p.inScope(); len(got) != 0 {
		t.Fatalf("refurbishment must start with nothing in scope: %v", got)
	}
	labels := map[string]string{}
	for _, h := range p.Header {
		labels[h.Key] = h.Label
	}
	if labels["hdr.building_class"] != "Building category" || labels["hdr.subclass"] != "Building class" {
		t.Fatalf("labels %v", labels)
	}
	if len(p.Presets) == 0 || p.Presets[0].ID != "typical_fit_out" {
		t.Fatalf("presets %+v", p.Presets)
	}

	path := "/api/projects/" + project + "/profile/scope"
	set := func(systems map[string]any, want int, out any) {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"systems": systems})
		m.call(t, http.MethodPut, path, raw, want, out)
	}
	set(map[string]any{"fire-active.sprinklers": "in"}, http.StatusOK, &p)
	if got := p.inScope(); len(got) != 1 || got["fire-active.sprinklers"] != "user" {
		t.Fatalf("scope %v", got)
	}
	if !p.relevant("det.storage_height") || !p.relevant("det.ncc_class") || p.relevant("det.site_class") || p.relevant("det.bal") {
		t.Fatal("compliance relevance does not follow the scope")
	}
	for _, g := range p.Compliance {
		for _, r := range g.Rows {
			if r.Key == "det.ncc_class" && r.Label != "NCC class" {
				t.Fatalf("ncc class label %q", r.Label)
			}
		}
	}

	set(map[string]any{"fire-active.sprinklers": nil}, http.StatusOK, &p)
	if got := p.inScope(); len(got) != 0 {
		t.Fatalf("null hands the system back: %v", got)
	}
	if !p.relevant("det.site_class") || !p.relevant("det.bal") {
		t.Fatal("with nothing in scope every compliance row shows")
	}

	for _, bad := range []map[string]any{
		{},
		{"fire-active": "in"},
		{"fire-active.nonsense": "in"},
		{"fire-active.sprinklers": "maybe"},
	} {
		set(bad, http.StatusUnprocessableEntity, nil)
	}
	put(t, m, project, "scope.fire-active.sprinklers", map[string]any{"value": "in"}, http.StatusUnprocessableEntity, nil)
	other := a.member(t, newUUID(t))
	raw, _ := json.Marshal(map[string]any{"systems": map[string]any{"fire-active.sprinklers": "in"}})
	if code := other.status(t, http.MethodPut, path, raw); code != http.StatusNotFound {
		t.Fatalf("cross-org scope write = %d", code)
	}
}

func TestNewBuildScopeStartsFromDefaults(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "House")
	var p scopeBody
	put(t, m, project, "hdr.subclass", map[string]any{"value": "house"}, http.StatusOK, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "new"}, http.StatusOK, &p)
	got := p.inScope()
	if got["substructure.footings"] != "default" || len(got) < 20 {
		t.Fatalf("house defaults %v", got)
	}
	if !p.relevant("det.site_class") || p.relevant("det.storage_height") {
		t.Fatal("a new house shows site class and not storage height")
	}
}
