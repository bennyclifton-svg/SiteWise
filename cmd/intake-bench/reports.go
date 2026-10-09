package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	body, _ = json.Marshal(map[string]int64{"version": version})
	if _, err := a.timed(ctx, "report_setting", http.MethodDelete, key, body, http.StatusOK); err != nil {
		return err
	}
	if _, err := a.timed(ctx, "report_read", http.MethodGet, base, nil, http.StatusOK); err != nil {
		return err
	}
	return nil
}
