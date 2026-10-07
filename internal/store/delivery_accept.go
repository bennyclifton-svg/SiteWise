package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
	"sitewise/internal/works"
)

// Assignments are explicit choices. A proposal cannot identify an appointed
// package, owner or released hold point from its label.
type DeliveryAcceptance struct {
	PackageID  string `json:"package_id,omitempty"`
	WorkItemID string `json:"work_item_id,omitempty"`
	StageID    string `json:"stage_id,omitempty"`
}

func readDeliveryRecord(ctx context.Context, tx pgx.Tx, org, project, id string) (delivery.Item, error) {
	var item delivery.Item
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(i) FROM project_delivery_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	err = json.Unmarshal(raw, &item)
	return item, err
}

func (s *Store) AcceptDeliveryProposal(ctx context.Context, org, project, key, actor, fingerprint string, choice DeliveryAcceptance) (delivery.Item, error) {
	var item delivery.Item
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return item, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return item, err
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return item, err
	}
	if d, ok := decisions[key]; ok && d.Decision == "accepted" {
		if d.CreatedRecordType != "delivery_item" {
			return item, ErrProposalUnavailable
		}
		var saved struct {
			Assignment DeliveryAcceptance `json:"assignment"`
		}
		if err := json.Unmarshal(d.InputsSnapshot, &saved); err != nil {
			return item, err
		}
		if d.InputsFingerprint != fingerprint || saved.Assignment != choice {
			return item, ErrVersionConflict
		}
		item, err = readDeliveryRecord(ctx, tx, org, project, d.CreatedRecordID)
		if err != nil {
			return item, err
		}
		return item, tx.Commit(ctx)
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM ranked_proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	var proposal works.Proposal
	if err := json.Unmarshal(raw, &proposal); err != nil {
		return item, err
	}
	if proposal.InputsFingerprint != fingerprint {
		return item, ErrVersionConflict
	}
	kind := "approval"
	if proposal.Kind == "hold_point" {
		kind = "milestone"
	} else if proposal.Kind != "approval" {
		return item, ErrProposalUnavailable
	}
	item = delivery.Item{Content: delivery.Content{Kind: kind, Title: proposal.Label, Status: delivery.DefaultStatus(kind), Details: json.RawMessage(`{}`), PackageID: choice.PackageID, WorkItemID: choice.WorkItemID, StageID: choice.StageID}, ID: newID(), ProjectID: project, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "requirement", SourceProposalKey: key, Version: 1}
	if err := delivery.Validate(item.Content); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
	}
	if err := s.deliveryTargets(ctx, tx, org, project, item.Content); err != nil {
		return item, err
	}
	item.Provenance, _ = json.Marshal(map[string]any{"actor": actor, "proposal": proposal})
	old, err := undoneProposalDelivery(ctx, tx, org, project, key)
	if err != nil {
		return item, err
	}
	if old.ID != "" {
		item.ID, item.Version = old.ID, old.Version+1
		err = tx.QueryRow(ctx, `UPDATE project_delivery_items SET kind=$4,title=$5,status=$6,package_id=NULLIF($7,'')::uuid,work_item_id=NULLIF($8,'')::uuid,stage_id=NULLIF($9,'')::uuid,provenance=$10::jsonb||jsonb_build_object('at',now(),'initial_status',$6::text),retired_at=NULL,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid RETURNING to_jsonb(project_delivery_items)`, org, project, item.ID, item.Kind, item.Title, item.Status, item.PackageID, item.WorkItemID, item.StageID, item.Provenance).Scan(&raw)
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,status,package_id,work_item_id,stage_id,origin,review_status,meaning,provenance,source_proposal_key) VALUES($1::uuid,$3::uuid,$2::uuid,$4,$5,$6,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,'calculation','accepted_for_planning','requirement',$10::jsonb||jsonb_build_object('at',now(),'initial_status',$6::text),$11) RETURNING to_jsonb(project_delivery_items)`, org, project, item.ID, item.Kind, item.Title, item.Status, item.PackageID, item.WorkItemID, item.StageID, item.Provenance, key).Scan(&raw)
	}
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	snapshot, err := json.Marshal(struct {
		works.Proposal
		Assignment DeliveryAcceptance `json:"assignment"`
	}{proposal, choice})
	if err != nil {
		return item, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id,created_record_version,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,(SELECT work_item_id FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$3::uuid AND proposal_key=$4 ORDER BY work_item_id LIMIT 1),'accepted',$6,$7::uuid,'delivery_item',$8::uuid,$10,$9::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET trigger_work_item_id=EXCLUDED.trigger_work_item_id,decision='accepted',inputs_fingerprint=EXCLUDED.inputs_fingerprint,actor=EXCLUDED.actor,decided_at=now(),created_record_type='delivery_item',created_record_id=EXCLUDED.created_record_id,created_record_version=EXCLUDED.created_record_version,inputs_snapshot=EXCLUDED.inputs_snapshot,rationale='',version=proposal_decisions.version+1,undone_at=NULL`, org, newID(), project, key, proposal.RecordID, fingerprint, actor, item.ID, snapshot, item.Version)
	if err != nil {
		return item, err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='accepted',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return item, err
	}
	if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
		return item, err
	}
	if err := workEvent(ctx, tx, s, org, project); err != nil {
		return item, err
	}
	return item, s.finishDeliveryWrite(ctx, tx, org, project)
}

func undoneProposalDelivery(ctx context.Context, tx pgx.Tx, org, project, key string) (delivery.Item, error) {
	var item delivery.Item
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND source_proposal_key=$3`, org, project, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, nil
	}
	if err != nil {
		return item, err
	}
	item, err = readDeliveryRecord(ctx, tx, org, project, id)
	if err != nil {
		return item, err
	}
	var eligible bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM proposal_decisions d CROSS JOIN LATERAL jsonb_array_elements(d.undo_history) h WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.proposal_key=$3 AND h->>'decision'='accepted' AND h->>'created_record_type'='delivery_item' AND h->>'created_record_id'=$4 AND (h->>'created_record_version')::bigint+1=$5)`, org, project, key, id, item.Version).Scan(&eligible)
	if err != nil {
		return item, err
	}
	if item.RetiredAt == nil || !eligible {
		return item, ErrProposalUndoBlocked
	}
	return item, nil
}

func undoCreatedDelivery(ctx context.Context, tx pgx.Tx, org, project, key string, d ProposalDecisionView) error {
	var used bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM delivery_dependencies WHERE org_id=$1::uuid AND project_id=$2::uuid AND (predecessor_id=$3::uuid OR successor_id=$3::uuid))
 OR EXISTS(SELECT 1 FROM proposal_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key<>$4 AND created_delivery_item_id=$3::uuid)
 OR EXISTS(SELECT 1 FROM report_versions rv CROSS JOIN LATERAL jsonb_array_elements(rv.sections) section CROSS JOIN LATERAL jsonb_array_elements(section->'blocks') block WHERE rv.org_id=$1::uuid AND rv.project_id=$2::uuid AND (block->>'id'='delivery:'||$3::text OR block->'basis'->>'delivery_item_id'=$3::text))`, org, project, d.CreatedRecordID, key).Scan(&used)
	if err != nil {
		return err
	}
	if used {
		return ErrProposalUndoBlocked
	}
	tag, err := tx.Exec(ctx, `UPDATE project_delivery_items SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND source_proposal_key=$4 AND version=$5 AND retired_at IS NULL AND review_status='accepted_for_planning'`, org, project, d.CreatedRecordID, key, d.CreatedRecordVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrProposalUndoBlocked
	}
	return nil
}
