package store_test

import (
	"context"
	"sitewise/internal/specifications"
	"testing"
)

func TestSpecificationStarterRetriesAndRetirement(t *testing.T) {
	s, _, _ := workStore(t)
	ctx := context.Background()
	prompt := specifications.Type{Title: "Paint system", Code: "P1", Kind: "finish", SystemID: "interiors.painting", StarterKey: "paint", Origin: "document"}
	first, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, prompt)
	if err != nil {
		t.Fatal(err)
	}
	if first.Origin != "user" || first.Authority != "sitewise" {
		t.Fatal("client forged source origin", first)
	}
	retry, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, prompt)
	if err != nil || retry.ID != first.ID {
		t.Fatal("retry duplicated starter", retry, err)
	}
	first.Status = "retired"
	if _, err = s.SaveSpecificationType(ctx, orgA, projectA, userA, first); err != nil {
		t.Fatal(err)
	}
	next, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, prompt)
	if err != nil || next.ID == first.ID || next.Status != "active" {
		t.Fatal("retired starter reused", next, err)
	}
}
