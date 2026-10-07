package httpapi

import (
	"net/http"
	"sitewise/internal/store"
)

func getDeliveryDependencies(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	items, err := d.Store.ReadDeliveryDependencies(r.Context(), s.OrgID, r.PathValue("id"))
	if err != nil {
		deliveryError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func writeDeliveryDependency(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in store.DeliveryDependencyInput
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	if !uuidPattern.MatchString(in.PredecessorID) || !uuidPattern.MatchString(in.SuccessorID) {
		http.Error(w, "invalid record", 422)
		return
	}
	if err := d.Store.WriteDeliveryDependency(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, in, r.Method == "DELETE"); err != nil {
		deliveryError(w, err)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
