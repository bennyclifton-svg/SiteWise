package store

import (
	"context"
	"errors"
	"strings"
)

const OCRTrialVersion = "ocr-tesseract-150-v1"

// CommitOCRTrial atomically promotes only a textless filing to review. A crash
// leaves it not-filed; a concurrent ordinary filing or user correction wins.
func (s *Store) CommitOCRTrial(ctx context.Context, orgID, documentID string, decisions []DecisionWrite) (FilingOutcome, error) {
	if len(decisions) == 0 {
		return FilingOutcome{}, errors.New("empty OCR trial")
	}
	for _, d := range decisions {
		if d.Field == "supersedes" || (d.Band != "amber" && d.Band != "blank") ||
			(d.DecidedBy != "rule" && d.DecidedBy != "jev") || !strings.HasPrefix(d.QuestionVersion, OCRTrialVersion+"+") {
			return FilingOutcome{}, errors.New("OCR trial decisions must retain review provenance")
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FilingOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documents SET status='pending', reason='' WHERE org_id=$1::uuid AND id=$2::uuid AND status='not_filed' AND reason='no_text_layer'`, orgID, documentID)
	if err != nil {
		return FilingOutcome{}, err
	}
	if tag.RowsAffected() != 1 {
		return FilingOutcome{}, ErrNotFound
	}
	out, err := commitFilingTx(ctx, tx, orgID, documentID, CommitFiling{Decisions: decisions})
	if err != nil {
		return FilingOutcome{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return FilingOutcome{}, err
	}
	return out, nil
}
