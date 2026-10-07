package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/reports"
	"strings"
)

func reportCommercial(ctx context.Context, tx pgx.Tx, org, project, pkg, kind string) ([]reports.BriefValue, string, error) {
	p, err := readCostPlan(ctx, tx, org, project, "")
	if errors.Is(err, ErrNotFound) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if p.Status == "unavailable" {
		return nil, "", nil
	}
	out := []reports.BriefValue{}
	for _, item := range p.Items {
		if kind == "pmp" || !item.Posting || item.Excluded || item.PackageID != pkg {
			continue
		}
		basis, _ := json.Marshal(map[string]any{"method": "pricing_schedule", "plan_version_id": p.ID, "cost_item_id": item.ID, "item_version": item.Version, "label": item.Label, "stage_id": item.PackageStageID, "currency": p.Currency, "tax_basis": p.TaxBasis, "budget": item.Values["budget"]})
		out = append(out, reports.BriefValue{ID: "line:" + item.ID, Label: item.Label, Text: fmt.Sprintf("Pricing ID %s; stage %s; price to be returned in %s (%s).", item.ID, item.PackageStageID, p.Currency, p.TaxBasis), Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "requirement", Basis: basis})
	}
	if kind == "pmp" {
		totals, err := costs.Summarize(p, "system")
		if err != nil {
			return nil, "", err
		}
		for _, metric := range []string{"budget", "estimate", "commitment", "claimed_to_date", "forecast"} {
			t := totals.Overall[metric]
			text := "Not available"
			if t.Amount != nil {
				text = string(*t.Amount) + " " + p.Currency
			} else if t.Unknown == 0 && t.Low != nil && t.High != nil {
				text = string(*t.Low) + "–" + string(*t.High) + " " + p.Currency
			} else if t.Lines > 0 {
				text = fmt.Sprintf("Incomplete: known subtotal %s %s; %d of %d lines unresolved", t.KnownSubtotal, p.Currency, t.Unknown, t.Lines)
			}
			basis, _ := json.Marshal(map[string]any{"method": "posting_leaf_sum", "plan_version_id": p.ID, "metric": metric, "total": t, "tax_basis": p.TaxBasis})
			out = append(out, reports.BriefValue{ID: "summary:" + metric, Label: strings.ReplaceAll(metric, "_", " "), Text: text, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated", Basis: basis, Unknown: t.Lines == 0})
		}
		variance := "Not available"
		if totals.Variance != nil {
			variance = string(*totals.Variance) + " " + p.Currency
		}
		// Freeze the exact ledger behind the summary once, without turning a
		// progress report into a tender return schedule.
		basis, _ := json.Marshal(map[string]any{"method": "forecast_minus_budget", "plan_version_id": p.ID, "totals": totals, "inputs": p})
		out = append(out, reports.BriefValue{ID: "variance", Label: "Forecast less budget", Text: variance, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "forecast", Basis: basis})
		basis, _ = json.Marshal(map[string]any{"method": "cost_plan_coverage", "plan_version_id": p.ID, "coverage": p.Coverage})
		out = append(out, reports.BriefValue{ID: "coverage", Label: "Cost coverage", Text: p.Coverage, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated", Basis: basis, Unknown: p.Coverage == ""})
	}
	return out, p.ID, nil
}
