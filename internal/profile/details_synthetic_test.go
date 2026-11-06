package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

// Phrases from the synthetic corpus runs of 4 November 2026
// (testing/synthetic/FINDINGS.md F2, F16 and the proj.client note).

func candidateNorms(t *testing.T, text, key string) []string {
	t.Helper()
	var got []string
	for _, c := range profile.Harvest(text, repoCatalog(t)).Candidates[key] {
		got = append(got, c.Norm)
	}
	return got
}

func sameNorms(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// F2: an s10.7 certificate names the council, then a plan title or a
// signature block. The name stops at the council.
func TestLGAStopsAtTheCouncilName(t *testing.T) {
	for text, want := range map[string]string{
		"   City of Canada Bay Section 7.11 Contributions Plan applies to the land.":             "City of Canada Bay",
		"   City of Ryde Section 7.11 Development Contributions Plan 2020 applies to the land.":  "City of Ryde",
		"For the General Manager, City of Canada Bay\nDuty planner, Customer Service Centre":     "City of Canada Bay",
		"For the General Manager, City of Ryde\nDuty planner, Customer Service Centre":           "City of Ryde",
		"For the General Manager, Georges River Council\nDuty planner, Customer Service Centre":  "Georges River",
		"This statement accompanies a development application to the City of Ryde. It describes": "City of Ryde",
		"Local government area: City of Canada Bay.":                                             "City of Canada Bay",
	} {
		if got := candidateNorms(t, text, "proj.lga"); !sameNorms(got, []string{want}) {
			t.Errorf("%q: lga = %q, want %q", text, got, want)
		}
	}
}

// F2 (address): a sentence end or the next line's label is not a suburb.
func TestAddressStopsAtSentenceEndAndLabels(t *testing.T) {
	for text, want := range map[string]string{
		"Title: Owner's brief: alterations and additions at 41 Tallowwood Avenue\nDate: 9 February 2026": "41 Tallowwood Avenue",
		"alterations and additions to our family house at 41 Tallowwood Avenue. We would like":           "41 Tallowwood Avenue",
		"for the house at 41 Tallowwood Avenue. It provides an AS 2870 site classification":              "41 Tallowwood Avenue",
		"Site: 41 Tallowwood Avenue, Concord NSW 2137":                                                   "41 Tallowwood Avenue, Concord NSW 2137",
		"Hazardous materials survey, 41 Tallowwood Avenue, Concord\n":                                    "41 Tallowwood Avenue, Concord",
	} {
		if got := candidateNorms(t, text, "site.address"); !sameNorms(got, []string{want}) {
			t.Errorf("%q: address = %q, want %q", text, got, want)
		}
	}
}

// Minor note: a care-of party is not part of the client's name.
func TestClientStopsBeforeCareOf(t *testing.T) {
	for _, text := range []string{
		"To: The Owners - Strata Plan No. 912345, c/- Rhiannon Teale, Northgate Strata Management\nProject: Balcony waterproofing",
		"**To:** The Owners - Strata Plan No. 912345 C/- Northgate Strata Management",
		"To: The Owners - Strata Plan No. 912345, care of Northgate Strata Management",
	} {
		got := candidateNorms(t, text, "proj.client")
		if !sameNorms(got, []string{"The Owners - Strata Plan No. 912345"}) {
			t.Errorf("%q: client = %q", text, got)
		}
	}
}

// F16: a defect report states the rise in storeys; the value is read
// wherever the phrase states a number, and nowhere else.
func TestRiseInStoreysAndEffectiveHeightAreReadWhereStated(t *testing.T) {
	cases := []struct {
		text, key string
		want      []string
	}{
		{"- The building has a rise in storeys of 6", "det.rise_in_storeys", []string{"6"}},
		{"Class 2 with a Class 7a basement car park, rise in storeys 6, 42 apartments", "det.rise_in_storeys", []string{"6"}},
		{"its rise in storeys is 6. The building is fully occupied", "det.rise_in_storeys", []string{"6"}},
		{"the multi-storey building exemption applies (rise in storeys of 6).", "det.rise_in_storeys", []string{"6"}},
		// The phrase alone, or other numbers nearby, offer nothing.
		{"Height of building / rise in storeys (i.e. measure of likely evacuation time)", "det.rise_in_storeys", nil},
		{"Built in 1986 with 42 apartments; the rise in storeys is not recorded.", "det.rise_in_storeys", nil},
		{"The building has an effective height of 16.2 m.", "det.effective_height", []string{"16.2"}},
		{"Effective height: 9 m", "det.effective_height", []string{"9"}},
		// A bound is not the building's height.
		{"buildings which are less than 25 m in effective height", "det.effective_height", nil},
		{"C1.1 Effective height < 25 m", "det.effective_height", nil},
	}
	for _, tc := range cases {
		if got := candidateNorms(t, tc.text, tc.key); !sameNorms(got, tc.want) {
			t.Errorf("%q: %s = %q, want %q", tc.text, tc.key, got, tc.want)
		}
	}
}

// F21: a suburb whose second word is a compass point ("Caringbah South"),
// or any two-word suburb followed by the state, keeps both words.
func TestAddressKeepsTwoWordSuburbEndingInADirection(t *testing.T) {
	for text, want := range map[string]string{
		"Site: 22 Marrakai Close, Caringbah South NSW 2229":    "22 Marrakai Close, Caringbah South NSW 2229",
		"Site: 22 Marrakai Close, Caringbah South\nDate: 2026": "22 Marrakai Close, Caringbah South",
		"22 MARRAKAI CLOSE, CARINGBAH SOUTH NSW 2229":          "22 Marrakai Close, Caringbah South NSW 2229",
		"Site: 9 Avon Street, Glebe Point NSW 2037":            "9 Avon Street, Glebe Point NSW 2037",
	} {
		if got := candidateNorms(t, text, "site.address"); !sameNorms(got, []string{want}) {
			t.Errorf("%q: address = %q, want %q", text, got, want)
		}
	}
}

// F22: a letter sent to an agent "for" or "on behalf of" the client offers
// the client, not the agent's whole line.
func TestClientIsThePartyAnAgentActsFor(t *testing.T) {
	for _, text := range []string{
		"To: Ines Hart, Marlowe Hart Architects, for Hamish and Rosa Delaney\nProject: New house",
		"To: Ines Hart, Marlowe Hart Architects, on behalf of Hamish and Rosa Delaney\nProject: New house",
		"**To:** Ines Hart, Marlowe Hart Architects; for Hamish and Rosa Delaney",
	} {
		got := candidateNorms(t, text, "proj.client")
		if !sameNorms(got, []string{"Hamish and Rosa Delaney"}) {
			t.Errorf("%q: client = %q", text, got)
		}
	}
}

// F48 (run 20261107-t03-t04-t06): a planning certificate states the area
// after a label, under a council heading in capitals.
func TestLGAFromLabelAndCapitalHeading(t *testing.T) {
	for text, want := range map[string]string{
		"| Legal description | Lot 12 DP 1184520 |\n\nLocal government area: North Sydney.\n": "North Sydney",
		"LGA: Inner West Council\n":                                          "Inner West",
		"NORTH SYDNEY COUNCIL\n\n# Planning Certificate\n":                   "North Sydney",
		"# GEORGES RIVER COUNCIL\nPlanning certificate under section 10.7\n": "Georges River",
	} {
		if got := candidateNorms(t, text, "proj.lga"); !sameNorms(got, []string{want}) {
			t.Errorf("%q: lga = %q, want %q", text, got, want)
		}
	}
	// Capitals inside running text are not a heading.
	if got := candidateNorms(t, "WORKS MUST BE APPROVED BY COUNCIL BEFORE START", "proj.lga"); len(got) != 0 {
		t.Errorf("capital sentence offered as lga: %q", got)
	}
}
