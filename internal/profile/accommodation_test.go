package profile_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"sitewise/internal/profile"
)

// The Accommodation schedule as lines
// (docs/plans/2026-11-11-accommodation-schedule-lines.md): placeholders from
// the building type, lines from the documents' readings and the card's
// counts, and the user's edits on top. Code only; nothing here calls Jev.

func srow(key, value, band string) profile.Row {
	return profile.Row{Key: key, Value: value, Band: band, Sources: []profile.Source{{DocumentID: "d1", Excerpt: key}}}
}

func rowsOf(rs ...profile.Row) map[string]profile.Row {
	out := map[string]profile.Row{}
	for _, r := range rs {
		out[r.Key] = r
	}
	return out
}

func lineIDs(s profile.Schedule) []string {
	var out []string
	for _, l := range s.Lines {
		out = append(out, l.ID)
	}
	return out
}

func lineByID(t *testing.T, s profile.Schedule, id string) profile.Line {
	t.Helper()
	for _, l := range s.Lines {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("no line %s in %v", id, lineIDs(s))
	return profile.Line{}
}

func office(extra ...profile.Row) map[string]profile.Row {
	return rowsOf(append([]profile.Row{
		srow("hdr.building_class", "commercial", "green"),
		srow("hdr.subclass", "office", "green"),
		srow("hdr.work_type", "refurb", "amber"),
	}, extra...)...)
}

func TestScheduleTypicalPlaceholders(t *testing.T) {
	cat := repoCatalog(t)
	s := profile.BuildSchedule(cat, office(), nil)
	if len(s.Lines) == 0 {
		t.Fatal("an office fit-out has no placeholders")
	}
	r := lineByID(t, s, "reception#1")
	if r.Name.Value != "Reception" || r.Name.Band != "typical" || r.Origin != "typical" || r.Level.Value != "" || r.Area.Value != "" {
		t.Errorf("reception = %+v", r)
	}
	if slices.Contains(lineIDs(s), "lobby#1") {
		t.Error("a fit-out has the base building's lobby")
	}
	// No building type yet: no placeholders.
	if got := profile.BuildSchedule(cat, rowsOf(), nil); len(got.Lines) != 0 {
		t.Errorf("unknown building = %v", lineIDs(got))
	}
}

func TestScheduleFromReadings(t *testing.T) {
	cat := repoCatalog(t)
	rows := office(
		srow("space.offices.provided", "yes", "green"),
		srow("space.offices.count", "8", "green"),
		srow("space.offices.level", "Level 3, Level 4", "amber"),
		srow("space.meeting_rooms.provided", "yes", "green"),
		srow("space.meeting_rooms.count", "6", "green"),
		srow("space.meeting_rooms.mix", "4x8,2x16", "green"),
		srow("space.meeting_rooms.need.av_ready", "yes", "amber"),
		srow("space.workstations.count", "42", "green"),
		srow("space.library.provided", "yes", "green"),
		srow("space.library.area_sqm", "180", "green"),
		srow("space.breakout.capacity", "60", "green"),
		srow("space.reception.provided", "yes", "green"),
		srow("space.boardroom.provided", "no", "green"),
	)
	s := profile.BuildSchedule(cat, rows, nil)
	for i := 1; i <= 8; i++ {
		l := lineByID(t, s, fmt.Sprintf("offices#%d", i))
		if l.Name.Value != fmt.Sprintf("Office %d", i) || l.Name.Band != "green" || l.Origin != "document" ||
			l.Level.Value != "Level 3, Level 4" || l.Level.Band != "amber" {
			t.Errorf("office %d = %+v", i, l)
		}
	}
	if slices.Contains(lineIDs(s), "offices#9") {
		t.Error("a ninth office")
	}
	if c := lineByID(t, s, "meeting_rooms#1").Comment; c.Value != "Capacity 8, AV-ready" || c.Band != "amber" {
		t.Errorf("meeting room 1 comment = %+v", c)
	}
	if c := lineByID(t, s, "meeting_rooms#5").Comment.Value; c != "Capacity 16, AV-ready" {
		t.Errorf("meeting room 5 comment = %q", c)
	}
	ws := lineByID(t, s, "workstations#1")
	if ws.Name.Value != "Open-plan workspace" || ws.Comment.Value != "42 workstations" || slices.Contains(lineIDs(s), "workstations#2") {
		t.Errorf("workstations = %+v", ws)
	}
	lib := lineByID(t, s, "library#1")
	if lib.Area.Value != "180" || lib.Area.Band != "green" || len(lib.Area.Keys) != 1 {
		t.Errorf("library = %+v", lib)
	}
	if c := lineByID(t, s, "breakout#1").Comment.Value; c != "Capacity 60" {
		t.Errorf("breakout comment = %q", c)
	}
	// A placeholder the documents confirm is the same line, now theirs; one
	// they say is not required goes.
	if r := lineByID(t, s, "reception#1"); r.Origin != "document" || r.Name.Band != "green" {
		t.Errorf("reception = %+v", r)
	}
	if slices.Contains(lineIDs(s), "boardroom#1") {
		t.Error("the boardroom the documents rule out is shown")
	}
	// A space only the documents name sits among the typical ones in the
	// catalogue's order.
	ids := lineIDs(s)
	if at := slices.Index(ids, "library#1"); at < slices.Index(ids, "quiet_rooms#1") || at > slices.Index(ids, "print_copy#1") {
		t.Errorf("library placed at %d in %v", at, ids)
	}
}

func TestScheduleAreasAndConflicts(t *testing.T) {
	cat := repoCatalog(t)
	group := profile.BuildSchedule(cat, office(
		srow("space.consult_rooms.count", "6", "green"),
		srow("space.consult_rooms.area_sqm", "140", "green"),
	), nil)
	l := lineByID(t, group, "consult_rooms#3")
	if l.Area.Value != "" || !strings.Contains(l.Comment.Value, "140 m² for all 6") {
		t.Errorf("an area for six rooms put on one: %+v", l)
	}
	each := profile.BuildSchedule(cat, office(
		srow("space.consult_rooms.count", "6", "green"),
		srow("space.consult_rooms.area_each_sqm", "14", "green"),
	), nil)
	if a := lineByID(t, each, "consult_rooms#6").Area; a.Value != "14" || a.Band != "green" {
		t.Errorf("per-room area = %+v", a)
	}
	conflict := srow("space.offices.count", "", "red")
	conflict.Alternatives = []profile.Alternative{{Value: "6"}, {Value: "8"}}
	s := profile.BuildSchedule(cat, office(srow("space.offices.provided", "yes", "green"), conflict), nil)
	o := lineByID(t, s, "offices#1")
	if slices.Contains(lineIDs(s), "offices#2") || !strings.Contains(o.Comment.Value, "6 or 8 in the documents") || o.Comment.Band != "red" {
		t.Errorf("disputed count = %+v", o)
	}
	adj := profile.BuildSchedule(cat, rowsOf(srow("hdr.subclass", "house", "user"),
		srow("space.kitchen.next_to.dining", "yes", "green")), nil)
	if c := lineByID(t, adj, "kitchen#1").Comment.Value; c != "Adjacent to dining" {
		t.Errorf("kitchen comment = %q", c)
	}
}

func TestScheduleCardCountsAndCap(t *testing.T) {
	cat := repoCatalog(t)
	house := rowsOf(srow("hdr.subclass", "house", "user"), srow("hdr.scale.bedrooms", "4", "green"), srow("hdr.scale.garage_spaces", "2", "amber"))
	s := profile.BuildSchedule(cat, house, nil)
	for i := 1; i <= 4; i++ {
		if b := lineByID(t, s, fmt.Sprintf("bedroom#%d", i)); b.Name.Value != fmt.Sprintf("Bedroom %d", i) || b.Origin != "card" || b.Name.Band != "green" {
			t.Errorf("bedroom %d = %+v", i, b)
		}
	}
	if g := lineByID(t, s, "garage#1"); g.Comment.Value != "2 car spaces" || g.Comment.Band != "amber" {
		t.Errorf("garage = %+v", g)
	}
	if len(s.Notices) != 0 {
		t.Errorf("notices = %v", s.Notices)
	}
	// One bedroom removed: the card keeps its count and the difference shows.
	s = profile.BuildSchedule(cat, house, []profile.StoredLine{{ID: "r1", Key: "bedroom#4", Removed: true}})
	if slices.Contains(lineIDs(s), "bedroom#4") || len(s.Notices) != 1 || s.Notices[0] != "The card has 4 bedrooms; the schedule lists 3." {
		t.Errorf("after removal: %v, notices %v", lineIDs(s), s.Notices)
	}
	hotel := rowsOf(srow("hdr.subclass", "hotel", "green"), srow("hdr.scale.rooms", "200", "green"))
	s = profile.BuildSchedule(cat, hotel, nil)
	g := lineByID(t, s, "guest_rooms#1")
	if g.Name.Value != "Guest rooms" || g.Comment.Value != "200 guest rooms" || g.Many != 200 || slices.Contains(lineIDs(s), "guest_rooms#2") {
		t.Errorf("capped = %+v", g)
	}
	s = profile.BuildSchedule(cat, hotel, []profile.StoredLine{{ID: "r1", Key: "guest_rooms#1", Split: true}})
	if !slices.Contains(lineIDs(s), "guest_rooms#200") || lineByID(t, s, "guest_rooms#200").Name.Value != "Guest room 200" || lineByID(t, s, "guest_rooms#1").Many != 0 {
		t.Errorf("split: %d lines", len(s.Lines))
	}
}

func TestScheduleUserEditsAreFinal(t *testing.T) {
	cat := repoCatalog(t)
	rows := office(
		srow("space.offices.provided", "yes", "green"),
		srow("space.offices.count", "6", "green"),
		srow("space.offices.level", "Level 3", "green"),
	)
	pos := func(f float64) *float64 { return &f }
	stored := []profile.StoredLine{
		{ID: "r1", Key: "offices#2", Name: "Partner office (Smith)", Set: []string{"name"}},
		{ID: "r2", Key: "reception#1", Removed: true},
		{ID: "r3", Name: "Server cupboard", Level: "Level 3", Set: []string{"name", "level"}, Position: pos(0.5)},
		{ID: "r4", Key: "offices#8", Level: "Level 4", Set: []string{"level"}},
		{ID: "r5", Key: "offices#7", Position: pos(40)},
	}
	s := profile.BuildSchedule(cat, rows, stored)
	o := lineByID(t, s, "offices#2")
	if o.Name.Value != "Partner office (Smith)" || o.Name.Band != "user" || o.Level.Value != "Level 3" || o.Level.Band != "green" {
		t.Errorf("edited office = %+v", o)
	}
	if slices.Contains(lineIDs(s), "reception#1") {
		t.Error("a deleted line came back")
	}
	if ids := lineIDs(s); len(ids) < 2 || ids[1] != "r3" {
		t.Errorf("the line placed second is at %v", ids)
	}
	u := lineByID(t, s, "r3")
	if u.Origin != "user" || u.Name.Band != "user" || u.Level.Value != "Level 3" {
		t.Errorf("added line = %+v", u)
	}
	// The documents now list six: the edited eighth office stays, flagged;
	// a seventh that held only its place goes.
	e := lineByID(t, s, "offices#8")
	if e.Flag != "The documents now list 6 offices." || e.Level.Value != "Level 4" || e.Name.Value != "Office 8" {
		t.Errorf("orphan = %+v", e)
	}
	if slices.Contains(lineIDs(s), "offices#7") {
		t.Error("a placeholder-only position kept a line the documents dropped")
	}
}
