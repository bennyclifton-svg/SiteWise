package jobs_test

import (
	"os"
	"strings"
	"testing"

	"sitewise/internal/jobs"
)

// Kavanagh, 24 October 2026: the consultant's letterhead and the "To /
// Project / Site" block were one passage, so the state was read from the
// consultant's office address. The addressee block is its own passage.
func TestAddresseeBlockIsItsOwnPassage(t *testing.T) {
	body, err := os.ReadFile("../proposals/testdata/kavanagh/01-fee-proposal-quoin-architecture.md")
	if err != nil {
		t.Fatal(err)
	}
	got := jobs.SplitPassages(string(body))
	var site, letterhead string
	for _, p := range got {
		if strings.Contains(p, "Paperbark Lane") {
			site = p
		}
		if strings.Contains(p, "Hickson Road") {
			letterhead = p
		}
	}
	if site == "" || site == letterhead || !strings.HasPrefix(site, "**To:**") || !strings.Contains(site, "**Project:**") {
		t.Fatalf("site block %q\nletterhead %q", site, letterhead)
	}
	if strings.Contains(letterhead, "Kavanagh Property Holdings") {
		t.Fatalf("addressee stayed with the letterhead: %q", letterhead)
	}
}

func TestPlainParagraphsAreNotSplitAtLabels(t *testing.T) {
	got := jobs.SplitPassages("Project: Harbour House\nSite: 1 Main Street, Sydney NSW 2000\nClient: Harbour Pty Ltd")
	if len(got) != 1 {
		t.Fatalf("a block that starts with labels split: %q", got)
	}
}

func TestBoilerplatePassagesAreNotRead(t *testing.T) {
	for _, tc := range []struct {
		p    jobs.Passage
		skip bool
	}{
		{jobs.Passage{Text: "**Accepted for client:** ____________________  **Date:** ____________________"}, true},
		{jobs.Passage{Kind: "commercial", Text: "## Billing\nStage 1 is billed on completion. Invoices are due 14 days from invoice date."}, true},
		{jobs.Passage{Kind: "contract", Text: "## Payment\nProgress claims monthly; retention 5% until practical completion."}, false},
		{jobs.Passage{Kind: "commercial", Text: "## Scope\nArchitectural services for a new two-storey dwelling and pool interface."}, false},
	} {
		if got := jobs.Boilerplate(tc.p); got != tc.skip {
			t.Errorf("%q: boilerplate %v, want %v", tc.p.Text, got, tc.skip)
		}
	}
}
