package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

func liveCostReference(ctx context.Context, tx pgx.Tx, org, project, field, id string) (bool, error) {
	if field != "package_id" && field != "package_stage_id" {
		return false, ErrInvalidPackage
	}
	var used bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cost_plans p JOIN cost_item_revisions r ON r.org_id=p.org_id AND r.plan_version_id=p.draft_version_id WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND r.`+field+`=$3::uuid AND r.posting AND NOT r.excluded)`, org, project, id).Scan(&used)
	return used, e
}
func costScopeChange(ctx context.Context, tx pgx.Tx, org, project, pkg, id string, old ScopeItem, next procurement.ScopeContent, retire bool) error {
	if retire {
		var linked bool
		e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cost_plans p JOIN scope_cost_links l ON l.org_id=p.org_id AND l.plan_version_id=p.draft_version_id JOIN cost_item_revisions r ON r.org_id=l.org_id AND r.plan_version_id=l.plan_version_id AND r.cost_item_id=l.cost_item_id WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND l.package_scope_item_id=$3::uuid AND r.posting AND NOT r.excluded)`, org, project, id).Scan(&linked)
		if e != nil {
			return e
		}
		if linked {
			return fmt.Errorf("%w: unlink live cost items before retiring scope", ErrInvalidPackage)
		}
	}
	if old.ItemKind != "responsibility" || old.Inclusion != "included" || (!retire && next.ItemKind == "responsibility" && next.Inclusion == "included" && next.WorkItemID == old.WorkItemID) {
		return nil
	}
	var orphan bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cost_plans p JOIN cost_item_revisions r ON r.org_id=p.org_id AND r.plan_version_id=p.draft_version_id WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND r.package_id=$3::uuid AND r.work_item_id=$4::uuid AND r.posting AND NOT r.excluded) AND NOT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND work_item_id=$4::uuid AND id<>$5::uuid AND item_kind='responsibility' AND inclusion='included' AND retired_at IS NULL)`, org, project, pkg, old.WorkItemID, id).Scan(&orphan)
	if e != nil {
		return e
	}
	if orphan {
		return fmt.Errorf("%w: reassign cost items before removing their last responsibility", ErrInvalidPackage)
	}
	return nil
}
