package httpapi

import "net/http"

func visualIdentity(w http.ResponseWriter, r *http.Request, deps Deps) {
	if r.Method == http.MethodPost && !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", 403)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	doc, ok := visibleDocument(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		if err := deps.Store.QueueVisualIdentity(r.Context(), session.OrgID, doc.ID); err != nil {
			http.Error(w, "Image review is available for filed PDFs only.", 409)
			return
		}
	}
	review, err := deps.Store.VisualIdentity(r.Context(), session.OrgID, doc.ID)
	if err != nil {
		http.Error(w, "Could not load image review. Try again.", 500)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, review)
}
