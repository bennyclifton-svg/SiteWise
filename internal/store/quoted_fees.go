package store

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"sitewise/internal/invoices"
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

var quoteID = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,100}$`)

// awardStages is the source's stage breakdown when it can be awarded as
// separate lines: the rule-read breakdown components, each with an ex-GST
// amount, adding to the PM's confirmed fee to the cent. Anything else is
// nil, so a breakdown that misses a stage or counts the total twice is
// never offered. Coverage and Jev suggestions play no part.
func awardStages(r *proposals.QuoteReview, confirmed int64) []proposals.QuotedFee {
	if r == nil {
		return nil
	}
	var out []proposals.QuotedFee
	var sum int64
	seen := map[string]bool{}
	for _, f := range r.Fees {
		if f.Role != proposals.RoleComponent || f.Amount == nil {
			continue
		}
		c, ok, err := invoices.ParseCents(*f.Amount)
		if err != nil || !ok || c <= 0 || seen[f.ID] || !quoteID.MatchString(f.ID) || strings.TrimSpace(f.Label) == "" {
			return nil
		}
		seen[f.ID] = true
		sum += c
		out = append(out, f)
	}
	if len(out) < 2 || sum != confirmed {
		return nil
	}
	return out
}
