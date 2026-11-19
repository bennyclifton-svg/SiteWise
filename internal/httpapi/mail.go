package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/mailin"
	"sitewise/internal/store"
)

// Email intake (docs/plans/2026-11-14-email-intake.md, Tasks 6 and 7, as
// amended by docs/plans/2026-11-17-correspondence-transmittals.md A1 to A7).
// Mailgun receives mail for the project addresses and posts each message,
// parsed, to one signed webhook. Nothing here calls Jev or the LLM: each
// filed document takes the ordinary filing, one fan-out per document, as a
// drop does. Nothing in an email is an instruction to anything.

// MailIn is email intake's configuration. Nil Deps.Mail is off: the webhook
// answers 404 and no project address is shown.
type MailIn struct {
	// Domain receives project mail ("in.sitewise.au").
	Domain string
	// SigningKey is the Mailgun account's HTTP webhook signing key. It is
	// never logged or returned.
	SigningKey string
	slots      chan struct{}
	now        func() time.Time
}

// NewMailIn switches email intake on, or returns nil when either setting
// is blank.
func NewMailIn(domain, key string) *MailIn {
	if domain == "" || key == "" {
		return nil
	}
	return &MailIn{Domain: strings.ToLower(domain), SigningKey: key}
}

func (m *MailIn) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

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
	"POST /mail/inbound":                         mailInbound,
	"GET /projects/{id}/mail":                    getMail,
	"GET /projects/{id}/mail/{message}":          getMailMessage,
	"POST /projects/{id}/mail/{message}/accept":  acceptMail,
	"POST /projects/{id}/mail/{message}/discard": discardMail,
	"POST /projects/{id}/mail/address":           rotateMailAddress,
}

func init() {
	routePaths["POST /mail/inbound"] = pathMailInbound
	routePaths["GET /projects/{id}/mail"] = pathMailRead
	routePaths["GET /projects/{id}/mail/{message}"] = pathMailRead
	routePaths["POST /projects/{id}/mail/{message}/accept"] = pathMailDecide
	routePaths["POST /projects/{id}/mail/{message}/discard"] = pathMailDecide
	routePaths["POST /projects/{id}/mail/address"] = pathMailDecide
}

// refuseMail is the answer to every refused post, whatever the reason: a bad
// signature, a stale or malformed timestamp, an unreadable or oversized
// body. 406 tells Mailgun not to retry. The refusal counts against the
// address.
func refuseMail(w http.ResponseWriter, r *http.Request, deps Deps, why string) {
	deps.limits.allow("mail-miss:"+clientIP(r), mailMissesPerIP, limitWindow)
	deps.Log.Printf("mail webhook: refused (%s)", why)
	http.Error(w, msgMailRefused, http.StatusNotAcceptable)
}

// mailInbound is Mailgun's forward() to POST /api/mail/inbound. It reads no
// session or cookie: the signature is the authentication. A store failure
// answers 500 so Mailgun retries; a message recorded once answers 200 to
// every retry.
func mailInbound(w http.ResponseWriter, r *http.Request, deps Deps) {
	cfg := deps.Mail
	if cfg == nil {
		http.NotFound(w, r)
		return
	}
	if deps.limits.full("mail-miss:"+clientIP(r), mailMissesPerIP, limitWindow) {
		refuseMail(w, r, deps, "too many refusals from this address")
		return
	}
	select {
	case cfg.slots <- struct{}{}:
		defer func() { <-cfg.slots }()
	default:
		w.Header().Set("Retry-After", "60")
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, mailin.MaxPostBytes)
	err := r.ParseMultipartForm(8 << 20)
	if errors.Is(err, http.ErrNotMultipart) {
		err = r.ParseForm()
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		refuseMail(w, r, deps, "unreadable or too large")
		return
	}
	now := cfg.clock()
	ts, token, sig := r.PostFormValue("timestamp"), r.PostFormValue("token"), r.PostFormValue("signature")
	if len(token) > 256 || !mailin.VerifySignature(cfg.SigningKey, ts, token, sig) {
		refuseMail(w, r, deps, "signature")
		return
	}
	if !mailin.Fresh(ts, now) {
		refuseMail(w, r, deps, "timestamp")
		return
	}
	outcome, org, project, parts, err := receiveMail(r, deps, token, now)
	if err != nil {
		deps.Log.Printf("mail webhook: not recorded: %v", err)
		http.Error(w, "not recorded", http.StatusInternalServerError)
		return
	}
	if org != "" {
		deps.Log.Printf("mail webhook org=%s project=%s: %s, %d parts", org, project, outcome, parts)
	} else {
		deps.Log.Printf("mail webhook: %s", outcome)
	}
	w.WriteHeader(http.StatusOK)
}

// dropMail records the token of a post that stores nothing, so it cannot be
// sent again with another body.
func dropMail(ctx context.Context, deps Deps, token, outcome string) (string, string, string, int, error) {
	if _, err := deps.Store.SeenWebhookToken(ctx, token); err != nil {
		return "", "", "", 0, err
	}
	return outcome, "", "", 0, nil
}

// receiveMail decides and records one verified post. It returns a word for
// the log line (never the subject, the addresses or the key).
func receiveMail(r *http.Request, deps Deps, token string, now time.Time) (outcome, org, project string, n int, err error) {
	ctx := r.Context()
	m, err := mailin.Read(r.PostFormValue)
	if err != nil {
		return dropMail(ctx, deps, token, "dropped: no From address")
	}
	local, ok := mailin.RecipientLocalPart(m.Recipient, deps.Mail.Domain)
	if !ok {
		return dropMail(ctx, deps, token, "dropped: not for the mail domain")
	}
	target, err := deps.Store.ProjectByMailAddress(ctx, local)
	if errors.Is(err, store.ErrNotFound) {
		return dropMail(ctx, deps, token, "dropped: unknown address")
	}
	if err != nil {
		return "", "", "", 0, err
	}
	if mailin.Dropped(m) {
		return dropMail(ctx, deps, token, "dropped: automatic, bulk or spam")
	}
	org, project = target.OrgID, target.ProjectID
	reason, member, err := mailDecision(ctx, deps, target, m)
	if err != nil {
		return "", org, project, 0, err
	}
	if m.MessageID == "" {
		m.MessageID = "<" + token + "@mailgun.invalid>"
	}
	in := store.MailIn{ProjectID: project, MessageID: m.MessageID, Token: token, Sender: m.FromAddress, SenderName: m.FromName,
		Subject: m.Subject, To: m.To, Cc: m.Cc, InReplyTo: m.InReplyTo, References: m.References, Body: m.BodyPlain,
		ReceivedAt: now, HeldReason: reason}
	// A team member's own message copied to the project is Out; a forward
	// is In, from the original sender when Go finds it (decision 7).
	if member {
		in.Direction = "out"
		if addr, name, fwd := mailin.Forwarded(m.Subject, m.BodyPlain); fwd {
			in.Direction = "in"
			if addr != "" {
				in.Sender, in.SenderName = addr, name
			}
		}
	}
	in.Parts, err = storeMailParts(ctx, r, deps, m, now)
	if err != nil {
		return "", org, project, 0, err
	}
	rec, err := deps.Store.RecordMail(ctx, org, in)
	if err != nil {
		return "", org, project, 0, err
	}
	removeUnreferenced(deps, append(rec.Expired, rec.Unkept...))
	if rec.Repeat {
		return "repeat", org, project, len(in.Parts), nil
	}
	outcome = "held: " + rec.HeldReason
	if rec.Status == "filed" {
		if err := fileMail(ctx, deps, org, project, rec.ID, ""); err != nil {
			// The message is recorded; its parts keep their bytes and the
			// PM sees them in the list. Mailgun need not retry.
			deps.Log.Printf("mail webhook org=%s project=%s: filing parts: %v", org, project, err)
		}
		outcome = "filed"
	}
	mailEvent(deps, org, project, rec.ID)
	return outcome, org, project, len(in.Parts), nil
}

// mailDecision says whether a message files ("") or is held, and why, in
// the plan's order: archived, then the sender, then the sender check, then
// the project's hourly cap and the org's allowance. member reports a
// confirmed member of the org sent it.
//
// Someone outside the org is filed only when Mailgun authenticated them and
// they reply to a message SiteWise sent for this project, or their address
// is exactly a firm's on this project (correspondence plan, A1). They never
// take the org's last tenth of its allowance.
func mailDecision(ctx context.Context, deps Deps, target store.MailTarget, m mailin.Message) (reason string, member bool, err error) {
	if target.Archived {
		return store.MailArchived, false, nil
	}
	user, err := deps.Store.MailMember(ctx, target.OrgID, m.FromAddress)
	if err != nil {
		return "", false, err
	}
	if user != "" {
		if !mailin.Authenticated(m) {
			return store.MailUnverifiedSender, true, nil
		}
		reason, err := admitMail(ctx, deps, target, 0)
		return reason, true, err
	}
	known, err := deps.Store.MailOutsideAdmitted(ctx, target.OrgID, target.ProjectID, m.FromAddress, m.InReplyTo, m.References)
	switch {
	case err != nil:
		return "", false, err
	case !known:
		return store.MailUnknownSender, false, nil
	case !mailin.Authenticated(m):
		return store.MailUnverifiedSender, false, nil
	}
	reason, err = admitMail(ctx, deps, target, outsideReserve)
	return reason, false, err
}

// admitMail is the last two checks for mail that would file: the project's
// hourly cap, then the org's allowance (reserve is the share an outside
// sender never takes).
func admitMail(ctx context.Context, deps Deps, target store.MailTarget, reserve float64) (string, error) {
	if !deps.limits.allow("mail-project:"+target.ProjectID, mailFiledPerProjectHour, time.Hour) {
		return store.MailBusy, nil
	}
	ok, _, err := orgUploadAllowance(ctx, deps, target.OrgID, reserve)
	if err != nil {
		return "", err
	}
	if !ok {
		return store.MailPlanLimit, nil
	}
	return "", nil
}

// storeMailParts stores the body document and each attachment by content
// hash. A part kept without bytes (a signature logo, a calendar invite)
// keeps its name and why. The hash comes from this put, never from the post.
func storeMailParts(ctx context.Context, r *http.Request, deps Deps, m mailin.Message, now time.Time) ([]store.MailPartIn, error) {
	var parts []store.MailPartIn
	if strings.TrimSpace(m.BodyPlain) != "" {
		name, body := mailin.BodyDocument(m, now)
		blob, err := deps.Blobs.Put(ctx, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		parts = append(parts, store.MailPartIn{N: 0, Filename: name, ContentType: "text/markdown", SHA256: blob.SHA256, Size: blob.Size})
	}
	for i, fh := range attachments(r) {
		n := i + 1
		name := mailin.AttachmentName(fh.Filename, n)
		p := store.MailPartIn{N: n, Filename: name, ContentType: mailin.OneLine(fh.Header.Get("Content-Type"), 255), Size: fh.Size}
		if p.Skipped = mailin.SkipPart(m, fh, name); p.Skipped == "" && n > mailMaxParts {
			p.Skipped = "not_kept"
		}
		if p.Skipped == "" {
			blob, err := putPart(ctx, deps.Blobs, fh)
			if err != nil {
				return nil, err
			}
			p.SHA256, p.Size = blob.SHA256, blob.Size
		}
		parts = append(parts, p)
	}
	return parts, nil
}

func putPart(ctx context.Context, blobs *files.Store, fh *multipart.FileHeader) (files.Blob, error) {
	f, err := fh.Open()
	if err != nil {
		return files.Blob{}, err
	}
	defer f.Close()
	return blobs.Put(ctx, f)
}

// attachments are the post's files in Mailgun's order: attachment-1, -2…
// Any other file field is ignored.
func attachments(r *http.Request) []*multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	type numbered struct {
		n  int
		fh *multipart.FileHeader
	}
	var list []numbered
	for field, fhs := range r.MultipartForm.File {
		rest, ok := strings.CutPrefix(field, "attachment-")
		n, err := strconv.Atoi(rest)
		if !ok || err != nil || n < 1 || n > 1000 || len(fhs) == 0 {
			continue
		}
		list = append(list, numbered{n, fhs[0]})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n < list[j].n })
	out := make([]*multipart.FileHeader, len(list))
	for i, x := range list {
		out[i] = x.fh
	}
	return out
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
	f := store.MailFilter{Held: r.URL.Query().Get("held") == "1"}
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
