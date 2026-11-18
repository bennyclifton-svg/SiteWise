package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/store"
)

// Share routes (docs/plans/2026-11-17-guest-access.md, Task 6): the PM makes,
// lists and revokes guest links on one project. A share makes no person: no
// user, membership, invite or seat. None calls Jev or the LLM.

const shareCreatesHour = 100

var shareRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /projects/{id}/shares":           postShare,
	"GET /projects/{id}/shares":            getShares,
	"DELETE /projects/{id}/shares/{share}": deleteShare,
}

func init() {
	for pattern := range shareRoutes {
		routePaths[pattern] = pathTeam
	}
}

func shareError(w http.ResponseWriter, deps Deps, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrInvalidInput):
		http.Error(w, "Name who it is shared with, one thing to share, and a life of 1 to 180 days.", http.StatusBadRequest)
	default:
		deps.Log.Printf("share failed: %v", err)
		http.Error(w, "share failed", http.StatusInternalServerError)
	}
}

// postShare makes one link and answers its address once: the plain token is
// in the link's fragment and nowhere else; only its hash is kept.
func postShare(w http.ResponseWriter, r *http.Request, deps Deps) {
	s, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.GuestGrantInput
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &in); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !deps.limits.allow("share:"+s.OrgID, shareCreatesHour, time.Hour) {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "That is a lot of links in an hour. Wait and try again.", http.StatusTooManyRequests)
		return
	}
	raw, hash, err := auth.NewToken()
	if err != nil {
		shareError(w, deps, err)
		return
	}
	project := r.PathValue("id")
	g, err := deps.Store.CreateGuestGrant(r.Context(), s.OrgID, project, s.UserID, in, hash, time.Now())
	if err != nil {
		shareError(w, deps, err)
		return
	}
	view, err := shareNamed(r.Context(), deps, s.OrgID, project, g.ID)
	if err != nil {
		shareError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(s.OrgID)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"share": view, "link": deps.PublicOrigin + "/g#t=" + raw})
}

func shareNamed(ctx context.Context, deps Deps, org, project, id string) (store.ShareView, error) {
	views, err := deps.Store.ListGuestGrants(ctx, org, project, store.ShareFilter{}, time.Now())
	if err != nil {
		return store.ShareView{}, err
	}
	for _, v := range views {
		if v.ID == id {
			return v, nil
		}
	}
	return store.ShareView{}, store.ErrNotFound
}

// sharesBody is the Share dialog's list for one thing: who holds a link,
// and what the dialog should warn of or suggest before a new one.
type sharesBody struct {
	Shares []store.ShareView `json:"shares"`
	// BudgetDisclosed: the newest issue of the shared report or RFT prints
	// the budget, which the dialog says before Create link.
	BudgetDisclosed bool `json:"budget_disclosed"`
	// Issued: the shared report or the package's RFP or RFT has an issue.
	Issued bool `json:"issued"`
	// SuggestTo is the invited firm's directory email, to fill "Shared with".
	SuggestTo string `json:"suggest_to,omitempty"`
}

func getShares(w http.ResponseWriter, r *http.Request, deps Deps) {
	s, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := store.ShareFilter{Kind: q.Get("kind"), ReportID: q.Get("report"), PackageID: q.Get("package"), FirmID: q.Get("firm"), DocumentID: strings.ToLower(q.Get("document"))}
	for _, id := range []string{f.ReportID, f.PackageID, f.FirmID, f.DocumentID} {
		if id != "" && !uuidPattern.MatchString(id) {
			http.NotFound(w, r)
			return
		}
	}
	out, err := deps.Store.ShareContext(r.Context(), s.OrgID, r.PathValue("id"), f)
	if err != nil {
		shareError(w, deps, err)
		return
	}
	views, err := deps.Store.ListGuestGrants(r.Context(), s.OrgID, r.PathValue("id"), f, time.Now())
	if err != nil {
		shareError(w, deps, err)
		return
	}
	writeJSON(w, http.StatusOK, sharesBody{Shares: views, BudgetDisclosed: out.BudgetDisclosed, Issued: out.Issued, SuggestTo: out.SuggestTo})
}

// deleteShare revokes a link at once. It cannot be put back, so chat only
// prepares it (shares.revoke, Click).
func deleteShare(w http.ResponseWriter, r *http.Request, deps Deps) {
	s, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	share := r.PathValue("share")
	if !uuidPattern.MatchString(share) {
		http.NotFound(w, r)
		return
	}
	v, err := deps.Store.RevokeGuestGrant(r.Context(), s.OrgID, r.PathValue("id"), share, time.Now())
	if err != nil {
		shareError(w, deps, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(s.OrgID)
	}
	writeJSON(w, http.StatusOK, v)
}
