package intake_test

import (
	"context"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"strings"
	"testing"
)

func TestMarkdownOwnIdentity(t *testing.T) {
	cases := []struct{ name, body, title, number, revision, date string }{
		{"report", "# CONDITION REPORT\nLicence BPC 998877\n**Report ref:** BC-DIL-2026-3319\n**Inspection date:** 16 May 2026\n## Findings\nRefer to report XX-1234 revision B", "CONDITION REPORT", "BC-DIL-2026-3319", "", ""},
		{"geotech", "# GEOTECHNICAL INVESTIGATION REPORT\n**Report No:** TT-GEO-2026-0442\n**Investigation date:** 15 May 2026\n**Issue date:** 22 May 2026\n## Findings", "GEOTECHNICAL INVESTIGATION REPORT", "TT-GEO-2026-0442", "", "22 May 2026"},
		{"programme", "# MASTER PROGRAMME\n**Issue date:** 28 May 2026\n**Version:** v0.1 (draft for owner review)\n## Stages\n16 May 2026", "MASTER PROGRAMME", "", "v0.1", "28 May 2026"},
		{"survey", "# SURVEY REPORT\n**Job No:** NSS-26-1188\n**Survey date:** 6 May 2026\n**Report date:** 9 May 2026", "SURVEY REPORT", "", "", "9 May 2026"},
		{"letter", "# LETTER OF ENGAGEMENT\nStudio Pty Ltd\n14 May 2026\nDear owners,\nProposal dated 28 April 2026\n## Acceptance\nDate: 16/05/2026", "LETTER OF ENGAGEMENT", "", "", "14 May 2026"},
		{"email", "From: Author <a@example.com>\nTo: Owner <b@example.com>\nSubject: Brief signoff\nDate: 13 May 2026 4:42 PM\nBody revision v1.1; signed v1.0 governs.\n---\nFrom: Owner <b@example.com>\nTo: Author <a@example.com>\nSubject: RE: Brief signoff\nDate: 14 May 2026 9:15 AM\nAttachment report no: ABC-1234", "Brief signoff", "", "", "14 May 2026"},
		{"conflict", "# REPORT\n**Report No:** AA-1234\n**Report No:** BB-1234\n**Date:** 1 May 2026\n**Date:** 2 May 2026", "REPORT", "", "", ""},
		{"body references", "# REPORT\n## Discussion\nDocument number: OTHER-1234\nRevision: C\nDate: 1 May 2026", "REPORT", "", "", ""},
		{"code example", "# REPORT\n```\nDocument number: OTHER-1234\nRevision: C\n```", "REPORT", "", "", ""},
		{"invalid date", "# REPORT\n**Date:** 31 February 2026", "REPORT", "", "", ""},
		{"licence only", "# REPORT\nLicence BPC 998877\nABN 12 345 678 901\n## Findings", "REPORT", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, err := identity.Extract(context.Background(), "md", strings.NewReader(c.body), int64(len(c.body)), identity.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			draft := intake.NewDraft(draftCatalog(t), "unrelated-filename.md", text, intake.Harvest("unrelated-filename.md", text), nil)
			got := map[string]string{}
			for _, d := range draft.Decisions() {
				got[d.Field] = d.Value
			}
			for field, want := range map[string]string{"title": c.title, "number": c.number, "revision": c.revision, "date": c.date} {
				if got[field] != want {
					t.Errorf("%s=%q want %q", field, got[field], want)
				}
			}
			if c.name == "email" && got["kind"] != "correspondence" {
				t.Errorf("kind=%q", got["kind"])
			}
			if c.name == "programme" && got["kind"] != "schedule" {
				t.Errorf("programme kind=%q", got["kind"])
			}
		})
	}
}

func TestMarkdownIdentityKeepsUserCorrectionsAndLiteralSpans(t *testing.T) {
	body := "# Source report\n**Report No:** ZZ-2026-001\n**Version:** v0.2 (draft)\n**Date:** 2 June 2026"
	text, err := identity.Extract(context.Background(), "markdown", strings.NewReader(body), int64(len(body)), identity.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	candidates := intake.Harvest("unrelated.md", text)
	for _, c := range candidates {
		p := c.Provenance
		if text.Runs[p.Run].Text[p.Start:p.End] != c.Display {
			t.Fatalf("nonliteral candidate: %+v", c)
		}
	}
	users := map[string]intake.Decision{"title": {Field: "title", Value: "Owner title", Band: "green", DecidedBy: "user"}, "number": {Field: "number", Value: "", Band: "green", DecidedBy: "user"}}
	draft := intake.NewDraft(draftCatalog(t), "unrelated.md", text, candidates, users)
	for _, d := range draft.Decisions() {
		if want, ok := users[d.Field]; ok && (d.Value != want.Value || d.DecidedBy != "user") {
			t.Fatalf("user correction overwritten: %+v", d)
		}
	}
}
