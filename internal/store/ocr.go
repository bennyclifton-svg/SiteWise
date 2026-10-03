package store

import (
	"context"
	"encoding/json"
	"errors"
	"sitewise/internal/db"
)

const JobKindOCR = "ocr"

// QueueOCR atomically hands a textless upload to the durable OCR queue.
// Finishing intake keeps restart recovery from submitting it a second time.
func (s *Store) QueueOCR(ctx context.Context, orgID, documentID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documents SET reason='ocr_queued' WHERE org_id=$1::uuid AND id=$2::uuid AND status='pending' AND reason=''`, orgID, documentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	q := s.q.WithTx(tx)
	if err := enqueueKind(ctx, q, orgID, documentID, JobKindOCR); err != nil {
		return err
	}
	if _, err := q.FinishIntakeJob(ctx, db.FinishIntakeJobParams{OrgID: orgID, DocumentID: documentID}); err != nil {
		return err
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	payload, _ := documentEventPayload(documentID, StatusPending, "ocr_queued", rows)
	if _, err := appendEvent(ctx, q, orgID, "ocr", documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) OCRProgress(ctx context.Context, orgID, documentID, stage string) error {
	if stage != "ocr_reading" && stage != "ocr_classifying" {
		return errors.New("invalid OCR stage")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documents SET reason=$3 WHERE org_id=$1::uuid AND id=$2::uuid AND status='pending' AND reason IN ('ocr_queued','ocr_reading','ocr_classifying')`, orgID, documentID, stage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	// Omit fields: a stage event must not erase corrections in the browser.
	body, _ := json.Marshal(map[string]string{"document_id": documentID, "status": StatusPending, "reason": stage})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), orgID, "ocr", documentID, string(body)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RetryOCR is explicit recovery for an old textless filing or a failed pass.
// Repeated clicks cannot reset a pending document or overwrite its decisions.
func (s *Store) RetryOCR(ctx context.Context, orgID, documentID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documents SET status='pending', reason='ocr_queued'
 WHERE org_id=$1::uuid AND id=$2::uuid AND status='not_filed'
 AND reason IN ('no_text_layer','ocr_failed','ocr_no_text','ocr_limit','ocr_unavailable')`, orgID, documentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	q := s.q.WithTx(tx)
	if err := enqueueKind(ctx, q, orgID, documentID, JobKindOCR); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE jobs SET status='queued', attempts=0, run_after=now(), locked_until=NULL, lease_token=NULL, last_error=''
 WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='ocr'`, orgID, documentID); err != nil {
		return err
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	payload, _ := documentEventPayload(documentID, StatusPending, "ocr_queued", rows)
	if _, err := appendEvent(ctx, q, orgID, "ocr", documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
