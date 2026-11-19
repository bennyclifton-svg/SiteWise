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
	if !mailin.Verify(ctx, signed, "pm@victim.example", keys.Lookup) {
		t.Fatal("an aligned, valid signature did not verify")
	}
	sub := keys.Sign(member().Build(), "mail.victim.example")
	if !mailin.Verify(ctx, sub, "pm@victim.example", keys.Lookup) {
		t.Fatal("a subdomain of the From domain is aligned")
	}
	// The review's probe: a valid d=evil.test signature beside a forged
	// d=victim.example one, and Mailgun-looking headers the sender wrote.
	forged := member()
	forged.Headers = [][2]string{{"X-Mailgun-Spf", "Pass"}, {"X-Mailgun-Dkim-Check-Result", "Pass"},
		{"DKIM-Signature", "v=1; a=rsa-sha256; d=victim.example; s=t; h=from; bh=AAAA; b=AAAA"}}
	probe := keys.Sign(forged.Build(), "evil.test")
	if mailin.Verify(ctx, probe, "pm@victim.example", keys.Lookup) {
		t.Fatal("a forged From was authenticated by another domain's signature")
	}
	// Tampering after signing breaks it.
	tampered := bytes.Replace(signed, []byte("See attached."), []byte("Pay this now."), 1)
	if mailin.Verify(ctx, tampered, "pm@victim.example", keys.Lookup) {
		t.Fatal("a changed body verified")
	}
	// A body length tag leaves content unsigned: never trusted.
	withL := bytes.Replace(signed, []byte("DKIM-Signature: "), []byte("DKIM-Signature: l=3; "), 1)
	if mailin.Verify(ctx, withL, "pm@victim.example", keys.Lookup) {
		t.Fatal("a signature with l= was trusted")
	}
	// DNS failure or a slow resolver: not authenticated.
	fail := func(context.Context, string) ([]string, error) { return nil, errors.New("SERVFAIL") }
	if mailin.Verify(ctx, signed, "pm@victim.example", fail) {
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
	if mailin.Verify(ctx, signed, "pm@victim.example", slow) || time.Since(start) > mailin.DNSTimeout+500*time.Millisecond {
		t.Fatal("a slow lookup authenticated or was waited for")
	}
	if mailin.Verify(ctx, nil, "pm@victim.example", keys.Lookup) {
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
