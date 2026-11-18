package httpapi

import (
	"context"
	"fmt"
	"strings"

	"sitewise/internal/store"
)

// Prepared reads (owner decision 1, 18 November 2026,
// docs/plans/2026-11-18-chat-agency.md Task 6). Chat may prepare a Jev read
// as a card; the PM's click starts it. The turn never calls Jev, and the
// click runs the screen's own handler, which only queues the background read
// (rows a worker picks up, or the reader's goroutine) and returns. A read
// spends credits and cannot be put back, so each is Click: it is always
// prepared, whatever the turn read, and its applied card offers no Undo.
//
// Each read has one resolver. It runs when the card is prepared, so a
// foreign id or a read already under way is refused before any card exists,
// and again on the click, because the state may have moved in between.

// readResolver checks the fields against this project and returns what the
// screen would send, the card's words, and the words for the applied card.
type readResolver func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (s sent, card, done string, err error)

// costWords is what every card says the click costs.
const costWords = "This uses Jev credits and can't be undone."

func readForward(name, about, fields, route string, resolve readResolver) forward {
	return forward{Name: name, About: about, Fields: fields, Route: route, Click: true,
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			s, _, done, err := resolve(ctx, d, who, project, f)
			f["_done"] = done
			return s, err
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			_, card, _, err := resolve(ctx, d, who, project, f)
			return card, err
		},
		Done: func(f map[string]any) string { return fieldText(f, "_done") },
	}
}

var readForwards = []forward{
	readForward("documents.read_missing_details",
		"start a Jev read of a scanned document's missing filing details (number, title, revision and the like); it uses Jev credits and can't be undone.",
		`document (id from documents)`, "POST /documents/{id}/details/reprocess",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
			switch {
			case err != nil:
				return sent{}, "", "", err
			case store.OCRDetailsActive(v.Reason):
				return sent{}, "", "", refuse("document", "Jev is already reading the missing details of %s; nothing more is queued.", name)
			case v.Status != store.StatusFiled || (v.Reason != "ocr_review" && !strings.HasPrefix(v.Reason, "ocr_details_")):
				return sent{}, "", "", refuse("document", "%s has no missing details for Jev to read from its scan.", name)
			}
			return sent{Path: map[string]string{"id": v.ID}},
				fmt.Sprintf("Read the missing filing details of %s from its scan? %s", name, costWords),
				fmt.Sprintf("Queued: Jev reads the missing details of %s.", name), nil
		}),
	readForward("documents.retry_filing",
		"start filing again a document whose filing stopped part way; Jev reads it, which uses credits and can't be undone.",
		`document (id from documents)`, "POST /documents/{id}/filing",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
			if err != nil {
				return sent{}, "", "", err
			}
			// The same three cases retryFiling acts on; anything else it
			// answers without starting anything.
			ocr := v.Status == store.StatusNotFiled && (v.Reason == "no_text_layer" || strings.HasPrefix(v.Reason, "ocr_"))
			expansion := v.Expansion != nil && v.Expansion.Status != "complete"
			if v.Status == store.StatusPending && strings.HasPrefix(v.Reason, "ocr_") {
				return sent{}, "", "", refuse("document", "%s is already queued to be read from its scan; nothing more is started.", name)
			}
			if !ocr && !expansion && v.Status != store.StatusPending {
				return sent{}, "", "", refuse("document", "%s is filed; there is no stopped filing to start again.", name)
			}
			return sent{Path: map[string]string{"id": v.ID}},
				fmt.Sprintf("Start filing %s again? If it is still being filed, nothing more is started. %s", name, costWords),
				fmt.Sprintf("Filing of %s started again.", name), nil
		}),
	readForward("documents.split_set",
		"split a drawing set the automatic split stopped on into its sheets, and file each sheet with Jev; it uses Jev credits and can't be undone.",
		`document (id of the drawing set, from documents)`, "POST /documents/{id}/split",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
			switch {
			case err != nil:
				return sent{}, "", "", err
			case v.Expansion != nil && v.Expansion.Manual && v.Expansion.Status != "review":
				return sent{}, "", "", refuse("document", "%s is already being split; nothing more is queued.", name)
			case v.Expansion == nil || !v.Expansion.CanSplit:
				return sent{}, "", "", refuse("document", "%s is not a drawing set waiting to be split.", name)
			}
			return sent{Path: map[string]string{"id": v.ID}},
				fmt.Sprintf("Split %s into its sheets and file each one? Jev reads every sheet. %s", name, costWords),
				fmt.Sprintf("Queued: %s is split into its sheets and each is filed.", name), nil
		}),
	readForward("documents.read_images",
		"start a Jev read of a filed PDF's page images for its title block; it uses Jev credits and can't be undone.",
		`document (id from documents)`, "POST /documents/{id}/visual-review",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
			if err != nil {
				return sent{}, "", "", err
			}
			if v.Status != store.StatusFiled || !strings.HasSuffix(strings.ToLower(v.Filename), ".pdf") {
				return sent{}, "", "", refuse("document", "Image review is available for filed PDFs only; %s is not one.", name)
			}
			review, err := d.Store.VisualIdentity(ctx, who.OrgID, v.ID)
			if err != nil {
				return sent{}, "", "", err
			}
			switch review.Status {
			case "queued", "reading":
				return sent{}, "", "", refuse("document", "The images of %s are already queued for Jev; nothing more is queued.", name)
			case "ready":
				return sent{}, "", "", refuse("document", "The images of %s have been read already; the result is on the document.", name)
			}
			return sent{Path: map[string]string{"id": v.ID}},
				fmt.Sprintf("Read the title block of %s from its page images? %s", name, costWords),
				fmt.Sprintf("Queued: Jev reads the page images of %s.", name), nil
		}),
	readForward("procurement.read_proposal",
		"place a filed document on a package as a proposal and start its read (Jev, and the tender pass when the model is on); it uses credits and can't be undone.",
		`package (id); document (id from documents); firm (optional: an invited firm's name, when the quote is that firm's)`,
		"POST /projects/{id}/procurement/packages/{pkg}/documents",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, "", "", err
			}
			v, name, err := projectDocument(ctx, d, who, project, fieldText(f, "document"))
			if err != nil {
				return sent{}, "", "", err
			}
			if v.Status != store.StatusFiled {
				return sent{}, "", "", refuse("document", "Wait for %s to finish filing.", name)
			}
			if sub, err := d.Store.SubmissionForDocument(ctx, who.OrgID, v.ID); err != nil {
				return sent{}, "", "", err
			} else if sub != "" && onPackage(ctx, d, who, project, p.ID, sub) {
				return sent{}, "", "", refuse("document", "%s is already on %s; nothing more is read.", name, p.Title)
			}
			body := map[string]any{"document_id": v.ID}
			as := ""
			if firm := fieldText(f, "firm"); firm != "" {
				id, sub, shown, err := firmOn(ctx, d, who, project, p.ID, firm)
				if err != nil {
					return sent{}, "", "", err
				}
				if sub != "" {
					return sent{}, "", "", refuse("firm", "%s has quoted on %s already; name an invited firm, or none.", shown, p.Title)
				}
				body["firm_id"], as = id, " as "+shown+"'s quote"
			}
			return sent{Path: map[string]string{"pkg": p.ID}, Body: body},
				fmt.Sprintf("Place %s on %s%s and read it? %s", name, p.Title, as, costWords),
				fmt.Sprintf("Placed %s on %s; its read has started.", name, p.Title), nil
		}),
	readForward("procurement.read_tender_again",
		"read a firm's quote for prices again (the tender pass); the firm's own lines entered by hand are kept and its column goes back to to check. It sends the quote to the model and can't be undone.",
		`package (id); firm (its name on that package)`, "POST /projects/{id}/submissions/{sub}/tender-read",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, "", "", err
			}
			_, sub, shown, err := firmOn(ctx, d, who, project, p.ID, fieldText(f, "firm"))
			if err != nil {
				return sent{}, "", "", err
			}
			if sub == "" {
				return sent{}, "", "", refuse("firm", "%s has not quoted on %s, so there is nothing to read.", shown, p.Title)
			}
			// The same refusals the route gives, before a card is shown.
			if d.TenderReader == nil {
				return sent{}, "", "", refuse("", "This server is not set up to use a model.")
			}
			if on, err := d.Store.ModelEnabled(ctx, who.OrgID); err != nil {
				return sent{}, "", "", err
			} else if !on {
				return sent{}, "", "", refuse("", "The model is switched off for this organisation.")
			}
			state, err := d.Store.TenderReadState(ctx, who.OrgID, project, sub)
			if err != nil {
				return sent{}, "", "", err
			}
			if state == store.TenderPending || state == store.TenderReading {
				return sent{}, "", "", refuse("firm", "%s's quote on %s is already being read; nothing more is queued.", shown, p.Title)
			}
			return sent{Path: map[string]string{"sub": sub}},
				fmt.Sprintf("Read %s's quote on %s for prices again? Lines entered by hand are kept and the column goes back to to check. This sends the quote to the model and can't be undone.", shown, p.Title),
				fmt.Sprintf("Queued: %s's quote on %s is read for prices again.", shown, p.Title), nil
		}),
	readForward("profile.update",
		"update the project profile: Jev reads the documents the profile has not read yet; it uses Jev credits and can't be undone.",
		`(none)`, "POST /projects/{id}/profile/read",
		func(ctx context.Context, d Deps, who opActor, project string, _ map[string]any) (sent, string, string, error) {
			run, err := d.Store.ProfileProgress(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, "", "", err
			}
			busy := 0
			for _, doc := range run.Documents {
				if doc.State == "queued" || doc.State == "reading" {
					busy++
				}
			}
			card := "Update the project profile? Jev reads the documents it has not read yet. " + costWords
			if busy > 0 {
				// RequestProfileRead restamps what is queued; it never queues a
				// document twice. Say so rather than refuse: the click still
				// retries failed passes, as the screen's Update does.
				card = fmt.Sprintf("Update the project profile? An update is already reading %s; nothing is queued twice, and failed reads are tried again. %s", countOf(busy, "document"), costWords)
			}
			return sent{}, card, "Queued: the profile update has started.", nil
		}),
	readForward("profile.document_reading",
		"choose whether the profile reads some documents (read, skip, or auto to let SiteWise decide by kind); a document set to read is read by Jev at the next profile update, which uses credits. It can't be undone from chat.",
		`documents (ids from documents); setting (read|skip|auto)`, "PUT /projects/{id}/documents/profile-read",
		func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, string, string, error) {
			setting := fieldText(f, "setting")
			if setting != "read" && setting != "skip" && setting != "auto" {
				return sent{}, "", "", refuse("setting", "setting is read, skip or auto.")
			}
			refs, _ := f["documents"].([]any)
			if len(refs) == 0 || len(refs) > maxScreenIDs {
				return sent{}, "", "", refuse("documents", "Name between 1 and %d documents.", maxScreenIDs)
			}
			views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, "", "", err
			}
			ids := make([]string, 0, len(refs))
			names := make([]string, 0, len(refs))
			for _, r := range refs {
				ref, _ := r.(string)
				v, err := byRef(views, func(v store.DocumentView) string { return v.ID }, ref, "document")
				if err != nil {
					return sent{}, "", "", err
				}
				ids, names = append(ids, v.ID), append(names, documentName(v))
			}
			which := listNames(names)
			card := map[string]string{
				"read": fmt.Sprintf("Have the profile read %s? Jev reads them at the next profile update, which uses credits. This can't be undone from chat.", which),
				"skip": fmt.Sprintf("Stop the profile reading %s? What Jev has read of them so far is kept. This can't be undone from chat.", which),
				"auto": fmt.Sprintf("Let SiteWise decide by kind whether the profile reads %s? This can't be undone from chat.", which),
			}[setting]
			return sent{Body: map[string]any{"document_ids": ids, "setting": setting}}, card,
				fmt.Sprintf("Profile reading set to %s for %s.", setting, which), nil
		}),
	readForward("delivery.read_notice",
		"read a filed notice of determination into the register's consent conditions: Go splits it, and Jev reads who acts, the system and whether evidence is needed for each condition, as rows to check; it uses Jev credits and can't be undone.",
		`document (id from documents)`, "POST /projects/{id}/delivery/imports", resolveReadNotice),
	readForward("specifications.update",
		"read the project's documents for the specification schedule (the Update button on Specifications); it uses Jev credits and can't be undone.",
		`(none)`, "POST /projects/{id}/specifications/update",
		func(context.Context, Deps, opActor, string, map[string]any) (sent, string, string, error) {
			// RequestSpecificationUpdate leaves out documents whose read is
			// queued or running, so a second click queues nothing twice.
			return sent{}, "Read the project's documents for the specification schedule? Documents already being read are not queued again. " + costWords,
				"Queued: the specification read has started.", nil
		}),
}

// projectDocument finds a document of this project by its id or short id,
// so a document of another project or org is never found.
func projectDocument(ctx context.Context, d Deps, who opActor, project, ref string) (store.DocumentView, string, error) {
	if ref == "" {
		return store.DocumentView{}, "", refuse("document", "document is needed.")
	}
	views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
	if err != nil {
		return store.DocumentView{}, "", err
	}
	v, err := byRef(views, func(v store.DocumentView) string { return v.ID }, ref, "document")
	if err != nil {
		return store.DocumentView{}, "", err
	}
	return v, documentName(v), nil
}

// documentName is how the card names a document: its number and title as
// filed, or its file name.
func documentName(v store.DocumentView) string {
	title := ""
	for _, f := range v.Fields {
		if f.Field == "title" {
			title = strings.TrimSpace(f.Value)
		}
	}
	switch {
	case title == "":
		return v.Filename
	case v.Number != "" && !strings.Contains(title, v.Number):
		return v.Number + " " + title
	}
	return title
}

// onPackage reports whether a submission is a firm card on the package.
func onPackage(ctx context.Context, d Deps, who opActor, project, pkg, sub string) bool {
	board, _, err := d.Store.ProcurementBoard(ctx, who.OrgID, project)
	if err != nil {
		return false
	}
	for _, c := range board[pkg] {
		if c.SubmissionID == sub {
			return true
		}
	}
	return false
}

func listNames(names []string) string {
	switch {
	case len(names) == 1:
		return names[0]
	case len(names) <= 3:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
	return fmt.Sprintf("%s, %s and %d more", names[0], names[1], len(names)-2)
}
