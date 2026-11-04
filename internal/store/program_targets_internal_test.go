package store

import (
	"strings"
	"testing"

	engine "sitewise/internal/program"
)

// Walsh brief, 2026-10-27: "DA lodged by mid-2026; construction start early
// 2027; maximum 18 months on site". Each target sits beside the computed
// date; the program's dates do not move.
func TestOwnerTargetsBesideTheProgram(t *testing.T) {
	st := ProgramState{
		Activities: []ProgramActivity{
			{ID: "lodge", Kind: "activity", SourceKey: "fragment:fr.da-lodgement:project:lodge"},
			{ID: "build", Kind: "stage", CatalogueStageID: "construction"},
		},
		Computed: engine.Output{Activities: engine.RowTimes{
			{ID: "lodge", Start: "2026-11-02", Finish: "2026-11-06"},
			{ID: "build", Start: "2027-03-01", Finish: "2028-01-31"},
		}},
	}
	got := ownerTargets(st, map[string]string{
		"proj.target_da_lodgement":             "mid-2026",
		"proj.target_construction_start":       "early 2027",
		"proj.max_construction_months":         "18 months",
		"proj.preliminary_construction_months": "14–16 months",
	})
	by := map[string]ProgramTarget{}
	for _, g := range got {
		by[g.Key] = g
	}
	if g := by["proj.target_da_lodgement"]; g.Status != "later" || g.Programmed != "6 November 2026" || !strings.Contains(g.Message, "31 August 2026") {
		t.Fatalf("DA lodgement %+v", g)
	}
	if g := by["proj.target_construction_start"]; g.Status != "met" || g.Programmed != "1 March 2027" {
		t.Fatalf("construction start %+v", g)
	}
	if g := by["proj.max_construction_months"]; g.Status != "within" || !strings.Contains(g.Programmed, "11.") {
		t.Fatalf("maximum on site %+v", g)
	}
	if g := by["proj.preliminary_construction_months"]; g.Status != "shorter" {
		t.Fatalf("builder estimate %+v", g)
	}
	// Walsh's stored program, 2026-10-27: about six weeks of construction
	// against the builder's 14–16 months is shorter, not within.
	short := st
	short.Computed.Activities = engine.RowTimes{{ID: "lodge", Start: "2026-11-02", Finish: "2026-11-06"}, {ID: "build", Start: "2027-04-27", Finish: "2027-06-07"}}
	if out := ownerTargets(short, map[string]string{"proj.preliminary_construction_months": "14–16 months", "proj.max_construction_months": "18 months"}); len(out) != 2 || out[0].Status != "within" || out[1].Status != "shorter" {
		t.Fatalf("short program %+v", out)
	}
	if out := ownerTargets(st, map[string]string{"proj.target_da_lodgement": "soon"}); len(out) != 1 || out[0].Status != "unknown" {
		t.Fatalf("an unreadable target %+v", out)
	}
}
