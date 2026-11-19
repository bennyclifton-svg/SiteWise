package mailin

import "strings"

// The sender check (email intake plan, Task 3) lives alone in this file so
// that, if Task 0 finds Mailgun does not put its SPF and DKIM results in
// message-headers, it can be replaced (for example by DKIM verified in Go
// from a raw-MIME route) without touching the webhook, the store or the
// screen. The webhook calls only Authenticated.
//
// Until Task 0 has been run, an absent header is never a pass: a message
// with neither result is unverified and is held (fail closed).

// Authenticated is true when the From domain is vouched for by Mailgun's
// checks of this message:
//
//   - SPF passed (X-Mailgun-Spf: Pass) and the envelope sender's domain
//     matches the From domain; or
//   - DKIM passed (X-Mailgun-Dkim-Check-Result: Pass) and a DKIM-Signature
//     header's d= matches the From domain.
//
// Two domains match when they are equal or one ends with a dot and the
// other (mail.example.com and example.com). Missing headers mean false.
// Mailgun reports one DKIM result for a message that may carry several
// signatures, so an aligned pass is strong evidence, not proof.
// https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/spam-filter
func Authenticated(m Message) bool {
	from := Domain(m.FromAddress)
	if from == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(m.Header("X-Mailgun-Spf")), "Pass") && aligned(Domain(m.EnvelopeSender), from) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(m.Header("X-Mailgun-Dkim-Check-Result")), "Pass") {
		for _, sig := range m.HeaderValues("DKIM-Signature") {
			if aligned(dkimDomain(sig), from) {
				return true
			}
		}
	}
	return false
}

func aligned(a, b string) bool {
	// A bare top-level name ("com") never aligns with anything.
	if !strings.Contains(a, ".") || !strings.Contains(b, ".") {
		return false
	}
	return a == b || strings.HasSuffix(a, "."+b) || strings.HasSuffix(b, "."+a)
}

// dkimDomain is the d= tag of a DKIM-Signature header.
func dkimDomain(sig string) string {
	for _, tag := range strings.Split(sig, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(tag), "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), "d") {
			return strings.ToLower(strings.TrimSpace(v))
		}
	}
	return ""
}
