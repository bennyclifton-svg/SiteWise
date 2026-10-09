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

func TestRealRFPCatalogue(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.RFPSkeleton(); !ok {
		t.Fatal("skeleton missing")
	}
	for _, id := range []string{"architecture", "town-planning", "traffic", "civil", "structural"} {
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
	if _, ok := cat.RFPSuggest("Geotechnical"); ok {
		t.Fatal("tranche 2/3 packages must keep the generic RFP")
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
