package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// stageSync measures stage_sync_write (program plan 11.1): Update on one
// RFP draft or one package's stages after a stage rename in Program, the
// recomputed document returned. Samples alternate between the two
// documents. The RFP is created through Procurement, which also creates
// the program (7.6); renames and setup are untimed.
func (a *apiClient) stageSync(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Stage sync benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project
	raw, err := a.call(ctx, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	var pkg struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return err
	}
	if raw, err = a.call(ctx, http.MethodPost, base+"/procurement/create", []byte(`{"package_id":"`+pkg.ID+`"}`), http.StatusCreated); err != nil {
		return err
	}
	var created struct {
		Report struct {
			ID string `json:"id"`
		} `json:"report"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		return err
	}
	if raw, err = a.call(ctx, http.MethodGet, base+"/program", nil, http.StatusOK); err != nil {
		return err
	}
	var v struct {
		Program struct {
			ID            string `json:"id"`
			Version       int64  `json:"version"`
			StageRevision int64  `json:"stage_revision"`
		} `json:"program"`
		Activities []struct {
			ID               string `json:"id"`
			Kind             string `json:"kind"`
			CatalogueStageID string `json:"catalogue_stage_id"`
		} `json:"activities"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	stage := ""
	for _, act := range v.Activities {
		if act.Kind == "stage" && act.CatalogueStageID == "concept_design" {
			stage = act.ID
		}
	}
	if stage == "" {
		return fmt.Errorf("no concept design stage")
	}
	docs := []string{"/reports/" + created.Report.ID + "/stage-sync", base + "/packages/" + pkg.ID + "/stage-sync"}
	for i := 0; i < 2*n; i++ {
		ops, _ := json.Marshal(map[string]any{"program_id": v.Program.ID, "version": v.Program.Version,
			"ops": []map[string]any{{"op": "update", "id": stage, "label": []string{"Sketch design", "Concept design"}[i%2]}}})
		if raw, err = a.call(ctx, http.MethodPost, base+"/program/ops", ops, http.StatusOK); err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]any{"choice": "update", "revision": v.Program.StageRevision})
		out, err := a.timed(ctx, "stage_sync_write", http.MethodPost, docs[i%2], body, http.StatusOK)
		if err != nil {
			return err
		}
		var sync struct {
			Offer bool `json:"offer"`
		}
		if err := json.Unmarshal(out, &sync); err != nil || sync.Offer {
			return fmt.Errorf("update left an offer: %s", out)
		}
	}
	return nil
}
