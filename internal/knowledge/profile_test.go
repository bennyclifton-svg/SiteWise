package knowledge_test

import (
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func profileFixture(t *testing.T, triggers string) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "systems.yaml"), `
version: 1
systems:
  - {id: envelope, label: Envelope, describes: The skin., status: draft}
  - {id: fire-passive, label: Passive fire, describes: Ratings., status: draft}
`)
	mustWrite(t, filepath.Join(root, "clusters", "a", "systems.yaml"), `
version: 1
systems:
  - {id: envelope.bushfire-construction, parent: envelope, label: Bushfire, describes: AS 3959., status: draft}
  - {id: fire-passive.bushfire-construction, parent: fire-passive, label: Bushfire, describes: AS 3959., status: deprecated, replaced_by: envelope.bushfire-construction}
  - {id: fire-passive.rated-elements, parent: fire-passive, label: Rated, describes: FRLs., status: draft}
`)
	mustWrite(t, filepath.Join(root, "determinants.yaml"), `
version: 1
determinants:
  - id: bal
    label: Bushfire attack level
    value: choice
    extraction: choice
    options: [{id: BAL-LOW, describes: Low.}, {id: BAL-40, describes: Forty.}]
    triggers: [`+triggers+`]
    stated_in: [{kind: report, discipline: consultant.bushfire, label: Bushfire report}]
    profile_group: site
    status: draft
    question: {type: choice, instructions: 'Using text, which BAL?', criteria: {}}
  - id: heritage
    label: Heritage
    value: boolean
    status: deprecated
    replaced_by: heritage_status
  - id: heritage_status
    label: Heritage status
    value: choice
    profile_group: site
    status: draft
  - id: importance_level
    label: Importance level
    value: choice
    status: draft
`)
	// Fixtures opt in to a catch-all scope; the real registry is tested on the repo.
	mustWrite(t, filepath.Join(root, "profile", "key_scope.yaml"), "version: 1\nfamilies:\n  - {match: \"*\", scope: project}\n")
	mustWrite(t, filepath.Join(root, "profile", "taxonomy.yaml"), `
version: 1
status: draft
work_types: [{id: new, label: New build}]
building_classes:
  - id: residential
    label: Residential
    subclasses:
      - id: house
        label: House (Class 1a)
        ncc_class: 1a
        scale_fields:
          - {key: storeys, label: Storeys, type: integer, triggers: ['\bstor(e)?y']}
conditions:
  - {key: planning, label: Planning, options: [{id: da, label: DA}]}
`)
	mustWrite(t, filepath.Join(root, "profile", "scope_defaults.yaml"), `
version: 1
status: draft
always_shown: [bal]
empty_work_types: [refurb]
presets:
  - {id: fit, label: Fit-out, systems: [envelope.bushfire-construction]}
classes:
  - {class: house, work_type: new, systems: [envelope.bushfire-construction]}
categories:
  - {category: residential, systems: [envelope.bushfire-construction]}
`)
	mustWrite(t, filepath.Join(root, "profile", "project_facts.yaml"), `
version: 1
facts:
  - id: contract_basis
    label: Contract pricing basis
    value: choice
    extraction: choice
    options: [{id: cost_plus, describes: Cost plus.}]
    triggers: ['cost plus']
    status: draft
`)
	return root
}

func TestLoadDeterminantsWithTriggers(t *testing.T) {
	cat, err := knowledge.Load(profileFixture(t, `'\bBAL[- ]?40\b'`))
	if err != nil {
		t.Fatal(err)
	}
	bal, ok := cat.Determinant("bal")
	if !ok || bal.ProfileGroup != "site" || len(bal.Options) != 2 || bal.StatedIn[0].Label != "Bushfire report" {
		t.Fatalf("bal = %+v", bal)
	}
	if !bal.Triggered("Windows with bal 40 compliance") || bal.Triggered("BALCONY") {
		t.Fatal("trigger must match case-insensitively and only the pattern")
	}
}

func TestBadTriggerFailsLoadWithID(t *testing.T) {
	_, err := knowledge.Load(profileFixture(t, `'(unclosed'`))
	if err == nil || !strings.Contains(err.Error(), "bal") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeprecatedSystemsAreHiddenFromChoices(t *testing.T) {
	cat, err := knowledge.Load(profileFixture(t, `'x'`))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Children("fire-passive") {
		if c.ID == "fire-passive.bushfire-construction" {
			t.Fatal("deprecated child offered")
		}
	}
	for _, l := range cat.Leaves() {
		if l.Status == "deprecated" {
			t.Fatal("deprecated leaf listed")
		}
	}
	if got := cat.Resolve("fire-passive.bushfire-construction"); got != "envelope.bushfire-construction" {
		t.Fatalf("resolve = %s", got)
	}
	if got := cat.Resolve("heritage"); got != "heritage_status" {
		t.Fatalf("resolve det = %s", got)
	}
}

func TestTaxonomyTypicalAndFacts(t *testing.T) {
	cat, err := knowledge.Load(profileFixture(t, `'x'`))
	if err != nil {
		t.Fatal(err)
	}
	tax := cat.Taxonomy()
	sub, ok := tax.Subclass("house")
	if !ok || sub.NCCClass != "1a" || !sub.ScaleFields[0].Triggered("double storey") {
		t.Fatalf("subclass = %+v", sub)
	}
	if got := cat.ScopeDefaults("", "house", "new"); len(got) != 1 || got[0] != "envelope.bushfire-construction" {
		t.Fatalf("scope defaults = %v", got)
	}
	if cat.ScopeDefaults("", "shed", "new") != nil || cat.ScopeDefaults("residential", "house", "refurb") != nil {
		t.Fatal("unknown class or an empty work type must be nil")
	}
	if p, ok := cat.Preset("fit"); !ok || p.Label != "Fit-out" || len(cat.AlwaysShown()) != 1 {
		t.Fatalf("preset %+v always %v", p, cat.AlwaysShown())
	}
	if f, ok := cat.ProjectFact("contract_basis"); !ok || !f.Triggered("on a Cost Plus basis") {
		t.Fatalf("fact = %+v", f)
	}
	var groups []string
	for _, d := range cat.ProfileDeterminants() {
		groups = append(groups, d.ID)
	}
	if strings.Join(groups, ",") != "bal,heritage_status" {
		t.Fatalf("profile determinants = %v", groups)
	}
}

func TestRealKnowledgeLoadsProfile(t *testing.T) {
	cat := loadRepo(t)
	if _, ok := cat.Taxonomy().Subclass("warehouse"); !ok {
		t.Fatal("warehouse subclass missing")
	}
	if len(cat.ScopeDefaults("", "house", "new")) == 0 {
		t.Fatal("house x new scope defaults missing")
	}
	bal, _ := cat.Determinant("bal")
	if !bal.Triggered("with BAL 40 compliance") || !bal.Triggered("Bush Fire Attack Level Low") {
		t.Fatal("bal triggers")
	}
	rise, _ := cat.Determinant("rise_in_storeys")
	if rise.Triggered("Construction of a 5 storey development") {
		t.Fatal("rise in storeys must not trigger on storey counts")
	}
}

// D-04: building facts key to the site, works facts to the project; the
// registry covers every key the catalogue can produce (Load would fail).
func TestKeyScopeFollowsD04(t *testing.T) {
	cat := loadRepo(t)
	for key, want := range map[string]string{
		"hdr.building_class":                   "site",
		"hdr.subclass":                         "site",
		"det.ncc_class":                        "site",
		"det.existing_building_year":           "site",
		"det.existing_building":                "project",
		"hdr.work_type":                        "project",
		"hdr.cond.procurement_route":           "project",
		"fact.consent_number":                  "project",
		"sys.fire-active.sprinklers.presence":  "project",
		"sys.fire-active.sprinklers.condition": "site",
		"sys.fire-active.sprinklers.existing":  "site",
		"scope.fire-active.sprinklers":         "project",
	} {
		if got := cat.KeyScope(key); got != want {
			t.Errorf("%s: %q, want %q", key, got, want)
		}
	}
}
