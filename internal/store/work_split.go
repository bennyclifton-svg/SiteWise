package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

type WorkBlocker struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}
type WorkReferencedError struct {
	Blockers []WorkBlocker `json:"blockers"`
}

func (e *WorkReferencedError) Error() string { return "referenced" }

func readWorkForWrite(ctx context.Context, tx pgx.Tx, org, project, id string, version int64) (works.Item, error) {
	var item works.Item
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(w)||jsonb_build_object('quantity',w.quantity::text) FROM work_items w WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL`, org, project, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	if version < 1 {
		return item, ErrInvalidWork
	}
	if version != item.Version {
		return item, ErrVersionConflict
	}
	return item, nil
}

func (s *Store) SplitWorkItem(ctx context.Context, org, project, id, actor string, request works.SplitRequest) ([]works.Item, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockProject(ctx, tx, org, project); err != nil {
		return nil, err
	}
	if err = packageActor(ctx, tx, org, actor); err != nil {
		return nil, err
	}
	parent, err := readWorkForWrite(ctx, tx, org, project, id, request.Version)
	if err != nil {
		return nil, err
	}
	children, err := works.SplitChildren(parent, request, actor, s.workCatalog())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWork, err)
	}
	for i := range children {
		child := &children[i]
		if _, err = userValueOwner(ctx, tx, org, project, child.PartID); err != nil {
			return nil, err
		}
		child.ID = newID()
		target, _ := json.Marshal(child.Target)
		prov, _ := json.Marshal(child.Provenance)
		_, err = tx.Exec(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,parent_id,title,existing_condition_note,target,origin,review_status,meaning,provenance,user_touched,layout_change) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9::uuid,$10,$11,$12,'user','accepted_for_planning','stated',$13,true,$14)`, org, child.ID, project, child.SiteID, child.PartID, child.SystemID, child.Action, child.Inclusion, id, child.Title, child.ExistingConditionNote, target, prov, child.LayoutChange)
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE work_items SET is_group=true,user_touched=true,version=version+1,provenance=provenance||jsonb_build_object('last_edited_by',$4::text,'last_edited_at',now()) WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id, actor)
	if err != nil {
		return nil, err
	}
	if err = s.finishWorksWrite(ctx, tx, org, project); err != nil {
		return nil, err
	}
	return children, nil
}

func (s *Store) RetireWorkItem(ctx context.Context, org, project, id, actor string, version int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err = packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	if _, err = readWorkForWrite(ctx, tx, org, project, id, version); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT 'responsibility',id::text FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND work_item_id=$3::uuid AND retired_at IS NULL
 UNION ALL SELECT 'delivery',id::text FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND work_item_id=$3::uuid AND retired_at IS NULL AND `+deliveryOpenSQL+`
 UNION ALL SELECT DISTINCT 'cost',r.cost_item_id::text FROM cost_plans p JOIN cost_item_revisions r ON r.org_id=p.org_id AND r.plan_version_id=p.draft_version_id JOIN cost_values v ON v.org_id=r.org_id AND v.plan_version_id=r.plan_version_id AND v.cost_item_id=r.cost_item_id WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND r.work_item_id=$3::uuid AND r.posting AND NOT r.excluded
 UNION ALL SELECT 'child',id::text FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND parent_id=$3::uuid AND retired_at IS NULL`, org, project, id)
	if err != nil {
		return err
	}
	blockers := []WorkBlocker{}
	for rows.Next() {
		var b WorkBlocker
		if err = rows.Scan(&b.Kind, &b.ID); err != nil {
			rows.Close()
			return err
		}
		blockers = append(blockers, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(blockers) > 0 {
		return &WorkReferencedError{blockers}
	}
	_, err = tx.Exec(ctx, `UPDATE work_items SET retired_at=now(),retired_by=$4::uuid,user_touched=true,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id, actor)
	if err != nil {
		return err
	}
	return s.finishWorksWrite(ctx, tx, org, project)
}
