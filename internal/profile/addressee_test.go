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
