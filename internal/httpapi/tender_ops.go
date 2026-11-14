package httpapi

import (
	"errors"
	"net/http"

	"sitewise/internal/store"
)

// postTenderOps applies a batch of named edits to one package's tender
// tables (contract C6) and answers with the recomputed table. No operation
// calls Jev or the model.
func postTenderOps(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in struct {
		Version int64            `json:"version"`
		Ops     []store.TenderOp `json:"ops"`
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	change, err := deps.Store.ApplyTenderOps(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID, in.Version, in.Ops)
	if err != nil {
		if errors.Is(err, store.ErrVersionConflict) {
			http.Error(w, "This table changed. It has been reloaded.", http.StatusConflict)
			return
		}
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	t, err := deps.Store.ReadPriceTable(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	award, err := deps.Store.ReadAward(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"))
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	body := tableBody(t, award, deps.TenderReader != nil)
	body.Change = change
	writeJSON(w, http.StatusOK, body)
}

// postChangeUndo puts back one applied change
// (docs/plans/2026-11-14-change-log-undo-operations.md, task 4). Only the
// user who made it may undo it.
func postChangeUndo(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "change") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	areas, err := deps.Store.UndoChange(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("change"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrChangeUndone):
		http.Error(w, "already undone", http.StatusConflict)
		return
	case errors.Is(err, store.ErrChangedSince):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "changed_since", "message": store.MsgUndoChanged})
		return
	case err != nil:
		deps.Log.Printf("change undo failed: %v", err)
		http.Error(w, "undo failed", http.StatusInternalServerError)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	if areas == nil {
		areas = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"undone": true, "areas": areas})
}
