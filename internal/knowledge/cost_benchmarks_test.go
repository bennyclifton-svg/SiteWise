package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

const benchmarkYAML = `version: 1
benchmarks:
 - id: bm.fixture
   version: 1
   basis: rate
   unit: m2
   amount: "12.3456"
   currency: AUD
   tax_basis: ex_tax
   price_date: "2026-10-01"
   geography: NSW Sydney metro
   quality: standard
   inclusions: [installation]
   exclusions: [demolition]
   applies_when: {det: climate_zone, eq: "5"}
   status: reviewed
   sources: [{design: test-fixture-only}]
`

func TestCostBenchmarksRequireReviewedExactBasisAndKnownApplicability(t *testing.T) {
	root := t.TempDir()
	if e := os.MkdirAll(filepath.Join(root, "costs"), 0755); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(root, "costs", "benchmarks.yaml")
	if e := os.WriteFile(path, []byte(benchmarkYAML), 0600); e != nil {
		t.Fatal(e)
	}
	c := &Catalog{root: root, loaded: map[string][]byte{}, profile: profileData{determinants: []Determinant{{ID: "climate_zone"}}}}
	if e := c.loadCostBenchmarks(root); e != nil {
		t.Fatal(e)
	}
	b := c.CostBenchmarks()[0]
	env := WorksEnv{Values: map[string]string{"climate_zone": "5"}}
	if !c.EligibleCostBenchmark(b, env, "AUD", "ex_tax", "2026-10-01", "NSW Sydney metro", "standard") {
		t.Fatal("valid basis rejected")
	}
	tests := []struct {
		date, geo, quality, currency, tax string
		env                               WorksEnv
	}{{"2026-10-02", "NSW Sydney metro", "standard", "AUD", "ex_tax", env}, {"2026-10-01", "Melbourne", "standard", "AUD", "ex_tax", env}, {"2026-10-01", "NSW Sydney metro", "premium", "AUD", "ex_tax", env}, {"2026-10-01", "NSW Sydney metro", "standard", "USD", "ex_tax", env}, {"2026-10-01", "NSW Sydney metro", "standard", "AUD", "inc_tax", env}, {"2026-10-01", "NSW Sydney metro", "standard", "AUD", "ex_tax", WorksEnv{}}}
	for _, test := range tests {
		if c.EligibleCostBenchmark(b, test.env, test.currency, test.tax, test.date, test.geo, test.quality) {
			t.Fatal("mismatched or unknown basis accepted", test)
		}
	}
	b.Status = "draft"
	c.costBenchmarks[b.ID] = b
	if len(c.CostBenchmarks()) != 0 || c.EligibleCostBenchmark(b, env, "AUD", "ex_tax", "2026-10-01", "NSW Sydney metro", "standard") {
		t.Fatal("draft offered")
	}
	if len(c.loaded) != 1 {
		t.Fatal("benchmark omitted from knowledge hash")
	}
}
func TestCostBenchmarkRejectsUnknownPredicateReferences(t *testing.T) {
	c := &Catalog{}
	for _, predicate := range []any{map[string]any{"made_up": true}, map[string]any{"system_present": "missing"}, map[string]any{"det": "missing", "eq": "yes"}, map[string]any{"works": map[string]any{"action": []any{"invented"}}}, map[string]any{"any": "not a list"}} {
		if c.validateCostPredicate(predicate, 0) == nil {
			t.Fatal(predicate)
		}
	}
}
