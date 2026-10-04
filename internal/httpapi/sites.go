package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"sitewise/internal/store"
)

// Sites: a project is an intervention on one site, the lasting record of what
// exists (migration 011). Version 1 shows nothing new; these routes let a
// later screen name the site. Neither waits on Jev.

func getProjectSite(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID := r.PathValue("id")
	if !uuidPattern.MatchString(projectID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	site, err := deps.Store.ProjectSite(r.Context(), session.OrgID, projectID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

// patchSite edits a site's label, address or lot. Body:
// {"version": n, "label"?: "...", "address"?: "...", "lot"?: "..."}.
// A stale version is 409 with the current version.
func patchSite(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	siteID := r.PathValue("site")
	if !uuidPattern.MatchString(siteID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		Version *int64  `json:"version"`
		Label   *string `json:"label"`
		Address *string `json:"address"`
		Lot     *string `json:"lot"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.Version == nil {
		http.Error(w, "a JSON body with version is required", http.StatusBadRequest)
		return
	}
	if msg := siteFieldsInvalid(&body.Label, &body.Address, &body.Lot); msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	site, err := deps.Store.UpdateSite(r.Context(), session.OrgID, siteID, *body.Version, body.Label, body.Address, body.Lot)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "version_conflict", "current_version": site.Version, "site": site})
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusOK, site)
	}
}

// siteFieldsInvalid trims the fields in place and checks the column limits.
func siteFieldsInvalid(label, address, lot **string) string {
	trim := func(p **string) {
		if *p != nil {
			v := strings.TrimSpace(**p)
			*p = &v
		}
	}
	trim(label)
	trim(address)
	trim(lot)
	if *label != nil && (**label == "" || len([]rune(**label)) > 120) {
		return "label must be 1 to 120 characters"
	}
	if *address != nil && len([]rune(**address)) > 200 {
		return "address must be at most 200 characters"
	}
	if *lot != nil && len([]rune(**lot)) > 80 {
		return "lot must be at most 80 characters"
	}
	return ""
}
