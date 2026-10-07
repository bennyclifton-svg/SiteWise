package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"sitewise/internal/store"
	"sitewise/internal/works"
)

func getWorks(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	items, err := deps.Store.ReadWorks(r.Context(), session.OrgID, project)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	for i := range items {
		if deps.Knowledge != nil {
			sys, ok := deps.Knowledge.System(items[i].SystemID)
			items[i].Deprecated = !ok || sys.Status == "deprecated"
		}
	}
	writeJSON(w, http.StatusOK, struct {
		ProjectID string       `json:"project_id"`
		Items     []works.Item `json:"items"`
	}{project, items})
}

func postWork(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, project); errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	var body struct {
		PartID                string          `json:"part_id"`
		SystemID              string          `json:"system_id"`
		Action                string          `json:"action"`
		Title                 string          `json:"title"`
		ExistingCondition     string          `json:"existing_condition"`
		ExistingConditionNote string          `json:"existing_condition_note"`
		Target                works.Target    `json:"target"`
		Quantity              json.RawMessage `json:"quantity"`
		Unit                  *string         `json:"unit"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !uuidPattern.MatchString(body.PartID) {
		http.Error(w, "part_id is required", http.StatusUnprocessableEntity)
		return
	}
	quantity, msg := planningValue(body.Quantity)
	if msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	item := works.Item{PartID: body.PartID, SystemID: body.SystemID, Action: body.Action, Title: strings.TrimSpace(body.Title),
		ExistingCondition: body.ExistingCondition, ExistingConditionNote: body.ExistingConditionNote, Target: body.Target, Quantity: quantity, Unit: body.Unit}
	if err := works.Validate(item, deps.Knowledge); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	item, err := deps.Store.CreateWorkItem(r.Context(), session.OrgID, project, session.UserID, item)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrWorkConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, store.ErrInvalidWork):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
	default:
		if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
		writeJSON(w, http.StatusCreated, item)
	}
}

func patchWork(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project, id := r.PathValue("id"), r.PathValue("wi")
	if !uuidPattern.MatchString(project) || !uuidPattern.MatchString(id) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, project); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, "read failed", http.StatusInternalServerError)
		}
		return
	}
	var patch works.Patch
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &patch); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := deps.Store.PatchWorkItem(r.Context(), session.OrgID, project, id, session.UserID, patch)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "work item changed; reload before editing", http.StatusConflict)
	case errors.Is(err, store.ErrInvalidWork):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
	default:
		if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
		writeJSON(w, http.StatusOK, item)
	}
}
