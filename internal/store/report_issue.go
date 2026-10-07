package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/files"
	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
)

var ErrReportStale = errors.New("report is stale; refresh or record a reason to issue the saved state")
var ErrReportUnreviewed = errors.New("report contains unreviewed standard clauses")

type IssueOptions struct {
	Version         int64  `json:"version"`
	ReportingDate   string `json:"reporting_date"`
	BudgetDisclosed bool   `json:"budget_disclosed"`
	AcceptStale     bool   `json:"accept_stale"`
	StaleReason     string `json:"stale_reason"`
}
type IssuedReport struct {
	ID             string                `json:"id"`
	Number         int                   `json:"number"`
	ReportingDate  string                `json:"reporting_date"`
	SnapshotSHA256 string                `json:"snapshot_sha256"`
	ExportSHA256   []byte                `json:"-"`
	Snapshot       reports.IssueSnapshot `json:"snapshot"`
}

func (s *Store) IssueReport(ctx context.Context, org, id, actor, appBuild string, options IssueOptions, blobs *files.Store) (IssuedReport, error) {
	var out IssuedReport
	if options.Version < 1 || blobs == nil || len(options.StaleReason) > 1000 {
		return out, ErrInvalidReport
	}
	if _, err := time.Parse("2006-01-02", options.ReportingDate); err != nil {
		return out, ErrInvalidReport
	}
	if options.AcceptStale && strings.TrimSpace(options.StaleReason) == "" {
		return out, ErrInvalidReport
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return out, err
	}
	if err = lockProject(ctx, tx, org, r.ProjectID); err != nil {
		return out, err
	}
	if err = packageActor(ctx, tx, org, actor); err != nil {
		return out, err
	}
	r, err = readReport(ctx, tx, org, id)
	if err != nil {
		return out, err
	}
	if r.CurrentDraftVersionID == "" {
		return out, ErrVersionConflict
	}
	d, err := readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return out, err
	}
	if d.Status != "draft" || d.Version != options.Version {
		return out, ErrVersionConflict
	}
	template, err := s.reportTemplate(r.Kind)
	if err != nil {
		return out, err
	}
	if r.Kind != "pmp" {
		p, e := readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true)
		if e != nil {
			return out, e
		}
		if p.RetiredAt != nil {
			return out, ErrReportPackageRetired
		}
	}
	v, err := readProfileTx(ctx, tx, org, r.ProjectID, s.profileBuild.ReadKinds)
	if err != nil {
		return out, err
	}
	stale := reports.StaleReasons(r.Kind, d.SourceRevisions, s.reportSourceState(v, template, appBuild))
	if v.PendingDocuments > 0 || v.FailedDocuments > 0 || len(v.StaleFor(*s.profileBuild)) > 0 {
		stale = append(stale, "reading")
	}
	if len(stale) > 0 && !options.AcceptStale {
		return out, ErrReportStale
	}
	for _, section := range d.Sections {
		for _, b := range section.Blocks {
			if b.Conflict || b.SourceMissing {
				return out, ErrVersionConflict
			}
			// Draft standard wording can be reviewed in-app, but cannot become an
			// issued standard obligation merely because its text was edited.
			if unreviewedReportClause(b) {
				return out, ErrReportUnreviewed
			}
		}
	}
	sections, refs, err := reports.IssueContent(d.Sections, r.Kind, options.BudgetDisclosed)
	if err != nil {
		return out, err
	}
	snap := reports.IssueSnapshot{SchemaVersion: 1, ReportID: r.ID, VersionID: d.ID, Kind: r.Kind, Title: r.Title, ReportingDate: options.ReportingDate, SourceRevisions: d.SourceRevisions, Sections: sections, References: refs, BudgetDisclosed: options.BudgetDisclosed, RendererVersion: render.Version}
	if len(stale) > 0 {
		snap.StaleReason = options.StaleReason
	}
	raw, hash, err := reports.CanonicalSnapshot(snap)
	if err != nil {
		return out, ErrInvalidReport
	}
	pdf, err := render.PDF(snap)
	if err != nil {
		return out, err
	}
	blob, err := blobs.Put(ctx, bytes.NewReader(pdf.PDF))
	if err != nil {
		return out, err
	}
	// Register the exported blob with existing backup/restore accounting. An
	// uncommitted blob after a crash is harmless and never an issued document.
	_, err = tx.Exec(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'application/pdf') ON CONFLICT(org_id,project_id,sha256) DO NOTHING`, org, newID(), r.ProjectID, blob.SHA256, blob.Size)
	if err != nil {
		return out, err
	}
	if err = saveReportReferences(ctx, tx, org, d.ID, refs); err != nil {
		return out, err
	}
	sectionRaw, _ := json.Marshal(sections)
	_, err = tx.Exec(ctx, `UPDATE report_versions SET status='issued',reporting_date=$3::date,budget_disclosed=$4,snapshot=$5::jsonb,snapshot_sha256=$6,export_file_sha256=$7,issued_at=now(),issued_by=$8::uuid,sections=$9::jsonb,version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, d.ID, options.ReportingDate, options.BudgetDisclosed, raw, hash, blob.SHA256, actor, sectionRaw)
	if err != nil {
		return out, err
	}
	_, err = tx.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID)
	if err != nil {
		return out, err
	}
	out = IssuedReport{ID: d.ID, Number: d.Number, ReportingDate: options.ReportingDate, SnapshotSHA256: hash, ExportSHA256: blob.SHA256, Snapshot: snap}
	return out, s.finishReportWrite(ctx, tx, org, r, d.ID, "issued")
}

func (s *Store) ReadIssuedReport(ctx context.Context, org, reportID, versionID string) (IssuedReport, error) {
	var out IssuedReport
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT id::text,number,reporting_date::text,snapshot_sha256,export_file_sha256,snapshot FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND id=$3::uuid AND status='issued'`, org, reportID, versionID).Scan(&out.ID, &out.Number, &out.ReportingDate, &out.SnapshotSHA256, &out.ExportSHA256, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out.Snapshot)
	return out, err
}

// Scope clauses use scope:<id> anchors, so an anchor prefix alone cannot prove
// that every catalogue obligation was reviewed before issue.
func unreviewedReportClause(b reports.Block) bool {
	var target struct {
		Reviewed *bool `json:"target_clauses_reviewed"`
	}
	if json.Unmarshal(b.Basis, &target) == nil && target.Reviewed != nil && !*target.Reviewed {
		return true
	}
	if !b.Provisional {
		return false
	}
	if strings.HasPrefix(b.ID, "clause:") {
		return true
	}
	var source struct {
		ClauseID string `json:"clause_id"`
	}
	if json.Unmarshal(b.Basis, &source) != nil {
		return true
	}
	return source.ClauseID != ""
}
