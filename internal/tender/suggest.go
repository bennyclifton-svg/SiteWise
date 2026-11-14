package tender

import (
	"sort"
	"strings"
)

// SuggestColumn is one tender-read firm as Suggest needs it.
type SuggestColumn struct {
	ID    string // submission id
	Lines []Line
	Cells map[string]Cell
	Marks map[string]string // by row key
}

// Amount is one firm's figure on a suggestion.
type Amount struct {
	Cents  *int64 // signed; nil for a gap with no figure yet
	Source string // "firm" or "none"
	LineID string
	Page   int
}

// Suggestion is one row the system proposes for the second table. Nothing
// here is applied: the manager applies.
type Suggestion struct {
	Key     string
	Label   string
	Reason  string // gap | allowance | option | saving
	Row     string
	Amounts map[string]Amount // by submission id
}

// Reasons of an adjustment, in the order the second table lists them.
const (
	ReasonGap       = "gap"
	ReasonAllowance = "allowance"
	ReasonOption    = "option"
	ReasonSaving    = "saving"
	ReasonTakenOut  = "taken_out"
	ReasonOther     = "other"
)

var reasonOrder = map[string]int{ReasonGap: 0, ReasonAllowance: 1, ReasonOption: 2, ReasonSaving: 3}

func prefixed(prefix, label string) string {
	if strings.HasPrefix(strings.ToLower(label), strings.ToLower(strings.TrimSuffix(prefix, ": "))) {
		return label
	}
	return prefix + label
}

// Suggest lists what a manager would want to consider when levelling: each
// option, saving and extra a firm prints, at the firm's own figure, and
// each row a firm left out that others priced, with no figure. The system
// never invents a figure: every amount here is a line's own cents.
// optionRows maps a schedule option row's key to its label.
func Suggest(rows []Row, optionRows map[string]string, cols []SuggestColumn) []Suggestion {
	type placed struct {
		s          *Suggestion
		page, ord  int
		reason, at int
	}
	byKey := map[string]*placed{}
	var list []*placed
	get := func(key, label, reason, row string, page, ord, at int) *Suggestion {
		if p, ok := byKey[key]; ok {
			return p.s
		}
		p := &placed{s: &Suggestion{Key: key, Label: label, Reason: reason, Row: row, Amounts: map[string]Amount{}}, page: page, ord: ord, reason: reasonOrder[reason], at: at}
		byKey[key] = p
		list = append(list, p)
		return p.s
	}
	rowAt := map[string]int{}
	rowLabel := map[string]string{}
	for i, r := range rows {
		rowAt[r.Key], rowLabel[r.Key] = i, r.Label
	}
	for _, col := range cols {
		for _, r := range rows {
			if col.Marks[r.Key] == MarkGap {
				s := get("gap:"+r.Key, "Allowance for "+r.Label, ReasonGap, r.Key, 0, 0, rowAt[r.Key])
				s.Amounts[col.ID] = Amount{Source: "none"}
			}
		}
		for _, l := range col.Lines {
			if l.Counted || l.Cents == nil || l.State != StatePriced {
				continue
			}
			var s *Suggestion
			switch {
			case l.Kind == "option" && optionRows[l.Row] != "":
				s = get("option:"+l.Row, optionRows[l.Row], ReasonOption, "", l.Page, l.Ordinal, 0)
			case l.Kind == "option":
				s = get("line:"+l.ID, prefixed("Option: ", l.Label), ReasonOption, "", l.Page, l.Ordinal, 0)
			case l.Kind == "saving":
				s = get("line:"+l.ID, prefixed("Saving: ", l.Label), ReasonSaving, "", l.Page, l.Ordinal, 0)
			case allowance(l.Kind) && l.Placed == "outside":
				s = get("line:"+l.ID, "Extra to the price: "+l.Label, ReasonAllowance, "", l.Page, l.Ordinal, 0)
			default:
				continue
			}
			c := *l.Cents
			if l.Kind == "saving" && c > 0 {
				c = -c
			}
			if had, ok := s.Amounts[col.ID]; ok && had.Cents != nil {
				c += *had.Cents // a firm's several lines on one option row
				s.Amounts[col.ID] = Amount{Cents: &c, Source: "firm", LineID: had.LineID, Page: had.Page}
				continue
			}
			s.Amounts[col.ID] = Amount{Cents: &c, Source: "firm", LineID: l.ID, Page: l.Page}
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch {
		case a.reason != b.reason:
			return a.reason < b.reason
		case a.at != b.at:
			return a.at < b.at
		case a.page != b.page:
			return a.page < b.page
		}
		return a.ord < b.ord
	})
	out := make([]Suggestion, len(list))
	for i, p := range list {
		out[i] = *p.s
	}
	return out
}

// Others is what the firms other than id priced a row at. The average
// rounds half a cent up.
func Others(cols []SuggestColumn, id, row string) Range {
	var r Range
	var sum int64
	for _, col := range cols {
		c, ok := col.Cells[row]
		if col.ID == id || !priced(c, ok) {
			continue
		}
		v := *c.Cents
		if r.Count == 0 || v < r.Low {
			r.Low = v
		}
		if r.Count == 0 || v > r.High {
			r.High = v
		}
		sum += v
		r.Count++
	}
	if r.Count > 0 {
		r.Average = (2*sum + int64(r.Count)) / (2 * int64(r.Count))
	}
	return r
}
