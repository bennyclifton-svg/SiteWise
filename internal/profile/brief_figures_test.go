package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

// The owner brief's contingency, owner-supplied items and budget exclusions
// become candidates Jev picks from (docs/plans/2026-10-21-source-to-cost-plan-design.md).
func TestBriefFigureCandidates(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		text, key string
		norms     []string
	}{
		{"Owners hold **$120,000 contingency** (approx. 6.5%) for unknowns on sloping/reactive site.", "proj.contingency_stated", []string{"$120,000"}},
		{"Owners will supply: pendant lights (allowance $8,000), appliances package (allowance $22,000).", "proj.owner_supply_stated", []string{"pendant lights (allowance $8,000), appliances package (allowance $22,000)"}},
		{"**Working budget ceiling:** $1,850,000 construction cost (excluding landscaping, loose furniture, and authority fees).", "proj.exclusions_stated", []string{"landscaping, loose furniture, and authority fees"}},
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
				t.Errorf("%s missing %q; got %q", tc.key, want, got)
			}
		}
	}
}
