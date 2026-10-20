package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
	"sitewise/internal/programread"
)

// Accepting, dismissing and adding from program file readings (plan 10.2
// "User sees"). Each is one program batch: one version bump, one undo step.
// A reading's status change is part of the batch's inverse, so undo puts the
// reading back to proposed along with the rows it changed. No path here
// calls Jev.

// readingWork tracks reading statuses a batch changes.
type readingWork struct {
	status map[string]string // current status by reading id
	before map[string]string // first status of a touched reading
	order  []string
}

func (w *programWork) readingState() *readingWork {
	if w.readings == nil {
		w.readings = &readingWork{status: map[string]string{}, before: map[string]string{}}
	}
	return w.readings
}

// setReading changes one loaded reading's status.
func (w *programWork) setReading(id, status string) {
	r := w.readingState()
	if _, seen := r.before[id]; !seen {
		r.before[id] = r.status[id]
		r.order = append(r.order, id)
	}
	r.status[id] = status
}

type readingStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// readingDiff adds changed reading statuses to the batch and its inverse.
func (w *programWork) readingDiff(c *programChanges) {
	if w.readings == nil {
		return
	}
	for _, id := range w.readings.order {
		before, after := w.readings.before[id], w.readings.status[id]
		if before == after {
			continue
		}
		c.readings = append(c.readings, readingStatus{id, after})
		b, _ := json.Marshal(readingStatus{id, before})
		a, _ := json.Marshal(readingStatus{id, after})
		c.inverse = append(c.inverse, programInverse{Op: "restore", Target: "reading", ID: id, Before: b, After: a})
	}
}

// writeReadings queues the status updates on the batch's statement batch.
func (c programChanges) writeReadings(b *pgx.Batch, org, programID string) (int, error) {
	if len(c.readings) == 0 {
		return 0, nil
	}
	raw, err := json.Marshal(c.readings)
	if err != nil {
		return 0, err
	}
	b.Queue(`UPDATE program_readings r SET status=x.status FROM jsonb_to_recordset($3::jsonb) AS x(id uuid,status text)
WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.id=x.id`, org, programID, raw)
	return len(c.readings), nil
}

// loadReadingStatus reads the current status of the readings an undo or
// redo touches.
func (w *programWork) loadReadingStatus(ctx context.Context, tx pgx.Tx, entries []programInverse) error {
	var ids []string
	for _, e := range entries {
		if e.Target == "reading" {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT id::text,status FROM program_readings WHERE org_id=$1::uuid AND program_id=$2::uuid AND id::text=ANY($3::text[])`, w.org, w.prog.ID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	r := w.readingState()
	for rows.Next() {
		var id, st string
		if err := rows.Scan(&id, &st); err != nil {
			return err
		}
		r.status[id] = st
	}
	return rows.Err()
}

// restoreReading flips one reading's status for undo or redo, only while it
// is as the batch (or the undo) left it.
func (w *programWork) restoreReading(e programInverse, from, to json.RawMessage, c *programChanges) bool {
	var want, next readingStatus
	if json.Unmarshal(from, &want) != nil || json.Unmarshal(to, &next) != nil {
		return false
	}
	r := w.readingState()
	if cur, ok := r.status[e.ID]; !ok || cur != want.Status {
		return false
	}
	r.status[e.ID] = next.Status
	c.readings = append(c.readings, readingStatus{e.ID, next.Status})
	return true
}

// programReading is one stored reading as accept needs it.
type programReading struct {
	ID, Document, Target, Catalogue, Field, Value string
	Parsed                                        struct {
		Row  string `json:"row"`
		Date string `json:"date"`
		Low  *int   `json:"low"`
		High *int   `json:"high"`
	}
}

func loadReadings(ctx context.Context, tx pgx.Tx, w *programWork, status string, ids []string, all bool) ([]programReading, error) {
	// A program file's readings, and an awarded proposal's R7 readings
	// (plan 10.3): a proposal's stay out until its submission is awarded.
	sql := `SELECT id::text,COALESCE(document_id::text,''),COALESCE(target_activity_id::text,''),COALESCE(catalogue_stage_id,''),field,COALESCE(value,''),parsed::text,status
FROM program_readings WHERE org_id=$1::uuid AND program_id=$2::uuid AND source IN ('program_file','proposal_r7') AND ` + readingVisible("program_readings") + ` AND status=$3`
	args := []any{w.org, w.prog.ID, status}
	if !all {
		sql += ` AND id::text=ANY($4::text[])`
		args = append(args, ids)
	}
	rows, err := tx.Query(ctx, sql+` ORDER BY created_at,id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []programReading
	r := w.readingState()
	for rows.Next() {
		var p programReading
		var parsed, st string
		if err := rows.Scan(&p.ID, &p.Document, &p.Target, &p.Catalogue, &p.Field, &p.Value, &parsed, &st); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(parsed), &p.Parsed); err != nil {
			return nil, err
		}
		if _, seen := r.status[p.ID]; !seen {
			r.status[p.ID] = st
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// readingGroups expands the named readings to every proposed reading of
// the same target from the same file: accepting a target's start accepts
// its finish and duration with it. Every id must be a proposed reading.
func readingGroups(ctx context.Context, tx pgx.Tx, w *programWork, ids []string, all bool) ([][]programReading, error) {
	everything, err := loadReadings(ctx, tx, w, "proposed", nil, true)
	if err != nil {
		return nil, err
	}
	byID := map[string]programReading{}
	key := func(p programReading) string { return p.Document + "|" + p.Target + "|" + p.Catalogue }
	groups := map[string][]programReading{}
	var order []string
	for _, p := range everything {
		byID[p.ID] = p
		k := key(p)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], p)
	}
	want := map[string]bool{}
	if all {
		for _, k := range order {
			want[k] = true
		}
	}
	for _, id := range ids {
		p, ok := byID[id]
		if !ok {
			return nil, opErr(-1, "ids", MsgNotInProgram)
		}
		want[key(p)] = true
	}
	var out [][]programReading
	for _, k := range order {
		if want[k] {
			out = append(out, groups[k])
		}
	}
	return out, nil
}

// readingTarget is the live row a reading group applies to (plan 7.7).
func (w *programWork) readingTarget(g []programReading) *ProgramActivity {
	if a := w.live(g[0].Target); a != nil {
		return a
	}
	// A user-made stage (`other`) receives evidence only when the user
	// assigns it (plan 7.7): evidence for a deleted one has no stage and
	// never moves to another user-made stage.
	if c := g[0].Catalogue; c != "" && c != "other" {
		for _, id := range w.stages {
			if string(w.acts[id].CatalogueStageID) == c {
				return w.acts[id]
			}
		}
	}
	return nil
}

// fileDays is the working days from start to finish inclusive in the
// program's calendar, counted from the file's own start date so it means
// the same on a program with no start date.
func (w *programWork) fileDays(start, finish string) (int, bool) {
	if start == "" || finish == "" || finish < start {
		return 0, false
	}
	in := w.input().Calendar
	in.Start = start
	cal := engine.NewCalendar(in)
	last, ok := cal.IndexOnOrBefore(finish)
	if !ok || last < 0 {
		return 0, false
	}
	days := last + 1
	if days > engine.MaxDuration {
		return 0, false
	}
	return days, true
}

// acceptProgramReadings applies the named readings (or all of them): the
// target's duration in working days from the file's start and finish, or
// the file's duration when no dates parse, and a start-no-earlier-than at
// the file's start. A user or accepted document value is never overwritten:
// such a reading stays proposed (plan section 2, Jev rule 14). Predecessor
// links between rows that matched are added from the file's predecessor
// column. ids that are program rows were handled by the caller.
func acceptProgramReadings(ctx context.Context, tx pgx.Tx, w *programWork, ids []string, all bool) error {
	if !all && len(ids) == 0 {
		return nil
	}
	groups, err := readingGroups(ctx, tx, w, ids, all)
	if err != nil {
		return err
	}
	docs := map[string]bool{}
	for _, g := range groups {
		a := w.readingTarget(g)
		if a == nil {
			w.fb.Toasts = append(w.fb.Toasts, "A reading's activity is no longer in the program; it stays proposed.")
			continue
		}
		var start, finish string
		var dur *engine.Duration
		var readingIDs []string
		for _, r := range g {
			readingIDs = append(readingIDs, r.ID)
			switch r.Field {
			case programread.FieldStart:
				start = r.Parsed.Date
			case programread.FieldFinish:
				finish = r.Parsed.Date
			case programread.FieldDuration:
				if r.Parsed.Low != nil && r.Parsed.High != nil {
					dur = &engine.Duration{Low: *r.Parsed.Low, High: *r.Parsed.High}
				}
			}
		}
		days, haveDays := w.fileDays(start, finish)
		if !haveDays && dur != nil && dur.Low == dur.High {
			days, haveDays = dur.Low, true
		}
		if a.Kind == engine.KindMilestone || a.Kind == engine.KindHoldPoint || a.Kind == engine.KindStage && len(w.kids[a.ID]) > 0 {
			haveDays = false // a milestone takes 0 days; a stage with activities takes its span from them
		}
		snet := start != "" && w.prog.StartDate != ""
		if haveDays && (a.DurationOrigin == "user" || a.DurationOrigin == "document") || snet && a.ConstraintKind != engine.ConstraintASAP {
			w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("Kept your values for %s; the reading stays proposed.", a.Label))
			continue
		}
		if haveDays || snet {
			a = w.touchAct(a.ID)
		}
		if haveDays {
			src, _ := json.Marshal(map[string]any{"document_id": g[0].Document, "readings": readingIDs})
			a.DurationDays, a.DurationLow, a.DurationHigh = intp(days), nil, nil
			a.DurationOrigin, a.DurationSource = "document", src
		}
		if snet {
			a.ConstraintKind, a.ConstraintDate = engine.ConstraintSNET, NullText(start)
		} else if start != "" && w.prog.StartDate == "" {
			w.fb.Toasts = append(w.fb.Toasts, "The program has no start date, so the file's start dates were not applied.")
		}
		for _, id := range readingIDs {
			w.setReading(id, "accepted")
		}
		docs[g[0].Document] = true
	}
	for doc := range docs {
		if err := w.linkFromFile(ctx, tx, doc); err != nil {
			return err
		}
	}
	return nil
}

// linkFromFile adds the file's predecessor links between rows that matched
// accepted readings or became activities, unless a link already joins them,
// one end is the other's stage, or the link would close a loop.
func (w *programWork) linkFromFile(ctx context.Context, tx pgx.Tx, doc string) error {
	var importID, raw string
	err := tx.QueryRow(ctx, `SELECT id::text,file_rows::text FROM program_imports WHERE org_id=$1::uuid AND program_id=$2::uuid AND document_id=$3::uuid`, w.org, w.prog.ID, doc).Scan(&importID, &raw)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var rows []programread.Row
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return err
	}
	accepted, err := loadReadings(ctx, tx, w, "accepted", nil, true)
	if err != nil {
		return err
	}
	rowTarget := map[string]string{}
	for _, r := range accepted {
		if r.Document == doc && r.Parsed.Row != "" {
			if a := w.readingTarget([]programReading{r}); a != nil {
				rowTarget[r.Parsed.Row] = a.ID
			}
		}
	}
	// Readings accepted in this batch are still "proposed" in the table.
	proposed, err := loadReadings(ctx, tx, w, "proposed", nil, true)
	if err != nil {
		return err
	}
	for _, r := range proposed {
		if r.Document == doc && r.Parsed.Row != "" && w.readingState().status[r.ID] == "accepted" {
			if a := w.readingTarget([]programReading{r}); a != nil {
				rowTarget[r.Parsed.Row] = a.ID
			}
		}
	}
	for _, id := range w.order() {
		if k := string(w.acts[id].SourceKey); strings.HasPrefix(k, "import:"+importID+":") {
			rowTarget[strings.TrimPrefix(k, "import:"+importID+":")] = id
		}
	}
	for _, l := range importLinks(rows, rowTarget) {
		pred, succ := w.live(l.PredID), w.live(l.SuccID)
		if pred == nil || succ == nil || w.linked(pred.ID, succ.ID) || string(pred.ParentID) == succ.ID || string(succ.ParentID) == pred.ID {
			continue
		}
		link := engine.Link{Pred: pred.ID, Succ: succ.ID, Type: l.Type, Lag: l.LagDays, LagUnit: l.LagUnit}
		if engine.ValidateLink(w.input(), link) != nil {
			continue
		}
		w.fb.Linked = append(w.fb.Linked, *w.addLink(pred.ID, succ.ID, l.Type, l.LagDays, l.LagUnit, "document"))
	}
	return nil
}

// DismissProgramReadings dismisses the named readings with their target's
// other readings from the same file. One undoable batch.
func (s *Store) DismissProgramReadings(ctx context.Context, org, project, actor string, version int64, ids []string) (ProgramResult, error) {
	if len(ids) == 0 {
		return ProgramResult{}, opErr(-1, "ids", "Choose at least one reading.")
	}
	op := map[string]any{"op": "dismiss_readings", "ids": ids}
	return s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		groups, err := readingGroups(ctx, tx, w, ids, false)
		if err != nil {
			return err
		}
		for _, g := range groups {
			for _, r := range g {
				w.setReading(r.ID, "dismissed")
			}
		}
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		return logProgramBatch(ctx, tx, w, actor, []any{op}, c)
	})
}

// ProgramRowAdd adds one file row no target matched as a new activity
// under ParentID, or as a stage when ParentID is empty.
type ProgramRowAdd struct {
	ImportID string `json:"import_id"`
	Row      string `json:"row"`
	ParentID string `json:"parent_id"`
}

// AddProgramImportRows adds file rows as activities in one click (code, no
// Jev): the label copied verbatim, the duration from the row's dates (or
// its duration), a start-no-earlier-than at its start when the program has
// a start date, origin document. One undoable batch; the file's
// predecessor links to matched rows are added with them.
func (s *Store) AddProgramImportRows(ctx context.Context, org, project, actor string, version int64, adds []ProgramRowAdd) (ProgramResult, error) {
	if len(adds) == 0 {
		return ProgramResult{}, opErr(-1, "add_rows", "Choose at least one row.")
	}
	if len(adds) > MaxProgramOps {
		return ProgramResult{}, opErr(-1, "add_rows", MsgTooManyOps)
	}
	op := map[string]any{"op": "add_import_rows", "rows": adds}
	return s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		type imported struct {
			doc  string
			rows map[string]programread.Row
		}
		files := map[string]*imported{}
		for i, add := range adds {
			f := files[add.ImportID]
			if f == nil {
				var doc, raw string
				err := tx.QueryRow(ctx, `SELECT document_id::text,file_rows::text FROM program_imports WHERE org_id=$1::uuid AND program_id=$2::uuid AND id::text=$3`, w.org, w.prog.ID, add.ImportID).Scan(&doc, &raw)
				if err == pgx.ErrNoRows {
					return opErr(i, "import_id", "That file is not imported into this program.")
				}
				if err != nil {
					return err
				}
				var rows []programread.Row
				if err := json.Unmarshal([]byte(raw), &rows); err != nil {
					return err
				}
				f = &imported{doc: doc, rows: map[string]programread.Row{}}
				for _, r := range rows {
					f.rows[r.ID] = r
				}
				files[add.ImportID] = f
			}
			r, ok := f.rows[add.Row]
			if !ok {
				return opErr(i, "row", "That row is not in the file.")
			}
			key := importRowKey(add.ImportID, r.ID)
			for _, id := range w.order() {
				if string(w.acts[id].SourceKey) == key {
					return opErr(i, "row", r.Label+" is already in the program.")
				}
			}
			label := strings.TrimSpace(r.Label)
			if len([]rune(label)) > 200 {
				label = string([]rune(label)[:200])
			}
			temp := fmt.Sprintf("import-row-%d", i)
			if err := w.opAdd(i, ProgramOp{Op: "add", TempID: temp, ParentID: add.ParentID, Label: &label}); err != nil {
				return err
			}
			a := w.touchAct(w.temp[temp])
			a.Origin, a.SourceKey = "document", NullText(key)
			days, have := w.fileDays(r.Start, r.Finish)
			if !have && r.Duration != nil && r.Duration.Low == r.Duration.High {
				days, have = r.Duration.Low, true
			}
			if have && (a.Kind != engine.KindStage || len(w.kids[a.ID]) == 0) {
				src, _ := json.Marshal(map[string]any{"document_id": f.doc, "row": r.ID})
				a.DurationDays, a.DurationOrigin, a.DurationSource = intp(days), "document", src
			}
			if r.Start != "" && w.prog.StartDate != "" {
				a.ConstraintKind, a.ConstraintDate = engine.ConstraintSNET, NullText(r.Start)
			}
		}
		for _, f := range files {
			if err := w.linkFromFile(ctx, tx, f.doc); err != nil {
				return err
			}
		}
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		return logProgramBatch(ctx, tx, w, actor, []any{op}, c)
	})
}
