package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"sitewise/internal/delivery"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// This synthetic rooftop-plant scenario measures decision endpoints, not
// extraction quality or the cost of generating the larger proposal projection.
func (a *apiClient) proposals(ctx context.Context, st *store.Store, cat *knowledge.Catalog, thresholds profile.Thresholds, reading profile.ReadPolicy, kind string, n int) error {
	project, err := a.createProject(ctx, "Proposal decision endpoint benchmark")
	if err != nil {
		return err
	}
	part, err := st.EnsureWholePart(ctx, benchOrg, project)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"part_id": part.ID, "system_id": "mechanical.air-conditioning", "action": "new", "title": "Rooftop plant"})
	if _, err := a.timed(ctx, "works_write", http.MethodPost, "/projects/"+project+"/works", body, http.StatusCreated); err != nil {
		return err
	}
	build := store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion, ThresholdsVersion: thresholds.Version, ReadKinds: reading.Kinds(), Compute: func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: thresholds, Read: reading}, cat)
	}}
	evaluator, err := works.NewEvaluator(cat)
	if err != nil {
		return err
	}
	if err := st.WithProfile(build).RebuildProposals(ctx, benchOrg, project, evaluator); err != nil {
		return err
	}
	listPath := "/projects/" + project + "/proposals?show=all"
	read := func() (store.ProposalView, error) {
		raw, err := a.timed(ctx, "proposals_read", http.MethodGet, listPath, nil, http.StatusOK)
		if err != nil {
			return store.ProposalView{}, err
		}
		var view struct {
			Items []store.ProposalView `json:"items"`
		}
		if err := json.Unmarshal(raw, &view); err != nil {
			return store.ProposalView{}, err
		}
		for _, p := range view.Items {
			if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" {
				return p, nil
			}
		}
		return store.ProposalView{}, fmt.Errorf("proposal benchmark missing supported-load investigation")
	}
	p, err := read()
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/proposals/" + url.PathEscape(p.Key)
	body, _ = json.Marshal(map[string]string{"inputs_fingerprint": p.InputsFingerprint})
	createdID := ""
	acceptPath, undoPath := "proposals_accept", "proposals_undo"
	if kind != "investigation" {
		acceptPath, undoPath = "proposals_accept_delivery", "proposals_undo_delivery"
	}
	for i := 0; i < n; i++ {
		for _, decision := range []string{"dismiss", "accept"} {
			// Ordinary writes now rebuild the real catalogue. Reapply this
			// synthetic acceptance fixture outside the endpoint's timing sample.
			if kind != "investigation" {
				if err := st.WithProfile(build).RebuildProposals(ctx, benchOrg, project, evaluator); err != nil {
					return err
				}
				current, err := read()
				if err != nil {
					return err
				}
				body, _ = json.Marshal(map[string]string{"inputs_fingerprint": current.InputsFingerprint})
			}

			path := "proposals_dismiss"
			if decision == "accept" {
				path = acceptPath
			}
			raw, err := a.timed(ctx, path, http.MethodPost, base+"/"+decision, body, http.StatusOK)
			if err != nil {
				return err
			}
			if decision == "accept" && kind != "investigation" {
				var item delivery.Item
				if err := json.Unmarshal(raw, &item); err != nil {
					return err
				}
				wantKind, wantStatus := "approval", "not_submitted"
				if kind == "hold_point" {
					wantKind, wantStatus = "milestone", "planned"
				}
				if item.ID == "" || item.Kind != wantKind || item.Status != wantStatus || item.ReviewStatus != "accepted_for_planning" || (createdID != "" && item.ID != createdID) {
					return fmt.Errorf("proposal benchmark lost delivery state or identity")
				}
				createdID = item.ID
			} else if decision == "accept" {
				var item works.Item
				if err := json.Unmarshal(raw, &item); err != nil {
					return err
				}
				if item.ID == "" || item.Action != "investigate" || item.ReviewStatus != "accepted_for_planning" || (createdID != "" && item.ID != createdID) {
					return fmt.Errorf("proposal benchmark acceptance lost action, review state or stable identity")
				}
				createdID = item.ID
			}
			current, err := read()
			if err != nil {
				return err
			}
			want := "dismissed"
			if decision == "accept" {
				want = "accepted"
			}
			if current.Decision == nil || current.Decision.Decision != want {
				return fmt.Errorf("proposal benchmark missing saved %s decision", want)
			}
			undo, _ := json.Marshal(map[string]int64{"version": current.Decision.Version})
			if _, err := a.timed(ctx, undoPath, http.MethodDelete, base+"/decision", undo, http.StatusNoContent); err != nil {
				return err
			}
			current, err = read()
			if err != nil {
				return err
			}
			if current.Decision != nil || current.State != "open" {
				return fmt.Errorf("proposal benchmark undo retained the decision")
			}
			if kind != "investigation" {
				items, err := st.ReadDelivery(ctx, benchOrg, project)
				if err != nil {
					return err
				}
				for _, item := range items {
					if item.ID == createdID {
						return fmt.Errorf("proposal undo left delivery item active")
					}
				}
				continue
			}
			items, err := st.ReadWorks(ctx, benchOrg, project)
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.ID == createdID {
					return fmt.Errorf("proposal benchmark undo left created work active")
				}
			}
		}
	}
	return nil
}
