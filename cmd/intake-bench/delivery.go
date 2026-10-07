package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/delivery"
)

func (a *apiClient) delivery(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Delivery endpoint benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/delivery"
	for i := 0; i < n; i++ {
		body, _ := json.Marshal(map[string]any{"kind": "approval", "title": fmt.Sprintf("Approval %d", i), "target_date": "2026-12-01", "details": map[string]string{"authority": "Authority"}})
		raw, err := a.timed(ctx, "delivery_write", http.MethodPost, base, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var item delivery.Item
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.ID == "" || item.Status != "not_submitted" {
			return fmt.Errorf("invalid delivery create response")
		}
		raw, err = a.timed(ctx, "delivery_write", http.MethodPatch, base+"/"+item.ID, []byte(`{"version":1,"status":"submitted","target_date":null}`), http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.Version != 2 || item.TargetDate != nil || item.Status != "submitted" {
			return fmt.Errorf("invalid delivery edit response")
		}
		raw, err = a.timed(ctx, "delivery_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view struct {
			Items []delivery.Item `json:"items"`
		}
		if err := json.Unmarshal(raw, &view); err != nil {
			return err
		}
		if len(view.Items) != i+1 {
			return fmt.Errorf("delivery list omitted records")
		}
	}
	return nil
}
