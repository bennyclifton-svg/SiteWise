package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/knowledge"
	"sitewise/internal/reports"
	"sitewise/internal/rfp"
)

// Schedule sources: the package's RFP, the discipline's RFP with no
// settings, or the generic lump-sum schedule.
const (
	ScheduleFromRFP     = "rfp"
	ScheduleFromDefault = "default"
	ScheduleGeneric     = "generic"
)

// syncRFPSchedule makes the package's rows the schedule its current RFP
// draft prints, so a proposal is read into exactly the fee rows the RFP
// asked for. It runs where the rows are used (price table, proposal read,
// added firm), not on each RFP edit: a setting change is a click the PM
// waits on, and its budget has no room for the extra reads and upsert.
// synced is false when the package has no consultant RFP draft.
func (s *Store) syncRFPSchedule(ctx context.Context, tx pgx.Tx, org, project, pkg string) (synced bool, err error) {
	var versionID, template string
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT v.id::text, v.template_id, v.source_revisions FROM reports r
JOIN report_versions v ON v.org_id = r.org_id AND v.id = r.current_draft_version_id
WHERE r.org_id = $1::uuid AND r.package_id = $2::uuid AND r.kind = 'rfp'
ORDER BY r.updated_at DESC LIMIT 1`, org, pkg).Scan(&versionID, &template, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil || template != rfpTemplateID {
		return false, err
	}
	var state reports.SourceState
	if json.Unmarshal(raw, &state) != nil {
		return false, nil
	}
	settings, err := readReportSettings(ctx, tx, org, versionID)
	if err != nil {
		return false, err
	}
	_, _, _, res, err := s.storedRFP(state, settings)
	if errors.Is(err, ErrInvalidReport) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, upsertSchedule(ctx, tx, org, project, pkg, rfp.Schedule(res), ScheduleFromRFP)
}

// upsertSchedule makes the package's live rows exactly items. A row the
// schedule no longer has is retired, so prices and cost lines that name it
// keep their label.
func upsertSchedule(ctx context.Context, tx pgx.Tx, org, project, pkg string, items []rfp.ScheduleItem, source string) error {
	keys := make([]string, len(items))
	kinds := make([]string, len(items))
	labels := make([]string, len(items))
	details := make([]string, len(items))
	bases := make([]bool, len(items))
	ordinals := make([]int32, len(items))
	for i, it := range items {
		keys[i], kinds[i], labels[i], details[i], bases[i], ordinals[i] = it.Key, it.Kind, clipText(it.Label, 500), clipText(it.Detail, 4000), it.Base, int32(i)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO schedule_items (org_id, project_id, package_id, key, kind, label, detail, base, ordinal, source)
SELECT $1::uuid, $2::uuid, $3::uuid, k, kd, l, d, b, o, $10
FROM unnest($4::text[], $5::text[], $6::text[], $7::text[], $8::bool[], $9::int[]) AS t(k, kd, l, d, b, o)
ON CONFLICT (org_id, package_id, key) DO UPDATE SET kind = EXCLUDED.kind, label = EXCLUDED.label, detail = EXCLUDED.detail,
  base = EXCLUDED.base, ordinal = EXCLUDED.ordinal, source = EXCLUDED.source, retired_at = NULL, updated_at = now()
WHERE (schedule_items.kind, schedule_items.label, schedule_items.detail, schedule_items.base, schedule_items.ordinal, schedule_items.source, schedule_items.retired_at IS NULL)
  IS DISTINCT FROM (EXCLUDED.kind, EXCLUDED.label, EXCLUDED.detail, EXCLUDED.base, EXCLUDED.ordinal, EXCLUDED.source, true)`,
		org, project, pkg, keys, kinds, labels, details, bases, ordinals, source); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE schedule_items SET retired_at = now(), updated_at = now()
WHERE org_id = $1::uuid AND package_id = $2::uuid AND retired_at IS NULL AND NOT (key = ANY($3::text[]))`, org, pkg, keys)
	return err
}

// ScheduleItem is one stored row; Retired rows are kept for history.
type ScheduleItem struct {
	rfp.ScheduleItem
	Source  string `json:"source"`
	Retired bool   `json:"retired"`
}

// readSchedule is a package's rows in print order, live first.
func readSchedule(ctx context.Context, q querier, org, pkg string) ([]ScheduleItem, error) {
	rows, err := q.Query(ctx, `SELECT key, kind, label, detail, base, ordinal, source, retired_at IS NOT NULL
FROM schedule_items WHERE org_id = $1::uuid AND package_id = $2::uuid ORDER BY retired_at IS NOT NULL, ordinal, key`, org, pkg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScheduleItem
	for rows.Next() {
		var it ScheduleItem
		var ord int32
		if err := rows.Scan(&it.Key, &it.Kind, &it.Label, &it.Detail, &it.Base, &ord, &it.Source, &it.Retired); err != nil {
			return nil, err
		}
		it.Ordinal = int(ord)
		out = append(out, it)
	}
	return out, rows.Err()
}

// ensureSchedule gives a package rows to read a proposal into. A package
// with an RFP takes its draft's schedule; otherwise the discipline's RFP
// resolved with no settings supplies a default, and a discipline with no RFP
// knowledge the generic lump sum.
func (s *Store) ensureSchedule(ctx context.Context, tx pgx.Tx, org, project, pkg string) error {
	if synced, err := s.syncRFPSchedule(ctx, tx, org, project, pkg); err != nil || synced {
		return err
	}
	var have bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schedule_items WHERE org_id = $1::uuid AND package_id = $2::uuid AND retired_at IS NULL)`, org, pkg).Scan(&have); err != nil || have {
		return err
	}
	cat := s.workCatalog()
	var sk knowledge.RFPSkeleton
	skOK := false
	if cat != nil {
		sk, skOK = cat.RFPSkeleton()
	}
	var returnables []rfp.Row
	if skOK {
		for _, r := range sk.Returnables {
			returnables = append(returnables, rfp.Row{Key: r.Key, Label: r.Title, Text: r.Text})
		}
	}
	items := rfp.GenericSchedule(returnables)
	source := ScheduleGeneric
	template, terr := s.reportTemplate("rfp")
	if cat != nil && skOK && terr == nil {
		if t, ok := cat.CurrentReportTemplate(rfpTemplateID); ok {
			template = t
		}
		snap, _, err := s.reportSnapshot(ctx, tx, org, project, pkg, "", template)
		if err != nil {
			return err
		}
		if d, ok := cat.RFPForDiscipline(snap.Package.DisciplineID); ok {
			items = rfp.Schedule(rfp.Resolve(d, sk, rfp.Drivers(snap, sk), rfp.Settings{}))
			source = ScheduleFromDefault
		}
	}
	return upsertSchedule(ctx, tx, org, project, pkg, items, source)
}

func clipText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut]
}

// PackageSchedule is a package's returnable schedule, creating the default
// one when the package has no RFP yet.
func (s *Store) PackageSchedule(ctx context.Context, org, project, pkg string) ([]ScheduleItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := packageInProject(ctx, tx, org, project, pkg); err != nil {
		return nil, err
	}
	if err := s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return nil, err
	}
	items, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return nil, err
	}
	return items, tx.Commit(ctx)
}

// packageInProject is ErrNotFound for a package of another project or org.
func packageInProject(ctx context.Context, q querier, org, project, pkg string) error {
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM packages WHERE org_id = $1::uuid AND project_id = $2::uuid AND id::text = $3)`, org, project, pkg).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ResponseForm is an RFP's response form: its schedule rows and returnables
// as the current draft resolves them.
type ResponseForm struct {
	Header      rfp.FormHeader
	Items       []rfp.ScheduleItem
	Returnables []rfp.Row
}

// ReadResponseForm resolves a consultant RFP's draft into its form. A report
// that is not a consultant RFP has no form (ErrInvalidReport).
func (s *Store) ReadResponseForm(ctx context.Context, org, reportID string) (ResponseForm, error) {
	var f ResponseForm
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return f, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, reportID)
	if err != nil {
		return f, err
	}
	if r.Kind != "rfp" || r.CurrentDraftVersionID == "" {
		return f, ErrInvalidReport
	}
	d, err := readReportDraft(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return f, err
	}
	if d.TemplateID != rfpTemplateID {
		return f, ErrInvalidReport
	}
	settings, err := readReportSettings(ctx, tx, org, d.ID)
	if err != nil {
		return f, err
	}
	_, _, _, res, err := s.storedRFP(d.SourceRevisions, settings)
	if err != nil {
		return f, err
	}
	var project string
	if err := tx.QueryRow(ctx, `SELECT name FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ProjectID).Scan(&project); err != nil {
		return f, err
	}
	f.Header = rfp.FormHeader{Title: r.Title, Project: project, Ref: res.Fields["ref"].Value}
	f.Items = rfp.Schedule(res)
	f.Returnables = res.Returnables
	return f, tx.Commit(ctx)
}

// AddFirmSubmission puts a directory firm under a package with no file yet;
// its prices are typed in the table.
func (s *Store) AddFirmSubmission(ctx context.Context, org, project, pkg, firmID string, now time.Time) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := packageInProject(ctx, tx, org, project, pkg); err != nil {
		return "", err
	}
	f, err := readFirm(ctx, tx, org, firmID)
	if err != nil {
		return "", err
	}
	id := newID()
	if _, err := tx.Exec(ctx, `INSERT INTO submissions (org_id,id,project_id,package_id,firm_id,filename,kind,status,received_on)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,'fee_proposal','received',$7::date)`, org, id, project, pkg, f.ID, clipText(f.Name, 255), SydneyDate(now)); err != nil {
		return "", err
	}
	if err := s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return "", err
	}
	if err := procurementEvent(ctx, tx, s, org, project, id, "received"); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}
