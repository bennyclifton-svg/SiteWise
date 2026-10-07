package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/knowledge"
	"sitewise/internal/reports"
)

var ErrInvalidReport = errors.New("invalid or unavailable report")
var ErrReportPackageRetired = errors.New("report package is retired")

type Report struct {
	ID                    string `json:"id"`
	ProjectID             string `json:"project_id"`
	Kind                  string `json:"kind"`
	PackageID             string `json:"package_id"`
	Title                 string `json:"title"`
	CurrentDraftVersionID string `json:"current_draft_version_id"`
	Version               int64  `json:"version"`
}

type ReportDraft struct {
	ID                  string              `json:"id"`
	ReportID            string              `json:"report_id"`
	ProjectID           string              `json:"project_id"`
	Number              int                 `json:"number"`
	Status              string              `json:"status"`
	ReportingDate       string              `json:"reporting_date"`
	SourceRevisions     reports.SourceState `json:"source_revisions"`
	TemplateID          string              `json:"template_id"`
	TemplateVersion     int                 `json:"template_version"`
	Sections            []reports.Section   `json:"sections"`
	References          []reports.Reference `json:"references"`
	MaterialAssumptions []reports.Reference `json:"material_assumptions"`
	Version             int64               `json:"version"`
}

type ReportView struct {
	Issues  []ReportIssueSummary `json:"issues"`
	Changes []reports.Change     `json:"changes_since_issue"`
	Report  Report               `json:"report"`
	Draft   *ReportDraft         `json:"draft"`
	Edits   []reports.Edit       `json:"edits"`
	Stale   []string             `json:"stale"`
}

func readReport(ctx context.Context, tx pgx.Tx, org, id string) (Report, error) {
	var r Report
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(r) FROM reports r WHERE org_id=$1::uuid AND id=$2::uuid`, org, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(raw, &r)
	return r, err
}

func readReportDraft(ctx context.Context, tx pgx.Tx, org, id string) (ReportDraft, error) {
	var d ReportDraft
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(v) FROM report_versions v WHERE org_id=$1::uuid AND id=$2::uuid`, org, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(raw, &d)
	if err == nil {
		d.References, err = readReportReferences(ctx, tx, org, id)
		d.MaterialAssumptions = reports.MaterialAssumptions(d.References)
	}
	return d, err
}

func readReportEdits(ctx context.Context, tx pgx.Tx, org, id string) ([]reports.Edit, error) {
	rows, err := tx.Query(ctx, `SELECT target_id,text,base_content_sha256,user_id::text,version,updated_at FROM report_edits WHERE org_id=$1::uuid AND report_version_id=$2::uuid ORDER BY target_id`, org, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reports.Edit{}
	for rows.Next() {
		var e reports.Edit
		if err := rows.Scan(&e.TargetID, &e.Text, &e.BaseContentSHA256, &e.UserID, &e.Version, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) reportTemplate(kind string) (knowledge.ReportTemplate, error) {
	if s.workCatalog() == nil {
		return knowledge.ReportTemplate{}, ErrInvalidReport
	}
	id := map[string]string{"rfp": "tpl.rfp-capex", "rft": "tpl.rft", "pmp": "tpl.pmp"}[kind]
	t, ok := s.workCatalog().CurrentReportTemplate(id)
	if !ok {
		return t, ErrInvalidReport
	}
	return t, nil
}

func (s *Store) CreateReport(ctx context.Context, org, project, actor, kind, pkg string) (Report, error) {
	var r Report
	var uuid pgtype.UUID
	if _, err := s.reportTemplate(kind); err != nil {
		return r, err
	}
	if kind == "pmp" && pkg != "" || kind != "pmp" && uuid.Scan(pkg) != nil {
		return r, ErrInvalidReport
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return r, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return r, err
	}
	var name string
	if kind == "pmp" {
		err = tx.QueryRow(ctx, `SELECT name FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, project).Scan(&name)
	} else {
		p, e := readPackage(ctx, tx, org, project, pkg)
		err = e
		name = p.Title
		if e == nil && (kind == "rfp" && p.Kind != "services" || kind == "rft" && p.Kind != "works") {
			return r, ErrInvalidReport
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	var raw []byte
	err = tx.QueryRow(ctx, `INSERT INTO reports(org_id,id,project_id,kind,package_id,title) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,'')::uuid,left($6,200)) RETURNING to_jsonb(reports)`, org, newID(), project, kind, pkg, strings.ToUpper(kind)+" — "+name).Scan(&raw)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	return r, s.finishReportWrite(ctx, tx, org, r, "", "drafting")
}

func (s *Store) RefreshReport(ctx context.Context, org, id, actor, appBuild string, useLastCompleted bool) (ReportDraft, error) {
	for attempt := 0; attempt < 3; attempt++ {
		d, err := s.refreshReportOnce(ctx, org, id, actor, appBuild, useLastCompleted)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "40001" {
			return d, err
		}
	}
	return ReportDraft{}, ErrVersionConflict
}

func (s *Store) refreshReportOnce(ctx context.Context, org, id, actor, appBuild string, useLastCompleted bool) (ReportDraft, error) {
	var d ReportDraft
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
	template, err := s.reportTemplate(r.Kind)
	if err != nil {
		return d, err
	}
	if r.Kind != "pmp" {
		p, err := readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true)
		if err != nil {
			return d, err
		}
		if p.RetiredAt != nil {
			return d, ErrReportPackageRetired
		}
	}
	previous := ""
	history, _, err := reportHistory(ctx, tx, org, r.ID)
	if err != nil {
		return d, err
	}
	if len(history) > 0 {
		previous = history[0].ID
	}
	edits := []reports.Edit{}
	if r.CurrentDraftVersionID == "" && previous != "" {
		edits, err = readReportEdits(ctx, tx, org, previous)
		if err != nil {
			return d, err
		}
	}

	if r.CurrentDraftVersionID != "" {
		d, err = readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
		if err != nil {
			return d, err
		}
		if d.Status != "draft" {
			return d, ErrInvalidReport
		}
		edits, err = readReportEdits(ctx, tx, org, d.ID)
		if err != nil {
			return d, err
		}
	}
	snap, state, err := s.reportSnapshot(ctx, tx, org, r.ProjectID, r.PackageID, appBuild, template)
	if err != nil {
		return d, err
	}
	sections, err := reports.Assemble(snap, template, s.workCatalog(), edits, useLastCompleted)
	if err != nil {
		return d, err
	}
	sections, refs, err := reports.Cite(sections)
	if err != nil {
		return d, err
	}
	stateRaw, _ := json.Marshal(state)
	sectionsRaw, _ := json.Marshal(sections)
	var raw []byte
	creatingDraft := d.ID == ""
	if creatingDraft {
		d.ID = newID()
		err = tx.QueryRow(ctx, `INSERT INTO report_versions(org_id,id,report_id,project_id,number,status,reporting_date,source_revisions,template_id,template_version,sections,previous_issue_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,(SELECT COALESCE(max(number),0)+1 FROM report_versions WHERE org_id=$1::uuid AND report_id=$3::uuid),'draft',CURRENT_DATE,$5::jsonb,$6,$7,$8::jsonb,NULLIF($9,'')::uuid) RETURNING to_jsonb(report_versions)`, org, d.ID, r.ID, r.ProjectID, stateRaw, template.ID, template.Version, sectionsRaw, previous).Scan(&raw)
	} else {
		err = tx.QueryRow(ctx, `UPDATE report_versions SET source_revisions=$3::jsonb,template_id=$4,template_version=$5,sections=$6::jsonb,updated_at=now(),version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING to_jsonb(report_versions)`, org, d.ID, stateRaw, template.ID, template.Version, sectionsRaw).Scan(&raw)
	}
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, err
	}
	if previous != "" && creatingDraft {
		_, err = tx.Exec(ctx, `INSERT INTO report_edits(org_id,report_version_id,target_id,text,base_content_sha256,user_id,version,created_at,updated_at) SELECT org_id,$3::uuid,target_id,text,base_content_sha256,user_id,version,created_at,updated_at FROM report_edits WHERE org_id=$1::uuid AND report_version_id=$2::uuid ON CONFLICT DO NOTHING`, org, previous, d.ID)
		if err != nil {
			return d, err
		}
	}
	if err := saveReportReferences(ctx, tx, org, d.ID, refs); err != nil {
		return d, err
	}
	d.References = refs
	d.MaterialAssumptions = reports.MaterialAssumptions(refs)
	if _, err := tx.Exec(ctx, `UPDATE reports SET current_draft_version_id=$3::uuid,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID, d.ID); err != nil {
		return d, err
	}
	return d, s.finishReportWrite(ctx, tx, org, r, d.ID, "ready")
}

func (s *Store) finishReportWrite(ctx context.Context, tx pgx.Tx, org string, r Report, versionID, state string) error {
	if err := BumpRevision(ctx, tx, org, r.ProjectID, "reports"); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"project_id": r.ProjectID, "report_id": r.ID, "version_id": versionID, "state": state})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), org, "report", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
