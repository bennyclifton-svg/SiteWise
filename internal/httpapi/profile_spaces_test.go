package httpapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
)

type scheduleLine struct {
	ID      string                       `json:"id"`
	Origin  string                       `json:"origin"`
	Name    struct{ Value, Band string } `json:"name"`
	Level   struct{ Value, Band string } `json:"level"`
	Area    struct{ Value, Band string } `json:"area"`
	Comment struct{ Value, Band string } `json:"comment"`
}

type scheduleBody struct {
	Lines   []scheduleLine `json:"lines"`
	Notices []string       `json:"notices"`
	Names   []string       `json:"names"`
}

func (s scheduleBody) ids() []string {
	var out []string
	for _, l := range s.Lines {
		out = append(out, l.ID)
	}
	return out
}

func (s scheduleBody) line(t *testing.T, id string) scheduleLine {
	t.Helper()
	for _, l := range s.Lines {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("no line %s in %v", id, s.ids())
	return scheduleLine{}
}

func editSchedule(t *testing.T, m *member, project string, body map[string]any, want int) scheduleBody {
	t.Helper()
	raw, _ := json.Marshal(body)
	var out scheduleBody
	var dest any
	if want == http.StatusOK {
		dest = &out
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/profile/accommodation", raw, want, dest)
	return out
}

// The Accommodation schedule (docs/plans/2026-11-11-accommodation-schedule-lines.md):
// placeholders from the building type, then the user's edits, additions,
// copies and deletions, each kept across reads.
func TestProfileAccommodation(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Fit-out")
	var p struct {
		Accommodation scheduleBody `json:"accommodation"`
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	if len(p.Accommodation.Lines) != 0 || len(p.Accommodation.Names) < 20 {
		t.Fatalf("a project with no building type: %d lines, %d names", len(p.Accommodation.Lines), len(p.Accommodation.Names))
	}
	for key, value := range map[string]string{"hdr.building_class": "commercial", "hdr.subclass": "office", "hdr.work_type": "refurb"} {
		put(t, m, project, key, map[string]any{"value": value}, http.StatusOK, nil)
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	s := p.Accommodation
	if r := s.line(t, "reception#1"); r.Origin != "typical" || r.Name.Band != "typical" {
		t.Fatalf("reception = %+v", r)
	}
	s = editSchedule(t, m, project, map[string]any{"op": "set", "lines": []string{"reception#1", "offices#1"}, "field": "level", "value": " Level 3 "}, http.StatusOK)
	if l := s.line(t, "reception#1").Level; l.Value != "Level 3" || l.Band != "user" {
		t.Fatalf("level = %+v", l)
	}
	s = editSchedule(t, m, project, map[string]any{"op": "set", "lines": []string{"offices#1"}, "field": "area", "value": "12.50 m²"}, http.StatusOK)
	if a := s.line(t, "offices#1").Area.Value; a != "12.5" {
		t.Fatalf("area = %q", a)
	}
	s = editSchedule(t, m, project, map[string]any{"op": "insert", "order": append(s.ids(), "new:0"), "value": "Server cupboard"}, http.StatusOK)
	added := s.Lines[len(s.Lines)-1]
	if added.Name.Value != "Server cupboard" || added.Origin != "user" || added.Name.Band != "user" {
		t.Fatalf("added = %+v", added)
	}
	s = editSchedule(t, m, project, map[string]any{"op": "duplicate", "lines": []string{"offices#1"}}, http.StatusOK)
	at := slices.Index(s.ids(), "offices#1")
	if c := s.Lines[at+1]; c.Name.Value != "Office 2" || c.Level.Value != "Level 3" || c.Area.Value != "12.5" || c.Origin != "user" {
		t.Fatalf("copy = %+v", c)
	}
	s = editSchedule(t, m, project, map[string]any{"op": "delete", "lines": []string{"reception#1", added.ID}}, http.StatusOK)
	if slices.Contains(s.ids(), "reception#1") || slices.Contains(s.ids(), added.ID) {
		t.Fatalf("deleted lines remain: %v", s.ids())
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	if slices.Contains(p.Accommodation.ids(), "reception#1") {
		t.Fatal("a deleted placeholder came back on the next read")
	}
	s = editSchedule(t, m, project, map[string]any{"op": "reset", "lines": []string{"reception#1"}}, http.StatusOK)
	if r := s.line(t, "reception#1"); r.Level.Value != "" || r.Origin != "typical" {
		t.Fatalf("reset = %+v", r)
	}
	order := s.ids()
	slices.Reverse(order)
	s = editSchedule(t, m, project, map[string]any{"op": "order", "order": order}, http.StatusOK)
	if !slices.Equal(s.ids(), order) {
		t.Fatalf("order = %v, want %v", s.ids(), order)
	}

	for _, bad := range []map[string]any{
		{"op": "paint", "lines": []string{"offices#1"}},
		{"op": "set", "lines": []string{"offices#1"}, "field": "colour", "value": "red"},
		{"op": "set", "lines": []string{"offices#1"}, "field": "area", "value": "large"},
		{"op": "set", "lines": []string{"offices#1"}, "field": "area", "value": "-4"},
		{"op": "set", "lines": []string{"offices#1"}, "field": "name", "value": "  "},
		{"op": "set", "lines": []string{"ballroom_x#1"}, "field": "level", "value": "1"},
		{"op": "set", "lines": []string{"offices#0"}, "field": "level", "value": "1"},
		{"op": "insert", "order": []string{"offices#1"}, "value": "No place"},
		{"op": "split", "lines": []string{"offices#1", "offices#2"}},
	} {
		editSchedule(t, m, project, bad, http.StatusUnprocessableEntity)
	}
	other := a.member(t, newUUID(t))
	editSchedule(t, other, project, map[string]any{"op": "set", "lines": []string{"offices#1"}, "field": "level", "value": "Level 9"}, http.StatusNotFound)
	editSchedule(t, other, project, map[string]any{"op": "delete", "lines": []string{added.ID}}, http.StatusNotFound)
}
