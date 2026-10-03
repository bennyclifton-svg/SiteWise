package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestCurrentRevisionCellIsNotIssueHistory(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "01", Source: identity.Source{Page: 1, X: 2332, Y: 36, Width: 13, Height: 16}},
		{Text: "P1 ISSUED FOR COORDINATION 14.04.23"},
		{Text: "Revision:", Source: identity.Source{Page: 1, X: 2249, Y: 36, Width: 49, Height: 16}},
	}}
	for _, r := range intake.Decide(intake.Harvest("H401.pdf", text)) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "01") {
			t.Fatalf("issue history overrode current revision cell: %+v", r)
		}
	}
	for _, r := range intake.Decide(intake.Harvest("H401-02.pdf", text)) {
		if r.Field == intake.FieldRevision && r.Settled {
			t.Fatalf("conflicting filename and current revision must stay open: %+v", r)
		}
	}
}

func TestInlineSheetTitleIncludesBothAlignedLines(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Title:", Source: identity.Source{Page: 1, X: 53, Y: 1549, Width: 60, Height: 13.5}},
		{Text: "STANDARD SLAB ON GROUND", Source: identity.Source{Page: 1, X: 115.5, Y: 1555.5, Width: 67, Height: 11}},
		{Text: "DETAILS SHEET 1", Source: identity.Source{Page: 1, X: 115.3, Y: 1546, Width: 39, Height: 11}},
		{Text: "Rev Date", Source: identity.Source{Page: 1, X: 53, Y: 1530, Width: 40, Height: 11}},
	}}
	for _, r := range intake.Decide(intake.Harvest("S113-01.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "STANDARD SLAB ON GROUND DETAILS SHEET 1") {
			t.Fatalf("partial multiline title: %+v", r)
		}
	}
}

func TestSpatialRevisionDoesNotPickFirstHistoryRow(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "REV B 06.11.2023"},
		{Text: "REV", Source: identity.Source{Page: 1, X: 100, Y: 40, Width: 20, Height: 10}},
		{Text: "A", Source: identity.Source{Page: 1, X: 100, Y: 20, Width: 10, Height: 10}},
	}}
	for _, r := range intake.Decide(intake.Harvest("A128.pdf", text)) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "B") {
			t.Fatalf("history table header introduced a current revision: %+v", r)
		}
	}
}

func TestDrawingNumberCaptionUsesGeometryNotContentOrder(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "A-101", Source: identity.Source{Page: 1, X: 100, Y: 10, Width: 30, Height: 10}},
		{Text: "Drawing No", Source: identity.Source{Page: 1, X: 100, Y: 30, Width: 40, Height: 6}},
		{Text: "S-201", Source: identity.Source{Page: 1, X: 300, Y: 10, Width: 30, Height: 10}},
	}}
	got := intake.Harvest("upload.pdf", text)
	c, ok := findCandidate(got, intake.FieldNumber, "A-101")
	if !ok || !c.Provenance.Labeled {
		t.Fatalf("own number not bound to caption: %+v", c)
	}
	c, _ = findCandidate(got, intake.FieldNumber, "S-201")
	if c.Provenance.Labeled {
		t.Fatal("distant reference bound to caption")
	}
}

func TestTitleNeedsFilenameAndSpatialCaptionAgreement(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "GROUND FLOOR PLAN", Source: identity.Source{Page: 1, X: 100, Y: 10, Width: 100, Height: 10}},
		{Text: "Drawing Title", Source: identity.Source{Page: 1, X: 100, Y: 30, Width: 40, Height: 6}},
		{Text: "PROJECT NAME", Source: identity.Source{Page: 1, X: 300, Y: 10, Width: 70, Height: 10}},
	}}
	for _, result := range intake.Decide(intake.Harvest("A101 GROUND FLOOR PLAN.pdf", text)) {
		if result.Field == intake.FieldTitle && (!result.Settled || result.Display != "GROUND FLOOR PLAN") {
			t.Fatalf("filename and spatial title should agree: %+v", result)
		}
	}
	for _, result := range intake.Decide(intake.Harvest("A101 ROOF PLAN.pdf", text)) {
		if result.Field == intake.FieldTitle && result.Settled {
			t.Fatal("conflicting file and sheet titles must stay open")
		}
	}
}

func TestInlineDrawingTitleBeatsRevisionTableHeader(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Title:", Source: identity.Source{Page: 1, X: 50, Y: 150, Width: 60, Height: 13}},
		{Text: "Rev Date", Source: identity.Source{Page: 1, X: 50, Y: 133, Width: 50, Height: 15}},
		{Text: "GROUND FLOOR SECTIONS AND DETAILS", Source: identity.Source{Page: 1, X: 114, Y: 151, Width: 100, Height: 11}},
		{Text: "Project Title: OTHER PROJECT"},
	}}
	for _, r := range intake.Decide(intake.Harvest("S502-03.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "GROUND FLOOR SECTIONS AND DETAILS") {
			t.Fatalf("wrong title: %+v", r)
		}
	}
}

func TestExplicitTitleCanBeSingleWordNotes(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Title:", Source: identity.Source{Page: 1, X: 50, Y: 150, Width: 60, Height: 13}},
		{Text: "NOTES", Source: identity.Source{Page: 1, X: 114, Y: 151, Width: 40, Height: 11}},
		{Text: "DA12345, MOD/2023/0177", Source: identity.Source{Page: 1, X: 50, Y: 130, Width: 100, Height: 13}},
	}}
	for _, r := range intake.Decide(intake.Harvest("A001 NOTES.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "NOTES") {
			t.Fatalf("wrong title: %+v", r)
		}
	}
}

func TestExplicitTitleOverridesCADLayoutExportName(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Drawing Title: LEVEL 3 - PRESSURE"}}}
	for _, r := range intake.Decide(intake.Harvest("H-205-L3-Layout1.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "LEVEL 3 - PRESSURE") {
			t.Fatalf("CAD layout name overrode title: %+v", r)
		}
	}
	for _, r := range intake.Decide(intake.Harvest("H-205 ROOF DRAINAGE.pdf", text)) {
		if r.Field == intake.FieldTitle && r.Settled {
			t.Fatal("conflicting descriptive filename must stay open")
		}
	}
}

func TestCombinedProjectDrawingNumberAndRevisionCell(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Project Number/Drawing Number", Source: identity.Source{Page: 1, X: 100, Y: 100, Width: 120, Height: 14}},
		{Text: "Revision", Source: identity.Source{Page: 1, X: 368, Y: 100, Width: 32, Height: 14}},
		{Text: "123456-00-CC-C05.08 2", Source: identity.Source{Page: 1, X: 116, Y: 61, Width: 278, Height: 39}},
		{Text: "123456-00-CC-C05.09", Source: identity.Source{Page: 1, X: 100, Y: 250, Width: 150, Height: 14}},
	}}
	got := intake.Decide(intake.Harvest("123456-00-CC-C05.01-C05.12 PLANS-C05.08.pdf", text))
	for _, r := range got {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "123456-00-CC-C05.08") {
			t.Fatalf("full own number missing: %+v", r)
		}
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "2") {
			t.Fatalf("own revision missing: %+v", r)
		}
	}
	// Without the Revision column this could be a reference plus a sheet count.
	text.Runs = append(text.Runs[:1], text.Runs[2:]...)
	for _, c := range intake.Harvest("upload.pdf", text) {
		if c.Provenance.OwnNumber || c.Provenance.OwnRevision {
			t.Fatalf("unbound merged cell treated as authoritative: %+v", c)
		}
	}
}

func TestBottomMergedRevisionDateTableUsesCurrentRow(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Revision: 2"},
		{Text: "First Issue Date: 13/10/2025"},
		{Text: "REV. DATE", Source: identity.Source{Page: 1, X: 58, Y: 56, Width: 53, Height: 14}},
		{Text: "1 13/10/2025 FIRST ISSUE", Source: identity.Source{Page: 1, X: 65, Y: 70, Width: 138, Height: 14}},
		{Text: "2 23/12/2025 TENDER ISSUE", Source: identity.Source{Page: 1, X: 65, Y: 84, Width: 176, Height: 14}},
		{Text: "2 01/01/2026 unrelated note", Source: identity.Source{Page: 1, X: 650, Y: 84, Width: 176, Height: 14}},
	}}
	for _, r := range intake.Decide(intake.Harvest("upload.pdf", text)) {
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "23/12/2025") {
			t.Fatalf("current revision row not paired: %+v", r)
		}
	}
}

func TestGenericTitleInCompleteDrawingBlockIncludesSheetLine(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Title", Source: identity.Source{Page: 1, X: 100, Y: 107, Width: 15, Height: 14}},
		{Text: "TURNING PATH PLAN", Source: identity.Source{Page: 1, X: 100, Y: 90, Width: 200, Height: 19}},
		{Text: "SHEET 04", Source: identity.Source{Page: 1, X: 100, Y: 57, Width: 54, Height: 19}},
		{Text: "CIVIL ENGINEERING WORKS", Source: identity.Source{Page: 1, X: 100, Y: 120, Width: 160, Height: 19}},
		{Text: "Project Number/Drawing Number", Source: identity.Source{Page: 1, X: 480, Y: 107, Width: 120, Height: 14}},
		{Text: "Revision", Source: identity.Source{Page: 1, X: 748, Y: 107, Width: 32, Height: 14}},
		{Text: "123456-00-CC-C22.04 2", Source: identity.Source{Page: 1, X: 496, Y: 68, Width: 278, Height: 39}},
	}}
	for _, r := range intake.Decide(intake.Harvest("123456-00-CC-C22.01 TURNING PATH PLANS.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "TURNING PATH PLAN SHEET 04") {
			t.Fatalf("own title incomplete: %+v", r)
		}
	}
	text.Runs[0].Text = "Project Title"
	for _, c := range intake.Harvest("upload.pdf", text) {
		if c.Provenance.OwnTitle {
			t.Fatalf("project title is not drawing title: %+v", c)
		}
	}
}

func TestDrawingScheduleTitleMatchesOwnNumberOnly(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Project Number/Drawing Number", Source: identity.Source{Page: 1, X: 100, Y: 100, Width: 120, Height: 14}},
		{Text: "Revision", Source: identity.Source{Page: 1, X: 368, Y: 100, Width: 32, Height: 14}},
		{Text: "123456-00-CC-C01.01 2", Source: identity.Source{Page: 1, X: 116, Y: 61, Width: 278, Height: 39}},
		{Text: "DRAWING NUMBER", Source: identity.Source{Page: 1, X: 100, Y: 500, Width: 105, Height: 19}},
		{Text: "DESCRIPTION", Source: identity.Source{Page: 1, X: 245, Y: 500, Width: 78, Height: 19}},
		{Text: "123456-00-CC-C01.01", Source: identity.Source{Page: 1, X: 100, Y: 480, Width: 82, Height: 14}},
		{Text: "COVER SHEET AND DRAWING SCHEDULE", Source: identity.Source{Page: 1, X: 245, Y: 480, Width: 165, Height: 14}},
		{Text: "123456-00-CC-C01.02", Source: identity.Source{Page: 1, X: 100, Y: 465, Width: 82, Height: 14}},
		{Text: "GENERAL NOTES", Source: identity.Source{Page: 1, X: 245, Y: 465, Width: 100, Height: 14}},
	}}
	for _, r := range intake.Decide(intake.Harvest("123456-00-CC-C01.01 COVER-C01.01.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "COVER SHEET AND DRAWING SCHEDULE") {
			t.Fatalf("own schedule row not selected: %+v", r)
		}
	}
	// The cover exports number and revision as separate runs.
	text.Runs[2].Text = "123456-00-CC-C01.01"
	text.Runs[2].Source.Width = 230
	text.Runs = append(text.Runs, identity.Run{Text: "2", Source: identity.Source{Page: 1, X: 382, Y: 61, Width: 12, Height: 39}})
	for _, r := range intake.Decide(intake.Harvest("cover.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "COVER SHEET AND DRAWING SCHEDULE") {
			t.Fatalf("separate own number cell not bound: %+v", r)
		}
	}
	text.Runs[5].Text = "123456-00-CC-C99.99"
	for _, c := range intake.Harvest("upload.pdf", text) {
		if c.Provenance.OwnTitle {
			t.Fatalf("other sheet title selected: %+v", c)
		}
	}
}

func TestDrawingNameMergedIdentityAndBottomIssueColumns(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Page: 1, X: 100, Y: 262, Width: 43, Height: 9}},
		{Text: "Drawing Name", Source: identity.Source{Page: 1, X: 168, Y: 262, Width: 52, Height: 9}},
		{Text: "L-000 Cover Sheet", Source: identity.Source{Page: 1, X: 100, Y: 232, Width: 127, Height: 19}},
		{Text: "L001", Source: identity.Source{Page: 1, X: 20, Y: 600, Width: 30, Height: 9}},
		{Text: "Issue", Source: identity.Source{Page: 1, X: 103, Y: 493, Width: 19, Height: 9}},
		{Text: "Revision Description", Source: identity.Source{Page: 1, X: 129, Y: 493, Width: 72, Height: 9}},
		{Text: "Drawn Check Date", Source: identity.Source{Page: 1, X: 217, Y: 493, Width: 74, Height: 9}},
		{Text: "T3", Source: identity.Source{Page: 1, X: 103, Y: 549, Width: 9, Height: 9}},
		{Text: "SS/YJ/JS NW 17.11.25", Source: identity.Source{Page: 1, X: 215, Y: 549, Width: 87, Height: 9}},
		{Text: "T4", Source: identity.Source{Page: 1, X: 103, Y: 560, Width: 9, Height: 9}},
		{Text: "SS/YJ/JS NW 27.11.25", Source: identity.Source{Page: 1, X: 215, Y: 560, Width: 88, Height: 9}},
	}}
	want := map[string]string{intake.FieldNumber: "L-000", intake.FieldTitle: "Cover Sheet", intake.FieldDate: "27.11.25"}
	for _, r := range intake.Decide(intake.Harvest("L-000[T4].pdf", text)) {
		if v, ok := want[r.Field]; ok && (!r.Settled || r.Display != v) {
			t.Errorf("%s: %+v", r.Field, r)
		}
	}
	// A row on another page cannot supply the current issue's date.
	text.Runs[10].Source.Page = 2
	for _, c := range intake.Harvest("L-000[T4].pdf", text) {
		if c.Field == intake.FieldDate && c.Display == "27.11.25" && c.Provenance.IssueTable {
			t.Fatal("cross-page table pairing")
		}
	}
}

func TestDrawingNameCaptionJoinsZoneSubtitle(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing Name", Source: identity.Source{Page: 1, X: 168, Y: 262, Width: 52, Height: 9}},
		{Text: "Hardworks and Grading Plan", Source: identity.Source{Page: 1, X: 168, Y: 238, Width: 132, Height: 12}},
		{Text: "Zone 14", Source: identity.Source{Page: 1, X: 168, Y: 226, Width: 39, Height: 12}},
	}}
	for _, r := range intake.Decide(intake.Harvest("L-314[T4] Hardworks and Grading Plan Zone 14.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "Hardworks and Grading Plan Zone 14") {
			t.Fatalf("title: %+v", r)
		}
	}
}

func TestSpecificationReferenceRetainsCompleteCompositeNumber(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "24-1068-L-SP001"}, {Text: "Project number: 24-1068"}}}
	for _, r := range intake.Decide(intake.Harvest("Landscape Specification [02].pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "24-1068-L-SP001") {
			t.Fatalf("number: %+v", r)
		}
	}
}

func TestDrawingScheduleTitleDoesNotBecomeOwnTitle(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "DRAWING SCHEDULE", Source: identity.Source{Page: 1, X: 20, Y: 700, Width: 160, Height: 18}},
		{Text: "DRAWING TITLE", Source: identity.Source{Page: 1, X: 80, Y: 660, Width: 70, Height: 10}},
		{Text: "Cover Sheet", Source: identity.Source{Page: 1, X: 80, Y: 640, Width: 60, Height: 10}},
		{Text: "Master Legend", Source: identity.Source{Page: 1, X: 80, Y: 628, Width: 70, Height: 10}},
	}}
	for _, c := range intake.Harvest("L-000[T4] Cover Sheet.pdf", text) {
		if c.Provenance.OwnTitle {
			t.Fatalf("register treated as own title: %+v", c)
		}
	}
}

func TestReportFooterNumberDoesNotBorrowRemoteHeaderTitle(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Example project", Source: identity.Source{Page: 1, X: 56, Y: 798, Width: 150, Height: 10}},
		{Text: "Example Landscape Pty Ltd", Source: identity.Source{Page: 1, X: 402, Y: 798, Width: 136, Height: 10}},
		{Text: "24-1068-L-SP001", Source: identity.Source{Page: 1, X: 67, Y: 48, Width: 60, Height: 10}},
		{Text: "Date 27-Nov-25", Source: identity.Source{Page: 1, X: 239, Y: 48, Width: 58, Height: 10}},
		{Text: "Revision 2", Source: identity.Source{Page: 1, X: 492, Y: 48, Width: 35, Height: 10}},
		{Text: "Reference:", Source: identity.Source{Page: 1, X: 56, Y: 130, Width: 47, Height: 13}},
		{Text: "24-1068-L-SP001", Source: identity.Source{Page: 1, X: 142, Y: 130, Width: 75, Height: 13}},
		{Text: "Issue:", Source: identity.Source{Page: 1, X: 56, Y: 105, Width: 26, Height: 13}},
		{Text: "2 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“ 70% Tender", Source: identity.Source{Page: 1, X: 142, Y: 105, Width: 65, Height: 13}},
	}}
	for _, c := range intake.Harvest("upload.pdf", text) {
		if c.Field == intake.FieldTitle && c.Provenance.Origin == intake.OriginText {
			t.Fatalf("footer adjacency invented title: %+v", c)
		}
	}
}

func TestStackedJobAndIssueCellConflictsWithFilename(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Page: 1, X: 100, Y: 262, Width: 43, Height: 9}},
		{Text: "Drawing Name", Source: identity.Source{Page: 1, X: 168, Y: 262, Width: 52, Height: 9}},
		{Text: "L-000 Cover Sheet", Source: identity.Source{Page: 1, X: 100, Y: 232, Width: 127, Height: 19}},
		{Text: "Job No.", Source: identity.Source{Page: 1, X: 100, Y: 199, Width: 27, Height: 9}},
		{Text: "Issue", Source: identity.Source{Page: 1, X: 100, Y: 185, Width: 19, Height: 9}},
		{Text: "24-1068", Source: identity.Source{Page: 1, X: 218, Y: 199, Width: 31, Height: 9}},
		{Text: "B", Source: identity.Source{Page: 1, X: 243, Y: 185, Width: 5, Height: 9}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Landscape Plans DA [C].pdf", text)) {
		if r.Field == intake.FieldRevision && r.Settled {
			t.Fatalf("conflict confirmed: %+v", r)
		}
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "Cover Sheet") {
			t.Fatalf("complete identity lost title: %+v", r)
		}
	}
	for _, r := range intake.Decide(intake.Harvest("Landscape Plans DA [B].pdf", text)) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "B") {
			t.Fatalf("agreed issue missing: %+v", r)
		}
	}
	text.Runs[3].Text = "Revision Description"
	for _, c := range intake.Harvest("upload.pdf", text) {
		if c.Provenance.OwnRevision {
			t.Fatalf("history table became own issue: %+v", c)
		}
	}
}

func TestSparseApplicationCoverOffersFullPrintedTitle(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Example Redevelopment", Source: identity.Source{Page: 1, X: 198, Y: 251, Width: 457, Height: 37}},
		{Text: "Development Application", Source: identity.Source{Page: 1, X: 198, Y: 222, Width: 312, Height: 36}},
		{Text: "Prepared for Example Client", Source: identity.Source{Page: 1, X: 198, Y: 164, Width: 309, Height: 16}},
		{Text: "March 2025", Source: identity.Source{Page: 1, X: 198, Y: 151, Width: 63, Height: 16}},
	}}
	found := false
	for _, c := range intake.Harvest("Landscape Report DA [F].pdf", text) {
		if c.Field == intake.FieldTitle && c.Display == "Example Redevelopment Development Application" && c.Provenance.Heading {
			found = true
		}
	}
	if !found {
		t.Fatal("printed cover heading missing")
	}
	for _, r := range intake.Decide(intake.Harvest("Landscape Report DA [F].pdf", text)) {
		if r.Field == intake.FieldTitle && r.Settled {
			t.Fatalf("filename contradiction confirmed: %+v", r)
		}
	}
}

func TestCenteredAssessmentHeadingRetainsAllThreeLines(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "ARBORICULTURAL ASSESSMENT", Source: identity.Source{Page: 1, X: 99, Y: 640, Width: 397, Height: 22.5}},
		{Text: "AND CONSTRUCTION IMPACT", Source: identity.Source{Page: 1, X: 119, Y: 608, Width: 357, Height: 22.5}},
		{Text: "ASSESSMENT", Source: identity.Source{Page: 1, X: 215, Y: 577, Width: 165, Height: 22.5}},
		{Text: "Example Client", Source: identity.Source{Page: 1, X: 240, Y: 510, Width: 115, Height: 18}},
		{Text: "Report Reference: ABC - DEF 08/14", Source: identity.Source{Page: 1, X: 214, Y: 391, Width: 201, Height: 14}},
		{Text: "15th August, 2014", Source: identity.Source{Page: 1, X: 248, Y: 362, Width: 100, Height: 14}},
		{Text: "Prepared by:", Source: identity.Source{Page: 1, X: 266, Y: 305, Width: 63, Height: 12}},
		{Text: "Consultant", Source: identity.Source{Page: 1, X: 259, Y: 276, Width: 77, Height: 14}},
	}}
	found := false
	for _, c := range intake.Harvest("Arborist Report.pdf", text) {
		if c.Field == intake.FieldTitle && c.Display == "ARBORICULTURAL ASSESSMENT AND CONSTRUCTION IMPACT ASSESSMENT" && c.Provenance.Heading {
			found = true
		}
	}
	if !found {
		t.Fatal("centred heading lost a line")
	}
}

func TestReportCoverSingleWordLinesAndReferenceCells(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Arboricultural", Source: identity.Source{Page: 1, X: 280, Y: 650, Width: 238, Height: 52}},
		{Text: "Impact", Source: identity.Source{Page: 1, X: 280, Y: 609, Width: 116, Height: 52}},
		{Text: "Assessment", Source: identity.Source{Page: 1, X: 280, Y: 567, Width: 212, Height: 52}},
		{Text: "Report", Source: identity.Source{Page: 1, X: 280, Y: 526, Width: 116, Height: 52}},
		{Text: "Date Prepared: 26 November 2025", Source: identity.Source{Page: 1, X: 280, Y: 233, Width: 222, Height: 20}},
		{Text: "Ref: 251126_Camp Y_AIA_R2", Source: identity.Source{Page: 1, X: 280, Y: 217, Width: 192, Height: 20}},
		{Text: "Rev: 2", Source: identity.Source{Page: 1, X: 280, Y: 201, Width: 42, Height: 20}},
		{Text: "Urban Arbor Pty Ltd", Source: identity.Source{Page: 1, X: 280, Y: 249, Width: 123, Height: 19}},
	}}
	found := false
	for _, c := range intake.Harvest("Assessment[02].pdf", text) {
		if c.Field == intake.FieldTitle && c.Display == "Arboricultural Impact Assessment Report" {
			found = true
		}
	}
	if !found {
		t.Fatal("single-word heading lines missing")
	}
	for _, r := range intake.Decide(intake.Harvest("Assessment[02].pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "251126_Camp Y_AIA_R2") {
			t.Fatalf("reference: %+v", r)
		}
	}
	separate := identity.Text{Runs: []identity.Run{
		{Text: "Our Reference:", Source: identity.Source{Page: 1, X: 78, Y: 615, Width: 76, Height: 15}},
		{Text: "251217_Camp Y_AIA_Add", Source: identity.Source{Page: 1, X: 226, Y: 616, Width: 126, Height: 14}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Addendum.pdf", separate)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "251217_Camp Y_AIA_Add") {
			t.Fatalf("separate reference: %+v", r)
		}
	}
}

func TestReportTypeCellSeparatesPrintedRevision(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Report Type:", Source: identity.Source{Page: 1, X: 78, Y: 593, Width: 65, Height: 15}},
		{Text: "Addendum to Arboricultural Impact Assessment Report Rev2", Source: identity.Source{Page: 1, X: 226, Y: 593, Width: 287, Height: 14}},
	}}
	found := false
	for _, c := range intake.Harvest("Addendum [02].pdf", text) {
		if c.Field == intake.FieldTitle && c.Display == "Addendum to Arboricultural Impact Assessment Report" && c.Provenance.Labeled {
			found = true
		}
	}
	if !found {
		t.Fatal("report type title was not offered without its revision suffix")
	}
}

func TestReportCoverSeparateProjectDateRevCells(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "ABC3293 Project name", Source: identity.Source{Page: 1, X: 40, Y: 30, Width: 180, Height: 9}},
		{Text: "Project No.", Source: identity.Source{Page: 1, X: 856, Y: 152, Width: 53, Height: 9}},
		{Text: "Date", Source: identity.Source{Page: 1, X: 856, Y: 127, Width: 22, Height: 9}},
		{Text: "Rev", Source: identity.Source{Page: 1, X: 856, Y: 102, Width: 18, Height: 9}},
		{Text: "ABC3293", Source: identity.Source{Page: 1, X: 935, Y: 153, Width: 43, Height: 9}},
		{Text: "16 December 2025", Source: identity.Source{Page: 1, X: 935, Y: 127, Width: 85, Height: 9}},
		{Text: "02", Source: identity.Source{Page: 1, X: 935, Y: 102, Width: 11, Height: 9}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Section J Report R02.pdf", text)) {
		if r.Field == intake.FieldNumber && r.Display != "" {
			t.Fatalf("project reference became document number: %+v", r)
		}
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "02") {
			t.Fatalf("cover revision missing: %+v", r)
		}
	}
	text.Runs[1].Text = "Description"
	for _, r := range intake.Decide(intake.Harvest("Report.pdf", text)) {
		if r.Field == intake.FieldRevision && r.Settled {
			t.Fatalf("ordinary table Rev treated as cover control: %+v", r)
		}
	}
}

func TestReportHeadingAmongLargeCoverControlText(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Proposed Residential", Source: identity.Source{Page: 1, X: 220, Y: 650, Width: 210, Height: 20.24}},
		{Text: "Development", Source: identity.Source{Page: 1, X: 250, Y: 630, Width: 150, Height: 20.24}},
		{Text: "Example Project Address", Source: identity.Source{Page: 1, X: 190, Y: 600, Width: 270, Height: 19.49}},
		{Text: "Assessment of Traffic and", Source: identity.Source{Page: 1, X: 220, Y: 560, Width: 210, Height: 20.24}},
		{Text: "Parking Implications", Source: identity.Source{Page: 1, X: 240, Y: 540, Width: 170, Height: 20.24}},
		{Text: "August 2014", Source: identity.Source{Page: 1, X: 280, Y: 400, Width: 90, Height: 15.97}},
		{Text: "Rev A", Source: identity.Source{Page: 1, X: 300, Y: 380, Width: 50, Height: 15.97}},
		{Text: "Reference 14169", Source: identity.Source{Page: 1, X: 260, Y: 300, Width: 130, Height: 15.97}},
	}}
	for i := 0; i < 5; i++ {
		text.Runs = append(text.Runs, identity.Run{Text: "Contact details", Source: identity.Source{Page: 1, X: 200, Y: float64(150 - i*15), Width: 250, Height: 13.97}})
	}
	found := false
	for _, c := range intake.Harvest("Traffic Report.pdf", text) {
		found = found || (c.Field == intake.FieldTitle && c.Display == "Assessment of Traffic and Parking Implications" && c.Provenance.Heading)
	}
	if !found {
		t.Fatal("cover control font sizes hid the report heading")
	}
}

func TestDottedReportReferenceExcludesEngineerRegistration(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Division of Example ABN: 45067491678 RPEQ: 19457"},
		{Text: "Reference: 240648.02FA"},
		{Text: "The earlier report (240648.01FA), dated 5 September 2025"},
	}}
	for _, r := range intake.Decide(intake.Harvest("Response.pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "240648.02FA") {
			t.Fatalf("own reference not preserved: %+v", r)
		}
	}
	for _, c := range intake.Harvest("Response.pdf", text) {
		if c.Field == intake.FieldNumber && c.Display == "19457" {
			t.Fatal("RPEQ registration became document number")
		}
	}
}

func TestCompactCenteredReportHeading(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "TRAFFIC AND PARKING IMPACT ASSESSMENT OF", Source: identity.Source{Page: 1, X: 150, Y: 491, Width: 295, Height: 11.25}},
		{Text: "THE PROPOSED EXAMPLE REDEVELOPMENT", Source: identity.Source{Page: 1, X: 135, Y: 470, Width: 325, Height: 11.25}},
		{Text: "AT EXAMPLE ROAD", Source: identity.Source{Page: 1, X: 197, Y: 449, Width: 201, Height: 11.25}},
		{Text: "Company address", Source: identity.Source{Page: 1, X: 140, Y: 200, Width: 313, Height: 9.34}},
		{Text: "Company contact", Source: identity.Source{Page: 1, X: 140, Y: 180, Width: 313, Height: 9.34}},
	}}
	want := "TRAFFIC AND PARKING IMPACT ASSESSMENT OF THE PROPOSED EXAMPLE REDEVELOPMENT AT EXAMPLE ROAD"
	found := false
	for _, c := range intake.Harvest("Traffic.pdf", text) {
		found = found || (c.Field == intake.FieldTitle && c.Display == want)
	}
	if !found {
		t.Fatal("compact centered heading missing")
	}
	text.Runs[1].Source.X += 100
	for _, c := range intake.Harvest("Traffic.pdf", text) {
		if c.Field == intake.FieldTitle && c.Display == want {
			t.Fatal("unrelated off-center line joined")
		}
	}
}

func TestApprovalTableKeepsIssueColumnSeparateFromAuthorInitials(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "5 September 2025", Source: identity.Source{Page: 1}},
		{Text: "Status", Source: identity.Source{Page: 2, X: 74, Y: 520, Width: 34, Height: 10}},
		{Text: "Issue", Source: identity.Source{Page: 2, X: 133, Y: 520, Width: 28, Height: 10}},
		{Text: "Prepared By", Source: identity.Source{Page: 2, X: 178, Y: 520, Width: 65, Height: 10}},
		{Text: "Date", Source: identity.Source{Page: 2, X: 462, Y: 520, Width: 24, Height: 10}},
		{Text: "B", Source: identity.Source{Page: 2, X: 143, Y: 497, Width: 8, Height: 10}},
		{Text: "AT", Source: identity.Source{Page: 2, X: 203, Y: 497, Width: 15, Height: 10}},
		{Text: "4 March 2025", Source: identity.Source{Page: 2, X: 436, Y: 497, Width: 75, Height: 10}},
		{Text: "B", Source: identity.Source{Page: 2, X: 143, Y: 404, Width: 8, Height: 10}},
		{Text: "AT", Source: identity.Source{Page: 2, X: 203, Y: 404, Width: 15, Height: 10}},
		{Text: "5 September 2025", Source: identity.Source{Page: 2, X: 427, Y: 404, Width: 94, Height: 10}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Assessment.pdf", text)) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "B") {
			t.Fatalf("cover date did not identify own issue: %+v", r)
		}
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "5 September 2025") {
			t.Fatalf("cover date lost: %+v", r)
		}
	}
}

func TestCodeInExplicitProjectNameIsNotDocumentNumber(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Project Name", Source: identity.Source{Page: 1, X: 106, Y: 1486, Width: 59, Height: 9.3}},
		{Text: "SYD3293 - Example Road", Source: identity.Source{Page: 1, X: 181, Y: 1486, Width: 220, Height: 9.3}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Safety Report.pdf", text)) {
		if r.Field == intake.FieldNumber && r.Display != "" {
			t.Fatalf("project-name code became document number: %+v", r)
		}
	}
	text.Runs = append(text.Runs, identity.Run{Text: "Document Number: SYD3293"})
	for _, r := range intake.Decide(intake.Harvest("Safety Report.pdf", text)) {
		if r.Field == intake.FieldNumber && (!r.Settled || r.Display != "SYD3293") {
			t.Fatalf("explicit document number incorrectly suppressed: %+v", r)
		}
	}
}
