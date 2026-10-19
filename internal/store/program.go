package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
)

// The program store (plan docs/plans/2026-10-18-program.md sections 5 and
// 8). Tables hold inputs only; every read and write recomputes dates with
// the engine. No path here calls Jev.

// Program limits (plan section 5, "Limits").
const (
	MaxProgramRows  = 5000
	MaxProgramLinks = 20000
	MaxProgramOps   = 500
)

// ErrProgramExists refuses a second program for a project (one per project).
var ErrProgramExists = errors.New("program already exists")

// NullText is a nullable text column: "" is SQL NULL and JSON null.
type NullText string

func (t NullText) MarshalJSON() ([]byte, error) {
	if t == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(t))
}

func (t *NullText) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*t = ""
		return nil
	}
	var s string
	err := json.Unmarshal(b, &s)
	*t = NullText(s)
	return err
}

// Program is the programs row (plan 8.2 "program").
type Program struct {
	ID               string   `json:"id"`
	ProjectID        string   `json:"project_id"`
	Version          int64    `json:"version"`
	StartDate        NullText `json:"start_date"`
	StartOrigin      NullText `json:"start_origin"`
	SkeletonID       NullText `json:"skeleton_id"`
	LibraryVersion   string   `json:"library_version"`
	StageRevision    int64    `json:"stage_revision"`
	WorkingWeek      int      `json:"working_week"`
	CreatedBySurface string   `json:"created_by_surface"`
}

// ProgramActivity is one program_activities row: a stage, or an activity,
// milestone or hold point under a stage. Retired is never true in a read;
// it is part of the row images the undo log keeps.
type ProgramActivity struct {
	ID               string          `json:"id"`
	ParentID         NullText        `json:"parent_id"`
	Ordinal          int             `json:"ordinal"`
	Kind             string          `json:"kind"`
	Label            string          `json:"label"`
	CatalogueStageID NullText        `json:"catalogue_stage_id"`
	PackageID        NullText        `json:"package_id"`
	PackageStageID   NullText        `json:"package_stage_id"`
	WorkItemID       NullText        `json:"work_item_id"`
	FragmentID       NullText        `json:"fragment_id"`
	InterfaceID      NullText        `json:"interface_id"`
	Systems          []string        `json:"systems"`
	DurationDays     *int            `json:"duration_days"`
	DurationLow      *int            `json:"duration_low"`
	DurationHigh     *int            `json:"duration_high"`
	DurationOrigin   NullText        `json:"duration_origin"`
	DurationSource   json.RawMessage `json:"duration_source"`
	ConstraintKind   string          `json:"constraint_kind"`
	ConstraintDate   NullText        `json:"constraint_date"`
	ReviewStatus     string          `json:"review_status"`
	Origin           string          `json:"origin"`
	SourceKey        NullText        `json:"source_key"`
	DismissedReason  NullText        `json:"dismissed_reason"`
	Version          int64           `json:"version"`
	Retired          bool            `json:"retired,omitempty"`
}

// ProgramLink is one program_links row.
type ProgramLink struct {
	ID          string   `json:"id"`
	PredID      string   `json:"pred_id"`
	SuccID      string   `json:"succ_id"`
	Type        string   `json:"type"`
	LagDays     int      `json:"lag_days"`
	LagUnit     string   `json:"lag_unit"`
	Origin      string   `json:"origin"`
	InterfaceID NullText `json:"interface_id"`
	SourceKey   NullText `json:"source_key"`
	Version     int64    `json:"version"`
	Retired     bool     `json:"retired,omitempty"`
}

// ProgramCalendarDay is one non-working (or override) day.
type ProgramCalendarDay struct {
	Day    string `json:"day"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Origin string `json:"origin"`
}

type ProgramCalendar struct {
	Days []ProgramCalendarDay `json:"days"`
}

// ProgramState is the stored inputs plus the engine's output: the program,
// activities (live, outline order: each stage then its activities), live
// links, the calendar and `computed` (plan 8.2). The browser gets the raw
// inputs so its engine can preview a drag without a round trip.
type ProgramState struct {
	Program    Program           `json:"program"`
	Activities []ProgramActivity `json:"activities"`
	Links      []ProgramLink     `json:"links"`
	Calendar   ProgramCalendar   `json:"calendar"`
	Computed   engine.Output     `json:"computed"`
}

// Input is the engine input for this state.
func (st ProgramState) Input() engine.Input {
	return programInput(st.Program, st.Activities, st.Links, st.Calendar.Days)
}

// programInput builds the engine input. Activities must already be live and
// in outline order; links live.
func programInput(p Program, acts []ProgramActivity, links []ProgramLink, days []ProgramCalendarDay) engine.Input {
	in := engine.Input{
		Calendar:   engine.Calendar{Start: string(p.StartDate), WorkingWeek: p.WorkingWeek, Days: make([]engine.CalendarDay, len(days))},
		Activities: make([]engine.Activity, len(acts)),
		Links:      make([]engine.Link, len(links)),
	}
	for i, d := range days {
		in.Calendar.Days[i] = engine.CalendarDay{Day: d.Day, Kind: d.Kind, Label: d.Label}
	}
	for i := range acts {
		in.Activities[i] = engineActivity(&acts[i])
	}
	for i, l := range links {
		in.Links[i] = engine.Link{ID: l.ID, Pred: l.PredID, Succ: l.SuccID, Type: l.Type, Lag: l.LagDays, LagUnit: l.LagUnit}
	}
	return in
}

func engineActivity(a *ProgramActivity) engine.Activity {
	return engine.Activity{ID: a.ID, ParentID: string(a.ParentID), Ordinal: a.Ordinal, Kind: a.Kind, Label: a.Label,
		Duration: a.DurationDays, Low: a.DurationLow, High: a.DurationHigh,
		Constraint: a.ConstraintKind, ConstraintDate: string(a.ConstraintDate)}
}

// outline sorts live rows: stages by ordinal, each followed by its children
// by ordinal; ties by id so the order never depends on the database.
func outline(rows []ProgramActivity) []ProgramActivity {
	stages := []ProgramActivity{}
	kids := map[string][]ProgramActivity{}
	for _, a := range rows {
		if a.ParentID == "" {
			stages = append(stages, a)
		} else {
			kids[string(a.ParentID)] = append(kids[string(a.ParentID)], a)
		}
	}
	byOrdinal := func(s []ProgramActivity) {
		sort.SliceStable(s, func(i, j int) bool {
			if s[i].Ordinal != s[j].Ordinal {
				return s[i].Ordinal < s[j].Ordinal
			}
			return s[i].ID < s[j].ID
		})
	}
	byOrdinal(stages)
	out := make([]ProgramActivity, 0, len(rows))
	for _, s := range stages {
		out = append(out, s)
		k := kids[s.ID]
		byOrdinal(k)
		out = append(out, k...)
	}
	return out
}

type programQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

const programColumns = `id::text,project_id::text,version,COALESCE(to_char(start_date,'YYYY-MM-DD'),''),COALESCE(start_origin,''),COALESCE(skeleton_id,''),library_version,stage_revision,working_week,created_by_surface`

const activityColumns = `a.id::text,COALESCE(a.parent_id::text,''),a.ordinal,a.kind,a.label,COALESCE(a.catalogue_stage_id,''),COALESCE(a.package_id::text,''),COALESCE(a.package_stage_id::text,''),COALESCE(a.work_item_id::text,''),COALESCE(a.fragment_id,''),COALESCE(a.interface_id,''),a.systems,a.duration_days,a.duration_low,a.duration_high,COALESCE(a.duration_origin,''),a.duration_source::text,a.constraint_kind,COALESCE(to_char(a.constraint_date,'YYYY-MM-DD'),''),a.review_status,a.origin,COALESCE(a.source_key,''),COALESCE(a.dismissed_reason,''),a.version,a.retired_at IS NOT NULL`

const linkColumns = `l.id::text,l.pred_id::text,l.succ_id::text,l.type,l.lag_days,l.lag_unit,l.origin,COALESCE(l.interface_id,''),COALESCE(l.source_key,''),l.version,l.retired_at IS NOT NULL`

func scanProgram(r pgx.Row) (Program, error) {
	var p Program
	err := r.Scan(&p.ID, &p.ProjectID, &p.Version, (*string)(&p.StartDate), (*string)(&p.StartOrigin), (*string)(&p.SkeletonID), &p.LibraryVersion, &p.StageRevision, &p.WorkingWeek, &p.CreatedBySurface)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func scanActivities(rows pgx.Rows) ([]ProgramActivity, error) {
	defer rows.Close()
	out := []ProgramActivity{}
	for rows.Next() {
		var a ProgramActivity
		var source string
		if err := rows.Scan(&a.ID, (*string)(&a.ParentID), &a.Ordinal, &a.Kind, &a.Label, (*string)(&a.CatalogueStageID), (*string)(&a.PackageID), (*string)(&a.PackageStageID), (*string)(&a.WorkItemID), (*string)(&a.FragmentID), (*string)(&a.InterfaceID), &a.Systems, &a.DurationDays, &a.DurationLow, &a.DurationHigh, (*string)(&a.DurationOrigin), &source, &a.ConstraintKind, (*string)(&a.ConstraintDate), &a.ReviewStatus, &a.Origin, (*string)(&a.SourceKey), (*string)(&a.DismissedReason), &a.Version, &a.Retired); err != nil {
			return nil, err
		}
		if a.Systems == nil {
			a.Systems = []string{}
		}
		a.DurationSource = json.RawMessage(source)
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanLinks(rows pgx.Rows) ([]ProgramLink, error) {
	defer rows.Close()
	out := []ProgramLink{}
	for rows.Next() {
		var l ProgramLink
		if err := rows.Scan(&l.ID, &l.PredID, &l.SuccID, &l.Type, &l.LagDays, &l.LagUnit, &l.Origin, (*string)(&l.InterfaceID), (*string)(&l.SourceKey), &l.Version, &l.Retired); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanDays(rows pgx.Rows) ([]ProgramCalendarDay, error) {
	defer rows.Close()
	out := []ProgramCalendarDay{}
	for rows.Next() {
		var d ProgramCalendarDay
		if err := rows.Scan(&d.Day, &d.Kind, &d.Label, &d.Origin); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// readProgramInputs loads the program row and its live inputs in one round
// trip. lock takes the program row FOR UPDATE, which serialises writers: the
// version pin and the op sequence are both read under it.
func readProgramInputs(ctx context.Context, q programQuerier, org, project string, lock bool) (Program, []ProgramActivity, []ProgramLink, []ProgramCalendarDay, error) {
	forUpdate := ""
	if lock {
		forUpdate = " FOR UPDATE"
	}
	scope := `(SELECT id FROM programs WHERE org_id=$1::uuid AND project_id=$2::uuid)`
	b := &pgx.Batch{}
	b.Queue(`SELECT `+programColumns+` FROM programs WHERE org_id=$1::uuid AND project_id=$2::uuid`+forUpdate, org, project)
	b.Queue(`SELECT `+activityColumns+` FROM program_activities a WHERE a.org_id=$1::uuid AND a.program_id=`+scope+` AND a.retired_at IS NULL`, org, project)
	b.Queue(`SELECT `+linkColumns+` FROM program_links l WHERE l.org_id=$1::uuid AND l.program_id=`+scope+` AND l.retired_at IS NULL ORDER BY l.id`, org, project)
	b.Queue(`SELECT to_char(day,'YYYY-MM-DD'),kind,label,origin FROM program_calendar_days WHERE org_id=$1::uuid AND program_id=`+scope+` ORDER BY day`, org, project)
	br := q.SendBatch(ctx, b)
	defer br.Close()
	p, err := scanProgram(br.QueryRow())
	if err != nil {
		return p, nil, nil, nil, err
	}
	rows, err := br.Query()
	if err != nil {
		return p, nil, nil, nil, err
	}
	acts, err := scanActivities(rows)
	if err != nil {
		return p, nil, nil, nil, err
	}
	if rows, err = br.Query(); err != nil {
		return p, nil, nil, nil, err
	}
	links, err := scanLinks(rows)
	if err != nil {
		return p, nil, nil, nil, err
	}
	if rows, err = br.Query(); err != nil {
		return p, nil, nil, nil, err
	}
	days, err := scanDays(rows)
	if err != nil {
		return p, nil, nil, nil, err
	}
	return p, outline(acts), links, days, br.Close()
}

// computeState runs the engine over loaded inputs.
func computeState(p Program, acts []ProgramActivity, links []ProgramLink, days []ProgramCalendarDay) ProgramState {
	st := ProgramState{Program: p, Activities: acts, Links: links, Calendar: ProgramCalendar{Days: days}}
	st.Computed = engine.Compute(st.Input())
	return st
}

// LoadProgram returns the project's program with dates computed, or
// ErrNotFound when the project has none (or is not in the caller's org).
func (s *Store) LoadProgram(ctx context.Context, org, project string) (ProgramState, error) {
	p, acts, links, days, err := readProgramInputs(ctx, s.pool, org, project, false)
	if err != nil {
		return ProgramState{}, err
	}
	return computeState(p, acts, links, days), nil
}

// ProgramCreate is everything a new program starts with. The generator
// (WP-P5) decides the rows; the store only checks and writes them.
type ProgramCreate struct {
	Surface        string // program, procurement or cost_plan
	SkeletonID     string
	LibraryVersion string
	StartDate      string // "" leaves the program relative
	StartOrigin    string
	WorkingWeek    int // 0 is Monday to Friday
	// Rows in outline order. A row's ParentKey names an earlier stage row;
	// ordinals are assigned in the order given, per parent.
	Rows     []ProgramRowInput
	Links    []ProgramLinkInput
	Calendar []ProgramCalendarDay
	// PackageStages maps a package_stages id to the Key of the program stage
	// it belongs to; each gets program_stage_id and the stage revision.
	PackageStages map[string]string
}

// ProgramRowInput is one row to create. Key is the caller's reference,
// used by ParentKey, links and PackageStages; it is not stored.
type ProgramRowInput struct {
	Key              string
	ParentKey        string
	Kind             string
	Label            string
	CatalogueStageID string
	PackageID        string
	PackageStageID   string
	WorkItemID       string
	FragmentID       string
	InterfaceID      string
	Systems          []string
	DurationDays     *int
	DurationLow      *int
	DurationHigh     *int
	DurationOrigin   string
	DurationSource   json.RawMessage
	ConstraintKind   string
	ConstraintDate   string
	ReviewStatus     string
	Origin           string
	SourceKey        string
}

// ProgramLinkInput joins two rows by Key.
type ProgramLinkInput struct {
	PredKey, SuccKey string
	Type             string
	LagDays          int
	LagUnit          string
	Origin           string
	InterfaceID      string
	SourceKey        string
}

// ProgramCreated returns the new ids: the program and each row Key's id.
type ProgramCreated struct {
	ProgramID string            `json:"program_id"`
	IDs       map[string]string `json:"ids"`
}

// CreateProgram writes a new program in its own transaction.
func (s *Store) CreateProgram(ctx context.Context, org, project string, in ProgramCreate) (ProgramCreated, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramCreated{}, err
	}
	defer tx.Rollback(ctx)
	out, err := CreateProgramTx(ctx, tx, org, project, in)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// CreateProgramTx writes a new program inside the caller's transaction, so
// generation can read its inputs and write the program atomically. It
// returns ErrProgramExists when the project already has one, without
// aborting the transaction.
func CreateProgramTx(ctx context.Context, tx pgx.Tx, org, project string, in ProgramCreate) (ProgramCreated, error) {
	out := ProgramCreated{IDs: map[string]string{}}
	p, acts, links, err := buildProgram(in)
	if err != nil {
		return out, err
	}
	for k, a := range acts {
		out.IDs[in.Rows[k].Key] = a.ID
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return out, err
	}
	if !exists {
		return out, ErrNotFound
	}
	err = tx.QueryRow(ctx, `INSERT INTO programs(org_id,id,project_id,start_date,start_origin,skeleton_id,library_version,working_week,created_by_surface)
VALUES($1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::date,NULLIF($5,''),NULLIF($6,''),$7,$8,$9)
ON CONFLICT (org_id,project_id) DO NOTHING RETURNING id::text`, org, newID(), project, string(p.StartDate), string(p.StartOrigin), string(p.SkeletonID), p.LibraryVersion, p.WorkingWeek, p.CreatedBySurface).Scan(&p.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrProgramExists
	}
	if err != nil {
		return out, err
	}
	out.ProgramID = p.ID
	if err := insertActivities(ctx, tx, org, p.ID, acts); err != nil {
		return out, err
	}
	if err := insertLinks(ctx, tx, org, p.ID, links); err != nil {
		return out, err
	}
	if len(in.Calendar) > 0 {
		raw, err := json.Marshal(in.Calendar)
		if err != nil {
			return out, err
		}
		if _, err := tx.Exec(ctx, dayInsert, org, p.ID, raw); err != nil {
			return out, err
		}
	}
	if len(in.PackageStages) > 0 {
		ids, stages := make([]string, 0, len(in.PackageStages)), make([]string, 0, len(in.PackageStages))
		for ps, key := range in.PackageStages {
			id, ok := out.IDs[key]
			if !ok || acts[indexOfKey(in.Rows, key)].Kind != engine.KindStage {
				return out, fmt.Errorf("%w: package stage %s points at %q, which is not a program stage", ErrInvalidInput, ps, key)
			}
			ids, stages = append(ids, ps), append(stages, id)
		}
		// The link is not package content, so the stage's version stays: an
		// open procurement form or a proposal undo pinned to it still holds.
		tag, err := tx.Exec(ctx, `UPDATE package_stages s SET program_stage_id=m.stage::uuid,stage_revision=1
FROM unnest($3::text[],$4::text[]) m(id,stage) WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND s.id::text=m.id AND s.retired_at IS NULL`, org, project, ids, stages)
		if err != nil {
			return out, err
		}
		if tag.RowsAffected() != int64(len(ids)) {
			return out, fmt.Errorf("%w: a package stage is not live in this project", ErrNotFound)
		}
	}
	return out, nil
}

func indexOfKey(rows []ProgramRowInput, key string) int {
	for i, r := range rows {
		if r.Key == key {
			return i
		}
	}
	return -1
}

// buildProgram checks a create request and turns it into rows with ids.
func buildProgram(in ProgramCreate) (Program, []ProgramActivity, []ProgramLink, error) {
	fail := func(format string, args ...any) (Program, []ProgramActivity, []ProgramLink, error) {
		return Program{}, nil, nil, fmt.Errorf("%w: "+format, append([]any{ErrInvalidInput}, args...)...)
	}
	p := Program{StartDate: NullText(in.StartDate), StartOrigin: NullText(in.StartOrigin), SkeletonID: NullText(in.SkeletonID),
		LibraryVersion: in.LibraryVersion, StageRevision: 1, WorkingWeek: in.WorkingWeek, CreatedBySurface: in.Surface, Version: 1}
	if p.WorkingWeek == 0 {
		p.WorkingWeek = engine.DefaultWorkingWeek
	}
	switch {
	case in.Surface != "program" && in.Surface != "procurement" && in.Surface != "cost_plan":
		return fail("surface %q", in.Surface)
	case strings.TrimSpace(in.LibraryVersion) == "":
		return fail("library version is required")
	case p.WorkingWeek < 1 || p.WorkingWeek > 127:
		return fail("working week %d", p.WorkingWeek)
	case in.StartDate != "" && !validDate(in.StartDate):
		return fail("start date %q", in.StartDate)
	case (in.StartDate == "") != (in.StartOrigin == "") || (in.StartOrigin != "" && in.StartOrigin != "user" && in.StartOrigin != "document" && in.StartOrigin != "profile"):
		return fail("start origin %q", in.StartOrigin)
	case len(in.Rows) > MaxProgramRows || len(in.Links) > MaxProgramLinks:
		return fail("program is over its limits")
	}
	keys := map[string]int{}
	ordinals := map[string]int{}
	acts := make([]ProgramActivity, len(in.Rows))
	for k, r := range in.Rows {
		if r.Key == "" {
			return fail("row %d has no key", k)
		}
		if _, dup := keys[r.Key]; dup {
			return fail("row key %q twice", r.Key)
		}
		a := ProgramActivity{ID: newID(), Kind: r.Kind, Label: strings.TrimSpace(r.Label), CatalogueStageID: NullText(r.CatalogueStageID),
			PackageID: NullText(r.PackageID), PackageStageID: NullText(r.PackageStageID), WorkItemID: NullText(r.WorkItemID),
			FragmentID: NullText(r.FragmentID), InterfaceID: NullText(r.InterfaceID), Systems: r.Systems,
			DurationDays: r.DurationDays, DurationLow: r.DurationLow, DurationHigh: r.DurationHigh, DurationOrigin: NullText(r.DurationOrigin),
			DurationSource: r.DurationSource, ConstraintKind: r.ConstraintKind, ConstraintDate: NullText(r.ConstraintDate),
			ReviewStatus: r.ReviewStatus, Origin: r.Origin, SourceKey: NullText(r.SourceKey), Version: 1}
		if a.Systems == nil {
			a.Systems = []string{}
		}
		if len(a.DurationSource) == 0 {
			a.DurationSource = json.RawMessage(`{}`)
		}
		if a.ConstraintKind == "" {
			a.ConstraintKind = engine.ConstraintASAP
		}
		if r.ParentKey != "" {
			pi, ok := keys[r.ParentKey]
			if !ok || acts[pi].Kind != engine.KindStage {
				return fail("row %q: parent %q is not an earlier stage", r.Key, r.ParentKey)
			}
			a.ParentID = NullText(acts[pi].ID)
		}
		if msg := checkRow(&a); msg != "" {
			return fail("row %q: %s", r.Key, msg)
		}
		if a.Origin != "library" && a.Origin != "user" && a.Origin != "document" {
			return fail("row %q: origin %q", r.Key, a.Origin)
		}
		if a.ReviewStatus != "proposed" && a.ReviewStatus != "accepted" {
			return fail("row %q: review status %q", r.Key, a.ReviewStatus)
		}
		a.Ordinal = ordinals[string(a.ParentID)]
		ordinals[string(a.ParentID)]++
		keys[r.Key] = k
		acts[k] = a
	}
	links := make([]ProgramLink, len(in.Links))
	pairs := map[[2]string]bool{}
	for k, l := range in.Links {
		pi, okp := keys[l.PredKey]
		si, oks := keys[l.SuccKey]
		if !okp || !oks {
			return fail("link %d joins an unknown row", k)
		}
		link := ProgramLink{ID: newID(), PredID: acts[pi].ID, SuccID: acts[si].ID, Type: l.Type, LagDays: l.LagDays, LagUnit: l.LagUnit,
			Origin: l.Origin, InterfaceID: NullText(l.InterfaceID), SourceKey: NullText(l.SourceKey), Version: 1}
		if link.Type == "" {
			link.Type = engine.LinkFS
		}
		if link.LagUnit == "" {
			link.LagUnit = engine.LagWorking
		}
		pair := [2]string{link.PredID, link.SuccID}
		if pairs[pair] {
			return fail("link %d: %s", k, engine.MsgLinkDuplicate)
		}
		pairs[pair] = true
		if msg := checkLinkShape(link); msg != "" {
			return fail("link %d: %s", k, msg)
		}
		links[k] = link
	}
	for _, d := range in.Calendar {
		if !validDate(d.Day) || !validDayKind(d.Kind) || !validLabel(d.Label) || (d.Origin != "library" && d.Origin != "user") {
			return fail("calendar day %q", d.Day)
		}
	}
	input := programInput(p, outline(acts), links, in.Calendar)
	if err := engine.Validate(input); err != nil {
		return fail("%v", err)
	}
	if c := engine.FindCycle(input); c != nil {
		return fail("links make a loop: %s", engine.CycleWords(input, c))
	}
	return p, acts, links, nil
}

// checkRow holds the row rules the schema also checks, so a bad request is
// refused with a message instead of a constraint error.
func checkRow(a *ProgramActivity) string {
	switch a.Kind {
	case engine.KindStage:
		if a.ParentID != "" {
			return "a stage has no parent"
		}
		if a.CatalogueStageID == "" {
			return "a stage needs a catalogue stage id"
		}
	case engine.KindActivity, engine.KindMilestone, engine.KindHoldPoint:
		if a.ParentID == "" {
			return "every activity sits under a stage"
		}
		if a.CatalogueStageID != "" {
			return "only stages carry a catalogue stage id"
		}
	default:
		return fmt.Sprintf("kind %q", a.Kind)
	}
	if !validLabel(a.Label) {
		return "label must be 1 to 200 characters"
	}
	if a.DurationDays != nil && (*a.DurationDays < 0 || *a.DurationDays > engine.MaxDuration) {
		return "duration must be 0 to 3,650 working days"
	}
	if (a.DurationLow == nil) != (a.DurationHigh == nil) || (a.DurationLow != nil && (*a.DurationLow > *a.DurationHigh || *a.DurationLow < 0 || *a.DurationHigh > engine.MaxDuration)) {
		return "low and high durations come together, low first"
	}
	if a.Kind == engine.KindMilestone && (a.DurationDays == nil || *a.DurationDays != 0) {
		return "a milestone takes 0 days"
	}
	if a.DurationOrigin != "" && a.DurationOrigin != "user" && a.DurationOrigin != "document" && a.DurationOrigin != "profile" && a.DurationOrigin != "library" && a.DurationOrigin != "calculation" {
		return fmt.Sprintf("duration origin %q", a.DurationOrigin)
	}
	switch a.ConstraintKind {
	case engine.ConstraintASAP:
		if a.ConstraintDate != "" {
			return "as soon as possible takes no date"
		}
	case engine.ConstraintSNET, engine.ConstraintMSO:
		if !validDate(string(a.ConstraintDate)) {
			return "constraint date must be YYYY-MM-DD"
		}
	default:
		return fmt.Sprintf("constraint %q", a.ConstraintKind)
	}
	if a.DismissedReason != "" && a.Kind != engine.KindHoldPoint {
		return "only hold points carry a dismissal reason"
	}
	return ""
}

func checkLinkShape(l ProgramLink) string {
	switch {
	case l.PredID == l.SuccID:
		return engine.MsgLinkSelf
	case l.Type != engine.LinkFS && l.Type != engine.LinkSS && l.Type != engine.LinkFF && l.Type != engine.LinkSF:
		return engine.MsgLinkType
	case l.LagUnit != engine.LagWorking && l.LagUnit != engine.LagCalendar:
		return engine.MsgLinkLagUnit
	case l.LagDays < -365 || l.LagDays > 365:
		return engine.MsgLinkLag
	case l.Origin == "chain" && (l.Type != engine.LinkFS || l.LagDays != 0):
		return "a chain link is FS with no lag"
	}
	switch l.Origin {
	case "chain", "library", "user", "document", "interface":
		return ""
	}
	return fmt.Sprintf("link origin %q", l.Origin)
}

func validLabel(s string) bool {
	n := len([]rune(strings.TrimSpace(s)))
	return n >= 1 && n <= 200
}

func validDayKind(k string) bool {
	switch k {
	case engine.DayPublicHoliday, engine.DayRDO, engine.DayShutdown, engine.DayWorkingOverride:
		return true
	}
	return false
}

// validDate accepts a strict, real YYYY-MM-DD date.
func validDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i != 4 && i != 7 && (c < '0' || c > '9') {
			return false
		}
	}
	return engine.Validate(engine.Input{Calendar: engine.Calendar{Start: s}}) == nil
}

// activityRows is the JSON the bulk writers hand PostgreSQL: one array of
// rows, read back with jsonb_populate_record so one statement writes them
// all, whatever their number.
func activityRows(acts []ProgramActivity) ([]byte, error) { return json.Marshal(acts) }

const activityInsert = `INSERT INTO program_activities(org_id,program_id,id,parent_id,ordinal,kind,label,catalogue_stage_id,package_id,package_stage_id,work_item_id,fragment_id,interface_id,systems,duration_days,duration_low,duration_high,duration_origin,duration_source,constraint_kind,constraint_date,review_status,origin,source_key,dismissed_reason,retired_at,version)
SELECT $1::uuid,$2::uuid,r.id,r.parent_id,r.ordinal,r.kind,r.label,r.catalogue_stage_id,r.package_id,r.package_stage_id,r.work_item_id,r.fragment_id,r.interface_id,r.systems,r.duration_days,r.duration_low,r.duration_high,r.duration_origin,r.duration_source,r.constraint_kind,r.constraint_date,r.review_status,r.origin,r.source_key,r.dismissed_reason,CASE WHEN (e->>'retired')::boolean THEN now() END,r.version
FROM jsonb_array_elements($3::jsonb) e, LATERAL jsonb_populate_record(NULL::program_activities,e) r`

const linkInsert = `INSERT INTO program_links(org_id,program_id,id,pred_id,succ_id,type,lag_days,lag_unit,origin,interface_id,source_key,retired_at,version)
SELECT $1::uuid,$2::uuid,r.id,r.pred_id,r.succ_id,r.type,r.lag_days,r.lag_unit,r.origin,r.interface_id,r.source_key,CASE WHEN (e->>'retired')::boolean THEN now() END,r.version
FROM jsonb_array_elements($3::jsonb) e, LATERAL jsonb_populate_record(NULL::program_links,e) r`

const dayInsert = `INSERT INTO program_calendar_days(org_id,program_id,day,kind,label,origin)
SELECT $1::uuid,$2::uuid,r.day,r.kind,r.label,r.origin FROM jsonb_populate_recordset(NULL::program_calendar_days,$3::jsonb) r`

func insertActivities(ctx context.Context, tx pgx.Tx, org, programID string, acts []ProgramActivity) error {
	if len(acts) == 0 {
		return nil
	}
	raw, err := activityRows(acts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, activityInsert, org, programID, raw)
	return err
}

func insertLinks(ctx context.Context, tx pgx.Tx, org, programID string, links []ProgramLink) error {
	if len(links) == 0 {
		return nil
	}
	raw, err := json.Marshal(links)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, linkInsert, org, programID, raw)
	return err
}
