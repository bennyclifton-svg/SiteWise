package httpapi

import (
	"errors"
	"net/http"
	"sitewise/internal/delivery"
	"sitewise/internal/store"
)

func deliverySession(w http.ResponseWriter, r *http.Request, deps Deps, write bool) (store.Session, bool) {
	if write && !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return store.Session{}, false
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return session, false
	}
	if !uuidPattern.MatchString(r.PathValue("id")) {
		http.Error(w, "not found", http.StatusNotFound)
		return session, false
	}
	if item := r.PathValue("item"); item != "" && !uuidPattern.MatchString(item) {
		http.Error(w, "not found", http.StatusNotFound)
		return session, false
	}
	if _, err := deps.Store.GetProject(r.Context(), session.OrgID, r.PathValue("id")); err != nil {
		deliveryError(w, err)
		return session, false
	}
	return session, true
}

func getDelivery(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	items, err := deps.Store.ReadDelivery(r.Context(), session.OrgID, r.PathValue("id"))
	if err != nil {
		deliveryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items []delivery.Item `json:"items"`
	}{items})
}

func postDelivery(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var input store.DeliveryInput
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &input); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := deps.Store.CreateDelivery(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, input)
	if err != nil {
		deliveryError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusCreated, item)
}

func patchDelivery(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var patch store.DeliveryPatch
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &patch); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := deps.Store.PatchDelivery(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("item"), session.UserID, patch)
	if err != nil {
		deliveryError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, item)
}

func deliveryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "delivery record changed", http.StatusConflict)
	case errors.Is(err, store.ErrInvalidDelivery), errors.Is(err, store.ErrInvalidPackage):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		http.Error(w, "delivery operation failed", http.StatusInternalServerError)
	}
}
