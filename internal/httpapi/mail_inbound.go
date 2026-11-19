package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/mailin"
	"sitewise/internal/store"
)

// The mail webhook (email intake Task 6), after the security review of 19
// November 2026. Mailgun's post is signed over its timestamp and token
// only, so the post is read in order and the signature is checked as soon
// as its three fields have arrived, before any file part is read where
// Mailgun's order allows; a read deadline bounds a slow post. The sender is
// authenticated only by a DKIM signature that verifies over the raw message
// (mailin.Verify); no header the sender can write is trusted. Token,
// Message-Id and the held cap are checked before any byte is stored, and
// every byte stored is staged until a part names it, so nothing is left
// behind by a repeat, a held-full message or a crash.

// MailIn is email intake's configuration. Nil Deps.Mail is off: the webhook
// answers 404 and no project address is shown.
type MailIn struct {
	// Domain receives project mail ("in.sitewise.au").
	Domain string
	// SigningKey is the Mailgun account's HTTP webhook signing key. It is
	// never logged or returned.
	SigningKey string
	// Lookup answers DKIM's key lookups: the system resolver, or a stub.
	Lookup mailin.LookupTXT
	slots  chan struct{}
	now    func() time.Time
}

// NewMailIn switches email intake on, or returns nil when either setting
// is blank.
func NewMailIn(domain, key string) *MailIn {
	if domain == "" || key == "" {
		return nil
	}
	return &MailIn{Domain: strings.ToLower(domain), SigningKey: key, Lookup: mailin.NetLookupTXT}
}

// WithDKIMLookup answers DKIM's key lookups with lookup (tests and the e2e
// server, so nothing reaches the network).
func (m *MailIn) WithDKIMLookup(lookup mailin.LookupTXT) *MailIn {
	if m != nil {
		m.Lookup = lookup
	}
	return m
}

func (m *MailIn) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

// mailReadDeadline bounds the whole post, so a slow sender cannot hold a slot.
const mailReadDeadline = 60 * time.Second

// mailFieldMax is how much of one small form field is read; the rest is
// discarded. A body beyond it is cut anyway (mailin.MaxBodyRunes).
const mailFieldMax = 1 << 20

// Refusal kinds (review LOW-4). A bad signature or timestamp is refused for
// good: 406, which Mailgun does not retry, and it counts against the
// address. A lockout, a deadline or a size or read failure answers 503, so a
// genuine post Mailgun retries later; those do not count. A post that is not
// multipart (Mailgun's raw route always is) answers 415 and counts.
type refusal struct {
	why    string
	status int
	count  bool
}

var (
	refuseSignature = refusal{"signature", http.StatusNotAcceptable, true}
	refuseTimestamp = refusal{"timestamp", http.StatusNotAcceptable, true}
	refuseFormat    = refusal{"not multipart", http.StatusUnsupportedMediaType, true}
	refuseRead      = refusal{"unreadable, too large or too slow", http.StatusServiceUnavailable, false}
	refuseLockout   = refusal{"too many refusals from this address", http.StatusServiceUnavailable, false}
)

// refuseMail answers a refused post with the one plain body, whatever the
// reason, and never reads the rest of it.
func refuseMail(w http.ResponseWriter, r *http.Request, deps Deps, why refusal) {
	if why.count {
		deps.limits.allow("mail-miss:"+clientIP(r), mailMissesPerIP, limitWindow)
	}
	deps.Log.Printf("mail webhook: refused (%s)", why.why)
	w.Header().Set("Connection", "close")
	if why.status == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", "300")
	}
	http.Error(w, msgMailRefused, why.status)
	_ = http.NewResponseController(w).Flush()
}

// RemoveMailTemps deletes temporary files a post left behind when the
// process stopped mid-request (review MEDIUM-2). The server runs it once at
// start.
func RemoveMailTemps() int {
	names, _ := filepath.Glob(filepath.Join(os.TempDir(), "sitewise-mail-*"))
	n := 0
	for _, f := range names {
		if os.Remove(f) == nil {
			n++
		}
	}
	return n
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
		refuseMail(w, r, deps, refuseLockout)
		return
	}
	// The deadline is set before a slot is taken, so a slow post can hold
	// one for at most this long (security review H1).
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(mailReadDeadline))
	select {
	case cfg.slots <- struct{}{}:
		defer func() { <-cfg.slots }()
	default:
		w.Header().Set("Retry-After", "60")
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, mailin.MaxPostBytes)
	now := cfg.clock()
	post, why := readMailPost(r, cfg.SigningKey, now)
	defer post.close()
	if why != nil {
		refuseMail(w, r, deps, *why)
		return
	}
	outcome, org, project, parts, err := receiveMail(r.Context(), deps, post, now)
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

// mailPost is one verified post: its small fields, its file parts (on disk)
// and the raw message when Mailgun sent it.
type mailPost struct {
	fields  map[string]string
	files   []mailin.Attachment
	raw     []byte
	rawFile string // the raw message spooled before the signature arrived
	temps   []string
}

func (p *mailPost) field(name string) string { return p.fields[name] }

func (p *mailPost) close() {
	for _, t := range p.temps {
		_ = os.Remove(t)
	}
}

// readMailPost reads a multipart post in order and checks the signature as
// soon as timestamp, token and signature have arrived. A part after a bad
// signature is never read. A part that comes before them (Task 0 records
// Mailgun's order), the raw message included, is spooled to a temporary
// file rather than held in memory, and removed when the request ends.
func readMailPost(r *http.Request, key string, now time.Time) (*mailPost, *refusal) {
	post := &mailPost{fields: map[string]string{}}
	verified := false
	check := func() *refusal {
		if verified {
			return nil
		}
		ts, token, sig := post.fields["timestamp"], post.fields["token"], post.fields["signature"]
		if ts == "" || token == "" || sig == "" {
			return nil
		}
		if len(token) > 256 || !mailin.VerifySignature(key, ts, token, sig) {
			return &refuseSignature
		}
		if !mailin.Fresh(ts, now) {
			return &refuseTimestamp
		}
		verified = true
		return nil
	}
	ctype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ctype != "multipart/form-data" {
		return post, &refuseFormat
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return post, &refuseFormat
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return post, &refuseRead
		}
		name, filename := part.FormName(), part.FileName()
		switch {
		case name == mailin.RawField && verified:
			raw, err := io.ReadAll(part)
			if err != nil {
				return post, &refuseRead
			}
			post.raw = raw
		case name == mailin.RawField:
			a, temp, err := spool(name, "", "", part)
			if temp != "" {
				post.temps = append(post.temps, temp)
			}
			if err != nil {
				return post, &refuseRead
			}
			_ = a
			post.rawFile = temp
		case filename == "" && !strings.HasPrefix(name, "attachment-"):
			v, err := io.ReadAll(io.LimitReader(part, mailFieldMax))
			if err != nil {
				return post, &refuseRead
			}
			_, _ = io.Copy(io.Discard, part)
			post.fields[name] = string(v)
			if why := check(); why != nil {
				return post, why
			}
		default:
			a, temp, err := spool(name, filename, part.Header.Get("Content-Type"), part)
			if temp != "" {
				post.temps = append(post.temps, temp)
			}
			if err != nil {
				return post, &refuseRead
			}
			post.files = append(post.files, a)
		}
	}
	if why := check(); why != nil {
		return post, why
	}
	if !verified {
		return post, &refuseSignature
	}
	if post.rawFile != "" {
		raw, err := os.ReadFile(post.rawFile)
		if err != nil {
			return post, &refuseRead
		}
		post.raw = raw
	}
	sort.SliceStable(post.files, func(i, j int) bool {
		return attachmentNumber(post.files[i].Field) < attachmentNumber(post.files[j].Field)
	})
	return post, nil
}

func attachmentNumber(field string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(field, "attachment-"))
	if err != nil {
		return 1 << 30
	}
	return n
}

// spool writes one file part to a temporary file.
func spool(field, filename, ctype string, r io.Reader) (mailin.Attachment, string, error) {
	f, err := os.CreateTemp("", "sitewise-mail-*")
	if err != nil {
		return mailin.Attachment{}, "", err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return mailin.Attachment{}, f.Name(), err
	}
	path := f.Name()
	return mailin.Attachment{Field: field, Name: filename, ContentType: ctype, Size: n,
		Open: func() (io.ReadCloser, error) { return os.Open(path) }}, path, nil
}

// dropMail records the token of a post that stores nothing, so it cannot be
// sent again with another body.
func dropMail(ctx context.Context, deps Deps, token, outcome string) (string, string, string, int, error) {
	if _, err := deps.Store.SeenWebhookToken(ctx, token); err != nil {
		return "", "", "", 0, err
	}
	return outcome, "", "", 0, nil
}

// readMessage reads the message from the raw message when Mailgun sent it,
// and from the parsed fields otherwise; it makes no DNS lookup. raw is true
// when the message came from an unambiguous raw message (only that can be
// authenticated). A raw message with a duplicated header, two From headers
// or too many headers is read from the fields for the PM to see and is never
// trusted.
func readMessage(post *mailPost) (m mailin.Message, files []mailin.Attachment, raw bool, err error) {
	recipient, envelope := post.field("recipient"), strings.ToLower(strings.TrimSpace(post.field("sender")))
	if len(post.raw) > 0 {
		if m, files, err := mailin.ParseRaw(post.raw); err == nil {
			m.Recipient, m.EnvelopeSender = recipient, envelope
			return m, files, true, nil
		}
	}
	m, err = mailin.Read(post.field)
	if err != nil {
		return m, nil, false, err
	}
	m.Recipient, m.EnvelopeSender = recipient, envelope
	return m, post.files, false, nil
}

// rawHash is the hash a repeated Message-ID is compared by: the raw message,
// or the parsed post's own words.
func rawHash(post *mailPost, m mailin.Message) []byte {
	h := sha256.New()
	if len(post.raw) > 0 {
		h.Write(post.raw)
	} else {
		h.Write([]byte(m.FromAddress + "\x00" + m.Subject + "\x00" + m.BodyPlain))
	}
	return h.Sum(nil)
}

// receiveMail decides and records one verified post. The project is found
// before any DNS lookup, so mail to no project costs nothing (review
// MEDIUM-1). It returns a word for the log line (never the subject, the
// addresses or the key).
func receiveMail(ctx context.Context, deps Deps, post *mailPost, now time.Time) (outcome, org, project string, n int, err error) {
	token := post.field("token")
	m, atts, fromRaw, err := readMessage(post)
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
	authenticated := false
	if fromRaw {
		// Only what the verifying signature signed counts (review HIGH-1).
		if signed, ok := mailin.Verify(ctx, post.raw, m.FromAddress, deps.Mail.Lookup); ok {
			authenticated = mailin.Trust(&m, signed, now)
		}
	}
	if !authenticated {
		// Words no signature vouched for never thread, link or admit.
		m.InReplyTo, m.References = "", ""
	}
	org, project = target.OrgID, target.ProjectID
	reason, member, err := mailDecision(ctx, deps, target, m, authenticated)
	if err != nil {
		return "", org, project, 0, err
	}
	if m.MessageID == "" {
		m.MessageID = "<" + token + "@mailgun.invalid>"
	}
	// Nothing is stored for a repeat, and nothing kept for held mail that
	// is full (security review H2).
	hash := rawHash(post, m)
	pre, err := deps.Store.MailPrecheck(ctx, org, project, token, m.MessageID, hash)
	if err != nil {
		return "", org, project, 0, err
	}
	if pre.Repeat {
		_, _ = deps.Store.SeenWebhookToken(ctx, token)
		return "repeat", org, project, 0, nil
	}
	keep := reason == "" || !pre.HeldFull
	in := store.MailIn{ProjectID: project, MessageID: m.MessageID, Token: token, Sender: m.FromAddress, SenderName: m.FromName,
		Subject: m.Subject, To: m.To, Cc: m.Cc, InReplyTo: m.InReplyTo, References: m.References, Body: m.BodyPlain,
		ReceivedAt: now, HeldReason: reason, RawSHA256: hash}
	// A confirmed member's own authenticated message copied to the project
	// is Out. A forward is In, from the original sender Go finds, shown as
	// forwarded by the member. Nothing but an authenticated member's mail,
	// its Subject signed (Trust), is ever rewritten (decision 7; reviews C1,
	// HIGH-1); a rewritten forward links no firm or package by rule (LOW-1).
	if member && authenticated {
		in.Direction = "out"
		if addr, name, fwd := mailin.Forwarded(m.Subject, m.BodyPlain); fwd {
			in.Direction = "in"
			if addr != "" {
				shown := name
				if shown == "" {
					shown = addr
				}
				in.Sender, in.SenderName = addr, mailin.OneLine(shown+" (forwarded by "+m.FromAddress+")", 200)
				in.NoRuleLinks = true
			}
		}
	}
	var staged [][]byte
	defer func() { finishStaged(deps, staged, token) }()
	in.Parts, staged, err = storeMailParts(ctx, deps, m, atts, now, keep, token)
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
// confirmed member of the org wrote From. authenticated is mailin.Verify's.
//
// Someone outside the org is filed only when authenticated and they reply
// to a message SiteWise sent them for this project, or their address is
// exactly a firm's on this project (correspondence plan, A1; review M1).
// They never take the org's last tenth of its allowance.
func mailDecision(ctx context.Context, deps Deps, target store.MailTarget, m mailin.Message, authenticated bool) (reason string, member bool, err error) {
	if target.Archived {
		return store.MailArchived, false, nil
	}
	user, err := deps.Store.MailMember(ctx, target.OrgID, m.FromAddress)
	if err != nil {
		return "", false, err
	}
	if user != "" {
		if !authenticated {
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
	case !authenticated:
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
// hash, staging every hash it puts (finishStaged clears them). keep false
// records every part by name only. The hash comes from this put, never from
// the post.
func storeMailParts(ctx context.Context, deps Deps, m mailin.Message, atts []mailin.Attachment, now time.Time, keep bool, request string) ([]store.MailPartIn, [][]byte, error) {
	var parts []store.MailPartIn
	var staged [][]byte
	put := func(r io.Reader) (files.Blob, error) {
		blob, err := deps.Blobs.Put(ctx, r)
		if err != nil {
			return blob, err
		}
		staged = append(staged, blob.SHA256)
		return blob, deps.Store.StageMailBlob(ctx, blob.SHA256, request)
	}
	if strings.TrimSpace(m.BodyPlain) != "" {
		name, body := mailin.BodyDocument(m, now)
		p := store.MailPartIn{N: 0, Filename: name, ContentType: "text/markdown", Size: int64(len(body)), Skipped: "not_kept"}
		if keep {
			blob, err := put(bytes.NewReader(body))
			if err != nil {
				return nil, staged, err
			}
			p.SHA256, p.Size, p.Skipped = blob.SHA256, blob.Size, ""
		}
		parts = append(parts, p)
	}
	for i, a := range atts {
		n := i + 1
		name := mailin.AttachmentName(a.Name, n)
		p := store.MailPartIn{N: n, Filename: name, ContentType: mailin.OneLine(a.ContentType, 255), Size: a.Size}
		if p.Skipped = mailin.SkipPart(m, a, name); p.Skipped == "" && (n > mailMaxParts || !keep) {
			p.Skipped = "not_kept"
		}
		if p.Skipped == "" {
			f, err := a.Open()
			if err != nil {
				return nil, staged, err
			}
			blob, err := put(f)
			f.Close()
			if err != nil {
				return nil, staged, err
			}
			p.SHA256, p.Size = blob.SHA256, blob.Size
		}
		parts = append(parts, p)
	}
	return parts, staged, nil
}

// finishStaged settles one request's staged bytes: removed at once when
// nothing references them and no other request stages them; the staging
// row is cleared only when the bytes were removed or a part now names them,
// so one request never clears another's (review LOW-5). Anything left is
// the sweep's.
func finishStaged(deps Deps, sums [][]byte, request string) {
	if len(sums) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, sum := range sums {
		settleStaged(ctx, deps.Store, deps.Blobs, sum, request, deps.Log.Printf)
	}
}

// settleStaged removes or keeps one staged blob, and clears this request's
// staging only when it is settled: the bytes removed or already gone, named
// by a part, or still staged by another request, whose own row then keeps
// them until that request settles (review LOW-5). A blob just reused by a
// concurrent put stays staged for the sweep.
func settleStaged(ctx context.Context, st *store.Store, blobs *files.Store, sum []byte, request string, logf func(string, ...any)) bool {
	used, others := false, false
	removed, err := blobs.RemoveNow(sum, func() (bool, error) {
		var err error
		if used, err = st.BlobReferenced(ctx, sum); err != nil || used {
			return used, err
		}
		others, err = st.MailBlobStagedByOthers(ctx, sum, request)
		return others, err
	})
	if err != nil {
		logf("mail: stored file kept: %v", err)
		return false
	}
	if !removed && !used && !others {
		// RemoveNow did not ask: the bytes are gone, or a put just reused them.
		path, err := blobs.Path(sum)
		if err != nil {
			return false
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	if err := st.UnstageMailBlob(ctx, sum, request); err != nil {
		logf("mail: staging not cleared: %v", err)
		return false
	}
	return true
}

// sweepStagedBlobs settles stagings more than an hour old: a crash between
// a put and its record, or a sheet whose send was never recorded (reviews
// H2, L6).
func sweepStagedBlobs(ctx context.Context, st *store.Store, blobs *files.Store, logf func(string, ...any)) int {
	stale, err := st.StaleMailBlobs(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		logf("mail sweep: %v", err)
		return 0
	}
	n := 0
	for _, b := range stale {
		if settleStaged(ctx, st, blobs, b.SHA256, b.Request, logf) {
			n++
		}
	}
	return n
}
