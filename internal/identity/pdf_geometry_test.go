package identity

import "testing"

func TestRequirementsCoverRetainsAmendmentSchedule(t *testing.T) {
	cover := []Run{{Text: "PRINCIPAL'S PROJECT REQUIREMENTS"}, {Text: "Project"}, {Text: "Version 3"}}
	if !probeControlPage(cover) {
		t.Fatal("requirements cover not probed")
	}
	history := []Run{{Text: "Specification Amendment Schedule"}, {Text: "Revision No."}, {Text: "Date"}, {Text: "Description"}}
	if !controlPage(history) || controlPage(history[1:]) || controlPage(history[:2]) {
		t.Fatal("amendment history must require its heading and control columns")
	}
}

func TestLargeBadgeDoesNotMergeWithSmallerHeading(t *testing.T) {
	chars := []rune("REPORT Heading")
	boxes := make([]charBox, len(chars))
	for i := 0; i < 6; i++ {
		boxes[i] = charBox{left: float64(i) * 20, right: float64(i)*20 + 18, bottom: 10, top: 50}
	}
	for i := 7; i < len(chars); i++ {
		boxes[i] = charBox{left: 125 + float64(i-7)*7, right: 131 + float64(i-7)*7, bottom: 32, top: 45}
	}
	got := pdfLines(chars, boxes)
	if len(got) != 2 {
		t.Fatalf("badge and heading merged: %+v", got)
	}
}

func TestOnlyTouchingSameLineFragmentsJoin(t *testing.T) {
	a := charBox{left: 10, right: 15, bottom: 20, top: 30}
	for _, tc := range []struct {
		name string
		b    charBox
		want bool
	}{
		{"touching", charBox{left: 15, right: 20, bottom: 20, top: 30}, true},
		{"next row", charBox{left: 15, right: 20, bottom: 5, top: 15}, false},
		{"separate cell", charBox{left: 50, right: 55, bottom: 20, top: 30}, false},
		{"reverse order", charBox{left: 5, right: 10, bottom: 20, top: 30}, false},
		{"invisible", charBox{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := touchingGlyphs(a, tc.b); got != tc.want {
				t.Fatalf("join=%v", got)
			}
		})
	}
}

func TestAssessmentCoverAndApprovalHistory(t *testing.T) {
	cover := []Run{{Text: "TRAFFIC AND PARKING IMPACT ASSESSMENT OF"}, {Text: "EXAMPLE DEVELOPMENT"}, {Text: "240648.01FB - 5 September 2025"}}
	if !probeControlPage(cover) {
		t.Fatal("assessment cover not probed")
	}
	history := []Run{{Text: "Document reference: 240648.01FB"}, {Text: "Status Issue Prepared By Checked By Approved By Date"}}
	if !controlPage(history) {
		t.Fatal("explicit approval history not retained")
	}
	if controlPage(history[1:]) {
		t.Fatal("ordinary status table treated as document control")
	}
}

func TestControlPageProbeRequiresSparseReportCover(t *testing.T) {
	report := []Run{{Text: "Project"}, {Text: "Civil Engineering Report"}, {Text: "13 March 2025"}}
	if !probeControlPage(report) {
		t.Fatal("report cover cannot reach its control page")
	}
	drawing := append(append([]Run{}, report...), Run{Text: "Project Number/Drawing Number"})
	if probeControlPage(drawing) {
		t.Fatal("drawing identity must not merge the next sheet")
	}
	dense := append([]Run{}, report...)
	for len(dense) < 21 {
		dense = append(dense, Run{Text: "body prose"})
	}
	if probeControlPage(dense) {
		t.Fatal("body page is not a sparse cover")
	}
	if controlPage([]Run{{Text: "Drawing Number"}, {Text: "Revision"}}) {
		t.Fatal("drawing sheet is not report control")
	}
	if !controlPage([]Run{{Text: "Document Information"}, {Text: "Revision"}}) {
		t.Fatal("explicit control page omitted")
	}
}

func TestApplicationCoverReadsExplicitAuthorisedRevisionHistory(t *testing.T) {
	cover := []Run{{Text: "Example redevelopment"}, {Text: "Development Application"}, {Text: "Prepared for Example Client"}, {Text: "March 2025"}}
	if !probeControlPage(cover) {
		t.Fatal("application cover omitted")
	}
	history := []Run{{Text: "Rev A Issued 25 October 2024 Authorised by NW"}, {Text: "Rev B Issued 15 November 2024 Authorised by NW"}, {Text: "Rev D DA Issue 03 March 2025 Authorised by NW"}}
	if !controlPage(history) {
		t.Fatal("explicit authorised issue history omitted")
	}
	if controlPage(history[:1]) {
		t.Fatal("one body reference mistaken for control history")
	}
	if controlPage([]Run{{Text: "Rev A Rev B Rev C"}, {Text: "Issued Issued Issued"}, {Text: "Authorised by NW"}}) {
		t.Fatal("unstructured mentions mistaken for complete history")
	}
}

func TestRepeatedCoverCanReachThirdPageControl(t *testing.T) {
	cover := []Run{{Text: "Hazardous Materials"}, {Text: "Assessment"}, {Text: "Project No:"}, {Text: "ABC1234"}, {Text: "Date: 18-09-2025"}}
	repeated := append([]Run{{Text: "Publisher contact details"}}, cover...)
	if !repeatedReportCover(cover, repeated) {
		t.Fatal("repeated sparse cover rejected")
	}
	if repeatedReportCover(cover, []Run{{Text: "Assessment"}, {Text: "Ordinary body prose"}, {Text: "Other project"}}) {
		t.Fatal("body page accepted")
	}
	if repeatedReportCover(cover, append(repeated, Run{Text: "Drawing Number"})) {
		t.Fatal("drawing sheet accepted")
	}
	history := []Run{{Text: "Document Number"}, {Text: "ABC1234"}, {Text: "Quality Information"}, {Text: "Distribution"}, {Text: "Issue"}, {Text: "Revision Issued To"}, {Text: "Date"}, {Text: "Prepared"}, {Text: "Reviewed"}}
	if !controlPage(history) {
		t.Fatal("explicit distribution control omitted")
	}
	if controlPage(history[2:]) {
		t.Fatal("table without document identity accepted")
	}
}

func TestExplicitReportMetadataExcludesDisclaimer(t *testing.T) {
	runs := []Run{{Text: "Disclaimer about other reports and dates"}, {Text: "Report: Ecology Assessment"}, {Text: "Prepared for: Client"}, {Text: "Prepared by: Ecologist"}, {Text: "Project No: JOB123"}, {Text: "Version: Final v1.0"}}
	if got := reportMetadata(runs); len(got) != 5 {
		t.Fatalf("metadata rows: %d", len(got))
	}
	if got := reportMetadata(runs[:4]); len(got) != 0 {
		t.Fatal("incomplete metadata accepted")
	}
	if !probeControlPage([]Run{{Text: "Vegetation Management Plan"}, {Text: "Example Site"}, {Text: "January 2026"}}) {
		t.Fatal("management plan cover omitted")
	}
}

func TestReportMetadataRequiresAlignedValues(t *testing.T) {
	var runs []Run
	for i, label := range []string{"Report:", "Prepared for:", "Prepared by:", "Version:"} {
		y := float64(500 - i*25)
		runs = append(runs,
			Run{Text: label, Source: Source{Page: 2, X: 78, Y: y, Width: 55, Height: 10}},
			Run{Text: "literal value", Source: Source{Page: 2, X: 172, Y: y, Width: 60, Height: 10}})
	}
	if got := reportMetadata(runs); len(got) != 8 {
		t.Fatalf("aligned metadata omitted: %d", len(got))
	}
	runs[7].Source.Y -= 20
	if got := reportMetadata(runs); len(got) != 0 {
		t.Fatal("unrelated text used as version value")
	}
}

func TestContentsControlSectionExcludesCitedReports(t *testing.T) {
	for _, entry := range []struct {
		value string
		want  bool
	}{{"Document Control ........ 3", true}, {"Document Control ........ 4", false}, {"Document Control 3", false}} {
		if nextPageControlReference([]Run{{Text: "Table of Contents"}, {Text: entry.value}}) != entry.want {
			t.Fatalf("contents reference: %s", entry.value)
		}
	}
	if nextPageControlReference([]Run{{Text: "Document Control ........ 3"}}) {
		t.Fatal("body reference accepted without contents heading")
	}
	runs := []Run{
		{Text: "footer reference", Source: Source{Page: 3, Y: 20}},
		{Text: "1", Source: Source{Page: 3, X: 50, Y: 700, Height: 20}},
		{Text: "Document Control", Source: Source{Page: 3, X: 100, Y: 700, Height: 20}},
		{Text: "Document Name | EX-123-REP", Source: Source{Page: 3, X: 50, Y: 650, Height: 10}},
		{Text: "R1 23.02.2026", Source: Source{Page: 3, X: 50, Y: 600, Height: 10}},
		{Text: "2", Source: Source{Page: 3, X: 50, Y: 500, Height: 20}},
		{Text: "Executive Summary", Source: Source{Page: 3, X: 100, Y: 500, Height: 20}},
		{Text: "Product report ABC999", Source: Source{Page: 3, X: 50, Y: 400, Height: 10}},
	}
	if got := boundedControlSection(runs); len(got) != 4 || !controlPage(got) {
		t.Fatalf("control region: %+v", got)
	}
	runs[6].Source.Height = 10
	if len(boundedControlSection(runs)) != 0 {
		t.Fatal("body-sized text accepted as sibling heading")
	}
}
