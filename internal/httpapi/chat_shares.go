package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/store"
)

// Guest links in chat (docs/plans/2026-11-17-guest-access.md, "From chat").
// read(shares) is printed by Go from the same list the Share dialog shows;
// a token is never in it. The Share dialog is opened filled in; the PM
// presses Create link. Revoking is shares.revoke (ops_shares.go).

// readShares renders every link on the project, newest first.
func readShares(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	views, err := d.Store.ListGuestGrants(ctx, who.OrgID, project, store.ShareFilter{}, time.Now())
	if err != nil {
		return err
	}
	if len(views) == 0 {
		r.line("Guest links: none. Nothing on this project has been shared outside the organisation.")
		return nil
	}
	r.line("Guest links (shared with | what | made by, on | opened | works until | state). A link shows only its one thing to whoever holds it.")
	for _, v := range views {
		what := v.What
		switch v.Kind {
		case store.GuestReport:
			what = "report " + what + ", every issued version"
		case store.GuestPackage:
			what = "the RFP or RFT for " + what + " (" + v.FirmName + "'s quote cell)"
		}
		opened := "never opened"
		if v.Opens > 0 || v.LastOpenedOn != "" {
			opened = fmt.Sprintf("opened %s, last %s", countOf(v.Opens, "time"), v.LastOpenedOn)
		}
		state := "works"
		switch {
		case v.RevokedAt != nil:
			state = "withdrawn"
		case !v.Live:
			state = "ended"
		}
		r.line("%s%s | %s | %s, %s | %s | %s | %s", r.id(v.ID), v.SharedWith, what, v.SharedBy, store.SydneyDate(v.CreatedAt), opened, v.ExpiresOn, state)
	}
	return nil
}

// linkWords is an invited firm's link state on the procurement read, or "".
func linkWords(c store.FirmCard) string {
	switch {
	case !c.Linked:
		return ""
	case c.LinkOpenedOn == "":
		return "link not opened"
	}
	return "link opened " + c.LinkOpenedOn
}

// packageLinkNotOpened is the named filter: rows with an invited firm that
// holds a live link it has never opened.
func packageLinkNotOpened(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
	board, _, err := d.Store.ProcurementBoard(ctx, who.OrgID, project)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for pkg, cards := range board {
		for _, c := range cards {
			if c.Linked && c.LinkOpenedOn == "" {
				set[pkg] = true
			}
		}
	}
	return inTableOrder(ctx, d, who, project, "procurement", set)
}

// msgShareKinds is chat's refusal for anything else (the plan's key case).
const msgShareKinds = "Only documents, an issued report or an issued RFP or RFT can be shared."

// checkShareDialog fills the Share dialog from ids Go has checked against
// this project. It makes no link: the PM presses Create link.
func checkShareDialog(ctx context.Context, d Deps, who opActor, project string, f map[string]string) (agent.View, string) {
	out := map[string]string{"kind": f["kind"]}
	if to := strings.TrimSpace(f["shared_with"]); to != "" {
		if len(to) > 254 || strings.ContainsAny(to, "\r\n") {
			return agent.View{}, "shared_with is one name or email address."
		}
		out["shared_with"] = to
	}
	if days := strings.TrimSpace(f["days"]); days != "" {
		n, err := strconv.Atoi(days)
		if err != nil || n < 1 || n > store.GuestMaxDays {
			return agent.View{}, fmt.Sprintf("days is a whole number from 1 to %d.", store.GuestMaxDays)
		}
		out["days"] = days
	}
	var what string
	switch f["kind"] {
	case store.GuestDocuments:
		views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
		if err != nil {
			return agent.View{}, "The documents could not be read."
		}
		var ids []string
		for _, ref := range strings.Split(f["documents"], ",") {
			if ref = strings.TrimSpace(ref); ref == "" {
				continue
			}
			v, err := byRef(views, func(v store.DocumentView) string { return v.ID }, ref, "document")
			if err != nil {
				return agent.View{}, err.Error()
			}
			if v.Status != store.StatusFiled {
				return agent.View{}, documentName(v) + " is not filed yet, so it cannot be shared."
			}
			if !containsID(ids, v.ID) {
				ids = append(ids, v.ID)
			}
		}
		if len(ids) == 0 || len(ids) > store.GuestMaxDocuments {
			return agent.View{}, fmt.Sprintf("Name 1 to %d documents by id.", store.GuestMaxDocuments)
		}
		out["documents"] = strings.Join(ids, ",")
		what = countOf(len(ids), "document")
	case store.GuestReport:
		reports, err := d.Store.ListReports(ctx, who.OrgID, project)
		if err != nil {
			return agent.View{}, "The reports could not be read."
		}
		r, err := byRef(reports, func(r store.Report) string { return r.ID }, f["report"], "report")
		if err != nil {
			return agent.View{}, err.Error()
		}
		c, err := d.Store.ShareContext(ctx, who.OrgID, project, store.ShareFilter{ReportID: r.ID})
		if err != nil {
			return agent.View{}, "The report could not be read."
		}
		if !c.Issued {
			return agent.View{}, r.Title + " has not been issued. " + msgShareKinds
		}
		out["report"], what = r.ID, r.Title
	case store.GuestPackage:
		p, err := pkgOf(ctx, d, who, project, map[string]any{"package": f["package"]})
		if err != nil {
			return agent.View{}, err.Error()
		}
		firm, sub, shown, err := firmOn(ctx, d, who, project, p.ID, f["firm"])
		if err != nil {
			return agent.View{}, err.Error()
		}
		if sub != "" {
			return agent.View{}, shown + " has quoted on " + p.Title + " already; a link is for a firm invited and not yet quoted."
		}
		c, err := d.Store.ShareContext(ctx, who.OrgID, project, store.ShareFilter{PackageID: p.ID, FirmID: firm})
		if err != nil {
			return agent.View{}, "Procurement could not be read."
		}
		if !c.Issued {
			return agent.View{}, "The RFP or RFT for " + p.Title + " has not been issued yet. Issue it first; then it can be shared."
		}
		if out["shared_with"] == "" && c.SuggestTo != "" {
			out["shared_with"] = c.SuggestTo
		}
		out["package"], out["firm"], out["firm_name"], what = p.ID, firm, shown, "the RFP or RFT for "+p.Title+" with "+shown
	default:
		return agent.View{}, msgShareKinds + " Say kind documents, report or package."
	}
	return agent.View{Do: "open", Dialog: "share", Fields: out,
		Line: "Opened Share for " + what + ". Press Create link there to make the link; chat makes none."}, ""
}
