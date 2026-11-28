package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

// Tests for the second security review of wave/mail.

// form posts fields urlencoded, as real Mailgun does for a message with no
// attachment (seen 20 November 2026).
func (f *mailFixture) form(t *testing.T, fields map[string]string) int {
	t.Helper()
	ctype, body := encodeForm(fields)
	return f.raw(t, ctype, body)
}

// retoken gives a post's fields a fresh signed token, the raw message
// unchanged: the same bytes posted again.
func (f *mailFixture) retoken(fields map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range fields {
		out[k] = v
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	token := "tok-" + newUUIDString()
	mac := hmac.New(sha256.New, []byte(f.signWith))
	mac.Write([]byte(ts + token))
	out["timestamp"], out["token"], out["signature"] = ts, token, hex.EncodeToString(mac.Sum(nil))
	return out
}

func (f *mailFixture) statusOf(t *testing.T, subject string) (status, reason string) {
	t.Helper()
	if err := f.conn.QueryRow(context.Background(), `SELECT status, held_reason FROM mail_messages WHERE org_id=$1::uuid AND subject=$2 ORDER BY received_at DESC LIMIT 1`, f.m.orgID, subject).Scan(&status, &reason); err != nil {
		t.Fatalf("%s: %v", subject, err)
	}
	return status, reason
}

// HIGH-1 (a): a header prepended above a genuinely signed member message
// is held, never filed under the member's name.
func TestMailPrependedHeaderIsHeld(t *testing.T) {
	f := newMailFixture(t)
	for _, h := range []string{"Subject: Pay this now", "In-Reply-To: <rfi9@in.example.test>", "Content-Type: text/html", "Message-ID: <other@example.test>"} {
		p := f.memberPost("Prepended " + h[:strings.Index(h, ":")])
		fields := f.fields(p)
		fields["body-mime"] = h + "\r\n" + fields["body-mime"]
		if got := f.sendFields(t, fields, nil); got != http.StatusOK {
			t.Fatalf("%s = %d", h, got)
		}
		if status, _ := f.statusOf(t, p.subject); status != "held" {
			t.Fatalf("prepended %q: %s", h, status)
		}
	}
	if f.documents(t) != 0 {
		t.Fatal("a prepended header filed documents")
	}
}

// HIGH-1 (c): a repeated Message-ID with other bytes is held as a
// duplicate, never dropped; the same bytes again are a plain repeat.
func TestMailRepeatedMessageIDWithOtherBytesIsHeld(t *testing.T) {
	f := newMailFixture(t)
	first := f.memberPost("Original")
	fields := f.fields(first)
	if got := f.sendFields(t, fields, nil); got != http.StatusOK {
		t.Fatal(got)
	}
	if got := f.sendFields(t, f.retoken(fields), nil); got != http.StatusOK || f.messages(t) != 1 {
		t.Fatalf("same bytes again: %d, %d messages", got, f.messages(t))
	}
	other := f.memberPost("Replaced words")
	other.messageID = first.messageID
	if got := f.send(t, other); got != http.StatusOK {
		t.Fatal(got)
	}
	if status, reason := f.statusOf(t, "Replaced words"); status != "held" || reason != store.MailDuplicate {
		t.Fatalf("other bytes, same Message-ID: %s %s", status, reason)
	}
	if f.messages(t) != 2 {
		t.Fatalf("%d messages", f.messages(t))
	}
}

// MEDIUM-1: mail to no project makes no DNS lookup; mail to a project does.
func TestMailUnknownAddressMakesNoLookup(t *testing.T) {
	var lookups atomic.Int64
	f := newMailFixture(t, func(o *httpapi.Options) {
		o.Mail = httpapi.NewMailIn(mailTestDomain, mailTestKey).WithDKIMLookup(func(ctx context.Context, d string) ([]string, error) {
			lookups.Add(1)
			return testKeys.Lookup(ctx, d)
		})
	})
	unknown := f.memberPost("Nobody's")
	unknown.to = "p-22222222222222222222222222222222@" + mailTestDomain
	if got := f.send(t, unknown); got != http.StatusOK || lookups.Load() != 0 {
		t.Fatalf("unknown address: %d, %d lookups", got, lookups.Load())
	}
	f.send(t, f.memberPost("Known"))
	if lookups.Load() == 0 {
		t.Fatal("a project's mail was not verified")
	}
}

// MEDIUM-2: the raw message arriving before the signature fields is spooled
// to a temporary file, read after verification, and the file is removed.
func TestMailRawBeforeSignatureIsSpooled(t *testing.T) {
	f := newMailFixture(t)
	before, _ := filepath.Glob(filepath.Join(os.TempDir(), "sitewise-mail-*"))
	p := f.memberPost("Raw first", pdfFile(1, "A-200.pdf"))
	fields := f.fields(p)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("body-mime", fields["body-mime"])
	for _, k := range []string{"recipient", "sender", "from", "subject", "timestamp", "token", "signature"} {
		_ = w.WriteField(k, fields[k])
	}
	_ = w.Close()
	if got := f.raw(t, w.FormDataContentType(), &body); got != http.StatusOK {
		t.Fatal(got)
	}
	if status, _ := f.statusOf(t, p.subject); status != "filed" || f.documents(t) != 2 {
		t.Fatalf("raw first: %s, %d documents", status, f.documents(t))
	}
	after, _ := filepath.Glob(filepath.Join(os.TempDir(), "sitewise-mail-*"))
	if len(after) > len(before) {
		t.Fatalf("temporary files left: %v", after)
	}
}

// MEDIUM-2: files a stopped process left behind are deleted at start.
func TestRemoveMailTemps(t *testing.T) {
	tmp, err := os.CreateTemp("", "sitewise-mail-*")
	if err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()
	if httpapi.RemoveMailTemps() < 1 {
		t.Fatal("nothing removed")
	}
	if _, err := os.Stat(tmp.Name()); !os.IsNotExist(err) {
		t.Fatalf("left behind: %v", err)
	}
}

// LOW-1: a rewritten forward links no firm or package by rule, even when
// the forwarded sender is a firm on the project.
func TestMailRewrittenForwardLinksNothing(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
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
	fwd := f.memberPost("FW: Wall")
	fwd.body = "See below.\n\n---------- Forwarded message ---------\nFrom: Eng Smith <eng@consult.example>\nSubject: Wall"
	f.send(t, fwd)
	var sender, firm, pkgID, by string
	if err := f.conn.QueryRow(ctx, `SELECT sender, COALESCE(firm_id::text,''), COALESCE(package_id::text,''), COALESCE(linked_by,'') FROM mail_messages WHERE message_id=$1`, fwd.messageID).Scan(&sender, &firm, &pkgID, &by); err != nil {
		t.Fatal(err)
	}
	if sender != "eng@consult.example" || firm != "" || pkgID != "" || by != "" {
		t.Fatalf("rewritten forward: sender %q firm %q package %q by %q", sender, firm, pkgID, by)
	}
}

// LOW-5: two requests staging the same bytes. The first to settle hands
// them to the other and clears only its own row; the last removes them.
func TestMailStagingTwoRequests(t *testing.T) {
	ctx := context.Background()
	f := newMailFixture(t)
	blob, err := f.blobs.Put(ctx, strings.NewReader("%PDF staged by two "+newUUIDString()))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"req-a", "req-b"} {
		if err := f.a.store.StageMailBlob(ctx, blob.SHA256, r); err != nil {
			t.Fatal(err)
		}
	}
	rows := func() int {
		return f.count(t, `SELECT count(*) FROM mail_blob_stages WHERE sha256=$1`, blob.SHA256)
	}
	exists := func() bool {
		p, _ := f.blobs.Path(blob.SHA256)
		_, err := os.Stat(p)
		return err == nil
	}
	if !httpapi.SettleStaged(ctx, f.a.store, f.blobs, blob.SHA256, "req-a", t.Logf) || !exists() || rows() != 1 {
		t.Fatalf("first settle: exists %v rows %d", exists(), rows())
	}
	if !httpapi.SettleStaged(ctx, f.a.store, f.blobs, blob.SHA256, "req-b", t.Logf) || exists() || rows() != 0 {
		t.Fatalf("last settle: exists %v rows %d", exists(), rows())
	}
}

// LOW-4: each refusal's answer.
func TestMailRefusalStatuses(t *testing.T) {
	f := newMailFixture(t)
	bad := f.fields(f.memberPost("Bad"))
	bad["signature"] = strings.Repeat("0", 64)
	stale := f.memberPost("Stale")
	stale.timestamp = time.Now().Add(-25 * time.Hour)
	for name, c := range map[string]struct{ got, want int }{
		"signature":                 {f.sendFields(t, bad, nil), http.StatusNotAcceptable},
		"timestamp":                 {f.send(t, stale), http.StatusNotAcceptable},
		"urlencoded, bad signature": {f.form(t, bad), http.StatusNotAcceptable},
	} {
		if c.got != c.want {
			t.Fatalf("%s = %d, want %d", name, c.got, c.want)
		}
	}
}
