package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestBracketedMonthYearIsNotDocumentNumber(t *testing.T) {
	for _, tag := range []string{"OCT25", "JAN2026", "dec24"} {
		got := intake.Harvest("BCA CONSULTANT ADVICE ["+tag+"].pdf", identity.Text{})
		for _, c := range got {
			if c.Field != intake.FieldTitle || c.Display != "BCA CONSULTANT ADVICE" {
				t.Errorf("%s: unexpected candidate %+v", tag, c)
			}
		}
		if len(got) != 1 {
			t.Errorf("%s: candidates = %d", tag, len(got))
		}
	}
}

func TestEquivalentReportDatesAgree(t *testing.T) {
	for _, date := range []string{"12/12/2025", "12.12.2025", "2025-12-12"} {
		got := intake.Decide(intake.Harvest("Compliance 12 DEC 2025.pdf", identity.Text{Runs: []identity.Run{{Text: "Date " + date}}}))
		if r := fieldResult(t, got, intake.FieldDate); !r.Settled {
			t.Errorf("equivalent %s left ambiguous: %+v", date, r)
		}
	}
	got := intake.Decide(intake.Harvest("Compliance 12 DEC 2025.pdf", identity.Text{Runs: []identity.Run{{Text: "Date 11/12/2025"}}}))
	if r := fieldResult(t, got, intake.FieldDate); r.Settled {
		t.Fatalf("conflicting dates settled: %+v", r)
	}
}

func TestProminentReportHeadingChallengesFilename(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "BCA CONSULTANT ADVICE", Source: identity.Source{Page: 1, X: 70, Y: 705, Width: 215, Height: 15}},
		{Text: "PRELIMINARIES", Source: identity.Source{Page: 1, X: 70, Y: 647, Width: 78, Height: 9}},
		{Text: "Property Details", Source: identity.Source{Page: 1, X: 76, Y: 598, Width: 69, Height: 8}},
		{Text: "Description of Scope", Source: identity.Source{Page: 1, X: 76, Y: 574, Width: 90, Height: 8}},
		{Text: "Client", Source: identity.Source{Page: 1, X: 76, Y: 550, Width: 24, Height: 8}},
	}}
	got := intake.Harvest("BCA COMPLIANCE STATEMENT.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "BCA CONSULTANT ADVICE"); !ok {
		t.Fatal("missing literal report heading")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatalf("contradictory filename must not be uniquely correct: %+v", r)
	}
	text.Runs[0].Source.Height = 8
	got = intake.Harvest("other.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "BCA CONSULTANT ADVICE"); ok {
		t.Fatal("ordinary body text became a heading")
	}
}

func TestReportNumberAndProjectNumberRoles(t *testing.T) {
	got := intake.Harvest("5698-1.1R draft.pdf", identity.Text{Runs: []identity.Run{{Text: "REPORT NUMBER"}, {Text: "5698-1.1R"}}})
	if r := fieldResult(t, intake.Decide(got), intake.FieldNumber); !r.Settled || r.Display != "5698-1.1R" {
		t.Fatalf("report number: %+v", r)
	}
	for _, label := range []string{"Project No:", "Project Number", "Job No:", "Project Ref."} {
		got = intake.Harvest("Acoustic specification.pdf", identity.Text{Runs: []identity.Run{{Text: label + " SYD3293 Date 2 December 2025"}}})
		if _, ok := findCandidate(got, intake.FieldNumber, "SYD3293"); ok {
			t.Errorf("%s became document number", label)
		}
		if _, ok := findCandidate(got, intake.FieldDate, "2 December 2025"); !ok {
			t.Errorf("%s suppressed following date", label)
		}
	}
}

func TestReportRevisionLabelsAndTemplateNumbers(t *testing.T) {
	if _, comparable := intake.Compare("1.2", "1.10"); comparable {
		t.Fatal("unestablished report-version ordering must not drive supersession")
	}
	for _, rev := range []string{"1.2", "PBDB1.0", "PBDB 2.0"} {
		got := intake.Harvest("report.pdf", identity.Text{Runs: []identity.Run{{Text: "Revision: " + rev}}})
		if _, ok := findCandidate(got, intake.FieldRevision, rev); !ok {
			t.Errorf("missing literal revision %q", rev)
		}
	}
	got := intake.Harvest("215066 Adapt CC.pdf", identity.Text{Runs: []identity.Run{{Text: "Document Control Job No. 215066"}, {Text: "All Rights Reserved. T0123"}}})
	for _, c := range got {
		if c.Field == intake.FieldNumber {
			t.Errorf("non-document reference harvested: %+v", c)
		}
	}
}

func TestWrappedReviewHeadingIsOneLiteralCandidate(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "ACCESS DESIGN REVIEW - DEVELOPMENT", Source: identity.Source{Page: 1, X: 80, Y: 697, Width: 446, Height: 19}},
		{Text: "APPLICATION (DA)", Source: identity.Source{Page: 1, X: 80, Y: 673, Width: 188, Height: 19}},
		{Text: "EXAMPLE PROJECT", Source: identity.Source{Page: 1, X: 80, Y: 635, Width: 200, Height: 12}},
		{Text: "Client", Source: identity.Source{Page: 1, X: 80, Y: 600, Width: 40, Height: 9}},
		{Text: "Date", Source: identity.Source{Page: 1, X: 80, Y: 580, Width: 40, Height: 9}},
	}}
	got := intake.Harvest("Access review.pdf", text)
	c, ok := findCandidate(got, intake.FieldTitle, "ACCESS DESIGN REVIEW - DEVELOPMENT APPLICATION (DA)")
	if !ok || len(c.Provenance.JoinedRuns) != 2 {
		t.Fatalf("missing complete heading: %+v", got)
	}
}

func TestReportHeadingFontBoundaryToleratesPDFRounding(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Statement of Compliance", Source: identity.Source{Page: 1, X: 80, Y: 301, Height: 13.13198}},
		{Text: "Access For People With A", Source: identity.Source{Page: 1, X: 80, Y: 285, Height: 13.13198}},
		{Text: "Disability", Source: identity.Source{Page: 1, X: 80, Y: 269, Height: 13.13198}},
	}}
	for i := 0; i < 5; i++ {
		text.Runs = append(text.Runs, identity.Run{Text: "Body", Source: identity.Source{Page: 1, X: 80, Y: float64(i) * 15, Height: 9.380005}})
	}
	got := intake.Harvest("Adapt CC.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "Statement of Compliance Access For People With A Disability"); !ok {
		t.Fatal("rounded geometry lost the full cover heading")
	}
}

func TestVersionAndLetterSubjectAreCandidates(t *testing.T) {
	got := intake.Harvest("Hazard [0.2].pdf", identity.Text{Runs: []identity.Run{{Text: "Version V1.0"}}})
	if _, ok := findCandidate(got, intake.FieldRevision, "V1.0"); !ok {
		t.Fatal("printed version missing")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldRevision); r.Settled {
		t.Fatal("conflicting filename version settled")
	}
	got = intake.Harvest("Advice [SEP 2025].pdf", identity.Text{Runs: []identity.Run{{Text: "Re: Vegetation Management Advice"}}})
	if _, ok := findCandidate(got, intake.FieldTitle, "Vegetation Management Advice"); !ok {
		t.Fatal("letter subject missing")
	}
	got = intake.Harvest("Waterproofing Notes.pdf", identity.Text{Runs: []identity.Run{{Text: "subject to movement."}}})
	if _, ok := findCandidate(got, intake.FieldTitle, "to movement"); ok {
		t.Fatal("ordinary subject prose became a title")
	}
}

func TestFilenameRevisionCorroboratedByMergedIssueRow(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Revision Issue Date", Source: identity.Source{Page: 1, X: 73, Y: 183, Width: 120, Height: 14}},
		{Text: "P1", Source: identity.Source{Page: 1, X: 88, Y: 158, Width: 15, Height: 14}},
		{Text: "16/05/2015", Source: identity.Source{Page: 1, X: 128, Y: 158, Width: 70, Height: 13}},
		{Text: "A", Source: identity.Source{Page: 1, X: 90, Y: 123, Width: 9, Height: 14}},
		{Text: "27/07/2015", Source: identity.Source{Page: 1, X: 128, Y: 123, Width: 70, Height: 13}},
	}}
	got := intake.Decide(intake.Harvest("Fire Report [A].pdf", text))
	if r := fieldResult(t, got, intake.FieldRevision); !r.Settled || r.Display != "A" {
		t.Fatalf("revision: %+v", r)
	}
	if r := fieldResult(t, got, intake.FieldDate); !r.Settled || r.Display != "27/07/2015" {
		t.Fatalf("date: %+v", r)
	}
	text.Runs = append(text.Runs, identity.Run{Text: "Revision: P1"})
	got = intake.Decide(intake.Harvest("Fire Report [A].pdf", text))
	if r := fieldResult(t, got, intake.FieldRevision); r.Settled {
		t.Fatal("issue row overrode conflicting explicit revision")
	}
}

func TestDocumentControlTitleAgreesWithCoverHeading(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "FIRE ENGINEERING REPORT", Source: identity.Source{Page: 1, X: 135, Y: 641, Width: 340, Height: 28}},
		{Text: "Document Verification History", Source: identity.Source{Page: 1, X: 72, Y: 307, Width: 190, Height: 14}},
		{Text: "Project:", Source: identity.Source{Page: 1, X: 72, Y: 290, Width: 50, Height: 13}},
		{Text: "Example project", Source: identity.Source{Page: 1, X: 180, Y: 290, Width: 150, Height: 13}},
		{Text: "Title:", Source: identity.Source{Page: 1, X: 72, Y: 255, Width: 28, Height: 13}},
		{Text: "Fire Engineering Report", Source: identity.Source{Page: 1, X: 180, Y: 255, Width: 150, Height: 13}},
		{Text: "Revision Issue Date", Source: identity.Source{Page: 1, X: 73, Y: 183, Width: 120, Height: 14}},
	}}
	got := intake.Decide(intake.Harvest("Example project FER [A].pdf", text))
	if r := fieldResult(t, got, intake.FieldTitle); !r.Settled || r.Display != "Fire Engineering Report" {
		t.Fatalf("control title: %+v", r)
	}
	text.Runs[4].Text = "Project Title:"
	got = intake.Decide(intake.Harvest("Example project FER [A].pdf", text))
	if r := fieldResult(t, got, intake.FieldTitle); r.Settled {
		t.Fatal("project title treated as document identity")
	}
}

func TestSparseCoverWithTwoFontBandsKeepsReportHeading(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Fire Services Specification", Source: identity.Source{Page: 1, X: 50, Y: 515, Height: 13.17}},
		{Text: "Prepared for Client", Source: identity.Source{Page: 1, X: 411, Y: 575, Height: 7.54}},
		{Text: "Project No. JOB1234", Source: identity.Source{Page: 1, X: 411, Y: 555, Height: 7.54}},
		{Text: "Date 12 December 2025", Source: identity.Source{Page: 1, X: 411, Y: 535, Height: 7.54}},
		{Text: "Revision T1", Source: identity.Source{Page: 1, X: 411, Y: 515, Height: 7.54}},
		{Text: "Example project", Source: identity.Source{Page: 1, X: 50, Y: 635, Height: 26.23}},
		{Text: "Accommodation Cabins", Source: identity.Source{Page: 1, X: 50, Y: 610, Height: 14.97}},
		{Text: "Example address", Source: identity.Source{Page: 1, X: 50, Y: 555, Height: 13.17}},
	}}
	got := intake.Harvest("Fire Protection Specification.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "Fire Services Specification"); !ok {
		t.Fatal("cover heading missing")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatal("conflicting filename settled")
	}
}

func TestReportRevisionHistoryMatchesCoverDate(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "13 March 2025", Source: identity.Source{Page: 1}},
		{Text: "Revision", Source: identity.Source{Page: 2, X: 77, Y: 492, Width: 38, Height: 8}},
		{Text: "Date", Source: identity.Source{Page: 2, X: 142, Y: 492, Width: 20, Height: 8}},
		{Text: "Prepared by", Source: identity.Source{Page: 2, X: 216, Y: 492, Width: 52, Height: 8}},
		{Text: "1", Source: identity.Source{Page: 2, X: 94, Y: 471, Width: 5, Height: 8}},
		{Text: "11/03/2025", Source: identity.Source{Page: 2, X: 129, Y: 471, Width: 45, Height: 8}},
		{Text: "2", Source: identity.Source{Page: 2, X: 94, Y: 441, Width: 5, Height: 8}},
		{Text: "13/03/2025", Source: identity.Source{Page: 2, X: 129, Y: 441, Width: 45, Height: 8}},
	}}
	for _, r := range intake.Decide(intake.Harvest("report.pdf", text)) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "2") {
			t.Fatalf("first history row treated as current: %+v", r)
		}
	}
	text.Runs[0].Text = "14 March 2025"
	for _, r := range intake.Decide(intake.Harvest("report.pdf", text)) {
		if r.Field == intake.FieldRevision && r.Settled {
			t.Fatalf("unmatched cover date guessed revision: %+v", r)
		}
	}
}

func TestReportSubjectCorroboratesCoverAndExcludesProjectNumber(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Civil Engineering Report", Source: identity.Source{Page: 1, X: 64, Y: 650, Width: 300, Height: 13}},
		{Text: "Document Information", Source: identity.Source{Page: 2, X: 72, Y: 724, Width: 170, Height: 15}},
		{Text: "Document Subject", Source: identity.Source{Page: 2, X: 77, Y: 603, Width: 79, Height: 8}},
		{Text: "Civil Engineering Report", Source: identity.Source{Page: 2, X: 194, Y: 603, Width: 200, Height: 8}},
		{Text: "Project Number", Source: identity.Source{Page: 2, X: 77, Y: 558, Width: 67, Height: 8}},
		{Text: "123456", Source: identity.Source{Page: 2, X: 194, Y: 558, Width: 30, Height: 8}},
	}}
	for _, r := range intake.Decide(intake.Harvest("Stormwater report.pdf", text)) {
		if r.Field == intake.FieldTitle && (!r.Settled || r.Display != "Civil Engineering Report") {
			t.Fatalf("corroborated cover subject lost: %+v", r)
		}
		if r.Field == intake.FieldNumber && r.Settled {
			t.Fatalf("project reference filed as number: %+v", r)
		}
	}
}

func TestCompactIssueTagAndAmbiguousFilenameDate(t *testing.T) {
	name := "123456 DSP ISSUE03 15-02-20.pdf"
	for _, r := range intake.Decide(intake.Harvest(name, identity.Text{})) {
		if r.Field == intake.FieldRevision && (!r.Settled || r.Display != "03") {
			t.Fatalf("issue tag not revision: %+v", r)
		}
		if r.Field == intake.FieldDate && r.Settled {
			t.Fatalf("ambiguous filename date confirmed: %+v", r)
		}
		if (r.Field == intake.FieldNumber || r.Field == intake.FieldTitle) && r.Settled {
			t.Fatalf("filename job code/acronym confirmed without document evidence: %+v", r)
		}
	}
	for _, r := range intake.Decide(intake.Harvest(name, identity.Text{Runs: []identity.Run{{Text: "Issue Date: 15-02-20"}}})) {
		if r.Field == intake.FieldDate && !r.Settled {
			t.Fatalf("literal document date lost: %+v", r)
		}
	}
}

func TestIssuedWordIsNotRevisionTag(t *testing.T) {
	for _, c := range intake.Harvest("Plan issued for review.pdf", identity.Text{}) {
		if c.Field == intake.FieldRevision {
			t.Fatalf("prose issue mistaken for revision: %+v", c)
		}
	}
}

func TestLandRegistryTitleSystemIsNotDocumentTitle(t *testing.T) {
	for _, c := range intake.Harvest("upload.pdf", identity.Text{Runs: []identity.Run{{Text: "Registered: Title System: LGA: Locality: Parish: County:"}}}) {
		if c.Field == intake.FieldTitle && c.Provenance.Origin == intake.OriginText {
			t.Fatalf("registry caption became document title: %+v", c)
		}
	}
}

func TestExplicitSpacedReportReferenceAndOrdinalDate(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Report Reference: COU - KIT 08/14"}, {Text: "15th August, 2014"}}}
	want := map[string]string{intake.FieldNumber: "COU - KIT 08/14", intake.FieldDate: "15th August, 2014"}
	for _, r := range intake.Decide(intake.Harvest("Arborist Report.pdf", text)) {
		if v, ok := want[r.Field]; ok && (!r.Settled || r.Display != v) {
			t.Errorf("%s: %+v", r.Field, r)
		}
	}
	for _, c := range intake.Harvest("upload.pdf", identity.Text{Runs: []identity.Run{{Text: "Project Reference: COU - KIT 08/14"}}}) {
		if c.Field == intake.FieldNumber {
			t.Fatalf("project ref promoted: %+v", c)
		}
	}
}

func TestOrdinalAndNumericIssueDatesAgree(t *testing.T) {
	for _, pair := range []struct {
		other   string
		settled bool
	}{{"10/10/2024", true}, {"11/10/2024", false}, {"10/10/24", false}} {
		text := identity.Text{Runs: []identity.Run{{Text: "Date of Issue: 10th October 2024"}, {Text: pair.other}}}
		r := fieldResult(t, intake.Decide(intake.Harvest("Report.pdf", text)), intake.FieldDate)
		if r.Settled != pair.settled {
			t.Fatalf("%s: %+v", pair.other, r)
		}
	}
}

func TestDottedReportReferenceWithTypeSuffix(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Report No. P24146.1_GI"}}}
	got := intake.Harvest("Investigation.pdf", text)
	if _, ok := findCandidate(got, intake.FieldNumber, "P24146.1_GI"); !ok {
		t.Fatalf("full report reference missing: %+v", displays(got, intake.FieldNumber))
	}
	if len(displays(got, intake.FieldNumber)) != 1 {
		t.Fatalf("fragmented report reference: %+v", displays(got, intake.FieldNumber))
	}
}

func TestExplicitFinalVersionChallengesNumericFilename(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Version: Final v1.0"}, {Text: "Report: Koala Assessment Report Example Site"}}}
	got := intake.Harvest("Koala Assessment Report [01].pdf", text)
	if _, ok := findCandidate(got, intake.FieldRevision, "Final v1.0"); !ok {
		t.Fatal("full version missing")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldRevision); r.Settled {
		t.Fatalf("filename conflict settled: %+v", r)
	}
	if _, ok := findCandidate(got, intake.FieldTitle, "Koala Assessment Report Example Site"); !ok {
		t.Fatal("full report title missing")
	}
}

func TestDraftAndFinalVersionsRemainDistinct(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Version: Final v1.0"}, {Text: "Version: Draft v1.0"}}}
	got := intake.Harvest("Report.pdf", text)
	for _, value := range []string{"Final v1.0", "Draft v1.0"} {
		if _, ok := findCandidate(got, intake.FieldRevision, value); !ok {
			t.Fatalf("stage lost: %s", value)
		}
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldRevision); r.Settled {
		t.Fatalf("different stages settled: %+v", r)
	}
}

func TestSpatialVersionCaptionRetainsFinalStage(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{
		{Text: "Version:", Source: identity.Source{Page: 2, X: 78, Y: 410, Width: 32, Height: 10}},
		{Text: "Final v1.0", Source: identity.Source{Page: 2, X: 172, Y: 410, Width: 38, Height: 10}},
		{Text: "Version: Draft v1.0"},
	}}
	got := intake.Harvest("Report [01].pdf", text)
	if _, ok := findCandidate(got, intake.FieldRevision, "Final v1.0"); !ok {
		t.Fatal("spatial version omitted")
	}
	text.Runs[1].Source.Page = 3
	if _, ok := findCandidate(intake.Harvest("Report.pdf", text), intake.FieldRevision, "Final v1.0"); ok {
		t.Fatal("version bound across pages")
	}
}

func TestMonthFirstIssueDate(t *testing.T) {
	for _, value := range []string{"SEPTEMBER 4, 2025", "Sep 4, 2025"} {
		got := intake.Harvest("Report.pdf", identity.Text{Runs: []identity.Run{{Text: value}, {Text: "04/09/2025"}}})
		if _, ok := findCandidate(got, intake.FieldDate, value); !ok {
			t.Fatalf("date missing: %s", value)
		}
		if r := fieldResult(t, intake.Decide(got), intake.FieldDate); !r.Settled {
			t.Fatalf("equivalent dates disagree: %+v", r)
		}
	}
	for _, value := range []string{"September 2025", "February 30, 2025", "Sep 4, 25"} {
		for _, c := range intake.Harvest("upload.pdf", identity.Text{Runs: []identity.Run{{Text: value}}}) {
			if c.Field == intake.FieldDate {
				t.Fatalf("invalid or incomplete date accepted: %s", value)
			}
		}
	}
}

func TestSparseManualCoverHeading(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{}}
	for i, value := range []string{"Example Contractor", "EXAMPLE PROJECT", "OPERATION & MAINTENANCE MANUAL", "Prepared by Example", "September 2015"} {
		text.Runs = append(text.Runs, identity.Run{Text: value, Source: identity.Source{Page: 1, X: 100, Y: float64(600 - 30*i), Width: 150, Height: 12}})
	}
	if _, ok := findCandidate(intake.Harvest("STP O&M MANUAL.pdf", text), intake.FieldTitle, "OPERATION & MAINTENANCE MANUAL"); !ok {
		t.Fatal("printed manual title omitted")
	}
	text.Runs[2].Text = "Refer to the operation manual"
	if _, ok := findCandidate(intake.Harvest("upload.pdf", text), intake.FieldTitle, text.Runs[2].Text); ok {
		t.Fatal("body reference treated as heading")
	}
}

func TestDocumentNameCodeAndPreparedHistory(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "Document Name | 25-ABC-1234-MCR"},
		{Text: "Document Control", Source: identity.Source{Page: 3, X: 50, Y: 700, Height: 20}},
		{Text: "Revision", Source: identity.Source{Page: 3, X: 50, Y: 600, Width: 40, Height: 10}},
		{Text: "Date", Source: identity.Source{Page: 3, X: 110, Y: 600, Width: 30, Height: 10}},
		{Text: "Prepared", Source: identity.Source{Page: 3, X: 200, Y: 600, Width: 50, Height: 10}},
		{Text: "R0", Source: identity.Source{Page: 3, X: 60, Y: 570, Width: 20, Height: 10}},
		{Text: "21.01.2026", Source: identity.Source{Page: 3, X: 100, Y: 570, Width: 50, Height: 10}},
		{Text: "R1", Source: identity.Source{Page: 3, X: 60, Y: 530, Width: 20, Height: 10}},
		{Text: "23.02.2026", Source: identity.Source{Page: 3, X: 100, Y: 530, Width: 50, Height: 10}},
	}}
	got := intake.Harvest("Report [1].pdf", text)
	if _, ok := findCandidate(got, intake.FieldNumber, "25-ABC-1234-MCR"); !ok {
		t.Fatal("document name code missing")
	}
	for _, value := range []string{"R0", "R1"} {
		c, ok := findCandidate(got, intake.FieldRevision, value)
		if !ok || !c.Provenance.IssueTable || c.Provenance.OwnRevision {
			t.Fatalf("history revision mistaken for current cell: %+v", c)
		}
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldRevision); r.Settled {
		t.Fatalf("filename conflict settled: %+v", r)
	}
	bad := intake.Harvest("upload.pdf", identity.Text{Runs: []identity.Run{{Text: "Document Name | Annual Report 2026"}}})
	if _, ok := findCandidate(bad, intake.FieldNumber, "Annual Report 2026"); ok {
		t.Fatal("prose title promoted to reference")
	}
}
