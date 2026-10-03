package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

const whole = "p-whole"

func conf(v float64) *float64 { return &v }

func str(s string) *string { return &s }

func input(facts ...profile.Fact) profile.Input {
	return profile.Input{
		Parts: []profile.Part{{ID: whole, Label: "Whole project", Kind: "whole"}, {ID: "p-b", Label: "Building B", Kind: "building"}},
		Facts: facts,
		Thresholds: profile.Thresholds{
			Amber: map[string]float64{"presence": 0.6, "provider": 0.6, "assertion": 0.6, "header": 0.6, "determinant": 0.6},
			Green: map[string]*float64{},
		},
	}
}

func jevFact(q, v, doc string, c float64) profile.Fact {
	return profile.Fact{QuestionID: q, Value: v, DocumentID: doc, PassageID: doc + "-p1", DocumentKind: "report",
		DecidedBy: "jev", Confidence: conf(c), Excerpt: "excerpt from " + doc}
}

func tender(f profile.Fact) profile.Fact { f.DocumentKind = "commercial"; return f }

func row(t *testing.T, rows []profile.Row, part, key string) profile.Row {
	t.Helper()
	for _, r := range rows {
		if r.PartID == part && r.Key == key {
			return r
		}
	}
	t.Fatalf("no row %s %s in %+v", part, key, rows)
	return profile.Row{}
}

func noRow(t *testing.T, rows []profile.Row, part, key string) {
	t.Helper()
	for _, r := range rows {
		if r.PartID == part && r.Key == key {
			t.Fatalf("unexpected row %+v", r)
		}
	}
}

func TestUserValueWins(t *testing.T) {
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.9))
	in.User = []profile.UserValue{{PartID: whole, Key: "sys.hydraulic.gas.presence", Value: str("not_included")}}
	r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, "sys.hydraulic.gas.presence")
	if r.Band != "user" || r.Value != "not_included" || len(r.Sources) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestSupersededFactsIgnored(t *testing.T) {
	f := jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.9)
	f.Superseded = true
	noRow(t, profile.Reconcile(input(f), repoCatalog(t)), whole, "sys.hydraulic.gas.presence")
}

func TestBelowAmberFloorIsBlankButKept(t *testing.T) {
	r := row(t, profile.Reconcile(input(jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.4)), repoCatalog(t)),
		whole, "sys.hydraulic.gas.presence")
	if r.Band != "blank" || r.Value != "" || len(r.Sources) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestNilConfidenceIsBlank(t *testing.T) {
	f := jevFact("sys.hydraulic.gas.presence", "included", "d1", 0)
	f.Confidence = nil
	if r := row(t, profile.Reconcile(input(f), repoCatalog(t)), whole, "sys.hydraulic.gas.presence"); r.Band != "blank" {
		t.Fatalf("%+v", r)
	}
}

func TestGreenOnlyWhenThresholdSet(t *testing.T) {
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.95))
	if r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, "sys.hydraulic.gas.presence"); r.Band != "amber" {
		t.Fatalf("green withheld: %+v", r)
	}
	in.Thresholds.Green["presence"] = conf(0.9)
	if r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, "sys.hydraulic.gas.presence"); r.Band != "green" {
		t.Fatalf("%+v", r)
	}
}

func TestPresenceConflictIsRed(t *testing.T) {
	rows := profile.Reconcile(input(
		jevFact("sys.electrical.solar-pv.presence", "included", "d1", 0.9),
		jevFact("sys.electrical.solar-pv.presence", "not_included", "d2", 0.8)), repoCatalog(t))
	r := row(t, rows, whole, "sys.electrical.solar-pv.presence")
	if r.Band != "red" || r.Value != "" || len(r.Alternatives) != 2 {
		t.Fatalf("%+v", r)
	}
}

func TestTendersConsistentAndDiffer(t *testing.T) {
	rows := profile.Reconcile(input(
		tender(jevFact("sys.electrical.solar-pv.presence", "included", "coastal", 0.9)),
		tender(jevFact("sys.electrical.solar-pv.presence", "included", "montique", 0.9)),
		tender(jevFact("sys.electrical.solar-pv.presence", "included", "toussaint", 0.9)),
		tender(jevFact("sys.electrical.solar-pv.provider", "owner", "montique", 0.9)),
		tender(jevFact("sys.electrical.solar-pv.provider", "contractor", "coastal", 0.9)),
		tender(jevFact("sys.electrical.solar-pv.provider", "contractor", "toussaint", 0.9))), repoCatalog(t))
	p := row(t, rows, whole, "sys.electrical.solar-pv.presence")
	if p.Band != "amber" || p.Value != "included" || p.Tenders != "consistent" {
		t.Fatalf("presence %+v", p)
	}
	v := row(t, rows, whole, "sys.electrical.solar-pv.provider")
	if v.Band != "red" || v.Tenders != "differ" || v.Alternatives[0].Value != "contractor" {
		t.Fatalf("provider %+v", v)
	}
}

func TestAllowanceShownNotDerived(t *testing.T) {
	rows := profile.Reconcile(input(
		jevFact("det.bal", "BAL-LOW", "newham", 0.9),
		jevFact("det.bal.assertion", "allowance", "newham", 0.9)), repoCatalog(t))
	r := row(t, rows, whole, "det.bal")
	if r.Band != "amber" || r.Value != "BAL-LOW" || r.Assertion != "allowance" {
		t.Fatalf("%+v", r)
	}
	noRow(t, rows, whole, "det.bal.assertion")
}

func TestDerivationUsesStatedAndUserOnly(t *testing.T) {
	in := input()
	in.User = []profile.UserValue{
		{PartID: whole, Key: "det.ncc_class", Value: str("7b")},
		{PartID: whole, Key: "det.rise_in_storeys", Value: str("1")},
	}
	cat := reviewedCatalog(t)
	rows := profile.Reconcile(in, cat)
	if r := row(t, rows, whole, "det.type_of_construction"); r.Value != "C" || r.Band != "green" || r.Derived == nil {
		t.Fatalf("toc %+v", r)
	}
	if r := row(t, rows, whole, "det.max_floor_area_m2"); r.Value != "2000" {
		t.Fatalf("area %+v", r)
	}
	if r := row(t, rows, whole, "det.max_volume_m3"); r.Value != "12000" {
		t.Fatalf("volume %+v", r)
	}
	// A Jev value asserted as required is shown but never derived from.
	in = input(jevFact("det.ncc_class.7b", "stated_true", "d1", 0.9), jevFact("det.ncc_class.assertion", "required", "d1", 0.9))
	in.User = []profile.UserValue{{PartID: whole, Key: "det.rise_in_storeys", Value: str("1")}}
	rows = profile.Reconcile(in, cat)
	if r := row(t, rows, whole, "det.ncc_class"); r.Value != "7b" || r.Assertion != "required" {
		t.Fatalf("class %+v", r)
	}
	if r := row(t, rows, whole, "det.type_of_construction"); r.Value != "" || r.Derived.Reason != "missing_evidence" {
		t.Fatalf("toc must not use a required value: %+v", r)
	}
	// The real catalog is draft, so derivations wait for owner review.
	in.User = append(in.User, profile.UserValue{PartID: whole, Key: "det.ncc_class", Value: str("7b")})
	if r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, "det.type_of_construction"); r.Derived.Reason != "unreviewed" {
		t.Fatalf("real %+v", r)
	}
}

func TestSuggestions(t *testing.T) {
	in := input(jevFact("sys.hydraulic.gas.presence", "not_included", "d1", 0.9))
	in.Suggested = []string{"hydraulic.gas", "hydraulic.hot-water"}
	rows := profile.Reconcile(in, repoCatalog(t))
	if r := row(t, rows, whole, "sys.hydraulic.gas.presence"); r.Value != "not_included" || r.Band != "amber" {
		t.Fatalf("evidence must outrank a suggestion: %+v", r)
	}
	if r := row(t, rows, whole, "sys.hydraulic.hot-water.presence"); r.Value != "included" || r.Band != "suggested" {
		t.Fatalf("%+v", r)
	}
}

func TestPartLabelExactMatchAndConflict(t *testing.T) {
	a := jevFact("hdr.scale.storeys", "5", "d1", 0.9)
	a.PartLabel = "Building B"
	b := jevFact("hdr.scale.storeys", "3", "d2", 0.9)
	b.PartLabel = "building b" // not exact
	c := jevFact("hdr.scale.storeys", "4", "d3", 0.9)
	rows := profile.Reconcile(input(a, b, c), repoCatalog(t))
	if r := row(t, rows, "p-b", "hdr.scale.storeys"); r.Value != "5" {
		t.Fatalf("%+v", r)
	}
	if r := row(t, rows, whole, "hdr.scale.storeys"); r.Band != "red" {
		t.Fatalf("two values on one part must conflict: %+v", r)
	}
}

func TestNoteIsVerbatimExcerptCut(t *testing.T) {
	f := jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.9)
	f.Excerpt = "Two 45 kg LPG gas bottles — " + string(make([]rune, 0)) + "with connections and gas lines to the alfresco barbecue and kitchen cooktop, plus regulator set-up and testing"
	r := row(t, profile.Reconcile(input(f), repoCatalog(t)), whole, "sys.hydraulic.gas.presence")
	if len([]rune(r.Note)) > 120 || r.Note == "" || f.Excerpt[:20] != r.Note[:20] {
		t.Fatalf("note %q", r.Note)
	}
}

func TestDeprecatedIDsResolve(t *testing.T) {
	rows := profile.Reconcile(input(jevFact("sys.fire-passive.bushfire-construction.presence", "included", "d1", 0.9)), repoCatalog(t))
	row(t, rows, whole, "sys.envelope.bushfire-construction.presence")
}

func TestDeterministicOrder(t *testing.T) {
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "d1", 0.9), jevFact("det.bal", "BAL-40", "d2", 0.9))
	a, b := profile.Reconcile(in, repoCatalog(t)), profile.Reconcile(in, repoCatalog(t))
	for i := range a {
		if a[i].Key != b[i].Key || a[i].PartID != b[i].PartID {
			t.Fatal("order differs")
		}
	}
}

// reviewedCatalog is a small knowledge tree whose rules and tables are
// reviewed, so derivations can run.
func reviewedCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("systems.yaml", "version: 1\nsystems:\n  - {id: fire-passive, label: Passive fire, describes: x, status: draft}\n")
	write("determinants.yaml", `version: 1
determinants:
  - {id: ncc_class, label: Class, value: multi_choice, profile_group: classification, status: draft}
  - {id: rise_in_storeys, label: Rise, value: integer, profile_group: classification, status: draft}
  - {id: type_of_construction, label: Type, value: choice, derived: true, by: rule.ncc.toc, profile_group: fire, status: draft}
  - {id: max_floor_area_m2, label: Area, value: number, derived: true, by: rule.ncc.area, profile_group: fire, status: draft}
  - {id: max_volume_m3, label: Volume, value: number, derived: true, by: rule.ncc.volume, profile_group: fire, status: draft}
`)
	write("clusters/fire/rules.yaml", `version: 1
rules:
  - {id: rule.ncc.toc, status: reviewed, derives: {table: type_of_construction, inputs: [ncc_class, rise_in_storeys], gives: type_of_construction}}
  - {id: rule.ncc.area, status: reviewed, derives: {table: compartment_limits, inputs: [ncc_class, type_of_construction], gives: max_floor_area_m2}}
  - {id: rule.ncc.volume, status: reviewed, derives: {table: compartment_limits, inputs: [ncc_class, type_of_construction], gives: max_volume_m3}}
`)
	write("tables/type_of_construction.yaml", "version: 1\nid: type_of_construction\nstatus: reviewed\nverified: true\ninputs: [ncc_class, rise_in_storeys]\noutputs: [type_of_construction]\nrows:\n  - {classes: ['7b'], rise_min: 1, rise_max: 1, type_of_construction: C}\n")
	write("tables/compartment_limits.yaml", "version: 1\nid: compartment_limits\nstatus: reviewed\nverified: true\ninputs: [ncc_class, type_of_construction]\noutputs: [max_floor_area_m2, max_volume_m3]\nrows:\n  - {classes: ['7b'], type_of_construction: C, max_floor_area_m2: 2000, max_volume_m3: 12000}\n")
	cat, err := knowledge.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}
