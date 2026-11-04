package httpapi

import (
	"net/http"
	"time"

	"sitewise/internal/auth"
)

func endSession(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	if err := deps.Store.EndSession(r.Context(), session); err != nil {
		http.Error(w, "sign out failed", http.StatusInternalServerError)
		return
	}
	deps.streams.end(session.ID)
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookie, Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0), HttpOnly: true,
		Secure: deps.SecureCookie, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
