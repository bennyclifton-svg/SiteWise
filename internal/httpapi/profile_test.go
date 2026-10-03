package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"sitewise/internal/httpapi"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sort"
	"strconv"
	"time"
)

func TestProfileSourceReadIsolationAndLatencyBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Source coverage")
	// The authenticated empty-project path is also a real endpoint budget check.
	var durations []time.Duration
	for i := 0; i < 25; i++ {
		started := time.Now()
		var v store.SourceRecords
		m.getJSON(t, "/api/projects/"+project+"/profile/sources", &v)
		durations = append(durations, time.Since(started))
		if v.Records == nil {
			t.Fatal("records must encode as an array")
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if durations[12] > 50*time.Millisecond || durations[22] > 150*time.Millisecond {
		t.Fatalf("source endpoint exceeded p50/p90 budget: %s/%s", durations[12], durations[22])
	}
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodGet, "/api/projects/"+project+"/profile/sources", nil); got != 404 {
		t.Fatalf("foreign source read %d", got)
	}
	for _, q := range []string{"offset=-1", "outcome=invalid", "system=invalid"} {
		if got := m.status(t, http.MethodGet, "/api/projects/"+project+"/profile/sources?"+q, nil); got != 400 {
			t.Fatalf("invalid query %s=%d", q, got)
		}
	}
	// Coverage remains org-scoped even though profiles expose per-document counts.
	if v, err := a.store.SourceCoverage(context.Background(), other.orgID, project); err != nil || len(v) != 0 {
		t.Fatalf("foreign coverage %+v %v", v, err)
	}
}

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
	reading, err := profile.LoadReadPolicy(filepath.Join("..", "..", "data", "profile", "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	return func(o *httpapi.Options) {
		o.Knowledge = cat
		o.ProfileReading = reading
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

func TestProfileShowsKnownScaleBeforeBuildingType(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Unclassified brief")
	var p profileBody
	put(t, m, project, "hdr.scale.gla_sqm", map[string]any{"value": "2135"}, http.StatusOK, &p)
	check := func() {
		t.Helper()
		for _, f := range p.Header {
			if f.Key == "hdr.scale.gla_sqm" && f.Value == "2135" {
				return
			}
		}
		t.Fatal("saved scale hidden by building type")
	}
	check()
	put(t, m, project, "hdr.subclass", map[string]any{"value": "warehouse"}, http.StatusOK, &p)
	check()
}

func TestProfileReadEditLatencyBudgets(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Profile timing")
	put(t, m, project, "hdr.subclass", map[string]any{"value": "warehouse"}, http.StatusOK, nil)
	put(t, m, project, "hdr.work_type", map[string]any{"value": "extend"}, http.StatusOK, nil)
	for _, path := range []string{"project_profile_read", "profile_edit"} {
		var durations []time.Duration
		for i := 0; i < 25; i++ {
			start := time.Now()
			if path == "profile_edit" {
				put(t, m, project, "hdr.scale.gla_sqm", map[string]any{"value": strconv.Itoa(2100 + i)}, http.StatusOK, nil)
			} else {
				var p profileBody
				m.getJSON(t, "/api/projects/"+project+"/profile", &p)
			}
			durations = append(durations, time.Since(start))
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		t.Logf("%s p50=%s p90=%s (budget 50ms/150ms)", path, durations[12], durations[22])
		if durations[12] > 50*time.Millisecond || durations[22] > 150*time.Millisecond {
			t.Fatalf("%s exceeded budget", path)
		}
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

func TestProfileReadRequest(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Hale")
	var got struct {
		ProjectID string `json:"project_id"`
		Queued    *int   `json:"queued"`
		Unread    *int   `json:"unread_documents"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/profile/read", nil, http.StatusOK, &got)
	if got.ProjectID != project || got.Queued == nil || *got.Queued != 0 || got.Unread == nil || *got.Unread != 0 {
		t.Fatalf("read request %+v", got)
	}
	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodPost, "/api/projects/"+project+"/profile/read", nil); code != http.StatusNotFound {
		t.Fatalf("cross-org read request = %d", code)
	}
}

func TestProfileReadingSetting(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Reading")
	doc := m.upload(t, project, "identity-page.pdf", http.StatusCreated)
	path := "/api/projects/" + project + "/documents/profile-read"
	body := func(setting string, ids ...string) []byte {
		raw, _ := json.Marshal(map[string]any{"document_ids": ids, "setting": setting})
		return raw
	}

	var p struct {
		ReadDocuments    *int `json:"read_documents"`
		SkippedDocuments *int `json:"skipped_documents"`
	}
	m.call(t, http.MethodPut, path, body("skip", doc.ID), http.StatusOK, &p)
	if p.SkippedDocuments == nil || *p.SkippedDocuments != 1 || *p.ReadDocuments != 0 {
		t.Fatalf("counts %+v", p)
	}
	var list struct {
		Documents []struct {
			ID          string `json:"id"`
			ProfileRead string `json:"profile_read"`
		} `json:"documents"`
	}
	m.getJSON(t, "/api/projects/"+project+"/documents", &list)
	if len(list.Documents) != 1 || list.Documents[0].ProfileRead != "skip" {
		t.Fatalf("list %+v", list)
	}
	m.call(t, http.MethodPut, path, body("read", doc.ID), http.StatusOK, &p)
	if *p.ReadDocuments != 1 {
		t.Fatalf("read counts %+v", p)
	}

	m.call(t, http.MethodPut, path, body("maybe", doc.ID), http.StatusBadRequest, nil)
	m.call(t, http.MethodPut, path, body("skip"), http.StatusBadRequest, nil)
	m.call(t, http.MethodPut, path, body("skip", "not-a-uuid"), http.StatusNotFound, nil)
	m.call(t, http.MethodPut, path, body("skip", newUUID(t)), http.StatusNotFound, nil)

	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodPut, path, body("skip", doc.ID)); code != http.StatusNotFound {
		t.Fatalf("cross-org setting = %d", code)
	}
	otherProject := other.createProject(t, "Theirs")
	if code := m.status(t, http.MethodPut, "/api/projects/"+otherProject+"/documents/profile-read", body("skip", doc.ID)); code != http.StatusNotFound {
		t.Fatalf("setting through another org's project = %d", code)
	}

	var cat struct {
		Kinds []string `json:"profile_read_kinds"`
	}
	m.getJSON(t, "/api/catalog", &cat)
	if len(cat.Kinds) == 0 {
		t.Fatal("catalog lacks the automatically read kinds")
	}
}
