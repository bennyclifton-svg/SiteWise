package store

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

// Version is the draft version the user saw, protecting both concurrent edits
// and a refresh that changed the generated source beneath the editor.
func (s *Store) EditReport(ctx context.Context, org, id, actor, target, text string, version int64) (ReportDraft, error) {
	var d ReportDraft
	if version < 1 || target == "" || len(target) > 500 || !utf8.ValidString(target) || !utf8.ValidString(text) || utf8.RuneCountInString(text) > 20000 {
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
	// Re-read after acquiring the lock: a concurrent refresh may have moved the
	// current draft while this writer waited.
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
	found := false
	var base string
	for i := range d.Sections {
		for j := range d.Sections[i].Blocks {
			b := &d.Sections[i].Blocks[j]
			if b.ID != target {
				continue
			}
			if b.SourceMissing || b.ContentSHA256 == "" {
				return d, ErrInvalidReport
			}
			found = true
			base = b.ContentSHA256
			if !b.Edited {
				original := b.Text
				b.GeneratedText = &original
			}
			b.Text = text
			b.Edited = true
			b.EditUserID = actor
			b.Conflict = false
		}
	}
	if !found {
		return d, ErrNotFound
	}
	var editTime time.Time
	err = tx.QueryRow(ctx, `INSERT INTO report_edits(org_id,report_version_id,target_id,text,base_content_sha256,user_id) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::uuid) ON CONFLICT(org_id,report_version_id,target_id) DO UPDATE SET text=EXCLUDED.text,base_content_sha256=EXCLUDED.base_content_sha256,user_id=EXCLUDED.user_id,version=report_edits.version+1,updated_at=now() RETURNING updated_at`, org, d.ID, target, text, base, actor).Scan(&editTime)
	if err != nil {
		return d, err
	}
	for i := range d.Sections {
		for j := range d.Sections[i].Blocks {
			if d.Sections[i].Blocks[j].ID == target {
				d.Sections[i].Blocks[j].EditUpdatedAt = editTime
			}
		}
	}
	sectionsWithCitations, refs, err := reports.Cite(d.Sections)
	if err != nil {
		return d, err
	}
	sections, _ := json.Marshal(sectionsWithCitations)
	var raw []byte
	err = tx.QueryRow(ctx, `UPDATE report_versions SET sections=$3::jsonb,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING to_jsonb(report_versions)`, org, d.ID, sections).Scan(&raw)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, err
	}
	if err := saveReportReferences(ctx, tx, org, d.ID, refs); err != nil {
		return d, err
	}
	d.References = refs
	d.MaterialAssumptions = reports.MaterialAssumptions(refs)
	return d, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}

func (s *Store) ReadReport(ctx context.Context, org, id, appBuild string) (ReportView, error) {
	view := ReportView{Edits: []reports.Edit{}, Stale: []string{}}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return view, err
	}
	defer tx.Rollback(ctx)
	view.Report, err = readReport(ctx, tx, org, id)
	if err != nil {
		return view, err
	}
	r := view.Report
	if r.CurrentDraftVersionID == "" {
		view.Stale = append(view.Stale, "not_assembled")
		return view, tx.Commit(ctx)
	}
	d, err := readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return view, err
	}
	view.Draft = &d
	view.Edits, err = readReportEdits(ctx, tx, org, d.ID)
	if err != nil {
		return view, err
	}
	template, err := s.reportTemplate(r.Kind)
	if err != nil {
		return view, err
	}
	if s.profileBuild == nil {
		return view, ErrInvalidReport
	}
	v, err := readProfileTx(ctx, tx, org, r.ProjectID, s.profileBuild.ReadKinds)
	if err != nil {
		return view, err
	}
	view.Stale = reports.StaleReasons(r.Kind, d.SourceRevisions, s.reportSourceState(v, template, appBuild))
	if v.PendingDocuments > 0 {
		view.Stale = append(view.Stale, "reading_pending")
	}
	if v.FailedDocuments > 0 {
		view.Stale = append(view.Stale, "reading_failed")
	}
	if len(v.StaleFor(*s.profileBuild)) > 0 {
		view.Stale = append(view.Stale, "saved_profile")
	}
	p, err := readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true)
	if err != nil {
		return view, err
	}
	if p.RetiredAt != nil {
		view.Stale = append(view.Stale, "package_retired")
	}
	return view, tx.Commit(ctx)
}
