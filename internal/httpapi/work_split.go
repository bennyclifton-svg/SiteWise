package httpapi

import (
	"errors"
	"net/http"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func postWorkSplit(w http.ResponseWriter, r *http.Request, deps Deps) {
	mutateWorkTree(w, r, deps, true)
}
func postWorkRetire(w http.ResponseWriter, r *http.Request, deps Deps) {
	mutateWorkTree(w, r, deps, false)
}
func mutateWorkTree(w http.ResponseWriter, r *http.Request, deps Deps, split bool) {
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
	var body works.SplitRequest
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	for _, child := range body.Children {
		if child.PartID != "" && !uuidPattern.MatchString(child.PartID) {
			http.Error(w, "invalid part", http.StatusUnprocessableEntity)
			return
		}
	}
	var result any
	var err error
	if split {
		result, err = deps.Store.SplitWorkItem(r.Context(), session.OrgID, project, id, session.UserID, body)
	} else {
		err = deps.Store.RetireWorkItem(r.Context(), session.OrgID, project, id, session.UserID, body.Version)
		result = map[string]bool{"retired": true}
	}
	var referenced *store.WorkReferencedError
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.As(err, &referenced):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "referenced", "blockers": referenced.Blockers})
	case errors.Is(err, store.ErrVersionConflict), errors.Is(err, store.ErrWorkConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, store.ErrInvalidWork):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
	default:
		if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
		writeJSON(w, http.StatusOK, result)
	}
}
