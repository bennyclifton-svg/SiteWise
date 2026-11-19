package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/store"
)

// Email intake (docs/plans/2026-11-14-email-intake.md, Tasks 6 and 7, as
// amended by docs/plans/2026-11-17-correspondence-transmittals.md A1 to A7).
// Mailgun receives mail for the project addresses and posts each message,
// parsed, to one signed webhook. Nothing here calls Jev or the LLM: each
// filed document takes the ordinary filing, one fan-out per document, as a
// drop does. Nothing in an email is an instruction to anything.

const (
	pathMailInbound = "mail_inbound_accept"
	pathMailRead    = "mail_read"
	pathMailDecide  = "mail_decide"
	// Two posts are parsed at once; a third waits for Mailgun's retry.
	mailSlots = 2
	// Refused posts per address per 15 minutes before the webhook stops
	// reading their bodies. Only refusals count, so Mailgun's own posts are
	// never blocked by someone else's bad ones.
	mailMissesPerIP = 30
	// Messages filed per project an hour; the rest are held, so a flood
	// from one sender cannot spend the org's Jev reads or allowance.
	mailFiledPerProjectHour = 60
	// Attachments read from one message; the rest are not kept.
	mailMaxParts = 50
	// msgMailRefused is the one answer to every refused post: no oracle.
	msgMailRefused = "not accepted"
)

// outsideReserve: mail from someone outside the org never takes the org's
// last tenth of its monthly allowance, as a guest's upload does not.
const outsideReserve = 0.10

var mailRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /mail/inbound": mailInbound,
	// Mailgun's raw-MIME route forwards to a URL ending in "mime".
	"POST /mail/inbound/mime":                    mailInbound,
	"GET /projects/{id}/mail":                    getMail,
	"GET /projects/{id}/mail/{message}":          getMailMessage,
	"POST /projects/{id}/mail/{message}/accept":  acceptMail,
	"POST /projects/{id}/mail/{message}/discard": discardMail,
	"POST /projects/{id}/mail/address":           rotateMailAddress,
	"PATCH /projects/{id}/mail/{message}":        patchMail,
}

func init() {
	routePaths["POST /mail/inbound"] = pathMailInbound
	routePaths["POST /mail/inbound/mime"] = pathMailInbound
	routePaths["GET /projects/{id}/mail"] = pathMailRead
	routePaths["GET /projects/{id}/mail/{message}"] = pathMailRead
	routePaths["POST /projects/{id}/mail/{message}/accept"] = pathMailDecide
	routePaths["POST /projects/{id}/mail/{message}/discard"] = pathMailDecide
	routePaths["POST /projects/{id}/mail/address"] = pathMailDecide
	routePaths["PATCH /projects/{id}/mail/{message}"] = pathMailDecide
}

// fileMail files every kept part of a message that has no document yet, as
// a drop would, and marks the message filed. Each part's document id is
// written before its filing starts (A3). The body document is never read by
// the profile. by is the member who accepted held mail, or "".
func fileMail(ctx context.Context, deps Deps, org, project, id, by string) error {
	m, err := deps.Store.MailMessageByID(ctx, org, project, id)
	if err != nil {
		return err
	}
	for _, p := range m.Parts {
		if p.SHA256 == nil || p.DocumentID != "" {
			continue
		}
		filing, err := deps.Uploader.File(ctx, org, project, p.Filename, files.Blob{SHA256: p.SHA256, Size: p.Size})
		if err != nil {
			return err
		}
		if err := deps.Store.SetMailPartDocument(ctx, org, id, p.N, filing.DocumentID); err != nil {
			return err
		}
		if p.N == 0 {
			if _, err := deps.Store.SetProfileReading(ctx, org, project, []string{filing.DocumentID}, "skip", deps.ProfileReading.Kinds()); err != nil {
				deps.Log.Printf("mail org=%s: body document profile reading not set: %v", org, err)
			}
		}
		if filing.Created {
			countDocument(deps, org)
		}
		if filing.Status == store.StatusPending && deps.Filer != nil {
			deps.Filer.Start(org, filing.DocumentID)
		}
	}
	return deps.Store.MarkMailFiled(ctx, org, project, id, by)
}

// removeUnreferenced removes bytes nothing references any more.
func removeUnreferenced(deps Deps, sums [][]byte) {
	if deps.Blobs == nil || len(sums) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, sum := range sums {
		if _, err := deps.Blobs.Remove(sum, blobMinAge, func() (bool, error) { return deps.Store.BlobReferenced(ctx, sum) }); err != nil {
			deps.Log.Printf("mail: stored file kept: %v", err)
		}
	}
}

// mailEvent tells the project's screens the message list moved.
func mailEvent(deps Deps, org, project, message string) {
	payload, _ := json.Marshal(map[string]string{"project_id": project, "message_id": message})
	if _, err := deps.Store.AppendEvent(context.Background(), org, "mail", "", string(payload)); err != nil {
		deps.Log.Printf("mail event org=%s: %v", org, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(org)
	}
}

// mailAddress is the project's full address, or "" with intake off.
func mailAddress(ctx context.Context, deps Deps, org, project string) (string, error) {
	if deps.Mail == nil {
		return "", nil
	}
	local, err := deps.Store.MailAddress(ctx, org, project)
	if err != nil {
		return "", err
	}
	return local + "@" + deps.Mail.Domain, nil
}

type mailList struct {
	Address  string              `json:"address"`
	Held     int                 `json:"held"`
	Messages []store.MailMessage `json:"messages"`
}

// getMail lists the project's messages, filed and held, newest first, with
// the project's address (empty with intake off). ?before= pages.
func getMail(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	project := r.PathValue("id")
	q := r.URL.Query()
	if doc := q.Get("document"); doc != "" {
		getDocumentTransmittals(w, r, deps, session.OrgID, project, doc)
		return
	}
	f := store.MailFilter{Held: q.Get("held") == "1", Unanswered: q.Get("unanswered") == "1", NotSent: q.Get("not_sent") == "1",
		FirmID: q.Get("firm"), PackageID: q.Get("package")}
	for _, id := range []string{f.FirmID, f.PackageID} {
		if id != "" && !uuidPattern.MatchString(id) {
			http.Error(w, "firm and package are ids", http.StatusBadRequest)
			return
		}
	}
	if b := r.URL.Query().Get("before"); b != "" {
		t, err := time.Parse(time.RFC3339Nano, b)
		if err != nil {
			http.Error(w, "before must be a time", http.StatusBadRequest)
			return
		}
		f.Before = t
	}
	out, err := readMailList(r.Context(), deps, session.OrgID, project, f)
	if err != nil {
		readFailed(w, deps, "mail", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func readMailList(ctx context.Context, deps Deps, org, project string, f store.MailFilter) (mailList, error) {
	msgs, err := deps.Store.MailMessages(ctx, org, project, f)
	if err != nil {
		return mailList{}, err
	}
	out := mailList{Messages: msgs}
	for _, m := range msgs {
		if m.Status == "held" {
			out.Held++
		}
	}
	out.Address, err = mailAddress(ctx, deps, org, project)
	return out, err
}

// mailMessage resolves {message} in this project, or answers 404.
func mailMessage(w http.ResponseWriter, r *http.Request, deps Deps, org string) (store.MailMessage, bool) {
	id := r.PathValue("message")
	if !uuidPattern.MatchString(id) {
		http.NotFound(w, r)
		return store.MailMessage{}, false
	}
	m, err := deps.Store.MailMessageByID(r.Context(), org, r.PathValue("id"), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return m, false
	}
	if err != nil {
		readFailed(w, deps, "mail", err)
		return m, false
	}
	return m, true
}

// getMailMessage is one message with its body, as plain text. The body is
// never HTML: it was stored as text and is shown as text.
func getMailMessage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	m, ok := mailMessage(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// acceptMail files a held message as the webhook would have, with the
// caller as the one who decided. It starts Jev filing, so chat prepares it
// as a card for the PM's click (ops_mail.go).
func acceptMail(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	if !uploadAllowed(w, r, deps, session) {
		return
	}
	project := r.PathValue("id")
	m, ok := mailMessage(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	switch {
	case m.Status != "held":
		http.Error(w, "This message is already filed.", http.StatusConflict)
		return
	case m.HeldReason == store.MailHeldFull:
		http.Error(w, "Held mail was full when this arrived, so its files were not kept. Ask the sender to send it again.", http.StatusConflict)
		return
	}
	if archived, err := deps.Store.ProjectArchived(r.Context(), session.OrgID, project); err != nil {
		readFailed(w, deps, "mail", err)
		return
	} else if archived {
		http.Error(w, "This project is archived. Restore it on the Projects page to file mail.", http.StatusConflict)
		return
	}
	if err := fileMail(r.Context(), deps, session.OrgID, project, m.ID, session.UserID); err != nil {
		deps.Log.Printf("mail accept org=%s: %v", session.OrgID, err)
		http.Error(w, "accept failed", http.StatusInternalServerError)
		return
	}
	mailEvent(deps, session.OrgID, project, m.ID)
	m, err := deps.Store.MailMessageByID(r.Context(), session.OrgID, project, m.ID)
	if err != nil {
		readFailed(w, deps, "mail", err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// discardMail deletes a held message and the bytes nothing else uses. A
// filed message is a record and is never discarded (A7).
func discardMail(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	if !uuidPattern.MatchString(r.PathValue("message")) {
		http.NotFound(w, r)
		return
	}
	project := r.PathValue("id")
	sums, err := deps.Store.DiscardMail(r.Context(), session.OrgID, project, r.PathValue("message"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrMailNotHeld):
		http.Error(w, "A filed message is part of the record and can't be deleted.", http.StatusConflict)
		return
	case err != nil:
		readFailed(w, deps, "mail", err)
		return
	}
	removeUnreferenced(deps, sums)
	mailEvent(deps, session.OrgID, project, r.PathValue("message"))
	w.WriteHeader(http.StatusNoContent)
}

// msgRotateWarning is A6's warning, shown before the PM rotates.
const msgRotateWarning = "Replies to messages sent from the old address will not arrive."

// rotateMailAddress gives the project a new address. It is the PM's act on
// the screen and never chat's.
func rotateMailAddress(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	if deps.Mail == nil {
		http.NotFound(w, r)
		return
	}
	project := r.PathValue("id")
	local, err := deps.Store.RotateMailAddress(r.Context(), session.OrgID, project)
	if err != nil {
		readFailed(w, deps, "mail address", err)
		return
	}
	mailEvent(deps, session.OrgID, project, "")
	writeJSON(w, http.StatusOK, map[string]string{"address": local + "@" + deps.Mail.Domain, "warning": msgRotateWarning})
}

// mailReasonWords are the held reasons in plain words (Task 9).
var mailReasonWords = map[string]string{
	store.MailUnknownSender:    "Not from your team",
	store.MailUnverifiedSender: "Could not confirm the sender",
	store.MailPlanLimit:        "Monthly document limit reached",
	store.MailArchived:         "Project is archived",
	store.MailHeldFull:         "Held mail is full",
	store.MailBusy:             "Too much mail this hour",
}

func mailReason(r string) string {
	if w, ok := mailReasonWords[r]; ok {
		return w
	}
	return fmt.Sprint(r)
}

// getDocumentTransmittals is ?document=: each transmittal that carried the
// document or a revision it supersedes, and for the newest revision who was
// sent an older one and never this one (Task 5).
func getDocumentTransmittals(w http.ResponseWriter, r *http.Request, deps Deps, org, project, doc string) {
	if !uuidPattern.MatchString(doc) {
		http.NotFound(w, r)
		return
	}
	out, err := deps.Store.TransmittalsOfDocument(r.Context(), org, project, doc)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		readFailed(w, deps, "transmittals", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// patchMail is the PM's two edits to a filed message: its package, and
// what was done with Jev's pick. Any other field is refused: the record is
// frozen. Both are operations with Undo (mail.link, mail.dismiss_pick).
func patchMail(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	if !uuidPattern.MatchString(r.PathValue("message")) {
		http.NotFound(w, r)
		return
	}
	var e store.MailEdit
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		http.Error(w, "Only a message's package and its pick can change; nothing else on a message is edited.", http.StatusBadRequest)
		return
	}
	m, err := deps.Store.EditMail(r.Context(), session.OrgID, r.PathValue("id"), r.PathValue("message"), e)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "This message changed while the edit was being made. Read it again.", http.StatusConflict)
		return
	case errors.Is(err, store.ErrMailHeld):
		http.Error(w, "A held message isn't part of the record yet: accept or discard it.", http.StatusConflict)
		return
	case errors.Is(err, store.ErrInvalidInput):
		http.Error(w, strings.TrimPrefix(err.Error(), store.ErrInvalidInput.Error()+": "), http.StatusBadRequest)
		return
	case err != nil:
		readFailed(w, deps, "mail", err)
		return
	}
	mailEvent(deps, session.OrgID, r.PathValue("id"), m.ID)
	writeJSON(w, http.StatusOK, m)
}
