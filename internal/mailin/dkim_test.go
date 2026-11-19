package mailin_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"sitewise/internal/mailin"
	"sitewise/internal/mailin/mailintest"
)

func member() mailintest.Message {
	return mailintest.Message{From: "Pat <pm@victim.example>", To: "p-1@in.example.test", Subject: "Drawings", MessageID: "<m1@victim.example>", Body: "See attached."}
}

// C1: only a signature that verifies, from the From domain's registrable
// domain, authenticates. Headers the sender writes count for nothing.
func TestVerifyDKIM(t *testing.T) {
	ctx := context.Background()
	keys := mailintest.NewKeys()
	signed := keys.Sign(member().Build(), "victim.example")
	if !verified(mailin.Verify(ctx, signed, "pm@victim.example", keys.Lookup)) {
		t.Fatal("an aligned, valid signature did not verify")
	}
	sub := keys.Sign(member().Build(), "mail.victim.example")
	if !verified(mailin.Verify(ctx, sub, "pm@victim.example", keys.Lookup)) {
		t.Fatal("a subdomain of the From domain is aligned")
	}
	// The review's probe: a valid d=evil.test signature beside a forged
	// d=victim.example one, and Mailgun-looking headers the sender wrote.
	forged := member()
	forged.Headers = [][2]string{{"X-Mailgun-Spf", "Pass"}, {"X-Mailgun-Dkim-Check-Result", "Pass"},
		{"DKIM-Signature", "v=1; a=rsa-sha256; d=victim.example; s=t; h=from; bh=AAAA; b=AAAA"}}
	probe := keys.Sign(forged.Build(), "evil.test")
	if verified(mailin.Verify(ctx, probe, "pm@victim.example", keys.Lookup)) {
		t.Fatal("a forged From was authenticated by another domain's signature")
	}
	// Tampering after signing breaks it.
	tampered := bytes.Replace(signed, []byte("See attached."), []byte("Pay this now."), 1)
	if verified(mailin.Verify(ctx, tampered, "pm@victim.example", keys.Lookup)) {
		t.Fatal("a changed body verified")
	}
	// A body length tag leaves content unsigned: never trusted.
	withL := bytes.Replace(signed, []byte("DKIM-Signature: "), []byte("DKIM-Signature: l=3; "), 1)
	if verified(mailin.Verify(ctx, withL, "pm@victim.example", keys.Lookup)) {
		t.Fatal("a signature with l= was trusted")
	}
	// DNS failure or a slow resolver: not authenticated.
	fail := func(context.Context, string) ([]string, error) { return nil, errors.New("SERVFAIL") }
	if verified(mailin.Verify(ctx, signed, "pm@victim.example", fail)) {
		t.Fatal("a failed lookup authenticated")
	}
	slow := func(ctx context.Context, d string) ([]string, error) {
		select {
		case <-time.After(mailin.DNSTimeout + time.Second):
			return keys.Lookup(ctx, d)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	start := time.Now()
	if verified(mailin.Verify(ctx, signed, "pm@victim.example", slow)) || time.Since(start) > mailin.DNSTimeout+500*time.Millisecond {
		t.Fatal("a slow lookup authenticated or was waited for")
	}
	if verified(mailin.Verify(ctx, nil, "pm@victim.example", keys.Lookup)) {
		t.Fatal("no raw message authenticated")
	}
	// A public suffix never aligns.
	if mailin.SameRegistrable("com.au", "example.com.au") || !mailin.SameRegistrable("a.example.com.au", "example.com.au") {
		t.Fatal("registrable domains")
	}
}

func TestParseRawRefusesAmbiguity(t *testing.T) {
	two := member()
	two.Headers = [][2]string{{"From", "other@evil.test"}}
	if _, _, err := mailin.ParseRaw(two.Build()); !errors.Is(err, mailin.ErrFromCount) {
		t.Fatalf("two From headers: %v", err)
	}
	many := member()
	for i := 0; i < mailin.MaxHeaders+1; i++ {
		many.Headers = append(many.Headers, [2]string{"X-Pad", "x"})
	}
	if _, _, err := mailin.ParseRaw(many.Build()); !errors.Is(err, mailin.ErrTooManyHeaders) {
		t.Fatalf("over %d headers: %v", mailin.MaxHeaders, err)
	}
}

func TestParseRawReadsBodyAndFiles(t *testing.T) {
	m := member()
	m.InReplyTo = "<tr1@in.example.test>"
	m.Files = []mailintest.File{{Name: "A-101.pdf", Type: "application/pdf", Body: []byte("%PDF dummy")},
		{Name: "logo.png", Type: "image/png", Body: []byte("png"), Inline: true, ContentID: "logo@x"}}
	got, files, err := mailin.ParseRaw(m.Build())
	if err != nil {
		t.Fatal(err)
	}
	if got.FromAddress != "pm@victim.example" || got.Subject != "Drawings" || got.InReplyTo != "<tr1@in.example.test>" || !strings.Contains(got.BodyPlain, "See attached.") || len(files) != 2 {
		t.Fatalf("%+v %d", got, len(files))
	}
	rc, _ := files[0].Open()
	b, _ := io.ReadAll(rc)
	if string(b) != "%PDF dummy" || mailin.SkipPart(got, files[1], files[1].Name) != mailin.SkipInline {
		t.Fatalf("file %q, logo %q", b, mailin.SkipPart(got, files[1], files[1].Name))
	}
}

func verified(_ mailin.Signed, ok bool) bool { return ok }

// Review HIGH-1: a header prepended above a genuinely signed message is
// refused; headers the signature did not sign are not trusted.
func TestTrustOnlyWhatIsSigned(t *testing.T) {
	ctx := context.Background()
	keys := mailintest.NewKeys()
	m := member()
	m.InReplyTo = "<rfi7@in.example.test>"
	signed := keys.Sign(m.Build(), "victim.example")
	for _, h := range []string{"Subject: Pay this now", "Message-ID: <other@victim.example>", "In-Reply-To: <rfi9@in.example.test>", "Content-Type: text/html", "To: x@evil.test"} {
		forged := append([]byte(h+"\r\n"), signed...)
		if _, _, err := mailin.ParseRaw(forged); !errors.Is(err, mailin.ErrDuplicateHeader) {
			t.Fatalf("prepended %q: %v", h, err)
		}
	}
	// Signed over From, Subject, Message-ID and Date only: To and In-Reply-To
	// are blanked; without Subject signed it is not trusted.
	narrow := keys.SignHeaders(m.Build(), "victim.example", []string{"From", "Subject", "Message-ID", "Date", "MIME-Version", "Content-Type"})
	got, _, err := mailin.ParseRaw(narrow)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := mailin.Verify(ctx, narrow, got.FromAddress, keys.Lookup)
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	if !ok || !mailin.Trust(&got, s, now) || got.InReplyTo != "" || got.To != "" {
		t.Fatalf("narrow: ok %v in-reply-to %q to %q", ok, got.InReplyTo, got.To)
	}
	noSubject := keys.SignHeaders(m.Build(), "victim.example", []string{"From", "Message-ID", "Date", "Content-Type"})
	got, _, _ = mailin.ParseRaw(noSubject)
	s, ok = mailin.Verify(ctx, noSubject, got.FromAddress, keys.Lookup)
	if !ok || mailin.Trust(&got, s, now) {
		t.Fatal("an unsigned Subject was trusted")
	}
	// An old genuine message is not trusted as new.
	got, _, _ = mailin.ParseRaw(signed)
	s, _ = mailin.Verify(ctx, signed, got.FromAddress, keys.Lookup)
	if mailin.Trust(&got, s, time.Now().Add(8*24*time.Hour)) {
		t.Fatal("a signature over a week old was trusted")
	}
}
