package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/costs"

	"sitewise/internal/store"
)

func (a *apiClient) costs(ctx context.Context, st *store.Store, n int) error {
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
	return a.costAutofill(ctx, st, n)
}

func (a *apiClient) costAutofill(ctx context.Context, st *store.Store, n int) error {
	for i := 0; i < n; i++ {
		project, e := a.createProject(ctx, "Cost autofill benchmark")
		if e != nil {
			return e
		}
		part, e := st.CreatePart(ctx, benchOrg, project, "Building", "part", "")
		if e != nil {
			return e
		}
		for _, system := range []string{"site", "substructure", "structure", "envelope", "interiors", "fire-passive", "fire-active", "mechanical", "hydraulic", "electrical", "comms-security", "vertical-transport", "access-egress"} {
			body, _ := json.Marshal(map[string]string{"part_id": part.ID, "system_id": system, "action": "new", "title": system})
			if _, e = a.timed(ctx, "works_write", http.MethodPost, "/projects/"+project+"/works", body, 201); e != nil {
				return e
			}
		}
		for j := 0; j < 8; j++ {
			body, _ := json.Marshal(map[string]string{"title": fmt.Sprintf("Consultant %d", j), "kind": "services", "lifecycle_status": "planned"})
			if _, e = a.timed(ctx, "packages_write", http.MethodPost, "/projects/"+project+"/packages", body, 201); e != nil {
				return e
			}
		}
		base := "/projects/" + project + "/cost-plan"
		raw, e := a.timed(ctx, "costs_read", http.MethodGet, base, nil, 200)
		if e != nil {
			return e
		}
		var plan costs.Plan
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		if len(plan.Suggestions) < 25 {
			return fmt.Errorf("autofill benchmark missing scope")
		}
		body, _ := json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version})
		raw, e = a.timed(ctx, "costs_write", http.MethodPost, base+"/suggestions/"+plan.Suggestions[0].Key+"/accept", body, 200)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		body, _ = json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version})
		raw, e = a.timed(ctx, "costs_accept_all", http.MethodPost, base+"/suggestions/accept", body, 200)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		if len(plan.Suggestions) != 0 {
			return fmt.Errorf("autofill benchmark left suggestions")
		}
		// Remove and restore source keys outside the measured prepare path.
		for len(plan.Items) > 0 {
			body, _ = json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version})
			raw, e = a.timed(ctx, "costs_delete", http.MethodDelete, base+"/items/"+plan.Items[0].ID, body, 200)
			if e != nil {
				return e
			}
			if e = json.Unmarshal(raw, &plan); e != nil {
				return e
			}
		}
		dismissed := append([]costs.Suggestion(nil), plan.Dismissed...)
		for _, sg := range dismissed {
			body, _ = json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version})
			raw, e = a.timed(ctx, "costs_write", http.MethodDelete, base+"/suggestions/"+sg.Key+"/dismiss", body, 200)
			if e != nil {
				return e
			}
			if e = json.Unmarshal(raw, &plan); e != nil {
				return e
			}
		}
		raw, e = a.timed(ctx, "costs_prepare", http.MethodPost, base+"/prepare", []byte(`{}`), 200)
		if e != nil {
			return e
		}
		var prepared store.PreparedCostPlan
		if e = json.Unmarshal(raw, &prepared); e != nil {
			return e
		}
		if len(prepared.Plan.Items) < 25 || len(prepared.Plan.Suggestions) != 0 {
			return fmt.Errorf("prepare did not save defaults")
		}
		plan = prepared.Plan
		raw, e = a.timed(ctx, "costs_prepare_noop", http.MethodPost, base+"/prepare", []byte(`{}`), 200)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &prepared); e != nil {
			return e
		}
		if prepared.Plan.Version != plan.Version {
			return fmt.Errorf("no-op prepare changed version")
		}
		// Exercise the two direct grid writes against the full representative plan.
		var target costs.Item
		for _, row := range plan.Items {
			if row.LineKind == "works" && row.Posting {
				target = row
				break
			}
		}
		body, _ = json.Marshal(map[string]any{"plan_version_id": plan.ID, "plan_version": plan.Version, "amount": "12345.67", "value_state": "known", "origin": "user", "meaning": "allowance"})
		raw, e = a.timed(ctx, "costs_contract_edit", http.MethodPut, base+"/items/"+target.ID+"/values/commitment", body, 200)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		raw, e = a.timed(ctx, "packages_write", http.MethodPost, "/projects/"+project+"/packages", []byte(`{"title":"Head contractor","kind":"works","works_scope":"head_contract","lifecycle_status":"planned"}`), 201)
		if e != nil {
			return e
		}
		var pkg struct {
			ID string `json:"id"`
		}
		if e = json.Unmarshal(raw, &pkg); e != nil {
			return e
		}
		target.PackageID = pkg.ID
		body, _ = json.Marshal(store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version}, Content: target.Content})
		raw, e = a.timed(ctx, "costs_package_assign", http.MethodPatch, base+"/items/"+target.ID, body, 200)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}

		body, _ = json.Marshal(map[string]any{"plan_version_id": plan.ID, "version": plan.Version})
		if _, e = a.timed(ctx, "costs_freeze", http.MethodPost, base+"/baseline", body, 200); e != nil {
			return e
		}

	}
	return nil
}
