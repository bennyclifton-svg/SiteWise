package tender

// Cell is one firm's figure on one row of the first table.
type Cell struct {
	State   string // priced | included | excluded | not_priced
	Cents   *int64 // nil when priced with no known amount, or not priced
	Lines   int    // counted lines behind it
	ToCheck bool   // a line in it has a flag
	Note    string // for included, excluded or not priced: the firm's words
}

// Cells is R9: one reading as a column of the first table. The system row
// holds the remainder, so the priced cells always add up to the tender sum.
func Cells(r Reading, rows []Row) map[string]Cell {
	known := map[string]bool{}
	for _, row := range rows {
		known[row.Key] = true
	}
	out := map[string]Cell{}
	sums := map[string]int64{}
	sysLines, sysCheck := 0, false
	for _, l := range r.Lines {
		if !l.Counted || l.Cents == nil {
			continue
		}
		if l.Row == "" || !known[l.Row] {
			sysLines++
			sysCheck = sysCheck || len(l.Flags) > 0
			continue
		}
		c := out[l.Row]
		sums[l.Row] += *l.Cents
		c.State, c.Lines, c.ToCheck = StatePriced, c.Lines+1, c.ToCheck || len(l.Flags) > 0
		out[l.Row] = c
	}
	for key, s := range sums {
		c := out[key]
		v := s
		c.Cents = &v
		out[key] = c
	}
	// A row with no counted line shows what the firm said about it. A priced
	// line whose amount was thrown away outranks the rest: it needs typing.
	rank := map[string]int{StatePriced: 4, StateIncluded: 3, StateExcluded: 2, StateNotPriced: 1}
	for _, l := range r.Lines {
		if l.Counted || l.Row == "" || !known[l.Row] || !scope(l.Kind) || (l.State == StatePriced && l.Cents != nil) {
			continue
		}
		if c, ok := out[l.Row]; !ok || (c.Lines == 0 && rank[l.State] > rank[c.State]) {
			out[l.Row] = Cell{State: l.State, Note: l.Note, ToCheck: l.State == StatePriced}
		}
	}
	if r.TenderSum != nil && (r.Remainder != 0 || sysLines > 0) {
		rem := r.Remainder
		out[SystemRow] = Cell{State: StatePriced, Cents: &rem, Lines: sysLines, ToCheck: sysCheck || r.Why == "differs"}
	}
	return out
}

// Recount refreshes the gap, the remainder and the column flags after the
// manager has edited lines or the tender sum. It moves and uncounts nothing.
func Recount(r *Reading) {
	keep := r.Flags[:0]
	for _, f := range r.Flags {
		switch f.Code {
		case FlagLinesShort, FlagLinesOver, FlagNotFixedPrice, FlagGSTUnknown, FlagOptionsOffered, FlagNoTotalPrinted:
			// finish writes these again from the lines as they now stand.
		default:
			keep = append(keep, f)
		}
	}
	r.Flags = keep
	finish(r)
}
