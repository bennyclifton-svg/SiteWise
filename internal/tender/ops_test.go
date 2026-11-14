package tender

import (
	"errors"
	"strconv"
	"testing"
)

// Every figure in these tests is invented.

var opRows = map[string]bool{"a": true, "b": true}

func cp(v int64) *int64   { return &v }
func sp(v string) *string { return &v }
func bp(v bool) *bool     { return &v }

func opLine(id string, cents int64, row string) Line {
	return Line{ID: id, Label: "Line " + id, Printed: dollars(cents), Cents: cp(cents), GST: "ex", State: StatePriced, Kind: "base", Placed: "sum", Row: row,
		Page: 3, Heading: "Pack 1", Counted: true, DecidedBy: "model"}
}

func ids() func() string {
	n := 0
	return func() string { n++; return "new" + strconv.Itoa(n) }
}

func apply(t *testing.T, lines []Line, op Op) []Line {
	t.Helper()
	out, err := Apply(lines, opRows, op, ids())
	if err != nil {
		t.Fatalf("%s: %v", op.Op, err)
	}
	return out
}

func refused(t *testing.T, lines []Line, op Op, field string) {
	t.Helper()
	before := append([]Line{}, lines...)
	_, err := Apply(lines, opRows, op, ids())
	var oe *OpError
	if !errors.As(err, &oe) || oe.Msg == "" || (field != "" && oe.Field != field) {
		t.Fatalf("%s: want a refusal on %q, got %v", op.Op, field, err)
	}
	for i := range before {
		if before[i].Row != lines[i].Row || before[i].Counted != lines[i].Counted || before[i].Label != lines[i].Label {
			t.Fatalf("%s changed the caller's lines on error", op.Op)
		}
	}
}

func TestMoveLine(t *testing.T) {
	l := opLine("1", 100000, "a")
	l.Flags = []Flag{{Code: FlagRowUnsure}, {Code: FlagFromScan}}
	out := apply(t, []Line{l}, Op{Op: "move_line", Line: "1", Row: sp("b")})
	if out[0].Row != "b" || out[0].DecidedBy != "user" || hasFlag(out[0].Flags, FlagRowUnsure) || !hasFlag(out[0].Flags, FlagFromScan) {
		t.Fatalf("%+v", out[0])
	}
	if out := apply(t, []Line{l}, Op{Op: "move_line", Line: "1", Row: sp("")}); out[0].Row != "" {
		t.Fatal("no row is the system row")
	}
	if out := apply(t, []Line{l}, Op{Op: "move_line", Line: "1", Row: sp(SystemRow)}); out[0].Row != "" {
		t.Fatal("the system row's key is stored as no row")
	}
	refused(t, []Line{l}, Op{Op: "move_line", Line: "1", Row: sp("nowhere")}, "row")
	refused(t, []Line{l}, Op{Op: "move_line", Line: "gone", Row: sp("b")}, "line")
}

func TestSplitAndUnsplit(t *testing.T) {
	lines := []Line{opLine("1", 10000000, "a"), opLine("2", 500, "a")}
	out := apply(t, lines, Op{Op: "split_line", Line: "1", Parts: []Part{{Row: "a", Amount: "60000.00"}}, RestRow: sp("b")})
	if len(out) != 4 || out[0].Counted || out[0].DecidedBy != "user" {
		t.Fatalf("%+v", out)
	}
	a, b := out[1], out[2]
	if *a.Cents != 6000000 || a.Row != "a" || *b.Cents != 4000000 || b.Row != "b" || !a.Counted || !b.Counted || a.SplitOf != "1" || b.SplitOf != "1" ||
		a.Label != "Line 1 (part)" || a.Page != 3 || a.Heading != "Pack 1" || a.GST != "ex" || a.ID == b.ID {
		t.Fatalf("parts %+v %+v", a, b)
	}
	refused(t, lines, Op{Op: "split_line", Line: "1", Parts: []Part{{Row: "a", Amount: "120000.00"}}, RestRow: sp("b")}, "parts")
	refused(t, lines, Op{Op: "split_line", Line: "1", Parts: []Part{{Row: "a", Amount: "100000.00"}}, RestRow: sp("b")}, "parts")
	refused(t, out, Op{Op: "split_line", Line: a.ID, Parts: []Part{{Row: "a", Amount: "1.00"}}, RestRow: sp("b")}, "line")
	refused(t, out, Op{Op: "split_line", Line: "1", Parts: []Part{{Row: "a", Amount: "1.00"}}, RestRow: sp("b")}, "line")
	refused(t, out, Op{Op: "set_line", Line: "1", Amount: sp("5.00")}, "line")
	back := apply(t, out, Op{Op: "unsplit_line", Line: "1"})
	if len(back) != 2 || !back[0].Counted || *back[0].Cents != 10000000 {
		t.Fatalf("%+v", back)
	}
	refused(t, back, Op{Op: "unsplit_line", Line: "1"}, "line")
}

func TestSplitOfAnIncGSTLineKeepsItsBasis(t *testing.T) {
	l := opLine("1", 10000, "a") // printed $110.00 including GST
	l.GST, l.Printed = "inc", "$110.00"
	out := apply(t, []Line{l}, Op{Op: "split_line", Line: "1", Parts: []Part{{Row: "a", Amount: "55.00"}}, RestRow: sp("b")})
	if *out[1].Cents != 5000 || *out[2].Cents != 5000 || *out[1].Cents+*out[2].Cents != *l.Cents {
		t.Fatalf("%d + %d", *out[1].Cents, *out[2].Cents)
	}
	odd := opLine("2", 10001, "a")
	odd.GST = "inc"
	out = apply(t, []Line{odd}, Op{Op: "split_line", Line: "2", Parts: []Part{{Row: "a", Amount: "33.33"}}, RestRow: sp("b")})
	if *out[1].Cents+*out[2].Cents != 10001 {
		t.Fatal("the parts do not add to the line")
	}
}

func TestSetLine(t *testing.T) {
	l := opLine("1", 100000, "a")
	l.GST = "inc"
	out := apply(t, []Line{l}, Op{Op: "set_line", Line: "1", Amount: sp("1100.00")})
	if *out[0].Cents != 100000 || out[0].Printed != "1100.00" || out[0].DecidedBy != "user" {
		t.Fatalf("%+v", out[0])
	}
	lost := Line{ID: "2", Label: "Lost", GST: "ex", State: StatePriced, Kind: "base", Placed: "sum", Row: "a", Flags: []Flag{{Code: FlagAmountNotFound}}, DecidedBy: "model"}
	out = apply(t, []Line{lost}, Op{Op: "set_line", Line: "2", Amount: sp("2,500")})
	if *out[0].Cents != 250000 || hasFlag(out[0].Flags, FlagAmountNotFound) || !out[0].Counted {
		t.Fatalf("%+v", out[0])
	}
	if out := apply(t, []Line{l}, Op{Op: "set_line", Line: "1", Label: sp("  Renamed ")}); out[0].Label != "Renamed" {
		t.Fatal("label")
	}
	out = apply(t, []Line{l}, Op{Op: "set_line", Line: "1", Kind: sp("option")})
	if out[0].Kind != "option" || out[0].Counted || out[0].Placed != "outside" {
		t.Fatalf("%+v", out[0])
	}
	out = apply(t, []Line{l}, Op{Op: "set_line", Line: "1", State: sp(StateIncluded)})
	if out[0].State != StateIncluded || out[0].Cents != nil || out[0].Counted {
		t.Fatalf("%+v", out[0])
	}
	ex := opLine("3", 11000, "a")
	ex.Printed = "$110.00"
	if out := apply(t, []Line{ex}, Op{Op: "set_line", Line: "3", GST: sp("inc")}); *out[0].Cents != 10000 || out[0].GST != "inc" {
		t.Fatalf("%+v", out[0])
	}
	refused(t, []Line{l}, Op{Op: "set_line", Line: "1"}, "")
	refused(t, []Line{l}, Op{Op: "set_line", Line: "1", Amount: sp("lots")}, "amount")
	refused(t, []Line{l}, Op{Op: "set_line", Line: "1", Kind: sp("total")}, "kind")
}

func TestAddAndDeleteLine(t *testing.T) {
	lines := []Line{opLine("1", 100000, "a")}
	out := apply(t, lines, Op{Op: "add_line", Label: sp("Site shed"), Amount: sp("5000"), Row: sp("b"), Page: 2})
	added := out[1]
	if !added.Counted || added.DecidedBy != "user" || *added.Cents != 500000 || added.Row != "b" || added.Kind != "base" || added.Ordinal <= out[0].Ordinal-1 || added.Note != AddedNote {
		t.Fatalf("%+v", added)
	}
	refused(t, lines, Op{Op: "add_line", Label: sp(""), Amount: sp("5")}, "label")
	refused(t, lines, Op{Op: "add_line", Label: sp("Shed")}, "amount")
	refused(t, lines, Op{Op: "add_line", Label: sp("Shed"), Amount: sp("5"), Row: sp("zz")}, "row")
	refused(t, out, Op{Op: "delete_line", Line: "1"}, "line")
	if out := apply(t, out, Op{Op: "delete_line", Line: added.ID}); len(out) != 1 {
		t.Fatal("the added line was not deleted")
	}
}

func TestCountLine(t *testing.T) {
	l := opLine("1", 100000, "a")
	l.Flags = []Flag{{Code: FlagBreakdown}}
	out := apply(t, []Line{l}, Op{Op: "count_line", Line: "1", Counted: bp(false)})
	if out[0].Counted || out[0].DecidedBy != "user" {
		t.Fatalf("%+v", out[0])
	}
	out = apply(t, out, Op{Op: "count_line", Line: "1", Counted: bp(true)})
	if !out[0].Counted || hasFlag(out[0].Flags, FlagBreakdown) {
		t.Fatalf("%+v", out[0])
	}
	none := Line{ID: "2", Label: "No amount", State: StatePriced, Kind: "base", Placed: "sum"}
	refused(t, []Line{none}, Op{Op: "count_line", Line: "2", Counted: bp(true)}, "line")
	for _, kind := range []string{"subtotal", "rate", "total", "gst"} {
		k := opLine("3", 100, "a")
		k.Kind, k.Counted = kind, false
		refused(t, []Line{k}, Op{Op: "count_line", Line: "3", Counted: bp(true)}, "line")
	}
}

func TestSetTenderSum(t *testing.T) {
	lines := []Line{opLine("1", 100000, "a")}
	if UserTenderSum(lines) != nil {
		t.Fatal("no user sum yet")
	}
	out := apply(t, lines, Op{Op: "set_tender_sum", Amount: sp("1100.00"), GST: sp("inc")})
	total := out[len(out)-1]
	if total.Kind != "total" || total.DecidedBy != "user" || total.Counted || total.Label != UserTotalLabel || total.Row != "" || *UserTenderSum(out) != 100000 {
		t.Fatalf("%+v", total)
	}
	if *out[0].Cents != 100000 || !out[0].Counted {
		t.Fatal("setting the sum changed a line of scope")
	}
	out = apply(t, out, Op{Op: "set_tender_sum", Amount: sp("2000")})
	if len(out) != 2 || *UserTenderSum(out) != 200000 {
		t.Fatalf("a second sum did not replace the first: %+v", out)
	}
	if out := apply(t, out, Op{Op: "set_tender_sum", Amount: sp("")}); UserTenderSum(out) != nil {
		t.Fatal("an empty amount clears the user's sum")
	}
	refused(t, lines, Op{Op: "set_tender_sum", Amount: sp("0")}, "amount")
	refused(t, lines, Op{Op: "fly"}, "op")
}
