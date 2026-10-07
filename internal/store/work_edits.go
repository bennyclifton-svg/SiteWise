package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

func (s *Store) PatchWorkItem(ctx context.Context, org, project, id, actor string, patch works.Patch) (works.Item, error) {
	var item works.Item
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
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(w)||jsonb_build_object('quantity',w.quantity::text) FROM work_items w WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL`, org, project, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	if patch.Version < 1 {
		return item, ErrInvalidWork
	}
	if item.Version != patch.Version {
		return item, ErrVersionConflict
	}
	if err := patch.Apply(&item); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidWork, err)
	}
	if item.LayoutChange == "" {
		item.LayoutChange = "unknown"
	}
	if err := works.Validate(item, s.workCatalog()); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidWork, err)
	}
	if _, err := userValueOwner(ctx, tx, org, project, item.PartID); err != nil {
		return item, err
	}
	target, err := json.Marshal(item.Target)
	if err != nil {
		return item, err
	}
	// Keep original evidence and proposal provenance; record this correction
	// separately. A corrected item is planning input, never owner verification.
	err = tx.QueryRow(ctx, `UPDATE work_items SET part_id=$12::uuid,system_id=$13,coarse_key=CASE WHEN coarse_key IS NULL THEN NULL ELSE $12||'|'||$13 END,layout_change=$14,action=$4,title=$5,inclusion=$6,existing_condition_note=$7,target=$8,quantity=$9::numeric,unit=$10,origin='user',review_status='accepted_for_planning',meaning='stated',user_touched=true,version=version+1,verified_by=NULL,verified_at=NULL,verification_basis=NULL,provenance=provenance||jsonb_build_object('last_edited_by',$11::text,'last_edited_at',now()) WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid RETURNING to_jsonb(work_items)||jsonb_build_object('quantity',quantity::text)`, org, project, id, item.Action, item.Title, item.Inclusion, item.ExistingConditionNote, target, item.Quantity, item.Unit, actor, item.PartID, item.SystemID, item.LayoutChange).Scan(&raw)
	if err != nil {
		return item, workError(err)
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	if err := s.finishWorksWrite(ctx, tx, org, project); err != nil {
		return item, err
	}
	return item, nil
}
