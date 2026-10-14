package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

// A house brief states the budget, bedrooms and bathrooms; each becomes a
// candidate Jev picks from (owner critique of the Custom Home 2 RFP,
// 13 October 2026).
func TestHouseBriefCandidates(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		text, key string
		norms     []string
	}{
		{"The construction budget is $1.8 million excluding GST.", "proj.budget_stated", []string{"$1.8 million excluding GST"}},
		{"Target construction cost: $1,250,000 (incl GST), with a fee allowance of $90,000.", "proj.budget_stated", []string{"$1,250,000 (incl GST)", "$90,000"}},
		{"A new two storey home with 4 bedrooms and 3 bathrooms over a double garage.", "hdr.scale.bathrooms", []string{"3"}},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.text, cat)
		if !h.Has(tc.key) {
			t.Errorf("%q: %s not triggered (got %v)", tc.text, tc.key, h.Triggered)
			continue
		}
		var got []string
		for _, c := range h.Candidates[tc.key] {
			got = append(got, c.Norm)
		}
		for _, want := range tc.norms {
			if !contains(got, want) {
				t.Errorf("%q: %s missing %q; got %q", tc.text, tc.key, want, got)
			}
		}
	}
}

// A passage that only lodges a document elsewhere or names a bath fitting
// offers no house figure.
func TestHouseBriefIgnoresLookalikes(t *testing.T) {
	cat := repoCatalog(t)
	if h := profile.Harvest("Freestanding bath to the main bathroom (1700mm).", cat); len(h.Candidates["hdr.scale.bathrooms"]) > 0 {
		t.Fatalf("a bath size is not a bathroom count: %+v", h.Candidates["hdr.scale.bathrooms"])
	}
	if h := profile.Harvest("The budget will be confirmed at concept stage.", cat); len(h.Candidates["proj.budget_stated"]) > 0 {
		t.Fatalf("no amount, no candidate: %+v", h.Candidates["proj.budget_stated"])
	}
}
