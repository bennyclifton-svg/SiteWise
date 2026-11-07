package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Combined engagements, slice 1 (docs/plans/2026-11-07-combined-engagements-slice1.md).
// Combine, uncombine and remove from group are package writes: code only,
// no Jev call, one store transaction each. Rename is the package title patch.

func postPackageCombine(w http.ResponseWriter, r *http.Request, deps Deps) {
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
	// SubmissionID, when given, is an unplaced proposal the group is made
	// for (slice 5): it is placed on the group in the same transaction.
	var body struct {
		PackageIDs   []string `json:"package_ids"`
		Title        string   `json:"title"`
		SubmissionID string   `json:"submission_id"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || len(body.PackageIDs) > 50 || len(body.Title) > 1000 || body.SubmissionID != "" && !uuidPattern.MatchString(body.SubmissionID) {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	for _, id := range body.PackageIDs {
		if !uuidPattern.MatchString(id) {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
	}
	var group procurement.Package
	var err error
	if body.SubmissionID != "" {
		group, err = deps.Store.CombineForProposal(r.Context(), session.OrgID, project, session.UserID, body.PackageIDs, strings.TrimSpace(body.Title), body.SubmissionID)
	} else {
		group, err = deps.Store.CombinePackages(r.Context(), session.OrgID, project, session.UserID, body.PackageIDs, strings.TrimSpace(body.Title))
	}
	if errors.Is(err, store.ErrConflict) {
		http.Error(w, "this proposal is already placed", http.StatusConflict)
		return
	}
	if err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, group)
}

// postPackageUncombine releases every member of the group {pkg}.
func postPackageUncombine(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	if err := deps.Store.UncombinePackages(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg")); err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// postPackageUngroup takes the member {pkg} out of its group.
func postPackageUngroup(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := packageEditSession(w, r, deps)
	if !ok {
		return
	}
	if err := deps.Store.RemoveFromGroup(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg")); err != nil {
		packageError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	w.WriteHeader(http.StatusNoContent)
}
