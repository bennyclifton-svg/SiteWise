package profile_test

import (
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

// Kavanagh, 24 October 2026: "To: Kavanagh Property Holdings Pty Ltd as
// trustee for the Kavanagh Family Trust" was never offered as the client.
func TestAddresseeLineIsAClientCandidate(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	passage := "**To:** Kavanagh Property Holdings Pty Ltd as trustee for the Kavanagh Family Trust\n**Project:** Kavanagh Residence — new detached dwelling and external works\n**Site:** 8 Paperbark Lane, Northbridge NSW 2063"
	h := profile.Harvest(passage, cat)
	got := map[string]bool{}
	for _, c := range h.Candidates["proj.client"] {
		got[c.Value] = true
	}
	for _, want := range []string{"Kavanagh Property Holdings Pty Ltd as trustee for the Kavanagh Family Trust", "Kavanagh Property Holdings Pty Ltd"} {
		if !got[want] {
			t.Errorf("missing client candidate %q in %v", want, h.Candidates["proj.client"])
		}
	}
	if email := profile.Harvest("To: pm@example.com\nSubject: Fees", cat); len(email.Candidates["proj.client"]) != 0 {
		t.Errorf("email address offered as client: %v", email.Candidates["proj.client"])
	}
}

// F46 (run 20261107-t03-t04-t06): "To: Name, Role, Company" names the
// company as the addressee; the person and role are not the client.
func TestAddresseeNameRoleCompanyOffersTheCompany(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{
		"To: Megan Holt, Asset Manager, Corvina Property Holdings Pty Ltd\nProject: Level 3": {"Corvina Property Holdings Pty Ltd"},
		"To: Daniel Okafor, Larchmont Project Services Pty Ltd\n":                            {"Larchmont Project Services Pty Ltd"},
		"To: Building Manager, The Owners - Strata Plan No. 912345\n":                        {"The Owners - Strata Plan No. 912345"},
		"To: Asha Rao, Planning, Northside Council\n":                                        {"Northside Council"},
		"To: Lee Tan, Director, Tan Holdings Pty Ltd as trustee for the Tan Family Trust\n": {
			"Tan Holdings Pty Ltd as trustee for the Tan Family Trust", "Tan Holdings Pty Ltd"},
		// Unchanged: the last part is not a company, so the line stays whole.
		"To: Avenor Capital Partners Pty Ltd, attention Priya Nandakumar\n": {"Avenor Capital Partners Pty Ltd, attention Priya Nandakumar"},
		// Unchanged: the party acted for, and a care-of party, still cut first.
		"To: Daniel Okafor, Larchmont Project Services Pty Ltd, for Avenor Capital Partners Pty Ltd\n": {"Avenor Capital Partners Pty Ltd"},
		"To: Avenor Capital Partners Pty Ltd, care of Larchmont Project Services Pty Ltd\n":            {"Avenor Capital Partners Pty Ltd"},
	}
	for text, want := range cases {
		var got []string
		for _, c := range profile.Harvest(text, cat).Candidates["proj.client"] {
			got = append(got, c.Value)
		}
		if len(got) != len(want) {
			t.Errorf("%q: client = %q, want %q", text, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%q: client = %q, want %q", text, got, want)
				break
			}
		}
	}
}

// Merricks, 11 November 2026: a builder's quote is addressed to the owners
// care of their designer, with no "To:" label, and no client was offered.
func TestCareOfLineOffersTheAddressee(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	passage := "Kind regards,\nSam Okoro\nSeaview Builders\nPriya Nair & Tom Whitlock C/O Harbour Lane Design\nOffice 15, 4 High St\nHastings"
	var got []string
	for _, c := range profile.Harvest(passage, cat).Candidates["proj.client"] {
		got = append(got, c.Value)
	}
	if len(got) != 1 || got[0] != "Priya Nair & Tom Whitlock" {
		t.Fatalf("client candidates %q", got)
	}
	// A "To:" line with a care-of party is still offered once.
	got = nil
	for _, c := range profile.Harvest("To: Avenor Capital Partners Pty Ltd, care of Larchmont Project Services Pty Ltd\n", cat).Candidates["proj.client"] {
		got = append(got, c.Value)
	}
	if len(got) != 1 || got[0] != "Avenor Capital Partners Pty Ltd" {
		t.Fatalf("labelled line %q", got)
	}
}
