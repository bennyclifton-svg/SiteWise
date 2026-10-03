package intake_test

import (
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestJobSheetFilenameNumber(t *testing.T) {
	for _, name := range []string{"15123_S0301_ground floor_Fmwk Plan-(04).pdf", "15123_S0012_Shoring_Details Sht 1-(04).pdf", "998877_A0100_Ground floor-(B).pdf"} {
		c := intake.Harvest(name, identity.Text{})
		got := fieldResult(t, intake.Decide(c), intake.FieldNumber)
		want := strings.Split(name, "_")[1]
		if !got.Settled || got.Display != want {
			t.Errorf("%s: got %+v, want %s", name, got, want)
		}
	}
}

func TestJobSheetFilenameDoesNotHidePrintedConflict(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Drawing No: S0999", Source: identity.Source{Page: 1}}}}
	got := fieldResult(t, intake.Decide(intake.Harvest("15123_S0301_ground floor_Fmwk Plan-(04).pdf", text)), intake.FieldNumber)
	if got.Settled && got.Display == "S0301" {
		t.Fatalf("filename overrode printed conflict: %+v", got)
	}
	for _, name := range []string{"15123 S0301 ground floor.pdf", "15123_S0301.pdf", "15123_S0301_notes.pdf"} {
		got := fieldResult(t, intake.Decide(intake.Harvest(name, identity.Text{})), intake.FieldNumber)
		if got.Settled {
			t.Errorf("ambiguous filename %s settled: %+v", name, got)
		}
	}
}
