package profile_test

import (
	"sitewise/internal/profile"
	"testing"
)

func TestResidentialProfileAreaCandidates(t *testing.T) {
	cat := repoCatalog(t)
	for _, tc := range []struct{ text, key, want string }{
		{"Lot area **651 m²** (registered).", "hdr.scale.site_sqm", "651"},
		{"Lot size: 651 m²", "hdr.scale.site_sqm", "651"},
		{"Proposed dwelling, 285 m² GFA (indicative).", "hdr.scale.gfa_sqm", "285"},
	} {
		h := profile.Harvest(tc.text, cat)
		cs := h.Candidates[tc.key]
		if len(cs) != 1 || cs[0].Norm != tc.want {
			t.Errorf("%q: %v", tc.text, cs)
		}
	}
}

func TestHyphenatedCouncilCandidate(t *testing.T) {
	h := profile.Harvest("Pursue DA + CC pathway via Ku-ring-gai Council.", repoCatalog(t))
	cs := h.Candidates["proj.lga"]
	if len(cs) != 1 || cs[0].Norm != "Ku-ring-gai" {
		t.Fatalf("council candidates: %v", cs)
	}
}

func TestProgrammeTargetsKeepQualificationsAndIgnoreServiceDates(t *testing.T) {
	cat := repoCatalog(t)
	text := "Stage-level programme aligned to target DA lodgement September 2026 and owner aspiration Christmas 2028 occupation. Subject to approval."
	h := profile.Harvest(text, cat)
	cs := h.Candidates["proj.program"]
	if len(cs) != 1 || cs[0].Norm != text {
		t.Fatalf("lost target qualification: %v", cs)
	}
	for _, kind := range []string{"schedule", "commercial"} {
		qs, _ := profile.LabelQuestions(profile.PassageInfo{Kind: kind, Text: text}, h, cat)
		_, asked := qs["proj.program"]
		if asked != (kind == "schedule") {
			t.Fatalf("programme routed from %s", kind)
		}
	}
	if cs := profile.Harvest("Report date: 28 May 2026", cat).Candidates["proj.program"]; len(cs) != 0 {
		t.Fatal(cs)
	}
	if cs := profile.Harvest("| Activity | Target |\n| DA | September 2026 |", cat).Candidates["proj.program"]; len(cs) != 0 {
		t.Fatal("an activity table must not compete with the programme summary", cs)
	}
}
