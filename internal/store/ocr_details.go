package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/db"
)

func OCRDetailsActive(reason string) bool {
	return reason == "ocr_details_queued" || reason == "ocr_details_reading" || reason == "ocr_details_classifying"
}

// ReprocessOCRDetails keeps the document filed and queues one explicit pass.
// A repeated request during that pass succeeds without resetting its lease.
func (s *Store) ReprocessOCRDetails(ctx context.Context, orgID, documentID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if doc.Status != StatusFiled {
		return ErrNotFound
	}
	// Load reason from the locked row; filing's generated lock result is small.
	var reason, filename string
	if err := tx.QueryRow(ctx, `SELECT reason,filename FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID).Scan(&reason, &filename); err != nil {
		return err
	}
	if OCRDetailsActive(reason) {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") || (reason != "ocr_review" && !strings.HasPrefix(reason, "ocr_details_")) {
		return ErrNotFound
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, d := range rows {
		present[d.Field] = d.Value != "" || d.DecidedBy == "user"
	}
	missing := false
	for _, f := range []string{"kind", "discipline", "number", "title", "revision"} {
		missing = missing || !present[f]
	}
	if !missing {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET reason='ocr_details_queued' WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID); err != nil {
		return err
	}
	if err := enqueueKind(ctx, q, orgID, documentID, JobKindOCR); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE jobs SET status='queued',attempts=0,run_after=now(),locked_until=NULL,lease_token=NULL,last_error='' WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='ocr'`, orgID, documentID); err != nil {
		return err
	}
	payload, err := documentEventPayload(documentID, StatusFiled, "ocr_details_queued", rows)
	if err != nil {
		return err
	}
	if _, err := appendEvent(ctx, q, orgID, "ocr", documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// OCRDetailsProgress also ends failed passes without unfiling the document.
func (s *Store) OCRDetailsProgress(ctx context.Context, orgID, documentID, stage string) error {
	switch stage {
	case "reading", "classifying", "failed", "no_text", "limit", "unavailable":
	default:
		return errors.New("invalid detail recovery stage")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documents SET reason=$3 WHERE org_id=$1::uuid AND id=$2::uuid AND status='filed' AND reason IN ('ocr_details_queued','ocr_details_reading','ocr_details_classifying')`, orgID, documentID, "ocr_details_"+stage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	q := s.q.WithTx(tx)
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	payload, err := documentEventPayload(documentID, StatusFiled, "ocr_details_"+stage, rows)
	if err != nil {
		return err
	}
	if _, err := appendEvent(ctx, q, orgID, "ocr", documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CommitMissingDetails checks the current values under the same lock as user
// corrections. It cannot replace a filled field, a deliberate blank, or links.
func (s *Store) CommitMissingDetails(ctx context.Context, orgID, documentID string, writes []DecisionWrite) (FilingOutcome, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FilingOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return FilingOutcome{}, ErrNotFound
	}
	if err != nil {
		return FilingOutcome{}, err
	}
	var reason string
	if err := tx.QueryRow(ctx, `SELECT reason FROM documents WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID).Scan(&reason); err != nil {
		return FilingOutcome{}, err
	}
	if doc.Status != StatusFiled || !OCRDetailsActive(reason) {
		return FilingOutcome{}, ErrNotFound
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	protected := map[string]bool{}
	for _, d := range rows {
		protected[d.Field] = d.Value != "" || d.DecidedBy == "user"
	}
	changed := false
	for _, d := range writes {
		if protected[d.Field] || d.Field == "supersedes" || d.Value == "" {
			continue
		}
		conf := 0.0
		if d.Confidence != nil {
			conf = *d.Confidence
		}
		n, err := q.UpsertFilingDecision(ctx, db.UpsertFilingDecisionParams{OrgID: orgID, ID: newID(), DocumentID: documentID, Field: d.Field, Value: strPtr(d.Value), Band: d.Band, DecidedBy: d.DecidedBy, QuestionVersion: strPtr(d.QuestionVersion), ConfidenceSet: d.Confidence != nil, Confidence: conf, ExpectedVersion: d.ExpectedVersion})
		if err != nil {
			return FilingOutcome{}, err
		}
		changed = changed || n > 0
	}
	rows, err = q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	number, revision := identityOf(rows)
	reason = "ocr_details_unchanged"
	if changed {
		reason = "ocr_details_review"
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET document_number=NULLIF($3,''),revision=NULLIF($4,''),reason=$5 WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID, number, revision, reason); err != nil {
		return FilingOutcome{}, err
	}
	payload, err := documentEventPayload(documentID, StatusFiled, reason, rows)
	if err != nil {
		return FilingOutcome{}, err
	}
	if _, err := appendEvent(ctx, q, orgID, EventFiling, documentID, payload); err != nil {
		return FilingOutcome{}, err
	}
	out, err := readOutcome(ctx, q, orgID, documentID)
	if err != nil {
		return FilingOutcome{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FilingOutcome{}, err
	}
	return out, nil
}
