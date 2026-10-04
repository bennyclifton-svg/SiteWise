package knowledge_test

import (
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

// The repository's works layer loads, and loading it adds no Jev question:
// signals are data until D-12 decides how they are asked.
func TestWorksLayerLoadsWithoutChangingEvidenceQuestions(t *testing.T) {
	cat := loadRepo(t)
	if got := len(cat.Actions()); got != 8 {
		t.Fatalf("actions %d", got)
	}
	if len(cat.InterfaceConsequences()) == 0 || len(cat.Consequences()) == 0 || len(cat.UnforeseenConditions()) == 0 {
		t.Fatalf("ic %d cq %d uc %d", len(cat.InterfaceConsequences()), len(cat.Consequences()), len(cat.UnforeseenConditions()))
	}
	if _, ok := cat.Signal("sig.existing-fire-water-test-results-stated"); !ok {
		t.Fatal("works signal missing")
	}
	if _, ok := cat.Signal("sig.fire-water-authority-capacity-advice-stated"); !ok {
		t.Fatal("cluster signal missing")
	}
	var all []string
	for _, s := range cat.TopSystems() {
		all = append(all, s.ID)
		for _, ch := range cat.Children(s.ID) {
			all = append(all, ch.ID)
		}
	}
	for _, q := range cat.EvidenceQuestions(all) {
		if strings.HasPrefix(q.ID, "sig:") {
			t.Fatalf("signal %s leaked into the evidence questions", q.ID)
		}
	}
}

func TestDefaultActionFollowsWorkType(t *testing.T) {
	cat := loadRepo(t)
	for wt, want := range map[string]string{"new": "new", "extend": "new", "refurb": "alter", "remediation": "repair", "advisory": "investigate", "unknown": ""} {
		if got := cat.DefaultAction(wt); got != want {
			t.Errorf("%s: %q, want %q", wt, got, want)
		}
	}
	if got := len(cat.ExistingConditions()); got != 7 {
		t.Fatalf("existing conditions %d", got)
	}
}

func uc(t *testing.T, cat *knowledge.Catalog, id string) knowledge.Unforeseen {
	t.Helper()
	for _, u := range cat.UnforeseenConditions() {
		if u.ID == id {
			return u
		}
	}
	t.Fatalf("%s not loaded", id)
	return knowledge.Unforeseen{}
}

// uc.existing-fire-water-fails-flow-test: new, upgrade or alter of sprinklers
// or hydrants in an existing building.
func TestWorksPredicateOnARealRecord(t *testing.T) {
	cat := loadRepo(t)
	when := uc(t, cat, "uc.existing-fire-water-fails-flow-test").When
	existing := map[string]string{"existing_building": "stated_true"}
	cases := []struct {
		name   string
		items  []knowledge.WorkItem
		values map[string]string
		want   knowledge.Truth
	}{
		{"upgrade sprinklers, existing building", []knowledge.WorkItem{{System: "fire-active.sprinklers", Action: "upgrade"}}, existing, knowledge.True},
		{"replace is not in the list", []knowledge.WorkItem{{System: "fire-active.sprinklers", Action: "replace"}}, existing, knowledge.False},
		{"no works at all", nil, existing, knowledge.False},
		{"another system", []knowledge.WorkItem{{System: "electrical", Action: "upgrade"}}, existing, knowledge.False},
		{"building age unknown keeps it", []knowledge.WorkItem{{System: "fire-active.hydrants", Action: "alter"}}, nil, knowledge.Unknown},
		{"new building", []knowledge.WorkItem{{System: "fire-active.sprinklers", Action: "new"}}, map[string]string{"existing_building": "stated_false"}, knowledge.False},
		{"coarse item on the whole family might be sprinklers", []knowledge.WorkItem{{System: "fire-active", Action: "upgrade"}}, existing, knowledge.Unknown},
	}
	for _, c := range cases {
		if got := cat.Holds(when, knowledge.WorksEnv{Values: c.values, Items: c.items}); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestSystemExistingAndPresent(t *testing.T) {
	cat := loadRepo(t)
	existing := map[string]any{"system_existing": "fire-active.fire-water"}
	if got := cat.Holds(existing, knowledge.WorksEnv{}); got != knowledge.Unknown {
		t.Fatalf("no existence input: %s", got)
	}
	yes := func(string) knowledge.Truth { return knowledge.True }
	if got := cat.Holds(existing, knowledge.WorksEnv{Existing: yes}); got != knowledge.True {
		t.Fatalf("existing: %s", got)
	}
	present := map[string]any{"system_present": "fire-active.sprinklers"}
	if got := cat.Holds(present, knowledge.WorksEnv{Present: func(s string) bool { return s == "fire-active.sprinklers" }}); got != knowledge.True {
		t.Fatalf("system_present keeps its meaning: %s", got)
	}
	if got := cat.Holds(present, knowledge.WorksEnv{}); got != knowledge.False {
		t.Fatalf("system_present without systems: %s", got)
	}
}

// A works predicate in a rule's applies_when does not hide the rule from
// scope relevance, where no work items are known.
func TestRelevanceTreatsWorksAsUnknown(t *testing.T) {
	root := writeFixture(t)
	mustWrite(t, filepath.Join(root, "clusters", "fire", "rules.yaml"), `
version: 1
rules:
  - id: rule.demo.works
    status: draft
    systems: [fire-passive]
    applies_when:
      works: {action: [alter]}
`)
	cat, err := knowledge.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	r := cat.Relevant([]string{"fire-passive"}, nil)
	if len(r.Rules) != 1 || r.Rules[0] != "rule.demo.works" {
		t.Fatalf("rules %v", r.Rules)
	}
}

func TestLoaderRejectsUnknownWorksReferences(t *testing.T) {
	actions := `
version: 1
actions:
  - {id: new, describes: d, excludes: e}
  - {id: alter, describes: d, excludes: e}
work_type_defaults: {new: new}
`
	for name, files := range map[string]map[string]string{
		"interface consequence names an unknown action": {
			"works/interface_consequences.yaml": "version: 1\ninterface_consequences:\n  - {id: ic.x, type: loads, touches: from, actions: [demolish], propose: {kind: investigation, label: x}}\n",
		},
		"unforeseen names an unknown system": {
			"clusters/fire/unforeseen.yaml": "version: 1\nunforeseen:\n  - id: uc.x\n    when: {works: {action: [alter], system: [no-such-system]}}\n",
		},
		"consequence cites an unknown signal": {
			"clusters/fire/consequences.yaml": "version: 1\nconsequences:\n  - {id: cq.x, signals: [sig.missing], when: {works: {action: [new]}}}\n",
		},
		"default action is not an action": {
			"works/actions.yaml": "version: 1\nactions:\n  - {id: new, describes: d, excludes: e}\nwork_type_defaults: {refurb: alter}\n",
		},
	} {
		root := writeFixture(t)
		mustWrite(t, filepath.Join(root, "works", "actions.yaml"), actions)
		var bad string
		for rel, body := range files {
			mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), body)
			bad = filepath.Base(rel)
		}
		_, err := knowledge.Load(root)
		if err == nil || !strings.Contains(err.Error(), bad) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
