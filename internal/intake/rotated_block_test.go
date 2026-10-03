package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func rotatedRun(text string, x, y, width, height float64) identity.Run {
	return identity.Run{Text: text, Source: identity.Source{Page: 1, Rotation: 270, X: x, Y: y, Width: width, Height: height}}
}

func TestRotatedDrawingTitleKeepsWrappedSuffix(t *testing.T) {
	for _, suffix := range []string{"LEGEND", "2"} {
		text := identity.Text{Runs: []identity.Run{
			rotatedRun("Drawing Title:", 1351, 2056, 7.5, 39),
			rotatedRun("MECHANICAL SERVICES PLAN", 1363, 2056, 13, 226),
			rotatedRun(suffix, 1379, 2056, 13, 46),
			rotatedRun("PROJECT NAME", 1200, 2056, 13, 120),
			rotatedRun("M-200", 1600, 2056, 13, 40),
		}}
		for _, r := range intake.Decide(intake.Harvest("M-200 [B].pdf", text)) {
			if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "MECHANICAL SERVICES PLAN "+suffix) {
				t.Fatalf("wrapped title %q: %+v", suffix, r)
			}
		}
	}
}

func TestRotatedIssueTableUsesCurrentRevisionNotLatestDate(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		rotatedRun("Issue", 670, 2057, 7.5, 15),
		rotatedRun("Date", 670, 2091, 7.5, 13),
		rotatedRun("A", 681, 2055, 7.5, 4),
		rotatedRun("09/06/15", 681, 2078, 7.5, 25),
		rotatedRun("B", 691, 2055, 7.5, 4),
		rotatedRun("08/07/15", 691, 2078, 7.5, 25),
	}}
	for rev, date := range map[string]string{"A": "09/06/15", "B": "08/07/15"} {
		for _, r := range intake.Decide(intake.Harvest("M-200 ["+rev+"].pdf", text)) {
			if r.Field == intake.FieldDate && (!r.Settled || r.Display != date) {
				t.Fatalf("revision %s: %+v", rev, r)
			}
		}
	}
}

func TestExplicitIdentityWinsOverInferredBlockAndHistory(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Title: BASEMENT PLAN"},
		{Text: "REV B 08/07/15"},
		rotatedRun("Drawing Title:", 1351, 2056, 7.5, 39),
		rotatedRun("MECHANICAL SERVICES", 1363, 2056, 13, 226),
		rotatedRun("Issue", 670, 2057, 7.5, 15),
		rotatedRun("Date", 670, 2091, 7.5, 13),
		rotatedRun("B", 681, 2055, 7.5, 4),
		rotatedRun("09/06/15", 681, 2078, 7.5, 25),
	}}
	for _, r := range intake.Decide(intake.Harvest("M-200 [B].pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "BASEMENT PLAN") {
			t.Fatalf("inferred block displaced inline title: %+v", r)
		}
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "08/07/15") {
			t.Fatalf("history displaced explicit issue statement: %+v", r)
		}
	}
}

func TestDrawingIdentityWithJobPrefixAndFourLineTitle(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Runs: []identity.Run{
		run("Drawing Number", 2171, 61, 62, 11), run("ME-2-301", 2180, 29, 80, 22),
		run("Job Number", 2075, 61, 45, 11), run("SYD3293", 2074, 29, 68, 22),
		run("Drawing Title", 2074, 303, 49, 11), run("MECHANICAL SERVICES", 2074, 275, 180, 22),
		run("RECREATION", 2074, 253, 100, 22), run("GROUND", 2074, 231, 70, 22), run("HVAC LAYOUT", 2074, 209, 110, 22),
		run("Revision Description", 2068, 1626, 85, 11), run("Initial Date", 2276, 1626, 50, 11),
		run("T1", 2078, 1608, 9, 14), run("RF 21.11.25", 2280, 1608, 60, 14),
		run("T2", 2078, 1594, 9, 14), run("RF 12.12.25", 2280, 1594, 60, 14),
	}}
	want := map[string]string{"number": "ME-2-301", "title": "MECHANICAL SERVICES RECREATION GROUND HVAC LAYOUT", "date": "12.12.25"}
	for _, r := range intake.Decide(intake.Harvest("SYD3293-ME-2-301-[T2].pdf", text)) {
		if v, ok := want[r.Field]; ok && (!r.Settled || r.Display != v) {
			t.Errorf("%s: %+v; want %s", r.Field, r, v)
		}
	}
}

func TestTitleGlyphBoxesMaySlightlyOverlap(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Title", Source: identity.Source{Page: 1, X: 2055.32, Y: 190.026, Width: 33.87, Height: 9.862}},
		{Text: "ELECTRICAL SERVICES", Source: identity.Source{Page: 1, X: 2055.08, Y: 169.211, Width: 180.827, Height: 20.923}},
		{Text: "LEGEND AND NOTES", Source: identity.Source{Page: 1, X: 2055.08, Y: 150.491, Width: 160.721, Height: 20.923}},
	}}
	for _, r := range intake.Decide(intake.Harvest("11049 E001 T1.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "ELECTRICAL SERVICES LEGEND AND NOTES") {
			t.Fatalf("%+v", r)
		}
	}
}

func TestCADPlotStampIsNotAnIssueDate(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "7/07/2015 3:53:34 PM, ISO A1 (841.00 x 594.00 MM), 1:1, COPYRIGHT - CONSULTANT"}}}
	for _, c := range intake.Harvest("S0201 (04).pdf", text) {
		if c.Field == intake.FieldDate {
			t.Fatalf("plot date harvested as issue date: %+v", c)
		}
	}
}

func TestDottedDrawingNumberAndRevisionIssueCaption(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Runs: []identity.Run{
		run("DRAWING No:", 1098, 35.6, 21.6, 5.5), run("S-2-1.00", 1099, 26.3, 21.8, 9.65),
		run("Z450", 400, 900, 30, 10), // steel coating reference, not the sheet number
		run("REV ISSUE:", 1138, 35.6, 18.2, 5.5), run("03", 1143, 26.8, 6.3, 9.65),
		run("DRAWING TITLE:", 986, 53, 26, 5.5), run("COVER SHEET", 986, 42.4, 40, 9.65),
		run("REV BY", 37, 25, 26, 8), run("DATE APPROVED BY", 273, 25, 57, 8),
		run("02", 40, 56, 4.5, 6.9), run("05.12.2025", 269, 56, 20, 6.9),
		run("03", 40, 66, 4.5, 6.9), run("16.12.2025", 269, 66, 20, 6.9),
	}}
	c := intake.Harvest("structural pack.pdf", text)
	for _, v := range c {
		if v.Field == intake.FieldRevision && v.Display == "BY" {
			t.Error("table heading became revision")
		}
	}
	for _, r := range intake.Decide(c) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "S-2-1.00") {
			t.Errorf("number: %+v", r)
		}
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "COVER SHEET") {
			t.Errorf("title: %+v", r)
		}
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "03") {
			t.Errorf("revision: %+v", r)
		}
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "16.12.2025") {
			t.Errorf("date: %+v", r)
		}
	}
	found := false
	for _, v := range c {
		if v.Field == intake.FieldNumber && v.Display == "S-2-1.00" {
			found = true
		}
	}
	if !found {
		t.Error("dotted number missing")
	}
}

func TestMergedIssueDateHeadingAndOverlappingNumber(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Runs: []identity.Run{
		run("Dwg No:", 2056, 87.5, 24.5, 10.86), run("H-103", 2055, 64.1, 44, 27.4), run("581006", 100, 500, 40, 10),
		run("Issue Date", 2055, 1003, 39.2, 10.86), run("C", 2056, 960, 4.6, 10.86), run("07/07/15", 2078, 960, 24.9, 10.86),
		run("D", 2056, 946, 4.6, 10.86), run("17/07/15", 2078, 946, 24.9, 10.86),
	}}
	for _, r := range intake.Decide(intake.Harvest("H-103 [D].pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "H-103") {
			t.Errorf("number: %+v", r)
		}
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "17/07/15") {
			t.Errorf("date: %+v", r)
		}
	}
}

func TestStackedJobAndDrawingCaptions(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Job No: Dwg No:", Source: identity.Source{Page: 1, Rotation: 90, X: 43.62, Y: 151.89, Width: 16.51, Height: 12.07}},
		{Text: "H-000", Source: identity.Source{Page: 1, Rotation: 90, X: 31.94, Y: 142.59, Width: 13.71, Height: 21.97}},
		{Text: "581006"},
	}}
	for _, r := range intake.Decide(intake.Harvest("H-000 [C].pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "H-000") {
			t.Fatalf("%+v", r)
		}
	}
}
