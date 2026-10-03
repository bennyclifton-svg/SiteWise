package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestPagedManufacturerDrawing(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
		run("Drg No.", 1028, 21, 20, 8), run("REV.", 1115, 21, 14, 8), run("Page:", 1145, 21, 15, 8),
		run("X123456-01", 1058, 12, 38, 9), run("1", 1125, 12, 4, 9), run("2/3", 1153, 11, 10, 9),
		run("Project Title:", 1028, 41, 33, 8), run("LIFT LAYOUT DRAWING", 1075, 32, 79, 9),
		run("HR05", 300, 400, 20, 8), run("STAINLESS STEEL", 340, 400, 90, 8),
	}}
	want := map[string]string{"number": "X123456-01", "revision": "1", "title": "LIFT LAYOUT DRAWING"}
	for _, r := range intake.Decide(intake.Harvest("", text)) {
		if v, ok := want[r.Field]; ok && (!r.Settled || r.Display != v) {
			t.Errorf("%+v want %s", r, v)
		}
	}
	text.Runs[2].Source.Page = 2
	for _, c := range intake.Harvest("", text) {
		if c.Provenance.BlockTitle || c.Provenance.OwnNumber {
			t.Fatalf("incomplete/cross-page grid admitted: %+v", c)
		}
	}
}
