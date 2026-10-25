package knowledge

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test fixtures only: every number, label and clause below is invented for
// the test and is not library content.
const fixtureSource = `[{design: test-fixture-only, anchor: "# Test"}]`

var programFixture = map[string]string{
	"durations.yaml": `version: 1
status: draft
sources: ` + fixtureSource + `
durations:
  - id: du.fixed
    version: 1
    basis: fixed
    unit_days: working
    low: "1"
    typical: "2.5"
    high: "4"
    geography: Test
    limitations: []
    status: reviewed
    sources: ` + fixtureSource + `
  - id: du.rate
    version: 1
    basis: per_unit
    driver: {planning_key: gross_floor_area, unit: m2}
    unit_days: working
    low: "0.1"
    typical: "0.15"
    high: "0.2"
    applies_when: {det: work_type, eq: new}
    geography: Test
    limitations: []
    status: draft
    sources: ` + fixtureSource + `
  - id: du.stat
    version: 1
    basis: statutory
    unit_days: calendar
    low: "10"
    typical: "10"
    high: "10"
    instrument: Test instrument
    clause: T1
    verified: false
    geography: Test
    limitations: []
    status: draft
    sources: ` + fixtureSource + `
  - id: du.tc
    version: 1
    basis: time_cost
    unit_days: calendar
    k: "100"
    b: "0.5"
    cost_base_date: "2020-01-01"
    index_basis: Test index
    geography: Test
    limitations: []
    status: draft
    sources: ` + fixtureSource + `
`,
	"fragments.yaml": `version: 1
status: draft
sources: ` + fixtureSource + `
fragments:
  - id: fr.a
    label: Fragment A
    stage: construction
    systems: [structure]
    applies_when: {work_action: {system: structure, any_of: [new, replace]}}
    activities:
      - {key: one, label: One, systems: [structure], duration: du.fixed}
      - {key: two, label: Two, systems: [structure.concrete], duration: null}
    links:
      - {from: one, to: two, type: FS, lag: du.fixed}
    attach: {after: stage_start, before: fr.b}
    status: draft
    sources: ` + fixtureSource + `
  - id: fr.b
    label: Fragment B
    stage: construction
    systems: [structure]
    activities:
      - {key: one, label: One, systems: [structure], duration: null}
    attach: {after: stage_start}
    status: draft
    sources: ` + fixtureSource + `
`,
	"skeletons.yaml": `version: 1
status: draft
sources: ` + fixtureSource + `
skeletons:
  - id: sk.generic
    label: Generic
    stages: [investigation, concept_design, approval_documentation, construction_documentation,
             approvals, procurement, construction, completion, defects]
    links:
      - {from: investigation, to: concept_design, type: FS, lag: null}
      - {from: concept_design, to: approval_documentation, type: FS, lag: null}
      - {from: approval_documentation, to: construction_documentation, type: FS, lag: null}
      - {from: construction_documentation, to: approvals, type: FS, lag: null}
      - {from: approvals, to: procurement, type: FS, lag: null}
      - {from: procurement, to: construction, type: FS, lag: null}
      - {from: construction, to: completion, type: FS, lag: null}
      - {from: completion, to: defects, type: FS, lag: null}
    status: draft
    sources: ` + fixtureSource + `
  - id: sk.new-da
    label: New with consent
    applies_when:
      all:
        - {det: work_type, eq: new}
        - {cond: planning, any_of: [da, cdc]}
    stages: [design, approvals, construction]
    links:
      - {from: design, to: approvals, type: FS, lag: null}
      - {from: approvals, to: construction, type: FS, lag: du.stat}
    status: draft
    sources: ` + fixtureSource + `
  - id: sk.new
    label: New
    applies_when: {det: work_type, eq: new}
    stages: [construction]
    links: []
    status: draft
    sources: ` + fixtureSource + `
`,
	"calendars.yaml": `version: 1
status: draft
sources: ` + fixtureSource + `
calendars:
  - id: cal.test-2027
    state: NSW
    year: 2027
    working_week: [mon, tue, wed, thu, fri]
    days:
      - {date: "2027-03-03", kind: public_holiday, label: Test day}
      - {from: "2027-12-24", to: "2027-12-31", kind: shutdown, label: Test shutdown}
    verified: false
    status: draft
    sources: ` + fixtureSource + `
`,
}

var repoCatalog *Catalog

// programCatalog loads the real catalogue once, then reloads only the
// program library from a temporary directory holding files.
func programCatalog(t *testing.T, files map[string]string) (*Catalog, error) {
	t.Helper()
	if repoCatalog == nil {
		c, err := Load(filepath.Join("..", "..", "knowledge"))
		if err != nil {
			t.Fatal(err)
		}
		repoCatalog = c
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "program"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, "program", name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := *repoCatalog
	c.root, c.loaded = root, map[string][]byte{}
	err := c.loadProgram(root)
	return &c, err
}

func fixtureWith(t *testing.T, file, old, replacement string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for k, v := range programFixture {
		out[k] = v
	}
	if !strings.Contains(out[file], old) {
		t.Fatalf("fixture %s has no %q", file, old)
	}
	out[file] = strings.Replace(out[file], old, replacement, 1)
	return out
}

func TestProgramLibraryInRepoLoads(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	if c.LibraryVersion() == "" {
		t.Fatal("program library not hashed")
	}
	skeletons := c.Skeletons()
	if len(skeletons) == 0 || skeletons[0].ID != GenericSkeletonID {
		t.Fatalf("want sk.generic first, got %+v", skeletons)
	}
	if got, want := strings.Join(skeletons[0].Stages, ","), strings.Join(c.GenericStages(), ","); got != want {
		t.Fatalf("generic stages %s, catalogue %s", got, want)
	}
	if _, ok := c.DeliveryStage(OtherStageID); !ok {
		t.Fatal("other stage missing from the catalogue")
	}
	choice, ok := c.MatchSkeleton(WorksEnv{})
	if !ok || choice.Skeleton.ID != GenericSkeletonID {
		t.Fatal("unknown project should fall back to sk.generic")
	}
}

func TestProgramFileNotes(t *testing.T) {
	for name := range programFixture {
		t.Run(name, func(t *testing.T) {
			files := fixtureWith(t, name, "version: 1", "notes: Descriptive library metadata\nversion: 1")
			if _, err := programCatalog(t, files); err != nil {
				t.Fatal(err)
			}
			files = fixtureWith(t, name, "version: 1", "notse: Misspelt metadata\nversion: 1")
			if _, err := programCatalog(t, files); err == nil {
				t.Fatal("unknown field accepted")
			}
		})
	}
}
func TestProgramFixtureLoads(t *testing.T) {
	c, err := programCatalog(t, programFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Fragments()) != 2 || len(c.Calendars()) != 1 || len(c.Skeletons()) != 3 {
		t.Fatal("fixture rows missing")
	}
	before := c.LibraryVersion()
	changed, err := programCatalog(t, fixtureWith(t, "fragments.yaml", "label: Fragment B", "label: Fragment B2"))
	if err != nil {
		t.Fatal(err)
	}
	if before == "" || before == changed.LibraryVersion() {
		t.Fatal("library version must follow the program files")
	}
}

// Each case breaks one checker rule (plan 4.7) and the loader must refuse it.
func TestProgramLoaderRejectsEachBadCase(t *testing.T) {
	cases := []struct{ name, file, old, replacement string }{
		// Rule 1: ids well-formed and unique; references resolve.
		{"bad skeleton id", "skeletons.yaml", "id: sk.new\n", "id: sk_New\n"},
		{"duplicate fragment id", "fragments.yaml", "id: fr.b", "id: fr.a"},
		{"duplicate duration id", "durations.yaml", "id: du.stat", "id: du.fixed"},
		{"bad duration id", "durations.yaml", "id: du.stat", "id: duration.stat"},
		{"bad calendar id", "calendars.yaml", "id: cal.test-2027", "id: nsw-2027"},
		{"unknown stage", "skeletons.yaml", "stages: [construction]", "stages: [building]"},
		{"other stage in library", "fragments.yaml", "label: Fragment B\n    stage: construction", "label: Fragment B\n    stage: other"},
		{"unknown fragment system", "fragments.yaml", "systems: [structure.concrete]", "systems: [structure.invented]"},
		{"unknown activity duration", "fragments.yaml", "duration: du.fixed}", "duration: du.missing}"},
		{"unknown lag duration", "skeletons.yaml", "lag: du.stat", "lag: du.missing"},
		{"literal lag", "skeletons.yaml", "lag: du.stat", "lag: 5"},
		{"unknown attach fragment", "fragments.yaml", "before: fr.b", "before: fr.missing"},
		{"bad attach value", "fragments.yaml", "after: stage_start, before: fr.b", "after: stage_middle, before: fr.b"},
		{"unknown planning key", "durations.yaml", "planning_key: gross_floor_area", "planning_key: excavation_volume"},
		{"unknown determinant", "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when: {det: invented, eq: new}"},
		{"unknown work type", "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when: {det: work_type, eq: demolish}"},
		{"unknown condition", "skeletons.yaml", "{cond: planning, any_of: [da, cdc]}", "{cond: approval_pathway, any_of: [da]}"},
		{"unknown condition option", "skeletons.yaml", "{cond: planning, any_of: [da, cdc]}", "{cond: planning, any_of: [da, council]}"},
		{"condition numeric operator", "skeletons.yaml", "{cond: planning, any_of: [da, cdc]}", "{cond: planning, gt: 1}"},
		// Rule 2: only documented operators.
		{"unknown operator", "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when: {work_type: new}"},
		{"work_action retain", "fragments.yaml", "any_of: [new, replace]", "any_of: [new, retain]"},
		{"work_action unknown system", "fragments.yaml", "{work_action: {system: structure,", "{work_action: {system: structure.invented,"},
		{"work_action without actions", "fragments.yaml", "{work_action: {system: structure, any_of: [new, replace]}}", "{work_action: {system: structure}}"},
		{"misspelt key", "fragments.yaml", "applies_when: {work_action", "applies_whne: {work_action"},
		// Rule 3: decimal strings, non-negative, at most 2 dp, ordered.
		{"unquoted decimal", "durations.yaml", `typical: "2.5"`, "typical: 2.5"},
		{"three decimal places", "durations.yaml", `typical: "2.5"`, `typical: "2.505"`},
		{"negative", "durations.yaml", `low: "1"`, `low: "-1"`},
		{"low above typical", "durations.yaml", `low: "1"`, `low: "3"`},
		{"time_cost without k", "durations.yaml", `k: "100"`, `k: ""`},
		{"time_cost with low", "durations.yaml", `k: "100"`, `k: "100"` + "\n    low: \"1\""},
		{"driver on fixed", "durations.yaml", "basis: fixed\n", "basis: fixed\n    driver: {planning_key: gross_floor_area, unit: m2}\n"},
		// Rule 4: statutory rows carry instrument, clause and verified.
		{"statutory without clause", "durations.yaml", "    clause: T1\n", ""},
		{"statutory without verified", "durations.yaml", "    verified: false\n", ""},
		{"statutory without instrument", "durations.yaml", "    instrument: Test instrument\n", ""},
		// Rule 5: no cycles.
		{"fragment link cycle", "fragments.yaml", "      - {from: one, to: two, type: FS, lag: du.fixed}", "      - {from: one, to: two, type: FS, lag: du.fixed}\n      - {from: two, to: one, type: SS, lag: null}"},
		{"skeleton link cycle", "skeletons.yaml", "      - {from: design, to: approvals, type: FS, lag: null}", "      - {from: design, to: approvals, type: FS, lag: null}\n      - {from: construction, to: design, type: FS, lag: null}"},
		{"attach cycle", "fragments.yaml", "attach: {after: stage_start}", "attach: {before: fr.a}"},
		{"bad link type", "skeletons.yaml", "{from: design, to: approvals, type: FS", "{from: design, to: approvals, type: XX"},
		// Rule 6: calendar dates parse, fall in the year, from ≤ to, kind closed.
		{"date outside year", "calendars.yaml", `date: "2027-03-03"`, `date: "2028-03-03"`},
		{"unquoted date", "calendars.yaml", `date: "2027-03-03"`, `date: 2027-03-03`},
		{"bad date", "calendars.yaml", `date: "2027-03-03"`, `date: "2027-02-30"`},
		{"from after to", "calendars.yaml", `from: "2027-12-24", to: "2027-12-31"`, `from: "2027-12-31", to: "2027-12-24"`},
		{"unknown kind", "calendars.yaml", "kind: shutdown", "kind: holiday"},
		{"bad working day", "calendars.yaml", "working_week: [mon,", "working_week: [monday,"},
		{"calendar without verified", "calendars.yaml", "    verified: false\n", ""},
		// Rule 7: exactly one sk.generic, which follows the catalogue; and no
		// two skeletons with identical predicates.
		{"no generic", "skeletons.yaml", "id: sk.generic", "id: sk.fallback\n    applies_when: {det: work_type, eq: refurb}"},
		{"generic with predicate", "skeletons.yaml", "    label: Generic\n", "    label: Generic\n    applies_when: {det: work_type, eq: refurb}\n"},
		{"generic wrong order", "skeletons.yaml", "[investigation, concept_design,", "[concept_design, investigation,"},
		{"generic with lag", "skeletons.yaml", "{from: completion, to: defects, type: FS, lag: null}", "{from: completion, to: defects, type: FS, lag: du.fixed}"},
		{"skeleton without predicate", "skeletons.yaml", "    applies_when: {det: work_type, eq: new}\n", ""},
		{"identical predicates", "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when:\n      all:\n        - {cond: planning, any_of: [cdc, da]}\n        - {det: work_type, eq: new}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programCatalog(t, fixtureWith(t, tc.file, tc.old, tc.replacement))
			if err == nil {
				t.Fatal("bad library loaded")
			}
			t.Log(err)
		})
	}
}

func TestDraftDurationsAreNeverApplied(t *testing.T) {
	c, err := programCatalog(t, programFixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.ReviewedDuration("du.rate"); ok {
		t.Fatal("draft duration offered")
	}
	draft := c.program.durations["du.rate"]
	if _, ok := ComputeDuration(draft, ExactInput(big.NewRat(100, 1))); ok {
		t.Fatal("draft duration computed")
	}
	env := WorksEnv{WorkTypes: []string{"new"}}
	if c.DurationApplies(draft, env) {
		t.Fatal("draft duration applies")
	}
	reviewed := draft
	reviewed.Status = statusReviewed
	if !c.DurationApplies(reviewed, env) || c.DurationApplies(reviewed, WorksEnv{}) {
		t.Fatal("reviewed applicability must hold, never be unknown")
	}
	if d, ok := c.ReviewedDuration("du.fixed"); !ok || d.ID != "du.fixed" {
		t.Fatal("reviewed duration hidden")
	}
}

func TestComputeDurationRoundsUp(t *testing.T) {
	fixed := Duration{Basis: "fixed", Low: "1", Typical: "2.5", High: "4.01", Status: statusReviewed}
	got, ok := ComputeDuration(fixed, DurationInput{})
	if !ok || got != (DayRange{Low: 1, Typical: 3, High: 5, TypicalKnown: true}) {
		t.Fatalf("fixed %+v", got)
	}
	rate := Duration{Basis: "per_unit", Low: "0.1", Typical: "0.15", High: "0.2", Status: statusReviewed}
	// 0.15 × 100 is exactly 15; float64 would make it 15.000000000000002 and
	// push it up a day.
	got, ok = ComputeDuration(rate, ExactInput(big.NewRat(100, 1)))
	if !ok || got != (DayRange{Low: 10, Typical: 15, High: 20, TypicalKnown: true}) {
		t.Fatalf("per_unit %+v", got)
	}
	got, ok = ComputeDuration(rate, DurationInput{Low: big.NewRat(101, 1), High: big.NewRat(201, 1)})
	if !ok || got != (DayRange{Low: 11, High: 41}) {
		t.Fatalf("per_unit range %+v", got)
	}
	cycle := Duration{Basis: "floor_cycle", Low: "7", Typical: "8.5", High: "10", Status: statusReviewed}
	got, ok = ComputeDuration(cycle, ExactInput(big.NewRat(3, 1)))
	if !ok || got != (DayRange{Low: 21, Typical: 26, High: 30, TypicalKnown: true}) {
		t.Fatalf("floor_cycle %+v", got)
	}
	if _, ok := ComputeDuration(rate, DurationInput{}); ok {
		t.Fatal("unknown driver must stay unknown")
	}
	if _, ok := ComputeDuration(rate, DurationInput{Low: big.NewRat(5, 1), High: big.NewRat(4, 1)}); ok {
		t.Fatal("inverted range accepted")
	}
	if _, ok := ComputeDuration(rate, ExactInput(big.NewRat(-1, 1))); ok {
		t.Fatal("negative driver accepted")
	}
}

func TestComputeTimeCost(t *testing.T) {
	tc := Duration{Basis: "time_cost", K: "100", B: "0.5", Status: statusReviewed}
	// 100 × 2.25^0.5 = 150 exactly; 100 × 2^0.5 = 141.42… rounds up to 142.
	got, ok := ComputeDuration(tc, DurationInput{Low: big.NewRat(2, 1), Typical: big.NewRat(9, 4), High: big.NewRat(4, 1)})
	if !ok || got != (DayRange{Low: 142, Typical: 150, High: 200, TypicalKnown: true}) {
		t.Fatalf("time_cost %+v", got)
	}
	if _, ok := ComputeDuration(tc, ExactInput(big.NewRat(0, 1))); ok {
		t.Fatal("zero cost accepted")
	}
	tc.Status = "draft"
	if _, ok := ComputeDuration(tc, ExactInput(big.NewRat(2, 1))); ok {
		t.Fatal("draft time_cost computed")
	}
}

func TestMatchSkeletonCountsTermsAndNeverTrustsUnknowns(t *testing.T) {
	c, err := programCatalog(t, programFixture)
	if err != nil {
		t.Fatal(err)
	}
	pick := func(env WorksEnv) SkeletonChoice {
		t.Helper()
		choice, ok := c.MatchSkeleton(env)
		if !ok {
			t.Fatal("no skeleton")
		}
		return choice
	}
	if got := pick(WorksEnv{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da"}}); got.Skeleton.ID != "sk.new-da" || got.Terms != 2 || len(got.Ties) != 0 {
		t.Fatalf("most terms should win: %+v", got)
	}
	// The planning condition is unknown, so sk.new-da cannot hold.
	if got := pick(WorksEnv{WorkTypes: []string{"new"}}); got.Skeleton.ID != "sk.new" {
		t.Fatalf("unknown condition matched: %s", got.Skeleton.ID)
	}
	if got := pick(WorksEnv{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "exempt"}}); got.Skeleton.ID != "sk.new" {
		t.Fatalf("false condition matched: %s", got.Skeleton.ID)
	}
	// Unknown work type: nothing holds, so the fallback applies.
	if got := pick(WorksEnv{}); got.Skeleton.ID != GenericSkeletonID {
		t.Fatalf("unknown work type matched: %s", got.Skeleton.ID)
	}
	if got := pick(WorksEnv{WorkTypes: []string{"refurb"}, Conditions: map[string]string{"planning": "da"}}); got.Skeleton.ID != GenericSkeletonID {
		t.Fatalf("wrong work type matched: %s", got.Skeleton.ID)
	}
	tied, err := programCatalog(t, fixtureWith(t, "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when: {cond: procurement_route, eq: traditional}"))
	if err != nil {
		t.Fatal(err)
	}
	choice, _ := tied.MatchSkeleton(WorksEnv{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da", "procurement_route": "traditional"}})
	if choice.Skeleton.ID != "sk.new-da" {
		t.Fatalf("more terms should beat fewer: %+v", choice)
	}
	if PredicateTerms(map[string]any{"all": []any{map[string]any{"det": "work_type", "eq": "new"}, map[string]any{"not": map[string]any{"cond": "planning", "eq": "da"}}}}) != 2 {
		t.Fatal("term count")
	}
}

func TestSkeletonTiesAreReported(t *testing.T) {
	c, err := programCatalog(t, fixtureWith(t, "skeletons.yaml", "applies_when: {det: work_type, eq: new}", "applies_when: {cond: procurement_route, eq: traditional}"))
	if err != nil {
		t.Fatal(err)
	}
	// Give sk.new two terms as well so both match with equal weight.
	s := c.program.skeletons[2]
	s.AppliesWhen = map[string]any{"all": []any{map[string]any{"cond": "procurement_route", "eq": "traditional"}, map[string]any{"det": "work_type", "eq": "new"}}}
	c.program.skeletons[2] = s
	choice, _ := c.MatchSkeleton(WorksEnv{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da", "procurement_route": "traditional"}})
	if choice.Skeleton.ID != "sk.new-da" || len(choice.Ties) != 1 || choice.Ties[0] != "sk.new" {
		t.Fatalf("tie not reported: %+v", choice)
	}
}

func TestWorkActionReadsIncludedWorkItems(t *testing.T) {
	c, err := programCatalog(t, programFixture)
	if err != nil {
		t.Fatal(err)
	}
	pred := c.Fragments()[0].AppliesWhen
	cases := []struct {
		items []WorkItem
		want  Truth
	}{
		{nil, False},
		{[]WorkItem{{System: "structure.concrete", Action: "new"}}, True},
		{[]WorkItem{{System: "structure", Action: "replace"}}, True},
		{[]WorkItem{{System: "structure", Action: "retain"}}, False},
		{[]WorkItem{{System: "structure", Action: "investigate"}}, False},
		{[]WorkItem{{System: "structure", Action: ""}}, Unknown},
		{[]WorkItem{{System: "envelope", Action: "new"}}, False},
	}
	for _, tc := range cases {
		if got := c.Holds(pred, WorksEnv{Items: tc.items}); got != tc.want {
			t.Fatalf("%+v: got %s want %s", tc.items, got, tc.want)
		}
	}
}

func TestSamePredicateIgnoresListOrder(t *testing.T) {
	a := map[string]any{"all": []any{map[string]any{"det": "work_type", "eq": "new"}, map[string]any{"cond": "planning", "any_of": []any{"da", "cdc"}}}}
	b := map[string]any{"all": []any{map[string]any{"cond": "planning", "any_of": []any{"cdc", "da"}}, map[string]any{"det": "work_type", "eq": "new"}}}
	if !SamePredicate(a, b) {
		t.Fatal("reordered predicate not identical")
	}
	if SamePredicate(a, map[string]any{"det": "work_type", "eq": "new"}) {
		t.Fatal("different predicates identical")
	}
}

func TestProjectFragmentScope(t *testing.T) {
	project := `
  - id: fr.project
    label: Project activity
    scope: project
    stage: approvals
    systems: [site]
    applies_when: {cond: planning, eq: da}
    activities:
      - {key: lodge, label: Lodge, systems: [site], duration: du.fixed}
    status: draft
    sources: ` + fixtureSource + `
`
	files := fixtureWith(t, "fragments.yaml", "fragments:\n", "fragments:\n"+project)
	c, err := programCatalog(t, files)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range c.Fragments() {
		if f.ID == "fr.project" {
			found = true
			if f.Scope != ScopeProject {
				t.Fatalf("scope %q", f.Scope)
			}
		}
	}
	if !found {
		t.Fatal("project fragment not loaded")
	}
	bad := fixtureWith(t, "fragments.yaml", "fragments:\n", "fragments:\n"+strings.Replace(project, "scope: project", "scope: job", 1))
	if _, err := programCatalog(t, bad); err == nil {
		t.Fatal("scope job accepted")
	}
	acted := strings.Replace(project, "applies_when: {cond: planning, eq: da}",
		"applies_when: {work_action: {system: site, any_of: [new]}}", 1)
	if _, err := programCatalog(t, fixtureWith(t, "fragments.yaml", "fragments:\n", "fragments:\n"+acted)); err == nil {
		t.Fatal("project fragment with a work action accepted")
	}
}
