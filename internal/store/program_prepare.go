package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// Program generation from the database (plan docs/plans/2026-10-18-program.md
// sections 7.1 to 7.3): reading the generation inputs, prepare, the extras a
// read returns beside the state (checks, the one question, the start
// suggestion, the skeleton and pending counts) and the question answer. All
// of it is code over stored values; no path here calls Jev.

// ValidProgramSurface reports a surface that may create the program (7.6).
func ValidProgramSurface(s string) bool {
	return s == "program" || s == "procurement" || s == "cost_plan"
}

// ProgramSkeleton names the stage pattern the program was generated from
// and why it was chosen, for the completion header. Reason is empty when
// the Profile now selects a different skeleton: the stored program keeps
// the one it was made from until the user adopts a change in the review.
type ProgramSkeleton struct {
	ID     NullText `json:"id"`
	Label  string   `json:"label"`
	Reason string   `json:"reason"`
}

// ProgramExtras are the computed parts of a read beyond the stored state
// (plan 8.2). None of it is stored except the question dismissals.
type ProgramExtras struct {
	Checks   []engine.Check   `json:"checks"`
	Question *engine.Question `json:"question"`
	// StartSuggestion is the consent date, offered as a one-tap start while
	// the program has no start date; never applied automatically (7.2.9).
	StartSuggestion NullText        `json:"start_suggestion"`
	Skeleton        ProgramSkeleton `json:"skeleton"`
	ReadingsPending int             `json:"readings_pending"`
	ChangesPending  int             `json:"changes_pending"`
}

// ProgramView is the full response of every program route: the state, the
// extras and, after a write, what the batch did. Feedback is nil (and its
// fields absent) on a read.
type ProgramView struct {
	ProgramState
	ProgramExtras
	*ProgramFeedback
}

// errNoProgramChange ends an answer that changes no program row without
// writing an empty batch, so undo never lands on a step that did nothing.
var errNoProgramChange = errors.New("no program change")

// programInputs is GenInput plus the projection hash of each whole-project
// profile row, which a question dismissal is pinned to.
type programInputs struct {
	gi     engine.GenInput
	hashes map[string]string
	years  []int // years with a reviewed calendar for the project's state
	cal    []ProgramCalendarDay
}

const programProfileSQL = `SELECT r.key,COALESCE(r.value,''),r.band,r.origin,r.meaning,r.review_status,r.value_state,COALESCE(r.projection_hash,''),p.kind='whole'
FROM profile_rows r JOIN project_parts p ON p.org_id=r.org_id AND p.site_id=r.site_id AND p.id=r.part_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND (r.key='hdr.work_type' OR (p.kind='whole' AND (r.key LIKE 'det.%' OR r.key LIKE 'hdr.cond.%' OR r.key LIKE 'plan.%' OR r.key LIKE 'fact.%' OR r.key LIKE 'sys.%.presence' OR r.key LIKE 'sys.%.existing')))
ORDER BY r.key,r.part_id`

// Work items in the Profile's order (readWorkItems); groups are containers
// and bring no work of their own, as in the cost plan suggestions.
const programWorkSQL = `SELECT id::text,system_id,part_id::text,action,title,layout_change FROM work_items
WHERE org_id=$1::uuid AND project_id=$2::uuid AND inclusion='included' AND retired_at IS NULL AND NOT is_group ORDER BY part_id,system_id,id`

// Live packages that are not cancelled, in Procurement order (readPackages),
// each with its live stages in ordinal order.
const programPackageSQL = `SELECT p.id::text,p.title,p.kind,COALESCE(p.discipline_id,''),COALESCE(s.id::text,''),COALESCE(s.stage_id,''),COALESCE(s.label,'')
FROM packages p LEFT JOIN package_stages s ON s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id AND s.retired_at IS NULL
WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.retired_at IS NULL AND p.lifecycle_status<>'cancelled'
ORDER BY p.title,p.id,s.ordinal,s.label,s.id`

// readProgramGenInput reads plan 7.1's inputs in one round trip. The
// construction budget is not read: it only feeds the time_cost basis, which
// stays unknown without a cost index (owner answer Q7 defers both).
func readProgramGenInput(ctx context.Context, q programQuerier, org, project string, cat *knowledge.Catalog) (programInputs, error) {
	in := programInputs{hashes: map[string]string{}, gi: engine.GenInput{Values: map[string]string{}, Conditions: map[string]string{},
		Planning: map[string]string{}, Present: map[string]bool{}, Existing: map[string]bool{}}}
	b := &pgx.Batch{}
	b.Queue(programProfileSQL, org, project)
	b.Queue(programWorkSQL, org, project)
	b.Queue(programPackageSQL, org, project)
	br := q.SendBatch(ctx, b)
	defer br.Close()

	rows, err := br.Query()
	if err != nil {
		return in, err
	}
	workTypes := map[string]bool{}
	present, existing := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var key, value, band, origin, meaning, review, state, hash string
		var whole bool
		if err := rows.Scan(&key, &value, &band, &origin, &meaning, &review, &state, &hash, &whole); err != nil {
			rows.Close()
			return in, err
		}
		if whole {
			in.hashes[key] = hash
		}
		value = strings.TrimSpace(value)
		applied := appliedProgramRow(band, value, state, review, origin, meaning)
		switch {
		case key == "hdr.work_type":
			// Every part's applied work type, as the package defaults read it.
			if applied && !workTypes[value] {
				workTypes[value] = true
				in.gi.WorkTypes = append(in.gi.WorkTypes, value)
			}
		case !whole:
		case strings.HasPrefix(key, knowledge.PlanningPrefix):
			// A planning value is a forecast or an assumption by nature, so
			// its meaning and origin do not exclude it; favourable values
			// already need the user's own word (L276).
			if value != "" && state != "unknown" && state != "cleared" && review != "superseded" {
				in.gi.Planning[strings.TrimPrefix(key, knowledge.PlanningPrefix)] = value
			}
		case !applied:
		case strings.HasSuffix(key, ".assertion"):
		case strings.HasPrefix(key, "det."):
			in.gi.Values[strings.TrimPrefix(key, "det.")] = value
		case strings.HasPrefix(key, "hdr.cond."):
			in.gi.Conditions[strings.TrimPrefix(key, "hdr.cond.")] = value
		case strings.HasPrefix(key, "fact."):
			setProgramFact(&in.gi.Facts, strings.TrimPrefix(key, "fact."), value)
		case strings.HasPrefix(key, "sys.") && strings.HasSuffix(key, ".presence"):
			setSystemState(present, strings.TrimSuffix(strings.TrimPrefix(key, "sys."), ".presence"), value)
		case strings.HasPrefix(key, "sys.") && strings.HasSuffix(key, ".existing"):
			setSystemState(existing, strings.TrimSuffix(strings.TrimPrefix(key, "sys."), ".existing"), value)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}

	if rows, err = br.Query(); err != nil {
		return in, err
	}
	for rows.Next() {
		var it engine.WorkItem
		var part string
		if err := rows.Scan(&it.ID, &it.System, &part, &it.Action, &it.Title, &it.LayoutChange); err != nil {
			rows.Close()
			return in, err
		}
		it.Part = part
		in.gi.WorkItems = append(in.gi.WorkItems, it)
		inferSystemState(present, existing, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}

	if rows, err = br.Query(); err != nil {
		return in, err
	}
	for rows.Next() {
		var id, title, kind, discipline, stageID, catalogue, label string
		if err := rows.Scan(&id, &title, &kind, &discipline, &stageID, &catalogue, &label); err != nil {
			rows.Close()
			return in, err
		}
		if n := len(in.gi.Packages); n == 0 || in.gi.Packages[n-1].ID != id {
			in.gi.Packages = append(in.gi.Packages, engine.Package{ID: id, Title: title, Kind: kind, DisciplineID: discipline})
		}
		if stageID != "" {
			p := &in.gi.Packages[len(in.gi.Packages)-1]
			p.Stages = append(p.Stages, engine.PackageStage{ID: stageID, CatalogueStageID: catalogue, Label: label})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}
	in.gi.Present, in.gi.Existing = present, existing
	in.cal, in.years = libraryCalendar(cat, in.gi.Values["state"])
	return in, br.Close()
}

// appliedProgramRow is the profile's applied-value rule (profile
// appliedWorkRow): a stated value from the user or a read document, never a
// guess, an assumption or an explicit unknown.
func appliedProgramRow(band, value, state, review, origin, meaning string) bool {
	return (band == "user" || band == "amber" || band == "green") && value != "" && state != "unknown" && state != "cleared" &&
		review != "superseded" && origin != "assumption" && (meaning == "" || meaning == "stated")
}

// setSystemState records a system's presence row: in the building, or
// stated as not in it. "not_stated" says nothing.
func setSystemState(m map[string]bool, system, value string) {
	switch value {
	case "included", "included_by_others", "present":
		m[system] = true
	case "not_included", "absent":
		m[system] = false
	}
}

// inferSystemState adds what an included work item states about its system
// where no Profile row says otherwise: a new, replaced, altered, upgraded,
// repaired or retained system is in the completed building, a removed one
// is not; every action but new and investigate works on something already
// on site. Investigate states nothing.
func inferSystemState(present, existing map[string]bool, it engine.WorkItem) {
	setIfUnset := func(m map[string]bool, v bool) {
		if _, ok := m[it.System]; !ok {
			m[it.System] = v
		}
	}
	switch it.Action {
	case "new":
		setIfUnset(present, true)
	case "replace", "alter", "upgrade", "repair", "retain":
		setIfUnset(present, true)
		setIfUnset(existing, true)
	case "remove":
		setIfUnset(present, false)
		setIfUnset(existing, true)
	}
}

// setProgramFact keeps the facts generation and the checks read. Dates are
// pre-parsed text copied from a document, so they are normalised to
// YYYY-MM-DD; a date code cannot read stays unknown and its check is off.
func setProgramFact(f *engine.Facts, id, value string) {
	switch id {
	case "consent_date":
		f.ConsentDate = isoDate(value)
	case "consent_lapse_date":
		f.ConsentLapseDate = isoDate(value)
	case "defects_liability_period":
		f.DefectsLiabilityPeriod = value
	case "contract_form":
		f.ContractForm = value
	}
}

// isoDate reads the date layouts Australian documents use, day first
// (plan 13.1: ambiguous day-month dates assume day/month). Anything else is
// unknown.
func isoDate(s string) string {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "2 January 2006", "2 Jan 2006", "January 2, 2006", "2/1/2006", "02/01/2006", "2.1.2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// libraryCalendar is the program calendar's starting copy (plan 4.4):
// reviewed library calendar days for the project's state, one row per day,
// and the years they cover. Draft calendars are never used at runtime; with
// no reviewed calendar the program runs on weekends only and the
// calendar_gap check says so.
func libraryCalendar(cat *knowledge.Catalog, state string) ([]ProgramCalendarDay, []int) {
	if cat == nil || state == "" {
		return nil, nil
	}
	var days []ProgramCalendarDay
	var years []int
	seen := map[string]bool{}
	add := func(day, kind, label string) {
		if seen[day] || !validDate(day) || !validDayKind(kind) || !validLabel(label) {
			return
		}
		seen[day] = true
		days = append(days, ProgramCalendarDay{Day: day, Kind: kind, Label: strings.TrimSpace(label), Origin: "library"})
	}
	for _, c := range cat.Calendars() {
		if c.State != state || c.Status != "reviewed" {
			continue
		}
		years = append(years, c.Year)
		for _, d := range c.Days {
			if d.Date != "" {
				add(d.Date, d.Kind, d.Label)
				continue
			}
			from, err1 := time.Parse("2006-01-02", d.From)
			to, err2 := time.Parse("2006-01-02", d.To)
			if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
				continue
			}
			for t := from; !t.After(to); t = t.AddDate(0, 0, 1) {
				add(t.Format("2006-01-02"), d.Kind, d.Label)
			}
		}
	}
	return days, years
}

// programCreate turns a generated plan into the store's create input.
func programCreate(plan engine.Plan, surface string, cal []ProgramCalendarDay) (ProgramCreate, error) {
	in := ProgramCreate{Surface: surface, SkeletonID: plan.SkeletonID, LibraryVersion: plan.LibraryVersion, Calendar: cal,
		PackageStages: map[string]string{}}
	for _, r := range plan.Rows {
		source := json.RawMessage(`{}`)
		if len(r.DurationSource) > 0 {
			raw, err := json.Marshal(r.DurationSource)
			if err != nil {
				return in, err
			}
			source = raw
		}
		in.Rows = append(in.Rows, ProgramRowInput{Key: r.SourceKey, ParentKey: r.ParentKey, Kind: r.Kind, Label: r.Label,
			CatalogueStageID: r.CatalogueStageID, PackageID: r.PackageID, PackageStageID: r.PackageStageID, WorkItemID: r.WorkItemID,
			FragmentID: r.FragmentID, InterfaceID: r.InterfaceID, Systems: r.Systems, DurationDays: r.Duration, DurationLow: r.Low,
			DurationHigh: r.High, DurationOrigin: r.DurationOrigin, DurationSource: source, ReviewStatus: r.ReviewStatus,
			Origin: r.Origin, SourceKey: r.SourceKey})
	}
	for _, l := range plan.Links {
		in.Links = append(in.Links, ProgramLinkInput{PredKey: l.PredKey, SuccKey: l.SuccKey, Type: l.Type, LagDays: l.Lag,
			LagUnit: l.LagUnit, Origin: l.Origin, InterfaceID: l.InterfaceID, SourceKey: l.SourceKey})
	}
	for _, m := range plan.StageMap {
		in.PackageStages[m.PackageStageID] = m.StageKey
	}
	return in, nil
}

// PrepareProgram returns the project's program, creating it first when none
// exists (plan 7.2): generation over the stored inputs and the insert run in
// one transaction, and each package stage gets its program_stage_id. It is
// idempotent: an existing program is returned unchanged with created false.
// Two concurrent prepares create one program; the loser returns it.
func (s *Store) PrepareProgram(ctx context.Context, org, project, surface string, cat *knowledge.Catalog) (ProgramState, bool, error) {
	if !ValidProgramSurface(surface) {
		return ProgramState{}, false, fmt.Errorf("%w: surface %q", ErrInvalidInput, surface)
	}
	if cat == nil {
		return ProgramState{}, false, errors.New("program generation needs the knowledge catalogue")
	}
	st, err := s.LoadProgram(ctx, org, project)
	if err == nil || !errors.Is(err, ErrNotFound) {
		return st, false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramState{}, false, err
	}
	defer tx.Rollback(ctx)
	in, err := readProgramGenInput(ctx, tx, org, project, cat)
	if err != nil {
		return ProgramState{}, false, err
	}
	plan, err := engine.Generate(cat, in.gi)
	if err != nil {
		return ProgramState{}, false, err
	}
	create, err := programCreate(plan, surface, in.cal)
	if err != nil {
		return ProgramState{}, false, err
	}
	if _, err := CreateProgramTx(ctx, tx, org, project, create); errors.Is(err, ErrProgramExists) {
		_ = tx.Rollback(ctx)
		st, err := s.LoadProgram(ctx, org, project)
		return st, false, err
	} else if err != nil {
		return ProgramState{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramState{}, false, err
	}
	st, err = s.LoadProgram(ctx, org, project)
	return st, err == nil, err
}

// ProgramProfileKey is the Profile row a question is answered through.
func ProgramProfileKey(q engine.Question) string {
	switch q.Kind {
	case engine.QuestionWorkType:
		return "hdr.work_type"
	case engine.QuestionCondition:
		return "hdr.cond." + q.Key
	case engine.QuestionPlanning:
		return knowledge.PlanningPrefix + q.Key
	}
	return "det." + q.Key
}

// questionKey is the bare key OneQuestion's dismissals use for a profile key.
func questionKey(profileKey string) string {
	if profileKey == "hdr.work_type" {
		return "work_type"
	}
	for _, p := range []string{"det.", "hdr.cond.", knowledge.PlanningPrefix} {
		if k, ok := strings.CutPrefix(profileKey, p); ok {
			return k
		}
	}
	return profileKey
}

// genInputFor completes the stored inputs with what the live program holds:
// its working week and the durations the user typed, so a regeneration
// keeps them (7.2 step 8: a user value wins).
func genInputFor(in engine.GenInput, p Program, acts []ProgramActivity) engine.GenInput {
	in.WorkingWeek = p.WorkingWeek
	in.UserDurations = map[string]int{}
	for _, a := range acts {
		if a.SourceKey != "" && a.DurationOrigin == "user" && a.DurationDays != nil {
			in.UserDurations[string(a.SourceKey)] = *a.DurationDays
		}
	}
	return in
}

// ProgramView adds the extras to a state (plan 8.2). Generation is pure and
// a 30-package plan regenerates in well under a millisecond, so the
// skeleton's reason, the checks and the question are recomputed on every
// read rather than stored: they then always follow the current Profile.
func (s *Store) ProgramView(ctx context.Context, org, project string, st ProgramState, cat *knowledge.Catalog) (ProgramView, error) {
	v := ProgramView{ProgramState: st}
	ex, err := s.programExtras(ctx, s.pool, org, project, st, cat)
	v.ProgramExtras = ex
	return v, err
}

// ResultView is ProgramView for a write's result, keeping its feedback.
func (s *Store) ResultView(ctx context.Context, org, project string, r ProgramResult, cat *knowledge.Catalog) (ProgramView, error) {
	v, err := s.ProgramView(ctx, org, project, r.ProgramState, cat)
	// Lists are [] and never null, so the client reads every field as sent.
	fb, empty := r.ProgramFeedback, emptyFeedback()
	if fb.TempIDs == nil {
		fb.TempIDs = empty.TempIDs
	}
	for _, l := range []*[]ProgramLink{&fb.HealedLinks, &fb.ChainAdded, &fb.ChainRemoved, &fb.BackwardLinks, &fb.Linked} {
		if *l == nil {
			*l = []ProgramLink{}
		}
	}
	if fb.Toasts == nil {
		fb.Toasts = []string{}
	}
	v.ProgramFeedback = &fb
	return v, err
}

func (s *Store) programExtras(ctx context.Context, q programQuerier, org, project string, st ProgramState, cat *knowledge.Catalog) (ProgramExtras, error) {
	ex := ProgramExtras{Checks: []engine.Check{}, Skeleton: ProgramSkeleton{ID: st.Program.SkeletonID}}
	var dismissedRaw string
	if err := q.QueryRow(ctx, `SELECT p.question_dismissed::text,
 (SELECT count(*) FROM program_readings r WHERE r.org_id=p.org_id AND r.program_id=p.id AND r.status='proposed'),
 (SELECT count(*) FROM program_changes c WHERE c.org_id=p.org_id AND c.program_id=p.id AND c.status='pending')
 FROM programs p WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, org, st.Program.ID).Scan(&dismissedRaw, &ex.ReadingsPending, &ex.ChangesPending); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ex, ErrNotFound
		}
		return ex, err
	}
	if cat == nil {
		return ex, nil
	}
	in, err := readProgramGenInput(ctx, q, org, project, cat)
	if err != nil {
		return ex, err
	}
	gi := genInputFor(in.gi, st.Program, st.Activities)
	plan, err := engine.Generate(cat, gi)
	if err != nil {
		return ex, err
	}
	for _, sk := range cat.Skeletons() {
		if sk.ID == string(st.Program.SkeletonID) {
			ex.Skeleton.Label = sk.Label
		}
	}
	same := plan.SkeletonID == string(st.Program.SkeletonID)
	if same {
		ex.Skeleton.Reason = plan.Reason
	}
	if st.Program.StartDate == "" && gi.Facts.ConsentDate != "" {
		ex.StartSuggestion = NullText(gi.Facts.ConsentDate)
	}

	byKey := map[string]string{}
	for _, a := range st.Activities {
		if a.SourceKey != "" {
			byKey[string(a.SourceKey)] = a.ID
		}
	}
	ci := engine.CheckInput{Input: st.Input(), Output: st.Computed, ConsentLapseDate: gi.Facts.ConsentLapseDate,
		TargetCompletion: isoDate(gi.Planning["target_completion"]), CalendarYears: in.years, SkeletonTie: same && plan.SkeletonTie}
	if same {
		ci.ConstructionStageID = byKey[plan.ConstructionStageKey]
	}
	// Edited rows of a package removed in Procurement stay until the user
	// decides in the review; the check names them meanwhile (plan 6.6).
	ci.RemovedPackageRows = engine.RemovedPackageRows(engine.Diff(liveRows(st.Activities), liveLinks(st.Links), plan))
	ex.Checks = append(ex.Checks, engine.Checks(ci)...)
	// Generation notes (an unknown lag, a library link that could not be
	// placed) still describe the stored rows they name; the skeleton tie is
	// already a check.
	for _, n := range plan.Notes {
		if n.Code == engine.CheckSkeletonTie || !same {
			continue
		}
		if n.ActivityID != "" {
			id, ok := byKey[n.ActivityID]
			if !ok {
				continue
			}
			n.ActivityID = id
		}
		ex.Checks = append(ex.Checks, n)
	}

	stored := map[string]string{}
	if err := json.Unmarshal([]byte(dismissedRaw), &stored); err != nil {
		return ex, err
	}
	dismissed := map[string]bool{}
	for key, hash := range stored {
		if in.hashes[key] == hash {
			dismissed[questionKey(key)] = true
		}
	}
	ex.Question, err = engine.OneQuestion(cat, gi, dismissed)
	return ex, err
}

// DismissProgramQuestion records "Not sure" on the question for profileKey:
// it is not asked again until that key's whole-project profile row changes.
// It bumps the version so every open view refetches its strip.
func (s *Store) DismissProgramQuestion(ctx context.Context, org, project string, version int64, profileKey string) (ProgramState, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE programs p SET question_dismissed=p.question_dismissed||jsonb_build_object($3::text,
 COALESCE((SELECT r.projection_hash FROM profile_rows r JOIN project_parts pp ON pp.org_id=r.org_id AND pp.site_id=r.site_id AND pp.id=r.part_id AND pp.kind='whole'
  WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.key=$3 LIMIT 1),'')),version=version+1,updated_at=now()
 WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.version=$4`, org, project, profileKey, version)
	if err != nil {
		return ProgramState{}, err
	}
	st, err := s.LoadProgram(ctx, org, project)
	if err != nil {
		return st, err
	}
	if tag.RowsAffected() == 0 {
		return st, ErrVersionConflict
	}
	return st, nil
}

// ApplyProgramAnswer applies, as one undoable batch, what a Profile answer
// given in the question strip changes (plan 7.3). The caller has already
// written the Profile value through the Profile edit path; this regenerates
// over the stored inputs and adopts every difference whose default is
// Adopt. Differences that default to Keep (rows the user created, edited or
// accepted) are left for the review, so a user value is never overwritten.
// When nothing changes it writes nothing and returns the current state with
// changed false.
func (s *Store) ApplyProgramAnswer(ctx context.Context, org, project, actor string, version int64, cat *knowledge.Catalog, q engine.Question, value string) (ProgramResult, int, error) {
	if cat == nil {
		return ProgramResult{}, 0, errors.New("program generation needs the knowledge catalogue")
	}
	var applied []engine.Op
	res, err := s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		in, err := readProgramGenInput(ctx, tx, org, project, cat)
		if err != nil {
			return err
		}
		live := w.state()
		plan, err := engine.Generate(cat, genInputFor(in.gi, w.prog, live.Activities))
		if err != nil {
			return err
		}
		for _, c := range engine.Diff(liveRows(live.Activities), liveLinks(live.Links), plan) {
			if c.DefaultChoice != engine.ChoiceAdopt {
				continue
			}
			for _, op := range c.Ops {
				ok, err := w.applyGenerated(len(applied), op)
				if err != nil {
					return err
				}
				if ok {
					applied = append(applied, op)
				}
			}
		}
		if len(applied) == 0 {
			return errNoProgramChange
		}
		w.fb.Toasts = append(w.fb.Toasts, answerToast(q, value, len(applied)))
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		ops := []any{map[string]any{"op": "profile_answer", "key": q.Key, "kind": q.Kind, "value": value, "changes": applied}}
		return logProgramBatch(ctx, tx, w, actor, ops, c)
	})
	if errors.Is(err, errNoProgramChange) {
		st, err := s.LoadProgram(ctx, org, project)
		fb := emptyFeedback()
		fb.Toasts = append(fb.Toasts, answerToast(q, value, 0))
		return ProgramResult{ProgramState: st, ProgramFeedback: fb}, 0, err
	}
	return res, len(applied), err
}

// answerToast says what the answer did, in the option's own words.
func answerToast(q engine.Question, value string, n int) string {
	shown := value
	for i, v := range q.Values {
		if v == value && i < len(q.Options) {
			shown = q.Options[i]
		}
	}
	switch n {
	case 0:
		return fmt.Sprintf("%s: %s saved to the Profile. The program did not change.", q.Label, shown)
	case 1:
		return fmt.Sprintf("%s: %s saved to the Profile. 1 program change applied; undo reverses it.", q.Label, shown)
	}
	return fmt.Sprintf("%s: %s saved to the Profile. %d program changes applied; undo reverses them.", q.Label, shown, n)
}

func liveRows(acts []ProgramActivity) []engine.LiveRow {
	out := make([]engine.LiveRow, len(acts))
	for i, a := range acts {
		out[i] = engine.LiveRow{ID: a.ID, SourceKey: string(a.SourceKey), ParentID: string(a.ParentID), Kind: a.Kind, Label: a.Label,
			CatalogueStageID: string(a.CatalogueStageID), PackageID: string(a.PackageID), Duration: a.DurationDays, Low: a.DurationLow,
			High: a.DurationHigh, DurationOrigin: string(a.DurationOrigin), Origin: a.Origin, ReviewStatus: a.ReviewStatus}
	}
	return out
}

func liveLinks(links []ProgramLink) []engine.LiveLink {
	out := make([]engine.LiveLink, len(links))
	for i, l := range links {
		out[i] = engine.LiveLink{ID: l.ID, PredID: l.PredID, SuccID: l.SuccID, Type: l.Type, Lag: l.LagDays, LagUnit: l.LagUnit,
			Origin: l.Origin, SourceKey: string(l.SourceKey)}
	}
	return out
}

// generatedOp is a review op (program.Op) in typed form. A JSON round trip
// turns the op's Go values (pointers, maps) into one shape.
type generatedOp struct {
	Op               string          `json:"op"`
	TempID           string          `json:"temp_id"`
	ID               string          `json:"id"`
	IDs              []string        `json:"ids"`
	ParentID         *string         `json:"parent_id"`
	AfterID          string          `json:"after_id"`
	Kind             string          `json:"kind"`
	Label            *string         `json:"label"`
	DurationDays     *int            `json:"duration_days"`
	DurationLow      *int            `json:"duration_low"`
	DurationHigh     *int            `json:"duration_high"`
	DurationOrigin   string          `json:"duration_origin"`
	DurationSource   json.RawMessage `json:"duration_source"`
	SourceKey        string          `json:"source_key"`
	Systems          []string        `json:"systems"`
	CatalogueStageID string          `json:"catalogue_stage_id"`
	PackageID        string          `json:"package_id"`
	PackageStageID   string          `json:"package_stage_id"`
	WorkItemID       string          `json:"work_item_id"`
	FragmentID       string          `json:"fragment_id"`
	InterfaceID      string          `json:"interface_id"`
	PredID           string          `json:"pred_id"`
	SuccID           string          `json:"succ_id"`
	Type             string          `json:"type"`
	LagDays          int             `json:"lag_days"`
	LagUnit          string          `json:"lag_unit"`
	Origin           string          `json:"origin"`
}

// applyGenerated applies one op of a generated difference. Unlike a user
// op it keeps the row the library's: an added row carries its source key,
// library origin and proposed status; a rename or duration change leaves
// origin and review status alone. An op whose rows are gone or not added
// (a link to a row kept out) is skipped and reports false; a link the
// engine refuses (a loop) is skipped too, never forced.
func (w *programWork) applyGenerated(i int, raw engine.Op) (bool, error) {
	body, err := json.Marshal(raw)
	if err != nil {
		return false, err
	}
	var o generatedOp
	if err := json.Unmarshal(body, &o); err != nil {
		return false, err
	}
	durationSent := func() bool { _, ok := raw["duration_days"]; return ok }
	switch o.Op {
	case "add":
		return w.addGenerated(i, o)
	case "update":
		a := w.live(w.resolve(o.ID))
		if a == nil {
			return false, nil
		}
		a = w.touchAct(a.ID)
		if o.Label != nil {
			label := strings.TrimSpace(*o.Label)
			if !validLabel(label) {
				return false, opErr(i, "label", MsgLabel)
			}
			if label != a.Label && a.Kind == engine.KindStage {
				w.stageChange = true
			}
			a.Label = label
		}
		if durationSent() && !(a.Kind == engine.KindStage && len(w.kids[a.ID]) > 0) {
			setGeneratedDuration(a, o)
		}
		w.dirty = true
		return true, nil
	case "delete":
		var ids []string
		for _, id := range o.IDs {
			if a := w.live(w.resolve(id)); a != nil {
				ids = append(ids, a.ID)
			}
		}
		if len(ids) == 0 {
			return false, nil
		}
		reason := w.genReason
		if reason == "" {
			reason = "No longer applies after the Profile answer."
		}
		return true, w.opDelete(i, ProgramOp{Op: "delete", IDs: ids, Reason: &reason})
	case "link_add":
		pred, succ := w.live(w.resolve(o.PredID)), w.live(w.resolve(o.SuccID))
		if pred == nil || succ == nil {
			return false, nil
		}
		if id, ok := w.pairs[[2]string{pred.ID, succ.ID}]; ok {
			// A chain link a placement just made is this library link: give
			// it the source key so the review matches it next time.
			l := w.links[id]
			if l.SourceKey == "" && l.Type == o.Type && l.LagDays == o.LagDays && l.LagUnit == o.LagUnit {
				l = w.touchLink(id)
				l.SourceKey, l.Origin, l.InterfaceID = NullText(o.SourceKey), o.Origin, NullText(o.InterfaceID)
				return true, nil
			}
			return false, nil
		}
		link := engine.Link{Pred: pred.ID, Succ: succ.ID, Type: o.Type, Lag: o.LagDays, LagUnit: o.LagUnit}
		if engine.ValidateLink(w.input(), link) != nil {
			return false, nil
		}
		l := w.addLink(pred.ID, succ.ID, o.Type, o.LagDays, o.LagUnit, o.Origin)
		l.SourceKey, l.InterfaceID = NullText(o.SourceKey), NullText(o.InterfaceID)
		return true, nil
	case "link_update":
		cur := w.links[w.resolve(o.ID)]
		if cur == nil || cur.Retired {
			return false, nil
		}
		link := engine.Link{ID: cur.ID, Pred: cur.PredID, Succ: cur.SuccID, Type: o.Type, Lag: o.LagDays, LagUnit: o.LagUnit}
		if engine.ValidateLink(w.input(), link) != nil {
			return false, nil
		}
		l := w.touchLink(cur.ID)
		l.Type, l.LagDays, l.LagUnit = o.Type, o.LagDays, o.LagUnit
		return true, nil
	case "link_delete":
		done := false
		for _, id := range o.IDs {
			if l := w.links[w.resolve(id)]; l != nil && !l.Retired {
				w.retireLink(l.ID)
				done = true
			}
		}
		return done, nil
	}
	return false, fmt.Errorf("generated op %q", o.Op)
}

// addGenerated places a generated row with the user add (so chains and a
// leaf stage's duration behave as they do for any add), then makes it the
// library's row.
func (w *programWork) addGenerated(i int, o generatedOp) (bool, error) {
	parent := ""
	if o.ParentID != nil {
		parent = w.resolve(*o.ParentID)
		if w.live(parent) == nil {
			return false, nil // its stage was kept out
		}
	}
	siblings := w.stages
	if parent != "" {
		siblings = w.kids[parent]
	}
	after := ""
	if o.AfterID != "" && indexOf(siblings, w.resolve(o.AfterID)) >= 0 {
		after = o.AfterID
	}
	kind, label := o.Kind, ""
	if o.Label != nil {
		label = *o.Label
	}
	op := ProgramOp{Op: "add", TempID: o.TempID, ParentID: parent, AfterID: after, Kind: &kind, Label: &label}
	if err := w.opAdd(i, op); err != nil {
		return false, err
	}
	a := w.touchAct(w.temp[o.TempID])
	a.SourceKey, a.Origin, a.ReviewStatus = NullText(o.SourceKey), engine.OriginLibrary, engine.ReviewProposed
	if a.Kind == engine.KindStage && o.CatalogueStageID != "" {
		a.CatalogueStageID = NullText(o.CatalogueStageID)
	}
	a.PackageID, a.PackageStageID, a.WorkItemID = NullText(o.PackageID), NullText(o.PackageStageID), NullText(o.WorkItemID)
	a.FragmentID, a.InterfaceID = NullText(o.FragmentID), NullText(o.InterfaceID)
	if o.Systems != nil {
		a.Systems = o.Systems
	}
	if a.Kind != engine.KindMilestone && a.Kind != engine.KindHoldPoint {
		setGeneratedDuration(a, o)
	}
	return true, nil
}

func setGeneratedDuration(a *ProgramActivity, o generatedOp) {
	a.DurationDays, a.DurationLow, a.DurationHigh, a.DurationOrigin = o.DurationDays, o.DurationLow, o.DurationHigh, NullText(o.DurationOrigin)
	if a.DurationDays == nil {
		a.DurationOrigin = ""
	}
	a.DurationSource = json.RawMessage(`{}`)
	if len(o.DurationSource) > 0 && string(o.DurationSource) != "null" {
		a.DurationSource = o.DurationSource
	}
}
