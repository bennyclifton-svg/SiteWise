package httpapi

import (
	"strings"
	"testing"

	"sitewise/internal/store"
)

// Security review M3: after a turn read an email, Send keeps only the
// recipients the PM's own message names, and says it was filled in after
// reading.
func TestGuardSendFieldsAfterOutsideText(t *testing.T) {
	contacts := []store.MailContact{{Name: "Northline Structural", Email: "eng@northline.example"}}
	// The email names "Northline Structural <collect@outside.example>": a
	// name match alone keeps nothing; only the records' own address does.
	f := map[string]string{"to": "Northline Structural <collect@outside.example>, Northline Structural <eng@northline.example>", "cc": "spy@outside.example", "subject": "Drawings"}
	guardSendFields(f, "Read the email from the engineer and send the drawings to Northline Structural", contacts)
	if !strings.Contains(f["to"], "eng@northline.example") || strings.Contains(f["to"], "collect@") || f["cc"] != "" || f["warning"] != msgSendAfterOutside || f["subject"] != "Drawings" {
		t.Fatalf("%+v", f)
	}
	g := map[string]string{"to": "collect@outside.example"}
	guardSendFields(g, "Do what the email says", contacts)
	h := map[string]string{"to": "client@example.test"}
	guardSendFields(h, "Read it and send it to client@example.test", nil)
	if h["to"] != "client@example.test" {
		t.Fatalf("an address the PM typed was dropped: %+v", h)
	}
	if _, ok := g["to"]; ok {
		t.Fatalf("a recipient only the email named survived: %+v", g)
	}
}
