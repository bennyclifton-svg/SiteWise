package store

import (
	"context"
	"encoding/json"

	"sitewise/internal/proposals"
)

func quotedReview(ctx context.Context, q querier, org, sub string) (*proposals.QuoteReview, error) {
	var raw []byte
	if err := q.QueryRow(ctx, `SELECT quote_review FROM submissions WHERE org_id=$1::uuid AND id=$2::uuid`, org, sub).Scan(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var review proposals.QuoteReview
	if err := json.Unmarshal(raw, &review); err != nil {
		return nil, err
	}
	return &review, nil
}
