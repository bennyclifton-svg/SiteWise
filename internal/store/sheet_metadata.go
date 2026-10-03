package store

import (
	"context"
	"errors"
	"sitewise/internal/db"
)

// RefreshSheetMetadata reapplies parser results to an existing sheet without
// recreating files, changing classification or overwriting user corrections.
// ExpectedVersion prevents overwriting a decision changed during extraction.
func (s *Store) RefreshSheetMetadata(ctx context.Context, orgID, documentID string, decisions []DecisionWrite) error {
	for _, d := range decisions {
		if (d.Field != "number" && d.Field != "revision" && d.Field != "title" && d.Field != "date") || d.DecidedBy != "rule" || d.Band != "green" || d.Value == "" {
			return ErrMetadata
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: documentID})
	if err != nil {
		return err
	}
	var sheet bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drawing_sheets WHERE org_id=$1::uuid AND document_id=$2::uuid)`, orgID, documentID).Scan(&sheet); err != nil {
		return err
	}
	if !sheet || doc.Status != "filed" {
		return errors.New("document is not a filed drawing-set sheet")
	}
	for _, d := range decisions {
		_, err = q.UpsertFilingDecision(ctx, db.UpsertFilingDecisionParams{OrgID: orgID, ID: newID(), DocumentID: documentID, Field: d.Field, Value: strPtr(d.Value), Band: d.Band, DecidedBy: d.DecidedBy, QuestionVersion: strPtr(d.QuestionVersion), ExpectedVersion: d.ExpectedVersion})
		if err != nil {
			return err
		}
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	number, revision := identityOf(rows)
	if _, err = tx.Exec(ctx, `UPDATE documents SET document_number=$3,revision=$4 WHERE org_id=$1::uuid AND id=$2::uuid AND status='filed'`, orgID, documentID, strPtr(number), strPtr(revision)); err != nil {
		return err
	}
	payload, err := filingEventPayload(documentID, rows)
	if err != nil {
		return err
	}
	if _, err = appendEvent(ctx, q, orgID, EventFiling, documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
