package store

import (
	"context"
	"sitewise/internal/costs"
)

type CostOrderInput struct {
	CostWrite
	IDs []string `json:"ids"`
}

// Reorder one complete sibling set atomically. Order is revision metadata so
// existing baseline cloning and undo snapshots preserve it without changing codes.
func (s *Store) ReorderCostItems(ctx context.Context, org, project, actor string, in CostOrderInput) (costs.Plan, error) {
	tx, p, err := s.costTx(ctx, org, project, actor, in.CostWrite)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if len(in.IDs) == 0 || len(in.IDs) > len(p.Items) {
		return p, costs.ErrInvalid
	}
	byID := map[string]costs.Item{}
	for _, item := range p.Items {
		byID[item.ID] = item
	}
	first, ok := byID[in.IDs[0]]
	if !ok {
		return p, ErrNotFound
	}
	sections := costs.CostSections(p.Items)
	seen := map[string]bool{}
	for _, id := range in.IDs {
		item, ok := byID[id]
		if !ok {
			return p, ErrNotFound
		}
		if seen[id] || item.ParentItemID != first.ParentItemID || sections[id] != sections[first.ID] {
			return p, costs.ErrInvalid
		}
		seen[id] = true
	}
	for _, item := range p.Items {
		if item.ParentItemID == first.ParentItemID && sections[item.ID] == sections[first.ID] && !seen[item.ID] {
			return p, costs.ErrInvalid
		}
	}
	// One set update, irrespective of the number of dragged rows.
	_, err = tx.Exec(ctx, `UPDATE cost_item_revisions r SET provenance=r.provenance||jsonb_build_object('display_order',o.position::int,'order_actor',$4::text),version=r.version+1 FROM unnest($5::text[]) WITH ORDINALITY AS o(id,position) WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.plan_version_id=$3::uuid AND r.cost_item_id::text=o.id`, org, project, p.ID, actor, in.IDs)
	if err != nil {
		return p, err
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
