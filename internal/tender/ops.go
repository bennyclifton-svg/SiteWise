package tender

import (
	"strings"

	"sitewise/internal/invoices"
)

// Op is one named edit to a submission's lines (contract C6). It never
// calls a model: every figure in it was typed by the manager.
type Op struct {
	Op         string  `json:"op"`
	Line       string  `json:"line,omitempty"`
	Submission string  `json:"submission,omitempty"`
	Row        *string `json:"row,omitempty"`
	Label      *string `json:"label,omitempty"`
	Amount     *string `json:"amount,omitempty"` // decimal, in the GST basis given or the line's own
	GST        *string `json:"gst,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	Placed     *string `json:"placed,omitempty"`
	State      *string `json:"state,omitempty"`
	Counted    *bool   `json:"counted,omitempty"`
	Page       int     `json:"page,omitempty"`
	Parts      []Part  `json:"parts,omitempty"`
	RestRow    *string `json:"rest_row,omitempty"`
	Slot       *int    `json:"slot,omitempty"`
}

// Part is one named share of a split. The rest goes to RestRow.
type Part struct {
	Row    string `json:"row"`
	Amount string `json:"amount"`
}

// OpError says which field of an operation is wrong, in plain words.
type OpError struct{ Field, Msg string }

func (e *OpError) Error() string { return e.Msg }

func opErr(field, msg string) error { return &OpError{Field: field, Msg: msg} }

// AddedNote marks a line the manager added. Only such a line, a figure
// they typed for a whole row, or their own tender sum can be deleted.
const AddedNote = "Added by you"

// RowFigureNote marks a line made from a figure the manager typed into a
// cell before the tender pass ran. It stands for its whole row.
const RowFigureNote = "Your figure for this row"

// LineOps are the operations that change a submission's lines.
var LineOps = map[string]bool{"move_line": true, "split_line": true, "unsplit_line": true, "set_line": true, "add_line": true,
	"count_line": true, "delete_line": true, "set_tender_sum": true, "set_slot": true}

// MaxSlot bounds a line's place within its row (the column's CHECK).
const MaxSlot = 999

func typedCents(amount, gst string) (int64, error) {
	c, ok, err := invoices.ParseCents(amount)
	if err != nil || !ok {
		return 0, opErr("amount", "Type an amount, such as 12500.00.")
	}
	if gst == "inc" {
		c = exGST(c)
	}
	return c, nil
}

func dropFlags(flags []Flag, codes ...string) []Flag {
	out := flags[:0:0]
	for _, f := range flags {
		keep := true
		for _, c := range codes {
			keep = keep && f.Code != c
		}
		if keep {
			out = append(out, f)
		}
	}
	return out
}

func validGST(g string) bool { return g == "ex" || g == "inc" || g == "unknown" }

// Apply applies one operation to one submission's lines and returns the
// lines as they then stand. rows holds the keys a line may sit on ("" is
// always allowed: it is the "Not broken down" row). It changes nothing
// when it returns an error: the caller's slice is never written to.
func Apply(lines []Line, rows map[string]bool, op Op, newID func() string) ([]Line, error) {
	out := make([]Line, len(lines))
	copy(out, lines)
	find := func() (int, error) {
		for i := range out {
			if out[i].ID == op.Line && op.Line != "" {
				return i, nil
			}
		}
		return -1, opErr("line", "That line is no longer there. Reload the table.")
	}
	rowOK := func(row string) bool { return row == "" || row == SystemRow || rows[row] }
	clean := func(row string) string {
		if row == SystemRow {
			return ""
		}
		return row
	}
	hasParts := func(id string) bool {
		for _, l := range out {
			if l.SplitOf == id {
				return true
			}
		}
		return false
	}
	maxOrdinal := 0
	for _, l := range out {
		maxOrdinal = max(maxOrdinal, l.Ordinal+1)
	}
	switch op.Op {
	case "move_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		if op.Row == nil || !rowOK(*op.Row) {
			return nil, opErr("row", "Choose a row of this schedule.")
		}
		// A line moved to another row takes the next free place there.
		out[i].Row, out[i].DecidedBy, out[i].Slot = clean(*op.Row), "user", nil
		out[i].Flags = dropFlags(out[i].Flags, FlagRowUnsure, FlagRowUnknown)
	case "set_slot":
		// Layout only: the line's place in its row. Its figures and who
		// decided them are untouched.
		i, err := find()
		if err != nil {
			return nil, err
		}
		if op.Slot == nil || *op.Slot < 0 || *op.Slot > MaxSlot {
			return nil, opErr("slot", "Choose a place in the row.")
		}
		v := *op.Slot
		out[i].Slot = &v
	case "split_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		l := out[i]
		switch {
		case l.SplitOf != "":
			return nil, opErr("line", "This is already part of a split. Put the line back together first.")
		case !l.Counted || l.Cents == nil:
			return nil, opErr("line", "Only a counted line can be split.")
		case len(op.Parts) == 0 || len(op.Parts) > 20:
			return nil, opErr("parts", "Give at least one part.")
		case op.RestRow == nil || !rowOK(*op.RestRow):
			return nil, opErr("rest_row", "Choose the row the rest goes to.")
		}
		rest := *l.Cents
		var parts []Line
		part := func(row string, cents int64) {
			c := cents
			p := Line{ID: newID(), Ordinal: l.Ordinal, Page: l.Page, Label: l.Label + " (part)", Cents: &c, GST: l.GST, State: StatePriced, Kind: l.Kind,
				Placed: l.Placed, Heading: l.Heading, Row: clean(row), Counted: true, DecidedBy: "user", SplitOf: l.ID}
			parts = append(parts, p)
		}
		for _, p := range op.Parts {
			if !rowOK(p.Row) {
				return nil, opErr("parts", "Choose a row of this schedule for each part.")
			}
			// A part is typed in the line's own GST basis.
			c, err := typedCents(p.Amount, l.GST)
			if err != nil || c <= 0 {
				return nil, opErr("parts", "Type an amount for each part.")
			}
			rest -= c
			part(p.Row, c)
		}
		if rest <= 0 {
			return nil, opErr("parts", "The parts add to as much as the line or more. Leave something for the rest.")
		}
		// The rest is what is left, so the parts always add to the line exactly.
		part(*op.RestRow, rest)
		out[i].Counted, out[i].DecidedBy = false, "user"
		out = append(out[:i+1], append(parts, out[i+1:]...)...)
	case "unsplit_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		if !hasParts(out[i].ID) {
			return nil, opErr("line", "This line is not split.")
		}
		id := out[i].ID
		out[i].Counted, out[i].DecidedBy = out[i].Cents != nil, "user"
		kept := out[:0]
		for _, l := range out {
			if l.SplitOf != id {
				kept = append(kept, l)
			}
		}
		out = kept
	case "set_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		l := &out[i]
		if op.Label == nil && op.Amount == nil && op.GST == nil && op.Kind == nil && op.Placed == nil && op.State == nil {
			return nil, opErr("", "Nothing to change.")
		}
		if (op.Amount != nil || op.GST != nil || op.State != nil || op.Kind != nil) && hasParts(l.ID) {
			return nil, opErr("line", "This line is split. Put it back together first.")
		}
		if op.Label != nil {
			label := strings.TrimSpace(*op.Label)
			if label == "" || len(label) > 500 {
				return nil, opErr("label", "Give the line a name.")
			}
			l.Label = label
		}
		if op.GST != nil {
			if !validGST(*op.GST) {
				return nil, opErr("gst", "GST is ex, inc or unknown.")
			}
			if op.Amount == nil && l.Cents != nil {
				// The same printed figure on another basis.
				if printed, ok, _ := parsePrinted(l.Printed); ok {
					c := printed
					if *op.GST == "inc" {
						c = exGST(c)
					}
					l.Cents = &c
				}
			}
			l.GST = *op.GST
		}
		if op.Amount != nil {
			c, err := typedCents(*op.Amount, l.GST)
			if err != nil {
				return nil, err
			}
			hadNone := l.Cents == nil
			l.Cents, l.Printed, l.State = &c, strings.TrimSpace(*op.Amount), StatePriced
			l.Flags = dropFlags(l.Flags, FlagAmountNotFound, FlagFromScan)
			if hadNone && scope(l.Kind) && l.Placed == "sum" && l.SplitOf == "" {
				l.Counted = true
			}
		}
		if op.Placed != nil {
			if *op.Placed != "sum" && *op.Placed != "inside" && *op.Placed != "outside" {
				return nil, opErr("placed", "Choose in the price, inside another line, or outside the price.")
			}
			l.Placed = *op.Placed
			if l.Placed != "sum" {
				l.Counted = false
			}
		}
		if op.Kind != nil {
			switch *op.Kind {
			case "base", "ps", "pc":
				l.Kind = *op.Kind
			case "option", "saving", "rate":
				l.Kind, l.Placed, l.Counted = *op.Kind, "outside", false
				if l.Kind == "saving" && l.Cents != nil && *l.Cents > 0 {
					c := -*l.Cents
					l.Cents = &c
				}
			default:
				return nil, opErr("kind", "Choose what kind of money this is.")
			}
		}
		if op.State != nil {
			switch *op.State {
			case StatePriced:
				if l.Cents == nil {
					return nil, opErr("state", "Type the amount to make this a priced line.")
				}
				l.State = StatePriced
			case StateIncluded, StateExcluded, StateNotPriced:
				l.State, l.Cents, l.Printed, l.Counted = *op.State, nil, "", false
			default:
				return nil, opErr("state", "Choose priced, included, excluded or not priced.")
			}
		}
		l.DecidedBy = "user"
	case "add_line":
		label := ""
		if op.Label != nil {
			label = strings.TrimSpace(*op.Label)
		}
		if label == "" || len(label) > 500 {
			return nil, opErr("label", "Give the line a name.")
		}
		if op.Amount == nil {
			return nil, opErr("amount", "Type an amount, such as 12500.00.")
		}
		gst := "ex"
		if op.GST != nil {
			gst = *op.GST
		}
		if !validGST(gst) {
			return nil, opErr("gst", "GST is ex, inc or unknown.")
		}
		row := ""
		if op.Row != nil {
			row = *op.Row
		}
		if !rowOK(row) {
			return nil, opErr("row", "Choose a row of this schedule.")
		}
		c, err := typedCents(*op.Amount, gst)
		if err != nil {
			return nil, err
		}
		out = append(out, Line{ID: newID(), Ordinal: maxOrdinal, Page: max(op.Page, 0), Label: label, Printed: strings.TrimSpace(*op.Amount), Cents: &c, GST: gst,
			State: StatePriced, Kind: "base", Placed: "sum", Row: clean(row), Note: AddedNote, Counted: true, DecidedBy: "user"})
	case "count_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		l := &out[i]
		if op.Counted == nil {
			return nil, opErr("counted", "Say whether the line is counted.")
		}
		if *op.Counted {
			switch {
			case l.Cents == nil:
				return nil, opErr("line", "Type the amount before counting this line.")
			case !scope(l.Kind):
				return nil, opErr("line", "Only a priced line of scope can be counted.")
			case hasParts(l.ID):
				return nil, opErr("line", "This line is split. Put it back together first.")
			}
			l.State, l.Placed = StatePriced, "sum"
			l.Flags = dropFlags(l.Flags, FlagBreakdown, FlagRepeat)
		}
		l.Counted, l.DecidedBy = *op.Counted, "user"
	case "delete_line":
		i, err := find()
		if err != nil {
			return nil, err
		}
		l := out[i]
		own := l.DecidedBy == "user" && (l.Note == AddedNote || l.Note == RowFigureNote || l.Kind == "total")
		if !own {
			return nil, opErr("line", "Only a line you added can be deleted. Take it out of the count instead.")
		}
		if hasParts(l.ID) {
			return nil, opErr("line", "This line is split. Put it back together first.")
		}
		out = append(out[:i], out[i+1:]...)
	case "set_tender_sum":
		// The manager's sum is their own total line, so undo covers it. It
		// never changes a line of scope.
		kept := out[:0]
		for _, l := range out {
			if !(l.Kind == "total" && l.DecidedBy == "user") {
				kept = append(kept, l)
			}
		}
		out = kept
		if op.Amount == nil || strings.TrimSpace(*op.Amount) == "" {
			break // cleared: the sum the reading chose stands again
		}
		gst := "ex"
		if op.GST != nil {
			gst = *op.GST
		}
		if !validGST(gst) {
			return nil, opErr("gst", "GST is ex, inc or unknown.")
		}
		c, err := typedCents(*op.Amount, gst)
		if err != nil {
			return nil, err
		}
		if c <= 0 {
			return nil, opErr("amount", "The tender sum is more than zero.")
		}
		out = append(out, Line{ID: newID(), Ordinal: maxOrdinal, Label: UserTotalLabel, Printed: strings.TrimSpace(*op.Amount), Cents: &c, GST: gst, State: StatePriced,
			Kind: "total", Placed: "inside", DecidedBy: "user"})
	default:
		return nil, opErr("op", "Unknown operation "+op.Op+".")
	}
	return out, nil
}
