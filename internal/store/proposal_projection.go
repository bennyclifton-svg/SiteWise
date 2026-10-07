package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

// The embedded typed payload uses the same JSON names as the persisted model.
// Omitted optional strings become SQL NULL through jsonb_to_recordset.
type proposalProjectionRow struct {
	works.Proposal
	Hash string `json:"projection_hash"`
}

type proposalProjectionStamp struct {
	hash          string
	state         string
	inputsChanged bool
}

type proposalPersistenceInput struct {
	decisions map[string]ProposalDecisionView
	previous  map[string]proposalProjectionStamp
}

func readProposalPersistenceInput(ctx context.Context, tx pgx.Tx, org, project string) (proposalPersistenceInput, error) {
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return proposalPersistenceInput{}, err
	}
	previous, err := readProposalProjectionStamps(ctx, tx, org, project)
	if err != nil {
		return proposalPersistenceInput{}, err
	}
	return proposalPersistenceInput{decisions, previous}, nil
}

type proposalTriggerRow struct {
	Key        string `json:"proposal_key"`
	WorkItemID string `json:"work_item_id"`
}

func proposalTriggerDelta(ctx context.Context, tx pgx.Tx, org, project string, keys []string, desired []proposalTriggerRow) ([]proposalTriggerRow, []proposalTriggerRow, error) {
	wanted := map[proposalTriggerRow]bool{}
	for _, r := range desired {
		wanted[r] = true
	}
	rows, err := tx.Query(ctx, `SELECT proposal_key,work_item_id::text FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key=ANY($3::text[])`, org, project, keys)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	remove := []proposalTriggerRow{}
	for rows.Next() {
		var r proposalTriggerRow
		if err := rows.Scan(&r.Key, &r.WorkItemID); err != nil {
			return nil, nil, err
		}
		if !wanted[r] {
			remove = append(remove, r)
		}
		delete(wanted, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	add := []proposalTriggerRow{}
	for _, r := range desired {
		if wanted[r] {
			add = append(add, r)
		}
	}
	return add, remove, nil
}

func readProposalProjectionStamps(ctx context.Context, tx pgx.Tx, org, project string) (map[string]proposalProjectionStamp, error) {
	rows, err := tx.Query(ctx, `SELECT key,projection_hash,state,inputs_changed FROM proposals WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]proposalProjectionStamp{}
	for rows.Next() {
		var key string
		var stamp proposalProjectionStamp
		if err := rows.Scan(&key, &stamp.hash, &stamp.state, &stamp.inputsChanged); err != nil {
			return nil, err
		}
		out[key] = stamp
	}
	return out, rows.Err()
}

// Every projected value, including trigger identities and decision state, is
// represented. Rank is computed by the scoped ranked_proposals view on reads.
func proposalProjectionHash(site string, p works.Proposal) (string, error) {
	p.Rank = 0
	raw, err := json.Marshal(struct {
		Site     string
		Proposal works.Proposal
	}{site, p})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}
