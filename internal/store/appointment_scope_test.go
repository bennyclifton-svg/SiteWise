package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

func appointmentFixture(t *testing.T) (*store.Store, string, string) {
	t.Helper()
	ctx := context.Background()
	s, _, _ := workStore(t)
	def, _ := procurement.DisciplinePackage("consultant.structural", "Structural")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, def)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Scope Engineers"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	amount := func(s string) *string { return &s }
	review := proposals.QuoteReview{Coverage: map[string]proposals.QuoteCoverage{}, Fees: []proposals.QuotedFee{
		{ID: "total", Label: "Total professional fee", Amount: amount("41800.00"), Role: proposals.RoleTotal},
		{ID: "schematic", Label: "Schematic design", Amount: amount("12540.00"), Role: proposals.RoleComponent},
		{ID: "detailed", Label: "Detailed design", Amount: amount("20900.00"), Role: proposals.RoleComponent},
		{ID: "construction", Label: "Construction services", Amount: amount("8360.00"), Role: proposals.RoleComponent},
	}}
	raw, _ := json.Marshal(review)
	if _, err := rawPool(t).Exec(ctx, `UPDATE submissions SET quote_review=$3::jsonb,stated_total_cents=4180000,gst_basis='ex' WHERE org_id=$1 AND id=$2`, orgA, sub, string(raw)); err != nil {
		t.Fatal(err)
	}
	return s, pkg.ID, sub
}

func TestAppointmentSelectionCarriesThroughAward(t *testing.T) {
	s, pkg, sub := appointmentFixture(t)
	ctx := context.Background()
	read := func() store.FirmColumn {
		t.Helper()
		table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg)
		if err != nil {
			t.Fatal(err)
		}
		return table.Columns[0]
	}
	col := read()
	if col.AppointmentTotal == nil || *col.AppointmentTotal != 4180000 {
		t.Fatalf("total %+v", col)
	}
	selectRow := func(id string, included bool, keys []string) {
		t.Helper()
		col = read()
		if err := s.SetAppointmentRow(ctx, orgA, projectA, sub, store.AppointmentRowInput{Version: col.Submission.Version, ID: id, AppointmentChoice: store.AppointmentChoice{Included: included, Keys: keys}}); err != nil {
			t.Fatal(err)
		}
	}
	old := col.Submission.Version
	selectRow("quote:construction", false, nil)
	col = read()
	if col.AppointmentTotal == nil || *col.AppointmentTotal != 3344000 {
		t.Fatalf("excluded fee still in total %+v", col.AppointmentTotal)
	}
	if err := s.SetAppointmentRow(ctx, orgA, projectA, sub, store.AppointmentRowInput{Version: old, ID: "quote:construction", AppointmentChoice: store.AppointmentChoice{Included: true}}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("stale edit %v", err)
	}
	if err := s.SetAppointmentRow(ctx, orgB, projectA, sub, store.AppointmentRowInput{Version: col.Submission.Version, ID: "quote:construction"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org %v", err)
	}
	if err := s.SetAppointmentRow(ctx, orgA, projectA, sub, store.AppointmentRowInput{Version: col.Submission.Version, ID: "quote:missing"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown row %v", err)
	}
	// Manual mapping folds the requested stage into the existing source row.
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	var stage string
	for _, it := range table.Items {
		if it.Kind == "stage" {
			stage = it.Key
			break
		}
	}
	if stage == "" {
		t.Fatal("no stage fixture")
	}
	selectRow("quote:schematic", true, []string{stage})
	col = read()
	for _, row := range col.AppointmentRows {
		if row.ID == "item:"+stage {
			t.Fatal("mapped stage shown twice")
		}
	}
	selectRow("quote:schematic", true, []string{})
	col = read()
	for _, row := range col.AppointmentRows {
		if row.ID == "quote:schematic" && len(row.Keys) > 0 {
			t.Fatal("empty mapping did not persist")
		}
	}
	selectRow("quote:schematic", true, []string{stage})
	col = read()
	input := store.AwardInput{SubmissionID: sub, Appointment: true, Version: col.Submission.Version, Total: "33440.00", Lines: []store.AwardLine{{Label: "Design", AppointmentIDs: []string{"quote:schematic", "quote:detailed"}}}}
	invalid := input
	invalid.Lines = []store.AwardLine{{Label: "Excluded", AppointmentIDs: []string{"quote:construction"}}}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg, invalid); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("excluded row awarded %v", err)
	}
	invalid.Lines = []store.AwardLine{{Label: "Duplicate", AppointmentIDs: []string{"quote:schematic", "quote:schematic", "quote:detailed"}}}
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg, invalid); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("duplicate awarded %v", err)
	}
	invalid = input
	invalid.Appointment = false
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg, invalid); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("legacy bypass %v", err)
	}
	// Changing selected scope after preview invalidates the old request.
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg, input); err != nil {
		t.Fatal(err)
	}
	selectRow("quote:construction", true, nil)
	if _, err := s.Award(ctx, orgA, projectA, pkg, userA, input); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("stale award %v", err)
	}
	selectRow("quote:construction", false, nil)
	col = read()
	input.Version = col.Submission.Version
	award, err := s.Award(ctx, orgA, projectA, pkg, userA, input)
	if err != nil {
		t.Fatal(err)
	}
	if award.Total != 3344000 || len(award.Lines) != 1 || len(award.Lines[0].QuoteIDs) != 2 {
		t.Fatalf("award %+v", award)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	children := 0
	for _, it := range plan.Items {
		if it.ParentItemID == award.ParentItemID {
			children++
			if it.Label != "Design" || it.Values["commitment"].Amount == nil || string(*it.Values["commitment"].Amount) != "33440.00" {
				t.Fatalf("cost child %+v", it)
			}
			if _, ok := it.Values["budget"]; ok {
				t.Fatal("budget written")
			}
		}
	}
	if children != 1 {
		t.Fatalf("children %d", children)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg, userA); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ReadAward(ctx, orgA, projectA, pkg); got != nil {
		t.Fatal("undo left award")
	}
	col = read()
	if *col.AppointmentTotal != 3344000 {
		t.Fatal("undo changed selected scope")
	}
	// Included but unpriced scope blocks award; included-in-fee scope does not
	// manufacture a zero-value monetary cost line.
	selectRow("quote:schematic", true, []string{})
	selectRow("item:"+stage, true, nil)
	if col = read(); col.AppointmentTotal != nil {
		t.Fatal("unknown scope became zero")
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, sub, stage, store.CellInput{State: "included", Note: "Included in design"}); err != nil {
		t.Fatal(err)
	}
	col = read()
	if col.AppointmentTotal == nil || *col.AppointmentTotal != 3344000 {
		t.Fatal("included scope changed money")
	}
	input.Version = col.Submission.Version
	if _, err := s.PreviewAward(ctx, orgA, projectA, pkg, input); err != nil {
		t.Fatal(err)
	}
}

func TestAppointmentSpeedBudgets(t *testing.T) {
	s, pkg, sub := appointmentFixture(t)
	ctx := context.Background()
	samples := map[string][]time.Duration{}
	for i := 0; i < 20; i++ {
		start := time.Now()
		table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg)
		if err != nil {
			t.Fatal(err)
		}
		samples["read"] = append(samples["read"], time.Since(start))
		col := table.Columns[0]
		start = time.Now()
		err = s.SetAppointmentRow(ctx, orgA, projectA, sub, store.AppointmentRowInput{Version: col.Submission.Version, ID: "quote:construction", AppointmentChoice: store.AppointmentChoice{Included: i%2 == 0}})
		if err != nil {
			t.Fatal(err)
		}
		table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg)
		if err != nil {
			t.Fatal(err)
		}
		samples["write"] = append(samples["write"], time.Since(start))
		col = table.Columns[0]
		lines := []store.AwardLine{}
		for _, row := range col.AppointmentRows {
			if row.Included && row.Amount != nil {
				lines = append(lines, store.AwardLine{Label: row.Label, AppointmentIDs: []string{row.ID}})
			}
		}
		start = time.Now()
		_, err = s.Award(ctx, orgA, projectA, pkg, userA, store.AwardInput{SubmissionID: sub, Appointment: true, Version: col.Submission.Version, Lines: lines})
		if err != nil {
			t.Fatal(err)
		}
		samples["award"] = append(samples["award"], time.Since(start))
		if err = s.UndoAward(ctx, orgA, projectA, pkg, userA); err != nil {
			t.Fatal(err)
		}
	}
	for name, budget := range map[string][2]time.Duration{"read": {150 * time.Millisecond, 300 * time.Millisecond}, "write": {100 * time.Millisecond, 250 * time.Millisecond}, "award": {200 * time.Millisecond, 500 * time.Millisecond}} {
		values := samples[name]
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		p50, p90 := values[9], values[17]
		t.Logf("%s p50=%s p90=%s budget=%s/%s (local store, write includes refreshed table)", name, p50, p90, budget[0], budget[1])
		if p50 > budget[0] || p90 > budget[1] {
			t.Errorf("%s exceeds budget", name)
		}
	}
}
