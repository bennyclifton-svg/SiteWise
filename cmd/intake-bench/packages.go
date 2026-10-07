package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func (a *apiClient) packages(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Package endpoint benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/packages"
	for key, value := range map[string]string{"hdr.building_class": "industrial", "hdr.work_type": "new",
		"det.heritage_status": "local_item", "det.bal": "BAL-40",
		"hdr.cond.planning": "da", "hdr.cond.environmental_sensitivity": "protected_habitat"} {
		body, _ := json.Marshal(map[string]string{"value": value})
		if _, err := a.timed(ctx, "profile_edit", http.MethodPut, "/projects/"+project+"/profile/"+key, body, http.StatusOK); err != nil {
			return err
		}
	}
	for i := 0; i < n; i++ {
		body, _ := json.Marshal(map[string]any{"kind": "services", "title": fmt.Sprintf("Consultant %d", i), "novation": i%2 == 0})
		raw, err := a.timed(ctx, "packages_write", http.MethodPost, base, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var p procurement.Package
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		if p.ID == "" || len(p.Stages) == 0 {
			return fmt.Errorf("package benchmark response has no stages")
		}
		body, _ = json.Marshal(map[string]any{"version": p.Version, "title": p.Title + " revised"})
		if _, err := a.timed(ctx, "packages_write", http.MethodPatch, base+"/"+p.ID, body, http.StatusOK); err != nil {
			return err
		}
		body, _ = json.Marshal(map[string]any{"version": p.Stages[0].Version, "label": "Concept stage revised"})
		if _, err := a.timed(ctx, "packages_write", http.MethodPatch, base+"/"+p.ID+"/stages/"+p.Stages[0].ID, body, http.StatusOK); err != nil {
			return err
		}
		body = []byte(`{"stage_id":"completion","label":"Handover","ordinal":5}`)
		if _, err := a.timed(ctx, "packages_write", http.MethodPost, base+"/"+p.ID+"/stages", body, http.StatusCreated); err != nil {
			return err
		}
		raw, err = a.timed(ctx, "packages_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var overview struct {
			Suggestions []procurement.PackageSuggestion `json:"suggestions"`
		}
		if err := json.Unmarshal(raw, &overview); err != nil {
			return err
		}
		if len(overview.Suggestions) != 12 {
			return fmt.Errorf("package benchmark expected twelve baseline/complexity suggestions, got %d", len(overview.Suggestions))
		}
		matched := 0
		for _, suggestion := range overview.Suggestions {
			if !suggestion.Draft || suggestion.ReviewStatus != "proposed" {
				return fmt.Errorf("package benchmark lost provisional status")
			}
			matched += len(suggestion.MatchedFields)
		}
		if matched != 4 {
			return fmt.Errorf("package benchmark lost complexity reasons: %d", matched)
		}
		if err := a.packageScope(ctx, base+"/"+p.ID+"/scope"); err != nil {
			return err
		}
	}
	return nil
}

func (a *apiClient) packageScope(ctx context.Context, path string) error {
	body := []byte(`{"item_kind":"obligation","clause_id":"cl.rfp-draft-status","clause_version":1,"inclusion":"included"}`)
	raw, err := a.timed(ctx, "package_scope_write", http.MethodPost, path, body, http.StatusCreated)
	if err != nil {
		return err
	}
	var item store.ScopeItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return err
	}
	if item.ID == "" || !item.Provisional {
		return fmt.Errorf("scope benchmark lost draft provenance")
	}
	body = []byte(`{"version":1,"deliverable":"Coordinated response"}`)
	if _, err := a.timed(ctx, "package_scope_write", http.MethodPatch, path+"/"+item.ID, body, http.StatusOK); err != nil {
		return err
	}
	raw, err = a.timed(ctx, "package_scope_read", http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return err
	}
	var view struct {
		Items []store.ScopeItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		return err
	}
	if len(view.Items) != 1 || view.Items[0].Version != 2 {
		return fmt.Errorf("scope benchmark edit missing")
	}
	_, err = a.timed(ctx, "package_scope_write", http.MethodPatch, path+"/"+item.ID, []byte(`{"version":2,"retired":true}`), http.StatusOK)
	return err
}
