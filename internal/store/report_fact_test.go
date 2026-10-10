package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/rfp"
	"sitewise/internal/store"
)

func factValue(t *testing.T, s *store.Store, r store.Report, key string) rfp.Control {
	t.Helper()
	view, err := s.ReadReport(context.Background(), orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	return rfpControl(view.RFP, key)
}

// A project fact edited in an RFP is written to its one home in Profile and
// the draft re-reads the project (owner decision D1).
func TestRFPFactWritesToProfile(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	r, d := rfpProject(t, s, b, part)
	str := func(v string) *string { return &v }

	d, err := s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "zone", str("R2"), d.Version)
	if err != nil {
		t.Fatal(err)
	}
	if c := factValue(t, s, r, "fact:zone"); c.Value != "R2" || c.Source != "profile" {
		t.Fatalf("zone = %+v", c)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || len(view.Stale) != 0 {
		t.Fatalf("the draft is current after its own edit: %v %v", view.Stale, err)
	}

	// A flag with a determinant home writes the determinant.
	if d, err = s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "flag.existing_building", str("no"), d.Version); err != nil {
		t.Fatal(err)
	}
	if c := factValue(t, s, r, "fact:flag.existing_building"); c.Value != "no" || c.Source != "profile" {
		t.Fatalf("existing building = %+v", c)
	}
	// The address lives on the site record.
	if d, err = s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "address", str("2 New St, Petersham NSW"), d.Version); err != nil {
		t.Fatal(err)
	}
	if site, _ := s.ProjectSite(ctx, orgA, projectA); site.Address != "2 New St, Petersham NSW" {
		t.Fatalf("site address = %q", site.Address)
	}
	// Clearing hands the fact back to the documents.
	if d, err = s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "zone", nil, d.Version); err != nil {
		t.Fatal(err)
	}
	if c := factValue(t, s, r, "fact:zone"); c.Value != "" {
		t.Fatalf("cleared zone = %+v", c)
	}

	for name, value := range map[string]string{"budget": "100", "flag.heritage": "yes", "flag.mixed_use": "yes", "work_type": "castle", "nope": "x", "flag.basement": "maybe", "site_area": "big"} {
		if _, err := s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", name, str(value), d.Version); !errors.Is(err, store.ErrInvalidSetting) {
			t.Fatalf("%s=%s must be refused: %v", name, value, err)
		}
	}
	if _, err := s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "zone", str("B4"), d.Version-1); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	if _, err := s.SetReportFact(ctx, orgB, r.ID, userB, "test-build", "zone", str("B4"), d.Version); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's report: %v", err)
	}
}

// Another draft that read the old value goes stale; it is not rewritten.
func TestRFPFactMakesOtherDraftsStale(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	r, d := rfpProject(t, s, b, part)
	other, err := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshReport(ctx, orgA, other.ID, userA, "test-build", true); err != nil {
		t.Fatal(err)
	}
	zone := "R2"
	if _, err := s.SetReportFact(ctx, orgA, r.ID, userA, "test-build", "zone", &zone, d.Version); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadReport(ctx, orgA, other.ID, "test-build")
	if err != nil || len(view.Stale) == 0 {
		t.Fatalf("the PMP read the old profile and must show it: %v %v", view.Stale, err)
	}
}
