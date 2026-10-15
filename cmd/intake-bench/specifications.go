package main

import (
	"context"
	"encoding/json"
	"net/http"

	"sitewise/internal/specifications"
)

func (a *apiClient) specifications(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Specification benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/specifications"
	// Match the golden schedule's row scale, using synthetic authored values.
	// Repeated issues also exercise growth of the immutable issue history.
	for i := 0; i < max(n, 131); i++ {
		raw, err := a.timed(ctx, "specification_write", http.MethodPost, base+"/types", []byte(`{"title":"Synthetic finish","kind":"finish"}`), http.StatusCreated)
		if err != nil {
			return err
		}
		var item specifications.Type
		if err = json.Unmarshal(raw, &item); err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]any{"type_id": item.ID, "location": "Office", "action": "repair", "inclusion": "included", "selection": "undecided"})
		if _, err = a.timed(ctx, "specification_write", http.MethodPost, base+"/applications", body, http.StatusCreated); err != nil {
			return err
		}
		for _, detail := range []struct{ key, text, meaning string }{{"finish", "Low sheen", "selection"}, {"installation", "Prepare the existing surface and protect adjacent retained finishes. Refer to the coordinated drawings for the application extent.", "requirement"}, {"warranty", "Supplier warranty terms remain subject to confirmation before procurement.", "requirement"}} {
			body, _ = json.Marshal(map[string]any{"type_id": item.ID, "key": detail.key, "meaning": detail.meaning, "value": map[string]string{"state": "known", "text": detail.text}})
			raw, err = a.timed(ctx, "specification_write", http.MethodPost, base+"/assertions", body, http.StatusCreated)
			if err != nil {
				return err
			}
			var assertion specifications.Assertion
			if err = json.Unmarshal(raw, &assertion); err != nil {
				return err
			}
			if _, err = a.timed(ctx, "specification_write", http.MethodPost, base+"/assertions/"+assertion.ID+"/review", []byte(`{"version":1,"status":"accepted"}`), http.StatusOK); err != nil {
				return err
			}
		}
		raw, err = a.timed(ctx, "specification_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view struct {
			Snapshot specifications.Snapshot `json:"snapshot"`
		}
		if err = json.Unmarshal(raw, &view); err != nil {
			return err
		}
		body, _ = json.Marshal(map[string]any{"title": "Synthetic schedule", "purpose": "preliminary", "revision": view.Snapshot.Revision})
		raw, err = a.timed(ctx, "specification_issue", http.MethodPost, base+"/issues", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var issue specifications.Issue
		if err = json.Unmarshal(raw, &issue); err != nil {
			return err
		}
		if _, err = a.timed(ctx, "specification_export", http.MethodGet, base+"/issues/"+issue.ID+"/export?format=pdf", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}
