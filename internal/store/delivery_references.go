package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

// Terminal statuses are disjoint from every kind's open statuses. Per-kind
// validation happens before storage; historical records still block undo.
const deliveryOpenSQL = `status NOT IN ('complete','cancelled','achieved','closed','resolved','decided','superseded','approved','rejected','withdrawn')`

func openDeliveryReference(ctx context.Context, tx pgx.Tx, org, project, column, id string) (bool, error) {
	switch column {
	case "package_id", "work_item_id", "stage_id":
	default:
		return false, fmt.Errorf("unknown delivery reference")
	}
	var used bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND `+column+`=$3::uuid AND retired_at IS NULL AND `+deliveryOpenSQL+`)`, org, project, id).Scan(&used)
	return used, err
}
