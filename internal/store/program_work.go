package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
)

// programWork is one write held in memory: the program's live inputs as
// loaded under the row lock, edited by ops, then diffed and written back in
// a handful of statements. Every row is touched before it changes, so the
// first image of each row is kept for the undo log and the diff.
type programWork struct {
	org, project string
	prog         Program
	progBefore   Program

	acts   map[string]*ProgramActivity
	stages []string            // live stages in order
	kids   map[string][]string // live children of each stage in order

	links     map[string]*ProgramLink
	linkOrder []string
	pairs     map[[2]string]string // live link id by (pred, succ)

	days map[string]ProgramCalendarDay

	actBefore  map[string]*ProgramActivity // first image of a touched row
	actNew     map[string]bool
	actOrder   []string // touch order, for a stable write and undo log
	linkBefore map[string]*ProgramLink
	linkNew    map[string]bool
	linkTouch  []string
	dayBefore  map[string]*ProgramCalendarDay // nil value: the day was absent
	dayTouch   []string

	stageChange bool
	temp        map[string]string
	// genReason is the reason a generated delete gives a hold point; empty
	// means the question strip's answer.
	genReason string
	// catalogue reports a delivery stage id; nil when no catalogue is loaded,
	// which leaves only `other` acceptable.
	catalogue func(string) bool

	fb ProgramFeedback

	dirty bool
	in    engine.Input
	out   engine.Output
}

// ProgramFeedback is what a batch did beyond the new state, for the toasts:
// temp_id → id, links healed by a delete, chain links a reorder (or add,
// indent, outdent) added or removed, user links a reorder left running
// backwards, links made by link_chain, and toast texts.
type ProgramFeedback struct {
	TempIDs       map[string]string `json:"temp_ids"`
	HealedLinks   []ProgramLink     `json:"healed_links"`
	ChainAdded    []ProgramLink     `json:"chain_added"`
	ChainRemoved  []ProgramLink     `json:"chain_removed"`
	BackwardLinks []ProgramLink     `json:"backward_links"`
	Linked        []ProgramLink     `json:"linked"`
	Toasts        []string          `json:"toasts"`
}

func newProgramWork(org, project string, p Program, acts []ProgramActivity, links []ProgramLink, days []ProgramCalendarDay) *programWork {
	w := &programWork{org: org, project: project, prog: p, progBefore: p,
		acts: make(map[string]*ProgramActivity, len(acts)), kids: map[string][]string{},
		links: make(map[string]*ProgramLink, len(links)), pairs: make(map[[2]string]string, len(links)),
		days: map[string]ProgramCalendarDay{}, actBefore: map[string]*ProgramActivity{}, actNew: map[string]bool{},
		linkBefore: map[string]*ProgramLink{}, linkNew: map[string]bool{}, dayBefore: map[string]*ProgramCalendarDay{},
		temp: map[string]string{}, dirty: true}
	w.fb = ProgramFeedback{TempIDs: w.temp, HealedLinks: []ProgramLink{}, ChainAdded: []ProgramLink{}, ChainRemoved: []ProgramLink{},
		BackwardLinks: []ProgramLink{}, Linked: []ProgramLink{}, Toasts: []string{}}
	for i := range acts { // outline order
		a := acts[i]
		w.acts[a.ID] = &a
		if a.ParentID == "" {
			w.stages = append(w.stages, a.ID)
		} else {
			w.kids[string(a.ParentID)] = append(w.kids[string(a.ParentID)], a.ID)
		}
	}
	for i := range links {
		l := links[i]
		w.links[l.ID] = &l
		w.linkOrder = append(w.linkOrder, l.ID)
		w.pairs[[2]string{l.PredID, l.SuccID}] = l.ID
	}
	for _, d := range days {
		w.days[d.Day] = d
	}
	return w
}

// addRetired brings retired rows into the work so undo can restore them.
func (w *programWork) addRetired(acts []ProgramActivity, links []ProgramLink) {
	for i := range acts {
		if _, ok := w.acts[acts[i].ID]; !ok {
			a := acts[i]
			w.acts[a.ID] = &a
		}
	}
	for i := range links {
		if _, ok := w.links[links[i].ID]; !ok {
			l := links[i]
			w.links[l.ID] = &l
			w.linkOrder = append(w.linkOrder, l.ID)
		}
	}
}

func (w *programWork) resolve(ref string) string {
	if id, ok := w.temp[ref]; ok {
		return id
	}
	return ref
}

func (w *programWork) touchAct(id string) *ProgramActivity {
	a := w.acts[id]
	if _, seen := w.actBefore[id]; !seen && !w.actNew[id] {
		c := *a
		w.actBefore[id] = &c
		w.actOrder = append(w.actOrder, id)
	}
	w.dirty = true
	return a
}

func (w *programWork) touchLink(id string) *ProgramLink {
	l := w.links[id]
	if _, seen := w.linkBefore[id]; !seen && !w.linkNew[id] {
		c := *l
		w.linkBefore[id] = &c
		w.linkTouch = append(w.linkTouch, id)
	}
	w.dirty = true
	return l
}

func (w *programWork) touchDay(day string) {
	if _, seen := w.dayBefore[day]; !seen {
		if d, ok := w.days[day]; ok {
			w.dayBefore[day] = &d
		} else {
			w.dayBefore[day] = nil
		}
		w.dayTouch = append(w.dayTouch, day)
	}
	w.dirty = true
}

func (w *programWork) newAct(a ProgramActivity) *ProgramActivity {
	a.ID, a.Version = newID(), 1
	if a.Systems == nil {
		a.Systems = []string{}
	}
	if len(a.DurationSource) == 0 {
		a.DurationSource = json.RawMessage(`{}`)
	}
	w.acts[a.ID] = &a
	w.actNew[a.ID] = true
	w.actOrder = append(w.actOrder, a.ID)
	w.dirty = true
	return &a
}

func (w *programWork) addLink(pred, succ, typ string, lag int, unit, origin string) *ProgramLink {
	l := &ProgramLink{ID: newID(), PredID: pred, SuccID: succ, Type: typ, LagDays: lag, LagUnit: unit, Origin: origin, Version: 1}
	w.links[l.ID] = l
	w.linkOrder = append(w.linkOrder, l.ID)
	w.linkNew[l.ID] = true
	w.linkTouch = append(w.linkTouch, l.ID)
	w.pairs[[2]string{pred, succ}] = l.ID
	w.dirty = true
	return l
}

func (w *programWork) retireLink(id string) {
	l := w.touchLink(id)
	if l.Retired {
		return
	}
	l.Retired = true
	if w.pairs[[2]string{l.PredID, l.SuccID}] == id {
		delete(w.pairs, [2]string{l.PredID, l.SuccID})
	}
}

// linked reports a live link between a and b in either direction.
func (w *programWork) linked(a, b string) bool {
	_, ab := w.pairs[[2]string{a, b}]
	_, ba := w.pairs[[2]string{b, a}]
	return ab || ba
}

func (w *programWork) live(id string) *ProgramActivity {
	if a := w.acts[id]; a != nil && !a.Retired {
		return a
	}
	return nil
}

func (w *programWork) liveLinks() []*ProgramLink {
	out := make([]*ProgramLink, 0, len(w.pairs))
	for _, id := range w.linkOrder {
		if l := w.links[id]; !l.Retired {
			out = append(out, l)
		}
	}
	return out
}

func (w *programWork) countLive() (rows, links int) {
	rows = len(w.stages)
	for _, k := range w.kids {
		rows += len(k)
	}
	return rows, len(w.pairs)
}

// order lists live rows in outline order.
func (w *programWork) order() []string {
	out := make([]string, 0, len(w.acts))
	for _, s := range w.stages {
		out = append(out, s)
		out = append(out, w.kids[s]...)
	}
	return out
}

// positions numbers live rows in outline order.
func (w *programWork) positions() map[string]int {
	pos := map[string]int{}
	for i, id := range w.order() {
		pos[id] = i
	}
	return pos
}

// input is the engine input for the current state, rebuilt only after a
// change.
func (w *programWork) input() engine.Input {
	if !w.dirty {
		return w.in
	}
	ids := w.order()
	in := engine.Input{Calendar: engine.Calendar{Start: string(w.prog.StartDate), WorkingWeek: w.prog.WorkingWeek},
		Activities: make([]engine.Activity, len(ids)), Links: make([]engine.Link, 0, len(w.pairs))}
	for i, id := range ids {
		in.Activities[i] = engineActivity(w.acts[id])
	}
	for _, l := range w.liveLinks() {
		in.Links = append(in.Links, engine.Link{ID: l.ID, Pred: l.PredID, Succ: l.SuccID, Type: l.Type, Lag: l.LagDays, LagUnit: l.LagUnit})
	}
	for _, d := range w.sortedDays() {
		in.Calendar.Days = append(in.Calendar.Days, engine.CalendarDay{Day: d.Day, Kind: d.Kind, Label: d.Label})
	}
	w.in, w.dirty = in, false
	w.out = engine.Compute(in)
	return in
}

func (w *programWork) output() engine.Output {
	w.input()
	return w.out
}

func (w *programWork) sortedDays() []ProgramCalendarDay {
	out := make([]ProgramCalendarDay, 0, len(w.days))
	for _, d := range w.days {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out
}

// state is the live inputs and their computed output.
func (w *programWork) state() ProgramState {
	ids := w.order()
	acts := make([]ProgramActivity, len(ids))
	for i, id := range ids {
		acts[i] = *w.acts[id]
	}
	ll := w.liveLinks()
	links := make([]ProgramLink, len(ll))
	for i, l := range ll {
		links[i] = *l
	}
	return ProgramState{Program: w.prog, Activities: acts, Links: links, Calendar: ProgramCalendar{Days: w.sortedDays()}, Computed: w.output()}
}

// renumber makes ordinals follow the sequences, touching only rows whose
// ordinal moves.
func (w *programWork) renumber() {
	set := func(id string, n int) {
		if w.acts[id].Ordinal != n {
			w.touchAct(id).Ordinal = n
		}
	}
	for i, s := range w.stages {
		set(s, i)
		for j, k := range w.kids[s] {
			set(k, j)
		}
	}
}

// checkStructure is the last guard before a write: two tiers, live parents
// and live link ends. Ops keep these by construction; undo can meet a
// program changed since, and is refused instead.
func (w *programWork) checkStructure() error {
	for _, id := range w.order() {
		a := w.acts[id]
		if a.ParentID != "" {
			p := w.live(string(a.ParentID))
			if p == nil || p.Kind != engine.KindStage {
				return errProgramStructure
			}
		} else if a.Kind != engine.KindStage {
			return errProgramStructure
		}
	}
	for _, l := range w.liveLinks() {
		if w.live(l.PredID) == nil || w.live(l.SuccID) == nil {
			return errProgramStructure
		}
	}
	return nil
}

var errProgramStructure = errors.New("program structure broken")

func sameActivity(a, b ProgramActivity) bool {
	a.Version, b.Version = 0, 0
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func sameLink(a, b ProgramLink) bool {
	a.Version, b.Version = 0, 0
	return a == b
}

// programChanges is a diff ready to write and to log for undo.
type programChanges struct {
	insActs, updActs   []ProgramActivity
	insLinks, updLinks []ProgramLink
	upsDays            []ProgramCalendarDay
	delDays            []string
	inverse            []programInverse
}

// programInverse is one entry of a batch's inverse: the image of a row
// before and after the batch. Undo writes Before when the row still equals
// After; redo writes After when it still equals Before. Images carry the
// row version, so an undo restores the exact earlier row, and a second undo
// of an older batch on the same row finds the version that batch produced.
type programInverse struct {
	Op     string          `json:"op"` // restore or stage_revision
	Target string          `json:"target,omitempty"`
	ID     string          `json:"id,omitempty"`
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
}

// diff collects every changed row with its new version (one more than its
// first image) and the inverse of the batch.
func (w *programWork) diff() programChanges {
	var c programChanges
	img := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	for _, id := range w.actOrder {
		a := w.acts[id]
		if w.actNew[id] {
			c.insActs = append(c.insActs, *a)
			gone := *a
			gone.Retired, gone.Version = true, a.Version+1
			c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "activity", ID: id, Before: img(gone), After: img(a)})
			continue
		}
		before := w.actBefore[id]
		if sameActivity(*before, *a) {
			*a = *before
			continue
		}
		a.Version = before.Version + 1
		c.updActs = append(c.updActs, *a)
		c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "activity", ID: id, Before: img(before), After: img(a)})
	}
	for _, id := range w.linkTouch {
		l := w.links[id]
		if w.linkNew[id] {
			c.insLinks = append(c.insLinks, *l)
			gone := *l
			gone.Retired, gone.Version = true, l.Version+1
			c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "link", ID: id, Before: img(gone), After: img(l)})
			continue
		}
		before := w.linkBefore[id]
		if sameLink(*before, *l) {
			*l = *before
			continue
		}
		l.Version = before.Version + 1
		c.updLinks = append(c.updLinks, *l)
		c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "link", ID: id, Before: img(before), After: img(l)})
	}
	for _, day := range w.dayTouch {
		before := w.dayBefore[day]
		after, ok := w.days[day]
		var afterImg json.RawMessage = json.RawMessage("null")
		if ok {
			afterImg = img(after)
		}
		var beforeImg json.RawMessage = json.RawMessage("null")
		if before != nil {
			beforeImg = img(before)
		}
		if string(beforeImg) == string(afterImg) {
			continue
		}
		if ok {
			c.upsDays = append(c.upsDays, after)
		} else {
			c.delDays = append(c.delDays, day)
		}
		c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "day", ID: day, Before: beforeImg, After: afterImg})
	}
	if w.prog.StartDate != w.progBefore.StartDate || w.prog.StartOrigin != w.progBefore.StartOrigin {
		c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "program", ID: w.prog.ID, Before: img(programStart(w.progBefore)), After: img(programStart(w.prog))})
	}
	if w.stageChange {
		c.inverse = append(c.inverse, programInverse{Op: "stage_revision"})
	}
	return c
}

// startImage is the part of the program row an op can change.
type startImage struct {
	StartDate   NullText `json:"start_date"`
	StartOrigin NullText `json:"start_origin"`
}

func programStart(p Program) startImage { return startImage{p.StartDate, p.StartOrigin} }

const activityUpdate = `UPDATE program_activities a SET parent_id=r.parent_id,ordinal=r.ordinal,kind=r.kind,label=r.label,catalogue_stage_id=r.catalogue_stage_id,package_id=r.package_id,package_stage_id=r.package_stage_id,work_item_id=r.work_item_id,fragment_id=r.fragment_id,interface_id=r.interface_id,systems=r.systems,duration_days=r.duration_days,duration_low=r.duration_low,duration_high=r.duration_high,duration_origin=r.duration_origin,duration_source=r.duration_source,constraint_kind=r.constraint_kind,constraint_date=r.constraint_date,review_status=r.review_status,origin=r.origin,source_key=r.source_key,dismissed_reason=r.dismissed_reason,retired_at=CASE WHEN (e->>'retired')::boolean THEN COALESCE(a.retired_at,now()) END,version=r.version
FROM jsonb_array_elements($3::jsonb) e, LATERAL jsonb_populate_record(NULL::program_activities,e) r
WHERE a.org_id=$1::uuid AND a.program_id=$2::uuid AND a.id=r.id`

const linkUpdate = `UPDATE program_links l SET pred_id=r.pred_id,succ_id=r.succ_id,type=r.type,lag_days=r.lag_days,lag_unit=r.lag_unit,origin=r.origin,interface_id=r.interface_id,source_key=r.source_key,retired_at=CASE WHEN (e->>'retired')::boolean THEN COALESCE(l.retired_at,now()) END,version=r.version
FROM jsonb_array_elements($3::jsonb) e, LATERAL jsonb_populate_record(NULL::program_links,e) r
WHERE l.org_id=$1::uuid AND l.program_id=$2::uuid AND l.id=r.id`

// write sends the diff in one batch. Rows leaving the live set go first and
// rows joining it last, because the live-only unique indexes (one live link
// per pair, one live row per source key) are checked row by row.
func (c programChanges) write(ctx context.Context, tx pgx.Tx, org, programID string) error {
	type stmt struct {
		sql  string
		rows any
		n    int
	}
	split := func(acts []ProgramActivity) (gone, live []ProgramActivity) {
		for _, a := range acts {
			if a.Retired {
				gone = append(gone, a)
			} else {
				live = append(live, a)
			}
		}
		return
	}
	splitLinks := func(links []ProgramLink) (gone, live []ProgramLink) {
		for _, l := range links {
			if l.Retired {
				gone = append(gone, l)
			} else {
				live = append(live, l)
			}
		}
		return
	}
	goneActs, liveActs := split(c.updActs)
	goneLinks, liveLinks := splitLinks(c.updLinks)
	stmts := []stmt{
		{linkUpdate, goneLinks, len(goneLinks)},
		{activityUpdate, goneActs, len(goneActs)},
		{activityInsert, c.insActs, len(c.insActs)},
		{activityUpdate, liveActs, len(liveActs)},
		{linkInsert, c.insLinks, len(c.insLinks)},
		{linkUpdate, liveLinks, len(liveLinks)},
	}
	b := &pgx.Batch{}
	var want []int
	for _, s := range stmts {
		if s.n == 0 {
			continue
		}
		raw, err := json.Marshal(s.rows)
		if err != nil {
			return err
		}
		b.Queue(s.sql, org, programID, raw)
		want = append(want, s.n)
	}
	if len(c.upsDays) > 0 {
		raw, err := json.Marshal(c.upsDays)
		if err != nil {
			return err
		}
		b.Queue(dayInsert+` ON CONFLICT (org_id,program_id,day) DO UPDATE SET kind=EXCLUDED.kind,label=EXCLUDED.label,origin=EXCLUDED.origin`, org, programID, raw)
		want = append(want, len(c.upsDays))
	}
	if len(c.delDays) > 0 {
		b.Queue(`DELETE FROM program_calendar_days WHERE org_id=$1::uuid AND program_id=$2::uuid AND day::text=ANY($3::text[])`, org, programID, c.delDays)
		want = append(want, len(c.delDays))
	}
	if b.Len() == 0 {
		return nil
	}
	br := tx.SendBatch(ctx, b)
	for _, n := range want {
		tag, err := br.Exec()
		if err != nil {
			br.Close()
			return err
		}
		if tag.RowsAffected() != int64(n) {
			br.Close()
			return fmt.Errorf("program write touched %d rows, want %d", tag.RowsAffected(), n)
		}
	}
	return br.Close()
}
