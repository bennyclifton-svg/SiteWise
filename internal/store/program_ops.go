package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// Plain messages returned with a 422 (plan 8.1 and 13.1).
const (
	MsgTwoTiers        = "Programs have two levels: stages and activities."
	MsgNotInProgram    = "That row is not in this program."
	MsgLinkNotFound    = "That link is not in this program."
	MsgLabel           = "Names are 1 to 200 characters."
	MsgDuration        = "Durations are 0 to 3,650 working days."
	MsgDate            = "Dates are written YYYY-MM-DD."
	MsgStageSpan       = "A stage with activities takes its length from them."
	MsgTooManyOps      = "A change holds at most 500 edits."
	MsgTooManyRows     = "This program holds at most 5,000 rows. Collapse stages to work faster."
	MsgTooManyLinks    = "This program holds at most 20,000 links."
	MsgNoStart         = "Set a program start date before moving bars to dates."
	MsgUndoChanged     = "Can't undo: changed since"
	MsgRedoChanged     = "Can't redo: changed since"
	MsgNothingToUndo   = "Nothing to undo."
	MsgNothingToRedo   = "Nothing to redo."
	MsgHoldPointReason = "Say why this hold point is not needed."
)

// ProgramOpError is a refused batch: a 422 naming the op (by index, −1 for
// the batch) and the field. Cycle holds a loop's ids when a link would close
// one; Held names rows a move could not shift.
type ProgramOpError struct {
	Index   int      `json:"op_index"`
	Field   string   `json:"field,omitempty"`
	Message string   `json:"message"`
	Cycle   []string `json:"cycle,omitempty"`
	Held    []string `json:"held,omitempty"`
}

func (e *ProgramOpError) Error() string { return e.Message }

func opErr(i int, field, msg string) *ProgramOpError {
	return &ProgramOpError{Index: i, Field: field, Message: msg}
}

// OptInt is a JSON number that may be absent, null or set: an update with
// "duration_days": null makes a duration unknown; leaving it out keeps it.
type OptInt struct {
	Set   bool
	Value *int
}

func (o *OptInt) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

func (o OptInt) MarshalJSON() ([]byte, error) {
	if o.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.Value)
}

// ProgramOp is one edit of a batch (plan 8.1). Any id field may name the
// temp_id of a row added earlier in the same batch.
type ProgramOp struct {
	Op               string   `json:"op"`
	TempID           string   `json:"temp_id,omitempty"`
	ID               string   `json:"id,omitempty"`
	IDs              []string `json:"ids,omitempty"`
	ParentID         string   `json:"parent_id,omitempty"`
	AfterID          string   `json:"after_id,omitempty"`
	Kind             *string  `json:"kind,omitempty"`
	Label            *string  `json:"label,omitempty"`
	DurationDays     OptInt   `json:"duration_days"`
	ConstraintKind   *string  `json:"constraint_kind,omitempty"`
	ConstraintDate   *string  `json:"constraint_date,omitempty"`
	CatalogueStageID *string  `json:"catalogue_stage_id,omitempty"`
	Days             *int     `json:"days,omitempty"`
	StartDate        *string  `json:"start_date,omitempty"`
	Reason           *string  `json:"reason,omitempty"`
	PredID           string   `json:"pred_id,omitempty"`
	SuccID           string   `json:"succ_id,omitempty"`
	Type             *string  `json:"type,omitempty"`
	LagDays          *int     `json:"lag_days,omitempty"`
	LagUnit          *string  `json:"lag_unit,omitempty"`
	Day              string   `json:"day,omitempty"`
}

// MarshalJSON leaves duration_days out when it was not sent, so the stored
// batch reads as it arrived.
func (o ProgramOp) MarshalJSON() ([]byte, error) {
	type plain ProgramOp
	var d json.RawMessage
	if o.DurationDays.Set {
		d, _ = o.DurationDays.MarshalJSON()
	}
	return json.Marshal(struct {
		plain
		DurationDays json.RawMessage `json:"duration_days,omitempty"`
	}{plain(o), d})
}

// ProgramResult is the authoritative state after a write plus what the
// batch did, serialised as plan 8.2 with the op feedback beside it.
type ProgramResult struct {
	ProgramState
	ProgramFeedback
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// checkBatch is validation at the boundary: shapes and ranges, before any
// row is read. Whether ids belong to the program is checked against the
// loaded state.
func checkBatch(ops []ProgramOp) error {
	if len(ops) == 0 {
		return opErr(-1, "ops", "No changes sent.")
	}
	if len(ops) > MaxProgramOps {
		return opErr(-1, "ops", MsgTooManyOps)
	}
	temps := map[string]bool{}
	for i, o := range ops {
		if err := o.shape(i, temps); err != nil {
			return err
		}
	}
	return nil
}

func (o ProgramOp) shape(i int, temps map[string]bool) *ProgramOpError {
	ref := func(field, v string, required bool) *ProgramOpError {
		if v == "" {
			if required {
				return opErr(i, field, MsgNotInProgram)
			}
			return nil
		}
		if !uuidPattern.MatchString(v) && !temps[v] {
			return opErr(i, field, MsgNotInProgram)
		}
		return nil
	}
	refs := func(field string, vs []string, min int) *ProgramOpError {
		if len(vs) < min {
			return opErr(i, field, "Choose at least one row.")
		}
		if len(vs) > MaxProgramRows {
			return opErr(i, field, MsgTooManyRows)
		}
		for _, v := range vs {
			if e := ref(field, v, true); e != nil {
				return e
			}
		}
		return nil
	}
	label := func(required bool) *ProgramOpError {
		if o.Label == nil {
			if required {
				return opErr(i, "label", MsgLabel)
			}
			return nil
		}
		if !validLabel(*o.Label) {
			return opErr(i, "label", MsgLabel)
		}
		return nil
	}
	duration := func(required bool) *ProgramOpError {
		if required && (!o.DurationDays.Set || o.DurationDays.Value == nil) {
			return opErr(i, "duration_days", MsgDuration)
		}
		if v := o.DurationDays.Value; v != nil && (*v < 0 || *v > engine.MaxDuration) {
			return opErr(i, "duration_days", MsgDuration)
		}
		return nil
	}
	date := func(field string, v *string) *ProgramOpError {
		if v != nil && *v != "" && !validDate(*v) {
			return opErr(i, field, MsgDate)
		}
		return nil
	}
	first := func(errs ...*ProgramOpError) *ProgramOpError {
		for _, e := range errs {
			if e != nil {
				return e
			}
		}
		return nil
	}
	switch o.Op {
	case "add":
		if o.TempID == "" || len(o.TempID) > 100 || temps[o.TempID] || uuidPattern.MatchString(o.TempID) {
			return opErr(i, "temp_id", "Each new row needs its own temp_id.")
		}
		e := first(ref("parent_id", o.ParentID, false), ref("after_id", o.AfterID, false), label(true), duration(false))
		temps[o.TempID] = true
		if e != nil {
			return e
		}
		if o.Kind != nil {
			switch *o.Kind {
			case engine.KindStage, engine.KindActivity, engine.KindMilestone, engine.KindHoldPoint:
			default:
				return opErr(i, "kind", "Kind is stage, activity, milestone or hold_point.")
			}
		}
		return nil
	case "update":
		e := first(ref("id", o.ID, true), label(false), duration(false), date("constraint_date", o.ConstraintDate))
		if e != nil {
			return e
		}
		if k := o.ConstraintKind; k != nil && *k != engine.ConstraintASAP && *k != engine.ConstraintSNET && *k != engine.ConstraintMSO {
			return opErr(i, "constraint_kind", "Constraint is asap, snet or mso.")
		}
		if c := o.CatalogueStageID; c != nil && (*c == "" || len(*c) > 200) {
			return opErr(i, "catalogue_stage_id", "Choose a catalogue stage or other.")
		}
		return nil
	case "move":
		if o.Days != nil {
			if o.StartDate != nil {
				return opErr(i, "start_date", "Move by days or to a date, not both.")
			}
			if *o.Days < -engine.MaxDuration || *o.Days > engine.MaxDuration {
				return opErr(i, "days", "A move is at most 3,650 working days.")
			}
			return refs("ids", o.IDs, 1)
		}
		if o.StartDate == nil || *o.StartDate == "" {
			return opErr(i, "start_date", MsgDate)
		}
		return first(ref("id", o.ID, true), date("start_date", o.StartDate))
	case "resize":
		return first(ref("id", o.ID, true), duration(true))
	case "delete":
		if o.Reason != nil && strings.TrimSpace(*o.Reason) != "" && !validLabel(*o.Reason) {
			return opErr(i, "reason", "A reason is at most 200 characters.")
		}
		return refs("ids", o.IDs, 1)
	case "indent", "outdent":
		return refs("ids", o.IDs, 1)
	case "reorder":
		return first(refs("ids", o.IDs, 1), ref("parent_id", o.ParentID, false), ref("after_id", o.AfterID, false))
	case "link_add":
		return first(ref("pred_id", o.PredID, true), ref("succ_id", o.SuccID, true), linkFields(i, o))
	case "link_chain":
		return refs("ids", o.IDs, 2)
	case "link_update":
		return first(ref("id", o.ID, true), linkFields(i, o))
	case "link_delete":
		return refs("ids", o.IDs, 1)
	case "set_start":
		return date("start_date", o.StartDate)
	case "calendar_day":
		if !validDate(o.Day) {
			return opErr(i, "day", MsgDate)
		}
		if o.Kind != nil {
			if !validDayKind(*o.Kind) {
				return opErr(i, "kind", "Day kind is public_holiday, rdo, shutdown or working_override.")
			}
			return label(true)
		}
		return nil
	}
	return opErr(i, "op", fmt.Sprintf("Unknown edit %q.", o.Op))
}

func linkFields(i int, o ProgramOp) *ProgramOpError {
	if t := o.Type; t != nil && *t != engine.LinkFS && *t != engine.LinkSS && *t != engine.LinkFF && *t != engine.LinkSF {
		return opErr(i, "type", engine.MsgLinkType)
	}
	if l := o.LagDays; l != nil && (*l < -365 || *l > 365) {
		return opErr(i, "lag_days", engine.MsgLinkLag)
	}
	if u := o.LagUnit; u != nil && *u != engine.LagWorking && *u != engine.LagCalendar {
		return opErr(i, "lag_unit", engine.MsgLinkLagUnit)
	}
	return nil
}

// ApplyProgramOps applies one batch atomically: one transaction, one
// version bump, one undo step. A stale version returns ErrVersionConflict
// with the current state in the result; a refused op returns a
// *ProgramOpError and writes nothing.
func (s *Store) ApplyProgramOps(ctx context.Context, org, project, actor, programID string, version int64, ops []ProgramOp) (ProgramResult, error) {
	if err := checkBatch(ops); err != nil {
		return ProgramResult{}, err
	}
	return s.programWrite(ctx, org, project, programID, version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		for i, o := range ops {
			if err := w.apply(i, o); err != nil {
				return err
			}
		}
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		return logProgramBatch(ctx, tx, w, actor, ops, c)
	})
}

type programEdit func(ctx context.Context, tx pgx.Tx, w *programWork) error
type programLog func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error

// programWrite is the shared write path: lock and load, pin the version,
// edit in memory, check, write the diff, log it, bump the program.
func (s *Store) programWrite(ctx context.Context, org, project, programID string, version int64, edit programEdit, log programLog) (ProgramResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramResult{}, err
	}
	defer tx.Rollback(ctx)
	p, acts, links, days, err := readProgramInputs(ctx, tx, org, project, true)
	if err != nil {
		return ProgramResult{}, err
	}
	if programID != "" && p.ID != programID {
		return ProgramResult{}, ErrNotFound
	}
	if p.Version != version {
		st := computeState(p, acts, links, days)
		return ProgramResult{ProgramState: st, ProgramFeedback: emptyFeedback()}, ErrVersionConflict
	}
	w := newProgramWork(org, project, p, acts, links, days)
	if cat := s.workCatalog(); cat != nil {
		w.catalogue = func(id string) bool { _, ok := cat.DeliveryStage(id); return ok }
	}
	if err := edit(ctx, tx, w); err != nil {
		return ProgramResult{}, err
	}
	if err := w.finish(); err != nil {
		return ProgramResult{}, err
	}
	c := w.diff()
	if err := c.write(ctx, tx, org, p.ID); err != nil {
		return ProgramResult{}, err
	}
	if err := log(ctx, tx, w, c); err != nil {
		return ProgramResult{}, err
	}
	bump := 0
	if w.stageChange {
		bump = 1
	}
	if err := tx.QueryRow(ctx, `UPDATE programs SET version=version+1,stage_revision=stage_revision+$3,start_date=NULLIF($4,'')::date,start_origin=NULLIF($5,''),updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid RETURNING version,stage_revision`,
		org, p.ID, bump, string(w.prog.StartDate), string(w.prog.StartOrigin)).Scan(&w.prog.Version, &w.prog.StageRevision); err != nil {
		return ProgramResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramResult{}, err
	}
	return ProgramResult{ProgramState: w.state(), ProgramFeedback: w.fb}, nil
}

func emptyFeedback() ProgramFeedback {
	return ProgramFeedback{TempIDs: map[string]string{}, HealedLinks: []ProgramLink{}, ChainAdded: []ProgramLink{}, ChainRemoved: []ProgramLink{},
		BackwardLinks: []ProgramLink{}, Linked: []ProgramLink{}, Toasts: []string{}}
}

func logProgramBatch(ctx context.Context, tx pgx.Tx, w *programWork, actor string, ops any, c programChanges) error {
	opsRaw, err := json.Marshal(ops)
	if err != nil {
		return err
	}
	inverse := c.inverse
	if inverse == nil {
		inverse = []programInverse{}
	}
	invRaw, err := json.Marshal(inverse)
	if err != nil {
		return err
	}
	// clock_timestamp, not now(): writers are serialised by the program row
	// lock, so times taken after it order batches and undos the way they
	// happened (redo compares them).
	_, err = tx.Exec(ctx, `INSERT INTO program_ops(org_id,program_id,seq,actor,ops,inverse,at)
SELECT $1::uuid,$2::uuid,COALESCE(max(seq),0)+1,$3::uuid,$4::jsonb,$5::jsonb,clock_timestamp() FROM program_ops WHERE org_id=$1::uuid AND program_id=$2::uuid`, w.org, w.prog.ID, actor, opsRaw, invRaw)
	return err
}

// finish normalises ordinals and checks the limits and the whole program
// before anything is written.
func (w *programWork) finish() error {
	w.renumber()
	rows, links := w.countLive()
	if rows > MaxProgramRows && rows > w.loadedRows() {
		return opErr(-1, "ops", MsgTooManyRows)
	}
	if links > MaxProgramLinks && links > w.loadedLinks() {
		return opErr(-1, "ops", MsgTooManyLinks)
	}
	if err := w.checkStructure(); err != nil {
		return opErr(-1, "ops", MsgTwoTiers)
	}
	in := w.input()
	if err := engine.Validate(in); err != nil {
		return opErr(-1, "ops", err.Error())
	}
	if c := w.output().Cycle; c != nil {
		return &ProgramOpError{Index: -1, Field: "ops", Message: "These changes would make a loop: " + engine.CycleWords(in, c), Cycle: c}
	}
	return nil
}

func (w *programWork) loadedRows() int {
	n := 0
	for id, a := range w.acts {
		before := a
		if b, ok := w.actBefore[id]; ok {
			before = b
		}
		if !w.actNew[id] && !before.Retired {
			n++
		}
	}
	return n
}

func (w *programWork) loadedLinks() int {
	n := 0
	for id, l := range w.links {
		before := l
		if b, ok := w.linkBefore[id]; ok {
			before = b
		}
		if !w.linkNew[id] && !before.Retired {
			n++
		}
	}
	return n
}

func (w *programWork) apply(i int, o ProgramOp) error {
	switch o.Op {
	case "add":
		return w.opAdd(i, o)
	case "update":
		return w.opUpdate(i, o)
	case "move":
		return w.opMove(i, o)
	case "resize":
		return w.opUpdate(i, ProgramOp{Op: "resize", ID: o.ID, DurationDays: o.DurationDays})
	case "delete":
		return w.opDelete(i, o)
	case "indent":
		return w.opIndent(i, o)
	case "outdent":
		return w.opOutdent(i, o)
	case "reorder":
		return w.opReorder(i, o)
	case "link_add":
		return w.opLinkAdd(i, o)
	case "link_chain":
		return w.opLinkChain(i, o)
	case "link_update":
		return w.opLinkUpdate(i, o)
	case "link_delete":
		return w.opLinkDelete(i, o)
	case "set_start":
		return w.opSetStart(o)
	case "calendar_day":
		return w.opCalendarDay(o)
	}
	return opErr(i, "op", fmt.Sprintf("Unknown edit %q.", o.Op))
}

// row resolves a live row named by an op field.
func (w *programWork) row(i int, field, ref string) (*ProgramActivity, error) {
	if a := w.live(w.resolve(ref)); a != nil {
		return a, nil
	}
	return nil, opErr(i, field, MsgNotInProgram)
}

// rowsOf resolves ids to live rows in outline order, once each.
func (w *programWork) rowsOf(i int, field string, refs []string) ([]*ProgramActivity, error) {
	seen := map[string]bool{}
	var out []*ProgramActivity
	for _, ref := range refs {
		a, err := w.row(i, field, ref)
		if err != nil {
			return nil, err
		}
		if !seen[a.ID] {
			seen[a.ID] = true
			out = append(out, a)
		}
	}
	pos := w.positions()
	sort.SliceStable(out, func(x, y int) bool { return pos[out[x].ID] < pos[out[y].ID] })
	return out, nil
}

// edited marks a row the user changed: a library row becomes the user's,
// and an edited row is accepted.
func edited(a *ProgramActivity) {
	if a.Origin == "library" {
		a.Origin = "user"
	}
	a.ReviewStatus = "accepted"
}

func intp(v int) *int { return &v }

func insertAfter(seq []string, id, after string) []string {
	out := make([]string, 0, len(seq)+1)
	if after == "" {
		return append(append(out, id), seq...)
	}
	for _, x := range seq {
		out = append(out, x)
		if x == after {
			out = append(out, id)
		}
	}
	return out
}

func without(seq []string, drop map[string]bool) []string {
	out := make([]string, 0, len(seq))
	for _, x := range seq {
		if !drop[x] {
			out = append(out, x)
		}
	}
	return out
}

func indexOf(seq []string, id string) int {
	for i, x := range seq {
		if x == id {
			return i
		}
	}
	return -1
}

// setKids replaces a stage's children, dropping the entry when empty.
func (w *programWork) setKids(stage string, seq []string) {
	if len(seq) == 0 {
		delete(w.kids, stage)
	} else {
		w.kids[stage] = seq
	}
	w.dirty = true
}

// rechain rewrites chain links where a sequence changed (plan 5 "Chain
// links", 9.4): a chain between two rows that are no longer neighbours is
// retired; two rows that became neighbours get a chain unless a link
// already joins them or it would close a loop. Inside a stage an activity
// joins the chain only when nothing else drives it. Other links never change.
func (w *programWork) rechain(old, new []string, stages bool) {
	pairs := func(seq []string) map[[2]string]bool {
		m := map[[2]string]bool{}
		for k := 1; k < len(seq); k++ {
			m[[2]string{seq[k-1], seq[k]}] = true
		}
		return m
	}
	before, after := pairs(old), pairs(new)
	for k := 1; k < len(old); k++ {
		p := [2]string{old[k-1], old[k]}
		if after[p] {
			continue
		}
		if id, ok := w.pairs[p]; ok && w.links[id].Origin == "chain" {
			w.retireLink(id)
			w.fb.ChainRemoved = append(w.fb.ChainRemoved, *w.links[id])
		}
	}
	for k := 1; k < len(new); k++ {
		a, b := new[k-1], new[k]
		if before[[2]string{a, b}] || w.linked(a, b) {
			continue
		}
		if !stages && w.hasPredecessor(b) {
			continue
		}
		if l, ok := w.tryLink(a, b, "chain"); ok {
			w.fb.ChainAdded = append(w.fb.ChainAdded, *l)
		}
	}
}

func (w *programWork) hasPredecessor(id string) bool {
	for p := range w.pairs {
		if p[1] == id {
			return true
		}
	}
	return false
}

// tryLink adds an FS, no-lag link when the engine accepts it.
func (w *programWork) tryLink(pred, succ, origin string) (*ProgramLink, bool) {
	if engine.ValidateLink(w.input(), engine.Link{Pred: pred, Succ: succ, Type: engine.LinkFS, LagUnit: engine.LagWorking}) != nil {
		return nil, false
	}
	return w.addLink(pred, succ, engine.LinkFS, 0, engine.LagWorking, origin), true
}

// firstChild gives a leaf stage's duration to the activity that becomes its
// first child (plan 6.5), when that activity has none of its own. Either
// way the stage no longer has a duration: its span comes from its children.
func (w *programWork) firstChild(stage, child *ProgramActivity) {
	if stage.DurationDays == nil {
		return
	}
	if child.Kind != engine.KindActivity || child.DurationDays != nil {
		stage = w.touchAct(stage.ID)
		stage.DurationDays, stage.DurationLow, stage.DurationHigh = nil, nil, nil
		stage.DurationOrigin, stage.DurationSource = "", json.RawMessage(`{}`)
		return
	}
	s, c, moved := engine.FirstChildMove(engineActivity(stage), engineActivity(child))
	if !moved {
		return
	}
	stage, child = w.touchAct(stage.ID), w.touchAct(child.ID)
	child.DurationDays, child.DurationLow, child.DurationHigh = c.Duration, c.Low, c.High
	child.DurationOrigin, child.DurationSource = stage.DurationOrigin, stage.DurationSource
	stage.DurationDays, stage.DurationLow, stage.DurationHigh = s.Duration, s.Low, s.High
	stage.DurationOrigin, stage.DurationSource = "", json.RawMessage(`{}`)
	w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("%s's %d days moved to %s.", stage.Label, *child.DurationDays, child.Label))
}

// lastChild makes a stage that lost its last activity a leaf again with
// its last span in working days (plan 6.5). out is the output from before
// the activity left.
func (w *programWork) lastChild(stageID string, out engine.Output) {
	stage := w.live(stageID)
	if stage == nil || len(w.kids[stageID]) > 0 {
		return
	}
	span, ok := engine.StageSpan(out, stageID)
	stage = w.touchAct(stageID)
	stage.DurationLow, stage.DurationHigh, stage.DurationSource = nil, nil, json.RawMessage(`{}`)
	if !ok {
		stage.DurationDays, stage.DurationOrigin = nil, ""
		return
	}
	stage.DurationDays, stage.DurationOrigin = intp(span), "calculation"
	w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("%s has no activities now; it keeps its span of %d days.", stage.Label, span))
}

func (w *programWork) opAdd(i int, o ProgramOp) error {
	kind := engine.KindActivity
	if o.ParentID == "" {
		kind = engine.KindStage
	}
	if o.Kind != nil {
		kind = *o.Kind
	}
	a := ProgramActivity{Kind: kind, Label: strings.TrimSpace(*o.Label), ConstraintKind: engine.ConstraintASAP, ReviewStatus: "accepted", Origin: "user"}
	var parent *ProgramActivity
	if o.ParentID != "" {
		p, err := w.row(i, "parent_id", o.ParentID)
		if err != nil {
			return err
		}
		if p.Kind != engine.KindStage || kind == engine.KindStage {
			return opErr(i, "parent_id", MsgTwoTiers)
		}
		parent = p
		a.ParentID = NullText(p.ID)
	} else if kind != engine.KindStage {
		return opErr(i, "parent_id", "Every activity sits under a stage.")
	}
	if kind == engine.KindStage {
		a.CatalogueStageID = knowledge.OtherStageID
	}
	if o.DurationDays.Value != nil {
		a.DurationDays, a.DurationOrigin = intp(*o.DurationDays.Value), "user"
	}
	if kind == engine.KindMilestone || kind == engine.KindHoldPoint {
		if a.DurationDays != nil && *a.DurationDays != 0 {
			return opErr(i, "duration_days", "A milestone takes 0 days.")
		}
		a.DurationDays, a.DurationOrigin = intp(0), "user"
	}
	siblings := w.stages
	if parent != nil {
		siblings = w.kids[parent.ID]
	}
	// No after_id adds at the end.
	after := ""
	if o.AfterID != "" {
		after = w.resolve(o.AfterID)
		if indexOf(siblings, after) < 0 {
			return opErr(i, "after_id", MsgNotInProgram)
		}
	} else if len(siblings) > 0 {
		after = siblings[len(siblings)-1]
	}
	row := w.newAct(a)
	w.temp[o.TempID] = row.ID
	next := insertAfter(siblings, row.ID, after)
	if parent == nil {
		old := w.stages
		w.stages = next
		w.dirty = true
		w.stageChange = true
		w.rechain(old, next, true)
		return nil
	}
	old := w.kids[parent.ID]
	w.setKids(parent.ID, next)
	if len(old) == 0 {
		w.firstChild(parent, row)
	}
	w.rechain(old, next, false)
	return nil
}

func (w *programWork) opUpdate(i int, o ProgramOp) error {
	a, err := w.row(i, "id", o.ID)
	if err != nil {
		return err
	}
	a = w.touchAct(a.ID)
	if o.Label != nil {
		label := strings.TrimSpace(*o.Label)
		if label != a.Label && a.Kind == engine.KindStage {
			w.stageChange = true
		}
		a.Label = label
	}
	if o.DurationDays.Set {
		if a.Kind == engine.KindStage && len(w.kids[a.ID]) > 0 {
			return opErr(i, "duration_days", MsgStageSpan)
		}
		v := o.DurationDays.Value
		switch a.Kind {
		case engine.KindMilestone:
			if v == nil {
				return opErr(i, "duration_days", "A milestone takes 0 days.")
			}
			if *v > 0 { // plan 13.1: a milestone resized above 0 becomes an activity
				a.Kind = engine.KindActivity
			}
		case engine.KindHoldPoint:
			if v == nil || *v != 0 {
				return opErr(i, "duration_days", "A hold point takes 0 days.")
			}
		}
		a.DurationLow, a.DurationHigh, a.DurationSource = nil, nil, json.RawMessage(`{}`)
		if v == nil {
			a.DurationDays, a.DurationOrigin = nil, ""
		} else {
			a.DurationDays, a.DurationOrigin = intp(*v), "user"
		}
	}
	if o.ConstraintKind != nil || o.ConstraintDate != nil {
		kind := a.ConstraintKind
		if o.ConstraintKind != nil {
			kind = *o.ConstraintKind
		}
		date := string(a.ConstraintDate)
		if o.ConstraintDate != nil {
			date = *o.ConstraintDate
		}
		if kind == engine.ConstraintASAP {
			if o.ConstraintDate != nil && *o.ConstraintDate != "" {
				return opErr(i, "constraint_date", "As soon as possible takes no date.")
			}
			date = ""
		} else if date == "" {
			return opErr(i, "constraint_date", MsgDate)
		}
		a.ConstraintKind, a.ConstraintDate = kind, NullText(date)
	}
	if o.CatalogueStageID != nil {
		if a.Kind != engine.KindStage {
			return opErr(i, "catalogue_stage_id", "Only a stage maps to a catalogue stage.")
		}
		if !w.knownStage(*o.CatalogueStageID) {
			return opErr(i, "catalogue_stage_id", "That is not a catalogue stage.")
		}
		if NullText(*o.CatalogueStageID) != a.CatalogueStageID {
			a.CatalogueStageID = NullText(*o.CatalogueStageID)
			w.stageChange = true // plan 13.3 Q5: a remap is a stage change
		}
	}
	edited(a)
	return nil
}

// knownStage accepts `other` or a stage of the delivery stage catalogue.
func (w *programWork) knownStage(id string) bool {
	if id == knowledge.OtherStageID {
		return true
	}
	return w.catalogue != nil && w.catalogue(id)
}

// opMove shifts rows (plan 8.1, owner answer Q2): later sets a start no
// earlier than the new date; earlier lowers or removes that constraint and
// never breaks a link. A stage with activities moves its activities. All or
// none: when links hold any row, nothing moves and the held rows are named.
func (w *programWork) opMove(i int, o ProgramOp) error {
	if w.prog.StartDate == "" {
		return opErr(i, "ids", MsgNoStart)
	}
	out0 := w.output()
	times := out0.Activities.Map()
	cal := engine.NewCalendar(w.input().Calendar)
	desired := map[string]int{}
	var targets []*ProgramActivity
	add := func(a *ProgramActivity, shift int, field string) error {
		t := times[a.ID]
		if t.Unknown {
			return opErr(i, field, a.Label+" has no dates yet.")
		}
		desired[a.ID] = t.ES + shift
		targets = append(targets, a)
		return nil
	}
	if o.Days != nil {
		rows, err := w.rowsOf(i, "ids", o.IDs)
		if err != nil {
			return err
		}
		for _, a := range rows {
			if kids := w.kids[a.ID]; len(kids) > 0 {
				for _, k := range kids {
					if _, dup := desired[k]; !dup {
						if err := add(w.acts[k], *o.Days, "ids"); err != nil {
							return err
						}
					}
				}
			} else if _, dup := desired[a.ID]; !dup {
				if err := add(a, *o.Days, "ids"); err != nil {
					return err
				}
			}
		}
	} else {
		a, err := w.row(i, "id", o.ID)
		if err != nil {
			return err
		}
		to, _ := cal.IndexOnOrAfter(*o.StartDate)
		if kids := w.kids[a.ID]; len(kids) > 0 {
			t := times[a.ID]
			if t.Unknown {
				return opErr(i, "id", a.Label+" has no dates yet.")
			}
			for _, k := range kids {
				if err := add(w.acts[k], to-t.ES, "id"); err != nil {
					return err
				}
			}
		} else if times[a.ID].Unknown {
			// Nothing to check against yet: the date is what the user asked for.
			a = w.touchAct(a.ID)
			if a.ConstraintKind != engine.ConstraintMSO {
				a.ConstraintKind = engine.ConstraintSNET
			}
			a.ConstraintDate = NullText(cal.DateOf(max(0, to)))
			edited(a)
			return nil
		} else if err := add(a, to-times[a.ID].ES, "id"); err != nil {
			return err
		}
	}
	// Nothing starts before the program; every other hold is found by
	// computing with all the new constraints at once, so a bulk move is
	// judged as one.
	var held []string
	for _, a := range targets {
		if desired[a.ID] < 0 {
			held = append(held, a.ID)
			continue
		}
		a = w.touchAct(a.ID)
		if a.ConstraintKind != engine.ConstraintMSO {
			a.ConstraintKind = engine.ConstraintSNET
		}
		a.ConstraintDate = NullText(cal.DateOf(desired[a.ID]))
		edited(a)
	}
	out1 := w.output()
	times = out1.Activities.Map()
	for _, a := range targets {
		if desired[a.ID] >= 0 && times[a.ID].ES != desired[a.ID] {
			held = append(held, a.ID)
		}
	}
	if len(held) > 0 {
		return w.heldError(i, held, desired, &out1)
	}
	// A link that now sets the start makes the constraint redundant: drop it,
	// so dragging earlier removes what a drag later added.
	driven := map[string]bool{}
	for _, id := range out1.DrivingLinks {
		driven[w.links[id].SuccID] = true
	}
	for _, a := range targets {
		if a.ConstraintKind == engine.ConstraintSNET && (driven[a.ID] || desired[a.ID] == 0) {
			a.ConstraintKind, a.ConstraintDate = engine.ConstraintASAP, ""
			w.dirty = true
		}
	}
	return nil
}

// heldError names what holds each row ("Held by Slab").
func (w *programWork) heldError(i int, held []string, desired map[string]int, out *engine.Output) error {
	pos := w.positions()
	sort.SliceStable(held, func(x, y int) bool { return pos[held[x]] < pos[held[y]] })
	by := func(id string) string {
		if desired[id] < 0 {
			return "the program start"
		}
		for _, lid := range out.DrivingLinks {
			if l := w.links[lid]; l.SuccID == id {
				return w.acts[l.PredID].Label
			}
		}
		if p := w.acts[id].ParentID; p != "" {
			return w.acts[string(p)].Label
		}
		return "the program start"
	}
	var parts []string
	for _, id := range held {
		parts = append(parts, fmt.Sprintf("%s is held by %s", w.acts[id].Label, by(id)))
	}
	msg := strings.Join(parts, "; ") + "."
	if len(held) == 1 {
		msg = "Held by " + by(held[0])
	}
	return &ProgramOpError{Index: i, Field: "ids", Message: msg, Held: held}
}

func (w *programWork) opDelete(i int, o ProgramOp) error {
	rows, err := w.rowsOf(i, "ids", o.IDs)
	if err != nil {
		return err
	}
	reason := ""
	if o.Reason != nil {
		reason = strings.TrimSpace(*o.Reason)
	}
	gone := map[string]bool{}
	for _, a := range rows {
		gone[a.ID] = true
		for _, k := range w.kids[a.ID] {
			gone[k] = true
		}
	}
	for id := range gone {
		if w.acts[id].Kind == engine.KindHoldPoint && reason == "" {
			return opErr(i, "reason", MsgHoldPointReason)
		}
	}
	// Stages left with no activities become leaves with their last span,
	// read from the program as it was before the delete.
	var emptied []string
	for _, a := range rows {
		if p := string(a.ParentID); p != "" && !gone[p] {
			left := 0
			for _, k := range w.kids[p] {
				if !gone[k] {
					left++
				}
			}
			if left == 0 && indexOf(emptied, p) < 0 {
				emptied = append(emptied, p)
			}
		}
	}
	var out0 engine.Output
	if len(emptied) > 0 {
		out0 = w.output()
	}
	// Chains through deleted rows heal: each live row with a chain into the
	// deleted run is linked to each live row the run's chains lead to.
	chainOut := map[string][]string{}
	var heal [][2]string
	for _, l := range w.liveLinks() {
		if l.Origin == "chain" {
			chainOut[l.PredID] = append(chainOut[l.PredID], l.SuccID)
		}
	}
	for _, l := range w.liveLinks() {
		if l.Origin != "chain" || gone[l.PredID] || !gone[l.SuccID] {
			continue
		}
		seen := map[string]bool{}
		queue := []string{l.SuccID}
		for len(queue) > 0 {
			x := queue[0]
			queue = queue[1:]
			if seen[x] {
				continue
			}
			seen[x] = true
			for _, s := range chainOut[x] {
				if gone[s] {
					queue = append(queue, s)
				} else {
					heal = append(heal, [2]string{l.PredID, s})
				}
			}
		}
	}
	for _, l := range w.liveLinks() {
		if gone[l.PredID] || gone[l.SuccID] {
			w.retireLink(l.ID)
		}
	}
	for id := range gone {
		a := w.touchAct(id)
		a.Retired = true
		if a.Kind == engine.KindHoldPoint {
			a.DismissedReason = NullText(reason)
		}
		if a.Kind == engine.KindStage {
			w.stageChange = true
		}
	}
	w.stages = without(w.stages, gone)
	for s := range gone {
		delete(w.kids, s)
	}
	for s, k := range w.kids {
		w.setKids(s, without(k, gone))
	}
	w.dirty = true
	for _, p := range heal {
		if w.linked(p[0], p[1]) {
			continue
		}
		if l, ok := w.tryLink(p[0], p[1], "chain"); ok {
			w.fb.HealedLinks = append(w.fb.HealedLinks, *l)
		}
	}
	for _, s := range emptied {
		w.lastChild(s, out0)
	}
	return nil
}

// ownStageLink refuses a move that would leave a row linked to its own
// stage (plan 5: never a row and its own parent).
func (w *programWork) ownStageLink(i int, row, stage *ProgramActivity) error {
	for _, p := range [][2]string{{row.ID, stage.ID}, {stage.ID, row.ID}} {
		if id, ok := w.pairs[p]; ok && w.links[id].Origin != "chain" {
			return opErr(i, "ids", fmt.Sprintf("%s is linked to %s; remove that link first.", row.Label, stage.Label))
		}
	}
	return nil
}

// demote turns a childless stage into the last activity (or the activity
// after `after`) of stage `into`.
func (w *programWork) demote(i int, a, into *ProgramActivity, after string) error {
	if len(w.kids[a.ID]) > 0 {
		return opErr(i, "ids", MsgTwoTiers)
	}
	if err := w.ownStageLink(i, a, into); err != nil {
		return err
	}
	oldStages := w.stages
	w.stages = without(w.stages, map[string]bool{a.ID: true})
	oldKids := w.kids[into.ID]
	w.setKids(into.ID, insertAfter(oldKids, a.ID, after))
	a = w.touchAct(a.ID)
	a.Kind, a.ParentID, a.CatalogueStageID = engine.KindActivity, NullText(into.ID), ""
	edited(a)
	w.stageChange = true
	if len(oldKids) == 0 {
		w.firstChild(into, a)
	}
	w.rechain(oldStages, w.stages, true)
	w.rechain(oldKids, w.kids[into.ID], false)
	return nil
}

func (w *programWork) opIndent(i int, o ProgramOp) error {
	rows, err := w.rowsOf(i, "ids", o.IDs)
	if err != nil {
		return err
	}
	for _, a := range rows {
		if a.Kind != engine.KindStage {
			return opErr(i, "ids", MsgTwoTiers)
		}
		k := indexOf(w.stages, a.ID)
		if k == 0 {
			return opErr(i, "ids", fmt.Sprintf("There is no stage above %s to hold it.", a.Label))
		}
		above := w.acts[w.stages[k-1]]
		kids := w.kids[above.ID]
		after := ""
		if len(kids) > 0 {
			after = kids[len(kids)-1]
		}
		if err := w.demote(i, a, above, after); err != nil {
			return err
		}
	}
	return nil
}

func (w *programWork) opOutdent(i int, o ProgramOp) error {
	rows, err := w.rowsOf(i, "ids", o.IDs)
	if err != nil {
		return err
	}
	var out0 engine.Output
	computed := false
	lastPlaced := map[string]string{}
	for _, a := range rows {
		if a.Kind == engine.KindStage {
			return opErr(i, "ids", MsgTwoTiers)
		}
		parent := string(a.ParentID)
		oldKids := w.kids[parent]
		if len(oldKids) == 1 && !computed {
			out0, computed = w.output(), true
		}
		w.setKids(parent, without(oldKids, map[string]bool{a.ID: true}))
		after := parent
		if p, ok := lastPlaced[parent]; ok {
			after = p
		}
		oldStages := w.stages
		w.stages = insertAfter(w.stages, a.ID, after)
		lastPlaced[parent] = a.ID
		a = w.touchAct(a.ID)
		a.Kind, a.ParentID, a.CatalogueStageID = engine.KindStage, "", knowledge.OtherStageID
		edited(a)
		w.stageChange = true
		w.rechain(oldKids, w.kids[parent], false)
		w.rechain(oldStages, w.stages, true)
		if len(w.kids[parent]) == 0 {
			w.lastChild(parent, out0)
		}
	}
	return nil
}

func (w *programWork) opReorder(i int, o ProgramOp) error {
	rows, err := w.rowsOf(i, "ids", o.IDs)
	if err != nil {
		return err
	}
	moving := map[string]bool{}
	for _, a := range rows {
		moving[a.ID] = true
	}
	after := ""
	if o.AfterID != "" {
		after = w.resolve(o.AfterID)
		if moving[after] {
			return opErr(i, "after_id", "Rows can't move after themselves.")
		}
	}
	posBefore := w.positions()
	chains := len(w.fb.ChainAdded) + len(w.fb.ChainRemoved)
	if o.ParentID == "" {
		for _, a := range rows {
			if a.Kind != engine.KindStage {
				return opErr(i, "parent_id", "Outdent an activity to make it a stage.")
			}
		}
		if after != "" && indexOf(w.stages, after) < 0 {
			return opErr(i, "after_id", MsgNotInProgram)
		}
		old := w.stages
		rest := without(old, moving)
		next := make([]string, 0, len(old))
		if after == "" {
			next = append(next, idsOf(rows)...)
		}
		for _, s := range rest {
			next = append(next, s)
			if s == after {
				next = append(next, idsOf(rows)...)
			}
		}
		if strings.Join(next, ",") != strings.Join(old, ",") {
			w.stages = next
			w.dirty = true
			w.stageChange = true
			w.rechain(old, next, true)
		}
	} else {
		into, err := w.row(i, "parent_id", o.ParentID)
		if err != nil {
			return err
		}
		if into.Kind != engine.KindStage || moving[into.ID] {
			return opErr(i, "parent_id", MsgTwoTiers)
		}
		if after != "" && indexOf(w.kids[into.ID], after) < 0 {
			return opErr(i, "after_id", MsgNotInProgram)
		}
		var out0 engine.Output
		sources := []string{}
		for _, a := range rows {
			if a.Kind == engine.KindStage {
				if len(w.kids[a.ID]) > 0 {
					return opErr(i, "ids", MsgTwoTiers)
				}
				continue
			}
			if err := w.ownStageLink(i, a, into); err != nil {
				return err
			}
			if p := string(a.ParentID); p != into.ID && indexOf(sources, p) < 0 {
				sources = append(sources, p)
			}
		}
		for _, p := range sources {
			if len(without(w.kids[p], moving)) == 0 {
				out0 = w.output()
				break
			}
		}
		oldInto := append([]string(nil), w.kids[into.ID]...)
		oldSources := map[string][]string{}
		for _, p := range sources {
			oldSources[p] = w.kids[p]
			w.setKids(p, without(w.kids[p], moving))
		}
		// Rows go in their current order after `after`, or first.
		place := after
		for _, a := range rows {
			if a.Kind == engine.KindStage {
				if err := w.demote(i, a, into, place); err != nil {
					return err
				}
			} else {
				kids := without(w.kids[into.ID], map[string]bool{a.ID: true})
				w.setKids(into.ID, insertAfter(kids, a.ID, place))
				if string(a.ParentID) != into.ID {
					a = w.touchAct(a.ID)
					a.ParentID = NullText(into.ID)
				}
			}
			place = a.ID
		}
		if len(oldInto) == 0 && len(w.kids[into.ID]) > 0 {
			w.firstChild(into, w.acts[w.kids[into.ID][0]])
		}
		for _, p := range sources {
			w.rechain(oldSources[p], w.kids[p], false)
		}
		w.rechain(oldInto, w.kids[into.ID], false)
		for _, p := range sources {
			if len(w.kids[p]) == 0 {
				w.lastChild(p, out0)
			}
		}
	}
	// User links never change on a reorder; name any that now run backwards.
	posAfter := w.positions()
	for _, l := range w.liveLinks() {
		if l.Origin == "chain" {
			continue
		}
		if posAfter[l.PredID] > posAfter[l.SuccID] && posBefore[l.PredID] < posBefore[l.SuccID] {
			w.fb.BackwardLinks = append(w.fb.BackwardLinks, *l)
			w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("%s still waits for %s.", w.acts[l.SuccID].Label, w.acts[l.PredID].Label))
		}
	}
	if len(w.fb.ChainAdded)+len(w.fb.ChainRemoved) > chains {
		what := rows[0].Label
		if len(rows) > 1 {
			what = fmt.Sprintf("%d rows", len(rows))
		}
		where := "to the top"
		if after != "" {
			where = "after " + w.acts[after].Label
		}
		w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("Moved %s %s; sequence updated.", what, where))
	}
	return nil
}

func idsOf(rows []*ProgramActivity) []string {
	out := make([]string, len(rows))
	for i, a := range rows {
		out[i] = a.ID
	}
	return out
}

// linkError turns an engine refusal into a 422.
func linkError(i int, field string, err error) error {
	var le *engine.LinkError
	if errors.As(err, &le) {
		return &ProgramOpError{Index: i, Field: field, Message: le.Message, Cycle: le.Cycle}
	}
	return opErr(i, field, err.Error())
}

func (w *programWork) opLinkAdd(i int, o ProgramOp) error {
	pred, err := w.row(i, "pred_id", o.PredID)
	if err != nil {
		return err
	}
	succ, err := w.row(i, "succ_id", o.SuccID)
	if err != nil {
		return err
	}
	l := engine.Link{Pred: pred.ID, Succ: succ.ID, Type: engine.LinkFS, LagUnit: engine.LagWorking}
	if o.Type != nil {
		l.Type = *o.Type
	}
	if o.LagDays != nil {
		l.Lag = *o.LagDays
	}
	if o.LagUnit != nil {
		l.LagUnit = *o.LagUnit
	}
	if err := engine.ValidateLink(w.input(), l); err != nil {
		return linkError(i, "succ_id", err)
	}
	w.addLink(l.Pred, l.Succ, l.Type, l.Lag, l.LagUnit, "user")
	return nil
}

// opLinkChain is "Link in sequence": FS, no lag, between each consecutive
// pair in grid order, skipping pairs already linked and any the engine
// refuses (a loop, a row and its own stage).
func (w *programWork) opLinkChain(i int, o ProgramOp) error {
	rows, err := w.rowsOf(i, "ids", o.IDs)
	if err != nil {
		return err
	}
	for k := 1; k < len(rows); k++ {
		a, b := rows[k-1].ID, rows[k].ID
		if w.linked(a, b) {
			continue
		}
		if l, ok := w.tryLink(a, b, "user"); ok {
			w.fb.Linked = append(w.fb.Linked, *l)
		}
	}
	return nil
}

func (w *programWork) opLinkUpdate(i int, o ProgramOp) error {
	cur := w.links[w.resolve(o.ID)]
	if cur == nil || cur.Retired {
		return opErr(i, "id", MsgLinkNotFound)
	}
	l := engine.Link{ID: cur.ID, Pred: cur.PredID, Succ: cur.SuccID, Type: cur.Type, Lag: cur.LagDays, LagUnit: cur.LagUnit}
	if o.Type != nil {
		l.Type = *o.Type
	}
	if o.LagDays != nil {
		l.Lag = *o.LagDays
	}
	if o.LagUnit != nil {
		l.LagUnit = *o.LagUnit
	}
	if err := engine.ValidateLink(w.input(), l); err != nil {
		return linkError(i, "id", err)
	}
	if l.Type == cur.Type && l.Lag == cur.LagDays && l.LagUnit == cur.LagUnit {
		return nil
	}
	x := w.touchLink(cur.ID)
	x.Type, x.LagDays, x.LagUnit, x.Origin = l.Type, l.Lag, l.LagUnit, "user"
	return nil
}

func (w *programWork) opLinkDelete(i int, o ProgramOp) error {
	for _, ref := range o.IDs {
		l := w.links[w.resolve(ref)]
		if l == nil || l.Retired {
			return opErr(i, "ids", MsgLinkNotFound)
		}
		w.retireLink(l.ID)
	}
	return nil
}

func (w *programWork) opSetStart(o ProgramOp) error {
	if o.StartDate == nil || *o.StartDate == "" {
		w.prog.StartDate, w.prog.StartOrigin = "", ""
	} else {
		w.prog.StartDate, w.prog.StartOrigin = NullText(*o.StartDate), "user"
	}
	w.dirty = true
	return nil
}

func (w *programWork) opCalendarDay(o ProgramOp) error {
	w.touchDay(o.Day)
	if o.Kind == nil {
		delete(w.days, o.Day)
		return nil
	}
	w.days[o.Day] = ProgramCalendarDay{Day: o.Day, Kind: *o.Kind, Label: strings.TrimSpace(*o.Label), Origin: "user"}
	return nil
}
