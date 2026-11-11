package httpapi

import (
	"net/http"
)

// postInvitation names a firm on a package row before its quote arrives:
// a directory firm by id, or a typed name
// (docs/plans/2026-11-11-procurement-invited-firms.md). No Jev.
func postInvitation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	firm, name, ok := namedFirm(w, r, deps)
	if !ok {
		return
	}
	inv, err := deps.Store.InviteFirm(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), firm, name)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, inv)
}

// deleteInvitation takes an invited firm off the row. A proposal the firm
// sent stays where it is.
func deleteInvitation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg", "firm") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	if err := deps.Store.RemoveInvitation(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), r.PathValue("firm")); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// putInvitation corrects an invited firm's name, or swaps in the directory
// firm the PM meant, in the same cell. No Jev.
func putInvitation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg", "firm") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	firm, name, ok := namedFirm(w, r, deps)
	if !ok {
		return
	}
	inv, err := deps.Store.CorrectInvitation(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), r.PathValue("firm"), firm, name)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, inv)
}

// namedFirm reads {firm_id} or {name}: exactly one.
func namedFirm(w http.ResponseWriter, r *http.Request, deps Deps) (string, string, bool) {
	var body struct {
		FirmID string  `json:"firm_id"`
		Name   *string `json:"name"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &body) != nil || (body.FirmID == "") == (body.Name == nil) || body.FirmID != "" && !uuidPattern.MatchString(body.FirmID) {
		http.Error(w, "firm_id or name is required", http.StatusBadRequest)
		return "", "", false
	}
	if body.Name == nil {
		return body.FirmID, "", true
	}
	return "", *body.Name, true
}
