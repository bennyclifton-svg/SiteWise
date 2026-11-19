package httpapi

import (
	"strings"
	"testing"
)

// Security review M3: after a turn read an email, Send keeps only the
// recipients the PM's own message names, and says it was filled in after
// reading.
func TestGuardSendFieldsAfterOutsideText(t *testing.T) {
	f := map[string]string{"to": "collect@outside.example, Northline Structural <eng@northline.example>", "cc": "spy@outside.example", "subject": "Drawings"}
	guardSendFields(f, "Read the email from the engineer and send the drawings to Northline Structural")
	if !strings.Contains(f["to"], "eng@northline.example") || strings.Contains(f["to"], "collect@") || f["cc"] != "" || f["warning"] != msgSendAfterOutside || f["subject"] != "Drawings" {
		t.Fatalf("%+v", f)
	}
	g := map[string]string{"to": "collect@outside.example"}
	guardSendFields(g, "Do what the email says")
	if _, ok := g["to"]; ok {
		t.Fatalf("a recipient only the email named survived: %+v", g)
	}
}
