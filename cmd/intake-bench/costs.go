package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/costs"
)

func (a *apiClient) costs(ctx context.Context, n int) error {
	project, e := a.createProject(ctx, "Cost plan endpoint benchmark")
	if e != nil {
		return e
	}
	base := "/projects/" + project + "/cost-plan"
	var plan costs.Plan
	for i := 0; i < n; i++ {
		body, _ := json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version, "label": fmt.Sprintf("Allowance %d", i), "line_kind": "project_wide", "category": "contingency", "posting": true, "origin": "user", "meaning": "allowance", "values": map[string]any{"budget": map[string]any{"amount": "100.01", "value_state": "known", "origin": "user", "meaning": "allowance"}}})
		raw, e := a.timed(ctx, "costs_write", http.MethodPost, base+"/items", body, http.StatusOK)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		if len(plan.Items) != i+1 {
			return fmt.Errorf("cost create missing item")
		}
		raw, e = a.timed(ctx, "costs_read", http.MethodGet, base+"/totals?by=overall", nil, http.StatusOK)
		if e != nil {
			return e
		}
		var totals costs.Totals
		if e = json.Unmarshal(raw, &totals); e != nil {
			return e
		}
		expected, e := costs.Multiply(fmt.Sprint(i+1), "100.01")
		if e != nil {
			return e
		}
		if totals.Overall["budget"].Amount == nil || *totals.Overall["budget"].Amount != expected {
			return fmt.Errorf("cost totals did not reconcile")
		}
	}
	return nil
}
