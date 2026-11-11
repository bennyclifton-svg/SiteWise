package knowledge_test

import (
	"slices"
	"testing"

	"sitewise/internal/knowledge"
)

// The Accommodation catalogue loads, its keys parse and are project scoped,
// and its names do not claim what the card's scale fields already count.
func TestSpacesCatalogue(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Spaces()) < 20 {
		t.Fatalf("spaces = %d, want the full catalogue", len(cat.Spaces()))
	}
	id, attr, ok := cat.SpaceKey("space.meeting_rooms.count")
	if !ok || id != "meeting_rooms" || attr != "count" {
		t.Fatalf("SpaceKey = %q %q %v", id, attr, ok)
	}
	for _, bad := range []string{"space.ballroom_x.count", "space.meeting_rooms.colour", "hdr.scale.units"} {
		if _, _, ok := cat.SpaceKey(bad); ok {
			t.Errorf("SpaceKey(%q) accepted", bad)
		}
	}
	if got := cat.KeyScope("space.meeting_rooms.count"); got != knowledge.ScopeProject {
		t.Errorf("scope = %q", got)
	}
	names := map[string]string{
		"8 partner offices":               "offices",
		"6 meeting rooms (AV-ready)":      "meeting_rooms",
		"42 workstations":                 "workstations",
		"Commercial-grade kitchenette":    "kitchenette",
		"1 × secure records room":         "records_room",
		"1 × comms room":                  "comms_room",
		"60-seat breakout":                "breakout",
		"1 × accessible WC":               "accessible_toilets",
		"Open-plan legal library":         "library",
		"butler's pantry":                 "pantry",
		"two cot rooms":                   "sleep_room",
		"a covered outdoor learning area": "cola",
	}
	for text, want := range names {
		found := false
		for _, s := range cat.Spaces() {
			if s.ID == want && len(s.FindAll(text)) > 0 {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: %s does not match", text, want)
		}
	}
	// Counted on the card, never read as a space: the same fact must not be
	// held twice. Those spaces take their number of lines from the card.
	for _, text := range []string{"4 bedrooms", "2 bathrooms", "12 car spaces"} {
		for _, s := range cat.Spaces() {
			if len(s.FindAll(text)) > 0 {
				t.Errorf("%q matched space %s", text, s.ID)
			}
		}
	}
	bed, ok := cat.Space("bedroom")
	if !ok || bed.Read() || !slices.Equal(bed.CountFrom, []string{"hdr.scale.bedrooms"}) || bed.Kind != "room" {
		t.Errorf("bedroom = %+v", bed)
	}
}

// Every space names its line and says whether it is a room (one line each)
// or a setting (one line, its count in the comment).
func TestSpaceLines(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range cat.Spaces() {
		if s.Line == "" || s.Unit == "" || (s.Kind != "room" && s.Kind != "setting") {
			t.Errorf("%s: line %q unit %q kind %q", s.ID, s.Line, s.Unit, s.Kind)
		}
	}
	ws, _ := cat.Space("workstations")
	if ws.Kind != "setting" || ws.Line != "Open-plan workspace" || ws.Unit != "workstations" {
		t.Errorf("workstations = %+v", ws)
	}
	if o, _ := cat.Space("offices"); o.Kind != "room" || o.Line != "Office" {
		t.Errorf("offices = %+v", o)
	}
}

// A comment's requirement phrases and adjacencies are their own keys, each a
// yes/no question about one space.
func TestSpaceRequirementKeys(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"space.meeting_rooms.need.av_ready": "need.av_ready",
		"space.kitchen.next_to.dining":      "next_to.dining",
		"space.meeting_rooms.mix":           "mix",
		"space.offices.area_each_sqm":       "area_each_sqm",
	} {
		if _, attr, ok := cat.SpaceKey(key); !ok || attr != want {
			t.Errorf("SpaceKey(%q) = %q %v", key, attr, ok)
		}
	}
	for _, bad := range []string{"space.kitchen.next_to.kitchen", "space.kitchen.need.colour", "space.kitchen.next_to.nowhere", "space.kitchen.need"} {
		if _, _, ok := cat.SpaceKey(bad); ok {
			t.Errorf("SpaceKey(%q) accepted", bad)
		}
	}
	av, ok := cat.SpaceRequirement("av_ready")
	if !ok || av.Label != "AV-ready" {
		t.Fatalf("av_ready = %+v %v", av, ok)
	}
	for _, text := range []string{"AV-ready", "video conferencing", "audio visual"} {
		if len(av.FindAll(text)) == 0 {
			t.Errorf("av_ready misses %q", text)
		}
	}
	// "Av" in an address is not audio-visual.
	if len(av.FindAll("12 Smith Av, Parramatta")) > 0 {
		t.Error("av_ready matched an avenue")
	}
}

// The typical spaces follow the building type: a fit-out gets the tenancy,
// a new building also gets its base building; an unknown type falls back to
// its class; infrastructure has none.
func TestTypicalSpaces(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ts []knowledge.TypicalSpace) []string {
		var out []string
		for _, s := range ts {
			out = append(out, s.Space)
		}
		return out
	}
	fitout := ids(cat.TypicalSpaces("office", "commercial", "refurb"))
	for _, want := range []string{"reception", "workstations", "offices", "meeting_rooms", "kitchenette"} {
		if !slices.Contains(fitout, want) {
			t.Errorf("office fit-out lacks %s: %v", want, fitout)
		}
	}
	if slices.Contains(fitout, "lobby") || slices.Contains(fitout, "loading_dock") {
		t.Errorf("office fit-out has base-building spaces: %v", fitout)
	}
	newOffice := ids(cat.TypicalSpaces("office", "commercial", "new"))
	if !slices.Contains(newOffice, "lobby") || !slices.Contains(newOffice, "end_of_trip") {
		t.Errorf("new office lacks base building: %v", newOffice)
	}
	house := cat.TypicalSpaces("house", "residential", "new")
	if got := ids(house); !slices.Contains(got, "bedroom") || !slices.Contains(got, "kitchen") || got[0] != "entry" {
		t.Errorf("house = %v", got)
	}
	school := cat.TypicalSpaces("education_primary_secondary", "institution", "new")
	staff := false
	for _, s := range school {
		staff = staff || s.Space == "breakout" && s.Line == "Staff room"
	}
	if !staff || !slices.Contains(ids(school), "classrooms") {
		t.Errorf("school = %+v", school)
	}
	// Mixed use takes each of its parts' lists once.
	mixed := ids(cat.TypicalSpaces("residential_retail", "mixed", "new"))
	if !slices.Contains(mixed, "mail_room") || !slices.Contains(mixed, "sales_floor") {
		t.Errorf("residential + retail = %v", mixed)
	}
	seen := map[string]bool{}
	for _, id := range mixed {
		if seen[id] {
			t.Errorf("%s listed twice", id)
		}
		seen[id] = true
	}
	if got := cat.TypicalSpaces("other", "commercial", "refurb"); len(got) == 0 {
		t.Error("an Other commercial building has no fallback list")
	}
	if got := cat.TypicalSpaces("roads_highways", "infrastructure", "new"); len(got) != 0 {
		t.Errorf("infrastructure = %v", ids(got))
	}
	if got := cat.TypicalSpaces("", "", ""); len(got) != 0 {
		t.Errorf("unknown building = %v", ids(got))
	}
}
