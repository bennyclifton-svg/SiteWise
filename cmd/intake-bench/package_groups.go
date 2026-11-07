package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sitewise/internal/invoices/sample"
	"sitewise/internal/procurement"
	psample "sitewise/internal/proposals/sample"
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

// combineForProposal times Combine for this proposal
// (docs/plans/2026-11-07-combined-engagements-slice5.md): an unplaced
// proposal on a split project with a Services Engineer row, the Procurement
// read with its combine prompt (packages_read), the combine that places the
// proposal on the new group, then uncombine, which leaves the proposal
// unplaced again for the next sample (both packages_combine).
func (a *apiClient) combineForProposal(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Combine for this proposal benchmark")
	if err != nil {
		return err
	}
	watch := &submissionWatch{read: map[string]chan struct{}{}}
	stop, err := a.procurementStream(ctx, project, watch)
	if err != nil {
		return err
	}
	defer stop()
	base := "/projects/" + project
	ids := []string{}
	for _, d := range []string{"consultant.mechanical", "consultant.electrical", procurement.ServicesFiling} {
		raw, err := a.call(ctx, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"`+d+`"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var p procurement.Package
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		ids = append(ids, p.ID)
	}
	pdf := psample.PDF(psample.Proposal{Firm: "Bench Services Pty Ltd", ABN: sample.ABNFor(900), Email: "office@benchservices.test", Lines: []psample.Line{{Label: "Design", Cents: 500000}}})
	resp, err := a.do(ctx, http.MethodPost, base+"/procurement/packages/"+ids[0]+"/submissions?name="+url.QueryEscape(benchFilePrefix+"fee proposal (Bench Services).pdf"), bytes.NewReader(pdf), int64(len(pdf)))
	if err != nil {
		return err
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("proposal upload %d: %s", resp.StatusCode, raw)
	}
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(raw, &up); err != nil {
		return err
	}
	select {
	case <-watch.wait(up.SubmissionID):
	case <-time.After(30 * time.Second):
		return errors.New("proposal not read within 30 s")
	case <-ctx.Done():
		return ctx.Err()
	}
	// Detached by the PM: it now waits for a package.
	if _, err := a.call(ctx, http.MethodPatch, base+"/submissions/"+up.SubmissionID, []byte(`{"package_id":""}`), http.StatusOK); err != nil {
		return err
	}
	members := ids[:2]
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "packages_read", http.MethodGet, base+"/packages", nil, http.StatusOK)
		if err != nil {
			return err
		}
		var o struct {
			Prompts []procurement.CombinePrompt `json:"prompts"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return err
		}
		if len(o.Prompts) != 1 {
			return fmt.Errorf("combine prompt benchmark: %d prompts", len(o.Prompts))
		}
		body, _ := json.Marshal(map[string]any{"package_ids": members, "submission_id": up.SubmissionID})
		raw, err = a.timed(ctx, "packages_combine", http.MethodPost, base+"/packages/combine", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var group procurement.Package
		if err := json.Unmarshal(raw, &group); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "packages_combine", http.MethodPost, base+"/packages/"+group.ID+"/uncombine", nil, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}
