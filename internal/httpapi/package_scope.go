package httpapi

import (
	"net/http"
	"sitewise/internal/store"
)

func getPackageScope(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	if !uuidPattern.MatchString(r.PathValue("id")) || !uuidPattern.MatchString(r.PathValue("pkg")) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	items, err := deps.Store.ReadPackageScope(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		packageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items []store.ScopeItem `json:"items"`
	}{items})
}

func postPackageScope(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	var body store.ScopeInput
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := deps.Store.CreatePackageScope(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID, body)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, item)
}

func patchPackageScope(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	if !uuidPattern.MatchString(r.PathValue("scope")) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body store.ScopePatch
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := deps.Store.PatchPackageScope(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), r.PathValue("scope"), session.UserID, body)
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, item)
}
