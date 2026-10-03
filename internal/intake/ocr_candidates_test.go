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

func TestOCRInlineSpacedNumberCompletesTitleBlock(t *testing.T) {
	text := ocrTitleBlock("10", "7", "F")
	text.Runs[2].Text = "DwG.No. CC- 10"
	text.Runs[2].Source.X = 2180.32
	text.Runs[2].Source.Y = 64
	text.Runs[2].Source.Width = 76.8
	text.Runs[2].Source.Height = 9.12
	text.Runs = append(text.Runs[:3], text.Runs[4:]...)
	results := intake.Decide(intake.Harvest("1115 CC-10 LEVEL 07 F.pdf", text))
	for field, want := range map[string]string{"number": "CC-10", "title": "LEVEL 7", "revision": "F"} {
		r := fieldResult(t, results, field)
		if !r.Settled || r.Display != want {
			t.Errorf("%s: %+v; want %s", field, r, want)
		}
	}
	text.Runs[3].Source.Page = 2
	for _, c := range intake.Harvest("scan.pdf", text) {
		if c.Field == intake.FieldNumber && c.Provenance.OwnNumber {
			t.Fatal("revision on another page completed the block")
		}
	}
}

func TestOCRTitleAgreesWithFilenameExceptPrintedRevision(t *testing.T) {
	for _, tc := range []struct {
		name, filename, revision string
		page                     int
		want                     bool
	}{
		{"match", "1115 CC-13 EAST ELEVATION E.pdf", "Revision E", 1, true},
		{"different revision", "1115 CC-13 EAST ELEVATION E.pdf", "Revision F", 1, false},
		{"missing revision", "1115 CC-13 EAST ELEVATION E.pdf", "", 1, false},
		{"different page", "1115 CC-13 EAST ELEVATION E.pdf", "Revision E", 2, false},
		{"different title", "1115 CC-13 WEST ELEVATION E.pdf", "Revision E", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := identity.Text{Format: "pdf", OCR: true, Runs: []identity.Run{
				{Text: "DRAWING NAME", Source: identity.Source{Page: 1, X: 2053.12, Y: 148.96, Width: 55.68, Height: 5.28}},
				{Text: "EAST ELEVATION", Source: identity.Source{Page: 1, X: 2125.12, Y: 148.96, Width: 98.4, Height: 9.12}},
				{Text: "Dwe.No. CC- 13", Source: identity.Source{Page: 1, X: 2180.8, Y: 63.52, Width: 76.32, Height: 8.64}},
				{Text: tc.revision, Source: identity.Source{Page: tc.page, X: 2279.68, Y: 64, Width: 48, Height: 8.64}},
			}}
			r := fieldResult(t, intake.Decide(intake.Harvest(tc.filename, text)), intake.FieldTitle)
			if r.Settled != tc.want || (tc.want && r.Display != "EAST ELEVATION") {
				t.Fatalf("title %+v; settled want %v", r, tc.want)
			}
		})
	}
}

func TestOCRSectionTitleAtCaptionBoundary(t *testing.T) {
	for _, tc := range []struct{ sheet, caption, section string }{
		{"17", "DwG.No. CC- 17", "A"}, {"18", "Dwe.No. CC- 18", "B"},
	} {
		t.Run(tc.sheet, func(t *testing.T) {
			text := identity.Text{Format: "pdf", OCR: true, Runs: []identity.Run{
				{Text: "DRAWING NAME", Source: identity.Source{Page: 1, X: 2053.12, Y: 148.96000000000004, Width: 55.68000000000001, Height: 5.279999999999973}},
				{Text: "GENERAL SECTION " + tc.section + "~" + tc.section, Source: identity.Source{Page: 1, X: 2124.6400000000003, Y: 148.96000000000015, Width: 136.32, Height: 9.12}},
				{Text: tc.caption, Source: identity.Source{Page: 1, X: 2180.8, Y: 63.52, Width: 76.32, Height: 8.64}},
				{Text: "REVISION C", Source: identity.Source{Page: 1, X: 2279.68, Y: 64, Width: 48, Height: 8.64}},
			}}
			filename := "1115 CC-" + tc.sheet + " SECTION " + tc.section + " C.pdf"
			check := func(want bool) {
				t.Helper()
				r := fieldResult(t, intake.Decide(intake.Harvest(filename, text)), intake.FieldTitle)
				if r.Settled != want || (want && r.Display != text.Runs[1].Text) {
					t.Fatalf("title %+v; settled want %v", r, want)
				}
			}
			check(true)
			text.Runs[1].Source.X += .01
			check(false)
			text.Runs[1].Source.X -= .01
			text.Runs[3].Source.Page = 2
			check(false)
			if tc.sheet == "18" {
				text.Runs[3].Source.Page = 1
				filename = "1115 CC-19 SECTION B C.pdf"
				check(false)
				filename = "1115 CC-18 SECTION B C.pdf"
				text.OCR = false
				check(false)
			}
		})
	}
}
