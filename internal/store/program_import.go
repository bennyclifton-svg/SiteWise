package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
	"sitewise/internal/programread"
)

// Program file import (plan docs/plans/2026-10-18-program.md section 10.2).
// A file dropped on Program is filed by intake unchanged; a program_imports
// row marks it for the program_import read that follows filing. The read's
// matches are written as program_readings, proposals a user accepts; this
// file never changes a program row.

// Import statuses.
const (
	ProgramImportFiling  = "filing"
	ProgramImportReading = "reading"
	ProgramImportRead    = "read"
	ProgramImportNotRead = "not_read"
)

// programImportMaxAttempts is how often a grey read is asked again, as for
// a proposal read (DueSubmissionRechecks).
const programImportMaxAttempts = 3

// ProgramImportRef names one import in its org. Began is when the file was
// dropped, for the program_import timing.
type ProgramImportRef struct {
	OrgID, ImportID string
	Began           time.Time
}

// OpenProgramImport marks a document of the project for a program read and
// returns the import id. The project must have a program. Dropping the same
// file again reads it again.
func (s *Store) OpenProgramImport(ctx context.Context, org, project, documentID string) (string, error) {
	var programID, filename string
	err := s.pool.QueryRow(ctx, `SELECT p.id::text, d.filename FROM programs p JOIN documents d ON d.org_id=p.org_id AND d.project_id=p.project_id
WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND d.id=$3::uuid`, org, project, documentID).Scan(&programID, &filename)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	var id string
	err = s.pool.QueryRow(ctx, `INSERT INTO program_imports(org_id,id,program_id,document_id,filename)
VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5)
ON CONFLICT (org_id,program_id,document_id) DO UPDATE SET status='filing',not_read=NULL,retry_after=NULL,read_attempts=0,updated_at=now()
RETURNING id::text`, org, newID(), programID, documentID, clipText(filename, 500)).Scan(&id)
	return id, err
}

// ClaimProgramImports moves a filed document's waiting imports to reading.
// It is the filing hook's lookup: a document with none is not a program
// import, and costs one indexed query.
func (s *Store) ClaimProgramImports(ctx context.Context, org, documentID string) ([]ProgramImportRef, error) {
	return s.claimImports(ctx, `i.document_id=$2::uuid`, org, documentID)
}

// ClaimProgramImport claims one import whose document is already filed.
func (s *Store) ClaimProgramImport(ctx context.Context, org, id string) ([]ProgramImportRef, error) {
	return s.claimImports(ctx, `i.id=$2::uuid`, org, id)
}

func (s *Store) claimImports(ctx context.Context, where, org, arg string) ([]ProgramImportRef, error) {
	rows, err := s.pool.Query(ctx, `UPDATE program_imports u SET status='reading' FROM program_imports i
WHERE i.org_id=$1::uuid AND `+where+` AND i.status='filing' AND u.org_id=i.org_id AND u.id=i.id
RETURNING u.id::text, i.updated_at`, org, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProgramImportRef
	for rows.Next() {
		r := ProgramImportRef{OrgID: org}
		if err := rows.Scan(&r.ImportID, &r.Began); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// PendingProgramImports are reads a restart must finish: claimed and not
// committed, or waiting on a document that has filed since.
func (s *Store) PendingProgramImports(ctx context.Context) ([]ProgramImportRef, error) {
	rows, err := s.pool.Query(ctx, `SELECT i.org_id::text, i.id::text, i.updated_at FROM program_imports i
JOIN documents d ON d.org_id=i.org_id AND d.id=i.document_id
WHERE i.status='reading' OR (i.status='filing' AND d.status=$1)`, StatusFiled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProgramImportRef
	for rows.Next() {
		var r ProgramImportRef
		if err := rows.Scan(&r.OrgID, &r.ImportID, &r.Began); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DueProgramImportRechecks are grey reads to ask again, at most three times.
func (s *Store) DueProgramImportRechecks(ctx context.Context, now time.Time) ([]ProgramImportRef, error) {
	rows, err := s.pool.Query(ctx, `SELECT org_id::text, id::text, updated_at FROM program_imports
WHERE retry_after IS NOT NULL AND retry_after <= $1 AND read_attempts < $2 AND status='read'`, now, programImportMaxAttempts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProgramImportRef
	for rows.Next() {
		var r ProgramImportRef
		if err := rows.Scan(&r.OrgID, &r.ImportID, &r.Began); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClaimProgramImportRecheck clears the retry so one sweep reads it once.
func (s *Store) ClaimProgramImportRecheck(ctx context.Context, ref ProgramImportRef, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE program_imports SET retry_after=NULL WHERE org_id=$1::uuid AND id=$2::uuid AND retry_after IS NOT NULL AND retry_after <= $3`, ref.OrgID, ref.ImportID, now)
	return tag.RowsAffected() == 1, err
}

// ProgramImportInput is what one read needs: the file, the program's
// working week, and its live rows as targets in program order.
type ProgramImportInput struct {
	ImportID, ProgramID, ProjectID, DocumentID, Filename, Status string
	SHA256                                                       []byte
	Size                                                         int64
	PerWeek                                                      int
	Targets                                                      []programread.Target
}

// ProgramImportForReading loads one import's file and targets.
func (s *Store) ProgramImportForReading(ctx context.Context, org, id string) (ProgramImportInput, error) {
	in := ProgramImportInput{ImportID: id}
	err := s.pool.QueryRow(ctx, `SELECT i.program_id::text, p.project_id::text, i.document_id::text, i.filename, i.status, f.sha256, f.byte_size
FROM program_imports i JOIN programs p ON p.org_id=i.org_id AND p.id=i.program_id
JOIN documents d ON d.org_id=i.org_id AND d.id=i.document_id JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE i.org_id=$1::uuid AND i.id=$2::uuid`, org, id).Scan(&in.ProgramID, &in.ProjectID, &in.DocumentID, &in.Filename, &in.Status, &in.SHA256, &in.Size)
	if errors.Is(err, pgx.ErrNoRows) {
		return in, ErrNotFound
	}
	if err != nil {
		return in, err
	}
	st, err := s.LoadProgram(ctx, org, in.ProjectID)
	if err != nil {
		return in, err
	}
	in.PerWeek = engine.NewCalendar(st.Input().Calendar).WorkingDaysPerWeek()
	in.Targets = programTargets(st.Activities)
	return in, nil
}

// programTargets lists live rows with their tier (plan 10.2 "Targets"):
// stages, then package activities, milestones and hold points, then
// fragment activities. The reader orders them; program order is kept
// within a tier.
func programTargets(acts []ProgramActivity) []programread.Target {
	labels := map[string]string{}
	for _, a := range acts {
		if a.Kind == engine.KindStage {
			labels[a.ID] = a.Label
		}
	}
	out := make([]programread.Target, 0, len(acts))
	for _, a := range acts {
		t := programread.Target{ID: a.ID, Label: a.Label, Kind: a.Kind, StageLabel: labels[string(a.ParentID)], Tier: programread.TierPackage}
		switch {
		case a.Kind == engine.KindStage:
			t.Tier, t.CatalogueStageID = programread.TierStage, string(a.CatalogueStageID)
			// A user-made stage receives evidence only when the user assigns
			// it (plan 7.7): it is matched, but carries no catalogue stage.
			if t.CatalogueStageID == "other" {
				t.CatalogueStageID = ""
			}
		case a.Kind == engine.KindActivity && a.FragmentID != "":
			t.Tier = programread.TierFragment
		}
		out = append(out, t)
	}
	return out
}

// ProgramImportCommitted is what the reader announces.
type ProgramImportCommitted struct {
	ProjectID, ProgramID, DocumentID string
	ReadingsPending                  int
}

type readingRecord struct {
	ID               string          `json:"id"`
	Target           string          `json:"target_activity_id"`
	CatalogueStageID *string         `json:"catalogue_stage_id"`
	Field            string          `json:"field"`
	Value            string          `json:"value"`
	Parsed           json.RawMessage `json:"parsed"`
	Band             string          `json:"band"`
	DecidedBy        string          `json:"decided_by"`
	Confidence       *float64        `json:"confidence"`
	QuestionVersion  *string         `json:"question_version"`
}

// CommitProgramImport stores one read. Proposed readings from an earlier
// read of the same file are replaced; a reading the user accepted or
// dismissed is never replaced or proposed again. notRead marks a file code
// could not read. A grey read is asked again after 30 seconds.
func (s *Store) CommitProgramImport(ctx context.Context, org, id string, out programread.Outcome, notRead string, recheck bool, now time.Time) (ProgramImportCommitted, bool, error) {
	var c ProgramImportCommitted
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return c, false, err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT i.status, i.program_id::text, i.document_id::text, p.project_id::text FROM program_imports i
JOIN programs p ON p.org_id=i.org_id AND p.id=i.program_id WHERE i.org_id=$1::uuid AND i.id=$2::uuid FOR UPDATE OF i`, org, id).Scan(&status, &c.ProgramID, &c.DocumentID, &c.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, false, ErrNotFound
	}
	if err != nil {
		return c, false, err
	}
	if status != ProgramImportReading && !(recheck && status == ProgramImportRead) {
		return c, false, nil
	}
	attempt := 0
	if recheck {
		attempt = 1
	}
	if notRead != "" {
		if _, err := tx.Exec(ctx, `UPDATE program_imports SET status='not_read',not_read=$3,retry_after=NULL,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, notRead); err != nil {
			return c, false, err
		}
		return c, true, tx.Commit(ctx)
	}
	decided := map[string]bool{}
	rows, err := tx.Query(ctx, `SELECT target_activity_id::text, field FROM program_readings
WHERE org_id=$1::uuid AND program_id=$2::uuid AND document_id=$3::uuid AND source='program_file' AND status<>'proposed' AND target_activity_id IS NOT NULL`, org, c.ProgramID, c.DocumentID)
	if err != nil {
		return c, false, err
	}
	for rows.Next() {
		var t, f string
		if err := rows.Scan(&t, &f); err != nil {
			rows.Close()
			return c, false, err
		}
		decided[t+"|"+f] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return c, false, err
	}
	// A row the user already added from this file is that activity: it is
	// not proposed back to itself.
	rows, err = tx.Query(ctx, `SELECT id::text FROM program_activities WHERE org_id=$1::uuid AND program_id=$2::uuid AND retired_at IS NULL AND source_key LIKE $3`, org, c.ProgramID, importRowKey(id, "")+"%")
	if err != nil {
		return c, false, err
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return c, false, err
		}
		for _, f := range []string{programread.FieldStart, programread.FieldFinish, programread.FieldDuration} {
			decided[t+"|"+f] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return c, false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM program_readings WHERE org_id=$1::uuid AND program_id=$2::uuid AND document_id=$3::uuid AND source='program_file' AND status='proposed'`, org, c.ProgramID, c.DocumentID); err != nil {
		return c, false, err
	}
	var recs []readingRecord
	for _, r := range out.Readings {
		if decided[r.TargetID+"|"+r.Field] {
			continue
		}
		parsed, err := json.Marshal(r.Parsed)
		if err != nil {
			return c, false, err
		}
		rec := readingRecord{ID: newID(), Target: r.TargetID, Field: r.Field, Value: clipText(r.Value, 500), Parsed: parsed, Band: r.Band, DecidedBy: r.DecidedBy, Confidence: r.Confidence}
		if r.CatalogueStageID != "" {
			v := r.CatalogueStageID
			rec.CatalogueStageID = &v
		}
		if r.QuestionVersion != "" {
			v := r.QuestionVersion
			rec.QuestionVersion = &v
		}
		recs = append(recs, rec)
	}
	if len(recs) > 0 {
		raw, err := json.Marshal(recs)
		if err != nil {
			return c, false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO program_readings(org_id,id,program_id,source,document_id,target_activity_id,catalogue_stage_id,field,value,parsed,band,decided_by,confidence,question_version,status)
SELECT $1::uuid,r.id,$2::uuid,'program_file',$3::uuid,r.target_activity_id,r.catalogue_stage_id,r.field,r.value,r.parsed,r.band,r.decided_by,r.confidence,r.question_version,'proposed'
FROM jsonb_to_recordset($4::jsonb) AS r(id uuid,target_activity_id uuid,catalogue_stage_id text,field text,value text,parsed jsonb,band text,decided_by text,confidence double precision,question_version text)`,
			org, c.ProgramID, c.DocumentID, raw); err != nil {
			return c, false, err
		}
	}
	fileRows, _ := json.Marshal(nonNilRows(out.Rows))
	notMatched, _ := json.Marshal(stringsOrEmpty(out.NotMatched))
	grey, _ := json.Marshal(stringsOrEmpty(out.Grey))
	checks, _ := json.Marshal(stringsOrEmpty(out.Checks))
	var retry *time.Time
	if len(out.Grey) > 0 {
		t := now.Add(30 * time.Second)
		retry = &t
	}
	if _, err := tx.Exec(ctx, `UPDATE program_imports SET status='read',not_read=NULL,file_rows=$3::jsonb,not_matched=$4::jsonb,grey_targets=$5::jsonb,checks=$6::jsonb,
 retry_after=$7,read_attempts=read_attempts+$8,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, fileRows, notMatched, grey, checks, retry, attempt); err != nil {
		return c, false, err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM program_readings r WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.status='proposed' AND `+readingVisible("r"), org, c.ProgramID).Scan(&c.ReadingsPending); err != nil {
		return c, false, err
	}
	return c, true, tx.Commit(ctx)
}

func nonNilRows(rows []programread.Row) []programread.Row {
	if rows == nil {
		return []programread.Row{}
	}
	return rows
}

// ProgramReadingView is one proposed reading as GET /program shows it,
// small: the target, the field, the file's text and code's parse.
// Applies is false when the target holds a user or accepted value, which
// the reading never overwrites (plan section 2, Jev rule 14).
type ProgramReadingView struct {
	ID               string          `json:"id"`
	ImportID         string          `json:"import_id"`
	TargetActivityID NullText        `json:"target_activity_id"`
	CatalogueStageID NullText        `json:"catalogue_stage_id"`
	Field            string          `json:"field"`
	Value            string          `json:"value"`
	Parsed           json.RawMessage `json:"parsed"`
	Band             string          `json:"band"`
	DecidedBy        string          `json:"decided_by"`
	DocumentID       string          `json:"document_id"`
	DocumentTitle    string          `json:"document_title"`
	Applies          bool            `json:"applies"`
	// Source is program_file or proposal_r7. Text is the line an awarded
	// proposal's reading shows: "From the awarded proposal: Design
	// development 6 weeks".
	Source string `json:"source"`
	Text   string `json:"text,omitempty"`
}

// catalogueLabel names a catalogue stage the way the user reads it; the
// id stands in only when the catalogue is not loaded.
func catalogueLabel(ex *ProgramExtras, id string) string {
	for _, c := range ex.CatalogueStages {
		if c.ID == id {
			return c.Label
		}
	}
	return id
}

// proposalReadingView resolves an awarded proposal's reading to its live
// row, or adds the "no stage" notice when the stage is gone (plan 7.7).
func proposalReadingView(v ProgramReadingView, acts, byCatalogue map[string]*ProgramActivity, p Program, ex *ProgramExtras) ProgramReadingView {
	target := resolveTarget(acts, byCatalogue, string(v.TargetActivityID), string(v.CatalogueStageID))
	if target == nil {
		if v.CatalogueStageID != "" {
			ex.Checks = append(ex.Checks, engine.Check{Code: "evidence_without_stage", Message: "Evidence for " + catalogueLabel(ex, string(v.CatalogueStageID)) + " has no stage. Add the stage back or dismiss."})
		}
		return v
	}
	v.TargetActivityID = NullText(target.ID)
	v.Applies = readingApplies(target, v.Field, p)
	return v
}

// ProgramFileRow is a file row no target matched, which the user can add
// as a new activity in one click.
type ProgramFileRow struct {
	Row      string `json:"row"`
	Label    string `json:"label"`
	Start    string `json:"start,omitempty"`
	Finish   string `json:"finish,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// ProgramProposedLink is a link code reads from the file's predecessor
// column between two rows that both matched an activity.
type ProgramProposedLink struct {
	PredID  string `json:"pred_id"`
	SuccID  string `json:"succ_id"`
	Type    string `json:"type"`
	LagDays int    `json:"lag_days"`
	LagUnit string `json:"lag_unit"`
}

// ProgramImportView is one imported file for the toolbar and the readings
// panel: "4 readings from Smith program.xlsx".
type ProgramImportView struct {
	ID            string                `json:"id"`
	DocumentID    string                `json:"document_id"`
	Filename      string                `json:"filename"`
	Status        string                `json:"status"`
	NotRead       NullText              `json:"not_read"`
	Text          string                `json:"text"`
	Readings      int                   `json:"readings"`
	NotMatched    []string              `json:"not_matched"`
	Grey          []string              `json:"grey"`
	UnmatchedRows []ProgramFileRow      `json:"unmatched_rows"`
	Unmatched     int                   `json:"unmatched"`
	Links         []ProgramProposedLink `json:"links"`
}

// maxUnmatchedShown keeps GET /program small for a large file; the count
// says how many there are.
const maxUnmatchedShown = 200

// readingExtras adds proposed readings, imports and their checks to a read.
func (s *Store) readingExtras(ctx context.Context, q programQuerier, org string, st ProgramState, ex *ProgramExtras) error {
	ex.Readings, ex.Imports = []ProgramReadingView{}, []ProgramImportView{}
	type imp struct {
		view   ProgramImportView
		rows   []programread.Row
		checks []string
		docSt  string
	}
	var imports []*imp
	byDoc := map[string]*imp{}
	rows, err := q.Query(ctx, `SELECT i.id::text,i.document_id::text,i.filename,i.status,COALESCE(i.not_read,''),i.file_rows::text,i.not_matched::text,i.grey_targets::text,i.checks::text,COALESCE(d.status,'')
FROM program_imports i LEFT JOIN documents d ON d.org_id=i.org_id AND d.id=i.document_id
WHERE i.org_id=$1::uuid AND i.program_id=$2::uuid AND i.status<>'dismissed' ORDER BY i.created_at,i.id`, org, st.Program.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var m imp
		var fileRows, notMatched, grey, checks string
		if err := rows.Scan(&m.view.ID, &m.view.DocumentID, &m.view.Filename, &m.view.Status, (*string)(&m.view.NotRead), &fileRows, &notMatched, &grey, &checks, &m.docSt); err != nil {
			rows.Close()
			return err
		}
		if err := errors.Join(json.Unmarshal([]byte(fileRows), &m.rows), json.Unmarshal([]byte(notMatched), &m.view.NotMatched), json.Unmarshal([]byte(grey), &m.view.Grey), json.Unmarshal([]byte(checks), &m.checks)); err != nil {
			rows.Close()
			return err
		}
		imports = append(imports, &m)
		byDoc[m.view.DocumentID] = &m
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	acts := map[string]*ProgramActivity{}
	byCatalogue := map[string]*ProgramActivity{}
	added := map[string]bool{}
	for i := range st.Activities {
		a := &st.Activities[i]
		acts[a.ID] = a
		if a.Kind == engine.KindStage && a.CatalogueStageID != "" && a.CatalogueStageID != "other" {
			byCatalogue[string(a.CatalogueStageID)] = a
		}
		if a.SourceKey != "" {
			added[string(a.SourceKey)] = true
		}
	}
	matched := map[string]map[string]string{} // document → row → target
	// Program file readings, and an awarded proposal's R7 readings (plan
	// 10.3): a proposal's readings stay hidden until its submission is
	// awarded, and hide again when the award is undone.
	rows, err = q.Query(ctx, `SELECT r.id::text,COALESCE(r.document_id::text,''),COALESCE(r.target_activity_id::text,''),COALESCE(r.catalogue_stage_id,''),r.field,COALESCE(r.value,''),r.parsed::text,r.band,r.decided_by,r.status,r.source,COALESCE(d.filename,'')
FROM program_readings r LEFT JOIN documents d ON d.org_id=r.org_id AND d.id=r.document_id
WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.source IN ('program_file','proposal_r7') AND `+readingVisible("r")+` AND r.status<>'dismissed' ORDER BY r.created_at,r.field,r.id`, org, st.Program.ID)
	if err != nil {
		return err
	}
	targetsWith := map[string]map[string]bool{}
	for rows.Next() {
		var v ProgramReadingView
		var parsed, status, filename string
		if err := rows.Scan(&v.ID, &v.DocumentID, (*string)(&v.TargetActivityID), (*string)(&v.CatalogueStageID), &v.Field, &v.Value, &parsed, &v.Band, &v.DecidedBy, &status, &v.Source, &filename); err != nil {
			rows.Close()
			return err
		}
		v.Parsed = json.RawMessage(parsed)
		if v.Source == "proposal_r7" {
			if status == "proposed" {
				v.DocumentTitle, v.Text = filename, proposalReadingText(v.Parsed)
				ex.Readings = append(ex.Readings, proposalReadingView(v, acts, byCatalogue, st.Program, ex))
			}
			continue
		}
		m := byDoc[v.DocumentID]
		if m == nil {
			continue
		}
		v.ImportID, v.DocumentTitle = m.view.ID, m.view.Filename
		target := resolveTarget(acts, byCatalogue, string(v.TargetActivityID), string(v.CatalogueStageID))
		if row := parsedRow(v.Parsed); row != "" && target != nil {
			if matched[v.DocumentID] == nil {
				matched[v.DocumentID] = map[string]string{}
			}
			matched[v.DocumentID][row] = target.ID
		}
		if status != "proposed" {
			continue
		}
		if target == nil {
			if v.CatalogueStageID != "" {
				label := catalogueLabel(ex, string(v.CatalogueStageID))
				ex.Checks = append(ex.Checks, engine.Check{Code: "evidence_without_stage", Message: "Evidence for " + label + " has no stage. Add the stage back or dismiss."})
			}
		} else {
			v.TargetActivityID = NullText(target.ID)
			v.Applies = readingApplies(target, v.Field, st.Program)
		}
		if targetsWith[m.view.ID] == nil {
			targetsWith[m.view.ID] = map[string]bool{}
		}
		targetsWith[m.view.ID][string(v.TargetActivityID)] = true
		ex.Readings = append(ex.Readings, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	linked := map[[2]string]bool{}
	for _, l := range st.Links {
		linked[[2]string{l.PredID, l.SuccID}], linked[[2]string{l.SuccID, l.PredID}] = true, true
	}
	for _, m := range imports {
		v := &m.view
		v.UnmatchedRows, v.Links = []ProgramFileRow{}, []ProgramProposedLink{}
		if v.Status == ProgramImportFiling && m.docSt == StatusNotFiled {
			v.Status, v.NotRead = ProgramImportNotRead, "not_filed"
		}
		v.Readings = len(targetsWith[v.ID])
		v.Text = importText(*v)
		rowTarget := matched[v.DocumentID]
		for _, r := range m.rows {
			if _, ok := rowTarget[r.ID]; ok || added[importRowKey(v.ID, r.ID)] {
				continue
			}
			v.Unmatched++
			if len(v.UnmatchedRows) < maxUnmatchedShown {
				v.UnmatchedRows = append(v.UnmatchedRows, ProgramFileRow{Row: r.ID, Label: r.Label, Start: r.Start, Finish: r.Finish, Duration: r.DurationRaw})
			}
		}
		for _, l := range importLinks(m.rows, rowTarget) {
			if !linked[[2]string{l.PredID, l.SuccID}] {
				v.Links = append(v.Links, l)
			}
		}
		for _, c := range m.checks {
			ex.Checks = append(ex.Checks, engine.Check{Code: "date_order_assumed", Message: c + " (" + v.Filename + ")."})
		}
		ex.Imports = append(ex.Imports, *v)
	}
	return nil
}

// importText is the toolbar line for one file.
func importText(v ProgramImportView) string {
	switch v.Status {
	case ProgramImportFiling, ProgramImportReading:
		return "Reading " + v.Filename + "…"
	case ProgramImportNotRead:
		return v.Filename + " could not be read."
	}
	switch v.Readings {
	case 0:
		return "No readings from " + v.Filename
	case 1:
		return "1 reading from " + v.Filename
	}
	return strconv.Itoa(v.Readings) + " readings from " + v.Filename
}

func importRowKey(importID, row string) string { return "import:" + importID + ":" + row }

// resolveTarget is the live row a reading applies to: its own target, or,
// for evidence about a stage, the live stage with its catalogue stage id,
// so renaming or reordering never loses it (plan 7.7). Nil when neither
// is live.
func resolveTarget(acts, byCatalogue map[string]*ProgramActivity, target, catalogue string) *ProgramActivity {
	if a := acts[target]; a != nil {
		return a
	}
	if catalogue != "" {
		return byCatalogue[catalogue]
	}
	return nil
}

func parsedRow(raw json.RawMessage) string {
	var p struct {
		Row string `json:"row"`
	}
	_ = json.Unmarshal(raw, &p)
	return p.Row
}

// readingApplies reports whether accepting would change the target: a user
// or accepted document value is never overwritten.
func readingApplies(a *ProgramActivity, field string, p Program) bool {
	switch field {
	case programread.FieldStart:
		return a.ConstraintKind == engine.ConstraintASAP && p.StartDate != ""
	default:
		return a.DurationOrigin != "user" && a.DurationOrigin != "document"
	}
}

// importLinks are the file's predecessor links between rows that both
// matched an activity, by activity id.
func importLinks(rows []programread.Row, rowTarget map[string]string) []ProgramProposedLink {
	var out []ProgramProposedLink
	for _, r := range rows {
		succ, ok := rowTarget[r.ID]
		if !ok {
			continue
		}
		for _, p := range r.PredLinks {
			pred, ok := rowTarget[p.Row]
			if !ok || pred == succ {
				continue
			}
			out = append(out, ProgramProposedLink{PredID: pred, SuccID: succ, Type: p.Type, LagDays: p.Lag, LagUnit: p.LagUnit})
		}
	}
	return out
}

// ImportReadingsPending counts proposed readings, for the reader's event.
func (s *Store) ImportReadingsPending(ctx context.Context, org, programID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM program_readings r WHERE r.org_id=$1::uuid AND r.program_id=$2::uuid AND r.status='proposed' AND `+readingVisible("r"), org, programID).Scan(&n)
	return n, err
}
