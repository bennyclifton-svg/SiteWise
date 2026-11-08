package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

// ReportExport is a report's current draft as it would issue: internal
// review lists left out and the budget withheld unless the issue form
// discloses it, which a draft download never does.
type ReportExport struct {
	Title    string
	Project  string
	Date     string
	Sections []reports.Section
}

// ReadReportExport reads what the Word and PDF downloads print. It does
// not call Jev and writes nothing.
func (s *Store) ReadReportExport(ctx context.Context, org, id string) (ReportExport, error) {
	var out ReportExport
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return out, err
	}
	if r.CurrentDraftVersionID == "" {
		return out, ErrInvalidReport
	}
	d, err := readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return out, err
	}
	if err := tx.QueryRow(ctx, `SELECT name FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ProjectID).Scan(&out.Project); err != nil {
		return out, err
	}
	if out.Sections, _, err = reports.IssueContent(d.Sections, r.Kind, false); err != nil {
		return out, err
	}
	out.Title, out.Date = r.Title, d.ReportingDate
	return out, tx.Commit(ctx)
}
