package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UndoProgram applies the inverse of the caller's most recent batch that is
// not undone (plan 8.1 "Undo"). Each row the batch touched must still be
// exactly as the batch left it; otherwise nothing changes and the refusal
// says "Can't undo: changed since". Undo restores retired rows and their
// links. History is per user.
func (s *Store) UndoProgram(ctx context.Context, org, project, actor string, version int64) (ProgramResult, error) {
	return s.flipProgram(ctx, org, project, actor, version, true)
}

// RedoProgram re-applies the caller's most recently undone batch, unless
// they have made a batch since, and only while its rows are as the undo
// left them.
func (s *Store) RedoProgram(ctx context.Context, org, project, actor string, version int64) (ProgramResult, error) {
	return s.flipProgram(ctx, org, project, actor, version, false)
}

func (s *Store) flipProgram(ctx context.Context, org, project, actor string, version int64, undo bool) (ProgramResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramResult{}, err
	}
	defer tx.Rollback(ctx)
	p, acts, links, days, err := readProgramInputs(ctx, tx, org, project, true)
	if err != nil {
		return ProgramResult{}, err
	}
	if p.Version != version {
		return ProgramResult{ProgramState: computeState(p, acts, links, days), ProgramFeedback: emptyFeedback()}, ErrVersionConflict
	}
	query := `SELECT seq,inverse FROM program_ops WHERE org_id=$1::uuid AND program_id=$2::uuid AND actor=$3::uuid AND undone_at IS NULL ORDER BY seq DESC LIMIT 1`
	empty, changed := MsgNothingToUndo, MsgUndoChanged
	if !undo {
		// The most recently undone batch, unless the user made a batch since:
		// a new edit ends the redo run.
		query = `SELECT o.seq,o.inverse FROM program_ops o WHERE o.org_id=$1::uuid AND o.program_id=$2::uuid AND o.actor=$3::uuid AND o.undone_at IS NOT NULL
 AND NOT EXISTS (SELECT 1 FROM program_ops n WHERE n.org_id=o.org_id AND n.program_id=o.program_id AND n.actor=o.actor AND n.at>o.undone_at)
 ORDER BY o.undone_at DESC LIMIT 1`
		empty, changed = MsgNothingToRedo, MsgRedoChanged
	}
	var seq int64
	var raw []byte
	err = tx.QueryRow(ctx, query, org, p.ID, actor).Scan(&seq, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramResult{}, opErr(-1, "", empty)
	}
	if err != nil {
		return ProgramResult{}, err
	}
	var entries []programInverse
	if err := json.Unmarshal(raw, &entries); err != nil {
		return ProgramResult{}, err
	}
	w := newProgramWork(org, project, p, acts, links, days)
	if err := w.loadTouched(ctx, tx, entries); err != nil {
		return ProgramResult{}, err
	}
	c, err := w.restore(entries, undo)
	if err != nil {
		return ProgramResult{}, opErr(-1, "", changed)
	}
	if err := w.checkStructure(); err != nil {
		return ProgramResult{}, opErr(-1, "", changed)
	}
	if !w.uniqueLive() || w.output().Cycle != nil {
		return ProgramResult{}, opErr(-1, "", changed)
	}
	if err := c.write(ctx, tx, org, p.ID); err != nil {
		return ProgramResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE program_ops SET undone_at=CASE WHEN $4 THEN clock_timestamp() END WHERE org_id=$1::uuid AND program_id=$2::uuid AND seq=$3`, org, p.ID, seq, undo); err != nil {
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
	return ProgramResult{ProgramState: w.state(), ProgramFeedback: emptyFeedback()}, nil
}

// loadTouched reads the rows a batch touched that are retired now.
func (w *programWork) loadTouched(ctx context.Context, tx pgx.Tx, entries []programInverse) error {
	var actIDs, linkIDs []string
	for _, e := range entries {
		switch {
		case e.Target == "activity" && w.acts[e.ID] == nil:
			actIDs = append(actIDs, e.ID)
		case e.Target == "link" && w.links[e.ID] == nil:
			linkIDs = append(linkIDs, e.ID)
		}
	}
	var acts []ProgramActivity
	var links []ProgramLink
	if len(actIDs) > 0 {
		rows, err := tx.Query(ctx, `SELECT `+activityColumns+` FROM program_activities a WHERE a.org_id=$1::uuid AND a.program_id=$2::uuid AND a.id::text=ANY($3::text[])`, w.org, w.prog.ID, actIDs)
		if err != nil {
			return err
		}
		if acts, err = scanActivities(rows); err != nil {
			return err
		}
	}
	if len(linkIDs) > 0 {
		rows, err := tx.Query(ctx, `SELECT `+linkColumns+` FROM program_links l WHERE l.org_id=$1::uuid AND l.program_id=$2::uuid AND l.id::text=ANY($3::text[])`, w.org, w.prog.ID, linkIDs)
		if err != nil {
			return err
		}
		if links, err = scanLinks(rows); err != nil {
			return err
		}
	}
	w.addRetired(acts, links)
	return nil
}

// restore checks every touched row against the image the flip expects and
// writes the other image exactly, version included.
func (w *programWork) restore(entries []programInverse, undo bool) (programChanges, error) {
	var c programChanges
	changed := errors.New("changed since")
	pick := func(e programInverse) (from, to json.RawMessage) {
		if undo {
			return e.After, e.Before
		}
		return e.Before, e.After
	}
	canon := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	for _, e := range entries {
		from, to := pick(e)
		switch e.Target {
		case "activity":
			var want, next ProgramActivity
			if json.Unmarshal(from, &want) != nil || json.Unmarshal(to, &next) != nil {
				return c, fmt.Errorf("bad undo image for %s", e.ID)
			}
			cur := w.acts[e.ID]
			if cur == nil || canon(*cur) != canon(want) {
				return c, changed
			}
			if next.Systems == nil {
				next.Systems = []string{}
			}
			*cur = next
			c.updActs = append(c.updActs, next)
		case "link":
			var want, next ProgramLink
			if json.Unmarshal(from, &want) != nil || json.Unmarshal(to, &next) != nil {
				return c, fmt.Errorf("bad undo image for %s", e.ID)
			}
			cur := w.links[e.ID]
			if cur == nil || *cur != want {
				return c, changed
			}
			*cur = next
			c.updLinks = append(c.updLinks, next)
		case "day":
			cur, ok := w.days[e.ID]
			curImg := "null"
			if ok {
				curImg = canon(cur)
			}
			var want *ProgramCalendarDay
			if json.Unmarshal(from, &want) != nil || (want == nil && curImg != "null") || (want != nil && curImg != canon(*want)) {
				return c, changed
			}
			var next *ProgramCalendarDay
			if json.Unmarshal(to, &next) != nil {
				return c, changed
			}
			if next == nil {
				delete(w.days, e.ID)
				c.delDays = append(c.delDays, e.ID)
			} else {
				w.days[e.ID] = *next
				c.upsDays = append(c.upsDays, *next)
			}
		case "program":
			var want, next startImage
			if json.Unmarshal(from, &want) != nil || json.Unmarshal(to, &next) != nil || programStart(w.prog) != want {
				return c, changed
			}
			w.prog.StartDate, w.prog.StartOrigin = next.StartDate, next.StartOrigin
		}
		if e.Op == "stage_revision" {
			w.stageChange = true
		}
	}
	w.rebuild()
	return c, nil
}

// rebuild derives the sequences and the live pairs from the rows after an
// undo wrote images straight into them.
func (w *programWork) rebuild() {
	live := make([]ProgramActivity, 0, len(w.acts))
	for _, a := range w.acts {
		if !a.Retired {
			live = append(live, *a)
		}
	}
	w.stages, w.kids = nil, map[string][]string{}
	for _, a := range outline(live) {
		if a.ParentID == "" {
			w.stages = append(w.stages, a.ID)
		} else {
			w.kids[string(a.ParentID)] = append(w.kids[string(a.ParentID)], a.ID)
		}
	}
	w.pairs = map[[2]string]string{}
	for _, id := range w.linkOrder {
		if l := w.links[id]; !l.Retired {
			w.pairs[[2]string{l.PredID, l.SuccID}] = id
		}
	}
	w.dirty = true
}

// uniqueLive reports whether the live rows still respect the live-only
// unique indexes: one live link per pair, one live row per source key.
func (w *programWork) uniqueLive() bool {
	if len(w.pairs) != len(w.liveLinks()) {
		return false
	}
	keys := map[NullText]bool{}
	for _, id := range w.order() {
		k := w.acts[id].SourceKey
		if k == "" {
			continue
		}
		if keys[k] {
			return false
		}
		keys[k] = true
	}
	return true
}

// AcceptProgram moves proposed rows to accepted: the ids given, or every
// proposed row with all. It is one undoable batch.
func (s *Store) AcceptProgram(ctx context.Context, org, project, actor string, version int64, ids []string, all bool) (ProgramResult, error) {
	if !all && len(ids) == 0 {
		return ProgramResult{}, opErr(-1, "ids", "Choose at least one row.")
	}
	if len(ids) > MaxProgramRows {
		return ProgramResult{}, opErr(-1, "ids", MsgTooManyRows)
	}
	op := map[string]any{"op": "accept", "ids": ids, "all": all}
	return s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		accept := func(a *ProgramActivity) {
			if a.ReviewStatus == "proposed" {
				w.touchAct(a.ID).ReviewStatus = "accepted"
			}
		}
		if all {
			for _, id := range w.order() {
				accept(w.acts[id])
			}
			return acceptProgramReadings(ctx, tx, w, nil, true)
		}
		var rest []string
		for _, id := range ids {
			if a := w.live(id); a != nil {
				accept(a)
			} else {
				rest = append(rest, id)
			}
		}
		return acceptProgramReadings(ctx, tx, w, rest, false)
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		return logProgramBatch(ctx, tx, w, actor, []any{op}, c)
	})
}

// acceptProgramReadings is the seam for document readings (WP-P12): ids
// that are not program rows will name program_readings rows. Until readings
// exist, any such id is not in this program.
func acceptProgramReadings(ctx context.Context, tx pgx.Tx, w *programWork, ids []string, all bool) error {
	if len(ids) > 0 {
		return opErr(-1, "ids", MsgNotInProgram)
	}
	return nil
}
