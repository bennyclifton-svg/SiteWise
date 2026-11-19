package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"sitewise/internal/correspondence"
	"sitewise/internal/events"
	"sitewise/internal/export"
	"sitewise/internal/files"
	"sitewise/internal/store"
)

// The one send path (docs/plans/2026-11-17-correspondence-transmittals.md,
// Task 3; the wave index, agreement 7: all mail leaves here, and only on the
// PM's press of Send). Nothing here calls Jev or the LLM. Chat never sends:
// POST /projects/{id}/transmittals is noOperation, and chat opens the Send
// dialog filled in (chat_view.go).

const pathTransmittalSend = "transmittal_send"

func init() {
	routePaths["POST /projects/{id}/transmittals"] = pathTransmittalSend
	routePaths["GET /projects/{id}/mail/{message}/sheet"] = pathMailRead
	routePaths["GET /projects/{id}/mail/{message}/parts/{n}"] = pathMailRead
}

var transmittalRoutes = map[string]func(http.ResponseWriter, *http.Request, Deps){
	"POST /projects/{id}/transmittals":            postTransmittal,
	"GET /projects/{id}/mail/{message}/sheet":     getTransmittalSheet,
	"GET /projects/{id}/mail/{message}/parts/{n}": getMailPart,
}

// Outbox sends queued messages after the request that queued them: Start
// in a goroutine, a sweep each minute for rows queued more than two minutes
// ago (a restart, a crash), five tries with backoff, then failed. One
// process serves every request, so an in-memory set keeps a message to one
// sender at a time.
type Outbox struct {
	store  *store.Store
	blobs  *files.Store
	sender correspondence.Sender
	broker *events.Broker
	log    *log.Logger
	// From is the address mail leaves from (SITEWISE_MAIL_FROM).
	From string
	// GlobalPerDay caps every org's sends together; zero is
	// store.MailGlobalPerDay.
	GlobalPerDay int
	backoff      []time.Duration
	now          func() time.Time

	mu       sync.Mutex
	inflight map[string]bool
	wg       sync.WaitGroup
}

// NewOutbox returns nil when there is nothing to send with: no message is
// ever queued without a sender.
func NewOutbox(st *store.Store, blobs *files.Store, sender correspondence.Sender, from string, broker *events.Broker, logger *log.Logger) *Outbox {
	if sender == nil || from == "" {
		return nil
	}
	return &Outbox{store: st, blobs: blobs, sender: sender, From: from, broker: broker, log: logger,
		backoff: []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}, now: time.Now, inflight: map[string]bool{}}
}

// Start sends one queued message in the background.
func (o *Outbox) Start(org, project, id string) {
	o.mu.Lock()
	if o.inflight[id] {
		o.mu.Unlock()
		return
	}
	o.inflight[id] = true
	o.mu.Unlock()
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		defer func() { o.mu.Lock(); delete(o.inflight, id); o.mu.Unlock() }()
		o.run(org, project, id)
	}()
}

// Sweep starts every message queued and last tried before cutoff.
func (o *Outbox) Sweep(ctx context.Context, cutoff time.Time) int {
	refs, err := o.store.QueuedOutbound(ctx, cutoff)
	if err != nil {
		o.log.Printf("outbox sweep: %v", err)
		return 0
	}
	for _, r := range refs {
		o.Start(r.OrgID, r.ProjectID, r.ID)
	}
	return len(refs)
}

// Loop sweeps every minute until closing.
func (o *Outbox) Loop(closing <-chan struct{}) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-closing:
			return
		case <-t.C:
			o.Sweep(context.Background(), o.now().Add(-2*time.Minute))
		}
	}
}

// Wait blocks until sends in flight finish or ctx ends.
func (o *Outbox) Wait(ctx context.Context) {
	done := make(chan struct{})
	go func() { o.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (o *Outbox) run(org, project, id string) {
	for attempt := 0; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err := o.sendOnce(ctx, org, project, id)
		cancel()
		text := ""
		if err != nil {
			text = err.Error()
		}
		status, rerr := o.store.RecordSendAttempt(context.Background(), org, id, err == nil, text, o.now())
		if rerr != nil {
			// Not queued any more (sent, failed, or gone): nothing to do.
			return
		}
		o.event(org, project, id)
		if status != "queued" {
			o.log.Printf("outbox org=%s: message %s %s", org, shortID(id), status)
			return
		}
		wait := o.backoff[len(o.backoff)-1]
		if attempt < len(o.backoff) {
			wait = o.backoff[attempt]
		}
		time.Sleep(wait)
	}
}

func (o *Outbox) event(org, project, id string) {
	mailEvent(Deps{Store: o.store, Broker: o.broker, Log: o.log}, org, project, id)
}

// sendOnce builds the MIME message from the record and hands it to the
// mail service. Every byte attached is a part of this message, read by its
// hash: the sheet and this project's documents as they went.
func (o *Outbox) sendOnce(ctx context.Context, org, project, id string) error {
	m, err := o.store.MailMessageByID(ctx, org, project, id)
	if err != nil {
		return err
	}
	if m.SendStatus != "queued" {
		return nil
	}
	out, err := outgoingFrom(m)
	if err != nil {
		return err
	}
	out.From.Address = o.From
	for _, p := range m.Parts {
		if p.SHA256 == nil {
			continue
		}
		body, err := readBlob(o.blobs, p.SHA256)
		if err != nil {
			return fmt.Errorf("attachment %d: %w", p.N, err)
		}
		out.Files = append(out.Files, correspondence.File{Name: p.Filename, ContentType: p.ContentType, Body: body})
	}
	raw, err := correspondence.Build(out)
	if err != nil {
		return err
	}
	return o.sender.Send(ctx, o.From, correspondence.Recipients(out), raw)
}

func readBlob(blobs *files.Store, sum []byte) ([]byte, error) {
	f, err := blobs.Open(sum)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, store.MailSendMaxBytes+1))
}

// outgoingFrom reads the stored record back into headers. The From name,
// Reply-To and Cc were fixed when it was queued; nothing here is new text.
func outgoingFrom(m store.MailMessage) (correspondence.Outgoing, error) {
	to, err := correspondence.ParseRecipients([]string{m.To})
	if err != nil {
		return correspondence.Outgoing{}, err
	}
	cc, err := correspondence.ParseRecipients([]string{m.Cc})
	if err != nil {
		return correspondence.Outgoing{}, err
	}
	from, err := mail.ParseAddress(m.Sender)
	if err != nil {
		return correspondence.Outgoing{}, err
	}
	replyTo, err := correspondence.ParseRecipients([]string{m.ReplyTo})
	if err != nil {
		return correspondence.Outgoing{}, err
	}
	inReplyTo, refs := m.OutboundThread()
	return correspondence.Outgoing{From: *from, ReplyTo: replyTo, To: to, Cc: cc, Subject: m.Subject, Body: m.Body,
		MessageID: m.MessageID, InReplyTo: inReplyTo, References: refs, Date: m.ReceivedAt}, nil
}

// transmittalBody is the Send dialog's request.
type transmittalBody struct {
	To          []string `json:"to"`
	Cc          []string `json:"cc"`
	Subject     string   `json:"subject"`
	Note        string   `json:"note"`
	DocumentIDs []string `json:"document_ids"`
	InReplyTo   string   `json:"in_reply_to"`
	PackageID   string   `json:"package_id"`
	// Retry sends a message that was not sent again; nothing else is read.
	Retry string `json:"retry"`
	// Issue attaches an issued version of this project's report, its frozen
	// PDF as issued (the report plan's Send issue N…).
	Issue string `json:"issue"`
	// RFI is the register's RFI this message sends; a reply in its thread
	// then answers it by rule.
	RFI string `json:"rfi"`
}

// sendError is a refusal in the PM's words, with its status.
type sendError struct {
	status int
	msg    string
}

func (e *sendError) Error() string { return e.msg }

// queueOutbound is the one way out for every plan (agreement 7). In order:
// sending is set up, the sender's email is confirmed, the project is live,
// the daily cap, 1 to 20 addresses with no line break, every document a
// filed document of this project, at most 10 MB; then, under the project
// lock, the number, the sheet, the record, queued; then the send starts.
// Callers keep the returned message's id on their own record.
func queueOutbound(ctx context.Context, deps Deps, session store.Session, project string, in store.Outbound, sheet bool, note string) (store.MailMessage, error) {
	if deps.Outbox == nil {
		return store.MailMessage{}, &sendError{http.StatusServiceUnavailable, "Sending mail is not set up on this server, so nothing was sent."}
	}
	who, err := deps.Store.MailSenderContext(ctx, session.OrgID, project, session.UserID)
	if err != nil {
		return store.MailMessage{}, err
	}
	if !who.Verified {
		return store.MailMessage{}, &sendError{http.StatusForbidden, "Confirm your email address to send mail. Use the link we emailed you, or send a new one from the Account page."}
	}
	// A free plan sends only from an account a week old, and less a day
	// (security review H3; the owner's numbers to change).
	in.Global = deps.Outbox.GlobalPerDay
	if deps.Billing != nil {
		b, err := deps.Store.OrgBilling(ctx, session.OrgID)
		if err != nil {
			return store.MailMessage{}, err
		}
		if b.Effective().MonthlyCents == 0 {
			if time.Since(who.UserSince) < mailFreeAccountAge {
				return store.MailMessage{}, &sendError{http.StatusForbidden, "Sending mail needs a paid plan, or an account at least 7 days old. Upgrade on the Account page, or send from your own email for now."}
			}
			in.PerDay, in.RecipientsPerDay = store.MailFreeSendPerDay, store.MailFreeRecipientsPerDay
		}
	}
	for _, v := range append([]string{in.Subject}, append(in.To, in.Cc...)...) {
		if correspondence.CleanHeader(v) != nil {
			return store.MailMessage{}, &sendError{http.StatusBadRequest, "A line break is not allowed in an address or the subject."}
		}
	}
	to, err := correspondence.ParseRecipients(in.To)
	if err != nil {
		return store.MailMessage{}, &sendError{http.StatusBadRequest, err.Error()}
	}
	cc, err := correspondence.ParseRecipients(in.Cc)
	if err != nil {
		return store.MailMessage{}, &sendError{http.StatusBadRequest, err.Error()}
	}
	// Decision 2: Cc the PM, Reply-To the PM and the project's address.
	pm := mail.Address{Address: strings.ToLower(who.Email)}
	if !containsAddress(to, pm.Address) && !containsAddress(cc, pm.Address) {
		cc = append(cc, pm)
	}
	if len(to) == 0 || len(to)+len(cc) > store.MailMaxRecipients+1 {
		return store.MailMessage{}, &sendError{http.StatusBadRequest, fmt.Sprintf("Send to between 1 and %d addresses.", store.MailMaxRecipients)}
	}
	replyTo := []mail.Address{pm}
	if address, err := mailAddress(ctx, deps, session.OrgID, project); err == nil && address != "" {
		replyTo = append(replyTo, mail.Address{Address: address})
	}
	from := fromAddress(deps.Outbox.From)
	in.ProjectID, in.SentBy, in.From = project, session.UserID, from.String()
	// Who sent it is said in a fixed footer, never in From's name.
	in.Body = strings.TrimRight(in.Body, "\n") + "\n\n--\n" + sentByLine(who) + "\n"
	in.To, in.Cc, in.ReplyTo = formatAll(to), formatAll(cc), formatAll(replyTo)
	in.MessageIDDomain = messageDomain(deps)
	if sheet {
		in.Sheet = func(number int, docs []store.SentDocument) (store.OutboundFile, error) {
			return transmittalSheet(ctx, deps, who, number, from, to, cc, in.Subject, note, docs)
		}
	}
	var sheetSum []byte
	if in.Sheet != nil {
		render := in.Sheet
		in.Sheet = func(number int, docs []store.SentDocument) (store.OutboundFile, error) {
			f, err := render(number, docs)
			sheetSum = f.SHA256
			return f, err
		}
	}
	m, err := deps.Store.QueueOutbound(ctx, session.OrgID, in, time.Now())
	if sheetSum != nil {
		finishStaged(deps, [][]byte{sheetSum})
	}
	if err != nil {
		return m, err
	}
	mailEvent(deps, session.OrgID, project, m.ID)
	deps.Outbox.Start(session.OrgID, project, m.ID)
	return m, nil
}

// fromAddress is always "SiteWise" at SITEWISE_MAIL_FROM: an org's own
// words never reach From (security review H3). The org and the member who
// sent it are named in the footer and on the sheet. Replies reach the PM
// and the project through Reply-To.
func fromAddress(from string) mail.Address {
	return mail.Address{Name: "SiteWise", Address: from}
}

// mailFreeAccountAge: a free plan's account sends after a week.
const mailFreeAccountAge = 7 * 24 * time.Hour

// sentByLine is the fixed footer naming the sender and the org.
func sentByLine(who store.MailSender) string {
	clean := func(s string) string {
		return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return ' '
			}
			return r
		}, s)), " ")
	}
	return "Sent by " + clean(who.Email) + " of " + clean(who.OrgName) + " using SiteWise."
}

func containsAddress(list []mail.Address, addr string) bool {
	for _, a := range list {
		if strings.EqualFold(a.Address, addr) {
			return true
		}
	}
	return false
}

// formatAll writes addresses for a header and the record: a bare address
// alone, a named one as "Name <address>" (quoted and encoded by net/mail).
func formatAll(list []mail.Address) []string {
	out := make([]string, len(list))
	for i, a := range list {
		out[i] = addressWords(a)
	}
	return out
}

func messageDomain(deps Deps) string {
	if deps.Mail != nil {
		return deps.Mail.Domain
	}
	if u, err := url.Parse(deps.PublicOrigin); err == nil && u.Hostname() != "" && u.Hostname() != "localhost" && !strings.HasPrefix(u.Hostname(), "127.") {
		return u.Hostname()
	}
	return "sitewise.invalid"
}

// transmittalSheet is the numbered one-page sheet in the house style:
// project, number, date, from, to, the documents as they go, the note.
func transmittalSheet(ctx context.Context, deps Deps, who store.MailSender, number int, from mail.Address, to, cc []mail.Address, subject, note string, docs []store.SentDocument) (store.OutboundFile, error) {
	date := time.Now().In(sydneyLocation())
	rows := make([]export.Row, 0, len(docs))
	for _, d := range docs {
		title := d.Title
		if title == "" {
			title = d.Filename
		}
		rows = append(rows, export.Row{Cells: []export.Cell{
			{Runs: []export.Run{{Text: orDash(d.Number)}}}, {Runs: []export.Run{{Text: title}}}, {Runs: []export.Run{{Text: orDash(d.Revision)}}},
		}})
	}
	details := fmt.Sprintf("Project: %s\nNumber: %s\nDate: %s\nFrom: %s\nTo: %s", who.ProjectName, TransmittalNumber(number), date.Format("2 January 2006"), sentByLine(who), strings.Join(formatAll(to), ", "))
	if len(cc) > 0 {
		details += "\nCc: " + strings.Join(formatAll(cc), ", ")
	}
	if subject != "" {
		details += "\nSubject: " + subject
	}
	doc := export.Doc{Title: "Transmittal " + TransmittalNumber(number), Subtitle: who.ProjectName, Date: date.Format("2006-01-02"),
		Sections: []export.Section{{Title: "Details", Blocks: []export.Block{{Text: details}}}}}
	if len(rows) > 0 {
		doc.Sections = append(doc.Sections, export.Section{Title: "Documents", Blocks: []export.Block{{Table: &export.Table{
			Columns: []export.Column{{Share: 0.25}, {Share: 0.6}, {Share: 0.15}},
			Head:    []export.Cell{{Runs: []export.Run{{Text: "No.", Bold: true}}, Head: true}, {Runs: []export.Run{{Text: "Title", Bold: true}}, Head: true}, {Runs: []export.Run{{Text: "Rev", Bold: true}}, Head: true}},
			Rows:    rows,
		}}}})
	}
	if strings.TrimSpace(note) != "" {
		doc.Sections = append(doc.Sections, export.Section{Title: "Note", Blocks: []export.Block{{Text: note}}})
	}
	pdf, err := export.PDF(doc)
	if err != nil {
		return store.OutboundFile{}, err
	}
	blob, err := deps.Blobs.Put(ctx, bytes.NewReader(pdf))
	if err != nil {
		return store.OutboundFile{}, err
	}
	// Staged until the send is recorded; the mail sweep takes it if the
	// queue never commits (security review L6).
	if err := deps.Store.StageMailBlob(ctx, blob.SHA256); err != nil {
		return store.OutboundFile{}, err
	}
	return store.OutboundFile{Name: "Transmittal " + TransmittalNumber(number) + ".pdf", ContentType: "application/pdf", SHA256: blob.SHA256, Size: blob.Size}, nil
}

// TransmittalNumber is "TR-001": one series per project, never reused.
func TransmittalNumber(n int) string { return fmt.Sprintf("TR-%03d", n) }

func sydneyLocation() *time.Location {
	if loc, err := time.LoadLocation("Australia/Sydney"); err == nil {
		return loc
	}
	return time.FixedZone("AEST", 10*3600)
}

// postTransmittal is the PM's press of Send (or Retry). It answers 202 once
// the message is queued; "Sent" arrives as a mail event.
func postTransmittal(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	project := r.PathValue("id")
	var body transmittalBody
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Retry != "" {
		retryTransmittal(w, r, deps, session, project, body.Retry)
		return
	}
	subject := strings.TrimSpace(body.Subject)
	if len(subject) > 300 || len(body.Note) > 10000 || len(body.DocumentIDs) > 200 {
		http.Error(w, "The subject, the note or the list of documents is too long.", http.StatusBadRequest)
		return
	}
	in := store.Outbound{To: body.To, Cc: body.Cc, Subject: subject, Body: body.Note, DocumentIDs: body.DocumentIDs,
		InReplyTo: body.InReplyTo, PackageID: body.PackageID}
	if body.Issue != "" {
		f, err := issueFile(r.Context(), deps, session.OrgID, project, body.Issue)
		if err != nil {
			writeSendError(w, deps, err)
			return
		}
		in.Files = append(in.Files, f)
	}
	if body.RFI != "" {
		if _, _, err := deps.Store.RFIToSend(r.Context(), session.OrgID, project, body.RFI); err != nil {
			writeSendError(w, deps, err)
			return
		}
	}
	m, err := queueOutbound(r.Context(), deps, session, project, in, true, body.Note)
	if err != nil {
		writeSendError(w, deps, err)
		return
	}
	if body.RFI != "" {
		if err := deps.Store.LinkRFIMessage(r.Context(), session.OrgID, project, body.RFI, m.ID); err != nil {
			deps.Log.Printf("transmittal org=%s: the RFI was not linked to its message: %v", session.OrgID, err)
		} else if deps.Broker != nil {
			deps.Broker.Wake(session.OrgID)
		}
	}
	writeJSON(w, http.StatusAccepted, m)
}

func retryTransmittal(w http.ResponseWriter, r *http.Request, deps Deps, session store.Session, project, id string) {
	if deps.Outbox == nil {
		http.Error(w, "Sending mail is not set up on this server, so nothing was sent.", http.StatusServiceUnavailable)
		return
	}
	err := deps.Store.RetryOutbound(r.Context(), session.OrgID, project, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrMailNotRetryable):
		http.Error(w, "Only a message that was not sent can be sent again.", http.StatusConflict)
		return
	case err != nil:
		readFailed(w, deps, "retry", err)
		return
	}
	deps.Outbox.Start(session.OrgID, project, id)
	mailEvent(deps, session.OrgID, project, id)
	m, err := deps.Store.MailMessageByID(r.Context(), session.OrgID, project, id)
	if err != nil {
		readFailed(w, deps, "mail", err)
		return
	}
	writeJSON(w, http.StatusAccepted, m)
}

func writeSendError(w http.ResponseWriter, deps Deps, err error) {
	var se *sendError
	var big store.MailTooLarge
	switch {
	case errors.As(err, &se):
		http.Error(w, se.msg, se.status)
	case errors.As(err, &big):
		http.Error(w, fmt.Sprintf("The documents come to %s, more than the 10 MB a message can carry. Send fewer at a time.", sizeLabel(big.Bytes)), http.StatusRequestEntityTooLarge)
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "A document or message named is not in this project.", http.StatusBadRequest)
	case errors.Is(err, store.ErrProjectArchived):
		http.Error(w, "This project is archived. Restore it on the Projects page to send mail.", http.StatusConflict)
	case errors.Is(err, store.ErrMailGlobalCap):
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "SiteWise has sent all the mail it sends in a day. Try again tomorrow, or send from your own email.", http.StatusTooManyRequests)
	case errors.Is(err, store.ErrMailCap):
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "Your team has sent all the messages or recipients it sends in a day. Try again tomorrow.", http.StatusTooManyRequests)
	case errors.Is(err, store.ErrInvalidInput):
		http.Error(w, strings.TrimPrefix(err.Error(), store.ErrInvalidInput.Error()+": "), http.StatusBadRequest)
	default:
		deps.Log.Printf("transmittal: %v", err)
		http.Error(w, "send failed", http.StatusInternalServerError)
	}
}

// getTransmittalSheet downloads a sent message's sheet (part 0).
func getTransmittalSheet(w http.ResponseWriter, r *http.Request, deps Deps) {
	r.SetPathValue("n", "0")
	getMailPart(w, r, deps)
}

// getMailPart downloads one part of a message of this project by its
// stored hash: a document deleted after sending still downloads from its
// transmittal.
func getMailPart(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, false)
	if !ok {
		return
	}
	m, ok := mailMessage(w, r, deps, session.OrgID)
	if !ok {
		return
	}
	for _, p := range m.Parts {
		if fmt.Sprint(p.N) == r.PathValue("n") && p.SHA256 != nil {
			serveFile(w, r, deps, p.SHA256, safePartType(p.ContentType), p.Filename)
			return
		}
	}
	http.NotFound(w, r)
}

// issueFile is an issued report version's frozen PDF as a file to send. Its
// size is read from the stored bytes so the 10 MB limit holds.
func issueFile(ctx context.Context, deps Deps, org, project, version string) (store.OutboundFile, error) {
	v, err := deps.Store.IssuedVersionOf(ctx, org, project, version)
	if err != nil {
		return store.OutboundFile{}, err
	}
	path, err := deps.Blobs.Path(v.SHA256)
	if err != nil {
		return store.OutboundFile{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return store.OutboundFile{}, err
	}
	return store.OutboundFile{Name: fmt.Sprintf("%s issue %d.pdf", cleanFileTitle(v.Title), v.Number), ContentType: "application/pdf", SHA256: v.SHA256, Size: info.Size()}, nil
}

// cleanFileTitle keeps a title safe as a file name: no path or control
// characters.
func cleanFileTitle(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`/\:*?"<>|`, r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "Report"
	}
	if len([]rune(s)) > 120 {
		s = string([]rune(s)[:120])
	}
	return s
}

// safePartType is a part's type when it is on a short safe list, else
// application/octet-stream (mail review L5). Every part downloads as an
// attachment either way: a sender's file is never shown in the app's origin.
func safePartType(ctype string) string {
	t, _, err := mime.ParseMediaType(ctype)
	if err != nil {
		return "application/octet-stream"
	}
	switch t {
	case "application/pdf", "image/png", "image/jpeg":
		return t
	case "text/plain":
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}
