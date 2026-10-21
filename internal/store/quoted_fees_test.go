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
}
