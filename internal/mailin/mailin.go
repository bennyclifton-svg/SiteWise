// Package mailin reads and judges one inbound message as Mailgun posts it
// (docs/plans/2026-11-14-email-intake.md, Task 3). No database and no HTTP:
// small functions over values, so untrusted input is handled in one place.
//
// Task 0 of the plan (the owner's check of a real Mailgun post) has not been
// run. Everything here follows Mailgun's documentation, and each fact Task 0
// must confirm is held by one function, named in the plan's "Task 0
// findings: not yet run" section:
//
//   - the signature and its freshness: VerifySignature, Fresh
//   - the raw message's field, and bytes DKIM can verify: RawField, Verify
//     (authenticated.go); no header the sender can write is trusted
//   - body-plain for an HTML-only message: PlainBody
//   - how recipient is written (Bcc, several addresses): RecipientLocalPart
//   - In-Reply-To and References in message-headers: ThreadHeaders
//   - the largest post: MaxPostBytes
//
// Sources: https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/receive-http,
// https://documentation.mailgun.com/docs/mailgun/user-manual/webhooks/securing-webhooks,
// https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/spam-filter.
package mailin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxPostBytes caps the whole post before it is parsed. Mailgun documents
// a 25 MB message limit for inbound mail; base64 and the form's other
// fields add about a third, so 40 MB holds the largest message Mailgun
// will post. Task 0 records the real limit.
const MaxPostBytes = 40 << 20

// MaxBodyRunes is how much of a body is kept on the message row and in its
// document; the rest is cut with a note.
const MaxBodyRunes = 20000

// FreshFor is how old a signed timestamp may be: the one constant for it
// (security review M2). Mailgun retries a failed post for 8 hours; this
// build assumes a retry reuses its first signature, so 9 hours is the
// shortest window that still takes every retry. Task 0 records whether a
// retry is signed afresh; if it is, this becomes 15 minutes. The token
// record (store.SeenWebhookToken, kept two days) refuses a second use.
const FreshFor = 9 * time.Hour

// VerifySignature checks Mailgun's hex HMAC-SHA256 of timestamp then token,
// keyed with the account's HTTP webhook signing key, in constant time.
// https://documentation.mailgun.com/docs/mailgun/user-manual/webhooks/securing-webhooks
func VerifySignature(key, timestamp, token, signature string) bool {
	if key == "" || timestamp == "" || token == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(timestamp + token))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(strings.ToLower(signature)))
}

// Fresh reports whether a signed timestamp (Unix seconds) is no older than
// FreshFor and not more than five minutes ahead of now.
func Fresh(timestamp string, now time.Time) bool {
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	at := time.Unix(sec, 0)
	return !at.Before(now.Add(-FreshFor)) && !at.After(now.Add(5*time.Minute))
}

// Message is one parsed inbound post, before any decision. Every text field
// is one line except BodyPlain.
type Message struct {
	Recipient, EnvelopeSender string
	FromAddress, FromName     string
	To, Cc, Subject           string
	Date                      time.Time // zero when the header is missing or unreadable
	DateText                  string    // the Date header as written
	MessageID                 string
	InReplyTo, References     string
	Headers                   [][2]string
	BodyPlain                 string
	// InlineIDs are the attachment field names content-id-map names: parts
	// shown inside the body, such as a signature logo.
	InlineIDs map[string]bool
}

// Header is the first value of a header, by name in any case.
func (m Message) Header(name string) string {
	for _, h := range m.Headers {
		if strings.EqualFold(h[0], name) {
			return h[1]
		}
	}
	return ""
}

// HeaderValues is every value of a header.
func (m Message) HeaderValues(name string) []string {
	var out []string
	for _, h := range m.Headers {
		if strings.EqualFold(h[0], name) {
			out = append(out, h[1])
		}
	}
	return out
}

// ErrPost is a post that is not a message we can read.
var ErrPost = errors.New("unreadable message")

// Field reads one form field; the webhook passes r.PostFormValue.
type Field func(name string) string

// Read turns Mailgun's parsed post into a Message. It never fails on an
// odd header; it fails only when there is no usable From address. A
// message without a Message-Id is recorded once by its signed token, which
// the caller uses as its id.
func Read(field Field) (Message, error) {
	m := Message{
		Recipient:      field("recipient"),
		EnvelopeSender: strings.ToLower(strings.TrimSpace(field("sender"))),
		Headers:        HeaderList(field("message-headers")),
		InlineIDs:      inlineIDs(field("content-id-map")),
	}
	addr, name, err := SenderAddress(field("from"))
	if err != nil {
		return Message{}, err
	}
	m.FromAddress, m.FromName = addr, name
	m.Subject = OneLine(field("subject"), 998)
	m.To = OneLine(firstNonEmpty(field("To"), m.Header("To")), 4000)
	m.Cc = OneLine(firstNonEmpty(field("Cc"), m.Header("Cc")), 4000)
	m.MessageID = OneLine(firstNonEmpty(field("Message-Id"), m.Header("Message-Id")), 998)
	m.InReplyTo, m.References = ThreadHeaders(m)
	m.DateText = OneLine(firstNonEmpty(field("Date"), m.Header("Date")), 200)
	if d, err := mail.ParseDate(m.DateText); err == nil {
		m.Date = d
	}
	m.BodyPlain = cutBody(PlainBody(field("body-plain"), field("body-html")))
	return m, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// HeaderList reads message-headers: a JSON array of [name, value] pairs.
// Anything else gives no headers, which every check treats as absent.
func HeaderList(raw string) [][2]string {
	var pairs [][]string
	if json.Unmarshal([]byte(raw), &pairs) != nil {
		return nil
	}
	out := make([][2]string, 0, len(pairs))
	for _, p := range pairs {
		if len(p) != 2 || len(out) >= 500 {
			continue
		}
		out = append(out, [2]string{strings.TrimSpace(p[0]), strings.TrimSpace(p[1])})
	}
	return out
}

// ThreadHeaders reads In-Reply-To and References from message-headers
// (correspondence plan, A5: Task 0 confirms Mailgun carries them). Each is
// kept as message ids only, one line.
func ThreadHeaders(m Message) (inReplyTo, references string) {
	return strings.Join(MessageIDs(m.Header("In-Reply-To")), " "), strings.Join(MessageIDs(strings.Join(m.HeaderValues("References"), " ")), " ")
}

var messageIDPattern = regexp.MustCompile(`<[^<>\s]{1,500}>`)

// MessageIDs are the <...> ids in a header, at most 50.
func MessageIDs(v string) []string {
	ids := messageIDPattern.FindAllString(v, 50)
	return ids
}

// inlineIDs reads content-id-map, {"<cid>": "attachment-1"}, into the set
// of attachment fields shown inside the body.
func inlineIDs(raw string) map[string]bool {
	out := map[string]bool{}
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) != nil {
		return out
	}
	for _, field := range m {
		out[field] = true
	}
	return out
}

// SenderAddress reads one From address: lower case, and the display name.
// Two addresses, none, or a malformed header is an error.
func SenderAddress(from string) (addr, name string, err error) {
	from = strings.TrimSpace(from)
	if from == "" || strings.ContainsAny(from, "\r\n") {
		return "", "", ErrPost
	}
	list, err := mail.ParseAddressList(from)
	if err != nil || len(list) != 1 {
		return "", "", ErrPost
	}
	a := strings.ToLower(list[0].Address)
	if len(a) > 320 || !strings.Contains(a, "@") {
		return "", "", ErrPost
	}
	return a, OneLine(list[0].Name, 200), nil
}

// Domain is an address's domain in lower case.
func Domain(addr string) string {
	_, d, ok := strings.Cut(strings.ToLower(strings.TrimSpace(addr)), "@")
	if !ok {
		return ""
	}
	return strings.Trim(d, "> ")
}

// RecipientLocalPart finds the project address among the recipients at
// domain. Mailgun writes the address it received for (also for Bcc, by the
// documentation); several are comma-separated. Task 0 confirms both.
func RecipientLocalPart(recipient, domain string) (string, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, r := range strings.Split(recipient, ",") {
		a, err := mail.ParseAddress(strings.TrimSpace(r))
		addr := strings.TrimSpace(r)
		if err == nil {
			addr = a.Address
		}
		local, d, ok := strings.Cut(strings.ToLower(addr), "@")
		if ok && d == domain && local != "" {
			return local, true
		}
	}
	return "", false
}

// PlainBody is body-plain, or when Mailgun gave none (an HTML-only message;
// Task 0 checks whether body-plain is filled for one) the HTML's text. HTML
// is never kept or shown: tags are dropped, links stay as their words, no
// image is fetched.
func PlainBody(plain, htmlBody string) string {
	if strings.TrimSpace(plain) != "" {
		return normaliseNewlines(plain)
	}
	if strings.TrimSpace(htmlBody) == "" {
		return ""
	}
	return textFromHTML(htmlBody)
}

var (
	htmlDrop  = regexp.MustCompile(`(?is)<(script|style|head)\b.*?</(script|style|head)\s*>`)
	htmlBreak = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/tr|/li|/h[1-6])\b[^>]*>`)
	htmlTag   = regexp.MustCompile(`(?s)<[^>]*>`)
	blankRuns = regexp.MustCompile(`\n{3,}`)
)

func textFromHTML(s string) string {
	s = htmlDrop.ReplaceAllString(s, "")
	s = htmlBreak.ReplaceAllString(s, "\n")
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(normaliseNewlines(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(blankRuns.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

func normaliseNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func cutBody(s string) string {
	s = strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "")
	if utf8.RuneCountInString(s) <= MaxBodyRunes {
		return s
	}
	r := []rune(s)
	return string(r[:MaxBodyRunes-40]) + "\n\n[The rest of this message was cut.]"
}

// OneLine collapses line breaks and control characters to spaces and cuts
// to max bytes, so a value can never add a header line or a heading.
func OneLine(s string, max int) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToValidUTF8(s, "") {
		if r < 0x20 || r == 0x7f || r == ' ' || r == ' ' {
			r = ' '
		}
		if r == ' ' {
			if space {
				continue
			}
			space = true
		} else {
			space = false
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	for len(out) > max {
		_, size := utf8.DecodeLastRuneInString(out)
		out = out[:len(out)-size]
	}
	return out
}

// Dropped reports mail that is never recorded: Mailgun's spam flag, an
// automatic reply, a bounce, bulk or mailing-list mail. Dropping these is
// what stops a mail loop.
func Dropped(m Message) bool {
	// Any spam flag drops it: a sender can write a "No" of its own, but
	// can never undo Mailgun's "Yes" (security review L1).
	for _, v := range m.HeaderValues("X-Mailgun-Sflag") {
		if strings.EqualFold(strings.TrimSpace(v), "Yes") {
			return true
		}
	}
	if a := strings.ToLower(strings.TrimSpace(m.Header("Auto-Submitted"))); a != "" && a != "no" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(m.Header("Precedence"))) {
	case "bulk", "junk", "list", "auto_reply":
		return true
	}
	for _, h := range []string{"X-Autoreply", "X-Autorespond", "X-Auto-Response-Suppress-Original", "List-Id", "List-Unsubscribe"} {
		if m.Header(h) != "" {
			return true
		}
	}
	// A bounce has an empty envelope sender, or comes from the mail system.
	if m.EnvelopeSender == "" || m.EnvelopeSender == "<>" {
		return true
	}
	local, _, _ := strings.Cut(m.FromAddress, "@")
	if local == "mailer-daemon" || local == "postmaster" {
		return true
	}
	return strings.Contains(strings.ToLower(m.Header("Content-Type")), "multipart/report")
}

// Part kinds that are recorded without their bytes.
const (
	SkipInline   = "inline_image"
	SkipCalendar = "calendar"
	SkipWinmail  = "winmail"
)

// inlineImageMax: an inline image under this is a signature logo.
const inlineImageMax = 100 << 10

// SkipPart says why an attachment is not kept, or "" to keep it. An inline
// image is one the body shows: marked inline in the raw message, or named in
// Mailgun's content-id-map. A file of an unknown type is kept and stored not
// filed, as a dropped one is.
func SkipPart(m Message, a Attachment, name string) string {
	ctype := strings.ToLower(a.ContentType)
	ext := strings.ToLower(filepath.Ext(name))
	inline := (a.Field != "" && m.InlineIDs[a.Field]) || (a.ContentID != "" && m.InlineIDs[a.ContentID])
	switch {
	case strings.EqualFold(name, "winmail.dat") || strings.Contains(ctype, "ms-tnef"):
		return SkipWinmail
	case ext == ".ics" || strings.HasPrefix(ctype, "text/calendar"):
		return SkipCalendar
	case strings.HasPrefix(ctype, "image/") && a.Size < inlineImageMax && inline:
		return SkipInline
	}
	return ""
}

// AttachmentName is a safe base name for an attachment: never a path.
func AttachmentName(raw string, n int) string {
	raw = strings.ReplaceAll(raw, "\\", "/")
	name := cleanName(filepath.Base("/" + raw))
	if name == "" || name == "." || name == "/" {
		name = fmt.Sprintf("attachment-%d", n)
	}
	return name
}

var unsafeName = regexp.MustCompile(`[\x00-\x1f\x7f/\\:*?"<>|]+`)

func cleanName(s string) string {
	s = strings.Join(strings.Fields(unsafeName.ReplaceAllString(strings.ToValidUTF8(s, ""), " ")), " ")
	s = strings.Trim(s, ". ")
	for len(s) > 255 {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return s
}

// BodyDocument is the message as a Markdown document in the layout intake
// already reads as an email (internal/intake/markdown_identity.go): From,
// To, Subject and Date lines, a blank line, then the plain body. The date
// is the header's as written, without converting its zone; with none, the
// time received in Sydney.
func BodyDocument(m Message, received time.Time) (filename string, content []byte) {
	date := m.Date
	if date.IsZero() {
		date = received.In(sydney())
	}
	from := m.FromAddress
	if m.FromName != "" {
		from = m.FromName + " <" + m.FromAddress + ">"
	}
	to := m.To
	if to == "" {
		to = "(not shown)"
	}
	var b strings.Builder
	b.WriteString("From: " + OneLine(from, 400) + "\n")
	b.WriteString("To: " + OneLine(to, 4000) + "\n")
	if m.Cc != "" {
		b.WriteString("Cc: " + OneLine(m.Cc, 4000) + "\n")
	}
	b.WriteString("Subject: " + OneLine(m.Subject, 998) + "\n")
	b.WriteString("Date: " + date.Format("2 January 2006 3:04 PM") + "\n\n")
	b.WriteString(m.BodyPlain)
	b.WriteString("\n")
	return BodyFilename(m.Subject, date), []byte(b.String())
}

// BodyFilename is "<subject> (email 2026-11-14).md", cleaned and at most
// 255 bytes, or "Email 2026-11-14.md" with no subject.
func BodyFilename(subject string, date time.Time) string {
	day := date.Format("2006-01-02")
	tail := " (email " + day + ").md"
	s := cleanName(OneLine(subject, 998))
	if s == "" {
		return "Email " + day + ".md"
	}
	for len(s)+len(tail) > 255 {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return strings.TrimSpace(s) + tail
}

func sydney() *time.Location {
	if loc, err := time.LoadLocation("Australia/Sydney"); err == nil {
		return loc
	}
	return time.FixedZone("AEST", 10*3600)
}

var (
	forwardSubject = regexp.MustCompile(`(?i)^\s*(fwd?|fw)\s*:`)
	forwardFrom    = regexp.MustCompile(`(?im)^[>\s*]*from:\s*(.+?)\s*$`)
)

// Forwarded reports a forward ("Fwd:" or "FW:" subject, correspondence
// decision 7) and the original sender Go finds in its first forwarded
// header block, if any. ok is true for a forward even when no original
// sender is found.
func Forwarded(subject, body string) (addr, name string, ok bool) {
	if !forwardSubject.MatchString(subject) {
		return "", "", false
	}
	if m := forwardFrom.FindStringSubmatch(body); m != nil {
		if a, n, err := SenderAddress(strings.Trim(m[1], "*_ ")); err == nil {
			return a, n, true
		}
	}
	return "", "", true
}
