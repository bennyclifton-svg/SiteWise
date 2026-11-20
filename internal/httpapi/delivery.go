package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"sitewise/internal/consent"
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
	q := store.RegisterQuery{Kind: r.URL.Query().Get("kind"), System: r.URL.Query().Get("system")}
	if q.Kind != "" && len(delivery.Statuses(q.Kind)) == 0 {
		http.Error(w, "unknown kind", http.StatusUnprocessableEntity)
		return
	}
	view, err := deps.Store.ReadRegister(r.Context(), session.OrgID, r.PathValue("id"), q, todaySydney())
	if err != nil {
		deliveryError(w, err)
		return
	}
	// SendOff is why the RFI sheet's Send is off, in the send route's own
	// words, so the button says so before it is pressed. Other refusals
	// (an unconfirmed email, the daily cap) come back when it is pressed.
	out := struct {
		store.RegisterView
		SendOff string `json:"send_off,omitempty"`
	}{RegisterView: view}
	if deps.Outbox == nil {
		out.SendOff = "Sending mail is not set up on this server."
	}
	writeJSON(w, http.StatusOK, out)
}

// putDeliverySettings sets the project's RFI response time
// (operation delivery.settings).
func putDeliverySettings(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		Days int `json:"rfi_response_days"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.SetDeliverySettings(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, body.Days); err != nil {
		deliveryError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, body)
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

// postDeliveryImport is Read conditions: POST /projects/{id}/delivery/imports
// {document_id} (docs/plans/2026-11-17-consent-conditions.md, Task 5; index
// agreement 4, the one read route into the register). Go checks the
// document's stored text is a notice of determination and queues the
// background read; nothing here calls Jev while the PM waits. It is the
// operation delivery.read_notice, which chat prepares for the PM's click.
func postDeliveryImport(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		DocumentID string `json:"document_id"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || !uuidPattern.MatchString(body.DocumentID) {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	project := r.PathValue("id")
	pages, err := deps.Store.ConsentDocument(r.Context(), session.OrgID, project, body.DocumentID)
	if err != nil {
		deliveryError(w, err)
		return
	}
	if msg := consentRefusal(pages); msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	switch err := deps.Store.QueueConsent(r.Context(), session.OrgID, project, body.DocumentID, session.UserID); {
	case errors.Is(err, store.ErrConsentBusy):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "Wait for the document to finish filing.", http.StatusUnprocessableEntity)
		return
	case err != nil:
		deliveryError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"document_id": body.DocumentID, "status": "queued"})
}

// consentRefusal is why a document cannot be read for conditions, in one
// plain line, or "".
func consentRefusal(pages []store.SourcePage) string {
	if len(pages) == 0 {
		return "No text has been read from this document yet."
	}
	in := make([]consent.Page, len(pages))
	for i, p := range pages {
		in[i] = consent.Page{Page: p.Page, Text: p.Text}
	}
	if !consent.Detect(in) {
		return "No numbered conditions found"
	}
	if n := len(consent.Split(in).Conditions); n > consent.MaxConditions {
		return fmt.Sprintf("The notice has %d numbered conditions, more than %d; split the document.", n, consent.MaxConditions)
	}
	return ""
}
