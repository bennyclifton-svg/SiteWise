package store

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
)

func calculateCostInput(in *CostValueInput, metric string, item costs.Item) error {
	if !in.Calculate {
		return nil
	}
	if metric != "budget" && metric != "estimate" {
		return costs.ErrInvalid
	}
	if item.Quantity == nil || item.Rate == nil || in.Amount != nil || in.Low != nil || in.High != nil {
		return costs.ErrInvalid
	}
	amount, e := costs.Multiply(string(*item.Quantity), string(*item.Rate))
	if e != nil {
		return e
	}
	in.Amount = &amount
	in.ValueState = "known"
	in.Origin = "user"
	if in.Meaning == "" {
		in.Meaning = "allowance"
	}
	return nil
}
func recordCostCalculation(ctx context.Context, tx pgx.Tx, org, version, id, metric string, item costs.Item) error {
	raw, _ := json.Marshal(map[string]any{"method": "quantity_times_rate", "method_version": 1, "quantity": item.Quantity, "rate": item.Rate, "unit": item.Unit, "rate_basis": item.RateBasis, "input_origin": item.Origin, "limitations": "User-supplied quantity and rate; not a benchmark or verified price"})
	_, e := tx.Exec(ctx, `UPDATE cost_values SET origin='calculation',provenance=provenance||$5::jsonb WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric=$4`, org, version, id, metric, raw)
	return e
}
