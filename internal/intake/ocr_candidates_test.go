package intake_test

import (
	"fmt"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

// Measured OCR boxes from CC-06/CC-07: the small caption and larger title
// have different baselines, and the drawing number is split across two runs.
func ocrTitleBlock(sheet, level, revision string) identity.Text {
	run := func(s string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: s, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	return identity.Text{Format: "pdf", OCR: true, PageCount: 1, Runs: []identity.Run{
		run("DRAWING NAME", 2052.64, 150.88, 56.16, 5.28),
		run("LEVEL "+level, 2123.68, 149.44, 45.12, 8.64),
		run("DWG.No. CC-", 2180.32, 65.44, 59.04, 8.64),
		run(sheet, 2249.92, 65.92, 12.48, 8.64),
		run("REVISION "+revision, 2279.68, 64.48, 46.08, 8.64),
		run("A210", 300, 500, 35, 8), run("12230", 400, 450, 35, 8),
	}}
}

func TestOCRFragmentedDrawingIdentity(t *testing.T) {
	for _, tc := range []struct{ sheet, level, revision string }{{"06", "2", "F"}, {"07", "3", "E"}} {
		t.Run(tc.sheet, func(t *testing.T) {
			text := ocrTitleBlock(tc.sheet, tc.level, tc.revision)
			filename := fmt.Sprintf("1115 CC-%s LEVEL 0%s %s.pdf", tc.sheet, tc.level, tc.revision)
			results := intake.Decide(intake.Harvest(filename, text))
			for field, want := range map[string]string{"number": "CC-" + tc.sheet, "title": "LEVEL " + tc.level, "revision": tc.revision} {
				found := false
				for _, r := range results {
					if r.Field == field {
						found = r.Settled && r.Display == want
						if !found {
							t.Errorf("%s: %+v; want %s", field, r, want)
						}
					}
				}
				if !found {
					t.Errorf("%s not recovered", field)
				}
			}
		})
	}
}

func TestOCRDoesNotJoinUnrelatedNumber(t *testing.T) {
	for _, change := range []func(*identity.Text){
		func(v *identity.Text) { v.Runs[3].Source.Y += 30 },
		func(v *identity.Text) { v.Runs[3].Source.Page = 2 },
		func(v *identity.Text) { v.Runs[3].Source.X += 100 },
		func(v *identity.Text) { v.Runs = append(v.Runs, v.Runs[3]) },
	} {
		text := ocrTitleBlock("06", "2", "F")
		change(&text)
		for _, c := range intake.Harvest("scan.pdf", text) {
			if c.Field == "number" && c.Display == "CC-06" {
				t.Fatal("joined unrelated or ambiguous number")
			}
		}
	}
}
