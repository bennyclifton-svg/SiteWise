package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
	"sitewise/internal/programread"
)

// Hold-point release evidence (plan docs/plans/2026-10-18-program.md 10.4).
// The background passage reading writes proposals here; only a user's
// release request marks a hold point released, as one undoable program
// batch. The release lives on the hold point row as
// duration_source.released, beside the generated {"hold_point": id}, so the
// existing activity diff makes it undoable without a new column. No path
// here calls Jev, and Jev never gives a date: the date shown is the
// document's filed date, read by code and labelled "document dated".

// HoldPointTarget is one live, unreleased program hold point whose release
// a document's passages may evidence.
type HoldPointTarget struct {
	ProgramID, ActivityID, InterfaceID string
}

// DocumentHoldPoints lists the live hold points, with an interface, in the
// program of the document's project. Empty when the project has no
// program or the program holds none: the passage reading then asks
// exactly what it asked before.
func (s *Store) DocumentHoldPoints(ctx context.Context, org, documentID string) ([]HoldPointTarget, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.program_id::text,a.id::text,a.interface_id
FROM documents d
JOIN programs p ON p.org_id=d.org_id AND p.project_id=d.project_id
JOIN program_activities a ON a.org_id=p.org_id AND a.program_id=p.id
WHERE d.org_id=$1::uuid AND d.id=$2::uuid AND a.kind='hold_point' AND a.retired_at IS NULL
 AND a.dismissed_reason IS NULL AND COALESCE(a.interface_id,'')<>'' AND NOT (a.duration_source ? 'released')
ORDER BY a.interface_id,a.id`, org, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HoldPointTarget
	for rows.Next() {
		var t HoldPointTarget
		if err := rows.Scan(&t.ProgramID, &t.ActivityID, &t.InterfaceID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// HoldPointReading is one banded release answer for one passage. Probability
// is the noul's probability of true; a noul has no confidence
// (https://docs.typesafe.ai/confidence).
type HoldPointReading struct {
	ProgramID, ActivityID, InterfaceID, PassageID string
	Band                                          string
	Probability                                   float64
}

// ReplaceHoldPointReadings replaces the document's proposed release
// readings with recs, all status proposed. A reading the user already
// accepted or dismissed for the same hold point and passage is not
// proposed again. Nothing here changes a program row.
func (s *Store) ReplaceHoldPointReadings(ctx context.Context, org, documentID string, recs []HoldPointReading) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM program_readings WHERE org_id=$1::uuid AND document_id=$2::uuid AND source='hold_point_evidence' AND status='proposed'`, org, documentID); err != nil {
		return err
	}
	if len(recs) > 0 {
		type row struct {
			ID      string          `json:"id"`
			Program string          `json:"program_id"`
			Target  string          `json:"target_activity_id"`
			Passage string          `json:"passage_id"`
			Band    string          `json:"band"`
			Parsed  json.RawMessage `json:"parsed"`
		}
		rows := make([]row, 0, len(recs))
		for _, r := range recs {
			parsed, err := json.Marshal(map[string]any{"probability": r.Probability, "interface_id": r.InterfaceID})
			if err != nil {
				return err
			}
			rows = append(rows, row{ID: newID(), Program: r.ProgramID, Target: r.ActivityID, Passage: r.PassageID, Band: r.Band, Parsed: parsed})
		}
		raw, err := json.Marshal(rows)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO program_readings(org_id,id,program_id,source,document_id,passage_id,target_activity_id,field,parsed,band,decided_by,question_version,status)
SELECT $1::uuid,r.id,r.program_id,'hold_point_evidence',$2::uuid,r.passage_id,r.target_activity_id,$4,r.parsed,r.band,'jev',$5,'proposed'
FROM jsonb_to_recordset($3::jsonb) AS r(id uuid,program_id uuid,target_activity_id uuid,passage_id uuid,band text,parsed jsonb)
WHERE EXISTS (SELECT 1 FROM programs p WHERE p.org_id=$1::uuid AND p.id=r.program_id)
 AND NOT EXISTS (SELECT 1 FROM program_readings x WHERE x.org_id=$1::uuid AND x.program_id=r.program_id AND x.source='hold_point_evidence'
  AND x.target_activity_id=r.target_activity_id AND x.passage_id=r.passage_id AND x.status<>'proposed')`,
			org, documentID, raw, programread.FieldReleased, programread.HoldPointReleaseVersion); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// holdRelease is duration_source.released on a released hold point.
type holdRelease struct {
	By           string `json:"by"`
	ReadingID    string `json:"reading_id,omitempty"`
	DocumentID   string `json:"document_id,omitempty"`
	PassageID    string `json:"passage_id,omitempty"`
	DocumentDate string `json:"document_date,omitempty"`
}

// MsgHoldReleased refuses a second release of the same hold point.
const MsgHoldReleased = "This hold point is already released."

// ReleaseHoldPoint marks one hold point released at the user's request,
// with a proposed release reading as its evidence (readingID) or without
// (readingID ""). The reading is accepted with it. One program batch: one
// version bump, one undo step that clears the release and puts the
// reading back to proposed.
func (s *Store) ReleaseHoldPoint(ctx context.Context, org, project, actor string, version int64, holdID, readingID string) (ProgramResult, error) {
	op := map[string]any{"op": "release_hold_point", "id": holdID, "reading_id": readingID}
	return s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		a := w.live(holdID)
		if a == nil || a.Kind != engine.KindHoldPoint {
			return opErr(-1, "id", MsgNotInProgram)
		}
		source := map[string]json.RawMessage{}
		if len(a.DurationSource) > 0 {
			if err := json.Unmarshal(a.DurationSource, &source); err != nil {
				return err
			}
		}
		if _, done := source["released"]; done {
			return opErr(-1, "id", MsgHoldReleased)
		}
		rel := holdRelease{By: "user"}
		if readingID != "" {
			var target, status string
			var created time.Time
			err := tx.QueryRow(ctx, `SELECT COALESCE(r.target_activity_id::text,''),r.status,COALESCE(r.document_id::text,''),COALESCE(r.passage_id::text,''),d.created_at
FROM program_readings r JOIN documents d ON d.org_id=r.org_id AND d.id=r.document_id
WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.id::text=$3 AND r.source='hold_point_evidence'`, w.org, w.prog.ID, readingID).Scan(&target, &status, &rel.DocumentID, &rel.PassageID, &created)
			if errors.Is(err, pgx.ErrNoRows) || err == nil && (target != holdID || status != "proposed") {
				return opErr(-1, "reading_id", "That evidence is not proposed for this hold point.")
			}
			if err != nil {
				return err
			}
			rel.ReadingID, rel.DocumentDate = readingID, SydneyDate(created)
			w.readingState().status[readingID] = status
			w.setReading(readingID, "accepted")
		}
		raw, err := json.Marshal(rel)
		if err != nil {
			return err
		}
		source["released"] = raw
		merged, err := json.Marshal(source)
		if err != nil {
			return err
		}
		a = w.touchAct(holdID)
		a.DurationSource = merged
		if a.ReviewStatus == "proposed" {
			a.ReviewStatus = "accepted"
		}
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		return logProgramBatch(ctx, tx, w, actor, []any{op}, c)
	})
}

// ProgramHoldPointView is a hold point's release evidence and release as
// GET /program shows them (plan 10.4). Only hold points with either are
// listed. Limit is the known-limit tooltip text.
type ProgramHoldPointView struct {
	ActivityID string                `json:"activity_id"`
	Released   *ProgramHoldRelease   `json:"released"`
	Evidence   []ProgramHoldEvidence `json:"evidence"`
	Limit      string                `json:"limit"`
}

// ProgramHoldEvidence is one proposed release reading: "Evidence found:
// <document title>, page <n>", linking to the document and page.
type ProgramHoldEvidence struct {
	ReadingID     string `json:"reading_id"`
	DocumentID    string `json:"document_id"`
	PassageID     string `json:"passage_id"`
	DocumentTitle string `json:"document_title"`
	Page          int    `json:"page"`
	Band          string `json:"band"`
	Text          string `json:"text"`
	// DocumentDate is the document's filed date, from code; DateLabel
	// says so ("document dated"). Jev never gives it.
	DocumentDate string `json:"document_date"`
	DateLabel    string `json:"date_label"`
}

// ProgramHoldRelease is a user's release of a hold point. DocumentID is
// empty when it was released without evidence.
type ProgramHoldRelease struct {
	By            string `json:"by"`
	DocumentID    string `json:"document_id,omitempty"`
	PassageID     string `json:"passage_id,omitempty"`
	DocumentTitle string `json:"document_title,omitempty"`
	Page          int    `json:"page,omitempty"`
	DocumentDate  string `json:"document_date,omitempty"`
	DateLabel     string `json:"date_label,omitempty"`
}

const holdDateLabel = "document dated"

// holdPointExtras fills ex.HoldPoints from the live hold points' releases
// and their proposed release readings.
func holdPointExtras(ctx context.Context, q programQuerier, org string, st ProgramState, ex *ProgramExtras) error {
	ex.HoldPoints = []ProgramHoldPointView{}
	byID := map[string]*ProgramHoldPointView{}
	var order []string
	view := func(id string) *ProgramHoldPointView {
		if v := byID[id]; v != nil {
			return v
		}
		v := &ProgramHoldPointView{ActivityID: id, Evidence: []ProgramHoldEvidence{}, Limit: programread.HoldPointLimit}
		byID[id] = v
		order = append(order, id)
		return v
	}
	live := map[string]bool{}
	var withDoc []*ProgramHoldRelease
	for _, a := range st.Activities {
		if a.Kind != engine.KindHoldPoint || a.Retired {
			continue
		}
		live[a.ID] = true
		var source struct {
			Released *holdRelease `json:"released"`
		}
		if len(a.DurationSource) == 0 || json.Unmarshal(a.DurationSource, &source) != nil || source.Released == nil {
			continue
		}
		r := source.Released
		rel := &ProgramHoldRelease{By: r.By, DocumentID: r.DocumentID, PassageID: r.PassageID, DocumentDate: r.DocumentDate}
		if r.DocumentID != "" {
			rel.DateLabel = holdDateLabel
			withDoc = append(withDoc, rel)
		}
		view(a.ID).Released = rel
	}
	if len(live) == 0 {
		return nil
	}
	if err := releaseTitles(ctx, q, org, withDoc); err != nil {
		return err
	}
	rows, err := q.Query(ctx, `SELECT r.id::text,r.target_activity_id::text,r.document_id::text,COALESCE(r.passage_id::text,''),r.band,
 COALESCE(NULLIF(t.value,''),d.filename),COALESCE(ps.page,0),d.created_at
FROM program_readings r
JOIN documents d ON d.org_id=r.org_id AND d.id=r.document_id
LEFT JOIN decisions t ON t.org_id=d.org_id AND t.document_id=d.id AND t.field='title'
LEFT JOIN passage_sources ps ON ps.org_id=r.org_id AND ps.passage_id=r.passage_id
WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.source='hold_point_evidence' AND r.status='proposed' AND r.target_activity_id IS NOT NULL
ORDER BY r.created_at,r.id`, org, st.Program.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e ProgramHoldEvidence
		var target string
		var created time.Time
		if err := rows.Scan(&e.ReadingID, &target, &e.DocumentID, &e.PassageID, &e.Band, &e.DocumentTitle, &e.Page, &created); err != nil {
			return err
		}
		if !live[target] {
			continue
		}
		e.DocumentDate, e.DateLabel = SydneyDate(created), holdDateLabel
		e.Text = "Evidence found: " + e.DocumentTitle
		if e.Page > 0 {
			e.Text += ", page " + strconv.Itoa(e.Page)
		}
		v := view(target)
		v.Evidence = append(v.Evidence, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range order {
		ex.HoldPoints = append(ex.HoldPoints, *byID[id])
	}
	return nil
}

// releaseTitles fills the document title and page of released hold points
// in one query, however many there are.
func releaseTitles(ctx context.Context, q programQuerier, org string, rels []*ProgramHoldRelease) error {
	if len(rels) == 0 {
		return nil
	}
	type key struct{ doc, passage string }
	var docs, passages []string
	for _, r := range rels {
		docs, passages = append(docs, r.DocumentID), append(passages, r.PassageID)
	}
	rows, err := q.Query(ctx, `SELECT x.doc,x.passage,COALESCE(NULLIF(t.value,''),d.filename),COALESCE(ps.page,0)
FROM unnest($2::text[],$3::text[]) AS x(doc,passage)
JOIN documents d ON d.org_id=$1::uuid AND d.id::text=x.doc
LEFT JOIN decisions t ON t.org_id=d.org_id AND t.document_id=d.id AND t.field='title'
LEFT JOIN passage_sources ps ON ps.org_id=d.org_id AND ps.passage_id::text=x.passage`, org, docs, passages)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := map[key]struct {
		title string
		page  int
	}{}
	for rows.Next() {
		var k key
		var title string
		var page int
		if err := rows.Scan(&k.doc, &k.passage, &title, &page); err != nil {
			return err
		}
		found[k] = struct {
			title string
			page  int
		}{title, page}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range rels {
		f := found[key{r.DocumentID, r.PassageID}]
		r.DocumentTitle, r.Page = f.title, f.page
	}
	return nil
}

// CheckHoldPointDismissed names a hold point the user deleted with a reason.
const CheckHoldPointDismissed = "hold_point_dismissed"

// dismissedHoldPointChecks keeps a check visible for each retired hold point
// that carries a dismissal reason (plan 13.1 "Deleting a hold point"), until
// undo restores the row and clears its reason.
func dismissedHoldPointChecks(ctx context.Context, q programQuerier, org, programID string, ex *ProgramExtras) error {
	// The row is retired, so the check names it in words, not by id.
	rows, err := q.Query(ctx, `SELECT label,dismissed_reason FROM program_activities
 WHERE org_id=$1::uuid AND program_id=$2::uuid AND kind='hold_point' AND retired_at IS NOT NULL AND COALESCE(trim(dismissed_reason),'')<>''
 ORDER BY retired_at,ordinal,id`, org, programID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var label, reason string
		if err := rows.Scan(&label, &reason); err != nil {
			return err
		}
		ex.Checks = append(ex.Checks, engine.Check{Code: CheckHoldPointDismissed,
			Message: "Hold point removed: " + label + ". Reason: " + strings.TrimSuffix(strings.TrimSpace(reason), ".") + "."})
	}
	return rows.Err()
}
