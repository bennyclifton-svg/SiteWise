package procurement

import (
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

// With a program, a services package's default stages are the program
// stages whose catalogue stage is a sub-stage DefaultStages would copy, in
// program order, with the program's labels (program plan 7.6).
func TestProgramDefaultStagesReadThePlan(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	plan := []ProgramStage{
		{ID: "s1", Label: "Investigation", CatalogueStageID: "investigation"},
		{ID: "s2", Label: "Documentation for approval", CatalogueStageID: "approval_documentation"},
		{ID: "s3", Label: "Sketch design", CatalogueStageID: "concept_design"},
		{ID: "s4", Label: "My stage", CatalogueStageID: "other"},
		{ID: "s5", Label: "Site inspections", CatalogueStageID: "construction_stage_inspections"},
	}
	p := Package{Kind: "services", Title: "Architect", Novation: NovationOption}
	got := ProgramDefaultStages(p, cat, plan, 7)
	var labels []string
	for i, s := range got {
		labels = append(labels, s.Label)
		if s.Ordinal != i || s.StageRevision != 7 || s.Origin != "calculation" || ValidateStage(s, p, cat) != nil {
			t.Fatalf("stage %+v", s)
		}
	}
	if strings.Join(labels, ",") != "Documentation for approval,Sketch design,Site inspections" {
		t.Fatalf("labels %v", labels)
	}
	if got[0].ProgramStageID != "s2" || got[0].StageID != "approval_documentation" || got[0].NovationPhase != "pre" || got[2].NovationPhase != "post" {
		t.Fatalf("stages %+v", got)
	}
	// Not novated: no side; works packages get none.
	if s := ProgramDefaultStages(Package{Kind: "services", Title: "X"}, cat, plan, 1); s[0].NovationPhase != "none" {
		t.Fatalf("phase %+v", s[0])
	}
	if s := ProgramDefaultStages(Package{Kind: "works", WorksScope: "trade", Title: "X"}, cat, plan, 1); len(s) != 0 {
		t.Fatalf("works %+v", s)
	}
}
