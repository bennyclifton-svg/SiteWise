package knowledge

import (
	"os"
	"strings"
	"testing"
)

func realRFPSkeleton(t *testing.T) RFPSkeleton {
	t.Helper()
	body, err := os.ReadFile("../../knowledge/rfp/skeleton.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s, err := parseRFPSkeleton(body)
	if err != nil {
		t.Fatal(err)
	}
	modules, err := os.ReadFile("../../knowledge/rfp/modules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Version int         `yaml:"version"`
		Status  string      `yaml:"status"`
		Modules []RFPModule `yaml:"modules"`
	}
	if err := strictYAML(modules, &file); err != nil {
		t.Fatal(err)
	}
	s.Modules = file.Modules
	return s
}

const minimalDiscipline = `
id: architecture
version: 1
status: draft
title: Architectural services
heading: RFP
discipline_ids: [consultant.architect]
boundary:
  - {start: concept, end: concept}
stages:
  - {key: concept, name: Concept, deliverable: x, fee_basis: fixed, shown_when: Always}
`

func parseWith(t *testing.T, extra string) error {
	t.Helper()
	_, err := ParseRFPDiscipline([]byte(minimalDiscipline+extra), realRFPSkeleton(t))
	return err
}

func TestLoadRFPMinimalDiscipline(t *testing.T) {
	if err := parseWith(t, ""); err != nil {
		t.Fatal(err)
	}
}

func TestDisciplineOverridesValidate(t *testing.T) {
	for _, extra := range []string{
		"positions:\n  - {key: attendance, label: Attendance, kind: technical, value: {value: I, when: [{stage_included: [missing]}], else: O}}\n",
		"terms:\n  - {key: missing, replace: true, text: x}\n",
		"evaluation:\n  - {key: price, label: Fee, default: 90}\n",
		"evaluation:\n  - {key: price, label: Fee, default: 110}\n  - {key: team, label: Team, default: -10}\n",
		"returnable_notes:\n  - {key: r3, text: x, replace: true, provenance: [{kind: client_decision, ref: r}]}\n",
	} {
		if err := parseWith(t, extra); err == nil {
			t.Fatalf("invalid override accepted: %s", extra)
		}
	}
}

func TestLoadRFPDisciplineRejectsUnknownDriver(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: I, when: [{colour: [red]}], provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), `unknown driver "colour"`) {
		t.Fatalf("want unknown driver error, got %v", err)
	}
}

func TestLoadRFPDisciplineRejectsBadPosition(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: "Y", provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "must be I, O, X") {
		t.Fatalf("want position value error, got %v", err)
	}
}

func TestLoadRFPRejectsUnverifiedStatutoryInWhen(t *testing.T) {
	thresholds := `thresholds:
  - {key: dbp, kind: statutory, text: t, source: s, verified: false}
`
	err := parseWith(t, thresholds+`positions:
  - {key: x, label: X, kind: technical, value: I, threshold: dbp, when: [{class: ["2"]}], suggest: [{class: ["2"]}], provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "cannot select scope") {
		t.Fatalf("an unverified statutory rule must not select scope, got %v", err)
	}
	err = parseWith(t, thresholds+`positions:
  - {key: x, label: X, kind: technical, value: I, threshold: dbp, when: [{regulated_design: ["yes"]}], provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "suggest") {
		t.Fatalf("an unverified statutory rule needs suggest, got %v", err)
	}
	if err := parseWith(t, thresholds+`positions:
  - {key: x, label: X, kind: technical, value: I, threshold: dbp, when: [{regulated_design: ["yes"]}], suggest: [{class: ["2"]}], provenance: [{kind: practice, ref: r}]}
`); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRFPRejectsScaleOnTechnicalItem(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: {value: I, when: [{scale: [complex]}], else: O}, provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "must not depend on scale") {
		t.Fatalf("want scale error, got %v", err)
	}
	err = parseWith(t, `positions:
  - {key: x, label: X, kind: technical, simple: I, complex: O, provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "one value cell") {
		t.Fatalf("technical items take no simple/complex columns, got %v", err)
	}
}

func TestLoadRFPRejectsExclusionWithoutResponsible(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: X, provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "responsible") {
		t.Fatalf("want responsible error, got %v", err)
	}
}

func TestLoadRFPRejectsItemIncludedAndExcluded(t *testing.T) {
	body := strings.Replace(minimalDiscipline, "shown_when: Always}", "shown_when: Always, items: [x]}", 1) + `positions:
  - {key: x, label: X, kind: technical, value: X, responsible: client, provenance: [{kind: practice, ref: r}]}
`
	_, err := ParseRFPDiscipline([]byte(body), realRFPSkeleton(t))
	if err == nil || !strings.Contains(err.Error(), "always X") {
		t.Fatalf("want included-and-excluded error, got %v", err)
	}
}

func TestLoadRFPRequiresProvenanceKind(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: I, provenance: [{kind: hunch, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("want provenance error, got %v", err)
	}
	err = parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: I, provenance: [{kind: client_decision, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "why") {
		t.Fatalf("a client decision needs a why, got %v", err)
	}
}

func TestLoadRFPConditionalCellAndIfUnknown(t *testing.T) {
	err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: {value: O, when: [{flag: [community]}], else: X}, responsible: client, if_unknown: option, provenance: [{kind: practice, ref: r}]}
`)
	if err == nil || !strings.Contains(err.Error(), "assumption") {
		t.Fatalf("if_unknown option needs assumption wording, got %v", err)
	}
	if err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: {value: O, when: [{flag: [community]}], else: X}, responsible: client, if_unknown: option, assumption: a, provenance: [{kind: practice, ref: r}]}
`); err != nil {
		t.Fatal(err)
	}
	if err := parseWith(t, `positions:
  - {key: x, label: X, kind: technical, value: {value: O, when: [{flag: [no_such_flag]}], else: X}, responsible: client, provenance: [{kind: practice, ref: r}]}
`); err == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestLoadRFPRejectsUnknownField(t *testing.T) {
	if err := parseWith(t, "colour: red\n"); err == nil {
		t.Fatal("unknown top-level field accepted")
	}
}

func TestScopeScheduleCannotHideUnconfirmedAllocations(t *testing.T) {
	row := `scope_schedule:
  - {key: matrix, label: Interface, text: Name both owners, provenance: [{kind: practice, ref: test}]}
`
	if err := parseWith(t, row); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"when: [{flag: [heritage]}], ", "replace: true, ", "module: existing, "} {
		if err := parseWith(t, strings.Replace(row, "key: matrix, ", "key: matrix, "+change, 1)); err == nil {
			t.Fatal("conditional or replacement allocation accepted")
		}
	}
	if err := parseWith(t, "scope_schedule:\n  - {key: matrix, label: Interface, text: Name both owners}\n"); err == nil {
		t.Fatal("allocation without provenance accepted")
	}
}

func TestRealRFPCatalogue(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.RFPSkeleton(); !ok {
		t.Fatal("skeleton missing")
	}
	for _, id := range cat.RFPDisciplines() {
		d, ok := cat.RFP(id)
		if !ok {
			t.Fatalf("%s discipline missing", id)
		}
		if d.Status != "draft" {
			t.Fatalf("%s: seed-derived files stay draft until the owner reviews them", id)
		}
		byID, ok := cat.RFPForDiscipline(d.DisciplineIDs[0])
		if !ok || byID.ID != id {
			t.Fatalf("%s not found by discipline id", id)
		}
	}
	if d, ok := cat.RFPSuggest("Town Planner"); !ok || d.ID != "town-planning" {
		t.Fatal("title suggestion for a town planner failed")
	}
	if d, ok := cat.RFPSuggest("Geotechnical"); !ok || d.ID != "geotechnical" {
		t.Fatal("encoded geotechnical RFP must be offered")
	}
	if _, ok := cat.RFPSuggest("Acoustic"); ok {
		t.Fatal("unencoded packages must keep the generic RFP")
	}
}

func TestRFPSuggestPhrasesAndAmbiguity(t *testing.T) {
	cat := &Catalog{rfp: rfpData{disciplines: map[string]RFPDiscipline{
		"fire-engineering": {ID: "fire-engineering", TitleAliases: []string{"fire engineering", "fire engineer"}},
		"fire-services":    {ID: "fire-services", TitleAliases: []string{"fire services", "sprinkler"}},
		"access":           {ID: "access", TitleAliases: []string{"access consultant", "accessibility"}},
	}}}
	for _, c := range []struct{ title, want string }{
		{"Fire-services design (sprinklers)", "fire-services"},
		{"Fire Engineering / performance solutions", "fire-engineering"},
		{"Access consultant", "access"},
		{"Fire engineering and fire services", ""},
		{"Access road designer", ""},
	} {
		got, ok := cat.RFPSuggest(c.title)
		if ok != (c.want != "") || ok && got.ID != c.want {
			t.Fatalf("%q: got %q, %v; want %q", c.title, got.ID, ok, c.want)
		}
	}
}

func TestLoadRFPReviewedNeedsOwnerRecord(t *testing.T) {
	body := strings.Replace(minimalDiscipline, "status: draft", "status: reviewed", 1)
	if _, err := ParseRFPDiscipline([]byte(body), realRFPSkeleton(t)); err == nil || !strings.Contains(err.Error(), "reviewed_by") {
		t.Fatalf("reviewed without the owner's record: %v", err)
	}
	body = strings.Replace(minimalDiscipline, "status: draft", "status: reviewed\nreviewed_by: Owner\nreviewed_on: \"2026-10-10\"", 1)
	if _, err := ParseRFPDiscipline([]byte(body), realRFPSkeleton(t)); err != nil {
		t.Fatal(err)
	}
}

// Owner decision D3: a discipline sets defaults for its RFP's terms, never
// for a project fact that lives in Profile.
func TestLoadRFPFieldDefaultsAreTermsOnly(t *testing.T) {
	if err := parseWith(t, "field_defaults:\n  pi: \"$5 million\"\n"); err != nil {
		t.Fatalf("a discipline may set its own PI default: %v", err)
	}
	if err := parseWith(t, "field_defaults:\n  client: Someone\n"); err == nil || !strings.Contains(err.Error(), "Profile") {
		t.Fatalf("client lives in Profile: %v", err)
	}
}

func TestSkeletonProfileHomes(t *testing.T) {
	s := realRFPSkeleton(t)
	homes := map[string]string{}
	for _, f := range s.Fields {
		homes[f.Key] = f.Profile
	}
	for key, want := range map[string]string{"client": "proj.client", "address": "site.address", "lot_dp": "site.lot", "zone": "proj.zone", "pi": "", "agreement": "", "design_loads": ""} {
		if homes[key] != want {
			t.Fatalf("%s home = %q, want %q", key, homes[key], want)
		}
	}
	bad := strings.Replace(mustRead(t, "../../knowledge/rfp/skeleton.yaml"), "group: issue, help: \"Name the agreement", "group: issue, profile: proj.agreement, help: \"Name the agreement", 1)
	if _, err := parseRFPSkeleton([]byte(bad)); err == nil {
		t.Fatal("an issue field cannot have a profile home")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
