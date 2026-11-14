package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/tender"
)

// tenderPrefix starts every schedule key the tender tables own. A schedule
// sync leaves those rows alone.
const tenderPrefix = "tender:"

// Tender pass states of a submission (contract C4).
const (
	TenderPending = "pending"
	TenderReading = "reading"
	TenderRead    = "read"
	TenderFailed  = "failed"
)

// ensureTenderRow adds the "Not broken down" row to a package's schedule.
// It is a schedule item so that award, the report and export treat it as
// any other row.
func ensureTenderRow(ctx context.Context, tx pgx.Tx, org, project, pkg string) error {
	_, err := tx.Exec(ctx, `INSERT INTO schedule_items(org_id,project_id,package_id,key,kind,label,detail,base,ordinal,source)
SELECT $1::uuid,$2::uuid,$3::uuid,$4,'stage','Not broken down','What the firm did not break down, and anything its lines do not add up to',true,COALESCE(MAX(ordinal),-1)+1,'generic'
FROM schedule_items WHERE org_id=$1::uuid AND package_id=$3::uuid
ON CONFLICT (org_id,package_id,key) DO NOTHING`, org, project, pkg, tender.SystemRow)
	return err
}

// TenderInput is what one tender pass needs from the database.
type TenderInput struct {
	Submission
	SHA256   []byte
	Size     int64
	Rows     []tender.Row // the package's priced rows, in schedule order
	StatedEx *int64       // the total the existing rules found, ex GST
}

// tenderRows is the package's priced rows as the pass is offered them:
// base rows and option rows, never the system row, rates or criteria.
func (s *Store) tenderRows(ctx context.Context, tx pgx.Tx, org, project, pkg string) ([]tender.Row, error) {
	if err := s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return nil, err
	}
	items, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return nil, err
	}
	var rows []tender.Row
	for _, it := range items {
		// Neither the system row nor a head contract's old single row is
		// offered to the model: a line belongs on a row of the works.
		if it.Retired || !it.Price() || strings.HasPrefix(it.Key, tenderPrefix) || it.Key == "stage:lump_sum" {
			continue
		}
		rows = append(rows, tender.Row{Key: it.Key, Label: it.Label, Detail: it.Detail})
	}
	return rows, nil
}

// TenderInput gathers one submission's file and its package's rows.
func (s *Store) TenderInput(ctx context.Context, org, id string) (TenderInput, error) {
	var in TenderInput
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return in, err
	}
	defer tx.Rollback(ctx)
	sub, err := readSubmission(ctx, tx, org, id)
	if err != nil {
		return in, err
	}
	if sub.DocumentID == "" || sub.PackageID == "" {
		return in, ErrNotFound
	}
	in.Submission = sub
	if err := tx.QueryRow(ctx, `SELECT f.sha256, f.byte_size FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, org, sub.DocumentID).Scan(&in.SHA256, &in.Size); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return in, ErrNotFound
		}
		return in, err
	}
	if in.Rows, err = s.tenderRows(ctx, tx, org, sub.ProjectID, sub.PackageID); err != nil {
		return in, err
	}
	if sub.StatedTotal != nil {
		v := *sub.StatedTotal
		if sub.GSTBasis == "inc" {
			v = tender.ExGST(v)
		}
		in.StatedEx = &v
	}
	return in, tx.Commit(ctx)
}

// QueueTenderRead marks a submission for the pass when the org's switch is
// on, it has a package and a file, and it is a live proposal that has not
// been read. again is "Read again": it also queues a read or failed one.
func (s *Store) QueueTenderRead(ctx context.Context, org, id string, again bool) (bool, error) {
	states := []string{""}
	if again {
		states = []string{"", TenderRead, TenderFailed}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var project, pkg string
	err = tx.QueryRow(ctx, `UPDATE submissions sub SET tender_state='pending', tender_attempts=0, tender_retry_after=NULL, tender_error='', updated_at=now()
FROM orgs o WHERE o.id=sub.org_id AND o.model_enabled_at IS NOT NULL
AND sub.org_id=$1::uuid AND sub.id::text=$2 AND sub.package_id IS NOT NULL AND sub.document_id IS NOT NULL
AND sub.status IN ('received','needs_review') AND sub.tender_state = ANY($3::text[])
RETURNING sub.project_id::text, sub.package_id::text`, org, id, states).Scan(&project, &pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := touchTender(ctx, tx, s, org, project, pkg, id, TenderPending); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// touchTender bumps the package's tender version and tells open screens.
func touchTender(ctx context.Context, tx pgx.Tx, s *Store, org, project, pkg, sub, state string) error {
	if _, err := tx.Exec(ctx, `UPDATE packages SET tender_version = tender_version + 1 WHERE org_id=$1::uuid AND id::text=$2`, org, pkg); err != nil {
		return err
	}
	return procurementEvent(ctx, tx, s, org, project, sub, "tender_"+state)
}

// ClaimTenderRead moves pending or due-failed to reading and counts the
// attempt. Exactly one caller wins.
func (s *Store) ClaimTenderRead(ctx context.Context, org, id string, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE submissions SET tender_state='reading', tender_attempts=tender_attempts+1, tender_retry_after=NULL, updated_at=now()
WHERE org_id=$1::uuid AND id::text=$2 AND (tender_state='pending' OR (tender_state='failed' AND tender_retry_after IS NOT NULL AND tender_retry_after <= $3))`, org, id, now)
	return err == nil && tag.RowsAffected() == 1, err
}

// TenderAttempts is how many times the pass has been tried since it was queued.
func (s *Store) TenderAttempts(ctx context.Context, org, id string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT tender_attempts FROM submissions WHERE org_id=$1::uuid AND id::text=$2`, org, id).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return n, err
}

// FailTenderRead records why a pass failed and when to try again (nil:
// never). reason is plain words for the screen, never a library's error.
func (s *Store) FailTenderRead(ctx context.Context, org, id, reason string, retry *time.Time) error {
	return s.endTenderRead(ctx, org, id, TenderFailed, reason, retry)
}

// ResetTenderRead puts a claimed submission back to never asked: the pass
// turned out to be off.
func (s *Store) ResetTenderRead(ctx context.Context, org, id string) error {
	return s.endTenderRead(ctx, org, id, "", "", nil)
}

func (s *Store) endTenderRead(ctx context.Context, org, id, state, reason string, retry *time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var project, pkg string
	err = tx.QueryRow(ctx, `UPDATE submissions SET tender_state=$3, tender_error=$4, tender_retry_after=$5, updated_at=now()
WHERE org_id=$1::uuid AND id::text=$2 AND tender_state='reading' RETURNING project_id::text, COALESCE(package_id::text,'')`, org, id, state, clipText(reason, 200), retry).Scan(&project, &pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := touchTender(ctx, tx, s, org, project, pkg, id, "failed"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PendingTenderReads are passes to start after a restart. A pass that was
// in flight when the server stopped is queued again.
func (s *Store) PendingTenderReads(ctx context.Context) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `UPDATE submissions SET tender_state='pending' WHERE tender_state IN ('pending','reading') RETURNING org_id::text, id::text`)
}

// DueTenderRetries are failed passes whose retry time has come.
func (s *Store) DueTenderRetries(ctx context.Context, now time.Time) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `SELECT org_id::text, id::text FROM submissions WHERE tender_state='failed' AND tender_retry_after IS NOT NULL AND tender_retry_after <= $1`, now)
}
