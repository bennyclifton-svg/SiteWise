package program

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/knowledge"
)

// Test fixtures only: every label, number and predicate in these library
// files is invented for the tests and is not library content.
const src = `[{design: test-fixture-only, anchor: "# Test"}]`

var genericSkeleton = `  - id: sk.generic
    label: Generic stage sequence
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
    sources: ` + src + `
`

var testLibrary = map[string]string{
	"skeletons.yaml": `version: 1
status: draft
sources: ` + src + `
skeletons:
` + genericSkeleton + `  - id: sk.new-da
    label: Test new with consent
    applies_when:
      all:
        - {det: work_type, eq: new}
        - {cond: planning, eq: da}
    stages: [investigation, concept_design, approval_documentation, approvals,
             construction_documentation, procurement, construction, completion]
    links:
      - {from: approvals, to: construction, type: FS, lag: du.draft}
    status: draft
    sources: ` + src + `
  - id: sk.new
    label: Test new
    applies_when: {det: work_type, eq: new}
    stages: [procurement, construction]
    links: []
    status: draft
    sources: ` + src + `
  - id: sk.tie-a
    label: Test tie A
    applies_when:
      all:
        - {det: work_type, eq: refurb}
        - {cond: planning, eq: cdc}
    stages: [construction]
    links: []
    status: draft
    sources: ` + src + `
  - id: sk.tie-b
    label: Test tie B
    applies_when:
      all:
        - {det: work_type, eq: refurb}
        - {cond: procurement_route, eq: traditional}
    stages: [construction]
    links: []
    status: draft
    sources: ` + src + `
  - id: sk.state
    label: Test state
    applies_when:
      all:
        - {det: work_type, eq: remediation}
        - {det: state, eq: NSW}
    stages: [construction]
    links: []
    status: draft
    sources: ` + src + `
`,
	"fragments.yaml": `version: 1
status: draft
sources: ` + src + `
fragments:
  - id: fr.slab
    label: Test slab
    stage: construction
    systems: [substructure]
    applies_when: {work_action: {system: substructure, any_of: [new, replace]}}
    activities:
      - {key: form, label: Form slab, systems: [substructure.slab-on-ground], duration: du.fixed}
      - {key: pour, label: Pour slab, systems: [substructure.slab-on-ground], duration: du.draft}
    links:
      - {from: form, to: pour, type: FS, lag: du.lag}
    attach: {after: stage_start}
    status: draft
    sources: ` + src + `
  - id: fr.demo
    label: Test demolition
    stage: construction
    systems: [structure]
    applies_when: {work_action: {system: structure, any_of: [remove]}}
    activities:
      - {key: demolish, label: Demolish, systems: [structure], duration: du.fixed}
    status: draft
    sources: ` + src + `
  - id: fr.frame
    label: Test frame
    stage: construction
    systems: [structure]
    applies_when:
      all:
        - {work_action: {system: structure, any_of: [new]}}
        - {det: flood_hazard, is: true}
    activities:
      - {key: frame, label: Erect frame, systems: [structure], duration: du.fixed}
    attach: {after: fr.slab}
    status: draft
    sources: ` + src + `
  - id: fr.roof
    label: Test roof
    stage: construction
    systems: [envelope]
    applies_when: {work_action: {system: envelope, any_of: [new]}}
    activities:
      - {key: roof, label: Roof covering, systems: [envelope.roof-coverings], duration: null}
    status: draft
    sources: ` + src + `
`,
	"durations.yaml": `version: 1
status: draft
sources: ` + src + `
durations:
  - id: du.fixed
    version: 1
    basis: fixed
    unit_days: working
    low: "3"
    typical: "5"
    high: "8"
    geography: Test
    limitations: []
    status: reviewed
    sources: ` + src + `
  - id: du.lag
    version: 1
    basis: fixed
    unit_days: working
    low: "2"
    typical: "2"
    high: "2"
    geography: Test
    limitations: []
    status: reviewed
    sources: ` + src + `
  - id: du.draft
    version: 1
    basis: fixed
    unit_days: working
    low: "1"
    typical: "1"
    high: "1"
    geography: Test
    limitations: []
    status: draft
    sources: ` + src + `
`,
	"calendars.yaml": `version: 1
status: draft
sources: ` + src + `
calendars: []
`,
}

// testHoldPoint is an interface with a hold point whose ends are system ids,
// appended to the envelope cluster in the copied catalogue.
const testHoldPoint = `- id: if.test-frame-before-roof
  type: sequences
  from: [structure]
  to: [envelope.roof-coverings]
  summary: Test only.
  status: draft
  resolved_when: []
  hold_point:
    before: envelope.roof-coverings
    after: structure
    evidence: Test frame inspection record.
`

var (
	catalogs   = map[string]*knowledge.Catalog{}
	catalogsMu sync.Mutex
)

// testCatalog copies the repository's knowledge directory, replaces the
// program library with files (nil keeps the repo's empty library), appends
// interfaces to the envelope cluster, and loads it once per distinct input.
func testCatalog(t *testing.T, files map[string]string, interfaces string) *knowledge.Catalog {
	t.Helper()
	key, _ := json.Marshal([]any{files, interfaces})
	catalogsMu.Lock()
	defer catalogsMu.Unlock()
	if c, ok := catalogs[string(key)]; ok {
		return c
	}
	repo := filepath.Join("..", "..", "knowledge")
	root, err := os.MkdirTemp("", "program-knowledge-")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(repo, path)
		dest := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, body, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, "program", name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if interfaces != "" {
		path := filepath.Join(root, "clusters", "envelope", "interfaces.yaml")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(body, []byte("\n"+interfaces)...), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := knowledge.Load(root)
	os.RemoveAll(root)
	if err != nil {
		t.Fatal(err)
	}
	catalogs[string(key)] = c
	return c
}

func mustGenerate(t *testing.T, cat *knowledge.Catalog, in GenInput) Plan {
	t.Helper()
	p, err := Generate(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rowKeys(p Plan, parent string) []string {
	var out []string
	for _, r := range p.Rows {
		if r.ParentKey == parent {
			out = append(out, r.SourceKey)
		}
	}
	return out
}

func hasLink(p Plan, pred, succ string) (PlanLink, bool) {
	for _, l := range p.Links {
		if l.PredKey == pred && l.SuccKey == succ {
			return l, true
		}
	}
	return PlanLink{}, false
}

func hasNote(p Plan, code string) bool {
	for _, n := range p.Notes {
		if n.Code == code {
			return true
		}
	}
	return false
}

func TestSkeletonChoice(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	cases := []struct {
		name  string
		in    GenInput
		want  string
		tie   bool
		terms []string
	}{
		{"most terms wins", GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da"}}, "sk.new-da", false,
			[]string{"Nature of the building works: New build", "Planning: DA"}},
		// The planning condition is unknown, so sk.new-da cannot hold even
		// though its other term does: unknown is never true.
		{"unknown term never true", GenInput{WorkTypes: []string{"new"}}, "sk.new", false, []string{"Nature of the building works: New build"}},
		{"nothing known", GenInput{}, "sk.generic", false, nil},
		{"tie", GenInput{WorkTypes: []string{"refurb"}, Conditions: map[string]string{"planning": "cdc", "procurement_route": "traditional"}}, "sk.tie-a", true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := mustGenerate(t, cat, c.in)
			if p.SkeletonID != c.want || p.SkeletonTie != c.tie || hasNote(p, CheckSkeletonTie) != c.tie {
				t.Fatalf("got %s tie=%v notes=%v", p.SkeletonID, p.SkeletonTie, p.Notes)
			}
			for _, term := range c.terms {
				if !strings.Contains(p.Reason, term) {
					t.Fatalf("reason %q lacks %q", p.Reason, term)
				}
			}
			sk := cat.Skeletons()
			var want []string
			for _, s := range sk {
				if s.ID == c.want {
					want = s.Stages
				}
			}
			var got []string
			for _, r := range p.Rows {
				if r.Kind == KindStage {
					got = append(got, r.CatalogueStageID)
					if r.SourceKey != "skeleton:"+c.want+":"+r.CatalogueStageID {
						t.Fatalf("stage key %s", r.SourceKey)
					}
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("stages %v, want %v", got, want)
			}
			ci := p.CheckInput(Input{}, Output{}, c.in, nil)
			if ci.SkeletonTie != c.tie {
				t.Fatal("check input lost the tie")
			}
		})
	}
}

func TestStageChainAndSkeletonLinks(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	p := mustGenerate(t, cat, GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da"}})
	stages := rowKeys(p, "")
	for i := 1; i < len(stages); i++ {
		l, ok := hasLink(p, stages[i-1], stages[i])
		if !ok || l.Origin != LinkOriginChain || l.Type != LinkFS || l.Lag != 0 {
			t.Fatalf("missing chain %s → %s: %+v", stages[i-1], stages[i], l)
		}
	}
	// approvals → construction names a draft lag: kept with no lag, flagged.
	l, ok := hasLink(p, "skeleton:sk.new-da:approvals", "skeleton:sk.new-da:construction")
	if !ok || l.Origin != LinkOriginLibrary || !l.LagUnknown || l.Lag != 0 || l.LagDurationID != "du.draft" {
		t.Fatalf("library link %+v", l)
	}
	if !hasNote(p, CheckLagUnknown) {
		t.Fatal("unknown lag not reported")
	}
	if p.ConstructionStageKey != "skeleton:sk.new-da:construction" {
		t.Fatal(p.ConstructionStageKey)
	}
}

func TestWorkActions(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	in := GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da"},
		Values: map[string]string{"flood_hazard": "true"},
		WorkItems: []WorkItem{
			{ID: "wi-slab", System: "substructure.slab-on-ground", Action: "new", Title: "New slab"},
			{ID: "wi-keep", System: "structure.timber", Action: "retain", Title: "Keep the frame"},
			{ID: "wi-geo", System: "site", Action: "investigate", Title: "Geotechnical investigation"},
			{ID: "wi-demo", System: "structure.masonry", Action: "remove", Title: "Demolish existing garage"},
		}}
	p := mustGenerate(t, cat, in)

	// retain adds nothing.
	for _, r := range p.Rows {
		if r.WorkItemID == "wi-keep" {
			t.Fatalf("retain added %s", r.SourceKey)
		}
	}
	// investigate adds one activity under Investigation, titled as written.
	inv := rowKeys(p, "skeleton:sk.new-da:investigation")
	if len(inv) != 1 || inv[0] != "investigate:wi-geo" {
		t.Fatalf("investigation rows %v", inv)
	}
	if r, _ := p.Row("investigate:wi-geo"); r.Label != "Geotechnical investigation" {
		t.Fatal(r.Label)
	}
	// remove selects only the demolition fragment, labelled with the title.
	demo, ok := p.Row("fragment:fr.demo:wi-demo:demolish")
	if !ok || demo.Label != "Demolish existing garage" {
		t.Fatalf("demolition row %+v", demo)
	}
	if _, ok := p.Row("fragment:fr.frame:wi-demo:frame"); ok {
		t.Fatal("remove item selected a non-demolition fragment")
	}
	// new selects the slab fragment, two activities keep library labels.
	form, ok1 := p.Row("fragment:fr.slab:wi-slab:form")
	pour, ok2 := p.Row("fragment:fr.slab:wi-slab:pour")
	if !ok1 || !ok2 || form.Label != "Form slab" || pour.Label != "Pour slab" {
		t.Fatalf("slab rows %+v %+v", form, pour)
	}
	l, ok := hasLink(p, form.SourceKey, pour.SourceKey)
	if !ok || l.Lag != 2 || l.LagUnknown || l.Origin != LinkOriginLibrary {
		t.Fatalf("internal link %+v", l)
	}
	// The demolition activity has no placed predecessor, so it is chained
	// behind the activity before it; the slab is anchored at stage start.
	if _, ok := hasLink(p, pour.SourceKey, demo.SourceKey); !ok {
		t.Fatal("demolition not chained")
	}
	for _, l := range p.Links {
		if l.SuccKey == form.SourceKey {
			t.Fatalf("anchored slab got predecessor %+v", l)
		}
	}
}

func TestConditionalFragment(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	items := []WorkItem{{ID: "wi-frame", System: "structure", Action: "new", Title: "Timber frame"},
		{ID: "wi-slab", System: "substructure", Action: "new", Title: "Slab"}}
	unknown := mustGenerate(t, cat, GenInput{WorkTypes: []string{"new"}, WorkItems: items})
	r, ok := unknown.Row("fragment:fr.frame:wi-frame:frame")
	if !ok {
		t.Fatal("unknown predicate dropped the fragment")
	}
	if r.Label != "If Site in a flood hazard area: Timber frame" {
		t.Fatalf("label %q", r.Label)
	}
	if *r.Duration != 0 || *r.Low != 0 || *r.High != 8 || r.DurationSource["conditional"] != true {
		t.Fatalf("conditional durations %v %v %v", *r.Duration, *r.Low, *r.High)
	}
	// attach after fr.slab links each slab sink to the frame.
	if _, ok := hasLink(unknown, "fragment:fr.slab:wi-slab:pour", r.SourceKey); !ok {
		t.Fatal("attach link missing")
	}
	no := mustGenerate(t, cat, GenInput{WorkTypes: []string{"new"}, WorkItems: items, Values: map[string]string{"flood_hazard": "false"}})
	if _, ok := no.Row(r.SourceKey); ok {
		t.Fatal("false predicate kept the fragment")
	}
	yes := mustGenerate(t, cat, GenInput{WorkTypes: []string{"new"}, WorkItems: items, Values: map[string]string{"flood_hazard": "true"}})
	if r, _ := yes.Row("fragment:fr.frame:wi-frame:frame"); r.Label != "Timber frame" || *r.Duration != 5 || *r.Low != 3 || *r.High != 8 {
		t.Fatalf("true predicate row %+v", r)
	}
}

func TestPackageActivitiesAndStageMap(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	in := GenInput{WorkTypes: []string{"new"}, Packages: []Package{
		{ID: "arch", Title: "Architect", Kind: "services", Stages: []PackageStage{
			{ID: "ps1", CatalogueStageID: "concept_design", Label: "Concept design"},
			{ID: "ps2", CatalogueStageID: "construction_stage_inspections", Label: "Construction-stage inspections"},
			{ID: "ps3", CatalogueStageID: "defects", Label: "Defects"},
			{ID: "ps4", CatalogueStageID: "procurement", Label: "Tender support"},
		}},
		{ID: "build", Title: "Head contract", Kind: "works"},
	}}
	p := mustGenerate(t, cat, in) // sk.new: [procurement, construction]
	want := map[string]struct{ stage, rule string }{
		"ps1": {"skeleton:sk.new:procurement", PlaceFirst},
		"ps2": {"skeleton:sk.new:construction", PlaceParent},
		"ps3": {"skeleton:sk.new:construction", PlaceNearest},
		"ps4": {"skeleton:sk.new:procurement", PlaceExact},
	}
	if len(p.StageMap) != 4 {
		t.Fatalf("stage map %+v", p.StageMap)
	}
	for _, m := range p.StageMap {
		w := want[m.PackageStageID]
		if m.StageKey != w.stage || m.Rule != w.rule || m.PackageID != "arch" {
			t.Fatalf("mapping %+v, want %+v", m, w)
		}
		r, ok := p.Row("package:arch:" + m.PackageStageID)
		if !ok || r.ParentKey != m.StageKey || r.PackageStageID != m.PackageStageID || r.PackageID != "arch" {
			t.Fatalf("row %+v", r)
		}
	}
	if r, _ := p.Row("package:arch:ps4"); r.Label != "Architect: Tender support" {
		t.Fatal(r.Label)
	}
	r, ok := p.Row("package:build:")
	if !ok || r.Label != "Head contract" || r.ParentKey != "skeleton:sk.new:construction" {
		t.Fatalf("works package row %+v", r)
	}
	// Two sub-stages split a top-level stage: Design goes to the first one.
	gp := mustGenerate(t, cat, GenInput{Packages: []Package{{ID: "x", Title: "X", Kind: "services",
		Stages: []PackageStage{{ID: "s", CatalogueStageID: "design", Label: "Design"}}}}})
	if m := gp.StageMap[0]; m.StageKey != "skeleton:sk.generic:concept_design" || m.Rule != PlaceSubstage {
		t.Fatalf("design mapping %+v", m)
	}
}

func TestHoldPoints(t *testing.T) {
	cat := testCatalog(t, testLibrary, testHoldPoint)
	items := []WorkItem{{ID: "wi-frame", System: "structure", Action: "new", Title: "Timber frame"},
		{ID: "wi-roof", System: "envelope", Action: "new", Title: "Roof"}}
	base := GenInput{WorkTypes: []string{"new"}, Values: map[string]string{"flood_hazard": "true"}, WorkItems: items}
	p := mustGenerate(t, cat, base)
	h, ok := p.Row("hold:if.test-frame-before-roof")
	if !ok || h.Kind != KindHoldPoint || h.Label != "Hold point: Test frame inspection record." || h.ParentKey != "skeleton:sk.new:construction" {
		t.Fatalf("hold point %+v", h)
	}
	frame, roof := "fragment:fr.frame:wi-frame:frame", "fragment:fr.roof:wi-roof:roof"
	for _, pair := range [][2]string{{frame, h.SourceKey}, {h.SourceKey, roof}} {
		l, ok := hasLink(p, pair[0], pair[1])
		if !ok || l.Origin != LinkOriginInterface || l.Type != LinkFS || l.Lag != 0 || l.InterfaceID != "if.test-frame-before-roof" {
			t.Fatalf("hold link %v: %+v", pair, l)
		}
	}
	out := Compute(p.Input(Calendar{}))
	if checks := Checks(CheckInput{Input: p.Input(Calendar{}), Output: out}); len(checks) != 0 {
		t.Fatalf("hold point unlinked: %+v", checks)
	}
	// Only one side present: no hold point.
	one := base
	one.WorkItems = items[:1]
	if _, ok := mustGenerate(t, cat, one).Row(h.SourceKey); ok {
		t.Fatal("hold point without its before system")
	}
	// The repository's hold points name free text, not systems: none.
	repo := testCatalog(t, nil, "")
	for _, r := range mustGenerate(t, repo, base).Rows {
		if r.Kind == KindHoldPoint {
			t.Fatalf("free-text hold point inserted: %+v", r)
		}
	}
}

func TestDurationPrecedence(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	form := "fragment:fr.slab:wi:form"
	pour := "fragment:fr.slab:wi:pour"
	stage := "skeleton:sk.new:construction"
	items := []WorkItem{{ID: "wi", System: "substructure", Action: "new", Title: "Slab"}}
	base := GenInput{WorkTypes: []string{"new"}, WorkItems: items, Planning: map[string]string{"construction_duration": "10"}}

	p := mustGenerate(t, cat, base)
	f, _ := p.Row(form)
	if *f.Duration != 5 || *f.Low != 3 || *f.High != 8 || f.DurationOrigin != DurationLibrary ||
		f.DurationSource["library"] != "du.fixed" || f.DurationSource["version"] != 1 {
		t.Fatalf("library duration %+v", f)
	}
	if r, _ := p.Row(pour); r.Duration != nil || r.Low != nil || r.DurationOrigin != "" {
		t.Fatalf("draft row used: %+v", r)
	}
	// A child has a duration, so construction_duration does not apply.
	if r, _ := p.Row(stage); r.Duration != nil {
		t.Fatal("profile duration applied over a child with a duration")
	}

	doc := base
	doc.DocumentDurations = map[string]Reading{form: {Days: 6, ReadingID: "rd1"}}
	if r, _ := mustGenerate(t, cat, doc).Row(form); *r.Duration != 6 || r.Low != nil || r.DurationOrigin != DurationDocument {
		t.Fatalf("document duration %+v", r)
	}
	user := doc
	user.UserDurations = map[string]int{form: 7}
	if r, _ := mustGenerate(t, cat, user).Row(form); *r.Duration != 7 || r.DurationOrigin != DurationUser {
		t.Fatalf("user duration %+v", r)
	}

	// No activities: the Construction stage takes construction_duration
	// weeks × 5 working days; a stage reading outranks it.
	empty := GenInput{WorkTypes: []string{"new"}, Planning: map[string]string{"construction_duration": "10"}}
	if r, _ := mustGenerate(t, cat, empty).Row(stage); r.Duration == nil || *r.Duration != 50 || r.DurationOrigin != DurationProfile {
		t.Fatalf("profile duration %+v", r)
	}
	empty.StageReadings = map[string]Reading{"construction": {Days: 40, ReadingID: "rd2"}}
	if r, _ := mustGenerate(t, cat, empty).Row(stage); *r.Duration != 40 || r.DurationOrigin != DurationDocument {
		t.Fatalf("stage reading %+v", r)
	}
}

// WP-P5 done: with the empty library a project gets sk.generic stages, one
// activity per package stage, every duration unknown, and status words that
// say so; no question is offered because nothing in the library reads a fact.
func TestEmptyLibrary(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := GenInput{Facts: Facts{ConsentDate: "2027-02-01"},
		WorkItems: []WorkItem{{ID: "wi", System: "substructure", Action: "new", Title: "Slab"}},
		Packages: []Package{
			{ID: "arch", Title: "Architect", Kind: "services", Stages: []PackageStage{
				{ID: "a1", CatalogueStageID: "concept_design", Label: "Concept design"},
				{ID: "a2", CatalogueStageID: "approval_documentation", Label: "Approval documentation"},
				{ID: "a3", CatalogueStageID: "construction_stage_inspections", Label: "Construction-stage inspections"}}},
			{ID: "hc", Title: "Head contract", Kind: "works"},
		}}
	p := mustGenerate(t, cat, in)
	if p.SkeletonID != knowledge.GenericSkeletonID || len(rowKeys(p, "")) != len(cat.GenericStages()) {
		t.Fatalf("skeleton %s stages %v", p.SkeletonID, rowKeys(p, ""))
	}
	if p.StartSuggestion != "2027-02-01" {
		t.Fatal("consent date not offered")
	}
	activities := 0
	for _, r := range p.Rows {
		if r.Kind != KindStage {
			activities++
			if r.PackageID == "" {
				t.Fatalf("non-package activity %s", r.SourceKey)
			}
		}
		if r.Duration != nil || r.ReviewStatus != ReviewProposed || r.Origin != OriginLibrary {
			t.Fatalf("row %+v", r)
		}
	}
	if activities != 4 || len(p.StageMap) != 3 {
		t.Fatalf("%d activities, %d mappings", activities, len(p.StageMap))
	}
	out := Compute(p.Input(Calendar{}))
	// 9 stages, 3 of them with activities; 6 leaf stages + 4 activities.
	if out.Status.Known != 0 || out.Status.Total != 10 || out.Status.Words != "Indicative: 0 of 10 durations known" {
		t.Fatalf("status %+v", out.Status)
	}
	if out.Cycle != nil {
		t.Fatal(out.Cycle)
	}
	q, err := OneQuestion(cat, in, nil)
	if err != nil || q != nil {
		t.Fatalf("question %+v %v", q, err)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	cat := testCatalog(t, testLibrary, testHoldPoint)
	in := GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "da"},
		Present: map[string]bool{"structure": true, "substructure.basement-retention": false},
		WorkItems: []WorkItem{
			{ID: "a", System: "substructure", Action: "new", Title: "Slab"},
			{ID: "b", System: "structure", Action: "new", Title: "Frame"},
			{ID: "c", System: "envelope", Action: "new", Title: "Roof"},
			{ID: "d", System: "site", Action: "investigate", Title: "Survey"}},
		Packages: []Package{{ID: "p", Title: "Engineer", Kind: "services",
			Stages: []PackageStage{{ID: "s", CatalogueStageID: "concept_design", Label: "Concept design"}}}}}
	first, _ := json.Marshal(mustGenerate(t, cat, in))
	for i := 0; i < 5; i++ {
		again, _ := json.Marshal(mustGenerate(t, cat, in))
		if string(again) != string(first) {
			t.Fatal("generation is not deterministic")
		}
	}
	p := mustGenerate(t, cat, in)
	if out := Compute(p.Input(Calendar{})); out.Cycle != nil {
		t.Fatalf("generated a loop: %v", out.Cycle)
	}
	if err := Validate(p.Input(Calendar{})); err != nil {
		t.Fatal(err)
	}
}

func TestLoopGuardSkipsLinks(t *testing.T) {
	rows := []Row{{SourceKey: "s1", Kind: KindStage}, {SourceKey: "s2", Kind: KindStage},
		{SourceKey: "a", ParentKey: "s1"}, {SourceKey: "b", ParentKey: "s2"}}
	g := newLoopGuard(rows, map[string]int{"s1": 1, "s2": 1})
	g.add(0, 1, LinkFS) // s1 → s2
	// b (inside s2) → a (inside s1) would hold s1 behind s2: a loop.
	if !g.closes(3, 2, LinkFS) {
		t.Fatal("loop through stages not seen")
	}
	if g.closes(2, 3, LinkFS) {
		t.Fatal("forward link refused")
	}
}
