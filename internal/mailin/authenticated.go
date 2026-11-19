package mailin

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-msgauth/dkim"
	"golang.org/x/net/publicsuffix"
)

// The sender check (email intake plan, Task 3; security review C1, 19
// November 2026). Headers in the post are written by the sender, so none
// is trusted: not X-Mailgun-Spf, not X-Mailgun-Dkim-Check-Result, not a
// DKIM-Signature's d= on its own. A sender is authenticated only when a
// DKIM signature over the raw message (Mailgun's raw-MIME post, RawField)
// verifies in Go and its signing domain is the From domain's registrable
// domain. Any doubt is not authenticated, and the message is held:
//
//   - no raw message, or anything other than exactly one From header
//     (ParseRaw refuses those);
//   - more than MaxHeaders headers (ParseRaw);
//   - a signature with a body length tag (l=), which leaves content unsigned;
//   - a DNS lookup that fails or takes longer than DNSTimeout.
//
// This file is the only place the check lives; the webhook calls Verify.

// DNSTimeout bounds all the DKIM key lookups of one message together
// (review MEDIUM-1).
const DNSTimeout = 1 * time.Second

// MaxSignatures is how many DKIM signatures are checked.
const MaxSignatures = 3

// MaxSignatureAge: a signature (by its t=, or the message's Date when it
// has none) older than this is not trusted, so an old genuine message
// cannot be posted again as new (review HIGH-1).
const MaxSignatureAge = 7 * 24 * time.Hour

// Signed is what the verifying, aligned signature covers: the names of
// the headers it signed (lower case) and when it was made (zero if t= is
// absent).
type Signed struct {
	Headers map[string]bool
	Time    time.Time
}

// Has reports a header the signature signed.
func (s Signed) Has(name string) bool { return s.Headers[strings.ToLower(name)] }

// LookupTXT returns a domain's TXT records. The webhook passes the server's
// resolver; tests pass a stub, so no test reaches the network.
type LookupTXT func(ctx context.Context, domain string) ([]string, error)

// NetLookupTXT is the system resolver.
func NetLookupTXT(ctx context.Context, domain string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, domain)
}

// Verify reports whether raw carries a DKIM signature that verifies and is
// aligned with from's registrable domain, and what that signature covers.
// from is the address read from raw's own single From header (ParseRaw),
// never from a post field. Trust then decides which recorded headers count.
func Verify(ctx context.Context, raw []byte, from string, lookup LookupTXT) (Signed, bool) {
	if len(raw) == 0 || lookup == nil {
		return Signed{}, false
	}
	fromDomain := registrable(Domain(from))
	if fromDomain == "" {
		return Signed{}, false
	}
	if bodyLengthTag(headBytes(raw)) {
		return Signed{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, DNSTimeout)
	defer cancel()
	verifications, err := dkim.VerifyWithOptions(bytes.NewReader(raw), &dkim.VerifyOptions{
		MaxVerifications: MaxSignatures,
		LookupTXT: func(domain string) ([]string, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return lookup(ctx, domain)
		},
	})
	if (err != nil && !errors.Is(err, dkim.ErrTooManySignatures)) || ctx.Err() != nil {
		return Signed{}, false
	}
	// The aligned signature that signs the most of what is recorded.
	var best *Signed
	for _, v := range verifications {
		if v.Err != nil || registrable(v.Domain) != fromDomain {
			continue
		}
		s := Signed{Headers: map[string]bool{}, Time: v.Time}
		for _, k := range v.HeaderKeys {
			s.Headers[strings.ToLower(strings.TrimSpace(k))] = true
		}
		if best == nil || len(s.Headers) > len(best.Headers) {
			best = &s
		}
	}
	if best == nil {
		return Signed{}, false
	}
	return *best, true
}

// Trust applies what the signature covers to the message (review HIGH-1).
// From, Subject, Message-Id and Date must be signed, and Content-Type when
// the message has one, or the sender is not authenticated. In-Reply-To,
// References, To and Cc that are not signed are blanked: no thread, no
// rule link, no reply admission from words the signature did not cover. A
// signature older than MaxSignatureAge (by t=, or by Date without it) is
// not trusted.
func Trust(m *Message, s Signed, now time.Time) bool {
	for _, h := range []string{"from", "subject", "message-id", "date"} {
		if !s.Has(h) {
			return false
		}
	}
	if m.Header("Content-Type") != "" && !s.Has("content-type") {
		return false
	}
	at := s.Time
	if at.IsZero() {
		at = m.Date
	}
	if at.IsZero() || now.Sub(at) > MaxSignatureAge {
		return false
	}
	if !s.Has("in-reply-to") {
		m.InReplyTo = ""
	}
	if !s.Has("references") {
		m.References = ""
	}
	if !s.Has("to") {
		m.To = ""
	}
	if !s.Has("cc") {
		m.Cc = ""
	}
	return true
}

func headBytes(raw []byte) []byte {
	if i := bytes.Index(raw, []byte("\r\n\r\n")); i >= 0 {
		return raw[:i]
	}
	if i := bytes.Index(raw, []byte("\n\n")); i >= 0 {
		return raw[:i]
	}
	return raw
}

// registrable is a domain's registrable part (example.com for
// mail.example.com, example.com.au for x.example.com.au), or "" for a bare
// public suffix.
func registrable(domain string) string {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" || !strings.Contains(domain, ".") {
		return ""
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return ""
	}
	return d
}

// SameRegistrable reports whether two domains share a registrable domain.
func SameRegistrable(a, b string) bool {
	ra := registrable(a)
	return ra != "" && ra == registrable(b)
}

// Registrable is a domain's registrable part, or "".
func Registrable(domain string) string { return registrable(domain) }

// bodyLengthTag reports a DKIM-Signature header with an l= tag.
func bodyLengthTag(head []byte) bool {
	for _, s := range unfolded(head, "DKIM-Signature") {
		for _, tag := range strings.Split(s, ";") {
			k, _, ok := strings.Cut(strings.TrimSpace(tag), "=")
			if ok && strings.EqualFold(strings.TrimSpace(k), "l") {
				return true
			}
		}
	}
	return false
}

// unfolded returns the values of every header named name in head.
func unfolded(head []byte, name string) []string {
	var out []string
	cur, in := "", false
	flush := func() {
		if in {
			out = append(out, cur)
		}
	}
	for _, l := range strings.Split(strings.ReplaceAll(string(head), "\r\n", "\n"), "\n") {
		if l != "" && (l[0] == ' ' || l[0] == '\t') {
			cur += l
			continue
		}
		flush()
		k, v, ok := strings.Cut(l, ":")
		in, cur = ok && strings.EqualFold(strings.TrimSpace(k), name), v
	}
	flush()
	return out
}
