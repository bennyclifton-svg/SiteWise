package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"sitewise/internal/procurement"
)

// packageGroups times combine then uncombine of two consultant rows whose
// draft fee lines exist, so each write re-parents fee lines in its one
// transaction (docs/plans/2026-11-07-combined-engagements-slice1.md). The
// Procurement and cost plan reads with a group in place count towards their
// own budgets.
func (a *apiClient) packageGroups(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		project, err := a.createProject(ctx, "Combined engagement benchmark")
		if err != nil {
			return err
		}
		base := "/projects/" + project
		ids := []string{}
		for _, d := range []string{"consultant.structural", "consultant.civil"} {
			raw, err := a.timed(ctx, "packages_write", http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"`+d+`"}`), http.StatusCreated)
			if err != nil {
				return err
			}
			var p procurement.Package
			if err := json.Unmarshal(raw, &p); err != nil {
				return err
			}
			ids = append(ids, p.ID)
		}
		if _, err := a.timed(ctx, "costs_prepare", http.MethodPost, base+"/cost-plan/prepare", []byte(`{}`), http.StatusOK); err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]any{"package_ids": ids})
		raw, err := a.timed(ctx, "packages_combine", http.MethodPost, base+"/packages/combine", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var group procurement.Package
		if err := json.Unmarshal(raw, &group); err != nil {
			return err
		}
		if !group.IsGroup() {
			return fmt.Errorf("combine benchmark made no group")
		}
		if _, err := a.timed(ctx, "packages_read", http.MethodGet, base+"/packages", nil, http.StatusOK); err != nil {
			return err
		}
		raw, err = a.timed(ctx, "costs_read", http.MethodGet, base+"/cost-plan", nil, http.StatusOK)
		if err != nil {
			return err
		}
		if err := nestedUnder(raw, group.ID, ids); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "packages_combine", http.MethodPost, base+"/packages/"+group.ID+"/uncombine", nil, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}

// nestedUnder checks the cost plan read holds each member's fee line under
// the group's line, so the benchmark times the real nesting.
func nestedUnder(raw []byte, group string, members []string) error {
	var plan struct {
		Items []struct {
			ID        string `json:"cost_item_id"`
			Parent    string `json:"parent_item_id"`
			PackageID string `json:"package_id"`
			LineKind  string `json:"line_kind"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return err
	}
	parent := ""
	for _, it := range plan.Items {
		if it.LineKind == "fee" && it.PackageID == group {
			parent = it.ID
		}
	}
	nested := 0
	for _, it := range plan.Items {
		for _, m := range members {
			if it.LineKind == "fee" && it.PackageID == m && parent != "" && it.Parent == parent {
				nested++
			}
		}
	}
	if nested != len(members) {
		return fmt.Errorf("combine benchmark: %d of %d member fee lines nested", nested, len(members))
	}
	return nil
}
