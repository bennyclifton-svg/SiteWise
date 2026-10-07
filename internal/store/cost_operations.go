package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
)

type CostSubdivision struct {
	CostWrite
	Children []CostItemInput `json:"children"`
	Residual string          `json:"residual"`
}

func (s *Store) SubdivideCostItem(ctx context.Context, org, project, item, actor string, in CostSubdivision) (costs.Plan, error) {
	if len(in.Children) < 1 || len(in.Children) > 200 || (in.Residual != "explicit" && in.Residual != "change_total") {
		return costs.Plan{}, costs.ErrInvalid
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	var parent *costs.Item
	for n := range p.Items {
		if p.Items[n].ID == item {
			parent = &p.Items[n]
			break
		}
	}
	if parent == nil {
		return p, ErrNotFound
	}
	if !parent.Posting || parent.Excluded {
		return p, costs.ErrInvalid
	}
	residual := map[string]costs.Value{}
	if in.Residual == "explicit" {
		for metric, pv := range parent.Values {
			vs := []costs.Value{}
			for _, c := range in.Children {
				v := c.Values[metric]
				if e = costs.ValidateValue(metric, &v); e != nil && v.ValueState != "" {
					return p, e
				}
				vs = append(vs, v)
			}
			v, e := costs.Residual(pv, vs)
			if e != nil {
				return p, fmt.Errorf("%w: %s over-allocation", e, metric)
			}
			residual[metric] = v
		}
	}
	// Values leave the posting parent before it becomes a group. The transaction
	// exposes either the complete allocation or the original allowance.
	if _, e = tx.Exec(ctx, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, item); e != nil {
		return p, e
	}
	if _, e = tx.Exec(ctx, `UPDATE cost_item_revisions SET posting=false,quantity=NULL,rate=NULL,unit=NULL,rate_basis=NULL,version=version+1 WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, item); e != nil {
		return p, e
	}
	for _, child := range in.Children {
		child.ParentItemID = item
		if !child.Posting || child.Excluded {
			return p, costs.ErrInvalid
		}
		id := newID()
		if e = insertCostItem(ctx, tx, org, project, p.ID, actor, id, child.Content); e != nil {
			return p, e
		}
		for m, v := range child.Values {
			if e = writeCostValue(ctx, tx, org, project, p.ID, id, actor, m, v); e != nil {
				return p, e
			}
		}
	}
	if in.Residual == "explicit" {
		c := parent.Content
		c.ParentItemID = item
		c.Label = "Unallocated — " + c.Label
		c.Code = ""
		c.Quantity = nil
		c.Rate = nil
		c.Unit = ""
		c.RateBasis = ""
		c.Meaning = "allowance"
		id := newID()
		if e = insertCostItem(ctx, tx, org, project, p.ID, actor, id, c); e != nil {
			return p, e
		}
		for m, v := range residual {
			if e = writeCostValue(ctx, tx, org, project, p.ID, id, actor, m, v); e != nil {
				return p, e
			}
		}
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func (s *Store) BaselineCostPlan(ctx context.Context, org, project, actor string, in CostWrite) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, in)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	// Capture dimensions now; later work-item edits cannot rewrite a baseline.
	if _, e = tx.Exec(ctx, `UPDATE cost_item_revisions r SET system_id=w.system_id,part_id=w.part_id FROM work_items w WHERE r.org_id=$1::uuid AND r.plan_version_id=$2::uuid AND w.org_id=r.org_id AND w.project_id=r.project_id AND w.id=r.work_item_id`, org, p.ID); e != nil {
		return p, e
	}
	if _, e = tx.Exec(ctx, `UPDATE cost_plan_versions SET status='baseline',frozen_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, p.ID); e != nil {
		return p, e
	}
	id := newID()
	if _, e = tx.Exec(ctx, `INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,currency,tax_basis,tax_rate,price_date,coverage,funding_target,funding_target_provenance) SELECT org_id,project_id,$3::uuid,revision+1,'draft',currency,tax_basis,tax_rate,price_date,coverage,funding_target,funding_target_provenance FROM cost_plan_versions WHERE org_id=$1::uuid AND id=$2::uuid`, org, p.ID, id); e != nil {
		return p, e
	}
	if e = cloneCostRows(ctx, tx, org, p.ID, id); e != nil {
		return p, e
	}
	if _, e = tx.Exec(ctx, `UPDATE cost_plans SET draft_version_id=$3::uuid,baseline_version_id=$4::uuid WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project, id, p.ID); e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, id)
}
func cloneCostRows(ctx context.Context, tx pgx.Tx, org, old, id string) error {
	// Explicit columns ensure later schema additions require a conscious snapshot
	// decision rather than silently disappearing from a baseline clone.
	queries := []string{
		`INSERT INTO cost_item_revisions(org_id,project_id,plan_version_id,cost_item_id,parent_item_id,code,label,line_kind,category,work_item_id,package_id,package_stage_id,posting,quantity,unit,rate,rate_basis,excluded,origin,review_status,meaning,rationale,provenance,system_id,part_id,version) SELECT org_id,project_id,$3::uuid,cost_item_id,parent_item_id,code,label,line_kind,category,work_item_id,package_id,package_stage_id,posting,quantity,unit,rate,rate_basis,excluded,origin,review_status,meaning,rationale,provenance,system_id,part_id,version FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid`,
		`INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,review_status,meaning,rationale,provenance,version) SELECT org_id,project_id,$3::uuid,cost_item_id,metric,value_state,amount,low,high,as_of,origin,review_status,meaning,rationale,provenance,version FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid`,
		`INSERT INTO scope_cost_links(org_id,project_id,plan_version_id,package_scope_item_id,cost_item_id) SELECT org_id,project_id,$3::uuid,package_scope_item_id,cost_item_id FROM scope_cost_links WHERE org_id=$1::uuid AND plan_version_id=$2::uuid`,
	}
	for _, q := range queries {
		if _, e := tx.Exec(ctx, q, org, old, id); e != nil {
			return e
		}
	}
	return nil
}

type CostLinkInput struct {
	CostWrite
	ScopeItemID string `json:"package_scope_item_id"`
	CostItemID  string `json:"cost_item_id"`
	Remove      bool   `json:"remove"`
}

func (s *Store) LinkScopeCost(ctx context.Context, org, project, actor string, in CostLinkInput) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	found := false
	for _, i := range p.Items {
		if i.ID == in.CostItemID {
			found = true
		}
	}
	if !found {
		return p, ErrNotFound
	}
	var ok bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL)`, org, project, in.ScopeItemID).Scan(&ok)
	if e != nil {
		return p, e
	}
	if !ok {
		return p, ErrNotFound
	}
	if in.Remove {
		_, e = tx.Exec(ctx, `DELETE FROM scope_cost_links WHERE org_id=$1::uuid AND project_id=$2::uuid AND plan_version_id=$3::uuid AND cost_item_id=$4::uuid AND package_scope_item_id=$5::uuid`, org, project, p.ID, in.CostItemID, in.ScopeItemID)
	} else {
		_, e = tx.Exec(ctx, `INSERT INTO scope_cost_links(org_id,project_id,plan_version_id,cost_item_id,package_scope_item_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid) ON CONFLICT DO NOTHING`, org, project, p.ID, in.CostItemID, in.ScopeItemID)
	}
	if e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}

// RemoveCostItem removes only the current draft occurrence. Stable identities
// and frozen baseline rows remain available to issued pricing schedules.
func (s *Store) RemoveCostItem(ctx context.Context, org, project, item, actor string, in CostWrite) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, in)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	found := false
	for _, i := range p.Items {
		if i.ID == item {
			found = true
		}
		if i.ParentItemID == item {
			return p, fmt.Errorf("%w: remove or reparent children first", costs.ErrInvalid)
		}
	}
	if !found {
		return p, ErrNotFound
	}
	for _, q := range []string{`DELETE FROM scope_cost_links WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, `DELETE FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`} {
		if _, e = tx.Exec(ctx, q, org, p.ID, item); e != nil {
			return p, e
		}
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
