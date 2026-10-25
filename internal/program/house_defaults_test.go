package program

import (
	"strings"
	"testing"
)

// chenHouse mirrors the Chen Residence: a new Class 1a house on a
// traditional DA route, its work items as the Profile holds them, no gross
// floor area, and three consultants with design stages.
func chenHouse() GenInput {
	systems := []string{"site.erosion-sediment-control", "site.earthworks", "substructure.footings",
		"substructure.termite-management", "site.stormwater-detention", "substructure.slab-on-ground",
		"structure.timber", "envelope.roof-coverings", "envelope.external-walls", "envelope.thermal-insulation",
		"envelope.windows-doors", "hydraulic.roof-drainage", "interiors.walls-linings", "interiors.ceilings",
		"interiors.wet-area-waterproofing", "interiors.internal-doors", "hydraulic.sanitary", "hydraulic.cold-water",
		"hydraulic.hot-water", "electrical.cabling-wiring", "electrical.lighting", "electrical.switchboards",
		"fire-active.smoke-alarms", "mechanical.air-conditioning", "interiors.tiling", "interiors.joinery",
		"interiors.painting", "interiors.floor-finishes", "site.landscape", "electrical.utility-supply",
		"site.services-connections"}
	items := []WorkItem{{ID: "geo", System: "site.ground-conditions", Action: "investigate", Title: "Geotechnical ground conditions"},
		{ID: "odd", System: "envelope.roof-coverings", Action: "investigate", Title: "Check the existing roof"}}
	for i, s := range systems {
		items = append(items, WorkItem{ID: "wi" + string(rune('a'+i/26)) + string(rune('a'+i%26)), System: s, Action: "new", Title: s})
	}
	stages := func(p string) []PackageStage {
		return []PackageStage{{ID: p + "1", CatalogueStageID: "concept_design", Label: "Concept design"},
			{ID: p + "2", CatalogueStageID: "approval_documentation", Label: "Approval documentation"},
			{ID: p + "3", CatalogueStageID: "construction_documentation", Label: "Construction documentation"},
			{ID: p + "4", CatalogueStageID: "construction_stage_inspections", Label: "Construction-stage inspections"}}
	}
	return GenInput{
		WorkTypes:  []string{"new"},
		Conditions: map[string]string{"planning": "da", "procurement_route": "traditional"},
		Values:     map[string]string{"ncc_class": "1a", "rise_in_storeys": "2", "state": "NSW"},
		WorkItems:  items,
		Packages: []Package{{ID: "arch", Title: "Architect", Kind: "services", Stages: stages("a")},
			{ID: "eco", Title: "Ecology", Kind: "services", Stages: stages("e")},
			{ID: "str", Title: "Structural", Kind: "services", Stages: stages("s")},
			{ID: "main", Title: "Main Works", Kind: "works"}},
	}
}

// Owner direction 25 October 2026: opening Program on a house shows a whole
// program at once. Consultants follow the stage's own activities (no row per
// consultant), an unknown floor area is taken as 250 m², investigations get
// default durations, and every activity has a duration so the chart draws.
func TestHouseOpensWithEveryDuration(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, chenHouse())

	if len(p.StageMap) != 12 {
		t.Fatalf("package stages still map to program stages: %d", len(p.StageMap))
	}
	for _, r := range p.Rows {
		if r.PackageID != "" {
			t.Fatalf("a row per consultant: %s", r.Label)
		}
		if r.Kind != KindStage && r.Duration == nil {
			t.Fatalf("no duration: %s (%s)", r.Label, r.SourceKey)
		}
	}

	concept := labelsUnder(p, "skeleton:sk.new-traditional-da:concept_design")
	want := []string{"Confirm the brief", "Prepare the concept design", "Client review of the concept", "Develop the design"}
	if strings.Join(concept, "|") != strings.Join(want, "|") {
		t.Fatalf("concept design lines %q", concept)
	}

	frame, ok := p.Row("fragment:fr.frame-timber-1a:wiag:walls")
	if !ok || frame.Duration == nil || !strings.Contains(assumed(frame), "250") {
		t.Fatalf("frame on assumed floor area: %+v %v", frame, frame.DurationSource)
	}
	geo := findRow(p, "Geotechnical ground conditions")
	if geo.Duration == nil || *geo.Duration != 20 || assumed(geo) == "" {
		t.Fatalf("geotechnical investigation %+v", geo)
	}
	odd := findRow(p, "Check the existing roof")
	if odd.Duration == nil || *odd.Duration != 10 || assumed(odd) == "" {
		t.Fatalf("other investigation %+v", odd)
	}

	out := Compute(p.Input(Calendar{}))
	if out.Status.Known != out.Status.Total || !out.Finish.Complete {
		t.Fatalf("status %+v finish %+v", out.Status, out.Finish)
	}
	for _, a := range out.Activities {
		if a.Unknown {
			t.Fatalf("undated row %s", a.ID)
		}
	}

	// A stated floor area replaces the assumption.
	in := chenHouse()
	in.Planning = map[string]string{"gross_floor_area": "280"}
	stated, _ := mustGenerate(t, cat, in).Row(frame.SourceKey)
	if assumed(stated) != "" || *stated.Duration < *frame.Duration {
		t.Fatalf("stated floor area %+v %v", stated, stated.DurationSource)
	}
}

func labelsUnder(p Plan, stage string) []string {
	var out []string
	for _, r := range p.Rows {
		if r.ParentKey == stage {
			out = append(out, r.Label)
		}
	}
	return out
}

func findRow(p Plan, label string) Row {
	for _, r := range p.Rows {
		if r.Label == label {
			return r
		}
	}
	return Row{}
}

func assumed(r Row) string {
	s, _ := r.DurationSource["assumed"].(string)
	return s
}
