package rfp

import (
	"encoding/json"
	"strings"
	"testing"
)

// The Terms table (owner, 8 Nov 2026): the agreement is chosen in the
// repository and named by title alone (no revision or date table), every
// other term starts with an answer, and the contact and questions date are
// gone.
func TestTermsTableAgreementDocuments(t *testing.T) {
	doc := `[{"id":"d1","label":"Document revision","text":"x","origin":"document","review_status":"proposed","meaning":"stated","basis":{"sources":[{"document_id":"d1","title":"Consultant Agreement","revision":"B","date":"2026-10-01"}]}}]`
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "work_type=new", "class=2"), set(AgreementDocs, doc))
	if a := r.Fields["agreement"]; a.Value != "Consultant Agreement" || a.Source != "document" {
		t.Fatalf("agreement = %+v", a)
	}
	for _, c := range r.Checks {
		if c.ID == "terms" {
			t.Fatalf("attached agreement still blocks issue: %+v", c)
		}
	}
	if !strings.HasPrefix(r.Terms[0].Text, "Engagement is under Consultant Agreement;") {
		t.Fatalf("first term = %q", r.Terms[0].Text)
	}
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sections {
		if s.ID != "terms" {
			continue
		}
		if len(s.Blocks) != 1 || s.Blocks[0].ID != "rfp:terms" {
			t.Fatalf("terms blocks = %+v", s.Blocks)
		}
		var basis struct {
			AgreementIDs []string `json:"agreement_ids"`
		}
		if err := json.Unmarshal(s.Blocks[0].Basis, &basis); err != nil || strings.Join(basis.AgreementIDs, ",") != "d1" {
			t.Fatalf("terms basis carries the chosen agreement ids: %s", s.Blocks[0].Basis)
		}
		if strings.Contains(s.Blocks[0].Text, "questions in writing") {
			t.Fatalf("questions term still prints: %q", s.Blocks[0].Text)
		}
		if !strings.Contains(s.Blocks[0].Text, "Rates hold until 12 months after appointment;") {
			t.Fatalf("rates term has no default: %q", s.Blocks[0].Text)
		}
	}
}

func TestTermsWithoutAgreementBlockIssue(t *testing.T) {
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "work_type=new", "class=2"), Settings{})
	found := false
	for _, c := range r.Checks {
		found = found || (c.ID == "terms" && c.Blocking && c.Control == "field:agreement")
	}
	if !found {
		t.Fatalf("a missing agreement must block issue: %+v", r.Checks)
	}
	if SharedSetting(skeleton(t), AgreementDocs) != true {
		t.Fatal("agreement documents print once in a combined engagement")
	}
}
