package auth

import (
	"bytes"
	"net/mail"
	"strings"
	"testing"
	"time"
)

// Gmail refuses or junks account mail (reset, confirm, invite) without a
// valid Date and Message-ID, so each message carries its own.
func TestAccountMessageHasDateAndMessageID(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC)
	raw, err := accountMessage("no-reply@sitewise.au", "pm@example.com", "Reset your SiteWise password", "Open this link.\r\n", now)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("not a parseable message: %v", err)
	}
	if d, err := m.Header.Date(); err != nil || !d.Equal(now) {
		t.Fatalf("Date = %v, %v", d, err)
	}
	id := m.Header.Get("Message-ID")
	if !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, "@sitewise.au>") {
		t.Fatalf("Message-ID = %q", id)
	}
	again, _ := accountMessage("no-reply@sitewise.au", "pm@example.com", "s", "b", now)
	m2, _ := mail.ReadMessage(bytes.NewReader(again))
	if m2.Header.Get("Message-ID") == id {
		t.Fatal("Message-ID repeated")
	}
	if m.Header.Get("To") != "pm@example.com" || m.Header.Get("Subject") != "Reset your SiteWise password" {
		t.Fatalf("headers = %v", m.Header)
	}
}
