package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/store"
)

// Team: the owner invites colleagues into their org. In v1 a person belongs
// to one org, so an email that already signs in cannot be invited.

const (
	inviteLifetime   = 7 * 24 * time.Hour
	limitInvitesHour = 20
	// noSeatCap is the seat count when billing is off (local development).
	noSeatCap = 1000
)

func getTeam(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	members, invites, err := deps.Store.Team(r.Context(), session.OrgID, time.Now())
	if err != nil {
		readFailed(w, deps, "team", err)
		return
	}
	seats := noSeatCap
	if deps.Billing != nil {
		b, err := deps.Store.OrgBilling(r.Context(), session.OrgID)
		if err != nil {
			readFailed(w, deps, "team", err)
			return
		}
		seats = b.Effective().Users
	}
	if members == nil {
		members = []store.TeamMember{}
	}
	if invites == nil {
		invites = []store.TeamInvite{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members, "invites": invites, "seats": seats})
}

func postTeamInvite(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
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
	if deps.AccountMail == nil {
		http.Error(w, "Email is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	if !deps.limits.allow("invite:"+session.OrgID, limitInvitesHour, time.Hour) {
		tooMany(w)
		return
	}
	ctx := r.Context()
	seats := noSeatCap
	planName := ""
	if deps.Billing != nil {
		b, err := deps.Store.OrgBilling(ctx, session.OrgID)
		if err != nil {
			readFailed(w, deps, "invite", err)
			return
		}
		seats, planName = b.Effective().Users, b.Effective().Name
	}
	profile, err := deps.Store.Profile(ctx, session.OrgID, session.UserID)
	if err != nil {
		readFailed(w, deps, "invite", err)
		return
	}
	raw, hash, err := auth.NewToken()
	if err != nil {
		readFailed(w, deps, "invite", err)
		return
	}
	now := time.Now()
	invite, err := deps.Store.CreateTeamInvite(ctx, session.OrgID, email, hash, now, now.Add(inviteLifetime), seats)
	switch {
	case errors.Is(err, store.ErrSeatLimit):
		msg := fmt.Sprintf("Your %s plan includes %d user%s. Upgrade to Team to invite colleagues.", planName, seats, plural(seats))
		if seats > 1 {
			msg = fmt.Sprintf("Your %s plan includes %d users, counting open invites. Remove a member or an invite first.", planName, seats)
		}
		http.Error(w, msg, http.StatusPaymentRequired)
		return
	case errors.Is(err, store.ErrEmailTaken):
		http.Error(w, "That email already has a SiteWise account. Each person can belong to one team for now.", http.StatusConflict)
		return
	case errors.Is(err, store.ErrInvitePending):
		http.Error(w, "That email already has an open invite.", http.StatusConflict)
		return
	case err != nil:
		readFailed(w, deps, "invite", err)
		return
	}
	writeJSON(w, http.StatusCreated, invite)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := deps.AccountMail.SendTeamInvite(ctx, email, profile.Email, deps.PublicOrigin+"/#invite="+raw); err != nil {
			deps.Log.Printf("team invite mail failed org=%s: %v", session.OrgID, err)
		}
	}()
}

func deleteTeamInvite(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	err := deps.Store.RevokeInvite(r.Context(), session.OrgID, r.PathValue("id"), time.Now())
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		readFailed(w, deps, "revoke invite", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteTeamMember(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := ownerSession(w, r, deps)
	if !ok {
		return
	}
	err := deps.Store.RemoveMember(r.Context(), session.OrgID, r.PathValue("id"))
	switch {
	case errors.Is(err, store.ErrLastOwner):
		http.Error(w, "The account owner cannot be removed.", http.StatusConflict)
		return
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case err != nil:
		readFailed(w, deps, "remove member", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// acceptTeamInvite is public: the invite token is the authority. It sets the
// invitee's password and signs them in to the inviting org.
func acceptTeamInvite(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !accountRequestOK(w, r, deps) {
		return
	}
	var body struct {
		Token       string `json:"token"`
		Password    string `json:"password"`
		AcceptTerms bool   `json:"accept_terms"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !body.AcceptTerms {
		http.Error(w, msgAcceptTerms, http.StatusBadRequest)
		return
	}
	if err := auth.CheckPasswordRule(body.Password); err != nil {
		http.Error(w, "Choose a password of 8 to 72 characters.", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		http.Error(w, "accept failed", http.StatusInternalServerError)
		return
	}
	session, err := deps.Store.AcceptTeamInvite(r.Context(), auth.HashToken(strings.TrimSpace(body.Token)), hash, TermsVersion, time.Now())
	switch {
	case errors.Is(err, store.ErrLinkUsed):
		http.Error(w, "That invite has expired or was already used. Ask for a new one.", http.StatusUnauthorized)
		return
	case errors.Is(err, store.ErrEmailTaken):
		http.Error(w, "That email already has a SiteWise account. Sign in with it instead.", http.StatusConflict)
		return
	case err != nil:
		deps.Log.Printf("accept invite: %v", err)
		http.Error(w, "accept failed", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, deps, session.ID)
	w.WriteHeader(http.StatusNoContent)
}
