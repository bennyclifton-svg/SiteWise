package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestAmbiguousDateIsAskedInSameFanout(t *testing.T) {
	text := identity.Text{TextLayer: true, Runs: []identity.Run{{Text: "Issue date 12/03/2024"}, {Text: "Printed 14/03/2024"}}}
	d := intake.NewDraft(draftCatalog(t), "plan.pdf", text, intake.Harvest("plan.pdf", text), nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("no fanout")
	}
	if _, ok := call.Questions[intake.FieldDate]; !ok {
		t.Fatal("ambiguous date silently abandoned")
	}
}

func TestExplicitRevisionDatePairDoesNotChoosePrintDate(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "REV B 06.11.2023"}, {Text: "Printed 14/03/2024"}}}
	for _, r := range intake.Decide(intake.Harvest("A101 [B].pdf", text)) {
		if r.Field == intake.FieldDate && (!r.Settled || r.Display != "06.11.2023") {
			t.Fatalf("wrong issue date: %+v", r)
		}
	}
	text.Runs = append(text.Runs, identity.Run{Text: "REV B 07.11.2023"})
	for _, r := range intake.Decide(intake.Harvest("A101 [B].pdf", text)) {
		if r.Field == intake.FieldDate && r.Settled {
			t.Fatal("conflicting explicit dates must remain open")
		}
	}
}

func TestIssueDateColumnWithPrintedFormat(t *testing.T) {
	run := func(v string, x, y, w float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: 7}}
	}
	for _, caption := range []string{"Date (dd.mm.yy)", "Date (dd/mm/yyyy)"} {
		text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
			run("Rev.", 48, 753, 15), run(caption, 73, 753, 58),
			run("000", 48, 744, 12), run("31.10.25", 73, 744, 27),
			run("Printed 02/11/2025", 48, 600, 100),
		}}
		found := false
		for _, r := range intake.Decide(intake.Harvest("Report [0].pdf", text)) {
			if r.Field == intake.FieldDate {
				found = true
				if !r.Settled || r.Display != "31.10.25" {
					t.Fatalf("%s: %+v", caption, r)
				}
			}
		}
		if !found {
			t.Fatal("missing date decision")
		}
	}
}

func TestDistributionRevisionColumnChallengesFilename(t *testing.T) {
	run := func(v string, x, y, w float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 3, X: x, Y: y, Width: w, Height: 9}}
	}
	text := identity.Text{Runs: []identity.Run{run("Issue", 38, 522, 24), run("Revision Issued To", 94, 522, 94), run("Date", 258, 522, 22), run("0", 94, 497, 5), run("08/09/2025", 258, 497, 51), run("0", 94, 470, 5), run("18/09/2025", 258, 470, 51), {Text: "Date: 18-09-2025", Source: identity.Source{Page: 1}}}}
	got := intake.Harvest("Report [A].pdf", text)
	if _, ok := findCandidate(got, intake.FieldRevision, "0"); !ok {
		t.Fatal("printed distribution revision missing")
	}
	r := fieldResult(t, intake.Decide(got), intake.FieldRevision)
	if r.Settled && r.Display == "A" {
		t.Fatal("conflicting filename revision confirmed")
	}

	date := fieldResult(t, intake.Decide(got), intake.FieldDate)
	if !date.Settled || date.Display != "18-09-2025" {
		t.Fatalf("corroborated cover date lost: %+v", date)
	}
	text.Runs = append(text.Runs, run("03/10/2025", 400, 300, 50))
	if date := fieldResult(t, intake.Decide(intake.Harvest("Report [A].pdf", text)), intake.FieldDate); date.Settled {
		t.Fatal("unrelated date conflict ignored")
	}

}
