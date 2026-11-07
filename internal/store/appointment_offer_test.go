package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

// A fee letter with a task table and no labelled total: the tasks go above
// the line, add to the computed total and award as they are. Moving a row
// is a placement; saving a stage mapping is not.
func TestAppointmentOfferLinesDefaultAboveTheLine(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	def, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, def)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Offer Planners"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	amount := func(s string) *string { return &s }
	review := proposals.QuoteReview{Coverage: map[string]proposals.QuoteCoverage{}, Fees: []proposals.QuotedFee{
		{ID: "a1", Label: "Task 1 - Preliminary Planning Advice Report", Printed: "$1,000", Amount: amount("1000.00"), Role: "Package / fee — scope awaiting alignment", Page: 3},
		{ID: "a2", Label: "Task 2 - Pre-DA Meeting", Printed: "$500", Amount: amount("500.00"), Role: "Package / fee — scope awaiting alignment", Page: 3},
		{ID: "a3", Label: "Task 3 - Collation and Lodgement", Printed: "$250", Amount: amount("250.00"), Role: "Package / fee — scope awaiting alignment", Page: 3},
		{ID: "a4", Label: "Site visit", Printed: "$400", Amount: amount("400.00"), Role: "Package / fee — scope awaiting alignment", Suggestion: "Conditional / optional", Page: 3},
		{ID: "a5", Label: "Public liability", Printed: "$20,000,000", Amount: amount("20000000.00"), Role: "Package / fee — scope awaiting alignment", Source: "Public Liability  Insurer  123  $20,000,000", Page: 9},
	}}
	raw, _ := json.Marshal(review)
	if _, err := rawPool(t).Exec(ctx, `UPDATE submissions SET quote_review=$3::jsonb,gst_basis='ex' WHERE org_id=$1 AND id=$2`, orgA, sub, string(raw)); err != nil {
		t.Fatal(err)
	}
	read := func() store.FirmColumn {
		t.Helper()
		table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
		if err != nil {
			t.Fatal(err)
		}
		return table.Columns[0]
	}
	row := func(col store.FirmColumn, id string) store.AppointmentRow {
		t.Helper()
		for _, r := range col.AppointmentRows {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("no row %s in %+v", id, col.AppointmentRows)
		return store.AppointmentRow{}
	}
	col := read()
	if col.AppointmentTotal == nil || *col.AppointmentTotal != 175000 || col.AppointmentBasis != store.BasisListed {
		t.Fatalf("total %v basis %q", col.AppointmentTotal, col.AppointmentBasis)
	}
	for _, id := range []string{"quote:a1", "quote:a2", "quote:a3"} {
		if !row(col, id).Included {
			t.Fatalf("%s below the line", id)
		}
	}
	if row(col, "quote:a4").Included {
		t.Fatal("optional fee above the line")
	}
	for _, r := range col.AppointmentRows {
		if r.ID == "quote:a5" {
			t.Fatal("insurance cover listed as a fee")
		}
	}
	// The stage names in the labels map the rows to the schedule. Planning
	// is one project stage (7 November 2026), so the first planning task
	// claims it and the lodgement task stays for the user to map.
	if r := row(col, "quote:a1"); len(r.Keys) != 1 || r.Keys[0] != "stage:planning" || !r.MappingSuggested {
		t.Fatalf("label mapping %+v", r)
	}
	if r := row(col, "quote:a3"); len(r.Keys) != 0 {
		t.Fatalf("label mapping %+v", r)
	}
	set := func(id string, included bool, keys []string) {
		t.Helper()
		col = read()
		if err := s.SetAppointmentRow(ctx, orgA, projectA, sub, store.AppointmentRowInput{Version: col.Submission.Version, ID: id, AppointmentChoice: store.AppointmentChoice{Included: included, Keys: keys}}); err != nil {
			t.Fatal(err)
		}
	}
	// A mapping save keeps the row where it is.
	set("quote:a2", true, []string{"stage:brief_concept"})
	if col = read(); !row(col, "quote:a2").Included || row(col, "quote:a2").Keys[0] != "stage:brief_concept" {
		t.Fatal("mapping save moved the row")
	}
	// An arrow moves it; a later mapping save keeps that place.
	set("quote:a2", false, nil)
	set("quote:a2", false, []string{})
	if col = read(); row(col, "quote:a2").Included || *col.AppointmentTotal != 125000 {
		t.Fatalf("arrow lost %+v %v", row(col, "quote:a2"), col.AppointmentTotal)
	}
	set("quote:a4", true, nil)
	if col = read(); *col.AppointmentTotal != 165000 {
		t.Fatalf("optional fee not added %v", *col.AppointmentTotal)
	}
	// Award takes exactly the included rows.
	input := store.AwardInput{SubmissionID: sub, Appointment: true, Version: col.Submission.Version, Lines: []store.AwardLine{{Label: "Planning", AppointmentIDs: []string{"quote:a1", "quote:a3", "quote:a4"}}}}
	view, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, input)
	if err != nil || view.Total != 165000 {
		t.Fatalf("preview %+v %v", view, err)
	}
}
