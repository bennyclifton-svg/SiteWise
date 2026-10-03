package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestCADTitleBlockAndUpwardRevisionHistory(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
		run("TITLE:", 2071, 125, 25, 11), run("MECHANICAL SERVICES - GENERAL NOTES", 2078, 97, 162, 29),
		run("DRAWING NO:", 2163, 55, 54, 11), run("M01", 2163, 37, 24, 15), run("REVISION:", 2292, 55, 39, 11), run("C", 2312, 35, 12, 15),
		run("REV:", 2083, 506, 20, 11), run("DESCRIPTION:", 2171, 506, 54, 11), run("DATE:", 2312, 506, 24, 11),
		run("B", 2089, 606, 9, 11), run("25/10/2023", 2298, 605, 53, 13), run("C", 2089, 626, 9, 11), run("01/12/2023", 2298, 625, 53, 14),
		run("Drawing Title:", 100, 700, 80, 11), run("GENERAL NOTES", 185, 700, 80, 11),
	}}
	want := map[string]string{"number": "M01", "revision": "C", "title": "MECHANICAL SERVICES - GENERAL NOTES", "date": "01/12/2023"}
	for _, r := range intake.Decide(intake.Harvest("", text)) {
		if r.Display != want[r.Field] || !r.Settled {
			t.Errorf("%s: %+v", r.Field, r)
		}
	}
	text.Runs[11].Source.Page = 2
	for _, r := range intake.Decide(intake.Harvest("", text)) {
		if r.Field == "date" && r.Settled {
			t.Fatalf("paired across pages: %+v", r)
		}
	}
	text.Runs[11].Source.Page = 1
	for i := 9; i <= 12; i++ {
		text.Runs[i].Source.Y -= 240
	}
	for _, r := range intake.Decide(intake.Harvest("", text)) {
		if r.Field == "date" && (!r.Settled || r.Display != want["date"]) {
			t.Fatalf("downward history lost: %+v", r)
		}
	}
}
