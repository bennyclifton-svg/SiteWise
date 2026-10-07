package httpapi

import (
	"net/http"
	"sitewise/internal/procurement"
)

func getGaps(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	items, err := deps.Store.ReadGaps(r.Context(), session.OrgID, project)
	if err != nil {
		packageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		ProjectID string            `json:"project_id"`
		Items     []procurement.Gap `json:"items"`
	}{project, items})
}
