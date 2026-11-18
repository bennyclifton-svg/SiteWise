package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func (a *apiClient) reports(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Report benchmark")
	if err != nil {
		return err
	}
	raw, err := a.timed(ctx, "packages_write", http.MethodPost, "/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Engineering"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	var pkg procurement.Package
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": pkg.ID})
		raw, err := a.timed(ctx, "report_write", http.MethodPost, "/projects/"+project+"/reports", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var r store.Report
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		base := "/reports/" + r.ID
		raw, err = a.timed(ctx, "report_assemble", http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK)
		if err != nil {
			return err
		}
		var d store.ReportDraft
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		issued := 0
		for _, section := range d.Sections {
			if !section.Internal {
				issued++
			}
		}
		if issued != 7 {
			return fmt.Errorf("report benchmark missing sections")
		}
		body, _ = json.Marshal(map[string]any{"version": d.Version, "text": "Review the engineering scope."})
		if _, err := a.timed(ctx, "report_write", http.MethodPut, base+"/edits/"+url.PathEscape("package:"+pkg.ID), body, http.StatusOK); err != nil {
			return err
		}
		raw, err = a.timed(ctx, "report_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil {
			return err
		}
		if len(view.Edits) != 1 {
			return fmt.Errorf("report benchmark missing protected edit")
		}
		raw, err = a.timed(ctx, "report_read", http.MethodGet, "/projects/"+project+"/reports", nil, http.StatusOK)
		if err != nil {
			return err
		}
		var listed []store.Report
		if err := json.Unmarshal(raw, &listed); err != nil {
			return err
		}
		if len(listed) != i+1 {
			return fmt.Errorf("report list omitted a saved report")
		}
		body, _ = json.Marshal(map[string]int64{"version": view.Draft.Version})
		if _, err := a.timed(ctx, "report_write", http.MethodDelete, base+"/edits/"+url.PathEscape("package:"+pkg.ID), body, http.StatusOK); err != nil {
			return err
		}
	}
	if err := a.consultantRFP(ctx, project, n); err != nil {
		return err
	}
	// PMP has no catalogue clauses requiring owner review. This exercises the
	// actual immutable issue and saved PDF download paths using synthetic data.
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "report_write", http.MethodPost, "/projects/"+project+"/reports", []byte(`{"kind":"pmp"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var r store.Report
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		base := "/reports/" + r.ID
		raw, err = a.timed(ctx, "report_assemble", http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK)
		if err != nil {
			return err
		}
		var d store.ReportDraft
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		body, _ := json.Marshal(store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Synthetic benchmark; missing profile disclosed"})
		raw, err = a.timed(ctx, "report_issue", http.MethodPost, base+"/issue", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var issue store.IssuedReport
		if err := json.Unmarshal(raw, &issue); err != nil {
			return err
		}
		raw, err = a.timed(ctx, "report_export", http.MethodGet, base+"/versions/"+issue.ID+"/file", nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(raw) < 5 || string(raw[:5]) != "%PDF-" {
			return fmt.Errorf("issued export is not a PDF")
		}
	}
	return nil
}

// consultantRFP times the consultant RFP paths: assembly from a discipline
// file, structured setting changes that re-resolve from frozen drivers, a
// reset, and the read that returns the editor controls.
func (a *apiClient) consultantRFP(ctx context.Context, project string, n int) error {
	raw, err := a.timed(ctx, "packages_write", http.MethodPost, "/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect","discipline_id":"consultant.architect"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	var pkg procurement.Package
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": pkg.ID})
	raw, err = a.timed(ctx, "report_write", http.MethodPost, "/projects/"+project+"/reports", body, http.StatusCreated)
	if err != nil {
		return err
	}
	var r store.Report
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	base := "/reports/" + r.ID
	raw, err = a.timed(ctx, "report_assemble", http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK)
	if err != nil {
		return err
	}
	var d store.ReportDraft
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	if d.TemplateID != "tpl.rfp-consultant" {
		return fmt.Errorf("consultant RFP benchmark assembled %s", d.TemplateID)
	}
	version := d.Version
	key := base + "/settings/" + url.PathEscape("position:consultant_selection")
	for i := 0; i < max(n, 20); i++ {
		value := "X"
		if i%2 == 1 {
			value = "I"
		}
		body, _ := json.Marshal(map[string]any{"value": value, "version": version})
		raw, err := a.timed(ctx, "report_setting", http.MethodPut, key, body, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil || view.Draft == nil || view.RFP == nil {
			return fmt.Errorf("setting returned no draft or controls: %v", err)
		}
		version = view.Draft.Version
	}
	// A term reworded in the terms table is the same settings write, with
	// its field kept as a slot (owner, 10 Nov 2026).
	termKey := base + "/settings/" + url.PathEscape("wording:terms.validity")
	for i := 0; i < max(n/2, 10); i++ {
		body, _ := json.Marshal(map[string]any{"value": fmt.Sprintf("Your proposal stays valid for {field:validity_days} days, revision %d.", i), "version": version})
		raw, err := a.timed(ctx, "report_setting", http.MethodPut, termKey, body, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil || view.Draft == nil {
			return fmt.Errorf("term setting returned no draft: %v", err)
		}
		version = view.Draft.Version
	}
	body, _ = json.Marshal(map[string]int64{"version": version})
	raw, err = a.timed(ctx, "report_setting", http.MethodDelete, key, body, http.StatusOK)
	if err != nil {
		return err
	}
	var deleted store.ReportView
	if err := json.Unmarshal(raw, &deleted); err != nil || deleted.Draft == nil {
		return fmt.Errorf("delete returned no draft: %v", err)
	}
	version = deleted.Draft.Version
	// A project fact edited in the RFP writes Profile and refreshes the draft.
	for i := 0; i < max(n, 20); i++ {
		body, _ := json.Marshal(map[string]any{"value": fmt.Sprintf("Bench zone %d", i), "version": version})
		raw, err := a.timed(ctx, "report_fact", http.MethodPut, base+"/facts/zone", body, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil || view.Draft == nil || view.RFP == nil {
			return fmt.Errorf("fact returned no draft or controls: %v", err)
		}
		version = view.Draft.Version
	}
	if _, err := a.timed(ctx, "report_read", http.MethodGet, base, nil, http.StatusOK); err != nil {
		return err
	}
	return nil
}

// monthlyReport times the project report on the plan's heavy fixture
// (docs/plans/2026-11-17-monthly-report.md, Task 8): 1,000 Program rows and
// 2,000 links, 1,000 invoices, 30 packages, 200 register items and a
// previous issue. Each round is Update, the read, keeping every carried
// comment and writing the summary, issue and the issued download.
func (a *apiClient) monthlyReport(ctx context.Context, st *store.Store, n int) error {
	project, lines, err := a.invoiceProject(ctx, "Project report benchmark")
	if err != nil {
		return err
	}
	for i := 0; i < 27; i++ {
		body, _ := json.Marshal(map[string]string{"kind": "services", "title": fmt.Sprintf("Consultant %02d", i)})
		if _, err := a.timed(ctx, "packages_write", http.MethodPost, "/projects/"+project+"/packages", body, http.StatusCreated); err != nil {
			return err
		}
	}
	if _, err := st.CreateProgram(ctx, benchOrg, project, programBenchCreate()); err != nil {
		return err
	}
	if _, err := seedLedger(ctx, st, project, lines, 1000); err != nil {
		return err
	}
	kinds := []string{"risk", "issue", "decision", "approval", "action"}
	now := time.Now()
	for i := 0; i < 200; i++ {
		kind := kinds[i%len(kinds)]
		target := now.AddDate(0, 0, i%60-20).Format("2006-01-02")
		body, _ := json.Marshal(delivery.Content{Kind: kind, Title: fmt.Sprintf("%s %d", kind, i), Status: delivery.DefaultStatus(kind), TargetDate: &target, Details: json.RawMessage(`{}`)})
		if _, err := a.timed(ctx, "delivery_write", http.MethodPost, "/projects/"+project+"/delivery", body, http.StatusCreated); err != nil {
			return err
		}
	}
	raw, err := a.timed(ctx, "report_write", http.MethodPost, "/projects/"+project+"/reports", []byte(`{"kind":"monthly"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	var r store.Report
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	base := "/reports/" + r.ID
	issue := func(version int64) (store.IssuedReport, error) {
		body, _ := json.Marshal(store.IssueOptions{Version: version, ReportingDate: now.Format("2006-01-02"), AcceptStale: true, StaleReason: "Benchmark fixture"})
		var out store.IssuedReport
		raw, err := a.timed(ctx, "report_issue", http.MethodPost, base+"/issue", body, http.StatusCreated)
		if err == nil {
			err = json.Unmarshal(raw, &out)
		}
		return out, err
	}
	raw, err = a.timed(ctx, "report_assemble_monthly", http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK)
	if err != nil {
		return err
	}
	var d store.ReportDraft
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	if _, err := issue(d.Version); err != nil {
		return err
	}
	for i := 0; i < max(n, 20); i++ {
		if _, err := a.timed(ctx, "report_assemble_monthly", http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK); err != nil {
			return err
		}
		raw, err := a.timed(ctx, "report_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view store.ReportView
		if err := json.Unmarshal(raw, &view); err != nil || view.Draft == nil {
			return fmt.Errorf("project report read: %v", err)
		}
		if view.PageFit == nil || !view.PageFit.Fits {
			return fmt.Errorf("the project report does not fit its four pages: %+v", view.PageFit)
		}
		version := view.Draft.Version
		edit := func(target, text string) error {
			body, _ := json.Marshal(map[string]any{"text": text, "version": version})
			raw, err := a.timed(ctx, "report_write", http.MethodPut, base+"/edits/"+url.PathEscape(target), body, http.StatusOK)
			if err != nil {
				return err
			}
			var saved store.ReportDraft
			err = json.Unmarshal(raw, &saved)
			version = saved.Version
			return err
		}
		for _, s := range view.Draft.Sections {
			for _, b := range s.Blocks {
				if b.Conflict {
					if err := edit(b.ID, b.Text); err != nil {
						return err
					}
				}
			}
		}
		if err := edit("comment:summary", fmt.Sprintf("Round %d: as the cost plan shows.", i)); err != nil {
			return err
		}
		issued, err := issue(version)
		if err != nil {
			return err
		}
		pdf, err := a.timed(ctx, "report_export", http.MethodGet, base+"/versions/"+issued.ID+"/file", nil, http.StatusOK)
		if err != nil {
			return err
		}
		if len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
			return fmt.Errorf("issued project report is not a PDF")
		}
	}
	return nil
}
