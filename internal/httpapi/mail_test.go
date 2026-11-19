package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"net/mail"
	"sitewise/internal/correspondence"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/mailin/mailintest"
	"sitewise/internal/store"
)

// Test posts are hand-written in Mailgun's documented forward() shape
// (docs/plans/2026-11-14-email-intake.md, Task 6), with dummy content only.
// The signing key is a test key; nothing here reaches Mailgun.
const (
	mailTestKey    = "test-webhook-signing-key"
	mailTestDomain = "in.example.test"
)

type mailFixture struct {
	a        *app
	m        *member
	conn     *pgx.Conn
	blobs    *files.Store
	project  string
	address  string
	email    string
	user     string
	seq      int
	signWith string
}

// testKeys is the stub DNS of DKIM keys every mail test signs with: no
// test reaches the network.
var testKeys = mailintest.NewKeys()

func withMail(o *httpapi.Options) {
	o.Mail = httpapi.NewMailIn(mailTestDomain, mailTestKey).WithDKIMLookup(testKeys.Lookup)
}

func newMailFixture(t *testing.T, adjust ...func(*httpapi.Options)) *mailFixture {
	t.Helper()
	ctx := context.Background()
	blobs, err := files.Open(t.TempDir(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	opts := append([]func(*httpapi.Options){withMail, func(o *httpapi.Options) { o.Blobs = blobs; o.MaxUploadBytes = 100 << 20 }}, adjust...)
	f := &mailFixture{a: newApp(t, opts...), blobs: blobs, signWith: mailTestKey}
	f.m = f.a.member(t, newUUID(t))
	f.project = f.m.createProject(t, "Mail project")
	if f.conn, err = pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.conn.Close(context.Background()) })
	if err := f.conn.QueryRow(ctx, `SELECT id::text, email FROM users WHERE org_id=$1::uuid LIMIT 1`, f.m.orgID).Scan(&f.user, &f.email); err != nil {
		t.Fatal(err)
	}
	var list struct {
		Address string `json:"address"`
	}
	f.m.getJSON(t, "/api/projects/"+f.project+"/mail", &list)
	f.address = list.Address
	return f
}

type mailFile struct {
	field, name, ctype string
	body               []byte
}

func pdfFile(n int, name string) mailFile {
	return mailFile{field: "attachment-" + strconv.Itoa(n), name: name, ctype: "application/pdf", body: []byte("%PDF-1.4 dummy " + name)}
}

// mailPost is one message as Mailgun's forward() posts it.
type mailPost struct {
	to, from, envelope, subject, body, messageID string
	headers                                      [][2]string
	files                                        []mailFile
	inline                                       map[string]string
	timestamp                                    time.Time
	token                                        string
	// The raw message is DKIM-signed by the From address's domain unless
	// unsigned, or by signAs. parsedOnly posts Mailgun's parsed fields with
	// no raw message (which is never authenticated).
	unsigned, parsedOnly bool
	signAs               string
}

// memberPost is a message from the fixture's member, DKIM-signed by the
// member's own domain.
func (f *mailFixture) memberPost(subject string, files ...mailFile) mailPost {
	f.seq++
	return mailPost{to: f.address, from: "Pat Member <" + f.email + ">", envelope: f.email, subject: subject,
		body: "Please file the attached.", messageID: fmt.Sprintf("<m%d-%s@example.test>", f.seq, newUUIDString()), files: files}
}

// rawOf is the post's raw message, signed as the post says.
func rawOf(p mailPost) []byte {
	m := mailintest.Message{From: p.from, To: p.to, Subject: p.subject, MessageID: p.messageID, Body: p.body, Headers: p.headers}
	inline := map[string]string{}
	for cid, field := range p.inline {
		inline[field] = strings.Trim(cid, "<>")
	}
	for _, f := range p.files {
		m.Files = append(m.Files, mailintest.File{Name: f.name, Type: f.ctype, Body: f.body, Inline: inline[f.field] != "", ContentID: inline[f.field]})
	}
	raw := m.Build()
	if p.unsigned {
		return raw
	}
	domain := p.signAs
	if domain == "" {
		if a, err := mail.ParseAddress(p.from); err == nil {
			_, domain, _ = strings.Cut(a.Address, "@")
		}
	}
	return testKeys.Sign(raw, domain)
}

var uuidSeq int

func newUUIDString() string {
	uuidSeq++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), uuidSeq)
}

func (f *mailFixture) fields(p mailPost) map[string]string {
	if p.timestamp.IsZero() {
		p.timestamp = time.Now()
	}
	if p.token == "" {
		p.token = "tok-" + newUUIDString()
	}
	ts := strconv.FormatInt(p.timestamp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(f.signWith))
	mac.Write([]byte(ts + p.token))
	headers := append([][2]string{{"Message-Id", p.messageID}, {"To", p.to}, {"Date", "Fri, 14 Nov 2026 09:15:00 +1100"}}, p.headers...)
	raw, _ := json.Marshal(headers)
	cid, _ := json.Marshal(p.inline)
	out := map[string]string{
		"recipient": p.to, "sender": p.envelope, "from": p.from, "subject": p.subject,
		"timestamp": ts, "token": p.token, "signature": hex.EncodeToString(mac.Sum(nil)),
	}
	if p.parsedOnly {
		out["body-plain"], out["message-headers"], out["content-id-map"], out["attachment-count"] = p.body, string(raw), string(cid), strconv.Itoa(len(p.files))
		return out
	}
	// Mailgun's raw-MIME route (forward URL ending in "mime").
	out["body-mime"] = string(rawOf(p))
	return out
}

// send posts as Mailgun would: the raw message in body-mime, urlencoded;
// a parsed-only post carries its files as multipart.
func (f *mailFixture) send(t *testing.T, p mailPost) int {
	t.Helper()
	if !p.parsedOnly {
		return f.sendFields(t, f.fields(p), nil)
	}
	return f.sendFields(t, f.fields(p), p.files)
}

func (f *mailFixture) sendFields(t *testing.T, fields map[string]string, attach []mailFile) int {
	t.Helper()
	ctype, body := encodePost(fields, attach)
	return f.raw(t, ctype, body)
}

// encodePost writes a post as Mailgun does: urlencoded with no files, else
// multipart.
func encodePost(fields map[string]string, attach []mailFile) (string, *bytes.Buffer) {
	var body bytes.Buffer
	ctype := "application/x-www-form-urlencoded"
	if len(attach) == 0 {
		v := url.Values{}
		for k, val := range fields {
			v.Set(k, val)
		}
		body.WriteString(v.Encode())
	} else {
		w := multipart.NewWriter(&body)
		for k, val := range fields {
			_ = w.WriteField(k, val)
		}
		for _, a := range attach {
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, a.field, a.name))
			h.Set("Content-Type", a.ctype)
			part, _ := w.CreatePart(h)
			_, _ = part.Write(a.body)
		}
		_ = w.Close()
		ctype = w.FormDataContentType()
	}
	return ctype, &body
}

func (f *mailFixture) raw(t *testing.T, ctype string, body *bytes.Buffer) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, f.a.url+"/api/mail/inbound", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", ctype)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

type mailListBody struct {
	Address  string              `json:"address"`
	Held     int                 `json:"held"`
	Messages []store.MailMessage `json:"messages"`
}

func (f *mailFixture) list(t *testing.T) mailListBody {
	t.Helper()
	var out mailListBody
	f.m.getJSON(t, "/api/projects/"+f.project+"/mail", &out)
	return out
}

func (f *mailFixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *mailFixture) documents(t *testing.T) int {
	return f.count(t, `SELECT count(*) FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid`, f.m.orgID, f.project)
}

func (f *mailFixture) messages(t *testing.T) int {
	return f.count(t, `SELECT count(*) FROM mail_messages WHERE org_id=$1::uuid`, f.m.orgID)
}

func TestMailInboundFilesAMembersMessage(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	if !strings.HasSuffix(f.address, "@"+mailTestDomain) || strings.Contains(strings.ToLower(f.address), "mail project") {
		t.Fatalf("address %q", f.address)
	}
	usage := func() int {
		u, err := f.a.store.Usage(ctx, f.m.orgID, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return u.DocumentsThisMonth
	}
	before := usage()
	p := f.memberPost("Basement drawings", pdfFile(1, "A-101.pdf"), pdfFile(2, "A-102.pdf"))
	p.token = "tok-first-" + newUUIDString()
	if got := f.send(t, p); got != http.StatusOK {
		t.Fatalf("post = %d", got)
	}
	if n := f.documents(t); n != 3 {
		var st, why string
		_ = f.conn.QueryRow(ctx, `SELECT status, held_reason FROM mail_messages WHERE org_id=$1::uuid`, f.m.orgID).Scan(&st, &why)
		t.Fatalf("documents = %d, want the body and two PDFs (message %s %s)", n, st, why)
	}
	if got := usage() - before; got != 3 {
		t.Fatalf("month count rose by %d", got)
	}
	l := f.list(t)
	if len(l.Messages) != 1 || l.Messages[0].Status != "filed" || len(l.Messages[0].Parts) != 3 {
		t.Fatalf("list %+v", l)
	}
	var reading string
	if err := f.conn.QueryRow(ctx, `SELECT d.profile_read FROM documents d JOIN mail_parts p ON p.org_id=d.org_id AND p.document_id=d.id WHERE p.n=0 AND d.org_id=$1::uuid`, f.m.orgID).Scan(&reading); err != nil {
		t.Fatal(err)
	}
	if reading != "skip" {
		t.Fatalf("body document profile reading %q", reading)
	}
	if n := f.count(t, `SELECT count(*) FROM events WHERE org_id=$1::uuid AND kind='mail'`, f.m.orgID); n == 0 {
		t.Fatal("no mail event")
	}
	// Mailgun's retry of the same post, and the same message under a new token.
	if got := f.send(t, p); got != http.StatusOK {
		t.Fatalf("retry = %d", got)
	}
	p.token = "tok-second-" + newUUIDString()
	if got := f.send(t, p); got != http.StatusOK {
		t.Fatalf("new token = %d", got)
	}
	if f.documents(t) != 3 || f.messages(t) != 1 {
		t.Fatalf("repeats filed again: %d documents, %d messages", f.documents(t), f.messages(t))
	}
	// No attachments: urlencoded, one document.
	if got := f.send(t, f.memberPost("A note")); got != http.StatusOK || f.documents(t) != 4 {
		t.Fatalf("no attachments = %d, %d documents", got, f.documents(t))
	}
}

func TestMailInboundHoldsWhatItCannotTrust(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	other := f.a.member(t, newUUID(t))
	var otherEmail string
	if err := f.conn.QueryRow(ctx, `SELECT email FROM users WHERE org_id=$1::uuid`, other.orgID).Scan(&otherEmail); err != nil {
		t.Fatal(err)
	}
	stranger := f.memberPost("From a stranger", pdfFile(1, "x.pdf"))
	stranger.from, stranger.envelope = "Someone <someone@outside.test>", "someone@outside.test"
	otherOrg := f.memberPost("From another org")
	otherOrg.from, otherOrg.envelope = otherEmail, otherEmail
	unverified := f.memberPost("No DKIM signature")
	unverified.unsigned = true
	misaligned := f.memberPost("Signed by another domain")
	misaligned.signAs = "attacker.test"
	for name, c := range map[string]struct {
		post   mailPost
		reason string
	}{
		"not a member":      {stranger, store.MailUnknownSender},
		"another org":       {otherOrg, store.MailUnknownSender},
		"headers absent":    {unverified, store.MailUnverifiedSender},
		"envelope mismatch": {misaligned, store.MailUnverifiedSender},
	} {
		docs := f.documents(t)
		if got := f.send(t, c.post); got != http.StatusOK {
			t.Fatalf("%s: post = %d", name, got)
		}
		if f.documents(t) != docs {
			t.Fatalf("%s: filed", name)
		}
		var reason string
		if err := f.conn.QueryRow(ctx, `SELECT held_reason FROM mail_messages WHERE org_id=$1::uuid AND message_id=$2`, f.m.orgID, c.post.messageID).Scan(&reason); err != nil || reason != c.reason {
			t.Fatalf("%s: reason %q %v", name, reason, err)
		}
	}
	// A member whose email is not confirmed is not a member here.
	if _, err := f.conn.Exec(ctx, `UPDATE users SET email_verified=false WHERE org_id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	p := f.memberPost("Unconfirmed")
	f.send(t, p)
	var reason string
	_ = f.conn.QueryRow(ctx, `SELECT held_reason FROM mail_messages WHERE message_id=$1`, p.messageID).Scan(&reason)
	if reason != store.MailUnknownSender {
		t.Fatalf("unconfirmed: %q", reason)
	}
	if _, err := f.conn.Exec(ctx, `UPDATE users SET email_verified=true WHERE org_id=$1::uuid`, f.m.orgID); err != nil {
		t.Fatal(err)
	}
	// An archived project holds everything.
	f.m.call(t, http.MethodPut, "/api/projects/"+f.project+"/archived", []byte(`{"archived":true}`), http.StatusNoContent, nil)
	p = f.memberPost("While archived", pdfFile(1, "a.pdf"))
	f.send(t, p)
	_ = f.conn.QueryRow(ctx, `SELECT held_reason FROM mail_messages WHERE message_id=$1`, p.messageID).Scan(&reason)
	if reason != store.MailArchived {
		t.Fatalf("archived: %q", reason)
	}
	if f.documents(t) != 0 {
		t.Fatalf("held mail filed %d documents", f.documents(t))
	}
}

func TestMailInboundRefusesAndDrops(t *testing.T) {
	f := newMailFixture(t)
	refused := func(name string, got int) {
		t.Helper()
		if got != http.StatusNotAcceptable {
			t.Fatalf("%s = %d, want 406", name, got)
		}
	}
	p := f.memberPost("Bad", pdfFile(1, "a.pdf"))
	fields := f.fields(p)
	fields["signature"] = strings.Repeat("0", 64)
	refused("bad signature", f.sendFields(t, fields, p.files))
	f.signWith = "another key"
	refused("other key", f.send(t, p))
	f.signWith = mailTestKey
	old := f.memberPost("Old")
	old.timestamp = time.Now().Add(-25 * time.Hour)
	refused("stale", f.send(t, old))
	future := f.memberPost("Future")
	future.timestamp = time.Now().Add(time.Hour)
	refused("future", f.send(t, future))
	unsigned := f.fields(f.memberPost("Unsigned"))
	delete(unsigned, "signature")
	refused("unsigned", f.sendFields(t, unsigned, nil))
	if f.messages(t) != 0 {
		t.Fatal("a refused post was stored")
	}
	// Dropped: 200 so Mailgun does not retry, nothing stored.
	unknown := f.memberPost("Unknown address")
	unknown.to = "p-00000000000000000000000000000000@" + mailTestDomain
	elsewhere := f.memberPost("Other domain")
	elsewhere.to = strings.Replace(f.address, mailTestDomain, "other.test", 1)
	spam := f.memberPost("Spam")
	spam.headers = append(spam.headers, [2]string{"X-Mailgun-Sflag", "Yes"})
	auto := f.memberPost("Out of office")
	auto.headers = append(auto.headers, [2]string{"Auto-Submitted", "auto-replied"})
	list := f.memberPost("Newsletter")
	list.headers = append(list.headers, [2]string{"List-Id", "<news.example.test>"})
	bounce := f.memberPost("Undeliverable")
	bounce.envelope = ""
	sflag := f.memberPost("Spam, the sender's No first")
	sflag.headers = append(sflag.headers, [2]string{"X-Mailgun-Sflag", "No"}, [2]string{"X-Mailgun-Sflag", "Yes"})
	for name, d := range map[string]mailPost{"unknown": unknown, "elsewhere": elsewhere, "spam": spam, "auto-reply": auto, "list": list, "bounce": bounce, "spam flag second": sflag} {
		if got := f.send(t, d); got != http.StatusOK {
			t.Fatalf("%s = %d", name, got)
		}
	}
	if f.messages(t) != 0 {
		t.Fatalf("dropped mail stored %d messages", f.messages(t))
	}
}

// Mailgun signs only the timestamp and the token, so a captured post could
// be sent again with another body inside the freshness window. Every signed
// token is recorded, whatever became of its post.
func TestMailInboundRefusesAReplayedToken(t *testing.T) {
	f := newMailFixture(t)
	dropped := f.memberPost("Unknown address")
	dropped.to = "p-11111111111111111111111111111111@" + mailTestDomain
	dropped.token = "tok-captured-" + newUUIDString()
	if got := f.send(t, dropped); got != http.StatusOK {
		t.Fatal(got)
	}
	forged := f.memberPost("Forged with a captured signature", pdfFile(1, "evil.pdf"))
	forged.token, forged.timestamp = dropped.token, time.Now()
	f.send(t, forged)
	if f.messages(t) != 0 || f.documents(t) != 0 {
		t.Fatal("a replayed token stored a message")
	}
}

func TestMailInboundParts(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	logo := mailFile{field: "attachment-1", name: "logo.png", ctype: "image/png", body: bytes.Repeat([]byte{1}, 2048)}
	exe := mailFile{field: "attachment-2", name: "setup.exe", ctype: "application/octet-stream", body: []byte("MZ dummy")}
	ics := mailFile{field: "attachment-3", name: "meeting.ics", ctype: "text/calendar", body: []byte("BEGIN:VCALENDAR")}
	sneaky := mailFile{field: "attachment-4", name: "../../etc/report.pdf", ctype: "application/pdf", body: []byte("%PDF dummy sneaky")}
	p := f.memberPost("Parts", logo, exe, ics, sneaky)
	p.inline = map[string]string{"<logo@x>": "attachment-1"}
	if got := f.send(t, p); got != http.StatusOK {
		t.Fatal(got)
	}
	rows, err := f.conn.Query(ctx, `SELECT p.filename,p.skipped,COALESCE(d.status,''),COALESCE(d.reason,'') FROM mail_parts p LEFT JOIN documents d ON d.org_id=p.org_id AND d.id=p.document_id WHERE p.org_id=$1::uuid ORDER BY p.n`, f.m.orgID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string][3]string{}
	for rows.Next() {
		var name, skipped, status, reason string
		if err := rows.Scan(&name, &skipped, &status, &reason); err != nil {
			t.Fatal(err)
		}
		got[name] = [3]string{skipped, status, reason}
	}
	if g := got["logo.png"]; g[0] != "inline_image" || g[1] != "" {
		t.Fatalf("logo %v", g)
	}
	if g := got["meeting.ics"]; g[0] != "calendar" {
		t.Fatalf("calendar %v", g)
	}
	if g := got["setup.exe"]; g[0] != "" || g[1] != store.StatusNotFiled {
		t.Fatalf("exe %v", g)
	}
	if _, ok := got["report.pdf"]; !ok {
		t.Fatalf("a path in the name was kept: %v", got)
	}
}

func TestMailInboundCapsTheBody(t *testing.T) {
	f := newMailFixture(t)
	p := f.memberPost("Huge", mailFile{field: "attachment-1", name: "huge.pdf", ctype: "application/pdf", body: bytes.Repeat([]byte("x"), 41<<20)})
	if got := f.send(t, p); got != http.StatusNotAcceptable {
		t.Fatalf("41 MB post = %d", got)
	}
	if f.messages(t) != 0 {
		t.Fatal("stored")
	}
}

// Refusals from one address stop being read after 30 in 15 minutes; the
// answer stays the same.
func TestMailInboundLimitsRefusals(t *testing.T) {
	f := newMailFixture(t)
	bad := f.fields(f.memberPost("Bad"))
	bad["signature"] = strings.Repeat("0", 64)
	for i := 0; i < 31; i++ {
		if got := f.sendFields(t, bad, nil); got != http.StatusNotAcceptable {
			t.Fatalf("refusal %d = %d", i, got)
		}
	}
	if got := f.send(t, f.memberPost("Now genuine")); got != http.StatusNotAcceptable {
		t.Fatalf("after 30 refusals a post was read: %d", got)
	}
}

func TestMailOffAnswers404(t *testing.T) {
	a := newApp(t)
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "No mail")
	var out mailListBody
	m.getJSON(t, "/api/projects/"+project+"/mail", &out)
	if out.Address != "" {
		t.Fatalf("address shown with intake off: %q", out.Address)
	}
	req, _ := http.NewRequest(http.MethodPost, a.url+"/api/mail/inbound", strings.NewReader("token=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("webhook off = %d", resp.StatusCode)
	}
	if got := m.status(t, http.MethodPost, "/api/projects/"+project+"/mail/address", nil); got != http.StatusNotFound {
		t.Fatalf("rotate off = %d", got)
	}
}

func (f *mailFixture) heldID(t *testing.T, messageID string) string {
	t.Helper()
	var id string
	if err := f.conn.QueryRow(context.Background(), `SELECT id::text FROM mail_messages WHERE message_id=$1`, messageID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestHeldMailAcceptDiscardRotate(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	foreign := f.a.member(t, newUUID(t))
	sibling := f.m.createProject(t, "Sibling")
	stranger := f.memberPost("From a consultant", pdfFile(1, "S-201.pdf"), pdfFile(2, "S-202.pdf"))
	stranger.from, stranger.envelope = "consultant@outside.test", "consultant@outside.test"
	f.send(t, stranger)
	id := f.heldID(t, stranger.messageID)
	base := "/api/projects/" + f.project + "/mail/" + id
	// Another org, another project of the same org, and a malformed id: 404.
	for _, path := range []string{"", "/accept", "/discard"} {
		method := http.MethodPost
		if path == "" {
			method = http.MethodGet
		}
		if got := foreign.status(t, method, base+path, nil); got != http.StatusNotFound {
			t.Fatalf("foreign %s %s = %d", method, path, got)
		}
		if got := f.m.status(t, method, "/api/projects/"+sibling+"/mail/"+id+path, nil); got != http.StatusNotFound {
			t.Fatalf("sibling %s %s = %d", method, path, got)
		}
		if got := f.m.status(t, method, "/api/projects/"+f.project+"/mail/not-an-id"+path, nil); got != http.StatusNotFound {
			t.Fatalf("malformed %s %s = %d", method, path, got)
		}
	}
	if got := foreign.status(t, http.MethodGet, "/api/projects/"+f.project+"/mail", nil); got != http.StatusNotFound {
		t.Fatalf("foreign list = %d", got)
	}
	if got := foreign.status(t, http.MethodPost, "/api/projects/"+f.project+"/mail/address", nil); got != http.StatusNotFound {
		t.Fatalf("foreign rotate = %d", got)
	}
	var one store.MailMessage
	f.m.getJSON(t, base, &one)
	if one.Body != "Please file the attached." || one.HeldReason != store.MailUnknownSender {
		t.Fatalf("read %+v", one)
	}
	// Accept files it and empties the held list.
	f.m.call(t, http.MethodPost, base+"/accept", nil, http.StatusOK, nil)
	if f.documents(t) != 3 || f.list(t).Held != 0 {
		t.Fatalf("accept: %d documents, %d held", f.documents(t), f.list(t).Held)
	}
	// A filed message can be neither accepted again nor discarded.
	if got := f.m.status(t, http.MethodPost, base+"/accept", nil); got != http.StatusConflict {
		t.Fatalf("accept filed = %d", got)
	}
	if got := f.m.status(t, http.MethodPost, base+"/discard", nil); got != http.StatusConflict {
		t.Fatalf("discard filed = %d", got)
	}
	// Discard removes a held message and its unshared bytes, and keeps bytes
	// a filed document also uses.
	shared := f.memberPost("Again, and something new", pdfFile(1, "S-201.pdf"), mailFile{field: "attachment-2", name: "new.pdf", ctype: "application/pdf", body: []byte("%PDF only here")})
	shared.from, shared.envelope = "consultant@outside.test", "consultant@outside.test"
	shared.body = "Please file the attached."
	f.send(t, shared)
	held := f.heldID(t, shared.messageID)
	var sums [][]byte
	rows, err := f.conn.Query(ctx, `SELECT sha256 FROM mail_parts WHERE org_id=$1::uuid AND message_id=$2::uuid AND n>0 ORDER BY n`, f.m.orgID, held)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s []byte
		_ = rows.Scan(&s)
		sums = append(sums, s)
	}
	rows.Close()
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/mail/"+held+"/discard", nil, http.StatusNoContent, nil)
	if f.messages(t) != 1 {
		t.Fatalf("messages after discard %d", f.messages(t))
	}
	if rc, err := f.blobs.Open(sums[0]); err != nil {
		t.Fatalf("shared bytes removed: %v", err)
	} else {
		rc.Close()
	}
	// The unshared part may be kept a while by the blob store's minimum age;
	// nothing references it any more.
	if ref, err := f.a.store.BlobReferenced(ctx, sums[1]); err != nil || ref {
		t.Fatalf("unshared bytes still referenced: %v %v", ref, err)
	}
	// Rotate: the old address stores nothing, the new one files.
	var rotated struct{ Address, Warning string }
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/mail/address", nil, http.StatusOK, &rotated)
	if rotated.Address == f.address || !strings.Contains(rotated.Warning, "Replies") {
		t.Fatalf("rotate %+v", rotated)
	}
	before := f.messages(t)
	f.send(t, f.memberPost("To the old address"))
	if f.messages(t) != before {
		t.Fatal("the old address still stores mail")
	}
	f.address = rotated.Address
	f.send(t, f.memberPost("To the new address"))
	if f.messages(t) != before+1 {
		t.Fatal("the new address stored nothing")
	}
}

// With billing on, a member's message at the plan's monthly documents is
// held, and accepting it is refused with the plan's words.
func TestMailAtThePlanLimit(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t, func(o *httpapi.Options) { o.Billing = &httpapi.Billing{} })
	for {
		u, err := f.a.store.Usage(ctx, f.m.orgID, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if u.DocumentsThisMonth >= 50 {
			break
		}
		if err := f.a.store.CountDocument(ctx, f.m.orgID, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	p := f.memberPost("At the limit", pdfFile(1, "L-1.pdf"))
	f.send(t, p)
	id := f.heldID(t, p.messageID)
	var reason string
	_ = f.conn.QueryRow(ctx, `SELECT held_reason FROM mail_messages WHERE id=$1::uuid`, id).Scan(&reason)
	if reason != store.MailPlanLimit || f.documents(t) != 0 {
		t.Fatalf("reason %q, %d documents", reason, f.documents(t))
	}
	resp := f.m.do(t, http.MethodPost, "/api/projects/"+f.project+"/mail/"+id+"/accept", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusPaymentRequired || f.list(t).Held != 1 {
		t.Fatalf("accept over the limit = %d", resp.StatusCode)
	}
}

// Decision 7 and A1: a member's own message is Out, a forward is In from its
// original sender, and an authenticated firm on the project is filed.
func TestMailDirectionAndOutsideSenders(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	own := f.memberPost("Our letter to the engineer")
	f.send(t, own)
	fwd := f.memberPost("FW: Retaining wall")
	fwd.body = "For the record.\n\n---------- Forwarded message ---------\nFrom: Eng Smith <eng@consult.example>\nSubject: Retaining wall"
	f.send(t, fwd)
	read := func(messageID string) (direction, sender, status string) {
		t.Helper()
		if err := f.conn.QueryRow(ctx, `SELECT direction,sender,status FROM mail_messages WHERE message_id=$1`, messageID).Scan(&direction, &sender, &status); err != nil {
			t.Fatal(err)
		}
		return
	}
	if d, _, _ := read(own.messageID); d != "out" {
		t.Fatalf("own copy %s", d)
	}
	if d, s, _ := read(fwd.messageID); d != "in" || s != "eng@consult.example" {
		t.Fatalf("forward %s %s", d, s)
	}
	// A firm invited on a package of this project, sending from its own
	// address with SPF aligned, is filed; the same firm unauthenticated is not.
	var pkg struct {
		ID string `json:"id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/rows", []byte(`{"discipline_id":"consultant.structural"}`), http.StatusCreated, &pkg)
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	f.m.call(t, http.MethodPost, "/api/projects/"+f.project+"/procurement/packages/"+pkg.ID+"/invitations", []byte(`{"name":"Consult Engineers"}`), http.StatusCreated, &inv)
	if _, err := f.conn.Exec(ctx, `UPDATE firms SET email='eng@consult.example' WHERE org_id=$1::uuid AND id=$2::uuid`, f.m.orgID, inv.FirmID); err != nil {
		t.Fatal(err)
	}
	firm := f.memberPost("Wall detail")
	firm.from, firm.envelope = "Eng Smith <eng@consult.example>", "eng@consult.example"
	f.send(t, firm)
	if d, _, st := read(firm.messageID); d != "in" || st != "filed" {
		t.Fatalf("firm on the project: %s %s", d, st)
	}
	var linkedFirm, linkedPkg string
	_ = f.conn.QueryRow(ctx, `SELECT COALESCE(firm_id::text,''),COALESCE(package_id::text,'') FROM mail_messages WHERE message_id=$1`, firm.messageID).Scan(&linkedFirm, &linkedPkg)
	if linkedFirm != inv.FirmID || linkedPkg != pkg.ID {
		t.Fatalf("firm %q package %q", linkedFirm, linkedPkg)
	}
	spoofed := f.memberPost("Wall detail, unsigned")
	spoofed.from, spoofed.envelope, spoofed.unsigned = "eng@consult.example", "eng@consult.example", true
	f.send(t, spoofed)
	if _, _, st := read(spoofed.messageID); st != "held" {
		t.Fatalf("unauthenticated firm address filed: %s", st)
	}
	colleague := f.memberPost("From the firm's other address")
	colleague.from, colleague.envelope = "jo@consult.example", "jo@consult.example"
	f.send(t, colleague)
	if _, _, st := read(colleague.messageID); st != "held" {
		t.Fatalf("a domain match admitted: %s", st)
	}
}

// The notice pick lands on the message whose body was filed, in the filing
// commit; with no cut-off it is stored blank and no chip shows.
func TestMailNoticePickLandsOnItsMessage(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name   string
		amber  float64
		band   string
		notice string
	}{{"no cut-off", 0, "blank", ""}, {"past a cut-off", 0.4, "amber", "direction"}} {
		t.Run(c.name, func(t *testing.T) {
			f := newMailFixture(t, func(o *httpapi.Options) { o.NoticeThresholds = correspondence.NoticeThresholds{Amber: c.amber} })
			p := f.memberPost("Please proceed")
			f.send(t, p)
			other := f.memberPost("Another")
			f.send(t, other)
			var band, read string
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				_ = f.conn.QueryRow(ctx, `SELECT notice_band, notice_read FROM mail_messages WHERE message_id=$1`, p.messageID).Scan(&band, &read)
				if band != "" {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			// The fake Jev picks the first option by name at 0.5.
			if band != c.band || read != "direction" {
				t.Fatalf("band %q read %q", band, read)
			}
			l := f.list(t)
			for _, m := range l.Messages {
				if m.Subject == "Please proceed" && m.Notice != c.notice {
					t.Fatalf("chip %q", m.Notice)
				}
			}
		})
	}
}

// Security review C1: the probe that filed a forged From. A valid
// signature by evil.test beside a garbage one claiming the victim's domain,
// with Mailgun-looking headers the sender wrote, is held, never filed or Out.
func TestMailForgedFromIsHeld(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	_, memberDomain, _ := strings.Cut(f.email, "@")
	reason := func(messageID string) (string, string) {
		t.Helper()
		var why, dir string
		if err := f.conn.QueryRow(ctx, `SELECT held_reason, direction FROM mail_messages WHERE message_id=$1`, messageID).Scan(&why, &dir); err != nil {
			t.Fatal(err)
		}
		return why, dir
	}
	probe := f.memberPost("Pay this invoice")
	probe.signAs = "evil.test"
	probe.headers = [][2]string{{"X-Mailgun-Spf", "Pass"}, {"X-Mailgun-Dkim-Check-Result", "Pass"}, {"X-Mailgun-Sflag", "No"},
		{"DKIM-Signature", "v=1; a=rsa-sha256; d=" + memberDomain + "; s=t; h=from; bh=AAAA; b=AAAA"}}
	f.send(t, probe)
	if why, dir := reason(probe.messageID); why != store.MailUnverifiedSender || dir != "in" {
		t.Fatalf("forged From: %q %q", why, dir)
	}
	// No raw message: never authenticated, even with Mailgun's fields saying Pass.
	parsed := f.memberPost("Parsed only")
	parsed.parsedOnly = true
	parsed.headers = [][2]string{{"X-Mailgun-Spf", "Pass"}, {"X-Mailgun-Dkim-Check-Result", "Pass"}}
	f.send(t, parsed)
	if why, _ := reason(parsed.messageID); why != store.MailUnverifiedSender {
		t.Fatalf("no raw message: %q", why)
	}
	// Two From headers, or over 500 headers: held, never cut and trusted.
	two := f.memberPost("Two From headers")
	two.headers = [][2]string{{"From", "boss@" + memberDomain}}
	f.send(t, two)
	padded := f.memberPost("Padded")
	for i := 0; i < 501; i++ {
		padded.headers = append(padded.headers, [2]string{"X-Pad", "x"})
	}
	f.send(t, padded)
	for _, p := range []mailPost{two, padded} {
		var status string
		if err := f.conn.QueryRow(ctx, `SELECT status FROM mail_messages WHERE org_id=$1::uuid AND subject=$2`, f.m.orgID, p.subject).Scan(&status); err != nil || status != "held" {
			t.Fatalf("%s: %q %v", p.subject, status, err)
		}
	}
	if f.documents(t) != 0 {
		t.Fatalf("forged mail filed %d documents", f.documents(t))
	}
}

// C1: a forward is rewritten only for an authenticated member, and shows
// who forwarded it; an outsider's body cannot name a sender.
func TestMailForwardRewriteIsTheMembersOnly(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	body := "See below.\n\n---------- Forwarded message ---------\nFrom: Eng Smith <eng@consult.example>\nSubject: Wall"
	fwd := f.memberPost("FW: Wall")
	fwd.body = body
	f.send(t, fwd)
	var sender, name, dir string
	if err := f.conn.QueryRow(ctx, `SELECT sender, sender_name, direction FROM mail_messages WHERE message_id=$1`, fwd.messageID).Scan(&sender, &name, &dir); err != nil {
		t.Fatal(err)
	}
	if sender != "eng@consult.example" || dir != "in" || !strings.Contains(name, "forwarded by "+f.email) {
		t.Fatalf("member forward: %q %q %q", sender, name, dir)
	}
	unsigned := f.memberPost("FW: Wall again")
	unsigned.body, unsigned.unsigned = body, true
	f.send(t, unsigned)
	outsider := f.memberPost("FW: from outside")
	outsider.body = body
	outsider.from, outsider.envelope = "x@outside.test", "x@outside.test"
	f.send(t, outsider)
	for _, p := range []mailPost{unsigned, outsider} {
		if err := f.conn.QueryRow(ctx, `SELECT sender FROM mail_messages WHERE message_id=$1`, p.messageID).Scan(&sender); err != nil || sender == "eng@consult.example" {
			t.Fatalf("%s rewritten to %q %v", p.subject, sender, err)
		}
	}
}

// H2: nothing stored is left behind by a repeat or by held mail that is full.
func TestMailLeavesNoOrphanBlobs(t *testing.T) {
	f := newMailFixture(t)
	count := func() int {
		t.Helper()
		l, err := f.blobs.List()
		if err != nil {
			t.Fatal(err)
		}
		return len(l)
	}
	first := f.memberPost("Once", pdfFile(1, "A.pdf"))
	f.send(t, first)
	before := count()
	for i := 0; i < 5; i++ {
		again := f.memberPost("Once", mailFile{field: "attachment-1", name: "B.pdf", ctype: "application/pdf", body: []byte(fmt.Sprintf("%%PDF other bytes %d", i))})
		again.messageID = first.messageID
		f.send(t, again)
	}
	if got := count(); got != before {
		t.Fatalf("five repeats changed the blob count %d → %d", before, got)
	}
	// Fill held mail, then one more: held full keeps no bytes.
	if _, err := f.conn.Exec(context.Background(), `INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,status,held_reason,thread_id)
SELECT $1::uuid, g.id, $2::uuid, '<full-'||g.n||'@x.test>', 'full-'||g.id, 'x@outside.test', 'held', 'unknown_sender', g.id
FROM (SELECT gen_random_uuid() AS id, n FROM generate_series(1,50) n) g`, f.m.orgID, f.project); err != nil {
		t.Fatal(err)
	}
	before = count()
	full := f.memberPost("Full", mailFile{field: "attachment-1", name: "C.pdf", ctype: "application/pdf", body: []byte("%PDF held full")})
	full.from, full.envelope = "x@outside.test", "x@outside.test"
	f.send(t, full)
	if got := count(); got != before {
		t.Fatalf("held full stored bytes %d → %d", before, got)
	}
}

// H1: the signature is checked as soon as its fields arrive, before any
// file part is read: a bad post is refused while its sender is still
// sending.
func TestMailRefusesBeforeReadingFiles(t *testing.T) {
	f := newMailFixture(t)
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)
	go func() {
		for k, v := range map[string]string{"timestamp": strconv.FormatInt(time.Now().Unix(), 10), "token": "tok-" + newUUIDString(), "signature": strings.Repeat("0", 64)} {
			_ = w.WriteField(k, v)
		}
		part, _ := w.CreateFormFile("attachment-1", "big.pdf")
		_, _ = part.Write([]byte("%PDF start"))
		// The sender stalls here and never finishes.
	}()
	req, _ := http.NewRequest(http.MethodPost, f.a.url+"/api/mail/inbound", pr)
	req.Header.Set("Content-Type", w.FormDataContentType())
	done := make(chan int, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			done <- 0
			return
		}
		resp.Body.Close()
		done <- resp.StatusCode
	}()
	select {
	case code := <-done:
		if code != http.StatusNotAcceptable {
			t.Fatalf("refused with %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the webhook waited for the file part of a post with a bad signature")
	}
	pw.CloseWithError(io.ErrClosedPipe)
}
