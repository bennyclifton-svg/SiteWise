package tender

// Column is one firm's cells for marking.
type Column struct {
	ID     string
	Sum    *int64          // tender sum
	OnRows int64           // counted dollars on real rows
	Cells  map[string]Cell // by row key
}

// MarkGap is the one cell mark. High and low marks were removed (owner,
// 15 Nov 2026): the manager reads the figures side by side.
const MarkGap = "gap"

// brokenDown: a firm whose lines put at least this share of its tender sum
// on rows has broken its price down, so a row it left empty is a gap and
// not just a lump sum (design section 9).
const brokenDown = 0.8

func priced(c Cell, ok bool) bool { return ok && c.State == StatePriced && c.Cents != nil }

// Marks returns, for each column id and row key, "gap": a row the firm
// excluded, did not price or left empty while at least one other firm, and
// at least half of the others, priced it. A priced cell and the system row
// are never marked.
func Marks(rows []Row, cols []Column) map[string]map[string]string {
	out := map[string]map[string]string{}
	set := func(id, row, mark string) {
		if out[id] == nil {
			out[id] = map[string]string{}
		}
		out[id][row] = mark
	}
	for _, row := range rows {
		if row.Key == SystemRow {
			continue
		}
		for i, col := range cols {
			var others []int64
			for j, o := range cols {
				if c, ok := o.Cells[row.Key]; j != i && priced(c, ok) {
					others = append(others, *c.Cents)
				}
			}
			cell, has := col.Cells[row.Key]
			if priced(cell, has) {
				continue
			}
			if len(others) == 0 || len(others)*2 < len(cols)-1 {
				continue
			}
			switch {
			case has && (cell.State == StateExcluded || cell.State == StateNotPriced):
				set(col.ID, row.Key, MarkGap)
			case !has && col.Sum != nil && *col.Sum > 0 && float64(col.OnRows) >= brokenDown*float64(*col.Sum):
				set(col.ID, row.Key, MarkGap)
			}
		}
	}
	return out
}

// OnRows is the counted dollars of a reading that sit on a known row.
func OnRows(r Reading, rows []Row) int64 {
	known := map[string]bool{}
	for _, row := range rows {
		known[row.Key] = true
	}
	var sum int64
	for _, l := range r.Lines {
		if l.Counted && l.Cents != nil && l.Row != "" && known[l.Row] {
			sum += *l.Cents
		}
	}
	return sum
}

// ExGST takes one eleventh off an amount that includes GST, to the cent.
func ExGST(inc int64) int64 { return exGST(inc) }

// UserTotalLabel names the line that holds a tender sum the manager set.
const UserTotalLabel = "Tender sum (set by you)"

// UserTenderSum is the tender sum the manager set, if any: the user's own
// total line. It is a line so that undo covers it.
func UserTenderSum(lines []Line) *int64 {
	for _, l := range lines {
		if l.Kind == "total" && l.DecidedBy == "user" && l.Cents != nil {
			c := *l.Cents
			return &c
		}
	}
	return nil
}

// Range is what the other firms priced a row at.
type Range struct {
	Low, High, Average int64
	Count              int
}
