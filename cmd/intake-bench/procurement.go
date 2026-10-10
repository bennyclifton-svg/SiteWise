package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Exercise the actual table routes, including atomic suggestion acceptance,
// rather than inferring their timings from the legacy package/report routes.
func (a *apiClient) procurement(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		project, err := a.createProject(ctx, "Procurement benchmark")
		if err != nil {
			return err
		}
		base := "/projects/" + project
		raw, err := a.timed(ctx, "packages_write", http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"consultant.structural"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var pkg procurement.Package
		if err := json.Unmarshal(raw, &pkg); err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"package_id": pkg.ID})
		var first store.Report
		for _, status := range []int{http.StatusCreated, http.StatusOK} {
			raw, err = a.timed(ctx, "report_write", http.MethodPost, base+"/procurement/create", body, status)
			if err != nil {
				return err
			}
			var result struct {
				Report store.Report `json:"report"`
			}
			if err := json.Unmarshal(raw, &result); err != nil {
				return err
			}
			if result.Report.ID == "" || result.Report.Kind != "rfp" || (first.ID != "" && first.ID != result.Report.ID) {
				return fmt.Errorf("procurement benchmark lost idempotent RFP")
			}
			first = result.Report
		}
		for key, value := range map[string]string{"hdr.building_class": "residential", "hdr.work_type": "new"} {
			body, _ := json.Marshal(map[string]string{"value": value})
			if _, err := a.timed(ctx, "profile_edit", http.MethodPut, base+"/profile/"+key, body, http.StatusOK); err != nil {
				return err
			}
		}
		raw, err = a.timed(ctx, "packages_read", http.MethodGet, base+"/packages", nil, http.StatusOK)
		if err != nil {
			return err
		}
		var overview store.PackageOverview
		if err := json.Unmarshal(raw, &overview); err != nil {
			return err
		}
		key := ""
		for _, suggestion := range overview.Suggestions {
			if suggestion.Title == "Architect" {
				key = suggestion.Key
			}
		}
		if key == "" {
			return fmt.Errorf("procurement benchmark missing Architect suggestion")
		}
		body, _ = json.Marshal(map[string]string{"proposal_key": key})
		if _, err := a.timed(ctx, "report_write", http.MethodPost, base+"/procurement/create", body, http.StatusCreated); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "report_read", http.MethodGet, base+"/reports", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}
