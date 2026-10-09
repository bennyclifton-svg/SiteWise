package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
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
	var prior []reports.Section
	view.Issues, prior, err = reportHistory(ctx, tx, org, id)
	if err != nil {
		return view, err
	}
	var pkg procurement.Package
	if r.Kind != "pmp" {
		if pkg, err = readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true); err != nil {
			return view, err
		}
		view.RFPSuggestion = s.rfpSuggestion(pkg)
	}
	if r.CurrentDraftVersionID == "" {
		view.Stale = append(view.Stale, "not_assembled")
		return view, tx.Commit(ctx)
	}
	d, err := readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return view, err
	}
	view.Draft = &d
	issued, refs, err := reports.IssueContent(d.Sections, r.Kind, false)
	if err != nil {
		return view, err
	}
	fit := render.Fit(reports.IssueSnapshot{SchemaVersion: 1, ReportID: r.ID, VersionID: d.ID, Kind: r.Kind, Title: r.Title, ReportingDate: d.ReportingDate, SourceRevisions: d.SourceRevisions, Sections: issued, References: refs, RendererVersion: render.Version})
	view.PageFit = &fit
	if len(view.Issues) > 0 {
		compared, _, compareErr := reports.IssueContent(d.Sections, r.Kind, view.Issues[0].BudgetDisclosed)
		if compareErr != nil {
			return view, compareErr
		}
		view.Changes = reports.ChangesSinceIssue(prior, compared)
	}
	view.Edits, err = readReportEdits(ctx, tx, org, d.ID)
	if err != nil {
		return view, err
	}
	if d.TemplateID == rfpTemplateID && d.Status == "draft" {
		// A draft whose discipline file changed shape still opens; the stale
		// notice asks for a refresh.
		if view.RFP, err = s.rfpControls(ctx, tx, org, d); err != nil && !errors.Is(err, ErrInvalidReport) {
			return view, err
		}
	}
	template, err := s.reportTemplateFor(r.Kind, pkg)
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
	if r.Kind != "pmp" && pkg.RetiredAt != nil {
		view.Stale = append(view.Stale, "package_retired")
	}
	return view, tx.Commit(ctx)
}
