package intake_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

// F40: a markdown town planning fee proposal and SEE filed with no
// discipline, and a BCA assessment report by a registered certifier filed as
// Certifier, because Jev saw only the bare labels. The markdown discipline
// question now describes these three options in one sentence each.
func TestMarkdownDisciplineOptionsDescribePlannerBCAAndCertifier(t *testing.T) {
	path := filepath.Join(repoRoot(t), "testing", "synthetic", "scenarios", "ST02-base", "corpus", "BCA assessment report FBR-26-118.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var runs []identity.Run
	for _, line := range strings.Split(string(body), "\n") {
		runs = append(runs, identity.Run{Text: line})
	}
	d := intake.NewDraft(draftCatalog(t), filepath.Base(path), identity.Text{Format: "md", Runs: runs}, nil, nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("no Jev call")
	}
	if !strings.HasSuffix(call.QuestionVersion, "+markdown-3") {
		t.Fatalf("question version %q: criteria changed, version must change", call.QuestionVersion)
	}
	q, ok := call.Questions[intake.FieldDiscipline]
	if !ok {
		t.Fatal("discipline settled without Jev")
	}
	raw, _ := json.Marshal(q.Criteria)
	var criteria map[string]any
	if err := json.Unmarshal(raw, &criteria); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"consultant.town_planner": "statements of environmental effects",
		"consultant.bca":          "BCA assessment",
		"consultant.certifier":    "certificates",
	} {
		got, _ := criteria[id].(string)
		if !strings.Contains(got, want) {
			t.Errorf("%s criterion %q lacks %q", id, got, want)
		}
	}
	if got, _ := criteria["consultant.architect"].(string); got != "Architect" {
		t.Errorf("undescribed options keep their label: %q", got)
	}
}

// PDF filings keep the label-only question, so recorded replays still match.
func TestPDFDisciplineOptionsStayLabels(t *testing.T) {
	d := intake.NewDraft(draftCatalog(t), "report.pdf", identity.Text{Format: "pdf", TextLayer: true}, nil, nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("no Jev call")
	}
	raw, _ := json.Marshal(call.Questions[intake.FieldDiscipline].Criteria)
	var criteria map[string]any
	if err := json.Unmarshal(raw, &criteria); err != nil {
		t.Fatal(err)
	}
	if got, _ := criteria["consultant.town_planner"].(string); got != "Town Planner" {
		t.Fatalf("pdf criterion changed: %q", got)
	}
}

// F49: FS-101 (a fire pump room layout by "Calder Fire and Hydraulic
// Engineers") filed as Fire Engineer because both options were bare labels.
// The markdown question now says fire services design drawings are Fire
// Services and keeps Fire Engineer for fire engineering reports and
// performance solutions.
func TestMarkdownDisciplineOptionsTellFireServicesFromFireEngineer(t *testing.T) {
	path := filepath.Join(repoRoot(t), "testing", "synthetic", "scenarios", "ST06-base", "corpus", "FS-101 Rev A - Basement fire pump room layout.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var runs []identity.Run
	for _, line := range strings.Split(string(body), "\n") {
		runs = append(runs, identity.Run{Text: line})
	}
	d := intake.NewDraft(draftCatalog(t), filepath.Base(path), identity.Text{Format: "md", Runs: runs}, nil, nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("no Jev call")
	}
	if !strings.HasSuffix(call.QuestionVersion, "+markdown-3") {
		t.Fatalf("question version %q: criteria changed, version must change", call.QuestionVersion)
	}
	q, ok := call.Questions[intake.FieldDiscipline]
	if !ok {
		t.Fatal("discipline settled without Jev")
	}
	raw, _ := json.Marshal(q.Criteria)
	var criteria map[string]any
	if err := json.Unmarshal(raw, &criteria); err != nil {
		t.Fatal(err)
	}
	services, _ := criteria["consultant.fire_services"].(string)
	for _, want := range []string{"hydrant", "sprinkler", "pump", "FS-"} {
		if !strings.Contains(services, want) {
			t.Errorf("fire services criterion %q lacks %q", services, want)
		}
	}
	engineer, _ := criteria["consultant.fire_engineer"].(string)
	for _, want := range []string{"fire engineering report", "performance solution"} {
		if !strings.Contains(engineer, want) {
			t.Errorf("fire engineer criterion %q lacks %q", engineer, want)
		}
	}
}
