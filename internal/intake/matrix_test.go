package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestResponsibilityMatrixPrintedHeadingChallengesFilename(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "1", Source: identity.Source{Page: 1, X: 518, Y: 68, Width: 6, Height: 14.65}},
		{Text: "North Street Development Application", Source: identity.Source{Page: 1, X: 72, Y: 714, Width: 444, Height: 23.56}},
		{Text: "Responsibility Matrix DA-123/2024", Source: identity.Source{Page: 1, X: 72, Y: 692.64, Width: 255, Height: 23.56}},
		{Text: "Condition Number", Source: identity.Source{Page: 1, X: 72, Y: 600, Width: 80, Height: 14.65}},
		{Text: "Condition", Source: identity.Source{Page: 1, X: 170, Y: 600, Width: 80, Height: 14.65}},
		{Text: "Responsibility", Source: identity.Source{Page: 1, X: 430, Y: 600, Width: 80, Height: 14.65}},
	}}
	got := intake.Harvest("DA Matrix renamed.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "North Street Development Application Responsibility Matrix DA-123/2024"); !ok {
		t.Fatal("missing complete printed matrix heading")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatalf("filename settled despite a different printed heading: %+v", r)
	}
}
