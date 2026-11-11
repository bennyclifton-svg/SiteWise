package store_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"sitewise/internal/profile"
	"sitewise/internal/store"
)

func storedByKey(t *testing.T, s *store.Store) map[string]profile.StoredLine {
	t.Helper()
	lines, err := s.AccommodationLines(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]profile.StoredLine{}
	for _, l := range lines {
		k := l.Key
		if k == "" {
			k = l.Name
		}
		out[k] = l
	}
	return out
}

// The schedule keeps only the user's word: a field set on a generated line,
// a removal, a split, a line added and its place
// (docs/plans/2026-11-11-accommodation-schedule-lines.md).
func TestAccommodationEdits(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	if err := s.SetAccommodationField(ctx, orgA, projectA, []string{"offices#1", "offices#2"}, "level", "Level 3"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAccommodationField(ctx, orgA, projectA, []string{"offices#1"}, "area", "12.5"); err != nil {
		t.Fatal(err)
	}
	got := storedByKey(t, s)
	o := got["offices#1"]
	if o.Level != "Level 3" || o.Area != "12.5" || !slices.Equal(o.Set, []string{"area", "level"}) || got["offices#2"].Level != "Level 3" {
		t.Fatalf("set = %+v / %+v", o, got["offices#2"])
	}
	// A blank the user sets stays set: the reading does not come back.
	if err := s.SetAccommodationField(ctx, orgA, projectA, []string{"offices#2"}, "level", ""); err != nil {
		t.Fatal(err)
	}
	if o := storedByKey(t, s)["offices#2"]; o.Level != "" || !slices.Contains(o.Set, "level") {
		t.Fatalf("blank = %+v", o)
	}
	if err := s.RemoveAccommodationLines(ctx, orgA, projectA, []string{"reception#1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SplitAccommodationLine(ctx, orgA, projectA, "guest_rooms#1"); err != nil {
		t.Fatal(err)
	}
	ids, err := s.PlaceAccommodationLines(ctx, orgA, projectA, []string{"offices#2", "new:0", "offices#1"},
		[]profile.StoredLine{{Name: "Server cupboard", Comment: "Dedicated cooling", Set: []string{"name", "comment"}}})
	if err != nil || len(ids) != 1 {
		t.Fatalf("place = %v %v", ids, err)
	}
	got = storedByKey(t, s)
	if !got["reception#1"].Removed || !got["guest_rooms#1"].Split {
		t.Fatalf("remove/split = %+v / %+v", got["reception#1"], got["guest_rooms#1"])
	}
	u := got["Server cupboard"]
	if u.ID != ids[0] || u.Key != "" || u.Comment != "Dedicated cooling" || u.Position == nil || *u.Position != 1 ||
		*got["offices#2"].Position != 0 || *got["offices#1"].Position != 2 {
		t.Fatalf("placed = %+v, offices %v %v", u, got["offices#2"].Position, got["offices#1"].Position)
	}
	// The user's own line takes edits by its id, and goes when deleted.
	if err := s.SetAccommodationField(ctx, orgA, projectA, []string{u.ID}, "name", "Comms cupboard"); err != nil {
		t.Fatal(err)
	}
	if storedByKey(t, s)["Comms cupboard"].ID != u.ID {
		t.Fatal("rename by id lost")
	}
	// Reset hands a generated line back to the documents.
	if err := s.ResetAccommodationLines(ctx, orgA, projectA, []string{"offices#1", "reception#1"}); err != nil {
		t.Fatal(err)
	}
	got = storedByKey(t, s)
	if o := got["offices#1"]; len(o.Set) != 0 || o.Level != "" || o.Area != "" || got["reception#1"].Removed {
		t.Fatalf("reset = %+v / %+v", o, got["reception#1"])
	}
	if err := s.RemoveAccommodationLines(ctx, orgA, projectA, []string{u.ID}); err != nil {
		t.Fatal(err)
	}
	if _, ok := storedByKey(t, s)["Comms cupboard"]; ok {
		t.Fatal("the user's line was kept")
	}
}

// Another org's project, an unknown line id and a malformed key are refused.
func TestAccommodationEditsAreScoped(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	if err := s.SetAccommodationField(ctx, orgB, projectA, []string{"offices#1"}, "level", "Level 3"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org = %v", err)
	}
	ids, err := s.PlaceAccommodationLines(ctx, orgA, projectA, []string{"new:0"}, []profile.StoredLine{{Name: "Store", Set: []string{"name"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetAccommodationField(ctx, orgB, projectB, ids, "name", "Taken"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's line = %v", err)
	}
	if err := s.RemoveAccommodationLines(ctx, orgB, projectB, ids); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's delete = %v", err)
	}
	for _, bad := range [][]string{{"offices"}, {"Offices#1"}, {"offices#0"}, {"offices#1; drop"}} {
		if err := s.SetAccommodationField(ctx, orgA, projectA, bad, "level", "x"); !errors.Is(err, store.ErrInvalidInput) {
			t.Errorf("%v = %v", bad, err)
		}
	}
	if err := s.SetAccommodationField(ctx, orgA, projectA, []string{"offices#1"}, "colour", "x"); !errors.Is(err, store.ErrInvalidInput) {
		t.Errorf("unknown field = %v", err)
	}
	if lines, _ := s.AccommodationLines(ctx, orgB, projectB); len(lines) != 0 {
		t.Errorf("org B sees %d lines", len(lines))
	}
}
