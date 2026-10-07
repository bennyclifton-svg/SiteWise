package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrProposalUndoBlocked = errors.New("created record has been edited or referenced")

func (s *Store) UndoProposalDecision(ctx context.Context, org, project, key, actor string, version int64) error {
	if version < 1 {
		return ErrInvalidProposalDecision
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE org_id=$1::uuid AND id=$2::uuid)`, org, actor).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	d, ok := decisions[key]
	if !ok {
		return ErrNotFound
	}
	if d.Version != version {
		return ErrVersionConflict
	}
	if d.Decision == "accepted" {
		if d.CreatedRecordType == "delivery_item" {
			if err := undoCreatedDelivery(ctx, tx, org, project, key, d); err != nil {
				return err
			}
		} else if d.CreatedRecordType == "package_scope_item" {
			if err := undoCreatedScope(ctx, tx, org, project, key, d); err != nil {
				return err
			}
		} else if d.CreatedRecordType == "package" {
			if err := undoCreatedPackage(ctx, tx, org, project, key, d); err != nil {
				return err
			}
		} else if d.CreatedRecordType != "work_item" {
			return ErrProposalUnavailable
		} else {
			// Derived projections can be rebuilt; an authoritative child or another
			// person's decision means the created scope has been used.
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND parent_id=$3::uuid)
 OR EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND work_item_id=$3::uuid)
 OR EXISTS(SELECT 1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND work_item_id=$3::uuid)
 OR EXISTS(SELECT 1 FROM report_versions rv CROSS JOIN LATERAL jsonb_array_elements(rv.sections) section CROSS JOIN LATERAL jsonb_array_elements(section->'blocks') block WHERE rv.org_id=$1::uuid AND rv.project_id=$2::uuid AND (block->>'id'='work:'||$3::text OR block->'basis'->>'work_item_id'=$3::text))
 OR EXISTS(SELECT 1 FROM proposal_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key<>$4 AND
 (trigger_work_item_id=$3::uuid OR created_record_id=$3::uuid OR inputs_snapshot @> jsonb_build_object('reason',jsonb_build_object('triggers',jsonb_build_array(jsonb_build_object('work_item_id',$3::text))))
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(undo_history) h
 WHERE h->>'trigger_work_item_id'=$3::text OR h->>'created_record_id'=$3::text
 OR h->'inputs_snapshot' @> jsonb_build_object('reason',jsonb_build_object('triggers',jsonb_build_array(jsonb_build_object('work_item_id',$3::text))))
 )))`, org, project, d.CreatedRecordID, key).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return ErrProposalUndoBlocked
			}
			tag, err := tx.Exec(ctx, `UPDATE work_items SET retired_at=now(),retired_by=$4::uuid,version=version+1
 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND version=$5 AND source_proposal_key=$6 AND retired_at IS NULL AND review_status='accepted_for_planning'`, org, project, d.CreatedRecordID, actor, d.CreatedRecordVersion, key)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return ErrProposalUndoBlocked
			}
		}
	}
	// Keep the decision-time snapshot and actor alongside each undo. Retaining
	// the current row also keeps decision versions monotonic across reacceptance.
	if _, err := tx.Exec(ctx, `UPDATE proposal_decisions d SET undo_history=undo_history||jsonb_build_array((to_jsonb(d)-'undo_history')||jsonb_build_object('undone_by',$4::text,'undone_at',now())),undone_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key=$3`, org, project, key, actor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='open',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return err
	}
	if d.Decision == "accepted" && d.CreatedRecordType == "delivery_item" {
		if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
			return err
		}
		if err := workEvent(ctx, tx, s, org, project); err != nil {
			return err
		}
		return s.finishDeliveryWrite(ctx, tx, org, project)
	}
	if d.Decision == "accepted" && (d.CreatedRecordType == "package" || d.CreatedRecordType == "package_scope_item") {
		if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
			return err
		}
		if err := workEvent(ctx, tx, s, org, project); err != nil {
			return err
		}
		return s.finishPackagesWrite(ctx, tx, org, project)
	}
	return s.finishWorksWrite(ctx, tx, org, project)
}

// Reacceptance revives only the exact item retired by undo, never an edited or
// independently retired record. The historical identity and source key survive.
func undoneProposalWork(ctx context.Context, tx pgx.Tx, org, project, key string) (string, int64, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(w) FROM work_items w WHERE org_id=$1::uuid AND project_id=$2::uuid AND source_proposal_key=$3`, org, project, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	var item struct {
		ID      string `json:"id"`
		Version int64  `json:"version"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return "", 0, err
	}
	var eligible bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM work_items w JOIN proposal_decisions d ON d.org_id=w.org_id AND d.project_id=w.project_id AND d.proposal_key=w.source_proposal_key CROSS JOIN LATERAL jsonb_array_elements(d.undo_history) h
 WHERE w.org_id=$1::uuid AND w.project_id=$2::uuid AND w.id=$3::uuid AND w.retired_at IS NOT NULL AND h->>'decision'='accepted' AND h->>'created_record_id'=w.id::text AND (h->>'created_record_version')::bigint+1=w.version)`, org, project, item.ID).Scan(&eligible)
	if err != nil {
		return "", 0, err
	}
	if !eligible {
		return "", 0, ErrProposalUndoBlocked
	}
	return item.ID, item.Version, nil
}
