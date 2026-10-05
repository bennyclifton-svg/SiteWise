package knowledge_test

import (
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func TestPlanningKeysRefuseMoneyTotals(t *testing.T) {
	for _, keys := range []string{
		`[{key: cost_total, label: Total, value: number, scope: project}]`,
		`[{key: budget, label: Budget, value: money, scope: project}]`,
		`[{key: gfa, label: GFA, value: number, scope: site, favourable: [big]}]`,
		`[{key: ok, label: OK, value: boolean, scope: site, favourable: ["true"], starting_value: "true"}]`,
	} {
		root := profileFixture(t, `'x'`)
		mustWrite(t, filepath.Join(root, "profile", "planning_keys.yaml"), "version: 1\nstatus: draft\nkeys: "+keys+"\n")
		if _, err := knowledge.Load(root); err == nil || !strings.Contains(err.Error(), "planning key") {
			t.Fatalf("%s: err = %v", keys, err)
		}
	}
}

// Starting values are offered only from a reviewed registry (NW-REQ-185).
func TestStartingValuesNeedAReviewedRegistry(t *testing.T) {
	for status, want := range map[string]int{"draft": 0, "reviewed": 1} {
		root := profileFixture(t, `'x'`)
		mustWrite(t, filepath.Join(root, "profile", "planning_keys.yaml"), "version: 1\nstatus: "+status+`
keys:
  - {key: duration, label: Duration, value: number, unit: weeks, scope: project, starting_value: "40"}
  - {key: ground, label: Ground, value: choice, scope: site, options: [{id: good}, {id: poor}], favourable: [good]}
`)
		cat, err := knowledge.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(cat.StartingValues()); got != want {
			t.Fatalf("%s: %d starting values", status, got)
		}
		if cat.KeyScope("plan.ground") != "site" || cat.KeyScope("plan.duration") != "project" || cat.KeyScope("plan.other") != "" {
			t.Fatal("planning key scope")
		}
	}
}

func TestRealPlanningKeys(t *testing.T) {
	cat := loadRepo(t)
	if len(cat.StartingValues()) != 0 {
		t.Fatal("a draft registry proposes nothing")
	}
	// L276: structure, ground and compliance are never assumed favourable.
	for key, value := range map[string]string{
		"existing_structure_adequate": "true",
		"site_classification":         "A",
		"contamination_present":       "false",
		"existing_compliance":         "complies",
	} {
		k, ok := cat.PlanningKey(key)
		if !ok || !k.IsFavourable(value) {
			t.Errorf("%s: %q must be a forbidden favourable default", key, value)
		}
	}
	for _, k := range cat.PlanningKeys() {
		if strings.HasPrefix(k.Key, "cost") {
			t.Errorf("money total %s", k.Key)
		}
	}
}
