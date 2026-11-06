package profile_test

import (
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

// Wording from the T13 shop-top corpus (synthetic findings F32, F33).
func TestHarvestSyntheticT13(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		line  string
		key   string
		norms []string
	}{
		// F32 / R14: apartment count and shop area are the key metrics.
		{line: "Halcyon Eastwood Pty Ltd will develop a new six-storey mixed-use building: ground-floor retail with\n42 apartments on the five levels above.",
			key: "hdr.scale.residential_units", norms: []string{"42"}},
		{line: "- About 600 m2 of retail GLA in three shell tenancies.",
			key: "hdr.scale.retail_gla_sqm", norms: []string{"600"}},
		{line: "The consolidated site is about 1,800 m2, with a frontage to Halloran Street and a rear lane behind.",
			key: "hdr.scale.site_sqm", norms: []string{"1800"}},
		{line: "basement levels of car parking, on a consolidated site of about 1,800 m2 in the Eastwood town centre.",
			key: "hdr.scale.site_sqm", norms: []string{"1800"}},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.line, cat)
		if !h.Has(tc.key) {
			t.Errorf("%q: %s not triggered (got %v)", tc.line, tc.key, h.Triggered)
			continue
		}
		var got []string
		for _, c := range h.Candidates[tc.key] {
			got = append(got, c.Norm)
		}
		if strings.Join(got, "|") != strings.Join(tc.norms, "|") {
			t.Errorf("%q: %s norms = %v, want %v", tc.line, tc.key, got, tc.norms)
		}
	}
	// Podium and tower levels stay unasked (R14).
	if h := profile.Harvest("The ground floor is a one-level retail podium; the five residential levels sit above it.", cat); h.Has("hdr.scale.podium_levels") || h.Has("hdr.scale.tower_levels") {
		t.Errorf("podium or tower levels asked: %v", h.Triggered)
	}
	// A site frontage in metres is not a site area.
	if h := profile.Harvest("The site has a frontage of 40.2 m to Halloran Street.", cat); h.Has("hdr.scale.site_sqm") {
		t.Error("site frontage triggered site_sqm")
	}
}

// F33: a PDF line wrap inside a multi-word trigger must not hide it.
func TestTriggerAcrossLineWrap(t *testing.T) {
	cat := repoCatalog(t)
	h := profile.Harvest("The proposal complies with the height and FSR standards, so no clause\n4.6 variation request is required.", cat)
	if !h.Has(knowledge.DetailPrefix + "flag.cl46") {
		t.Fatalf("flag.cl46 not triggered across a line wrap (got %v)", h.Triggered)
	}
}

// F34: a stated low or compliant noise level denies a noise-affected site.
func TestNoiseCriteriaDenyLowLevel(t *testing.T) {
	d, ok := repoCatalog(t).Determinant("noise_affected_site")
	if !ok {
		t.Fatal("noise_affected_site missing")
	}
	criteria, _ := d.Question.Criteria.(map[string]any)
	deny := strings.ToLower(fmt.Sprint(criteria["stated_false"]))
	if !strings.Contains(deny, "low") || !strings.Contains(deny, "compl") {
		t.Fatalf("stated_false does not cover a low or compliant level: %q", deny)
	}
}

// F36: an existing basement under other works is not basement works.
func TestBasementFlagAsksAboutTheWorks(t *testing.T) {
	f, ok := repoCatalog(t).FlagReading("basement")
	if !ok {
		t.Fatal("basement flag missing")
	}
	asks := strings.ToLower(f.Asks)
	if !strings.Contains(asks, "works") || !strings.Contains(asks, "existing basement") {
		t.Fatalf("basement question does not separate the works from an existing basement: %q", f.Asks)
	}
}
