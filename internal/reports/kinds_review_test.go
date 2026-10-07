package reports

import (
	"encoding/json"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
	"strings"
	"testing"
)

func TestRFTWorkRequiresIncludedPackageScope(t *testing.T) {
	parent := works.Item{ID: "parent", IsGroup: true, Inclusion: "included"}
	child := works.Item{ID: "child", ParentID: parent.ID, Inclusion: "included"}
	rows := []ScopeRecord{{PackageID: "trade", ScopeContent: procurement.ScopeContent{WorkItemID: parent.ID, Inclusion: "excluded"}}}
	if workInPackage(child, []works.Item{parent, child}, rows, "trade") {
		t.Fatal("excluded responsibility included child work")
	}
	rows[0].Inclusion = "included"
	if !workInPackage(child, []works.Item{parent, child}, rows, "trade") {
		t.Fatal("inherited included responsibility lost")
	}
	if workInPackage(child, []works.Item{parent, child}, rows, "other") {
		t.Fatal("other package included")
	}
}
func TestWorkDescriptionUsesExactTypedTargetsAndExistingCondition(t *testing.T) {
	q, u := "12.5", "m2"
	w := works.Item{Title: "Retain partition", Action: "retain", ExistingConditionNote: "Damaged finish", Quantity: &q, Unit: &u, Target: works.Target{Values: []works.TargetValue{{Key: "rating", Value: json.RawMessage(`"FRL 60/60/60"`)}, {Key: "serial", Value: json.RawMessage(`9007199254740993`)}, {Key: "required", Value: json.RawMessage(`true`)}}, ClauseRefs: []works.ClauseRef{{ID: "cl.target", Version: 2}}}}
	text := workDescription(w, "Level 1")
	for _, want := range []string{"Existing condition: Damaged finish", "Quantity: 12.5 m2", "rating: FRL 60/60/60", "serial: 9007199254740993", "required: true", "Target clause cl.target v2"} {
		if !strings.Contains(text, want) {
			t.Fatal(want, text)
		}
	}
}
