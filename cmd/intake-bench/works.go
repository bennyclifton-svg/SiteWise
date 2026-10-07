package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func (a *apiClient) workItems(ctx context.Context, st *store.Store, n int) error {
	project, err := a.createProject(ctx, "Works endpoint benchmark")
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		part, err := st.CreatePart(ctx, benchOrg, project, fmt.Sprintf("Work area %d", i), "part", "")
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"part_id": part.ID, "system_id": "structure", "action": "investigate"})
		created, err := a.timed(ctx, "works_write", http.MethodPost, "/projects/"+project+"/works", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var item works.Item
		if err := json.Unmarshal(created, &item); err != nil {
			return err
		}
		patch, _ := json.Marshal(map[string]any{"version": item.Version, "title": "Inspect existing structure", "quantity": 12.5, "unit": "m2"})
		if _, err := a.timed(ctx, "works_write", http.MethodPatch, "/projects/"+project+"/works/"+item.ID, patch, http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "works_read", http.MethodGet, "/projects/"+project+"/works", nil, http.StatusOK); err != nil {
			return err
		}
		raw, err := a.timed(ctx, "gap_check", http.MethodGet, "/projects/"+project+"/gaps", nil, http.StatusOK)
		if err != nil {
			return err
		}
		var gaps struct {
			Items []procurement.Gap `json:"items"`
		}
		if err := json.Unmarshal(raw, &gaps); err != nil {
			return err
		}
		if len(gaps.Items) != i+1 {
			return fmt.Errorf("gap benchmark expected %d gaps, got %d", i+1, len(gaps.Items))
		}
	}
	if err := a.workTree(ctx, st, n); err != nil {
		return err
	}
	return a.layoutWork(ctx, st, n)
}
