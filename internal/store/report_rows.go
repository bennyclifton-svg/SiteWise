package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/reports"
)

// The PM's rows in the project report's tables (project report plan,
// sections 6 and 7). A row write lays the rows over the draft's saved
// generated sections in place; it does not run the loader. Every query is
// scoped to the org and the report.

// ErrReportRow is a row write the report cannot take (an unknown table or
// column, a row of another report, words too long).
var ErrReportRow = errors.New("invalid report row")

const reportCellMax = 4000

// RowWrite is one change to the report's tables. Table and Key name a
// generated row; Row names a report_rows row; neither is the PM's new row.
type RowWrite struct {
	Table string `json:"table"`
	Key   string `json:"key"`
	Row   string `json:"row"`
	// Cells: column index to words; null takes SiteWise's back.
	Cells map[string]*string `json:"cells"`
	// Keep: columns whose SiteWise value has moved, where the PM keeps
	// their words against the new value.
	Keep   []string `json:"keep"`
	Hidden *bool    `json:"hidden"`
	// Move is "up" or "down".
	Move   string `json:"move"`
	Source string `json:"source"`
	// After places a new row of the PM's own after this row key or id.
	After string `json:"after"`
}

func readReportRows(ctx context.Context, q querier, org, report string) ([]reports.ReportRow, error) {
	rows, err := q.Query(ctx, `SELECT id::text, table_id, COALESCE(source_key,''), position, cells, hidden, COALESCE(sources->>0,''), created_at, version
FROM report_rows WHERE org_id=$1::uuid AND report_id=$2::uuid ORDER BY created_at, id`, org, report)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []reports.ReportRow
	for rows.Next() {
		var r reports.ReportRow
		var cells []byte
		if err := rows.Scan(&r.ID, &r.TableID, &r.SourceKey, &r.Position, &cells, &r.Hidden, &r.Source, &r.CreatedAt, &r.Version); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cells, &r.Cells); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// lastIssuedAt is when the report was last issued; zero for none.
func lastIssuedAt(ctx context.Context, q querier, org, report string) (time.Time, error) {
	var at *time.Time
	err := q.QueryRow(ctx, `SELECT max(issued_at) FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND status='issued'`, org, report).Scan(&at)
	if err != nil || at == nil {
		return time.Time{}, err
	}
	return *at, nil
}

// layDraftRows lays the PM's rows and today's edits over a draft's saved
// generated sections, marks what is new against the last issue, and saves
// the result as the draft's sections.
func (s *Store) layDraftRows(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) (ReportDraft, error) {
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT generated FROM report_versions WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, d.ID).Scan(&raw); err != nil {
		return d, err
	}
	if raw == nil {
		return d, ErrInvalidReport
	}
	var generated []reports.Section
	if err := json.Unmarshal(raw, &generated); err != nil {
		return d, err
	}
	edits, err := readReportEdits(ctx, tx, org, d.ID)
	if err != nil {
		return d, err
	}
	sections, err := reports.ReapplyEdits(generated, edits)
	if err != nil {
		return d, err
	}
	sections, err = s.overlayRows(ctx, tx, org, r, sections)
	if err != nil {
		return d, err
	}
	out, _ := json.Marshal(sections)
	var saved []byte
	if err := tx.QueryRow(ctx, `UPDATE report_versions SET sections=$3::jsonb,updated_at=now(),version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING to_jsonb(report_versions)`, org, d.ID, out).Scan(&saved); err != nil {
		return d, err
	}
	// A fresh value: decoding over the old draft would keep fields the new
	// one omits (a cleared flag) in reused slice elements.
	var fresh ReportDraft
	if err := json.Unmarshal(saved, &fresh); err != nil {
		return d, err
	}
	d = fresh
	d.References, err = readReportReferences(ctx, tx, org, d.ID)
	d.MaterialAssumptions = reports.MaterialAssumptions(d.References)
	return d, err
}

// overlayRows puts the PM's rows on sections and marks new rows against
// the last issue.
func (s *Store) overlayRows(ctx context.Context, tx pgx.Tx, org string, r Report, sections []reports.Section) ([]reports.Section, error) {
	rows, err := readReportRows(ctx, tx, org, r.ID)
	if err != nil {
		return nil, err
	}
	at, err := lastIssuedAt(ctx, tx, org, r.ID)
	if err != nil {
		return nil, err
	}
	sections = reports.ApplyRows(sections, rows, at)
	_, prior, err := reportHistory(ctx, tx, org, r.ID)
	if err != nil {
		return nil, err
	}
	reports.MarkNew(sections, prior)
	return sections, nil
}

// WriteReportRow makes one change to the project report's tables and
// returns the draft with it laid in. version is the draft the screen saw.
func (s *Store) WriteReportRow(ctx context.Context, org, id, actor string, version int64, w RowWrite) (ReportDraft, string, error) {
	var d ReportDraft
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return d, "", err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return d, "", err
	}
	if err := lockProject(ctx, tx, org, r.ProjectID); err != nil {
		return d, "", err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return d, "", err
	}
	if r.Kind != reports.KindMonthly || r.CurrentDraftVersionID == "" {
		return d, "", ErrInvalidReport
	}
	if d, err = readReportDraft(ctx, tx, org, r.CurrentDraftVersionID); err != nil {
		return d, "", err
	}
	if d.Status != "draft" || version != 0 && d.Version != version {
		return d, "", ErrVersionConflict
	}
	table, index, err := findRow(d.Sections, w)
	if err != nil {
		return d, "", err
	}
	rowID, err := s.writeRow(ctx, tx, org, r, actor, table, index, w)
	if err != nil {
		return d, "", err
	}
	if d, err = s.layDraftRows(ctx, tx, org, r, d); err != nil {
		return d, "", err
	}
	return d, rowID, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}

// findRow finds the table a write names, and the row's index in it (-1 for
// a new row of the PM's own).
func findRow(sections []reports.Section, w RowWrite) (reports.Table, int, error) {
	for _, s := range sections {
		for _, b := range s.Blocks {
			t := b.Table
			if t == nil || t.ID == "" || t.Meta == nil {
				continue
			}
			if w.Table != "" && t.ID != w.Table {
				continue
			}
			for i, f := range t.Meta {
				if w.Row != "" && f.ID == w.Row || w.Row == "" && w.Key != "" && f.Key == w.Key {
					return *t, i, nil
				}
			}
			if w.Table != "" && w.Row == "" && w.Key == "" {
				return *t, -1, nil
			}
		}
	}
	return reports.Table{}, 0, ErrNotFound
}

func (s *Store) writeRow(ctx context.Context, tx pgx.Tx, org string, r Report, actor string, t reports.Table, index int, w RowWrite) (string, error) {
	width := len(t.Columns)
	for col, text := range w.Cells {
		n, err := strconv.Atoi(col)
		if err != nil || n < 0 || n >= width || text != nil && len(*text) > reportCellMax {
			return "", ErrReportRow
		}
	}
	if len(w.Source) > 100 || w.Move != "" && w.Move != "up" && w.Move != "down" {
		return "", ErrReportRow
	}
	// A new row of the PM's own.
	if index < 0 {
		cells := map[string]reports.RowCell{}
		for col, text := range w.Cells {
			if text != nil {
				cells[col] = reports.RowCell{Text: strings.TrimSpace(*text)}
			}
		}
		var pos *float64
		if w.After != "" {
			for i, f := range t.Meta {
				if f.Key == w.After || f.ID == w.After {
					p := f.Pos + 0.5
					if i+1 < len(t.Meta) {
						p = (f.Pos + t.Meta[i+1].Pos) / 2
					}
					pos = &p
				}
			}
		}
		raw, _ := json.Marshal(cells)
		id := newID()
		_, err := tx.Exec(ctx, `INSERT INTO report_rows(org_id,id,project_id,report_id,table_id,position,cells,sources,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::jsonb,$8::jsonb,$9::uuid)`,
			org, id, r.ProjectID, r.ID, t.ID, pos, raw, sourcesOf(w.Source), actor)
		return id, err
	}
	f := t.Meta[index]
	// The row the write lands on: the PM's own, or one laid over a generated row.
	var row reports.ReportRow
	var cellsRaw []byte
	err := tx.QueryRow(ctx, `SELECT id::text, COALESCE(source_key,''), position, cells, hidden FROM report_rows WHERE org_id=$1::uuid AND report_id=$2::uuid AND table_id=$3
 AND (id::text=$4 OR ($4='' AND source_key=$5)) FOR UPDATE`, org, r.ID, t.ID, f.ID, f.Key).Scan(&row.ID, &row.SourceKey, &row.Position, &cellsRaw, &row.Hidden)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	row.Cells = map[string]reports.RowCell{}
	if exists {
		if err := json.Unmarshal(cellsRaw, &row.Cells); err != nil {
			return "", err
		}
	} else {
		row.ID, row.SourceKey = newID(), f.Key
		if f.Kind == reports.RowOwn && f.Key == "" {
			return "", ErrNotFound
		}
	}
	base := func(col int) string {
		if col < len(f.Base) && col < len(f.Typed) && f.Typed[col] {
			return f.Base[col]
		}
		return t.Rows[index][col]
	}
	for col, text := range w.Cells {
		n, _ := strconv.Atoi(col)
		if text == nil {
			delete(row.Cells, col)
			continue
		}
		cell := reports.RowCell{Text: strings.TrimSpace(*text)}
		if row.SourceKey != "" {
			cell.Base = reports.CellHash(base(n))
		}
		row.Cells[col] = cell
	}
	for _, col := range w.Keep {
		n, err := strconv.Atoi(col)
		if c, ok := row.Cells[col]; ok && err == nil && n < width {
			c.Base = reports.CellHash(base(n))
			row.Cells[col] = c
		}
	}
	if w.Hidden != nil {
		row.Hidden = *w.Hidden
	}
	if w.Move != "" {
		dir := -1
		if w.Move == "down" {
			dir = 1
		}
		p, ok := reports.MovePosition(t, index, dir)
		if !ok {
			return "", ErrReportRow
		}
		row.Position = &p
	}
	// A row of the PM's own with nothing left in it, or a generated row with
	// nothing over it, is removed.
	if exists && len(row.Cells) == 0 && !row.Hidden && row.Position == nil && row.SourceKey != "" {
		_, err := tx.Exec(ctx, `DELETE FROM report_rows WHERE org_id=$1::uuid AND id=$2::uuid`, org, row.ID)
		return row.ID, err
	}
	raw, _ := json.Marshal(row.Cells)
	if exists {
		_, err = tx.Exec(ctx, `UPDATE report_rows SET cells=$3::jsonb,hidden=$4,position=$5,sources=CASE WHEN $6='' THEN sources ELSE $7::jsonb END,updated_at=now(),version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`,
			org, row.ID, raw, row.Hidden, row.Position, w.Source, sourcesOf(w.Source))
		return row.ID, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO report_rows(org_id,id,project_id,report_id,table_id,source_key,position,cells,hidden,sources,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8::jsonb,$9,$10::jsonb,$11::uuid)`,
		org, row.ID, r.ProjectID, r.ID, t.ID, row.SourceKey, row.Position, raw, row.Hidden, sourcesOf(w.Source), actor)
	return row.ID, err
}

func sourcesOf(doc string) string {
	if doc == "" {
		return "[]"
	}
	raw, _ := json.Marshal([]string{doc})
	return string(raw)
}

// DeleteReportRow removes a row of the PM's own, or gives a generated row
// back to SiteWise (its words, its place and its hidden mark go).
func (s *Store) DeleteReportRow(ctx context.Context, org, id, actor, row string, version int64) (ReportDraft, error) {
	var d ReportDraft
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return d, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return d, err
	}
	if err := lockProject(ctx, tx, org, r.ProjectID); err != nil {
		return d, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return d, err
	}
	if r.Kind != reports.KindMonthly || r.CurrentDraftVersionID == "" || !validUUID(row) {
		return d, ErrNotFound
	}
	if d, err = readReportDraft(ctx, tx, org, r.CurrentDraftVersionID); err != nil {
		return d, err
	}
	if version != 0 && d.Version != version {
		return d, ErrVersionConflict
	}
	tag, err := tx.Exec(ctx, `DELETE FROM report_rows WHERE org_id=$1::uuid AND report_id=$2::uuid AND id=$3::uuid`, org, r.ID, row)
	if err != nil {
		return d, err
	}
	if tag.RowsAffected() != 1 {
		return d, ErrNotFound
	}
	if d, err = s.layDraftRows(ctx, tx, org, r, d); err != nil {
		return d, err
	}
	return d, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}

// ReportPeriod is the window a draft reports on.
type ReportPeriod struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

// SetReportPeriod types the draft's period. It must not start before the
// prior period ends, nor end before it starts. The draft is assembled
// again so every "this period" reads the new window.
func (s *Store) SetReportPeriod(ctx context.Context, org, id, actor, appBuild string, version int64, p ReportPeriod) (ReportDraft, error) {
	var d ReportDraft
	p.From, p.To, p.Label = strings.TrimSpace(p.From), strings.TrimSpace(p.To), strings.TrimSpace(p.Label)
	for _, v := range []string{p.From, p.To} {
		if v == "" {
			continue
		}
		if t, err := time.Parse("2006-01-02", v); err != nil || t.Year() < 2000 || t.Year() > 2099 {
			return d, ErrReportRow
		}
	}
	if p.To == "" || p.From != "" && p.From > p.To || len(p.Label) > 80 {
		return d, ErrReportRow
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return d, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return d, err
	}
	if err := lockProject(ctx, tx, org, r.ProjectID); err != nil {
		return d, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return d, err
	}
	if r.Kind != reports.KindMonthly || r.CurrentDraftVersionID == "" {
		return d, ErrInvalidReport
	}
	if d, err = readReportDraft(ctx, tx, org, r.CurrentDraftVersionID); err != nil {
		return d, err
	}
	if version != 0 && d.Version != version {
		return d, ErrVersionConflict
	}
	var priorTo *string
	if err := tx.QueryRow(ctx, `SELECT period_to::text FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND status='issued' ORDER BY number DESC LIMIT 1`, org, r.ID).Scan(&priorTo); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return d, err
	}
	if priorTo != nil && p.From != "" && p.From <= *priorTo {
		return d, ErrReportRow
	}
	if p.Label == "" {
		p.Label = reports.PeriodLabel(p.From, p.To)
	}
	if _, err := tx.Exec(ctx, `UPDATE report_versions SET period_from=NULLIF($3,'')::date, period_to=$4::date, period_label=$5 WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, d.ID, p.From, p.To, p.Label); err != nil {
		return d, err
	}
	if d, err = s.assembleDraftTx(ctx, tx, org, r, appBuild, true); err != nil {
		return d, err
	}
	return d, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}
