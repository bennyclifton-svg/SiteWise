package store

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func undoCreatedScope(ctx context.Context, tx pgx.Tx, org, project, key string, d ProposalDecisionView) error {
	var used bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM proposal_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key<>$3 AND created_scope_item_id=$4::uuid)
 OR EXISTS(SELECT 1 FROM report_versions rv CROSS JOIN LATERAL jsonb_array_elements(rv.sections) section CROSS JOIN LATERAL jsonb_array_elements(section->'blocks') block WHERE rv.org_id=$1::uuid AND rv.project_id=$2::uuid AND block->>'id'='scope:'||$4::text)`, org, project, key, d.CreatedRecordID).Scan(&used); err != nil {
		return err
	}
	if used {
		return ErrProposalUndoBlocked
	}
	// Future cost/delivery references must be added with their migrations.
	tag, err := tx.Exec(ctx, `UPDATE package_scope_items SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND version=$4 AND source_proposal_key=$5 AND retired_at IS NULL AND origin='calculation' AND review_status='accepted_for_planning'`, org, project, d.CreatedRecordID, d.CreatedRecordVersion, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrProposalUndoBlocked
	}
	return nil
}
