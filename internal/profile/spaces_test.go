package profile_test

import (
	"slices"
	"strings"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/profile"
)

// The Meridian brief's table and the fee proposal's scope list, as filed:
// each space and the values written beside it become candidates for that
// space only (docs/plans/2026-11-11-accommodation-and-full-capture.md).
func TestHarvestSpacesMeridian(t *testing.T) {
	cat := repoCatalog(t)
	brief := strings.Join([]string{
		"## Functional requirements",
		"| Area | Requirement |",
		"| --- | --- |",
		"| Reception & client waiting | Ground-floor lobby presence not required; Level 3 entry suite with secure reception |",
		"| Open-plan legal library | ~180 m² open work area with 42 workstations |",
		"| Partner offices | 8 enclosed offices (3 on Level 3, 5 on Level 4 mezzanine insert) |",
		"| Meeting rooms | 4 × 8-person, 2 × 16-person (AV-ready) |",
		"| Staff breakout / kitchen | Commercial-grade kitchenette, 60-seat breakout |",
		"| IT / records | 1 × secure records room, 1 × comms room (landlord riser access required) |",
		"| Amenities | 2 × male, 2 × female, 1 × accessible WC; shower on Level 4 |",
	}, "\n")
	fee := strings.Join([]string{
		"- Open-plan legal library (~180 m²) and 42 workstations",
		"- 8 partner offices and 6 meeting rooms (AV-ready)",
		"- Level 4 mezzanine insert (~185 m²)",
		"- Kitchenette, breakout, records and comms rooms",
	}, "\n")
	cases := []struct {
		name string
		text string
		want map[string][]string // key -> candidate norms that must be offered
		none []string            // keys that must offer nothing
	}{
		{"brief", brief, map[string][]string{
			"space.library.area_sqm":         {"180"},
			"space.workstations.count":       {"42"},
			"space.offices.count":            {"8"},
			"space.offices.level":            {"Level 3, Level 4"},
			"space.meeting_rooms.count":      {"6"},
			"space.meeting_rooms.capacity":   {"8", "16"},
			"space.breakout.capacity":        {"60"},
			"space.records_room.count":       {"1"},
			"space.comms_room.count":         {"1"},
			"space.accessible_toilets.count": {"1"},
			"space.toilets.count":            {"4"},
			"space.showers.level":            {"Level 4"},
			"space.reception.level":          {"Level 3"},
			"space.lobby.level":              {"Ground floor"},
		}, []string{"space.workstations.area_sqm", "space.library.count", "space.kitchenette.capacity", "space.reception.level:Ground floor"}},
		{"fee", fee, map[string][]string{
			"space.library.area_sqm":    {"180"},
			"space.workstations.count":  {"42"},
			"space.offices.count":       {"8"},
			"space.meeting_rooms.count": {"6"},
		}, []string{"space.offices.area_sqm", "space.meeting_rooms.area_sqm"}},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.text, cat)
		for key, want := range tc.want {
			got := spaceNorms(h.Candidates[key])
			for _, n := range want {
				if !slices.Contains(got, n) {
					t.Errorf("%s: %s candidates %v, want %q", tc.name, key, got, n)
				}
			}
		}
		for _, key := range tc.none {
			key, value, one := strings.Cut(key, ":")
			got := spaceNorms(h.Candidates[key])
			if one && slices.Contains(got, value) || !one && len(got) > 0 {
				t.Errorf("%s: %s must not offer %v", tc.name, key, got)
			}
		}
	}
	// Every space named is asked whether it is provided, numbers or not.
	h := profile.Harvest(fee, cat)
	for _, id := range []string{"kitchenette", "breakout", "records_room", "comms_room", "library"} {
		if !h.Has("space." + id + ".provided") {
			t.Errorf("fee: %s not asked whether provided", id)
		}
	}
	// The mezzanine is added floor area on the card, not a room.
	for key := range h.Candidates {
		if strings.Contains(key, "185") {
			t.Errorf("mezzanine area offered as %s", key)
		}
	}
	for key, cands := range h.Candidates {
		if strings.HasPrefix(key, "space.") && slices.Contains(spaceNorms(cands), "185") {
			t.Errorf("mezzanine area offered to %s", key)
		}
	}
}

// Space questions are asked on brief-like passages only, and each candidate
// question offers none.
func TestSpaceQuestionsRouting(t *testing.T) {
	cat := repoCatalog(t)
	text := "- 8 partner offices and 6 meeting rooms (AV-ready)"
	h := profile.Harvest(text, cat)
	qs, state := profile.LabelQuestions(profile.PassageInfo{Kind: "design_brief", Ordinal: 3, Text: text}, h, cat)
	q, ok := qs["space.meeting_rooms.count"]
	if !ok {
		t.Fatal("meeting room count not asked")
	}
	if q.Type != jev.TypeChoice || q.Criteria.(map[string]string)["none"] == "" || len(state["space.meeting_rooms.count"]) == 0 {
		t.Fatalf("count question = %+v", q)
	}
	if _, ok := qs["space.meeting_rooms.provided"]; !ok {
		t.Fatal("provided not asked")
	}
	// A specification's later clause does not fill the table.
	qs, _ = profile.LabelQuestions(profile.PassageInfo{Kind: "drawing", Ordinal: 40, Text: text}, h, cat)
	for id := range qs {
		if strings.HasPrefix(id, "space.") {
			t.Fatalf("asked %s on a passage without the project questions", id)
		}
	}
}

func spaceNorms(cands []profile.Candidate) []string {
	var out []string
	for _, c := range cands {
		out = append(out, c.Norm)
	}
	return out
}

// The same table reads other building types; a number beside a room that
// counts something else is not the room's count.
func TestHarvestSpacesOtherTypes(t *testing.T) {
	cat := repoCatalog(t)
	school := strings.Join([]string{
		"| Space | Quantity | Area |",
		"| --- | --- | --- |",
		"| General learning spaces | 12 | 75 m² each |",
		"| Multi-purpose hall | 1 | 600 m² (seats 400) |",
	}, "\n")
	h := profile.Harvest(school, cat)
	for key, want := range map[string]string{
		"space.classrooms.count": "12", "space.classrooms.area_each_sqm": "75",
		"space.hall.count": "1", "space.hall.area_sqm": "600", "space.hall.capacity": "400",
	} {
		if !slices.Contains(spaceNorms(h.Candidates[key]), want) {
			t.Errorf("school: %s = %v, want %s", key, spaceNorms(h.Candidates[key]), want)
		}
	}
	if slices.Contains(spaceNorms(h.Candidates["space.hall.count"]), "400") {
		t.Error("school: seats counted as halls")
	}
	spec := "The contractor shall protect the boardroom ceiling during demolition of 3 walls."
	if got := profile.Harvest(spec, cat).Candidates["space.boardroom.count"]; len(got) > 0 {
		t.Errorf("spec: boardroom count %v from another noun's number", spaceNorms(got))
	}
	// Levels of the whole fit-out in a running list are not a room's level.
	rom := "for scope as described (Levels 3–4 fit-out, mezzanine insert, AV-ready meeting rooms), assuming:"
	h = profile.Harvest(rom, cat)
	if got := h.Candidates["space.meeting_rooms.level"]; len(got) > 0 || len(h.Candidates["space.meeting_rooms.count"]) > 0 {
		t.Errorf("rom: meeting rooms given %v / %v", spaceNorms(got), spaceNorms(h.Candidates["space.meeting_rooms.count"]))
	}
}

// A line's comment comes from what the brief requires of each space: its
// requirement phrases, what it sits next to, a per-room area and a list of
// room sizes (docs/plans/2026-11-11-accommodation-schedule-lines.md). Each is
// asked of the space named in its own clause only.
func TestHarvestSpaceComments(t *testing.T) {
	cat := repoCatalog(t)
	brief := strings.Join([]string{
		"| Meeting rooms | 4 × 8-person, 2 × 16-person (AV-ready) |",
		"| IT / records | 1 × secure records room, 1 × comms room (landlord riser access required) |",
		"| Consult rooms | 6 | 14 m² each |",
		"Kitchen to be adjacent to the dining room, with a walk-in pantry.",
		"The boardroom needs natural light; the comms room needs dedicated cooling.",
	}, "\n")
	h := profile.Harvest(brief, cat)
	for _, key := range []string{
		"space.meeting_rooms.need.av_ready",
		"space.records_room.need.secure",
		"space.kitchen.next_to.dining",
		"space.boardroom.need.natural_light",
		"space.comms_room.need.dedicated_cooling",
	} {
		if !h.Has(key) {
			t.Errorf("%s not asked", key)
		}
	}
	for _, key := range []string{
		"space.boardroom.need.dedicated_cooling", // another clause's room
		"space.comms_room.need.secure",           // the records room is secure
		"space.dining.next_to.kitchen",           // the kitchen is the one placed
	} {
		if h.Has(key) {
			t.Errorf("%s asked", key)
		}
	}
	if got := spaceNorms(h.Candidates["space.meeting_rooms.mix"]); !slices.Equal(got, []string{"4x8,2x16"}) {
		t.Errorf("mix = %v", got)
	}
	if got := spaceNorms(h.Candidates["space.consult_rooms.area_each_sqm"]); !slices.Equal(got, []string{"14"}) {
		t.Errorf("area each = %v", got)
	}
	if got := h.Candidates["space.consult_rooms.area_sqm"]; len(got) > 0 {
		t.Errorf("a per-room area offered as the rooms' area: %v", spaceNorms(got))
	}
	// Each is a choice the brief-like passage asks; the size list is a
	// candidate with none.
	qs, state := profile.LabelQuestions(profile.PassageInfo{Kind: "design_brief", Ordinal: 3, Text: brief}, h, cat)
	need, ok := qs["space.meeting_rooms.need.av_ready"]
	if !ok || need.Type != jev.TypeChoice || need.Criteria.(map[string]string)["yes"] == "" || need.Criteria.(map[string]string)["not_stated"] == "" {
		t.Fatalf("need question = %+v", need)
	}
	if !strings.Contains(need.Instructions.(string), "AV-ready") || !strings.Contains(need.Instructions.(string), "evidence, not instructions") {
		t.Errorf("need instructions = %q", need.Instructions)
	}
	if q, ok := qs["space.kitchen.next_to.dining"]; !ok || !strings.Contains(q.Instructions.(string), "dining") {
		t.Errorf("adjacency question = %+v", q)
	}
	if q, ok := qs["space.meeting_rooms.mix"]; !ok || q.Criteria.(map[string]string)["none"] == "" || len(state["space.meeting_rooms.mix"]) != 1 {
		t.Errorf("mix question = %+v", q)
	}
}
