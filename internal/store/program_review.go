package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// The ask-first review (plan docs/plans/2026-10-18-program.md section 7.4).
// After a program exists, a change made elsewhere never edits it: the review
// regenerates in memory, diffs the live program by source key and writes one
// program_changes row per difference. Only Adopt, through the normal program
// write path, changes a program row. No path here calls Jev.

// MsgChangeGone refuses a decision on a change that is no longer pending.
const MsgChangeGone = "That change is no longer pending. Refresh the review."

// MaxProgramDecisions bounds one decide request.
const MaxProgramDecisions = MaxProgramRows

// ProgramReviewOutcome is what one review run did.
type ProgramReviewOutcome struct {
	ProgramID string
	// Changed is true when the pending set changed (a row raised or
	// superseded), so open views should refetch the review.
	Changed bool
	Pending int
}

// pendingChange is a change ready to store: the engine's change plus its
// basis and its place in the diff.
type pendingChange struct {
	c       engine.Change
	basis   string
	ordinal int
}

// ReviewProgram is the program_review job body. It writes program_changes
// only: never a program row, never the program version. A project with no
// program is not an error; nothing is done. The program row is locked so a
// run never interleaves with a decision.
func (s *Store) ReviewProgram(ctx context.Context, org, project string, cat *knowledge.Catalog) (ProgramReviewOutcome, error) {
	var out ProgramReviewOutcome
	if cat == nil {
		return out, errors.New("program review needs the knowledge catalogue")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	p, acts, links, _, err := readProgramInputs(ctx, tx, org, project, true)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.ProgramID = p.ID
	in, err := readProgramGenInput(ctx, tx, org, project, cat)
	if err != nil {
		return out, err
	}
	plan, err := engine.Generate(cat, genInputFor(in.gi, p, acts))
	if err != nil {
		return out, err
	}
	changes := reviewChanges(engine.Diff(liveRows(acts), liveLinks(links), plan), acts, links)

	type stored struct {
		id, basis, status string
	}
	pending := map[string]stored{}
	latest := map[string]stored{} // newest row per source key, any status
	rows, err := tx.Query(ctx, `SELECT id::text,source_key,COALESCE(basis,''),status FROM program_changes
 WHERE org_id=$1::uuid AND program_id=$2::uuid ORDER BY created_at,ordinal,id`, org, p.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var key string
		var r stored
		if err := rows.Scan(&r.id, &key, &r.basis, &r.status); err != nil {
			rows.Close()
			return out, err
		}
		latest[key] = r
		if r.status == "pending" {
			pending[key] = r
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	var supersede, keepIDs []string
	var keepOrd []int
	var insert []pendingChange
	seen := map[string]bool{}
	for _, pc := range changes {
		key := pc.c.SourceKey
		seen[key] = true
		if cur, ok := pending[key]; ok {
			if cur.basis == pc.basis {
				keepIDs, keepOrd = append(keepIDs, cur.id), append(keepOrd, pc.ordinal)
				continue
			}
			supersede = append(supersede, cur.id)
		} else if last, ok := latest[key]; ok && last.status == "kept" && last.basis == pc.basis {
			continue // kept, and its inputs have not changed since
		}
		insert = append(insert, pc)
	}
	// A pending difference that no longer exists (resolved by an edit, an
	// undo or its inputs changing back) is superseded.
	for key, cur := range pending {
		if !seen[key] {
			supersede = append(supersede, cur.id)
		}
	}

	b := &pgx.Batch{}
	if len(supersede) > 0 {
		b.Queue(`UPDATE program_changes SET status='superseded' WHERE org_id=$1::uuid AND program_id=$2::uuid AND id::text=ANY($3::text[]) AND status='pending'`, org, p.ID, supersede)
	}
	if len(keepIDs) > 0 {
		b.Queue(`UPDATE program_changes c SET ordinal=u.ordinal FROM unnest($3::text[],$4::int[]) u(id,ordinal)
 WHERE c.org_id=$1::uuid AND c.program_id=$2::uuid AND c.id::text=u.id`, org, p.ID, keepIDs, keepOrd)
	}
	for _, pc := range insert {
		ops, err := json.Marshal(pc.c.Ops)
		if err != nil {
			return out, err
		}
		b.Queue(`INSERT INTO program_changes(org_id,id,program_id,source,kind,target_activity_id,source_key,proposal,summary,default_choice,basis,ordinal)
 VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,'')::uuid,$7,$8::jsonb,$9,$10,$11,$12)`,
			org, newID(), p.ID, pc.c.Source, pc.c.Kind, pc.c.TargetActivityID, pc.c.SourceKey, ops, clipSummary(pc.c.Summary),
			pc.c.DefaultChoice, pc.basis, pc.ordinal)
	}
	b.Queue(`SELECT count(*) FROM program_changes WHERE org_id=$1::uuid AND program_id=$2::uuid AND status='pending'`, org, p.ID)
	br := tx.SendBatch(ctx, b)
	for i := 0; i < b.Len()-1; i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return out, err
		}
	}
	if err := br.QueryRow().Scan(&out.Pending); err != nil {
		br.Close()
		return out, err
	}
	if err := br.Close(); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out.Changed = len(supersede) > 0 || len(insert) > 0
	return out, nil
}

// reviewChanges turns the diff into one change per source key, numbered in
// diff order, each with its basis. A row whose label and duration both
// differ is one change (kind rename) carrying both ops, because the pending
// index allows one pending change per source key.
func reviewChanges(diff []engine.Change, acts []ProgramActivity, links []ProgramLink) []pendingChange {
	idKey := map[string]string{}
	for _, a := range acts {
		if a.SourceKey != "" {
			idKey[a.ID] = "key:" + string(a.SourceKey)
		}
	}
	for _, l := range links {
		if l.SourceKey != "" {
			idKey[l.ID] = "key:" + string(l.SourceKey)
		}
	}
	var out []pendingChange
	at := map[string]int{}
	for _, c := range diff {
		if i, ok := at[c.SourceKey]; ok {
			m := &out[i].c
			m.Kind = engine.ChangeRename
			m.Summary = strings.Replace(m.Summary, " "+engine.EditedSince, "", 1) + " " + c.Summary
			m.Ops = append(m.Ops, c.Ops...)
			if c.DefaultChoice == engine.ChoiceKeep {
				m.DefaultChoice = engine.ChoiceKeep
			}
			m.Edited = m.Edited || c.Edited
			continue
		}
		at[c.SourceKey] = len(out)
		out = append(out, pendingChange{c: c, ordinal: len(out)})
	}
	for i := range out {
		out[i].basis = changeBasis(out[i].c, idKey)
	}
	return out
}

// changeBasis hashes a change's kind, source key and ops with stored ids
// written as source keys, so the same difference hashes the same however
// the ids it names were made.
func changeBasis(c engine.Change, idKey map[string]string) string {
	norm := make([]map[string]any, len(c.Ops))
	swap := func(v any) any {
		if s, ok := v.(string); ok {
			if k, ok := idKey[s]; ok {
				return k
			}
		}
		return v
	}
	for i, op := range c.Ops {
		m := map[string]any{}
		for k, v := range op {
			switch k {
			case "id", "parent_id", "after_id", "pred_id", "succ_id":
				m[k] = swap(v)
			case "ids":
				var ids []any
				switch list := v.(type) {
				case []string:
					for _, id := range list {
						ids = append(ids, swap(id))
					}
				case []any:
					for _, id := range list {
						ids = append(ids, swap(id))
					}
				}
				m[k] = ids
			default:
				m[k] = v
			}
		}
		norm[i] = m
	}
	body, _ := json.Marshal(struct {
		Kind string           `json:"kind"`
		Key  string           `json:"key"`
		Ops  []map[string]any `json:"ops"`
	}{c.Kind, c.SourceKey, norm})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func clipSummary(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 300 {
		return string(r[:299]) + "…"
	}
	return s
}

// ProgramChange is one pending review item as the review panel shows it.
// DefaultChoice and the summary are computed at read time: a change raised
// for an untouched row whose row the user has edited since defaults to Keep
// and says so (plan 13.1).
type ProgramChange struct {
	ID               string          `json:"id"`
	Kind             string          `json:"kind"`
	Source           string          `json:"source"`
	SourceKey        string          `json:"source_key"`
	TargetActivityID NullText        `json:"target_activity_id"`
	Summary          string          `json:"summary"`
	DefaultChoice    string          `json:"default_choice"`
	Edited           bool            `json:"edited"`
	EditedSince      bool            `json:"edited_since"`
	Ops              json.RawMessage `json:"ops"`
	CreatedAt        string          `json:"created_at"`
	ordinal          int
	storedDefault    string
}

// ProgramChangeList is GET /program/changes: the pending items in diff
// order, their counts by kind and the program version Adopt pins to.
type ProgramChangeList struct {
	ProgramID string          `json:"program_id"`
	Version   int64           `json:"version"`
	Changes   []ProgramChange `json:"changes"`
	Counts    map[string]int  `json:"counts"`
}

// ProgramChanges returns the pending review items of the project's program.
func (s *Store) ProgramChanges(ctx context.Context, org, project string) (ProgramChangeList, error) {
	p, acts, links, _, err := readProgramInputs(ctx, s.pool, org, project, false)
	if err != nil {
		return ProgramChangeList{}, err
	}
	changes, err := readPendingChanges(ctx, s.pool, org, p.ID, false)
	if err != nil {
		return ProgramChangeList{}, err
	}
	list := ProgramChangeList{ProgramID: p.ID, Version: p.Version, Changes: changes,
		Counts: map[string]int{engine.ChangeAdd: 0, engine.ChangeRemove: 0, engine.ChangeRename: 0, engine.ChangeDuration: 0, engine.ChangeLink: 0}}
	ed := newEditedIndex(acts, links)
	for i := range list.Changes {
		ed.apply(&list.Changes[i])
		list.Counts[list.Changes[i].Kind]++
	}
	return list, nil
}

func readPendingChanges(ctx context.Context, q programQuerier, org, programID string, lock bool) ([]ProgramChange, error) {
	sql := `SELECT id::text,kind,source,source_key,COALESCE(target_activity_id::text,''),summary,default_choice,proposal::text,ordinal,
 to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
 FROM program_changes WHERE org_id=$1::uuid AND program_id=$2::uuid AND status='pending' ORDER BY ordinal,created_at,id`
	if lock {
		sql += ` FOR UPDATE`
	}
	rows, err := q.Query(ctx, sql, org, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProgramChange{}
	for rows.Next() {
		var c ProgramChange
		var target, ops string
		if err := rows.Scan(&c.ID, &c.Kind, &c.Source, &c.SourceKey, &target, &c.Summary, &c.storedDefault, &ops, &c.ordinal, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.TargetActivityID, c.Ops, c.DefaultChoice = NullText(target), json.RawMessage(ops), c.storedDefault
		out = append(out, c)
	}
	return out, rows.Err()
}

// editedIndex answers whether a change's target has been created, edited or
// accepted by the user, now.
type editedIndex struct {
	acts  map[string]ProgramActivity
	kids  map[string][]string
	links map[string]ProgramLink
}

func newEditedIndex(acts []ProgramActivity, links []ProgramLink) editedIndex {
	ix := editedIndex{acts: map[string]ProgramActivity{}, kids: map[string][]string{}, links: map[string]ProgramLink{}}
	for _, a := range acts {
		ix.acts[a.ID] = a
		if a.ParentID != "" {
			ix.kids[string(a.ParentID)] = append(ix.kids[string(a.ParentID)], a.ID)
		}
	}
	for _, l := range links {
		ix.links[l.ID] = l
	}
	return ix
}

func rowEdited(a ProgramActivity) bool {
	return a.Origin != engine.OriginLibrary || a.ReviewStatus != engine.ReviewProposed
}

// edited reports the change's target as edited now. A stage's removal takes
// its activities, so an edited activity counts; a link change looks at the
// link it names.
func (ix editedIndex) edited(c ProgramChange) bool {
	if c.TargetActivityID != "" {
		a, ok := ix.acts[string(c.TargetActivityID)]
		if !ok {
			return false
		}
		if rowEdited(a) {
			return true
		}
		if c.Kind == engine.ChangeRemove {
			for _, k := range ix.kids[a.ID] {
				if rowEdited(ix.acts[k]) {
					return true
				}
			}
		}
		return false
	}
	var ops []generatedOp
	if json.Unmarshal(c.Ops, &ops) != nil {
		return false
	}
	for _, o := range ops {
		ids := append([]string{o.ID}, o.IDs...)
		if o.Op != "link_update" && o.Op != "link_delete" {
			continue
		}
		for _, id := range ids {
			if l, ok := ix.links[id]; ok && (l.Origin == engine.OriginUser || l.Origin == engine.OriginDocument) {
				return true
			}
		}
	}
	return false
}

// apply sets the read-time default: Keep for an edited target, with "You've
// edited this since" when the change was raised for an untouched row.
func (ix editedIndex) apply(c *ProgramChange) {
	now := ix.edited(*c)
	c.Edited = c.storedDefault == engine.ChoiceKeep || now
	if now && c.storedDefault == engine.ChoiceAdopt {
		c.DefaultChoice, c.EditedSince = engine.ChoiceKeep, true
		if !strings.Contains(c.Summary, engine.EditedSince) {
			c.Summary += " " + engine.EditedSince
		}
	}
}

// ProgramDecision is one choice in a decide request.
type ProgramDecision struct {
	ID     string `json:"id"`
	Choice string `json:"choice"`
}

// DecideProgramChanges records the user's choices on pending review items.
// Adopted changes apply as one batch through the program write path, so the
// batch is version-pinned, logged for undo and one undo reverses it. Kept
// changes are marked kept and are not raised again until their inputs
// change. adoptAll adopts every pending change whose default (computed now)
// is Adopt; changes that default to Keep stay pending for an explicit
// choice, so a user value is never overwritten by "Adopt all".
func (s *Store) DecideProgramChanges(ctx context.Context, org, project, actor string, version int64, decisions []ProgramDecision, adoptAll bool) (ProgramResult, error) {
	if !adoptAll && len(decisions) == 0 {
		return ProgramResult{}, opErr(-1, "decisions", "Choose at least one change.")
	}
	if len(decisions) > MaxProgramDecisions {
		return ProgramResult{}, opErr(-1, "decisions", "Too many changes in one decision.")
	}
	choice := map[string]string{}
	for i, d := range decisions {
		if !uuidPattern.MatchString(d.ID) {
			return ProgramResult{}, opErr(i, "id", MsgChangeGone)
		}
		if d.Choice != engine.ChoiceAdopt && d.Choice != engine.ChoiceKeep {
			return ProgramResult{}, opErr(i, "choice", "Choose adopt or keep.")
		}
		if _, dup := choice[d.ID]; dup {
			return ProgramResult{}, opErr(i, "id", "Each change is decided once.")
		}
		choice[d.ID] = d.Choice
	}

	var adopted, kept, applied []string
	var appliedOps []engine.Op
	mark := func(ctx context.Context, tx pgx.Tx, programID string) error {
		if len(adopted) > 0 {
			if _, err := tx.Exec(ctx, `UPDATE program_changes SET status='adopted',decided_by=$3::uuid,decided_at=now()
 WHERE org_id=$1::uuid AND program_id=$2::uuid AND id::text=ANY($4::text[]) AND status='pending'`, org, programID, actor, adopted); err != nil {
				return err
			}
		}
		if len(kept) > 0 {
			if _, err := tx.Exec(ctx, `UPDATE program_changes SET status='kept',decided_by=$3::uuid,decided_at=now()
 WHERE org_id=$1::uuid AND program_id=$2::uuid AND id::text=ANY($4::text[]) AND status='pending'`, org, programID, actor, kept); err != nil {
				return err
			}
		}
		return nil
	}
	// pick sorts the pending items into adopted and kept, in diff order.
	pick := func(ctx context.Context, tx pgx.Tx, w *programWork) ([]ProgramChange, error) {
		pending, err := readPendingChanges(ctx, tx, org, w.prog.ID, true)
		if err != nil {
			return nil, err
		}
		state := w.state()
		ix := newEditedIndex(state.Activities, state.Links)
		byID := map[string]bool{}
		var adopt []ProgramChange
		adopted, kept = nil, nil
		for i := range pending {
			c := &pending[i]
			ix.apply(c)
			byID[c.ID] = true
			ch, ok := choice[c.ID]
			if !ok && adoptAll && c.DefaultChoice == engine.ChoiceAdopt {
				ch, ok = engine.ChoiceAdopt, true
			}
			switch {
			case !ok:
			case ch == engine.ChoiceAdopt:
				adopted = append(adopted, c.ID)
				adopt = append(adopt, *c)
			default:
				kept = append(kept, c.ID)
			}
		}
		for i, d := range decisions {
			if !byID[d.ID] {
				return nil, opErr(i, "id", MsgChangeGone)
			}
		}
		return adopt, nil
	}

	res, err := s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		adopt, err := pick(ctx, tx, w)
		if err != nil {
			return err
		}
		keyIdx := map[string]string{}
		for _, id := range w.order() {
			if k := w.acts[id].SourceKey; k != "" {
				keyIdx[string(k)] = id
			}
		}
		w.genReason = "Removed in the program review."
		for _, c := range adopt {
			var ops []engine.Op
			if err := json.Unmarshal(c.Ops, &ops); err != nil {
				return err
			}
			did := false
			for _, op := range ops {
				op = adoptRefs(op, w, keyIdx)
				if op == nil {
					continue
				}
				ok, err := w.applyGenerated(len(appliedOps), op)
				if err != nil {
					return err
				}
				if ok {
					did = true
					appliedOps = append(appliedOps, op)
				}
			}
			if did {
				applied = append(applied, c.ID)
			}
		}
		if len(appliedOps) == 0 {
			return errNoProgramChange
		}
		if err := mark(ctx, tx, w.prog.ID); err != nil {
			return err
		}
		w.fb.Toasts = append(w.fb.Toasts, decideToast(len(applied), len(kept)))
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		ops := []any{map[string]any{"op": "review_adopt", "changes": applied, "applied": appliedOps}}
		return logProgramBatch(ctx, tx, w, actor, ops, c)
	})
	if !errors.Is(err, errNoProgramChange) {
		return res, err
	}
	// Nothing to apply to the program (only Keep, or adopted changes that
	// no longer change anything): record the choices without a program
	// batch, under the same lock and version pin.
	applied = nil
	return s.markProgramChanges(ctx, org, project, version, pick, mark)
}

// markProgramChanges records decisions that change no program row. The
// program version does not move, so no undo step is made.
func (s *Store) markProgramChanges(ctx context.Context, org, project string, version int64,
	pick func(context.Context, pgx.Tx, *programWork) ([]ProgramChange, error),
	mark func(context.Context, pgx.Tx, string) error) (ProgramResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramResult{}, err
	}
	defer tx.Rollback(ctx)
	p, acts, links, days, err := readProgramInputs(ctx, tx, org, project, true)
	if err != nil {
		return ProgramResult{}, err
	}
	st := computeState(p, acts, links, days)
	if p.Version != version {
		return ProgramResult{ProgramState: st, ProgramFeedback: emptyFeedback()}, ErrVersionConflict
	}
	w := newProgramWork(org, project, p, acts, links, days)
	adopt, err := pick(ctx, tx, w)
	if err != nil {
		return ProgramResult{}, err
	}
	if err := mark(ctx, tx, p.ID); err != nil {
		return ProgramResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramResult{}, err
	}
	fb := emptyFeedback()
	fb.Toasts = append(fb.Toasts, decideToast(0, 0))
	if len(adopt) > 0 {
		fb.Toasts[0] = "Those changes are already in the program."
	}
	return ProgramResult{ProgramState: st, ProgramFeedback: fb}, nil
}

func decideToast(adopted, _ int) string {
	switch {
	case adopted == 0:
		return "Kept the program as it is."
	case adopted == 1:
		return "1 change adopted; undo reverses it."
	}
	return fmt.Sprintf("%d changes adopted; undo reverses them.", adopted)
}

// adoptRefs points a stored op at today's rows: a temp id that is a source
// key now stored becomes that row's id. An add whose source key is already
// live returns nil: the row exists and adding it again would duplicate it.
func adoptRefs(op engine.Op, w *programWork, keyIdx map[string]string) engine.Op {
	ref := func(v any) any {
		s, ok := v.(string)
		if !ok || s == "" {
			return v
		}
		if _, ok := w.temp[s]; ok {
			return v
		}
		if _, ok := w.acts[s]; ok {
			return v
		}
		if id, ok := keyIdx[s]; ok && w.live(id) != nil {
			return id
		}
		return v
	}
	out := engine.Op{}
	for k, v := range op {
		switch k {
		case "id", "parent_id", "after_id", "pred_id", "succ_id":
			out[k] = ref(v)
		default:
			out[k] = v
		}
	}
	if out["op"] == "add" {
		if key, _ := out["source_key"].(string); key != "" {
			if id, ok := keyIdx[key]; ok && w.live(id) != nil {
				return nil
			}
		}
	}
	return out
}
