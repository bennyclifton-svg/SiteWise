package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
)

// Consent conditions in the register (docs/plans/2026-11-17-consent-conditions.md).
// The background read writes proposed rows (UpsertConditions); the PM's
// edits accept them; evidence is linked here. Nothing on these paths calls
// Jev or the model.

// addEvidence links a filed document of this project to a condition as its
// evidence. An open condition becomes evidence filed; nothing here ever
// makes one satisfied (owner decision 6).
func addEvidence(ctx context.Context, tx pgx.Tx, org, project, actor string, c *delivery.Content, document string) error {
	if c.Kind != "condition" {
		return fmt.Errorf("%w: only a consent condition takes evidence", ErrInvalidDelivery)
	}
	if err := projectDocument(ctx, tx, org, project, document); err != nil {
		return err
	}
	d := delivery.Condition(*c)
	if !slices.ContainsFunc(d.Evidence, func(e delivery.ConditionEvidence) bool { return e.DocumentID == document }) {
		d.Evidence = append(d.Evidence, delivery.ConditionEvidence{DocumentID: document, LinkedBy: actor,
			LinkedAt: time.Now().In(invoices.Sydney).Format("2006-01-02")})
	}
	if len(d.Evidence) > 50 {
		return fmt.Errorf("%w: a condition keeps at most 50 evidence documents", ErrInvalidDelivery)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	c.Details = raw
	if c.Status == "open" {
		c.Status = "evidence_filed"
	}
	return nil
}
