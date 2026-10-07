package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

func TestSiteReadingRoutingPersistsProvenance(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	compute := func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User,
			Thresholds: profile.Thresholds{Amber: map[string]float64{"determinant": .6}}}, cat)
	}
	facts := []store.StoredFact{
		{PassageID: passageA, QuestionID: "det.existing_building_year", Value: "1985", Excerpt: "Built in 1985", Confidence: conf(.9), DecidedBy: "jev"},
		{PassageID: passageA, QuestionID: "det.existing_building", Value: "stated_true", Confidence: conf(.9), DecidedBy: "jev"},
	}
	if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"det."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	rebuild := func() profile.Row {
		t.Helper()
		if err := st.RebuildProfile(ctx, orgA, projectA, "wp15", compute); err != nil {
			t.Fatal(err)
		}
		view, err := st.ReadProfile(ctx, orgA, projectA, nil)
		if err != nil {
			t.Fatal(err)
		}
		var year profile.Row
		for _, r := range view.Rows {
			if r.Key == "det.existing_building" && r.Scope != "project" {
				t.Fatalf("project key routed to site: %+v", r)
			}
			if r.Key == "det.existing_building_year" {
				year = r
			}
		}
		if year.Scope != "site" || year.PartID != whole.ID || len(year.Sources) != 1 || year.Sources[0].DocumentID != docA || year.Sources[0].PassageID != passageA || year.Sources[0].Excerpt != "Built in 1985" {
			t.Fatalf("persisted provenance: %+v", year)
		}
		return year
	}
	if r := rebuild(); r.Value != "1985" {
		t.Fatalf("reading: %+v", r)
	}
	if _, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "det.existing_building_year", store.UserWrite{Scope: "site", Value: strPtr("1990")}); err != nil {
		t.Fatal(err)
	}
	if r := rebuild(); r.Value != "1990" || r.Band != "user" {
		t.Fatalf("user precedence: %+v", r)
	}
	if _, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "det.ncc_class", store.UserWrite{Scope: "site", Value: strPtr("7b")}); err != nil {
		t.Fatal(err)
	}
	rebuild()
	view, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	inputFound := false
	for _, r := range view.Rows {
		if r.Derived == nil {
			continue
		}
		for _, input := range r.Derived.Provenance.Inputs {
			if input.Ref == whole.ID+"/det.ncc_class" {
				inputFound = true
				if input.Origin != profile.OriginUser || input.ReviewStatus != profile.ReviewAccepted {
					t.Fatalf("input provenance changed in persistence: %+v", input)
				}
			}
		}
	}
	if !inputFound {
		t.Fatal("derived input provenance lost in persistence")
	}
	if _, err := st.ReadProfile(ctx, orgB, projectA, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org: %v", err)
	}
}
