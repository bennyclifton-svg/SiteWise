package correspondence

import (
	"bytes"
	"errors"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestBuildRefusesHeaderInjection(t *testing.T) {
	base := Outgoing{From: mail.Address{Name: "Org via SiteWise", Address: "s@example.test"}, To: []mail.Address{{Address: "a@example.net"}},
		Subject: "TR-001", MessageID: "<1@example.test>", Date: time.Now()}
	for name, m := range map[string]Outgoing{
		"subject CRLF": func() Outgoing { x := base; x.Subject = "Hi\r\nBcc: v@example.net"; return x }(),
		"subject LF":   func() Outgoing { x := base; x.Subject = "Hi\nBcc: v@example.net"; return x }(),
		"from name":    func() Outgoing { x := base; x.From.Name = "Org\r\nBcc: v@example.net"; return x }(),
		"file name":    func() Outgoing { x := base; x.Files = []File{{Name: "a\r\n.pdf", Body: []byte("x")}}; return x }(),
		"reference":    func() Outgoing { x := base; x.References = []string{"<a>\r\nX: y"}; return x }(),
	} {
		if _, err := Build(m); !errors.Is(err, ErrHeader) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, bad := range []string{"a@example.net\r\nBcc: v@example.net", "a@example.net\n", "a@example.net "} {
		if _, err := ParseRecipients([]string{bad}); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
	raw, err := Build(func() Outgoing { x := base; x.Subject = "Éclairage — TR-001"; x.Body = "Line one\nLine two"; return x }())
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Header) == 0 || msg.Header.Get("Bcc") != "" || !strings.HasPrefix(msg.Header.Get("Subject"), "=?utf-8?q?") {
		t.Fatalf("headers %v", msg.Header)
	}
}
