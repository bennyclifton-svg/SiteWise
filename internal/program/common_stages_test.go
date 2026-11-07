package program

import (
	"strings"
	"testing"
)

// Owner direction 7 November 2026: every route runs brief and concept,
// schematic design, planning and design development before the tender. On a
// traditional route the library's construction documentation is the
// developed set the tender is priced on; on design and construct the client
// team takes the design to about 70-80% and the contractor finalises it.
func TestDocumentationStageFollowsTheRoute(t *testing.T) {
	cat := testCatalog(t, nil, "")
	gen := func(route string) Plan {
		return mustGenerate(t, cat, GenInput{
			WorkTypes:  []string{"new"},
			Conditions: map[string]string{"planning": "da", "procurement_route": route},
			Values:     map[string]string{"ncc_class": "2", "rise_in_storeys": "4", "state": "NSW"},
		})
	}
	stageOf := func(p Plan, fragment string) string {
		for _, r := range p.Rows {
			if r.FragmentID == fragment {
				return strings.TrimPrefix(r.ParentKey, "skeleton:"+p.SkeletonID+":")
			}
		}
		return ""
	}

	trad := gen("traditional")
	if got := stageOf(trad, "fr.construction-docs-2"); got != "design_development" {
		t.Fatalf("traditional construction documentation in %q", got)
	}
	if got := stageOf(trad, "fr.design-development-dc-2"); got != "" {
		t.Fatalf("contractor-route design development on a traditional job, in %q", got)
	}

	dc := gen("design_construct")
	if got := stageOf(dc, "fr.construction-docs-2"); got != "design_finalisation" {
		t.Fatalf("design and construct construction documentation in %q", got)
	}
	if got := stageOf(dc, "fr.design-development-dc-2"); got != "design_development" {
		t.Fatalf("design and construct design development in %q", got)
	}
	// No sourced duration yet: the default length, said so, never blank.
	for _, r := range dc.Rows {
		if r.FragmentID == "fr.design-development-dc-2" && r.Duration == nil {
			t.Fatalf("undated design development %+v", r)
		}
	}
	times := timesByID(t, dc)
	key := func(id string) string { return "skeleton:" + dc.SkeletonID + ":" + id }
	order := []string{"brief_concept", "schematic_design", "planning", "design_development", "tender_documentation", "design_finalisation", "delivery"}
	for i := 1; i < len(order); i++ {
		if times[key(order[i])].ES < times[key(order[i-1])].EF {
			t.Errorf("%s starts before %s finishes", order[i], order[i-1])
		}
	}
}
