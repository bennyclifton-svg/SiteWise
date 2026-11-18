package httpapi

import (
	"context"
	"fmt"
	"time"

	"sitewise/internal/store"
)

// Guest links from chat (docs/plans/2026-11-17-guest-access.md, "From
// chat"). Revoking is the one change chat may prepare: a revoked link
// cannot be put back, so it always waits for the PM's click. Making a link
// is not an operation: chat opens Share filled in and the PM presses Create
// link. The token is never in a read, a card or a summary.
var shareForwards = []forward{{
	Name:   "shares.revoke",
	About:  "withdraw a guest link at once, so whoever holds it can no longer open it; a withdrawn link can't be put back.",
	Fields: `share (id from read shares)`,
	Route:  "DELETE /projects/{id}/shares/{share}",
	Click:  true,
	Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
		v, err := liveShare(ctx, d, who, project, fieldText(f, "share"))
		if err != nil {
			return sent{}, err
		}
		f["_done"] = fmt.Sprintf("Withdrew the link shared with %s for %s.", v.SharedWith, v.What)
		return sent{Path: map[string]string{"share": v.ID}}, nil
	},
	Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
		v, err := liveShare(ctx, d, who, project, fieldText(f, "share"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Withdraw the link shared with %s for %s? Whoever holds it can no longer open it, and it can't be put back.", v.SharedWith, v.What), nil
	},
	Done: func(f map[string]any) string { return fieldText(f, "_done") },
}}

// liveShare finds a working link of this project by its id or short id.
func liveShare(ctx context.Context, d Deps, who opActor, project, ref string) (store.ShareView, error) {
	if ref == "" {
		return store.ShareView{}, refuse("share", "share is needed.")
	}
	views, err := d.Store.ListGuestGrants(ctx, who.OrgID, project, store.ShareFilter{}, time.Now())
	if err != nil {
		return store.ShareView{}, err
	}
	v, err := byRef(views, func(v store.ShareView) string { return v.ID }, ref, "link")
	if err != nil {
		return store.ShareView{}, err
	}
	if !v.Live {
		return store.ShareView{}, refuse("share", "The link shared with %s no longer works; there is nothing to withdraw.", v.SharedWith)
	}
	return v, nil
}
