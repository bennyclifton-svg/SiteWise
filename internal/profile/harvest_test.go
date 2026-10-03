package profile_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

func repoCatalog(t testing.TB) *knowledge.Catalog {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	cat, err := knowledge.Load(filepath.Join(filepath.Dir(file), "..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestHarvest(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		line      string
		triggered []string // must be present
		absent    []string // must not be present
		key       string   // candidate key to check
		norms     []string // expected Norm values for key, in order
	}{
		{line: "We have allowed for Bush Fire Attack Level Low to all sites.", triggered: []string{"det.bal"}},
		{line: "Thermally Broken windows/doors, with BAL 40 compliance", triggered: []string{"det.bal"}},
		{line: "The size of the fire compartments must remain under 2,000sqm so that Type C construction can be achieved with a FRL of 90/90/90.",
			triggered: []string{"det.floor_area"}, key: "det.floor_area", norms: []string{"2000"}},
		{line: "Construction of a 5 storey development consisting of a residential flat building",
			triggered: []string{"hdr.scale.storeys"}, absent: []string{"det.rise_in_storeys"}, key: "hdr.scale.storeys", norms: []string{"5"}},
		{line: "Development consent number: DA201500704, and Section 96: DA201500704.01",
			triggered: []string{"fact.consent_number"}, key: "fact.consent_number", norms: []string{"DA201500704", "DA201500704.01"}},
		{line: "Consent will lapse: 17.10.2023", triggered: []string{"fact.consent_lapse_date"},
			key: "fact.consent_lapse_date", norms: []string{"2023-10-17"}},
		{line: "Design & Construct Contract AS 4902-2000 (Amendment 1)", triggered: []string{"fact.contract_form"}},
		{line: "Supply and install Aqua Nova Advance Septic Treatment System.", triggered: []string{"det.sewer_connection"}},
		{line: "Two 45 kg LPG gas bottles, gas bottle connections and gas lines", triggered: []string{"det.gas_supply"}},
		{line: "All building areas are measured as Gross Lettable Area (GLA)\nTenancy 01 1,920 m2",
			triggered: []string{"hdr.scale.gla_sqm"}, key: "hdr.scale.gla_sqm", norms: []string{"1920"}},
		{line: "TOTAL GROSS FLOOR 4,013", absent: []string{"hdr.scale.gfa_sqm", "det.floor_area"}},
		{line: "Min. top of storage height TBC", absent: []string{"det.storage_height"}},
		{line: "Building Type: Single storey", triggered: []string{"hdr.scale.storeys"}, key: "hdr.scale.storeys", norms: []string{"1"}},
		{line: "a three-storey brick residential apartment building", triggered: []string{"hdr.scale.storeys"},
			key: "hdr.scale.storeys", norms: []string{"3"}},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.line, cat)
		for _, k := range tc.triggered {
			if !h.Has(k) {
				t.Errorf("%q: %s not triggered (got %v)", tc.line, k, h.Triggered)
			}
		}
		for _, k := range tc.absent {
			if h.Has(k) {
				t.Errorf("%q: %s must not trigger", tc.line, k)
			}
		}
		if tc.key == "" {
			continue
		}
		var got []string
		for _, c := range h.Candidates[tc.key] {
			got = append(got, c.Norm)
			if c.Value == "" || c.ID == "" || !strings.Contains(tc.line, c.Value) {
				t.Errorf("%q: candidate must be verbatim with an id: %+v", tc.line, c)
			}
		}
		if strings.Join(got, "|") != strings.Join(tc.norms, "|") {
			t.Errorf("%q: %s norms = %v, want %v", tc.line, tc.key, got, tc.norms)
		}
	}
}

func TestHarvestBasisAndUnit(t *testing.T) {
	h := profile.Harvest("All building areas are measured as Gross Lettable Area (GLA)\nTenancy 01 1,920 m2", repoCatalog(t))
	c := h.Candidates["hdr.scale.gla_sqm"][0]
	if c.Unit != "m2" || c.Basis != "GLA" {
		t.Fatalf("candidate = %+v", c)
	}
}

func BenchmarkHarvest(b *testing.B) {
	cat := repoCatalog(b)
	text := strings.Repeat("The warehouse floor slab shall be 1,920 m2 with BAL 40 windows and a septic system. ", 24)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile.Harvest(text, cat)
	}
}
