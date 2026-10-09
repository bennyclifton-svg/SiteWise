package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/rfp"
)

// rfpTemplateID is the consultant RFP built from a discipline file
// (knowledge/rfp). Packages with no confirmed discipline keep tpl.rfp-capex.
const rfpTemplateID = "tpl.rfp-consultant"

var ErrInvalidSetting = rfp.ErrInvalidSetting

// ErrRFPOpenChecks refuses issue while a decision, client quantity or
// required detail is still open (Before issue, blocking).
var ErrRFPOpenChecks = errors.New("consultant RFP has open decisions")

// rfpState is frozen in source_revisions at refresh: the discipline version
// and the drivers the selection is made from.
type rfpState struct {
	Discipline string        `json:"discipline"`
	Version    int           `json:"version"`
	Status     string        `json:"status"`
	Drivers    rfp.DriverSet `json:"drivers"`
}

// RFPSuggestion proposes a discipline for a services package whose title
// suggests one. It is confirmed by setting the package's discipline id.
type RFPSuggestion struct {
	Discipline   string `json:"discipline"`
	Title        string `json:"title"`
	DisciplineID string `json:"discipline_id"`
}

// reportTemplateFor chooses the consultant RFP for a services package with
// a confirmed discipline id. A title alone never switches the template.
func (s *Store) reportTemplateFor(kind string, pkg procurement.Package) (knowledge.ReportTemplate, error) {
	if kind == "rfp" && s.workCatalog() != nil {
		if _, ok := s.workCatalog().RFPForDiscipline(pkg.DisciplineID); ok {
			if t, ok := s.workCatalog().CurrentReportTemplate(rfpTemplateID); ok {
				return t, nil
			}
		}
	}
	return s.reportTemplate(kind)
}

func (s *Store) rfpSuggestion(pkg procurement.Package) *RFPSuggestion {
	cat := s.workCatalog()
	if cat == nil || pkg.Kind != "services" {
		return nil
	}
	if _, ok := cat.RFPForDiscipline(pkg.DisciplineID); ok {
		return nil
	}
	d, ok := cat.RFPSuggest(pkg.Title)
	if !ok {
		return nil
	}
	return &RFPSuggestion{Discipline: d.ID, Title: d.Title, DisciplineID: d.DisciplineIDs[0]}
}

func readReportSettings(ctx context.Context, tx pgx.Tx, org, version string) (rfp.Settings, error) {
	out := rfp.Settings{}
	if version == "" {
		return out, nil
	}
	rows, err := tx.Query(ctx, `SELECT key,value,basis FROM report_settings WHERE org_id=$1::uuid AND report_version_id=$2::uuid`, org, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value, basis []byte
		if err := rows.Scan(&key, &value, &basis); err != nil {
			return nil, err
		}
		out[key] = rfp.Setting{Value: value, Basis: basis}
	}
	return out, rows.Err()
}

// assembleRFP builds a consultant RFP draft from the snapshot. It freezes
// the drivers in the source state so later setting changes need no reload.
func (s *Store) assembleRFP(snap reports.Snapshot, template knowledge.ReportTemplate, settings rfp.Settings, edits []reports.Edit, useLastCompleted bool, state *reports.SourceState) ([]reports.Section, error) {
	if reports.ReadingIncomplete(snap) && !useLastCompleted {
		return nil, reports.ErrReadingIncomplete
	}
	if !reports.ValidReportPackage("rfp", snap.Package) {
		return nil, ErrInvalidReport
	}
	cat := s.workCatalog()
	d, ok := cat.RFPForDiscipline(snap.Package.DisciplineID)
	sk, skOK := cat.RFPSkeleton()
	if !ok || !skOK {
		return nil, ErrInvalidReport
	}
	drivers := rfp.Drivers(snap, sk)
	raw, err := json.Marshal(rfpState{Discipline: d.ID, Version: d.Version, Status: d.Status, Drivers: drivers})
	if err != nil {
		return nil, err
	}
	state.RFP = raw
	sections, err := rfp.Sections(rfp.Resolve(d, sk, drivers, settings), template)
	if err != nil {
		return nil, err
	}
	return reports.ApplyEdits(sections, edits)
}

// storedRFP resolves a draft from its frozen drivers and its settings. It
// reads no project records.
func (s *Store) storedRFP(state reports.SourceState, settings rfp.Settings) (knowledge.RFPDiscipline, knowledge.RFPSkeleton, rfp.DriverSet, rfp.Resolved, error) {
	var st rfpState
	cat := s.workCatalog()
	if cat == nil || len(state.RFP) == 0 || json.Unmarshal(state.RFP, &st) != nil {
		return knowledge.RFPDiscipline{}, knowledge.RFPSkeleton{}, rfp.DriverSet{}, rfp.Resolved{}, ErrInvalidReport
	}
	d, ok := cat.RFP(st.Discipline)
	sk, skOK := cat.RFPSkeleton()
	if !ok || !skOK {
		return d, sk, st.Drivers, rfp.Resolved{}, ErrInvalidReport
	}
	return d, sk, st.Drivers, rfp.Resolve(d, sk, st.Drivers, settings), nil
}

// SetReportSetting stores one structured RFP choice and re-resolves the
// draft from its frozen drivers. A nil value deletes the setting, which
// restores the rule. Version is the draft version the user saw.
func (s *Store) SetReportSetting(ctx context.Context, org, id, actor, key string, value json.RawMessage, version int64) (ReportDraft, error) {
	var d ReportDraft
	if version < 1 || key == "" || len(key) > 120 {
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
	// Re-read under the lock: a refresh may have moved the current draft.
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
	if d.Status != "draft" || d.TemplateID != rfpTemplateID {
		return d, ErrInvalidReport
	}
	if d.Version != version {
		return d, ErrVersionConflict
	}
	settings, err := readReportSettings(ctx, tx, org, d.ID)
	if err != nil {
		return d, err
	}
	disc, sk, drivers, _, err := s.storedRFP(d.SourceRevisions, settings)
	if err != nil {
		return d, err
	}
	if value == nil {
		if _, ok := settings[key]; !ok {
			return d, ErrNotFound
		}
		if _, err = tx.Exec(ctx, `DELETE FROM report_settings WHERE org_id=$1::uuid AND report_version_id=$2::uuid AND key=$3`, org, d.ID, key); err != nil {
			return d, err
		}
		delete(settings, key)
	} else {
		if err := rfp.ValidateSetting(disc, sk, drivers, key, value); err != nil {
			return d, err
		}
		basis := rfp.SettingBasis(drivers, key)
		_, err = tx.Exec(ctx, `INSERT INTO report_settings(org_id,report_version_id,key,value,basis,user_id) VALUES($1::uuid,$2::uuid,$3,$4::jsonb,$5::jsonb,$6::uuid) ON CONFLICT(org_id,report_version_id,key) DO UPDATE SET value=EXCLUDED.value,basis=EXCLUDED.basis,user_id=EXCLUDED.user_id,version=report_settings.version+1,updated_at=now()`, org, d.ID, key, []byte(value), []byte(basis), actor)
		if err != nil {
			return d, err
		}
		settings[key] = rfp.Setting{Value: value, Basis: basis}
	}
	template, ok := s.workCatalog().CurrentReportTemplate(rfpTemplateID)
	if !ok {
		return d, ErrInvalidReport
	}
	edits, err := readReportEdits(ctx, tx, org, d.ID)
	if err != nil {
		return d, err
	}
	sections, err := rfp.Sections(rfp.Resolve(disc, sk, drivers, settings), template)
	if err != nil {
		return d, err
	}
	if sections, err = reports.ApplyEdits(sections, edits); err != nil {
		return d, err
	}
	sections, refs, err := reports.Cite(sections)
	if err != nil {
		return d, err
	}
	sectionsRaw, _ := json.Marshal(sections)
	var raw []byte
	err = tx.QueryRow(ctx, `UPDATE report_versions SET sections=$3::jsonb,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING to_jsonb(report_versions)`, org, d.ID, sectionsRaw).Scan(&raw)
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

// rfpControls is the editor payload for a consultant RFP draft.
func (s *Store) rfpControls(ctx context.Context, tx pgx.Tx, org string, d ReportDraft) (*rfp.Controls, error) {
	settings, err := readReportSettings(ctx, tx, org, d.ID)
	if err != nil {
		return nil, err
	}
	disc, sk, _, resolved, err := s.storedRFP(d.SourceRevisions, settings)
	if err != nil {
		return nil, err
	}
	c := rfp.BuildControls(resolved, disc, sk, settings)
	return &c, nil
}

// rfpIssueChecks refuses issue of a consultant RFP while blocking checks
// remain. Other report kinds have none.
func (s *Store) rfpIssueChecks(ctx context.Context, tx pgx.Tx, org string, d ReportDraft) ([]rfp.Check, error) {
	if d.TemplateID != rfpTemplateID {
		return nil, nil
	}
	settings, err := readReportSettings(ctx, tx, org, d.ID)
	if err != nil {
		return nil, err
	}
	_, _, _, resolved, err := s.storedRFP(d.SourceRevisions, settings)
	if err != nil {
		return nil, err
	}
	return resolved.Blocking(), nil
}
