package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
)

// One stage plan shared with Procurement and the Cost Plan (plan
// docs/plans/2026-10-18-program.md section 7.6, WP-P10). The program's
// stages are the project's stage plan; package stage sets and RFP versions
// record the plan as it was when they were built (038_stage_sync.sql). When
// the program's stage revision has moved on, the document shows the
// differences, keyed by the stable program stage id, and offers Update or
// Keep. Flow is one way, Program to downstream. All of it is code; no Jev.

// Stage-sync choices.
const (
	StageSyncUpdate = "update"
	StageSyncKeep   = "keep"
)

// stagePlan is the program's live stages in program order.
type stagePlan struct {
	ProgramID string
	Revision  int64
	Stages    []procurement.ProgramStage
}

func (p *stagePlan) stage(id string) (procurement.ProgramStage, bool) {
	if p == nil {
		return procurement.ProgramStage{}, false
	}
	for _, s := range p.Stages {
		if s.ID == id {
			return s, true
		}
	}
	return procurement.ProgramStage{}, false
}

func (p *stagePlan) index(id string) int {
	for i, s := range p.Stages {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// planStage is one entry of a recorded stage_plan.
type planStage struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (p *stagePlan) recorded() []planStage {
	out := make([]planStage, len(p.Stages))
	for i, s := range p.Stages {
		out[i] = planStage{ID: s.ID, Label: s.Label}
	}
	return out
}

// readStagePlan reads the project's program stages, or nil when the
// project has no program. lock holds the program row so its stage revision
// cannot move under a stage-sync write.
func readStagePlan(ctx context.Context, q programQuerier, org, project string, lock bool) (*stagePlan, error) {
	p := &stagePlan{Stages: []procurement.ProgramStage{}}
	sql := `SELECT id::text,stage_revision FROM programs WHERE org_id=$1::uuid AND project_id=$2::uuid`
	if lock {
		sql += ` FOR SHARE`
	}
	err := q.QueryRow(ctx, sql, org, project).Scan(&p.ProgramID, &p.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id::text,label,COALESCE(catalogue_stage_id,'') FROM program_activities
WHERE org_id=$1::uuid AND program_id=$2::uuid AND kind='stage' AND retired_at IS NULL ORDER BY ordinal,id`, org, p.ProgramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s procurement.ProgramStage
		if err := rows.Scan(&s.ID, &s.Label, &s.CatalogueStageID); err != nil {
			return nil, err
		}
		p.Stages = append(p.Stages, s)
	}
	return p, rows.Err()
}

// EnsureProgram creates the program from a downstream surface when the
// project has none (plan 7.6: whichever surface comes first creates it).
// An existing program costs one indexed lookup; the full prepare runs only
// the first time.
func (s *Store) EnsureProgram(ctx context.Context, org, project, surface string, cat *knowledge.Catalog) error {
	if cat == nil {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM programs WHERE org_id=$1::uuid AND project_id=$2::uuid)`, org, project).Scan(&exists); err != nil || exists {
		return err
	}
	_, _, err := s.PrepareProgram(ctx, org, project, surface, cat)
	return err
}

// recordPackagePlans stamps the plan on every live package whose stages
// point at the program and that has no recorded plan yet: prepare linked
// them in the same transaction.
func recordPackagePlans(ctx context.Context, tx pgx.Tx, org, project string) error {
	plan, err := readStagePlan(ctx, tx, org, project, false)
	if err != nil || plan == nil {
		return err
	}
	raw, _ := json.Marshal(plan.recorded())
	_, err = tx.Exec(ctx, `UPDATE packages p SET stage_plan=$3::jsonb,stage_revision=$4 WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.retired_at IS NULL AND p.stage_plan IS NULL
 AND EXISTS(SELECT 1 FROM package_stages s WHERE s.org_id=p.org_id AND s.package_id=p.id AND s.retired_at IS NULL AND s.program_stage_id IS NOT NULL)`, org, project, raw, plan.Revision)
	return err
}

// defaultStages are a new package's stages: from the program when the
// project has one, else copied from the catalogue as before.
func (s *Store) defaultStages(p procurement.Package, plan *stagePlan) []procurement.Stage {
	if plan != nil {
		return procurement.ProgramDefaultStages(p, s.workCatalog(), plan.Stages, plan.Revision)
	}
	return procurement.DefaultStages(p, s.workCatalog())
}

// linkPackageStage points a stage being created at the program. A stage
// naming its program stage takes the program's catalogue stage and label
// (Procurement cannot add or rename stages); one that does not is linked to
// the first program stage with the same catalogue stage, if any.
func (s *Store) linkPackageStage(p procurement.Package, stage *procurement.Stage, plan *stagePlan) error {
	if plan == nil {
		if stage.ProgramStageID != "" {
			return ErrNotFound
		}
		stage.StageRevision = 0
		return nil
	}
	if stage.ProgramStageID != "" {
		ps, ok := plan.stage(stage.ProgramStageID)
		if !ok {
			return ErrNotFound
		}
		stage.StageID, stage.Label = ps.CatalogueStageID, ps.Label
		if stage.NovationPhase == "" || stage.NovationPhase == "none" {
			stage.NovationPhase = procurement.ProgramNovationPhase(p, s.workCatalog(), ps.CatalogueStageID)
		}
	} else {
		for _, ps := range plan.Stages {
			if ps.CatalogueStageID == stage.StageID {
				stage.ProgramStageID = ps.ID
				break
			}
		}
	}
	stage.StageRevision = 0
	if stage.ProgramStageID != "" {
		stage.StageRevision = plan.Revision
	}
	return nil
}

// StageSyncChange is one difference with what Update does to it here.
type StageSyncChange struct {
	engine.StageChange
	Action string `json:"action"`
	// Held is true when invoices, awarded money or another live record keep
	// the downstream stage: it is never removed downstream.
	Held bool `json:"held,omitempty"`
}

// PickerStage is one checkbox of a package's stage picker.
type PickerStage struct {
	ProgramStageID      string `json:"program_stage_id"`
	Label               string `json:"label"`
	CatalogueStageID    string `json:"catalogue_stage_id"`
	Ticked              bool   `json:"ticked"`
	PackageStageID      string `json:"package_stage_id,omitempty"`
	PackageStageVersion int64  `json:"package_stage_version,omitempty"`
}

// StageRow is one stage row of an RFP version.
type StageRow struct {
	ProgramStageID     string `json:"program_stage_id"`
	Label              string `json:"label"`
	RemovedFromProgram bool   `json:"removed_from_program,omitempty"`
}

// StageSync is a document's view of the stage plan: the differences since
// the revision it was built from (empty when it is up to date or kept),
// what Update does to each, and the stage picker or the RFP's rows.
type StageSync struct {
	Document      string            `json:"document"`
	ProgramExists bool              `json:"program_exists"`
	Revision      int64             `json:"revision"`
	BuiltFrom     int64             `json:"built_from"`
	Acknowledged  int64             `json:"acknowledged"`
	Offer         bool              `json:"offer"`
	Changes       []StageSyncChange `json:"changes"`
	Reordered     bool              `json:"reordered"`
	// ProposalsFlagged counts the proposals received for the package when
	// a stage was added or removed: they never change, and no longer match
	// the stages they priced.
	ProposalsFlagged int    `json:"proposals_flagged"`
	Awarded          bool   `json:"awarded"`
	PackageID        string `json:"package_id"`
	// Package: the picker, and stages kept after leaving the program.
	Stages []PickerStage       `json:"stages"`
	Kept   []procurement.Stage `json:"kept"`
	// RFP: the report, the version the rows belong to, and its rows.
	ReportID  string     `json:"report_id,omitempty"`
	VersionID string     `json:"version_id,omitempty"`
	Rows      []StageRow `json:"rows"`
}

func emptySync(doc string) StageSync {
	return StageSync{Document: doc, Changes: []StageSyncChange{}, Stages: []PickerStage{}, Kept: []procurement.Stage{}, Rows: []StageRow{}}
}

// diffPlan compares a recorded plan with the program's current stages.
func diffPlan(raw []byte, plan *stagePlan) (engine.StagePlanDiff, error) {
	var was []planStage
	if err := json.Unmarshal(raw, &was); err != nil {
		return engine.StagePlanDiff{}, err
	}
	rec := make([]engine.RecordedStage, len(was))
	for i, w := range was {
		rec[i] = engine.RecordedStage{ProgramStageID: w.ID, Label: w.Label}
	}
	cur := make([]engine.ProgramStage, len(plan.Stages))
	for i, s := range plan.Stages {
		cur[i] = engine.ProgramStage{ID: s.ID, Label: s.Label, CatalogueStageID: s.CatalogueStageID}
	}
	return engine.DiffStages(rec, cur), nil
}

// packageAwarded reports a live award on the package.
func packageAwarded(ctx context.Context, tx pgx.Tx, org, project, pkg string) (bool, error) {
	var awarded bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM awards WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND undone_at IS NULL)`, org, project, pkg).Scan(&awarded)
	return awarded, err
}

// stageHeld reports whether a package stage must stay downstream when its
// program stage is deleted: invoices or awarded money, or a live record the
// existing retire path refuses on (a live cost line, an open delivery
// record, scope).
func stageHeld(ctx context.Context, tx pgx.Tx, org, project, pkg, stage string, awarded bool) (bool, error) {
	if awarded {
		return true, nil
	}
	var held bool
	err := tx.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM invoice_allocations a JOIN invoices i ON i.org_id=a.org_id AND i.id=a.invoice_id
  JOIN cost_item_revisions r ON r.org_id=a.org_id AND r.cost_item_id=a.cost_item_id
  WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND r.package_stage_id=$4::uuid AND i.status NOT IN ('void','superseded'))
 OR EXISTS(SELECT 1 FROM cost_plans p JOIN cost_item_revisions r ON r.org_id=p.org_id AND r.plan_version_id=p.draft_version_id
  WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND r.package_stage_id=$4::uuid AND r.posting AND NOT r.excluded)
 OR EXISTS(SELECT 1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND stage_id=$4::uuid AND retired_at IS NULL AND `+deliveryOpenSQL+`)
 OR EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND stage_id=$4::uuid AND retired_at IS NULL)`,
		org, project, pkg, stage).Scan(&held)
	return held, err
}

// proposalsReceived counts the package's proposals that were read.
func proposalsReceived(ctx context.Context, tx pgx.Tx, org, project, pkg string) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND status IN ('received','needs_review','awarded','unsuccessful')`, org, project, pkg).Scan(&n)
	return n, err
}

// pkgStageRecord is what a package's stage sync is computed from.
type pkgStageRecord struct {
	plan      []byte
	revision  *int64
	acked     *int64
	pkg       procurement.Package
	byProgram map[string]procurement.Stage // live, by program stage id
	heldByID  map[string]bool              // package stage id → held
}

func readPackageStageRecord(ctx context.Context, tx pgx.Tx, org, project, pkg string) (pkgStageRecord, error) {
	rec := pkgStageRecord{byProgram: map[string]procurement.Stage{}, heldByID: map[string]bool{}}
	p, err := readPackage(ctx, tx, org, project, pkg)
	if err != nil {
		return rec, err
	}
	rec.pkg = p
	if err := tx.QueryRow(ctx, `SELECT stage_plan,stage_revision,stage_acknowledged FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, pkg).Scan(&rec.plan, &rec.revision, &rec.acked); err != nil {
		return rec, err
	}
	for _, st := range p.Stages {
		if st.ProgramStageID != "" {
			if _, dup := rec.byProgram[st.ProgramStageID]; !dup {
				rec.byProgram[st.ProgramStageID] = st
			}
		}
	}
	return rec, nil
}

// computePackageSync is the package's stage sync inside tx.
func computePackageSync(ctx context.Context, tx pgx.Tx, org, project, pkg string, plan *stagePlan) (StageSync, pkgStageRecord, error) {
	out := emptySync(engine.DocPackageStages)
	out.PackageID = pkg
	rec, err := readPackageStageRecord(ctx, tx, org, project, pkg)
	if err != nil {
		return out, rec, err
	}
	if out.Awarded, err = packageAwarded(ctx, tx, org, project, pkg); err != nil {
		return out, rec, err
	}
	if out.Awarded {
		out.Document = engine.DocAwardedContract
	}
	if plan == nil {
		return out, rec, nil
	}
	out.ProgramExists, out.Revision = true, plan.Revision
	for _, ps := range plan.Stages {
		pick := PickerStage{ProgramStageID: ps.ID, Label: ps.Label, CatalogueStageID: ps.CatalogueStageID}
		if st, ok := rec.byProgram[ps.ID]; ok {
			pick.Ticked, pick.PackageStageID, pick.PackageStageVersion = true, st.ID, st.Version
		}
		out.Stages = append(out.Stages, pick)
	}
	// Stages that point at no live program stage: kept after leaving it, or
	// never linked.
	for _, st := range rec.pkg.Stages {
		if _, live := plan.stage(st.ProgramStageID); !live {
			out.Kept = append(out.Kept, st)
		}
	}
	if rec.revision != nil {
		out.BuiltFrom = *rec.revision
	}
	if rec.acked != nil {
		out.Acknowledged = *rec.acked
	}
	if rec.plan == nil || len(rec.byProgram) == 0 || !engine.NeedsOffer(out.BuiltFrom, out.Acknowledged, plan.Revision) {
		return out, rec, nil
	}
	d, err := diffPlan(rec.plan, plan)
	if err != nil {
		return out, rec, err
	}
	addedOrRemoved := false
	for _, c := range d.Changes {
		st, has := rec.byProgram[c.ProgramStageID]
		switch c.Kind {
		case engine.StageAdded:
			if has {
				continue // already ticked
			}
		default:
			if !has || st.RemovedFromProgram != nil {
				continue // not this package's stage, or already kept
			}
		}
		held := false
		if c.Kind == engine.StageRemoved {
			if held, err = stageHeld(ctx, tx, org, project, pkg, st.ID, out.Awarded); err != nil {
				return out, rec, err
			}
			rec.heldByID[st.ID] = held
		}
		out.Changes = append(out.Changes, StageSyncChange{StageChange: c, Action: engine.UpdateAction(out.Document, c, held), Held: held})
		addedOrRemoved = addedOrRemoved || c.Kind != engine.StageRenamed
	}
	out.Reordered = d.Reordered && len(rec.byProgram) > 1 && ticksReordered(rec.pkg.Stages, plan)
	out.Offer = len(out.Changes) > 0 || out.Reordered
	if !out.Offer {
		return out, rec, nil
	}
	if addedOrRemoved {
		if out.ProposalsFlagged, err = proposalsReceived(ctx, tx, org, project, pkg); err != nil {
			return out, rec, err
		}
	}
	return out, rec, nil
}

// ticksReordered is true when the package's linked stages are not in
// program order.
func ticksReordered(stages []procurement.Stage, plan *stagePlan) bool {
	last := -1
	for _, st := range stages {
		i := plan.index(st.ProgramStageID)
		if i < 0 {
			continue
		}
		if i < last {
			return true
		}
		last = i
	}
	return false
}

// ReadPackageStageSync is GET /packages/{pkg}/stage-sync.
func (s *Store) ReadPackageStageSync(ctx context.Context, org, project, pkg string) (StageSync, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return StageSync{}, err
	}
	defer tx.Rollback(ctx)
	plan, err := readStagePlan(ctx, tx, org, project, false)
	if err != nil {
		return StageSync{}, err
	}
	out, _, err := computePackageSync(ctx, tx, org, project, pkg, plan)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// SyncPackageStages applies Update or Keep to a package's stages (plan 7.6
// table). revision pins the program stage revision the user saw: a newer
// one is ErrVersionConflict. Update copies renames, leaves added stages as
// unticked boxes (or, on an awarded package, adds an unpriced cost line
// under the award), retires a removed stage unless something holds it, in
// which case it is marked "removed from program" and kept, and puts the
// stages in program order. Contract values, issued history and frozen
// baselines are never touched. Keep records the revision as acknowledged.
func (s *Store) SyncPackageStages(ctx context.Context, org, project, pkg, actor, choice string, revision int64) (StageSync, error) {
	if choice != StageSyncUpdate && choice != StageSyncKeep {
		return StageSync{}, ErrInvalidPackage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StageSync{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return StageSync{}, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return StageSync{}, err
	}
	plan, err := readStagePlan(ctx, tx, org, project, true)
	if err != nil {
		return StageSync{}, err
	}
	cur, rec, err := computePackageSync(ctx, tx, org, project, pkg, plan)
	if err != nil {
		return cur, err
	}
	if plan == nil {
		return cur, ErrNotFound
	}
	if revision != plan.Revision {
		return cur, ErrVersionConflict
	}
	if choice == StageSyncKeep {
		if _, err := tx.Exec(ctx, `UPDATE packages SET stage_acknowledged=$4 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, pkg, plan.Revision); err != nil {
			return cur, err
		}
	} else if err := s.updatePackageStages(ctx, tx, org, project, actor, cur, rec, plan); err != nil {
		return cur, err
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return cur, err
	}
	return s.ReadPackageStageSync(ctx, org, project, pkg)
}

func (s *Store) updatePackageStages(ctx context.Context, tx pgx.Tx, org, project, actor string, cur StageSync, rec pkgStageRecord, plan *stagePlan) error {
	pkg := rec.pkg.ID
	costLines := []string{}
	for _, c := range cur.Changes {
		st := rec.byProgram[c.ProgramStageID]
		switch c.Action {
		case engine.CopyLabel:
			if _, err := tx.Exec(ctx, `UPDATE package_stages SET label=$4,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, st.ID, c.NewLabel); err != nil {
				return err
			}
		case engine.Retire:
			if _, err := tx.Exec(ctx, `UPDATE package_stages SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, st.ID); err != nil {
				return err
			}
		case engine.MarkRemovedKeep:
			if _, err := tx.Exec(ctx, `UPDATE package_stages SET removed_from_program=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, st.ID); err != nil {
				return err
			}
		case engine.AddUnpricedLine:
			costLines = append(costLines, c.NewLabel)
		}
	}
	// Program order for the stages that point at it; any others after.
	if cur.Reordered || len(cur.Changes) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE package_stages s SET ordinal=o.n-1,version=s.version+CASE WHEN s.ordinal=o.n-1 THEN 0 ELSE 1 END
FROM (SELECT s2.id, row_number() OVER (ORDER BY COALESCE(array_position($4::text[],s2.program_stage_id::text),2147483647),s2.ordinal,s2.id) n
 FROM package_stages s2 WHERE s2.org_id=$1::uuid AND s2.project_id=$2::uuid AND s2.package_id=$3::uuid AND s2.retired_at IS NULL) o
WHERE s.org_id=$1::uuid AND s.id=o.id`, org, project, pkg, stageIDs(plan)); err != nil {
			return err
		}
	}
	raw, _ := json.Marshal(plan.recorded())
	if _, err := tx.Exec(ctx, `UPDATE packages SET stage_plan=$4::jsonb,stage_revision=$5 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, pkg, raw, plan.Revision); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE package_stages SET stage_revision=$4 WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND retired_at IS NULL AND program_stage_id IS NOT NULL`, org, project, pkg, plan.Revision); err != nil {
		return err
	}
	if len(costLines) > 0 {
		return s.addUnpricedAwardLines(ctx, tx, org, project, pkg, actor, costLines)
	}
	return nil
}

func stageIDs(plan *stagePlan) []string {
	out := make([]string, len(plan.Stages))
	for i, s := range plan.Stages {
		out[i] = s.ID
	}
	return out
}

// addUnpricedAwardLines adds, under an awarded package's group in the draft
// cost plan, one line per program stage added since: no value of any kind,
// so the contract stays exactly as the firm priced it. A frozen baseline is
// a separate version and is never touched; with no draft or no live award
// group there is nothing to add to.
func (s *Store) addUnpricedAwardLines(ctx context.Context, tx pgx.Tx, org, project, pkg, actor string, labels []string) error {
	var parent string
	err := tx.QueryRow(ctx, `SELECT parent_item_id::text FROM awards WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND undone_at IS NULL`, org, project, pkg).Scan(&parent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	p, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil || p.Status != "draft" {
		return err
	}
	var group *costs.Item
	children := 0
	for i, it := range p.Items {
		if it.ID == parent {
			group = &p.Items[i]
		}
		if it.ParentItemID == parent {
			children++
		}
	}
	if group == nil || group.Posting {
		return nil
	}
	code := group.Code
	if code == "" {
		code = "A"
	}
	for _, label := range labels {
		children++
		c := costs.Content{ParentItemID: parent, Code: clipText(code+"."+strconv.Itoa(children), 100), Label: clipText(label, 500), LineKind: "fee", PackageID: pkg,
			Posting: true, Origin: "user", Meaning: "stated", Rationale: "Stage added in the program after award; not priced by the firm."}
		if err := s.insertCostItem(ctx, tx, org, project, p.ID, actor, newID(), c); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE cost_plan_versions SET version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, p.ID); err != nil {
		return err
	}
	if err := BumpRevision(ctx, tx, org, project, "costs"); err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]any{"project_id": project, "plan_version_id": p.ID})
	_, err = appendEvent(ctx, s.q.WithTx(tx), org, "costs", "", string(raw))
	return err
}

// rfpStageRows are an RFP version's stage rows: the package's stages as
// the program names them, in program order, then any kept after their
// program stage was deleted because something holds them.
func rfpStageRows(stages []procurement.Stage, plan *stagePlan, held map[string]bool) []StageRow {
	rows := []StageRow{}
	byProgram := map[string]procurement.Stage{}
	for _, st := range stages {
		if st.ProgramStageID != "" {
			if _, dup := byProgram[st.ProgramStageID]; !dup {
				byProgram[st.ProgramStageID] = st
			}
		}
	}
	for _, ps := range plan.Stages {
		if _, ok := byProgram[ps.ID]; ok {
			rows = append(rows, StageRow{ProgramStageID: ps.ID, Label: ps.Label})
		}
	}
	for _, st := range stages {
		if st.ProgramStageID == "" {
			continue
		}
		if _, live := plan.stage(st.ProgramStageID); live {
			continue
		}
		if st.RemovedFromProgram != nil || held[st.ID] {
			rows = append(rows, StageRow{ProgramStageID: st.ProgramStageID, Label: st.Label, RemovedFromProgram: true})
		}
	}
	return rows
}

// recordRFPStages stamps a new RFP draft with the stage plan it is built
// from. Called when the draft is created, never on a refresh: after that
// only Update rewrites its rows.
func recordRFPStages(ctx context.Context, tx pgx.Tx, org string, r Report, versionID string) error {
	if r.Kind != "rfp" || r.PackageID == "" {
		return nil
	}
	plan, err := readStagePlan(ctx, tx, org, r.ProjectID, false)
	if err != nil || plan == nil {
		return err
	}
	pkg, err := readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true)
	if err != nil {
		return err
	}
	held, err := heldOffProgram(ctx, tx, org, r.ProjectID, pkg, plan)
	if err != nil {
		return err
	}
	return writeRFPStages(ctx, tx, org, versionID, plan, rfpStageRows(pkg.Stages, plan, held))
}

func writeRFPStages(ctx context.Context, tx pgx.Tx, org, versionID string, plan *stagePlan, rows []StageRow) error {
	planRaw, _ := json.Marshal(plan.recorded())
	rowsRaw, _ := json.Marshal(rows)
	tag, err := tx.Exec(ctx, `UPDATE report_versions SET stage_plan=$3::jsonb,stage_revision=$4,stage_rows=$5::jsonb WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, versionID, planRaw, plan.Revision, rowsRaw)
	if err == nil && tag.RowsAffected() != 1 {
		err = fmt.Errorf("%w: RFP draft %s", ErrNotFound, versionID)
	}
	return err
}

// heldOffProgram marks which of the package's stages whose program stage
// is gone something still holds.
func heldOffProgram(ctx context.Context, tx pgx.Tx, org, project string, pkg procurement.Package, plan *stagePlan) (map[string]bool, error) {
	held := map[string]bool{}
	awarded, err := packageAwarded(ctx, tx, org, project, pkg.ID)
	if err != nil {
		return held, err
	}
	for _, st := range pkg.Stages {
		if _, live := plan.stage(st.ProgramStageID); st.ProgramStageID == "" || live {
			continue
		}
		if held[st.ID], err = stageHeld(ctx, tx, org, project, pkg.ID, st.ID, awarded); err != nil {
			return held, err
		}
	}
	return held, nil
}

// rfpVersionStages is the stage record of the RFP version a sync works on.
type rfpVersionStages struct {
	id, status string
	plan       []byte
	revision   *int64
	rows       []byte
}

func readRFPVersionStages(ctx context.Context, tx pgx.Tx, org, versionID string) (rfpVersionStages, error) {
	v := rfpVersionStages{id: versionID}
	err := tx.QueryRow(ctx, `SELECT status,stage_plan,stage_revision,stage_rows FROM report_versions WHERE org_id=$1::uuid AND id=$2::uuid`, org, versionID).Scan(&v.status, &v.plan, &v.revision, &v.rows)
	return v, err
}

// computeReportSync is an RFP's stage sync inside tx: its current draft, or
// its latest issue when no draft is open.
func computeReportSync(ctx context.Context, tx pgx.Tx, org string, r Report, plan *stagePlan) (StageSync, procurement.Package, error) {
	out := emptySync(engine.DocDraftRFP)
	out.ReportID, out.PackageID = r.ID, r.PackageID
	var pkg procurement.Package
	if r.Kind != "rfp" || r.PackageID == "" {
		return out, pkg, nil
	}
	versionID := r.CurrentDraftVersionID
	if versionID == "" {
		history, _, err := reportHistory(ctx, tx, org, r.ID)
		if err != nil {
			return out, pkg, err
		}
		if len(history) == 0 {
			return out, pkg, nil
		}
		versionID, out.Document = history[0].ID, engine.DocIssuedRFP
	}
	out.VersionID = versionID
	v, err := readRFPVersionStages(ctx, tx, org, versionID)
	if err != nil {
		return out, pkg, err
	}
	if len(v.rows) > 0 {
		if err := json.Unmarshal(v.rows, &out.Rows); err != nil {
			return out, pkg, err
		}
	}
	var acked *int64
	if err := tx.QueryRow(ctx, `SELECT stage_acknowledged FROM reports WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID).Scan(&acked); err != nil {
		return out, pkg, err
	}
	if acked != nil {
		out.Acknowledged = *acked
	}
	if v.revision != nil {
		out.BuiltFrom = *v.revision
	}
	if plan == nil {
		return out, pkg, nil
	}
	out.ProgramExists, out.Revision = true, plan.Revision
	if pkg, err = readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true); err != nil {
		return out, pkg, err
	}
	if out.Awarded, err = packageAwarded(ctx, tx, org, r.ProjectID, r.PackageID); err != nil {
		return out, pkg, err
	}
	if v.plan == nil || !engine.NeedsOffer(out.BuiltFrom, out.Acknowledged, plan.Revision) {
		return out, pkg, nil
	}
	d, err := diffPlan(v.plan, plan)
	if err != nil {
		return out, pkg, err
	}
	inRows := map[string]bool{}
	for _, row := range out.Rows {
		inRows[row.ProgramStageID] = !row.RemovedFromProgram
	}
	ticked := map[string]procurement.Stage{}
	for _, st := range pkg.Stages {
		if st.ProgramStageID != "" {
			ticked[st.ProgramStageID] = st
		}
	}
	addedOrRemoved := false
	for _, c := range d.Changes {
		if c.Kind == engine.StageAdded {
			if _, ok := ticked[c.ProgramStageID]; !ok {
				continue // the package does not cover it: not a row
			}
		} else if !inRows[c.ProgramStageID] {
			continue
		}
		held := false
		if st, ok := ticked[c.ProgramStageID]; ok && c.Kind == engine.StageRemoved {
			if held, err = stageHeld(ctx, tx, org, r.ProjectID, r.PackageID, st.ID, out.Awarded); err != nil {
				return out, pkg, err
			}
		}
		out.Changes = append(out.Changes, StageSyncChange{StageChange: c, Action: engine.UpdateAction(out.Document, c, held), Held: held})
		addedOrRemoved = addedOrRemoved || c.Kind != engine.StageRenamed
	}
	out.Reordered = d.Reordered && len(out.Rows) > 1
	out.Offer = len(out.Changes) > 0 || out.Reordered
	if out.Offer && addedOrRemoved {
		if out.ProposalsFlagged, err = proposalsReceived(ctx, tx, org, r.ProjectID, r.PackageID); err != nil {
			return out, pkg, err
		}
	}
	return out, pkg, nil
}

// ReadReportStageSync is GET /reports/{r}/stage-sync. Another org's report
// is ErrNotFound.
func (s *Store) ReadReportStageSync(ctx context.Context, org, id string) (StageSync, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return StageSync{}, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return StageSync{}, err
	}
	plan, err := readStagePlan(ctx, tx, org, r.ProjectID, false)
	if err != nil {
		return StageSync{}, err
	}
	out, _, err := computeReportSync(ctx, tx, org, r, plan)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// SyncReportStages applies Update or Keep to an RFP (plan 7.6 table). On a
// draft, Update rewrites its stage rows from the package's stages as the
// program now names them; a stage something holds stays, marked "removed
// from program". When the latest version is issued, Update starts a new
// draft revision, built from the current plan; the issued version stays as
// history. Keep records the revision on the report. revision pins the
// program stage revision the user saw.
func (s *Store) SyncReportStages(ctx context.Context, org, id, actor, appBuild, choice string, revision int64) (StageSync, error) {
	if choice != StageSyncUpdate && choice != StageSyncKeep {
		return StageSync{}, ErrInvalidReport
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StageSync{}, err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, id)
	if err != nil {
		return StageSync{}, err
	}
	if err := lockProject(ctx, tx, org, r.ProjectID); err != nil {
		return StageSync{}, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return StageSync{}, err
	}
	// Re-read under the lock: a refresh may have moved the current draft.
	if r, err = readReport(ctx, tx, org, id); err != nil {
		return StageSync{}, err
	}
	plan, err := readStagePlan(ctx, tx, org, r.ProjectID, true)
	if err != nil {
		return StageSync{}, err
	}
	cur, pkg, err := computeReportSync(ctx, tx, org, r, plan)
	if err != nil {
		return cur, err
	}
	if plan == nil || cur.VersionID == "" || r.Kind != "rfp" {
		return cur, ErrNotFound
	}
	if revision != plan.Revision {
		return cur, ErrVersionConflict
	}
	switch {
	case choice == StageSyncKeep:
		if _, err := tx.Exec(ctx, `UPDATE reports SET stage_acknowledged=$3 WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID, plan.Revision); err != nil {
			return cur, err
		}
		if err := s.finishReportWrite(ctx, tx, org, r, r.CurrentDraftVersionID, "ready"); err != nil {
			return cur, err
		}
	case cur.Document == engine.DocIssuedRFP:
		// A new draft revision: the report's own assembly creates it and
		// records the current plan; the issued version is never touched.
		if err := tx.Rollback(ctx); err != nil {
			return cur, err
		}
		if _, err := s.EnsureReportDraft(ctx, org, r.ID, actor, appBuild); err != nil {
			return cur, err
		}
	default:
		held, err := heldOffProgram(ctx, tx, org, r.ProjectID, pkg, plan)
		if err != nil {
			return cur, err
		}
		if err := writeRFPStages(ctx, tx, org, cur.VersionID, plan, rfpStageRows(pkg.Stages, plan, held)); err != nil {
			return cur, err
		}
		if _, err := tx.Exec(ctx, `UPDATE report_versions SET version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft'`, org, cur.VersionID); err != nil {
			return cur, err
		}
		if err := s.finishReportWrite(ctx, tx, org, r, cur.VersionID, "ready"); err != nil {
			return cur, err
		}
	}
	return s.ReadReportStageSync(ctx, org, id)
}
