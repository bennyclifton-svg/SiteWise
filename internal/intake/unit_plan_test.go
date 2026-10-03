package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestUnitPlanHeadingRetainsWrappedUnitsAndSubtitle(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "UNITS", Source: identity.Source{Page: 1, X: 970, Y: 643, Width: 102, Height: 42}},
		{Text: "201, 301,", Source: identity.Source{Page: 1, X: 942, Y: 614, Width: 158, Height: 42}},
		{Text: "501, 601", Source: identity.Source{Page: 1, X: 945, Y: 585, Width: 151, Height: 42}},
		{Text: "2 BEDROOM", Source: identity.Source{Page: 1, X: 944, Y: 531, Width: 149, Height: 30}},
		{Text: "Internal:", Source: identity.Source{Page: 1, X: 979, Y: 435, Width: 75, Height: 25}},
	}}
	got := intake.Harvest("renamed marketing.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "UNITS 201, 301, 501, 601 2 BEDROOM"); !ok {
		t.Fatal("wrapped printed unit-plan heading omitted")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatalf("filename settled over printed unit heading: %+v", r)
	}
	text.Runs[3].Source.Page = 2
	if _, ok := findCandidate(intake.Harvest("other.pdf", text), intake.FieldTitle, "UNITS 201, 301, 501, 601 2 BEDROOM"); ok {
		t.Fatal("joined subtitle from another page")
	}
}
