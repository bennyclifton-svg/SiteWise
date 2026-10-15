package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/specifications"
	"sitewise/internal/store"
)

func TestSpecificationAuthoringReviewIsolationAndFrozenIssue(t *testing.T) {
	s, _, part := workStore(t)
	ctx := context.Background()
	tile, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Code: "TIL1", Title: "Bathroom tile", Kind: "material", SystemID: "interiors.tiling"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationType(ctx, orgB, projectA, userB, tile); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign edit: %v", err)
	}
	stale := tile
	tile.Title = "Selected tile"
	tile, err = s.SaveSpecificationType(ctx, orgA, projectA, userA, tile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationType(ctx, orgA, projectA, userA, stale); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale: %v", err)
	}
	app, err := s.SaveSpecificationApplication(ctx, orgA, projectA, userA, specifications.Application{TypeID: tile.ID, PartID: part, Location: "Bathroom", Surface: "floor", Action: "new", Inclusion: "included", Selection: "selected"})
	if err != nil {
		t.Fatal(err)
	}
	value, err := s.AddSpecificationAssertion(ctx, orgA, projectA, userA, specifications.Assertion{TypeID: tile.ID, ApplicationID: app.ID, Key: "grout", Meaning: "selection", Value: specifications.Value{State: "unknown"}})
	if err != nil {
		t.Fatal(err)
	}
	value, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, value.ID, store.SpecificationReview{Version: value.Version, Status: "accepted"})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(t.TempDir(), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	issue, err := s.IssueSpecification(ctx, orgA, projectA, userA, "Materials and finishes", "preliminary", snap.Revision, blobs)
	if err != nil {
		t.Fatal(err)
	}
	newValue, err := s.AddSpecificationAssertion(ctx, orgA, projectA, userA, specifications.Assertion{TypeID: tile.ID, ApplicationID: app.ID, Key: "grout", Meaning: "selection", Value: specifications.Value{State: "known", Text: "Grey"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, newValue.ID, store.SpecificationReview{Version: 1, Status: "accepted"}); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("accepted replacement without checking previous decision", err)
	}
	_, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, newValue.ID, store.SpecificationReview{Version: 1, Status: "accepted", ExpectedAcceptedID: value.ID, Reason: "Owner selected grey"})
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := s.GetSpecificationIssue(ctx, orgA, projectA, issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := specifications.Assemble(frozen.Snapshot)[0].Fields[0].Value; got != "unknown" {
		t.Fatal("issued schedule changed", got)
	}
	if _, err = s.GetSpecificationIssue(ctx, orgB, projectA, issue.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign issue", err)
	}
	if _, err = s.ReadSpecifications(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign read", err)
	}
}

func TestSpecificationSourceReviewAndCycles(t *testing.T) {
	s, _, _ := workStore(t)
	ctx := context.Background()
	parent, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Industrial protective coating", Kind: "assembly"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Primer", Kind: "finish"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AddSpecificationRelationship(ctx, orgA, projectA, userA, specifications.Relationship{FromID: child.ID, ToID: parent.ID, Kind: "component_of", Sequence: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddSpecificationRelationship(ctx, orgA, projectA, userA, specifications.Relationship{FromID: parent.ID, ToID: child.ID, Kind: "requires"}); !errors.Is(err, store.ErrInvalidSpecification) {
		t.Fatal("cycle accepted", err)
	}
	a := specifications.Assertion{TypeID: child.ID, Key: "finish", Meaning: "selection", Value: specifications.Value{State: "known", Text: "White"}, Source: specifications.Source{DocumentID: docB, Page: 1, Excerpt: "White"}}
	if _, err = s.AddSpecificationAssertion(ctx, orgA, projectA, userA, a); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign source accepted", err)
	}
	a.Source.DocumentID = docA
	a.Source.Marker = "withdrawn"
	a, err = s.AddSpecificationAssertion(ctx, orgA, projectA, userA, a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, a.ID, store.SpecificationReview{Version: 1, Status: "accepted", SourceReviewed: true}); !errors.Is(err, store.ErrInvalidSpecification) {
		t.Fatal("withdrawn source accepted", err)
	}
	if a.Source.Hash == "" || a.Status != "proposed" {
		t.Fatal("source identity or proposal lost", a)
	}
}
