package tender

import "testing"

// Every firm, wording and figure in these tests is invented.

var alignRows = []Row{{Key: "site", Label: "Siteworks"}, {Key: "roof", Label: "Roofing"}, {Key: SystemRow, Label: "Not broken down"}}

func al(id, row, label string) Line {
	c := int64(100)
	return Line{ID: id, Row: row, Label: label, Kind: "base", State: StatePriced, Cents: &c, Counted: true}
}

func slotted(l Line, s int) Line { l.Slot = &s; return l }

func TestLinesSharingWordsShareASlot(t *testing.T) {
	a := []Line{al("a1", "site", "Bulk excavation and cart away"), al("a2", "site", "Retaining wall to rear boundary"), al("a3", "site", "Concrete plinth for pizza oven")}
	b := []Line{al("b1", "site", "Pizza oven base"), al("b2", "site", "Retaining walls"), al("b3", "site", "Landscaping")}
	s := Slots(alignRows, [][]Line{a, b})
	if s["b2"] != s["a2"] || s["b1"] != s["a3"] {
		t.Fatalf("like lines apart: %v", s)
	}
	// The first firm keeps its own order; a line with no match takes a new place below.
	if !(s["a1"] < s["a2"] && s["a2"] < s["a3"]) || s["b3"] <= s["a3"] {
		t.Fatalf("order: %v", s)
	}
}

func TestEachFirmHasOneLinePerSlot(t *testing.T) {
	a := []Line{al("a1", "site", "Retaining wall north")}
	b := []Line{al("b1", "site", "Retaining wall east"), al("b2", "site", "Retaining wall west")}
	s := Slots(alignRows, [][]Line{a, b})
	if s["b1"] == s["b2"] {
		t.Fatalf("two lines of one firm in one slot: %v", s)
	}
}

func TestRowsAreAlignedApart(t *testing.T) {
	a := []Line{al("a1", "site", "Retaining wall"), al("a2", "roof", "Metal roof sheeting")}
	b := []Line{al("b1", "roof", "Roof sheeting")}
	s := Slots(alignRows, [][]Line{a, b})
	if s["a2"] != 0 || s["b1"] != 0 || s["a1"] != 0 {
		t.Fatalf("each row starts at its own first slot: %v", s)
	}
}

func TestTheManagersSlotsWin(t *testing.T) {
	// One placed line freezes the row: the rest keep their saved places and
	// a line without one goes below them, never matched by words.
	a := []Line{slotted(al("a1", "site", "Retaining wall"), 2), slotted(al("a2", "site", "Excavation"), 0)}
	b := []Line{slotted(al("b1", "site", "Retaining walls"), 0), al("b2", "site", "Retaining wall extra")}
	s := Slots(alignRows, [][]Line{a, b})
	if s["a1"] != 2 || s["a2"] != 0 || s["b1"] != 0 || s["b2"] != 3 {
		t.Fatalf("%v", s)
	}
}

func TestLinesOutOfTheCountGoBelow(t *testing.T) {
	out := al("a2", "site", "Retaining wall")
	out.Counted = false
	a := []Line{out, al("a1", "site", "Excavation")}
	b := []Line{al("b1", "site", "Retaining wall")}
	s := Slots(alignRows, [][]Line{a, b})
	if s["a1"] != 0 || s["a2"] <= s["b1"] {
		t.Fatalf("%v", s)
	}
}

func TestLinesOnNoRowAlignInNotBrokenDown(t *testing.T) {
	a := []Line{al("a1", "", "Site establishment"), al("a2", "gone", "Builder's margin")}
	b := []Line{al("b1", "", "Margin, builder's")}
	sub := al("a3", "", "Subtotal")
	sub.Kind = "subtotal"
	s := Slots(alignRows, [][]Line{append(a, sub), b})
	if s["a2"] != s["b1"] {
		t.Fatalf("%v", s)
	}
	if _, ok := s["a3"]; ok {
		t.Fatal("a subtotal is not shown, so it has no slot")
	}
}
