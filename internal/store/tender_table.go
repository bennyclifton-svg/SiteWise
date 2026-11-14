package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
	"sitewise/internal/tender"
)

// TenderTable is the tender member of the price table (contract C5), with
// cents where the JSON has decimal strings.
type TenderTable struct {
	On          bool // the org has switched the model on
	Version     int64
	Rows        []TenderRow
	Columns     []TenderColumn
	Adjustments []TenderAdjustment
	Suggested   int
}

// TenderRow is one row of the first table.
type TenderRow struct {
	Key, Label string
	System     bool
}

// TenderCell is one firm's figure on one row.
type TenderCell struct {
	State   string
	Cents   *int64
	Lines   int
	Mark    string
	ToCheck bool
	Note    string
}

// TenderColumn is one firm's tender pass: its state while it is read, and
// its sum, cells and lines once it is.
type TenderColumn struct {
	SubmissionID string
	State, Error string
	Basis        string
	Sum          *int64
	SumHow       string
	SumBy        string
	Why          string
	LikeForLike  *int64
	// AppliedCents is the firm's applied adds and subs added up, over
	// AppliedRows rows.
	AppliedCents int64
	AppliedRows  int
	Lowest       bool
	Checked      bool
	CheckedAt    *time.Time
	Flags        []tender.Flag
	Notes        []string
	Exclusions   []tender.RawExclusion
	Cells        map[string]TenderCell
	Lines        []tender.Line
	// Gaps is how many rows carry a gap mark that no applied add answers.
	Gaps int
}

// TenderAdjustment is one row of the second table (filled from slice 5).
type TenderAdjustment struct {
	ID, Label, Note, Origin, Reason, Row string
	Amounts                              map[string]TenderAmount
	Others                               map[string]tender.Range
}

// TenderAmount is one firm's signed amount on an adjustment.
type TenderAmount struct {
	Cents  *int64
	State  string
	Source string
	LineID string
	Page   int
}

// tenderRowsOf is the first table's rows: the base rows, then the system row.
func tenderRowsOf(items []ScheduleItem) []TenderRow {
	var rows []TenderRow
	system := false
	for _, it := range items {
		if it.Retired || !it.Price() || !it.Base {
			continue
		}
		if it.Key == tender.SystemRow {
			system = true
			continue
		}
		rows = append(rows, TenderRow{Key: it.Key, Label: it.Label})
	}
	if system {
		rows = append(rows, TenderRow{Key: tender.SystemRow, Label: "Not broken down", System: true})
	}
	return rows
}

// tenderExtras is what the second table adds to a package's columns.
type tenderExtras struct {
	Adjustments []TenderAdjustment
	Suggested   int
}

// packageTender is every tender column of a package's live submissions with
// its marks, its like-for-like total and the package's adds and subs. It is
// shared by the table read and the board.
func packageTender(ctx context.Context, tx pgx.Tx, org, pkg string, items []ScheduleItem, subs []string) ([]TenderColumn, tenderExtras, error) {
	cols, _, err := packageTenderColumns(ctx, tx, org, items, subs)
	if err != nil {
		return nil, tenderExtras{}, err
	}
	adjustments, suggested, err := withAdjustments(ctx, tx, org, pkg, cols)
	return cols, tenderExtras{Adjustments: adjustments, Suggested: suggested}, err
}

// packageTenderColumns is the first table's columns: state, sum, cells,
// lines and marks, before any add or sub.
func packageTenderColumns(ctx context.Context, tx pgx.Tx, org string, items []ScheduleItem, subs []string) ([]TenderColumn, map[string]tender.Reading, error) {
	stored, err := readTenderStored(ctx, tx, org, subs)
	if err != nil {
		return nil, nil, err
	}
	var read []string
	for _, id := range subs {
		if stored[id].State == TenderRead {
			read = append(read, id)
		}
	}
	lines, err := tenderLines(ctx, tx, org, read, false)
	if err != nil {
		return nil, nil, err
	}
	rows := baseTenderRows(items)
	var cols []TenderColumn
	var marking []tender.Column
	readings := map[string]tender.Reading{}
	for _, id := range subs {
		st := stored[id]
		if st.State == "" {
			continue
		}
		col := TenderColumn{SubmissionID: id, State: st.State, Error: st.Error, Cells: map[string]TenderCell{}, Lines: []tender.Line{}, Flags: []tender.Flag{}, Notes: []string{}, Exclusions: []tender.RawExclusion{}}
		if st.State != TenderRead {
			cols = append(cols, col)
			continue
		}
		r := st.reading(lines[id])
		readings[id] = r
		cells := tender.Cells(r, rows)
		col.Basis, col.Sum, col.SumHow, col.SumBy, col.Why = st.Basis, st.Sum, st.SumHow, st.SumBy, r.Why
		col.Checked, col.CheckedAt = st.CheckedAt != nil, st.CheckedAt
		col.Flags, col.Lines = flagsOrEmpty(r.Flags), r.Lines
		if r.Notes != nil {
			col.Notes = r.Notes
		}
		if r.Exclusions != nil {
			col.Exclusions = r.Exclusions
		}
		if col.Lines == nil {
			col.Lines = []tender.Line{}
		}
		for key, c := range cells {
			col.Cells[key] = TenderCell{State: c.State, Cents: c.Cents, Lines: c.Lines, ToCheck: c.ToCheck, Note: c.Note}
		}
		if st.Sum != nil {
			v := *st.Sum
			col.LikeForLike = &v
		}
		cols = append(cols, col)
		marking = append(marking, tender.Column{ID: id, Sum: st.Sum, OnRows: tender.OnRows(r, rows), Cells: cells})
	}
	marks := tender.Marks(rows, marking)
	for i := range cols {
		for row, mark := range marks[cols[i].SubmissionID] {
			c := cols[i].Cells[row] // a silent gap has no cell of its own
			c.Mark = mark
			cols[i].Cells[row] = c
			if mark == tender.MarkGap {
				cols[i].Gaps++
			}
		}
	}
	return cols, readings, nil
}

// applyTender lays the tender columns over the firm columns: a tender-read
// column's totals are its tender sum, the lowest is marked among them, and
// it needs review until the manager has checked it.
func applyTender(cols []FirmColumn, totals []proposals.Column, tcols []TenderColumn) {
	by := map[string]*TenderColumn{}
	anyRead := false
	for i := range tcols {
		by[tcols[i].SubmissionID] = &tcols[i]
		anyRead = anyRead || tcols[i].State == TenderRead
	}
	if anyRead {
		// Old-style columns are not compared with tender-read ones.
		for i := range totals {
			totals[i].Lowest = false
		}
	}
	best := -1
	for i := range cols {
		t := by[cols[i].Submission.ID]
		if t == nil {
			continue
		}
		cols[i].Tender = t
		if t.State != TenderRead {
			cols[i].NeedsReview = true
			continue
		}
		cols[i].NeedsReview = !t.Checked
		// The old reading's source quote and builder review gave way to the
		// tender lines: the award and the screen work from the cells alone.
		cols[i].Quote, cols[i].AppointmentRows, cols[i].AwardStages, cols[i].AppointmentTotal, cols[i].AppointmentBasis, cols[i].Builder = nil, nil, nil, nil, "", nil
		totals[i] = proposals.Column{NotPriced: t.Gaps}
		if t.Sum != nil {
			v := *t.Sum
			totals[i].BaseTotal, totals[i].KnownBase = &v, v
		}
		if t.LikeForLike == nil || t.Gaps > 0 {
			continue
		}
		if best < 0 || *t.LikeForLike < *cols[best].Tender.LikeForLike {
			best = i
		}
	}
	if best >= 0 {
		totals[best].Lowest = true
		cols[best].Tender.Lowest = true
	}
}

func tenderVersion(ctx context.Context, q querier, org, pkg string) (int64, bool, error) {
	var version int64
	var on bool
	err := q.QueryRow(ctx, `SELECT p.tender_version, o.model_enabled_at IS NOT NULL FROM packages p JOIN orgs o ON o.id=p.org_id WHERE p.org_id=$1::uuid AND p.id::text=$2`, org, pkg).Scan(&version, &on)
	return version, on, err
}
