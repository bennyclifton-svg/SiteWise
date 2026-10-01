package httpapi

import (
	"context"
	"net/http"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/store"
)

// DevOrgID is the fixed organisation local sign-in uses, so every visit lands
// in the same projects.
const DevOrgID = "de7e1000-0000-4000-8000-000000000001"

const devEmail = "owner@sitewise.local"

// devLogin signs the visitor in as the local owner and sends them to the app.
// It mints a one-minute owner invite and consumes it, the same path an emailed
// invite takes, so no session is made any other way. It is only mounted when
// serve is started with -dev-login on a loopback address outside production.
func devLogin(st *store.Store, secure bool, logger interface{ Printf(string, ...any) }) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, err := devSession(ctx, st)
		if err != nil {
			logger.Printf("dev login failed: %v", err)
			http.Error(w, "dev login failed", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookie,
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func devSession(ctx context.Context, st *store.Store) (store.Session, error) {
	if err := st.EnsureOrg(ctx, DevOrgID, "Local dev"); err != nil {
		return store.Session{}, err
	}
	raw, err := auth.CreateInvite(ctx, st, &auth.Sink{}, DevOrgID, devEmail, "owner", time.Now().Add(time.Minute), nil)
	if err != nil {
		return store.Session{}, err
	}
	return st.ConsumeInvite(ctx, auth.HashToken(raw), time.Now())
}
