package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

func TestManualFirmConfirmedFeeDoesNotAddItsSchedule(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Manual Planning"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var key string
	for _, it := range table.Items {
		if it.Base {
			key = it.Key
			break
		}
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, key, store.CellInput{State: "priced", Amount: "100"}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: "500"}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("missing scope allowed: %v", err)
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: "500", Note: "Complete appointment including the component"}); err != nil {
		t.Fatal(err)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c := table.Columns[0]; c.Quote != nil || c.Totals.BaseTotal == nil || *c.Totals.BaseTotal != 50000 {
		t.Fatalf("manual confirmed fee double counted: %+v", c)
	}
	_, err = s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Duplicate", Keys: []string{key, "comparison:appointment"}}}})
	if !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("mixed award allowed: %v", err)
	}
	if err := s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Label: "Agreed contingency", Section: "base"}); err != nil {
		t.Fatal(err)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if table.Columns[0].AppointmentTotal != nil {
		t.Fatal("unpriced addition must leave total unknown")
	}
	var extra string
	for _, item := range table.Items {
		if item.Label == "Agreed contingency" {
			extra = item.Key
		}
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, extra, store.CellInput{State: "priced", Amount: "75", Note: "Agreed with consultant"}); err != nil {
		t.Fatal(err)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	c := table.Columns[0]
	if c.AppointmentTotal == nil || *c.AppointmentTotal != 57500 || c.Totals.BaseTotal == nil || *c.Totals.BaseTotal != 57500 {
		t.Fatalf("addition not counted exactly once: %+v", c)
	}
	preview, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Professional fee", Keys: []string{"comparison:appointment"}}, {Label: "Contingency", Keys: []string{extra}}}})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Total != 57500 {
		t.Fatalf("award total: %+v", preview)
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, extra, store.CellInput{State: "included", Note: "Included in professional fee"}); err != nil {
		t.Fatal(err)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c := table.Columns[0]; c.AppointmentTotal == nil || *c.AppointmentTotal != 50000 {
		t.Fatal("included addition counted twice")
	}

}
