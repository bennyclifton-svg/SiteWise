package httpapi

import (
	"time"

	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// The tender member of the price table (contract C5 of
// docs/plans/2026-11-14-tender-comparison-plan.md). Money is an exact
// decimal string or null; the browser never adds.

type tenderJSON struct {
	On          bool                   `json:"on"`
	Version     int64                  `json:"version"`
	Rows        []tenderRowJSON        `json:"rows"`
	Columns     []tenderColumnJSON     `json:"columns"`
	Adjustments []tenderAdjustmentJSON `json:"adjustments"`
	Suggested   int                    `json:"suggested"`
}

type tenderRowJSON struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	System bool   `json:"system"`
}

type tenderCellJSON struct {
	State   string     `json:"state"`
	Amount  moneyField `json:"amount"`
	Lines   int        `json:"lines"`
	Mark    string     `json:"mark"`
	ToCheck bool       `json:"to_check"`
	Note    string     `json:"note,omitempty"`
}

type tenderLineJSON struct {
	ID        string        `json:"id"`
	Ordinal   int           `json:"ordinal"`
	Page      int           `json:"page"`
	Label     string        `json:"label"`
	Printed   string        `json:"printed"`
	Amount    moneyField    `json:"amount"`
	GST       string        `json:"gst"`
	State     string        `json:"state"`
	Kind      string        `json:"kind"`
	Placed    string        `json:"placed"`
	Heading   string        `json:"heading"`
	Row       string        `json:"row"`
	Note      string        `json:"note"`
	Counted   bool          `json:"counted"`
	Flags     []tender.Flag `json:"flags"`
	DecidedBy string        `json:"decided_by"`
	SplitOf   string        `json:"split_of"`
}

type tenderColumnJSON struct {
	SubmissionID string                    `json:"submission_id"`
	State        string                    `json:"state"`
	Error        string                    `json:"error"`
	Basis        string                    `json:"basis"`
	TenderSum    moneyField                `json:"tender_sum"`
	TenderSumHow string                    `json:"tender_sum_how"`
	TenderSumBy  string                    `json:"tender_sum_by"`
	RemainderWhy string                    `json:"remainder_why"`
	LikeForLike  moneyField                `json:"like_for_like"`
	Lowest       bool                      `json:"lowest"`
	Checked      bool                      `json:"checked"`
	CheckedAt    *time.Time                `json:"checked_at"`
	Flags        []tender.Flag             `json:"flags"`
	Notes        []string                  `json:"notes"`
	Exclusions   []tender.RawExclusion     `json:"exclusions"`
	Cells        map[string]tenderCellJSON `json:"cells"`
	Lines        []tenderLineJSON          `json:"lines"`
}

type tenderAmountJSON struct {
	Amount moneyField `json:"amount"`
	State  string     `json:"state"`
	Source string     `json:"source"`
	LineID string     `json:"line_id"`
	Page   int        `json:"page"`
}

type tenderRangeJSON struct {
	Low     moneyField `json:"low"`
	High    moneyField `json:"high"`
	Average moneyField `json:"average"`
	Count   int        `json:"count"`
}

type tenderAdjustmentJSON struct {
	ID      string                      `json:"id"`
	Label   string                      `json:"label"`
	Note    string                      `json:"note"`
	Origin  string                      `json:"origin"`
	Reason  string                      `json:"reason"`
	Row     string                      `json:"row"`
	Amounts map[string]tenderAmountJSON `json:"amounts"`
	Others  map[string]tenderRangeJSON  `json:"others"`
}

// tenderBody writes the tender tables. available is whether this server
// has a model key: with none, "on" is false whatever the org chose.
func tenderBody(t *store.TenderTable, available bool) *tenderJSON {
	if t == nil {
		return nil
	}
	out := &tenderJSON{On: t.On && available, Version: t.Version, Rows: []tenderRowJSON{}, Columns: []tenderColumnJSON{}, Adjustments: []tenderAdjustmentJSON{}, Suggested: t.Suggested}
	for _, r := range t.Rows {
		out.Rows = append(out.Rows, tenderRowJSON{Key: r.Key, Label: r.Label, System: r.System})
	}
	for _, c := range t.Columns {
		col := tenderColumnJSON{SubmissionID: c.SubmissionID, State: c.State, Error: c.Error, Basis: c.Basis, TenderSum: cents(c.Sum), TenderSumHow: c.SumHow, TenderSumBy: c.SumBy,
			RemainderWhy: c.Why, LikeForLike: cents(c.LikeForLike), Lowest: c.Lowest, Checked: c.Checked, CheckedAt: c.CheckedAt, Flags: c.Flags, Notes: c.Notes, Exclusions: c.Exclusions,
			Cells: map[string]tenderCellJSON{}, Lines: []tenderLineJSON{}}
		for key, cell := range c.Cells {
			col.Cells[key] = tenderCellJSON{State: cell.State, Amount: cents(cell.Cents), Lines: cell.Lines, Mark: cell.Mark, ToCheck: cell.ToCheck, Note: cell.Note}
		}
		for _, l := range c.Lines {
			flags := l.Flags
			if flags == nil {
				flags = []tender.Flag{}
			}
			col.Lines = append(col.Lines, tenderLineJSON{ID: l.ID, Ordinal: l.Ordinal, Page: l.Page, Label: l.Label, Printed: l.Printed, Amount: cents(l.Cents), GST: l.GST, State: l.State, Kind: l.Kind,
				Placed: l.Placed, Heading: l.Heading, Row: l.Row, Note: l.Note, Counted: l.Counted, Flags: flags, DecidedBy: l.DecidedBy, SplitOf: l.SplitOf})
		}
		out.Columns = append(out.Columns, col)
	}
	for _, a := range t.Adjustments {
		adj := tenderAdjustmentJSON{ID: a.ID, Label: a.Label, Note: a.Note, Origin: a.Origin, Reason: a.Reason, Row: a.Row, Amounts: map[string]tenderAmountJSON{}, Others: map[string]tenderRangeJSON{}}
		for sub, m := range a.Amounts {
			adj.Amounts[sub] = tenderAmountJSON{Amount: cents(m.Cents), State: m.State, Source: m.Source, LineID: m.LineID, Page: m.Page}
		}
		for sub, o := range a.Others {
			adj.Others[sub] = tenderRangeJSON{Low: centsValue(o.Low), High: centsValue(o.High), Average: centsValue(o.Average), Count: o.Count}
		}
		out.Adjustments = append(out.Adjustments, adj)
	}
	return out
}
