package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
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

// groupRFP times a combined engagement's RFP for five consultants
// (docs/plans/2026-11-07-combined-engagements-slice2.md): the first build,
// which also assembles every member's draft in its transaction, a refresh,
// the group's read and a member's setting edited through the group. The
// builds count towards report_assemble_group (the report_assemble budget);
// the read and settings towards report_read and report_setting.
func (a *apiClient) groupRFP(ctx context.Context, n int) error {
	disciplines := []string{"consultant.structural", "consultant.civil", "consultant.mechanical", "consultant.electrical", "consultant.hydraulic"}
	for i := 0; i < n; i++ {
		project, err := a.createProject(ctx, "Combined engagement RFP benchmark")
		if err != nil {
			return err
		}
		base := "/projects/" + project
		ids := []string{}
		for _, d := range disciplines {
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
		body, _ := json.Marshal(map[string]any{"package_ids": ids})
		raw, err := a.timed(ctx, "packages_combine", http.MethodPost, base+"/packages/combine", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var group procurement.Package
		if err := json.Unmarshal(raw, &group); err != nil {
			return err
		}
		raw, err = a.timed(ctx, "report_write", http.MethodPost, base+"/procurement/create", []byte(`{"package_id":"`+group.ID+`"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var created struct {
			Report store.Report `json:"report"`
		}
		if err := json.Unmarshal(raw, &created); err != nil {
			return err
		}
		report := "/reports/" + created.Report.ID
		for build := 0; build < 2; build++ {
			raw, err = a.timed(ctx, "report_assemble_group", http.MethodPost, report+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK)
			if err != nil {
				return err
			}
			var d store.ReportDraft
			if err := json.Unmarshal(raw, &d); err != nil {
				return err
			}
			if d.SourceRevisions.RFPParts != len(disciplines) {
				return fmt.Errorf("group RFP benchmark built %d parts", d.SourceRevisions.RFPParts)
			}
		}
		raw, err = a.timed(ctx, "report_read", http.MethodGet, report, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil || view.RFPGroup == nil || len(view.RFPGroup.Members) != len(disciplines) {
			return fmt.Errorf("group RFP benchmark read no members: %v", err)
		}
		// One member's service toggled through the group, as the editor does.
		member := view.RFPGroup.Members[i%len(disciplines)]
		key := ""
		for _, g := range member.Controls.Groups {
			for _, c := range g.Controls {
				if key == "" && strings.HasPrefix(c.Key, "position:") && (c.Value == "I" || c.Value == "X") {
					key = c.Key
				}
			}
		}
		if key == "" {
			return fmt.Errorf("group RFP benchmark: %s has no service to toggle", member.Title)
		}
		version := member.DraftVersion
		for j := 0; j < 4; j++ {
			value := []string{"X", "I"}[j%2]
			body, _ := json.Marshal(map[string]any{"value": value, "version": version, "view": created.Report.ID})
			raw, err := a.timed(ctx, "report_setting", http.MethodPut, "/reports/"+member.ReportID+"/settings/"+url.PathEscape(key), body, http.StatusOK)
			if err != nil {
				return err
			}
			var next store.ReportView
			if err := json.Unmarshal(raw, &next); err != nil || next.RFPGroup == nil {
				return fmt.Errorf("group setting returned no group view: %v", err)
			}
			version = 0
			for _, m := range next.RFPGroup.Members {
				if m.ReportID == member.ReportID {
					version = m.DraftVersion
				}
			}
		}
	}
	return nil
}
