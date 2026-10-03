package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestScheduleHeadingChallengesFilename(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "SCHEDULE OF FINISHES", Source: identity.Source{Page: 1, X: 150, Y: 700, Width: 250, Height: 20}},
		{Text: "Option 1 - Light", Source: identity.Source{Page: 1, X: 210, Y: 670, Width: 130, Height: 13}},
		{Text: "16 April, 2015", Source: identity.Source{Page: 1, X: 70, Y: 645, Width: 90, Height: 9}},
		{Text: "ITEM", Source: identity.Source{Page: 1, X: 70, Y: 620, Width: 30, Height: 12}},
		{Text: "DESCRIPTION", Source: identity.Source{Page: 1, X: 220, Y: 620, Width: 80, Height: 12}},
	}}
	got := intake.Harvest("Amended schedule - As at 20-4-2015.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "SCHEDULE OF FINISHES"); !ok {
		t.Fatal("printed schedule heading missing")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatal("renamed schedule filename settled despite printed heading")
	}
	text.Runs[0].Text = "COLUMN SCHEDULE AND DETAILS"
	for _, c := range intake.Harvest("renamed.pdf", text) {
		if c.Field == intake.FieldTitle && c.Provenance.Heading {
			t.Fatal("technical schedule sheet treated as standalone report cover")
		}
	}
}
