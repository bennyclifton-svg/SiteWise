package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/store"
)

// Accounts: anyone can sign up; each account is its own org with one owner.
// None of these routes call Jev. See docs/plans/2026-11-03-accounts.md.
var accountRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /signup":          signUp,
	"POST /signin":          signIn,
	"POST /password/forgot": forgotPassword,
	"POST /password/reset":  resetPassword,
}

const pathAccountAuth = "account_auth"

func init() {
	for pattern := range accountRoutes {
		routePaths[pattern] = pathAccountAuth
	}
}

const (
	resetLifetime = time.Hour
	limitWindow   = 15 * time.Minute
	// Per client address across all account routes.
	limitPerIP = 30
	// Per email: wrong-password guesses, and reset mails per hour.
	limitSignInPerEmail = 10
	limitResetPerEmail  = 3
)

const msgWrongPassword = "Wrong email or password."

const msgAcceptTerms = "Accept the Terms of Service and Privacy Policy to continue."

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// AcceptTerms is required at sign-up and ignored at sign-in.
	AcceptTerms bool `json:"accept_terms"`
}

func signUp(w http.ResponseWriter, r *http.Request, deps Deps) {
	c, ok := readCredentials(w, r, deps)
	if !ok {
		return
	}
	if !c.AcceptTerms {
		http.Error(w, msgAcceptTerms, http.StatusBadRequest)
		return
	}
	if err := auth.CheckPasswordRule(c.Password); err != nil {
		http.Error(w, "Choose a password of 8 to 72 characters.", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(c.Password)
	if err != nil {
		http.Error(w, "sign up failed", http.StatusInternalServerError)
		return
	}
	session, err := deps.Store.CreateAccount(r.Context(), c.Email, hash, TermsVersion)
	if errors.Is(err, store.ErrEmailTaken) {
		http.Error(w, "An account with that email already exists. Sign in instead.", http.StatusConflict)
		return
	}
	if err != nil {
		deps.Log.Printf("sign up: %v", err)
		http.Error(w, "sign up failed", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, deps, session.ID)
	w.WriteHeader(http.StatusNoContent)
	go sendVerify(deps, session.OrgID, session.UserID, c.Email)
}

func signIn(w http.ResponseWriter, r *http.Request, deps Deps) {
	c, ok := readCredentials(w, r, deps)
	if !ok {
		return
	}
	if !deps.limits.allow("signin:"+c.Email, limitSignInPerEmail, limitWindow) {
		tooMany(w)
		return
	}
	account, err := deps.Store.AccountByEmail(r.Context(), c.Email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		deps.Log.Printf("sign in: %v", err)
		http.Error(w, "sign in failed", http.StatusInternalServerError)
		return
	}
	if !auth.PasswordMatches(account.PasswordHash, c.Password) {
		http.Error(w, msgWrongPassword, http.StatusUnauthorized)
		return
	}
	session, err := deps.Store.StartSession(r.Context(), account.OrgID, account.UserID)
	if err != nil {
		deps.Log.Printf("sign in: %v", err)
		http.Error(w, "sign in failed", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, deps, session.ID)
	w.WriteHeader(http.StatusNoContent)
}

// forgotPassword always answers 204 so it never reveals which emails have
// accounts. The mail goes out after the response, so timing reveals nothing
// either, and a slow provider never holds the request.
func forgotPassword(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !accountRequestOK(w, r, deps) {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email, err := auth.NormalEmail(body.Email)
	if err != nil {
		http.Error(w, "Enter a valid email address.", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	if deps.ResetMail == nil || !deps.limits.allow("reset:"+email, limitResetPerEmail, time.Hour) {
		return
	}
	go sendReset(deps, email)
}

func sendReset(deps Deps, email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	account, err := deps.Store.AccountByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		return
	}
	if err != nil {
		deps.Log.Printf("password reset: %v", err)
		return
	}
	raw, hash, err := auth.NewToken()
	if err != nil {
		deps.Log.Printf("password reset: %v", err)
		return
	}
	if err := deps.Store.CreatePasswordReset(ctx, account, hash, time.Now().Add(resetLifetime)); err != nil {
		deps.Log.Printf("password reset: %v", err)
		return
	}
	// The token rides in the fragment, which browsers never send to a server.
	if err := deps.ResetMail.SendReset(ctx, account.Email, deps.PublicOrigin+"/#reset="+raw); err != nil {
		deps.Log.Printf("password reset mail failed: %v", err)
	}
}

func resetPassword(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !accountRequestOK(w, r, deps) {
		return
	}
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := auth.CheckPasswordRule(body.Password); err != nil {
		http.Error(w, "Choose a password of 8 to 72 characters.", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		http.Error(w, "reset failed", http.StatusInternalServerError)
		return
	}
	session, err := deps.Store.ResetPassword(r.Context(), auth.HashToken(strings.TrimSpace(body.Token)), hash, time.Now())
	if errors.Is(err, store.ErrResetUsed) {
		http.Error(w, "That reset link has expired or was already used. Ask for a new one.", http.StatusUnauthorized)
		return
	}
	if err != nil {
		deps.Log.Printf("password reset: %v", err)
		http.Error(w, "reset failed", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, deps, session.ID)
	w.WriteHeader(http.StatusNoContent)
}

// readCredentials checks origin and the per-address limit, then reads and
// normalises the email.
func readCredentials(w http.ResponseWriter, r *http.Request, deps Deps) (credentials, bool) {
	if !accountRequestOK(w, r, deps) {
		return credentials{}, false
	}
	var c credentials
	if err := readJSON(w, r, deps.MaxBodyBytes, &c); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return credentials{}, false
	}
	email, err := auth.NormalEmail(c.Email)
	if err != nil {
		http.Error(w, "Enter a valid email address.", http.StatusBadRequest)
		return credentials{}, false
	}
	c.Email = email
	return c, true
}

func accountRequestOK(w http.ResponseWriter, r *http.Request, deps Deps) bool {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return false
	}
	if !deps.limits.allow("ip:"+clientIP(r), limitPerIP, limitWindow) {
		tooMany(w)
		return false
	}
	return true
}

func tooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "900")
	http.Error(w, "Too many attempts. Wait 15 minutes and try again.", http.StatusTooManyRequests)
}

func setSessionCookie(w http.ResponseWriter, deps Deps, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   deps.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// clientIP trusts X-Forwarded-For only from a loopback peer, which is Caddy
// on the same host. Caddy appends the real client last.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return host
}

// authLimits counts attempts in fixed windows. One process serves every
// request, so memory is enough; a restart clears the counts.
type authLimits struct {
	mu     sync.Mutex
	counts map[string]*window
	now    func() time.Time
}

type window struct {
	start time.Time
	n     int
}

func newAuthLimits() *authLimits {
	return &authLimits{counts: map[string]*window{}, now: time.Now}
}

func (l *authLimits) allow(key string, max int, span time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.counts) > 100_000 {
		for k, win := range l.counts {
			if now.Sub(win.start) >= time.Hour {
				delete(l.counts, k)
			}
		}
	}
	win := l.counts[key]
	if win == nil || now.Sub(win.start) >= span {
		l.counts[key] = &window{start: now, n: 1}
		return true
	}
	if win.n >= max {
		return false
	}
	win.n++
	return true
}
