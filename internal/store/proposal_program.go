package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	engine "sitewise/internal/program"
	"sitewise/internal/programread"
)

// Reading B (plan docs/plans/2026-10-18-program.md section 10.3): the R7
// Program returnable in a firm's proposal. After the proposal read commits,
// a proposal_program read runs on the same submission; its stage durations
// are stored as program_readings with source 'proposal_r7'. Before award
// nothing changes in Program: such a reading is hidden until its submission
// holds a live award, and an undone award hides it again (readingVisible).
// Accepting one goes through the program's own accept path. No path here
// calls Jev.

// Proposal program read states, on the submission.
const (
	ProposalProgramReading = "reading"
	ProposalProgramRead    = "read"
	ProposalProgramNotRead = "not_read"
)

// Reasons a proposal's program is not read.
const (
	ProposalProgramNoProgram = "no_program"
	ProposalProgramNoStages  = "no_stages"
)

// proposalProgramMaxAttempts is how often a grey read is asked again, as
// for the proposal read and the program import.
const proposalProgramMaxAttempts = 3

// readingVisible is the condition under which a stored reading shows in
// Program and can be accepted: a proposal_r7 reading only while its
// submission holds a live award (plan 10.3 "User sees"). alias names the
// program_readings row.
func readingVisible(alias string) string {
	return "(" + alias + ".source<>'proposal_r7' OR EXISTS (SELECT 1 FROM awards aw WHERE aw.org_id=" + alias + ".org_id AND aw.submission_id::text=" + alias + ".parsed->>'submission_id' AND aw.undone_at IS NULL))"
}

// QueueProposalProgram marks a read submission for its program read and
// reports whether to start it: the submission must have a document and a
// package, and be a proposal. A read already queued is read again.
func (s *Store) QueueProposalProgram(ctx context.Context, org, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE submissions SET program_read='reading',program_not_read=NULL,program_retry_after=NULL,program_read_attempts=0
WHERE org_id=$1::uuid AND id::text=$2 AND document_id IS NOT NULL AND package_id IS NOT NULL AND status NOT IN ('reading','not_proposal','withdrawn')`, org, id)
	return tag.RowsAffected() == 1, err
}

// PendingProposalPrograms are program reads a restart must finish.
func (s *Store) PendingProposalPrograms(ctx context.Context) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `SELECT org_id::text, id::text FROM submissions WHERE program_read='reading'`)
}

// DueProposalProgramRechecks are grey program reads to ask again, at most
// three times.
func (s *Store) DueProposalProgramRechecks(ctx context.Context, now time.Time) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `SELECT org_id::text, id::text FROM submissions WHERE program_read='read' AND program_retry_after IS NOT NULL AND program_retry_after <= $1 AND program_read_attempts < $2`, now, proposalProgramMaxAttempts)
}

// ClaimProposalProgramRecheck clears the retry so one sweep reads it once.
func (s *Store) ClaimProposalProgramRecheck(ctx context.Context, ref SubmissionRef, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE submissions SET program_retry_after=NULL WHERE org_id=$1::uuid AND id=$2::uuid AND program_retry_after IS NOT NULL AND program_retry_after <= $3`, ref.OrgID, ref.SubmissionID, now)
	return tag.RowsAffected() == 1, err
}

// ProposalProgramInput is what one read needs: the proposal file, the
// program's working week, and the package's live stages with their
// program rows. NotRead is set when there is nothing to read into.
type ProposalProgramInput struct {
	SubmissionID, ProjectID, DocumentID, PackageID, Filename, Status string
	SHA256                                                           []byte
	Size                                                             int64
	ProgramID                                                        string
	PerWeek                                                          int
	Stages                                                           []programread.Stage
	NotRead                                                          string
}

// ProposalProgramForReading loads one submission's program read.
func (s *Store) ProposalProgramForReading(ctx context.Context, org, id string) (ProposalProgramInput, error) {
	in := ProposalProgramInput{SubmissionID: id}
	var status *string
	err := s.pool.QueryRow(ctx, `SELECT s.project_id::text, COALESCE(s.document_id::text,''), COALESCE(s.package_id::text,''), s.filename, s.program_read, f.sha256, f.byte_size
FROM submissions s JOIN documents d ON d.org_id=s.org_id AND d.id=s.document_id JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE s.org_id=$1::uuid AND s.id::text=$2`, org, id).Scan(&in.ProjectID, &in.DocumentID, &in.PackageID, &in.Filename, &status, &in.SHA256, &in.Size)
	if errors.Is(err, pgx.ErrNoRows) {
		return in, ErrNotFound
	}
	if err != nil {
		return in, err
	}
	if status != nil {
		in.Status = *status
	}
	st, err := s.LoadProgram(ctx, org, in.ProjectID)
	if errors.Is(err, ErrNotFound) {
		in.NotRead = ProposalProgramNoProgram
		return in, nil
	}
	if err != nil {
		return in, err
	}
	in.ProgramID = st.Program.ID
	in.PerWeek = engine.NewCalendar(st.Input().Calendar).WorkingDaysPerWeek()
	rows, err := s.pool.Query(ctx, `SELECT id::text, label, COALESCE(program_stage_id::text,''), stage_id FROM package_stages
WHERE org_id=$1::uuid AND package_id::text=$2 AND retired_at IS NULL ORDER BY ordinal, id`, org, in.PackageID)
	if err != nil {
		return in, err
	}
	defer rows.Close()
	var stages []packageStageRef
	for rows.Next() {
		var p packageStageRef
		if err := rows.Scan(&p.ID, &p.Label, &p.ProgramStage, &p.Catalogue); err != nil {
			return in, err
		}
		stages = append(stages, p)
	}
	if err := rows.Err(); err != nil {
		return in, err
	}
	in.Stages = programStagesFor(st.Activities, stages)
	if len(in.Stages) == 0 {
		in.NotRead = ProposalProgramNoStages
	}
	return in, nil
}

// packageStageRef is one live package stage: its program stage, and the
// catalogue stage it was made from.
type packageStageRef struct{ ID, Label, ProgramStage, Catalogue string }

// programStagesFor pairs each package stage with the program's package
// activity for it and the catalogue stage of the program stage it sits in,
// so the evidence survives a rename or reorder (plan 7.7). A user-made
// stage ("other") receives evidence only when the user assigns it.
func programStagesFor(acts []ProgramActivity, stages []packageStageRef) []programread.Stage {
	byPackageStage := map[string]string{}
	catalogueOf := map[string]string{}
	for _, a := range acts {
		if a.PackageStageID != "" && a.Kind != engine.KindStage {
			byPackageStage[string(a.PackageStageID)] = a.ID
		}
		if a.Kind == engine.KindStage {
			catalogueOf[a.ID] = string(a.CatalogueStageID)
		}
	}
	out := make([]programread.Stage, 0, len(stages))
	for _, p := range stages {
		catalogue := p.Catalogue
		if c, ok := catalogueOf[p.ProgramStage]; ok {
			catalogue = c
		}
		if catalogue == "other" {
			catalogue = ""
		}
		out = append(out, programread.Stage{ID: p.ID, Label: p.Label, TargetActivityID: byPackageStage[p.ID], CatalogueStageID: catalogue})
	}
	return out
}

// ProposalProgramCommitted is what the reader announces.
type ProposalProgramCommitted struct {
	ProjectID, ProgramID, DocumentID string
	Readings                         int
}

// CommitProposalProgram stores one read. Proposed readings from an
// earlier read of the same proposal are replaced; a reading the user
// accepted or dismissed is never replaced or proposed again. notRead marks
// a proposal with nothing to read into. A grey read is asked again after
// 30 seconds.
func (s *Store) CommitProposalProgram(ctx context.Context, org, id string, in ProposalProgramInput, out programread.StageOutcome, recheck bool, now time.Time) (ProposalProgramCommitted, bool, error) {
	c := ProposalProgramCommitted{ProjectID: in.ProjectID, ProgramID: in.ProgramID, DocumentID: in.DocumentID}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return c, false, err
	}
	defer tx.Rollback(ctx)
	var status *string
	var doc string
	err = tx.QueryRow(ctx, `SELECT program_read, COALESCE(document_id::text,'') FROM submissions WHERE org_id=$1::uuid AND id::text=$2 FOR UPDATE`, org, id).Scan(&status, &doc)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, false, ErrNotFound
	}
	if err != nil {
		return c, false, err
	}
	if status == nil || !(*status == ProposalProgramReading || recheck && *status == ProposalProgramRead) || doc != in.DocumentID {
		return c, false, nil
	}
	attempt := 0
	if recheck {
		attempt = 1
	}
	if in.NotRead != "" {
		if _, err := tx.Exec(ctx, `UPDATE submissions SET program_read='not_read',program_not_read=$3,program_retry_after=NULL WHERE org_id=$1::uuid AND id::text=$2`, org, id, in.NotRead); err != nil {
			return c, false, err
		}
		return c, true, tx.Commit(ctx)
	}
	decided := map[string]bool{}
	rows, err := tx.Query(ctx, `SELECT COALESCE(parsed->>'package_stage_id','') FROM program_readings
WHERE org_id=$1::uuid AND program_id=$2::uuid AND document_id=$3::uuid AND source='proposal_r7' AND status<>'proposed'`, org, in.ProgramID, in.DocumentID)
	if err != nil {
		return c, false, err
	}
	for rows.Next() {
		var ps string
		if err := rows.Scan(&ps); err != nil {
			rows.Close()
			return c, false, err
		}
		decided[ps] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return c, false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM program_readings WHERE org_id=$1::uuid AND program_id=$2::uuid AND document_id=$3::uuid AND source='proposal_r7' AND status='proposed'`, org, in.ProgramID, in.DocumentID); err != nil {
		return c, false, err
	}
	var recs []readingRecord
	for _, r := range out.Readings {
		ps, _ := r.Parsed["package_stage_id"].(string)
		if decided[ps] || r.TargetID == "" && r.CatalogueStageID == "" {
			continue
		}
		parsed := map[string]any{"submission_id": id}
		for k, v := range r.Parsed {
			parsed[k] = v
		}
		raw, err := json.Marshal(parsed)
		if err != nil {
			return c, false, err
		}
		rec := readingRecord{ID: newID(), Field: r.Field, Value: clipText(r.Value, 500), Parsed: raw, Band: r.Band, DecidedBy: r.DecidedBy, Confidence: r.Confidence}
		if r.TargetID != "" {
			rec.Target = r.TargetID
		}
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
SELECT $1::uuid,r.id,$2::uuid,'proposal_r7',$3::uuid,NULLIF(r.target_activity_id,'')::uuid,r.catalogue_stage_id,r.field,r.value,r.parsed,r.band,r.decided_by,r.confidence,r.question_version,'proposed'
FROM jsonb_to_recordset($4::jsonb) AS r(id uuid,target_activity_id text,catalogue_stage_id text,field text,value text,parsed jsonb,band text,decided_by text,confidence double precision,question_version text)`,
			org, in.ProgramID, in.DocumentID, raw); err != nil {
			return c, false, err
		}
	}
	c.Readings = len(recs)
	grey, _ := json.Marshal(stringsOrEmpty(out.Grey))
	var retry *time.Time
	if len(out.Grey) > 0 {
		t := now.Add(30 * time.Second)
		retry = &t
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET program_read='read',program_not_read=NULL,program_grey=$3::jsonb,program_retry_after=$4,program_read_attempts=program_read_attempts+$5
WHERE org_id=$1::uuid AND id::text=$2`, org, id, grey, retry, attempt); err != nil {
		return c, false, err
	}
	return c, true, tx.Commit(ctx)
}

// proposalReadingText is the line Program shows for an awarded proposal's
// reading: "From the awarded proposal: Design development 6 weeks". The
// stage label and the stated duration are the package's and the
// proposal's own text; code only joins them.
func proposalReadingText(parsed json.RawMessage) string {
	var p struct {
		Stage  string `json:"stage_label"`
		Stated string `json:"stated"`
	}
	_ = json.Unmarshal(parsed, &p)
	return "From the awarded proposal: " + p.Stage + " " + p.Stated
}
