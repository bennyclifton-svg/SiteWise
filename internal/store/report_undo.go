package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

// touchReportsAfterUndo shows an undone report comment: every draft whose
// protected edits the change wrote is re-rendered from the edits now saved,
// in the undo's transaction (monthly plan, Task 10). An issued version is
// never touched: its edits are frozen, and the trigger refuses the undo.
func (s *Store) touchReportsAfterUndo(ctx context.Context, tx pgx.Tx, org, project string, log []changeRow) error {
	versions := map[string]bool{}
	reportsTouched := map[string]bool{}
	for _, r := range log {
		if r.Table == "report_rows" {
			image := r.New
			if image == nil {
				image = r.Old
			}
			var row struct {
				Report string `json:"report_id"`
			}
			if err := json.Unmarshal(image, &row); err == nil && row.Report != "" {
				reportsTouched[row.Report] = true
			}
			continue
		}
		if r.Table != "report_edits" {
			continue
		}
		image := r.New
		if image == nil {
			image = r.Old
		}
		var row struct {
			Version string `json:"report_version_id"`
		}
		if err := json.Unmarshal(image, &row); err == nil && row.Version != "" {
			versions[row.Version] = true
		}
	}
	for id := range versions {
		var reportID string
		err := tx.QueryRow(ctx, `SELECT report_id::text FROM report_versions WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND status='draft'`, org, project, id).Scan(&reportID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		d, err := readReportDraft(ctx, tx, org, id)
		if err != nil {
			return err
		}
		edits, err := readReportEdits(ctx, tx, org, id)
		if err != nil {
			return err
		}
		sections, err := reports.ReapplyEdits(d.Sections, edits)
		if err != nil {
			return err
		}
		sections, refs, err := reports.Cite(sections)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(sections)
		if _, err := tx.Exec(ctx, `UPDATE report_versions SET sections=$3::jsonb,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, id, raw); err != nil {
			return err
		}
		if err := saveReportReferences(ctx, tx, org, id, refs); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"project_id": project, "report_id": reportID, "version_id": id, "state": "ready"})
		if _, err := appendEvent(ctx, s.q.WithTx(tx), org, "report", "", string(payload)); err != nil {
			return err
		}
	}
	// The PM's rows: each report's current draft has them laid again.
	for id := range reportsTouched {
		rep, err := readReport(ctx, tx, org, id)
		if errors.Is(err, ErrNotFound) || err == nil && rep.CurrentDraftVersionID == "" {
			continue
		}
		if err != nil {
			return err
		}
		d, err := readReportDraft(ctx, tx, org, rep.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if _, err := s.layDraftRows(ctx, tx, org, rep, d); err != nil {
			return err
		}
	}
	if len(versions) == 0 && len(reportsTouched) == 0 {
		return nil
	}
	return bumpRevisionLocked(ctx, tx, org, project, "reports")
}
