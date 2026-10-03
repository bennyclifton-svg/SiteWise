package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestRightAlignedDrawingBlockIgnoresRegisterRevisions(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Runs: []identity.Run{
		run("Drawing Number:", 868, -758, 67, 11.45), run("A-0-11-01", 1003, -777, 53, 16.38),
		run("Revision:", 1068, -758, 36, 11.45), run("08", 1133, -780, 16, 19.1),
		run("Drawing Title", 868, -477, 50, 11.45), run("PROPOSED SITE PLAN", 993, -493, 155, 19.1),
		run("Current", -567, -126, 33, 13.65), run("Revision", -569, -137, 38, 13.65), run("04", -568, -160, 10, 13.65),
		run("Rev Description", 868, 745, 63, 11.45), run("Date", 1105, 745, 17, 11.45),
		run("07 FOR COORDINATION", 868, 651, 100, 10.92), run("10/12/2025", 1105, 651, 40, 10.92),
		run("08 TENDER ADDENDUM", 868, 640, 100, 10.92), run("22/01/2026", 1105, 640, 40, 10.92),
		run("10000", 800, 400, 40, 10),
	}}
	want := map[string]string{"number": "A-0-11-01", "revision": "08", "title": "PROPOSED SITE PLAN", "date": "22/01/2026"}
	for _, r := range intake.Decide(intake.Harvest("A-0-11-01-PROPOSED SITE PLAN[08].pdf", text)) {
		if !r.Settled || r.Display != want[r.Field] {
			t.Errorf("%s: %+v; want %s", r.Field, r, want[r.Field])
		}
	}
}
