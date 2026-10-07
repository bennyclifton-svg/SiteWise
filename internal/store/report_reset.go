package store

import (
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"sitewise/internal/reports"
)

func (s *Store) ResetReportEdit(ctx context.Context, org, id, actor, target string, version int64) (ReportDraft, error) {
	var d ReportDraft
	if version < 1 || target == "" || len(target) > 500 || !utf8.ValidString(target) {
		return d, ErrInvalidReport
	}
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
	r, err = readReport(ctx, tx, org, id)
	if err != nil {
		return d, err
	}
	if r.CurrentDraftVersionID == "" {
		return d, ErrInvalidReport
	}
	d, err = readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return d, err
	}
	if d.Status != "draft" {
		return d, ErrInvalidReport
	}
	if d.Version != version {
		return d, ErrVersionConflict
	}
	sections, err := reports.ResetEdit(d.Sections, target)
	if errors.Is(err, reports.ErrProtectedEditNotFound) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, ErrInvalidReport
	}
	sections, refs, err := reports.Cite(sections)
	if err != nil {
		return d, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM report_edits WHERE org_id=$1::uuid AND report_version_id=$2::uuid AND target_id=$3`, org, d.ID, target)
	if err != nil {
		return d, err
	}
	if tag.RowsAffected() != 1 {
		return d, ErrNotFound
	}
	raw, err := json.Marshal(sections)
	if err != nil {
		return d, err
	}
	var saved []byte
	err = tx.QueryRow(ctx, `UPDATE report_versions SET sections=$3::jsonb,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING to_jsonb(report_versions)`, org, d.ID, raw).Scan(&saved)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(saved, &d); err != nil {
		return d, err
	}
	if err := saveReportReferences(ctx, tx, org, d.ID, refs); err != nil {
		return d, err
	}
	d.References = refs
	d.MaterialAssumptions = reports.MaterialAssumptions(refs)
	return d, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}
