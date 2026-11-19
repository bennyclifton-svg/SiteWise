package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// The register's post-filing job (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 4; index agreement 2). A filed document is matched to the project's
// Open RFIs: by Go's rule when it cites the RFI's number or the sender's
// ref, and for correspondence by one Jev fan-out on the RFIs the rule left.
// Plan 6 adds its evidence step to this same job kind and queue point.

// JobKindRegister is the post-filing reading for the register.
const JobKindRegister = "register"

// queueRegister queues the job beside queueVisual, inside the filing
// commit, and only when the project has an Open RFI or an open consent
// condition that may need evidence: a project with neither files exactly
// as before (index agreement 2).
func queueRegister(ctx context.Context, tx pgx.Tx, orgID, id string) error {
	_, err := tx.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status)
 SELECT d.org_id,$2::uuid,d.id,'register','queued' FROM documents d
 WHERE d.org_id=$1::uuid AND d.id=$3::uuid AND d.status='filed'
 AND EXISTS(SELECT 1 FROM project_delivery_items i WHERE i.org_id=d.org_id AND i.project_id=d.project_id AND i.retired_at IS NULL
  AND (i.kind='rfi' AND i.status='open' OR i.kind='condition' AND i.status='open' AND COALESCE(i.details->>'evidence_needed','')<>'false'))
 ON CONFLICT (org_id,document_id,kind) DO UPDATE SET status='queued',attempts=0,run_after=now(),locked_until=NULL,lease_token=NULL,last_error='' WHERE jobs.status IN ('done','failed')`, orgID, newID(), id)
	return err
}

// RegisterDocument is the filed document the job reads.
type RegisterDocument struct {
	ProjectID, Title, Kind, Discipline string
}

// RegisterRFI is one Open RFI the document may answer.
type RegisterRFI struct {
	ID, Subject, Question, Ref, OwnerText, Discipline string
	Number                                            int
	// Threaded: the document is the body of a message received in the
	// thread of the message this RFI went out on (correspondence plan).
	Threaded bool
}

// RegisterReading is one possible answer the job found, or (Field
// "evidence") one possible piece of evidence for a consent condition.
type RegisterReading struct {
	ItemID, DecidedBy, Band, Basis, QuestionVersion string
	Probability                                     *float64
	Field                                           string // "answers" (the default) or "evidence"
}

// RegisterCondition is one open consent condition a document may evidence.
type RegisterCondition struct {
	ID, Number, ConsentRef, Heading, Text, Stage, Discipline string
}

// RegisterConditions are the project's open conditions whose evidence is
// not known to be unneeded, leaving out ones the document already
// evidences or the PM already decided for it.
func (s *Store) RegisterConditions(ctx context.Context, org, project, document string) ([]RegisterCondition, error) {
	rows, err := s.pool.Query(ctx, `SELECT i.id::text, COALESCE(i.details->>'number',''), COALESCE(i.details->>'consent_ref',''), COALESCE(i.details->>'heading',''),
 left(COALESCE(i.details->>'text',''),400), COALESCE(i.details->>'stage',''), COALESCE(i.details->>'discipline','')
 FROM project_delivery_items i
 WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.kind='condition' AND i.status='open' AND i.retired_at IS NULL
 AND COALESCE(i.details->>'evidence_needed','')<>'false'
 AND NOT COALESCE(i.details->'evidence','[]'::jsonb) @> jsonb_build_array(jsonb_build_object('document_id',$3::text))
 AND COALESCE(i.details->'source'->>'document_id','')<>$3::text
 AND NOT EXISTS(SELECT 1 FROM delivery_readings r WHERE r.org_id=i.org_id AND r.item_id=i.id AND r.document_id=$3::uuid AND r.status<>'proposed')
 ORDER BY i.details->>'number', i.id`, org, project, document)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegisterCondition
	for rows.Next() {
		var c RegisterCondition
		if err := rows.Scan(&c.ID, &c.Number, &c.ConsentRef, &c.Heading, &c.Text, &c.Stage, &c.Discipline); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RegisterCandidates reads the document and the Open RFIs it may answer,
// newest first. An RFI is left out when the document is already one of its
// sources or answers (the filed document is the RFI itself), or when the
// PM has already accepted or dismissed this document for it.
func (s *Store) RegisterCandidates(ctx context.Context, org, document string) (RegisterDocument, []RegisterRFI, error) {
	var doc RegisterDocument
	err := s.pool.QueryRow(ctx, `SELECT d.project_id::text,
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='title' LIMIT 1), d.filename),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='kind' LIMIT 1), ''),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='discipline' LIMIT 1), '')
 FROM documents d WHERE d.org_id=$1::uuid AND d.id=$2::uuid AND d.status='filed'`, org, document).Scan(&doc.ProjectID, &doc.Title, &doc.Kind, &doc.Discipline)
	if errors.Is(err, pgx.ErrNoRows) {
		return doc, nil, ErrNotFound
	}
	if err != nil {
		return doc, nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT i.id::text, i.number, i.title, COALESCE(i.details->>'question',''), COALESCE(i.details->>'ref',''), i.owner_text, COALESCE(p.discipline_id,''),
 EXISTS(SELECT 1 FROM mail_parts mp JOIN mail_messages m ON m.org_id=mp.org_id AND m.id=mp.message_id
   JOIN mail_messages sent ON sent.org_id=m.org_id AND sent.project_id=m.project_id AND sent.thread_id=m.thread_id
  WHERE mp.org_id=i.org_id AND mp.document_id=$3::uuid AND mp.n=0 AND m.direction='in'
   AND sent.id::text=COALESCE(i.details->>'sent_message_id',''))
 FROM project_delivery_items i LEFT JOIN packages p ON p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id
 WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.kind='rfi' AND i.status='open' AND i.retired_at IS NULL
 AND NOT COALESCE(i.provenance->'sources','[]'::jsonb) @> jsonb_build_array(jsonb_build_object('document_id',$3::text))
 AND NOT COALESCE(i.details->'answer_documents','[]'::jsonb) @> to_jsonb($3::text)
 AND NOT EXISTS(SELECT 1 FROM delivery_readings r WHERE r.org_id=i.org_id AND r.item_id=i.id AND r.document_id=$3::uuid AND r.status<>'proposed')
 ORDER BY i.number DESC`, org, doc.ProjectID, document)
	if err != nil {
		return doc, nil, err
	}
	defer rows.Close()
	var out []RegisterRFI
	for rows.Next() {
		var r RegisterRFI
		if err := rows.Scan(&r.ID, &r.Number, &r.Subject, &r.Question, &r.Ref, &r.OwnerText, &r.Discipline, &r.Threaded); err != nil {
			return doc, nil, err
		}
		out = append(out, r)
	}
	return doc, out, rows.Err()
}

// SaveRegisterReadings stores the job's readings while it still holds its
// lease. A rule reading is never replaced by a Jev one, and a reading the
// PM accepted or dismissed is never reopened.
func (s *Store) SaveRegisterReadings(ctx context.Context, job ClaimedJob, readings []RegisterReading) error {
	if len(readings) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var project string
	err = tx.QueryRow(ctx, `SELECT d.project_id::text FROM documents d WHERE d.org_id=$1::uuid AND d.id=$2::uuid
 AND EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND id=$3::uuid AND lease_token=$4::uuid AND status='leased' AND locked_until>now())`, job.OrgID, job.DocumentID, job.ID, job.Token).Scan(&project)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := lockProject(ctx, tx, job.OrgID, project); err != nil {
		return err
	}
	for _, r := range readings {
		switch {
		case r.DecidedBy != "rule" && r.DecidedBy != "jev":
			return fmt.Errorf("invalid reading source %q", r.DecidedBy)
		case r.Band != "amber" && r.Band != "blank" && r.Band != "grey":
			return fmt.Errorf("invalid reading band %q", r.Band)
		}
		field, kind := "answers", "rfi"
		if r.Field == "evidence" {
			field, kind = "evidence", "condition"
		}
		// The item must be an RFI (or, for evidence, a condition) of this
		// document's project: the id came from the candidates, but the write
		// checks it again.
		_, err := tx.Exec(ctx, `INSERT INTO delivery_readings(org_id,id,project_id,item_id,document_id,field,decided_by,probability,band,basis,question_version)
 SELECT i.org_id,$3::uuid,i.project_id,i.id,$4::uuid,$11,$5,$6,$7,$8,$9 FROM project_delivery_items i
 WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.id=$10::uuid AND i.kind=$12
 ON CONFLICT (org_id,item_id,document_id,field) DO UPDATE SET decided_by=EXCLUDED.decided_by,probability=EXCLUDED.probability,band=EXCLUDED.band,basis=EXCLUDED.basis,question_version=EXCLUDED.question_version
 WHERE delivery_readings.status='proposed' AND NOT (delivery_readings.decided_by='rule' AND EXCLUDED.decided_by='jev')`,
			job.OrgID, project, newID(), job.DocumentID, r.DecidedBy, r.Probability, r.Band, truncateRunes(r.Basis, 200), r.QuestionVersion, r.ItemID, field, kind)
		if err != nil {
			return err
		}
	}
	if err := bumpRevisionLocked(ctx, tx, job.OrgID, project, "delivery"); err != nil {
		return err
	}
	revisions, err := readRevisions(ctx, tx, job.OrgID, project)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"project_id": project, "revision": revisions.Delivery})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), job.OrgID, "delivery", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// QueueRegister queues the register's reply match for a filed document,
// as filing does: only when its project has an Open RFI.
func (s *Store) QueueRegister(ctx context.Context, org, document string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := queueRegister(ctx, tx, org, document); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
