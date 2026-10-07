package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

func packageStagesUnchanged(p procurement.Package, snapshot json.RawMessage) bool {
	var saved struct {
		CreatedStages []procurement.Stage `json:"created_stages"`
	}
	if json.Unmarshal(snapshot, &saved) != nil || saved.CreatedStages == nil || len(saved.CreatedStages) != len(p.Stages) {
		return false
	}
	versions := map[string]int64{}
	for _, stage := range saved.CreatedStages {
		versions[stage.ID] = stage.Version
	}
	for _, stage := range p.Stages {
		if versions[stage.ID] != stage.Version || stage.Origin != "calculation" {
			return false
		}
	}
	return true
}

func undoCreatedPackage(ctx context.Context, tx pgx.Tx, org, project, key string, d ProposalDecisionView) error {
	p, err := readPackage(ctx, tx, org, project, d.CreatedRecordID)
	if errors.Is(err, ErrNotFound) {
		return ErrProposalUndoBlocked
	}
	if err != nil {
		return err
	}
	if p.Version != d.CreatedRecordVersion || p.SourceProposalKey != key || !packageStagesUnchanged(p, d.InputsSnapshot) {
		return ErrProposalUndoBlocked
	}
	var used bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM proposal_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key<>$3 AND created_package_id=$4::uuid) OR EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$4::uuid) OR EXISTS(SELECT 1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$4::uuid) OR EXISTS(SELECT 1 FROM reports WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$4::uuid) OR EXISTS(SELECT 1 FROM cost_item_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$4::uuid)`, org, project, key, p.ID).Scan(&used); err != nil {
		return err
	}
	if used {
		return ErrProposalUndoBlocked
	}
	_, err = tx.Exec(ctx, `UPDATE packages SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, p.ID)
	return err
}

func undoneProposalPackage(ctx context.Context, tx pgx.Tx, org, project, key string) (procurement.Package, error) {
	var p procurement.Package
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND source_proposal_key=$3`, org, project, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	p, err = readPackageRecord(ctx, tx, org, project, id, true)
	if err != nil {
		return p, err
	}
	if p.RetiredAt == nil {
		return p, ErrProposalUndoBlocked
	}
	var snapshot []byte
	err = tx.QueryRow(ctx, `SELECT h->'inputs_snapshot' FROM proposal_decisions d CROSS JOIN LATERAL jsonb_array_elements(d.undo_history) h WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.proposal_key=$3 AND h->>'created_record_type'='package' AND h->>'created_record_id'=$4 AND (h->>'created_record_version')::bigint+1=$5 ORDER BY (h->>'version')::bigint DESC LIMIT 1`, org, project, key, id, p.Version).Scan(&snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrProposalUndoBlocked
	}
	if err != nil {
		return p, err
	}
	if !packageStagesUnchanged(p, snapshot) {
		return p, ErrProposalUndoBlocked
	}
	return p, nil
}
