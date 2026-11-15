package tender

import "testing"

// Every figure in these tests is invented.

func pc(dollars int64) Cell {
	c := dollars * 100
	return Cell{State: StatePriced, Cents: &c, Lines: 1}
}

func col(id string, sum, onRows int64, cells map[string]Cell) Column {
	s := sum * 100
	return Column{ID: id, Sum: &s, OnRows: onRows * 100, Cells: cells}
}

var markRows = []Row{{Key: "a", Label: "A"}, {Key: "b", Label: "B"}, {Key: SystemRow, Label: "Not broken down"}}

// A priced cell is never marked high or low, however far it sits from the
// others (owner, 15 Nov 2026): the manager reads the figures side by side.
func TestPricedCellsAreNotMarked(t *testing.T) {
	for _, outlier := range []int64{300, 40} {
		m := Marks(markRows, []Column{col("1", 100, 100, map[string]Cell{"a": pc(100)}), col("2", 110, 110, map[string]Cell{"a": pc(110)}), col("3", outlier, outlier, map[string]Cell{"a": pc(outlier)})})
		if len(m) != 0 {
			t.Fatalf("outlier %d marked: %v", outlier, m)
		}
	}
}

func TestExplicitGap(t *testing.T) {
	cols := []Column{col("1", 100, 100, map[string]Cell{"a": pc(100)}), col("2", 110, 110, map[string]Cell{"a": pc(110)}),
		col("3", 50, 0, map[string]Cell{"a": {State: StateExcluded}})}
	if m := Marks(markRows, cols); m["3"]["a"] != MarkGap {
		t.Fatalf("%v", m)
	}
	cols[2].Cells["a"] = Cell{State: StateIncluded}
	if m := Marks(markRows, cols); m["3"]["a"] != "" {
		t.Fatalf("included is not a gap: %v", m)
	}
}

func TestSilentGapNeedsABreakdown(t *testing.T) {
	others := []Column{col("1", 200, 200, map[string]Cell{"a": pc(100), "b": pc(100)}), col("2", 220, 220, map[string]Cell{"a": pc(110), "b": pc(110)})}
	broken := col("3", 100, 90, map[string]Cell{"b": pc(90)})
	if m := Marks(markRows, append(others, broken)); m["3"]["a"] != MarkGap {
		t.Fatalf("%v", m)
	}
	lump := col("3", 100, 20, map[string]Cell{"b": pc(20)})
	if m := Marks(markRows, append(others, lump)); m["3"]["a"] != "" {
		t.Fatalf("a lump sum has no gaps: %v", m)
	}
}

func TestGapNeedsHalfTheOthers(t *testing.T) {
	cols := []Column{col("1", 100, 100, map[string]Cell{"a": pc(100)})}
	for _, id := range []string{"2", "3", "4"} {
		cols = append(cols, col(id, 100, 100, map[string]Cell{"b": pc(100)}))
	}
	cols = append(cols, col("5", 100, 100, map[string]Cell{"a": {State: StateExcluded}, "b": pc(100)}))
	if m := Marks(markRows, cols); m["5"]["a"] != "" {
		t.Fatalf("one of four others priced it: %v", m)
	}
}

func TestTheSystemRowIsNeverMarked(t *testing.T) {
	cols := []Column{col("1", 100, 100, map[string]Cell{SystemRow: pc(100)}), col("2", 110, 110, map[string]Cell{SystemRow: pc(110)}), col("3", 900, 900, map[string]Cell{SystemRow: pc(900)})}
	if m := Marks(markRows, cols); len(m) != 0 {
		t.Fatalf("%v", m)
	}
}

func TestUserTenderSum(t *testing.T) {
	c := int64(500)
	lines := []Line{{Kind: "total", DecidedBy: "model", Cents: &c}, {Kind: "total", DecidedBy: "user", Cents: &c, Label: UserTotalLabel}}
	if got := UserTenderSum(lines); got == nil || *got != 500 {
		t.Fatalf("%v", got)
	}
	if UserTenderSum(lines[:1]) != nil {
		t.Fatal("a model's total is not the user's")
	}
}
