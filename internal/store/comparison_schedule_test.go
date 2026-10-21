package store_test

import (
	"context"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
)

func TestComparisonOverridesSurviveRFPRefresh(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	_, r, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, pkg.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.PackageSchedule(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var key string
	for _, it := range items {
		if it.Base {
			key = it.Key
			break
		}
	}
	if key == "" {
		t.Fatal("missing base row")
	}
	if err = s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Key: key, Label: "Reviewed scope", Section: "option"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetReportSetting(ctx, orgA, r.ID, userA, key, []byte(`"exclude"`), d.Version); err != nil {
		t.Fatal(err)
	}
	items, err = s.PackageSchedule(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range items {
		if it.Key == key {
			found = true
			if it.Retired || it.Base || it.Label != "Reviewed scope" {
				t.Fatalf("override replaced: %+v", it)
			}
		}
	}
	if !found {
		t.Fatal("override lost")
	}
	form, err := s.ReadResponseForm(ctx, orgA, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range form.Items {
		if it.Label == "Reviewed scope" {
			t.Fatal("comparison override changed RFP")
		}
	}
}
