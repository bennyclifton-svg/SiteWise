package httpapi

import (
	"errors"
	"net/http"

	"sitewise/internal/store"
)

type awardLineJSON struct {
	store.AwardLineView
	Amount moneyField `json:"amount"`
}

type awardJSON struct {
	store.AwardView
	Budget moneyField      `json:"budget"`
	Total  moneyField      `json:"total"`
	Lines  []awardLineJSON `json:"lines"`
}

func awardBody(v store.AwardView) awardJSON {
	out := awardJSON{AwardView: v, Budget: cents(v.Budget), Total: centsValue(v.Total), Lines: []awardLineJSON{}}
	for _, l := range v.Lines {
		out.Lines = append(out.Lines, awardLineJSON{AwardLineView: l, Amount: centsValue(l.Cents)})
	}
	return out
}

func awardInput(w http.ResponseWriter, r *http.Request, deps Deps) (store.Session, store.AwardInput, bool) {
	var in store.AwardInput
	if !pathIDs(w, r, "pkg") {
		return store.Session{}, in, false
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return session, in, false
	}
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil || !uuidPattern.MatchString(in.SubmissionID) {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return session, in, false
	}
	return session, in, true
}

// postAwardPreview checks the PM's lines and shows the parent and children
// that confirming would write. Nothing is written.
func postAwardPreview(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, in, ok := awardInput(w, r, deps)
	if !ok {
		return
	}
	v, err := deps.Store.PreviewAward(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), in)
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, awardBody(v))
}

func postAward(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, in, ok := awardInput(w, r, deps)
	if !ok {
		return
	}
	if _, err := deps.Store.Award(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID, in); err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writePriceTable(w, r, deps, session, http.StatusCreated)
}

func deleteAward(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !pathIDs(w, r, "pkg") {
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	err := deps.Store.UndoAward(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("pkg"), session.UserID)
	if errors.Is(err, store.ErrAwardLocked) {
		http.Error(w, "An invoice is allocated to this award's lines, so it can no longer be undone. Re-map the invoice first.", http.StatusConflict)
		return
	}
	if err != nil {
		procurementError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writePriceTable(w, r, deps, session, http.StatusOK)
}
