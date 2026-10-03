package httpapi

import (
	"context"
	"net/http"
	"strings"
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

// Local testing should work from a bookmarked project, not only /dev/login.
// This is mounted only behind the existing loopback-only DevLogin option.
// API calls and assets never establish a session on their own.
func localAppSession(next http.Handler, st *store.Store, secure bool, logger interface{ Printf(string, ...any) }) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Path == "/" || r.URL.Path == "/index.html" || strings.HasPrefix(r.URL.Path, "/projects/")
		if r.Method != http.MethodGet || !page {
			next.ServeHTTP(w, r)
			return
		}
		if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
			if _, expires, err := st.LookupSession(r.Context(), cookie.Value); err == nil && expires.After(time.Now()) {
				next.ServeHTTP(w, r)
				return
			}
		}
		session, err := devSession(r.Context(), st)
		if err != nil {
			logger.Printf("local sign-in failed: %v", err)
			http.Error(w, "Local sign-in failed. Refresh to try again.", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: session.ID, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
		next.ServeHTTP(w, r)
	})
}
