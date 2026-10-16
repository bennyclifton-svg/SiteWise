package store_test

import (
	"context"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// An RFP draft stores the schedule it prints; changing a stage changes the
// rows with no code change; a package with no RFP gets its discipline's
// default schedule; another org sees nothing.
func TestRFPDraftWritesItsSchedule(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "industrial", "hdr.work_type": "extend"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: map[string]string{"hdr.building_class": "site", "hdr.work_type": "project"}[key], Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	def, err := s.PackageSchedule(ctx, orgA, projectA, pkg.ID)
	if err != nil || len(def) == 0 || def[0].Source != store.ScheduleFromDefault {
		t.Fatalf("default schedule %+v %v", def, err)
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
	var stage string
	prices := 0
	for _, it := range items {
		if it.Source != store.ScheduleFromRFP && !it.Retired {
			t.Fatalf("live row not from the RFP: %+v", it)
		}
		if it.Kind == "stage" && stage == "" && !it.Retired {
			stage = it.Key
		}
		if it.Price() {
			prices++
		}
	}
	if stage == "" || prices < 3 {
		t.Fatalf("schedule %+v", items)
	}
	if _, err := s.SetReportSetting(ctx, orgA, r.ID, userA, stage, []byte(`"exclude"`), d.Version); err != nil {
		t.Fatal(err)
	}
	items, err = s.PackageSchedule(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Key == stage && !it.Retired {
			t.Fatalf("excluded stage still live: %+v", it)
		}
	}
	if _, err := s.PackageSchedule(ctx, orgB, projectA, pkg.ID); err == nil {
		t.Fatal("another org read the schedule")
	}
}
