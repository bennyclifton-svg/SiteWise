package store_test

import (
	"context"
	"errors"
	"sitewise/internal/store"
	"testing"
)

func TestDocumentLocationPartsRespectOwnership(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	a, err := s.CreatePart(ctx, orgA, projectA, "Plant room", "plant_area", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePart(ctx, orgB, projectB, "Other plant room", "plant_area", ""); err != nil {
		t.Fatal(err)
	}
	parts, err := s.DocumentParts(ctx, orgA, docA)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range parts {
		found = found || p.ID == a.ID
		if p.Label == "Other plant room" {
			t.Fatal("foreign site part exposed")
		}
	}
	if !found {
		t.Fatal("own site part missing")
	}
	if _, err := s.DocumentParts(ctx, orgB, docA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign document %v", err)
	}
}
