package mailin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func sign(key, ts, token string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(ts + token))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	const key, ts, token = "test-signing-key", "1763510400", "abcdef0123456789"
	good := sign(key, ts, token)
	cases := []struct {
		name                string
		key, ts, token, sig string
		want                bool
	}{
		{"good", key, ts, token, good, true},
		{"upper-case signature", key, ts, token, strings.ToUpper(good), true},
		{"changed token", key, ts, token + "x", good, false},
		{"changed timestamp", key, "1763510401", token, good, false},
		{"empty key", "", ts, token, sign("", ts, token), false},
		{"empty signature", key, ts, token, "", false},
		{"other key", "another-key", ts, token, good, false},
	}
	for _, c := range cases {
		if got := VerifySignature(c.key, c.ts, c.token, c.sig); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestFresh(t *testing.T) {
	now := time.Unix(1763510400, 0)
	for ts, want := range map[string]bool{
		"1763510400": true,
		"1763424001": true,  // 23:59:59 old
		"1763423999": false, // over a day old
		"1763510700": true,  // five minutes ahead
		"1763511000": false, // ten minutes ahead
		"soon":       false,
		"":           false,
	} {
		if got := Fresh(ts, now); got != want {
			t.Errorf("%q: got %v", ts, got)
		}
	}
}

func TestSenderAddress(t *testing.T) {
	addr, name, err := SenderAddress("Jo Chen <Jo.Chen@Example.com>")
	if err != nil || addr != "jo.chen@example.com" || name != "Jo Chen" {
		t.Fatalf("%q %q %v", addr, name, err)
	}
	for _, bad := range []string{"", "a@example.com, b@example.com", "not an address", "Jo <jo@example.com>\r\nBcc: x@y.z"} {
		if _, _, err := SenderAddress(bad); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

func headers(pairs ...string) [][2]string {
	var out [][2]string
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, [2]string{pairs[i], pairs[i+1]})
	}
	return out
}

func TestAuthenticated(t *testing.T) {
	cases := []struct {
		spf, envelope, dkim, d string
		want                   bool
	}{
		{"Pass", "pm@example.com", "", "", true},
		{"Pass", "bounce@mail.example.com", "", "", true},
		{"Pass", "x@attacker.net", "", "", false},
		{"Fail", "pm@example.com", "Pass", "example.com", true},
		{"", "pm@example.com", "Pass", "attacker.net", false},
		{"", "pm@example.com", "", "", false},
		{"SoftFail", "pm@example.com", "Fail", "example.com", false},
		{"Neutral", "pm@example.com", "", "", false},
		{"Pass", "x@com", "", "", false},
	}
	for i, c := range cases {
		m := Message{FromAddress: "pm@example.com", EnvelopeSender: c.envelope}
		if c.spf != "" {
			m.Headers = append(m.Headers, [2]string{"X-Mailgun-Spf", c.spf})
		}
		if c.dkim != "" {
			m.Headers = append(m.Headers, [2]string{"X-Mailgun-Dkim-Check-Result", c.dkim})
		}
		if c.d != "" {
			m.Headers = append(m.Headers, [2]string{"DKIM-Signature", "v=1; a=rsa-sha256; d=" + c.d + "; s=sel; b=abc"})
		}
		if got := Authenticated(m); got != c.want {
			t.Errorf("case %d %+v: got %v", i, c, got)
		}
	}
}

func TestDropped(t *testing.T) {
	base := func(h ...string) Message {
		return Message{FromAddress: "pm@example.com", EnvelopeSender: "pm@example.com", Headers: headers(h...)}
	}
	for name, m := range map[string]Message{
		"spam flag":    base("X-Mailgun-Sflag", "Yes"),
		"auto-replied": base("Auto-Submitted", "auto-replied"),
		"bulk":         base("Precedence", "bulk"),
		"x-autoreply":  base("X-Autoreply", "yes"),
		"mailing list": base("List-Id", "<team.lists.example.com>"),
		"bounce":       {FromAddress: "mailer-daemon@example.com", EnvelopeSender: ""},
		"report":       base("Content-Type", "multipart/report; report-type=delivery-status"),
		"empty sender": {FromAddress: "pm@example.com", EnvelopeSender: "<>"},
	} {
		if !Dropped(m) {
			t.Errorf("%s: not dropped", name)
		}
	}
	for name, m := range map[string]Message{
		"auto-submitted no": base("Auto-Submitted", "no"),
		"ordinary":          base("Subject", "Drawings"),
	} {
		if Dropped(m) {
			t.Errorf("%s: dropped", name)
		}
	}
}

func fileHeader(field, name, ctype string, size int64) *multipart.FileHeader {
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+name+`"`)
	h.Set("Content-Type", ctype)
	return &multipart.FileHeader{Filename: name, Header: h, Size: size}
}

func TestSkipPart(t *testing.T) {
	m := Message{InlineIDs: map[string]bool{"attachment-1": true, "attachment-2": true}}
	cases := []struct {
		f    *multipart.FileHeader
		want string
	}{
		{fileHeader("attachment-1", "logo.png", "image/png", 20<<10), SkipInline},
		{fileHeader("attachment-2", "photo.jpg", "image/jpeg", 2<<20), ""},
		{fileHeader("attachment-3", "small.png", "image/png", 10<<10), ""}, // attached, not inline
		{fileHeader("attachment-4", "invite.ics", "application/octet-stream", 1000), SkipCalendar},
		{fileHeader("attachment-5", "meeting", "text/calendar", 1000), SkipCalendar},
		{fileHeader("attachment-6", "winmail.dat", "application/ms-tnef", 1000), SkipWinmail},
		{fileHeader("attachment-7", "A-101.pdf", "application/pdf", 1<<20), ""},
	}
	for _, c := range cases {
		if got := SkipPart(m, c.f, c.f.Filename); got != c.want {
			t.Errorf("%s: got %q want %q", c.f.Filename, got, c.want)
		}
	}
}

func TestAttachmentName(t *testing.T) {
	for raw, want := range map[string]string{
		"A-101.pdf":              "A-101.pdf",
		"../../etc/passwd":       "passwd",
		`C:\Users\pm\report.pdf`: "report.pdf",
		"":                       "attachment-3",
		"..":                     "attachment-3",
		"bad\x00name.pdf":        "bad name.pdf",
	} {
		if got := AttachmentName(raw, 3); got != want {
			t.Errorf("%q: got %q want %q", raw, got, want)
		}
	}
}

func TestBodyDocument(t *testing.T) {
	date := time.Date(2026, 11, 14, 9, 15, 0, 0, time.FixedZone("AEDT", 11*3600))
	m := Message{FromAddress: "jo@example.com", FromName: "Jo Chen", To: "Project <p-1@in.example.test>",
		Subject: "Basement drawings\r\nBcc: victim@example.net", Date: date, BodyPlain: "Please see attached.\n# not a heading at the top"}
	name, body := BodyDocument(m, time.Now())
	want := "From: Jo Chen <jo@example.com>\nTo: Project <p-1@in.example.test>\nSubject: Basement drawings Bcc: victim@example.net\nDate: 14 November 2026 9:15 AM\n\nPlease see attached.\n# not a heading at the top\n"
	if string(body) != want {
		t.Fatalf("body:\n%s\nwant:\n%s", body, want)
	}
	if name != "Basement drawings Bcc victim@example.net (email 2026-11-14).md" {
		t.Fatalf("name %q", name)
	}
	m.Subject = "# Heading?"
	_, body = BodyDocument(m, time.Now())
	if strings.Contains(string(body), "\n# Heading") || !strings.Contains(string(body), "Subject: # Heading?\n") {
		t.Fatalf("subject became a heading:\n%s", body)
	}
	if n := BodyFilename("", date); n != "Email 2026-11-14.md" {
		t.Fatalf("empty subject %q", n)
	}
	if n := BodyFilename(strings.Repeat("ä", 300), date); len(n) > 255 || !strings.HasSuffix(n, "(email 2026-11-14).md") {
		t.Fatalf("long subject %d %q", len(n), n)
	}
	// No Date header: the time received in Sydney.
	m.Date = time.Time{}
	name, body = BodyDocument(m, time.Date(2026, 11, 13, 22, 0, 0, 0, time.UTC))
	if !strings.Contains(string(body), "Date: 14 November 2026 9:00 AM") || !strings.Contains(name, "2026-11-14") {
		t.Fatalf("received date: %q\n%s", name, body)
	}
}

func TestPlainBody(t *testing.T) {
	if got := PlainBody("plain\r\ntext", "<b>html</b>"); got != "plain\ntext" {
		t.Fatalf("plain %q", got)
	}
	h := `<html><head><style>p{}</style></head><body><p>Hello &amp; welcome</p><img src="https://tracker.example/x.gif"><a href="https://evil.example">click</a><script>alert(1)</script></body></html>`
	got := PlainBody("", h)
	if got != "Hello & welcome\nclick" {
		t.Fatalf("html %q", got)
	}
}

func TestRecipientLocalPart(t *testing.T) {
	for raw, want := range map[string]string{
		"p-abc@in.example.test":                     "p-abc",
		"P-ABC@IN.EXAMPLE.TEST":                     "p-abc",
		"Project <p-abc@in.example.test>":           "p-abc",
		"someone@other.test, p-def@in.example.test": "p-def",
		"p-abc@in.example.test.evil.test":           "",
		"":                                          "",
	} {
		got, ok := RecipientLocalPart(raw, "in.example.test")
		if got != want || ok != (want != "") {
			t.Errorf("%q: got %q %v", raw, got, ok)
		}
	}
}

func TestReadPost(t *testing.T) {
	form := map[string]string{
		"recipient":       "p-abc@in.example.test",
		"sender":          "Bounce@Mail.Example.com",
		"from":            "Jo Chen <jo@example.com>",
		"subject":         "Re: TR-001",
		"body-plain":      "Thanks.",
		"message-headers": `[["Message-Id","<m2@example.com>"],["In-Reply-To","<tr1@in.example.test>"],["References","<a@x> <tr1@in.example.test>"],["To","p-abc@in.example.test"],["Date","Fri, 14 Nov 2026 09:15:00 +1100"],["X-Mailgun-Spf","Pass"]]`,
		"content-id-map":  `{"<logo>":"attachment-1"}`,
	}
	m, err := Read(func(k string) string { return form[k] })
	if err != nil {
		t.Fatal(err)
	}
	if m.FromAddress != "jo@example.com" || m.EnvelopeSender != "bounce@mail.example.com" || m.MessageID != "<m2@example.com>" ||
		m.InReplyTo != "<tr1@in.example.test>" || m.References != "<a@x> <tr1@in.example.test>" || m.To != "p-abc@in.example.test" ||
		m.Date.IsZero() || !m.InlineIDs["attachment-1"] || !Authenticated(m) {
		t.Fatalf("%+v", m)
	}
	form["message-headers"] = "not json"
	m, err = Read(func(k string) string { return form[k] })
	if err != nil || len(m.Headers) != 0 || Authenticated(m) {
		t.Fatalf("bad headers read as %+v %v", m, err)
	}
	form["from"] = ""
	if _, err := Read(func(k string) string { return form[k] }); err == nil {
		t.Fatal("no From accepted")
	}
}
