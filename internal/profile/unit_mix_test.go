package profile_test

import (
	"strings"
	"testing"

	"sitewise/internal/profile"
)

// A unit mix count is only a number written against a type phrase; a type
// phrase on its own (a house's "4 bedroom home") gives nothing to ask.
func TestHarvestUnitMix(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		line string
		want map[string][]string // key -> candidate norms, in order; nil means not triggered
	}{
		{"Our current target mix is 14 one-bedroom, 38 two-bedroom and 12 three-bedroom apartments.",
			map[string][]string{"units_1_bed": {"14"}, "units_2_bed": {"38"}, "units_3_bed": {"12"}, "units_studio": nil}},
		{"Apartment mix: 4 x 1 bed, 4 x 2 bed, 2 x 3 bed",
			map[string][]string{"units_1_bed": {"4"}, "units_2_bed": {"4"}, "units_3_bed": {"2"}}},
		{"Studio: 6\n1 Bed | 12\n2 Bed | 20 | 76 m2\n4 Bed: 2",
			map[string][]string{"units_studio": {"6"}, "units_1_bed": {"12"}, "units_2_bed": {"20"}, "units_4_bed": {"2"}}},
		{"6 × 2B and 3 × 3B units over the podium",
			map[string][]string{"units_2_bed": {"6"}, "units_3_bed": {"3"}}},
		{"comprising six two-bedroom apartments and two-bedroom apartments (38) in Building B",
			map[string][]string{"units_2_bed": {"6", "38"}}},
		{"A new 4 bedroom home with a double garage.", map[string][]string{"units_4_bed": nil}},
		{"64 apartments: a mix of one-, two- and three-bedroom apartments.",
			map[string][]string{"units_1_bed": nil, "units_2_bed": nil, "units_3_bed": nil}},
		{"Target 40% two-bedroom and 25 % three-bedroom.", map[string][]string{"units_2_bed": nil, "units_3_bed": nil}},
		{"2 Bed | 76 m2 | 1.5 baths", map[string][]string{"units_2_bed": nil}},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.line, cat)
		for key, norms := range tc.want {
			full := "hdr.scale." + key
			if norms == nil {
				if h.Has(full) {
					t.Errorf("%q: %s must not be asked (candidates %v)", tc.line, key, h.Candidates[full])
				}
				continue
			}
			var got, vals []string
			for _, c := range h.Candidates[full] {
				got = append(got, c.Norm)
				vals = append(vals, c.Value)
				if c.ID == "" || !strings.Contains(tc.line, c.Value) {
					t.Errorf("%q: candidate must be verbatim with an id: %+v", tc.line, c)
				}
			}
			if !h.Has(full) || strings.Join(got, "|") != strings.Join(norms, "|") {
				t.Errorf("%q: %s norms = %v (values %v), want %v", tc.line, key, got, vals, norms)
			}
		}
	}
}
