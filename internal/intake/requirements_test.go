package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestRequirementsTitleAndRevisionNoHistory(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "PRINCIPAL'S PROJECT REQUIREMENTS", Source: identity.Source{Page: 1, X: 50, Y: 700, Width: 350, Height: 22}},
		{Text: "Project", Source: identity.Source{Page: 1, X: 50, Y: 640, Width: 70, Height: 12}},
		{Text: "Prepared by", Source: identity.Source{Page: 1, X: 50, Y: 400, Width: 100, Height: 10}},
		{Text: "Version 3", Source: identity.Source{Page: 1, X: 50, Y: 100, Width: 60, Height: 10}},
		{Text: "Revision No.", Source: identity.Source{Page: 2, X: 50, Y: 700, Width: 70, Height: 10}},
		{Text: "Date", Source: identity.Source{Page: 2, X: 150, Y: 700, Width: 30, Height: 10}},
		{Text: "Description", Source: identity.Source{Page: 2, X: 300, Y: 700, Width: 100, Height: 10}},
		{Text: "1", Source: identity.Source{Page: 2, X: 50, Y: 680, Width: 8, Height: 10}},
		{Text: "11 July 2015", Source: identity.Source{Page: 2, X: 150, Y: 680, Width: 90, Height: 10}},
		{Text: "3", Source: identity.Source{Page: 2, X: 50, Y: 640, Width: 8, Height: 10}},
		{Text: "07 October 2015", Source: identity.Source{Page: 2, X: 150, Y: 640, Width: 90, Height: 10}},
	}}
	got := intake.Harvest("Renamed PPR.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "PRINCIPAL'S PROJECT REQUIREMENTS"); !ok {
		t.Fatal("printed requirements heading omitted")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldDate); !r.Settled || r.Display != "07 October 2015" {
		t.Fatalf("date not paired to cover version: %+v", r)
	}
}
