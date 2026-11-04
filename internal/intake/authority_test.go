package intake

import (
	"testing"

	"sitewise/internal/identity"
)

// F7: a council's planning certificate was filed as a town planner's
// document. A council issues it; no consultant discipline applies.
func TestAuthorityDocumentsHaveNoDiscipline(t *testing.T) {
	md := func(lines ...string) identity.Text {
		t := identity.Text{Format: "md"}
		for _, l := range lines {
			t.Runs = append(t.Runs, identity.Run{Text: l})
		}
		return t
	}
	pdf := func(lines ...string) identity.Text {
		t := md(lines...)
		t.Format = "pdf"
		return t
	}
	for _, tc := range []struct {
		name     string
		filename string
		text     identity.Text
		want     bool
	}{
		{"synthetic s10.7 certificate", "Planning certificate s10.7 - Lot 23 DP 30418.md",
			md("CITY OF CANADA BAY", "Customer Service Centre, Drummoyne NSW", "# Planning Certificate", "Issued under section 10.7(2) and (5)"), true},
		{"certificate by heading only", "pc.md", md("# Planning Certificate", "body"), true},
		{"certificate pdf by filename", "App 19 - 10.7 Planning Certificate.pdf", pdf("STRATHFIELD COUNCIL", "Planning Certificate"), true},
		{"section 149 certificate pdf", "scan.pdf", pdf("Ref: 123", "SECTION 149 (2) CERTIFICATE", "Property: Lot 1"), true},
		{"council letter", "Letter.md", md("Inner West Council", "Our ref: DA2026/1", "Dear Sir"), true},
		{"shire letterhead", "x.pdf", pdf("Shire of Hornsby", "Notice of determination"), true},

		{"town planner SEE mentions the certificate in body", "Statement of environmental effects.md",
			md("# Statement of environmental effects", "The section 10.7 planning certificate states the land is zoned R2."), false},
		{"planner's review of a certificate", "Planning certificate review - Willowtree.pdf", pdf("Willowtree Planning", "Planning certificate review"), false},
		{"fee proposal for a certificate", "Fee proposal - planning certificate.md", md("# Fee proposal: planning certificate"), false},
		{"architect drawing for a council client", "A-101.pdf", pdf("City of Sydney", "Drawing No A-101", "Ground floor plan"), false},
		{"council named in body", "Report.md", md("# Geotechnical report", "Inner West Council"), false},
		{"consultant report", "Geo.pdf", pdf("Sandstone Geotechnical", "Geotechnical investigation"), false},
	} {
		got := authorityIssuer(tc.filename, tc.text)
		if (len(got) == 1) != tc.want {
			t.Errorf("%s: %+v", tc.name, got)
			continue
		}
		if tc.want && (got[0].Field != FieldDiscipline || !got[0].Settled || got[0].Display != "") {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}
}

// The rule settles discipline as none before the fan-out, so Jev is not asked
// a question whose options cannot hold the issuer.
func TestPlanningCertificateSkipsDisciplineQuestion(t *testing.T) {
	text := identity.Text{Format: "md", Runs: []identity.Run{{Text: "CITY OF CANADA BAY"}, {Text: "# Planning Certificate"}}}
	d := NewDraft(Catalog{}, "Planning certificate s10.7 - Lot 23 DP 30418.md", text, nil, nil)
	got, ok := d.by[FieldDiscipline]
	if !ok || got.Value != "" || got.Band != BandGreen || got.DecidedBy != DecidedByRule {
		t.Fatalf("discipline %+v", got)
	}
	d.Plan(nil, "self")
	for _, q := range d.questions {
		if q.Field == FieldDiscipline {
			t.Fatal("discipline still asked")
		}
	}
}
