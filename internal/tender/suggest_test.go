package tender

import "testing"

// Every figure in these tests is invented.

var sugRows = []Row{{Key: "a", Label: "Substructure"}, {Key: "b", Label: "Structure"}}

func outLine(id, kind, label string, cents int64, row string) Line {
	return Line{ID: id, Label: label, Cents: cp(cents), State: StatePriced, Kind: kind, Placed: "outside", Row: row, Page: 34, Ordinal: 7}
}

func TestOptionLineIsSuggestedAtTheFirmsFigure(t *testing.T) {
	cols := []SuggestColumn{{ID: "A", Lines: []Line{outLine("o1", "option", "epoxy grout", 1800000, "")}}, {ID: "B"}}
	got := Suggest(sugRows, nil, cols)
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	s := got[0]
	a := s.Amounts["A"]
	if s.Key != "line:o1" || s.Label != "Option: epoxy grout" || s.Reason != ReasonOption || *a.Cents != 1800000 || a.Source != "firm" || a.LineID != "o1" || a.Page != 34 {
		t.Fatalf("%+v %+v", s, a)
	}
	if _, has := s.Amounts["B"]; has {
		t.Fatal("firm B has no figure on firm A's option")
	}
	already := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: []Line{outLine("o2", "option", "Option: epoxy grout", 100, "")}}})
	if already[0].Label != "Option: epoxy grout" {
		t.Fatalf("label %q", already[0].Label)
	}
}

func TestOptionOnAScheduleOptionRowIsShared(t *testing.T) {
	cols := []SuggestColumn{
		{ID: "A", Lines: []Line{outLine("o1", "option", "Extra meetings", 100000, "option:x")}},
		{ID: "B", Lines: []Line{outLine("o2", "option", "Additional meeting", 150000, "option:x")}},
	}
	got := Suggest(sugRows, map[string]string{"option:x": "Extra design meetings"}, cols)
	if len(got) != 1 || got[0].Key != "option:option:x" || got[0].Label != "Extra design meetings" || *got[0].Amounts["A"].Cents != 100000 || *got[0].Amounts["B"].Cents != 150000 {
		t.Fatalf("%+v", got)
	}
}

func TestSavingIsNegative(t *testing.T) {
	got := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: []Line{outLine("s1", "saving", "laminate tops", 650000, "")}}})
	if len(got) != 1 || *got[0].Amounts["A"].Cents != -650000 || got[0].Reason != ReasonSaving || got[0].Label != "Saving: laminate tops" {
		t.Fatalf("%+v", got)
	}
}

func TestAllowanceOutsideThePrice(t *testing.T) {
	ps := outLine("p1", "ps", "Asbestos removal", 1500000, "a")
	got := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: []Line{ps}}})
	if len(got) != 1 || got[0].Label != "Extra to the price: Asbestos removal" || got[0].Reason != ReasonAllowance {
		t.Fatalf("%+v", got)
	}
	ps.Counted, ps.Flags = true, []Flag{{Code: FlagCountedToMatch}}
	if got := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: []Line{ps}}}); len(got) != 0 {
		t.Fatalf("a counted allowance was suggested: %+v", got)
	}
}

func TestGapSuggestsARowWithNoFigureAndIsShared(t *testing.T) {
	cols := []SuggestColumn{{ID: "A"}, {ID: "B", Marks: map[string]string{"b": MarkGap}}, {ID: "C"}, {ID: "D", Marks: map[string]string{"b": MarkGap, "a": "high"}}}
	got := Suggest(sugRows, nil, cols)
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	s := got[0]
	if s.Key != "gap:b" || s.Label != "Allowance for Structure" || s.Reason != ReasonGap || s.Row != "b" || len(s.Amounts) != 2 {
		t.Fatalf("%+v", s)
	}
	if b := s.Amounts["B"]; b.Cents != nil || b.Source != "none" {
		t.Fatalf("%+v", b)
	}
}

func TestRatesSubtotalsAndCountedLinesSuggestNothing(t *testing.T) {
	counted := outLine("c1", "base", "Works", 100, "a")
	counted.Counted, counted.Placed = true, "sum"
	inside := outLine("i1", "ps", "Allowance within tiling", 100, "a")
	inside.Placed = "inside"
	none := Line{ID: "n1", Kind: "option", State: StateNotPriced, Label: "Option with no price"}
	lines := []Line{counted, inside, none, outLine("r1", "rate", "Director", 35000, ""), outLine("t1", "subtotal", "Total options", 500, "")}
	if got := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: lines}}); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestOrderIsGapsThenAllowancesThenOptionsThenSavings(t *testing.T) {
	early := outLine("o0", "option", "early option", 1, "")
	early.Page = 2
	lines := []Line{outLine("s1", "saving", "saving", 1, ""), outLine("o1", "option", "late option", 1, ""), early, outLine("p1", "ps", "extra", 1, "")}
	got := Suggest(sugRows, nil, []SuggestColumn{{ID: "A", Lines: lines, Marks: map[string]string{"b": MarkGap, "a": MarkGap}}, {ID: "B"}})
	var keys []string
	for _, s := range got {
		keys = append(keys, s.Key)
	}
	want := []string{"gap:a", "gap:b", "line:p1", "line:o0", "line:o1", "line:s1"}
	if len(keys) != len(want) {
		t.Fatalf("%v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("order %v, want %v", keys, want)
		}
	}
}

func TestOthersRange(t *testing.T) {
	cell := func(c int64) map[string]Cell { return map[string]Cell{"a": {State: StatePriced, Cents: cp(c)}} }
	cols := []SuggestColumn{{ID: "B"}, {ID: "A", Cells: cell(100)}, {ID: "C", Cells: cell(300)}, {ID: "D", Cells: cell(500)}, {ID: "E", Cells: map[string]Cell{"a": {State: StateExcluded}}}}
	if r := Others(cols, "B", "a"); r.Low != 100 || r.High != 500 || r.Average != 300 || r.Count != 3 {
		t.Fatalf("%+v", r)
	}
	odd := []SuggestColumn{{ID: "B"}, {ID: "A", Cells: cell(100)}, {ID: "C", Cells: cell(101)}}
	if r := Others(odd, "B", "a"); r.Average != 101 {
		t.Fatalf("half a cent rounds up: %+v", r)
	}
	if r := Others(cols[:1], "B", "a"); r.Count != 0 {
		t.Fatalf("%+v", r)
	}
}
