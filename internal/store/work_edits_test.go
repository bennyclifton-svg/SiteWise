package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestWorkCorrectionSurvivesRebuildAndRacingEdits(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	facts := []store.StoredFact{{PassageID: passageA, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Confidence: conf(.9), DecidedBy: "jev", Excerpt: "Gas included"}}
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 1 {
		t.Fatalf("items %+v %v", items, err)
	}
	before := items[0]
	action, title, unit := "repair", "Repair gas connection", "m"
	patch := works.Patch{Version: before.Version, Action: &action, Title: &title, Quantity: json.RawMessage(`123456789012345678.123456789`), Unit: &unit, Target: &works.Target{Text: "Make safe"}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := s.PatchWorkItem(ctx, orgA, projectA, before.ID, userA, patch); results <- err })
	}
	wg.Wait()
	close(results)
	success, stale := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, store.ErrVersionConflict) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success=%d stale=%d", success, stale)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 1 {
		t.Fatalf("items %+v %v", items, err)
	}
	got := items[0]
	// Column decoding must retain the complete previous JSON-backed contract,
	// including exact decimals, source payloads and correction audit fields.
	var storedJSON []byte
	if err := rawPool(t).QueryRow(ctx, `SELECT to_jsonb(w)||jsonb_build_object('quantity',quantity::text) FROM work_items w WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, got.ID).Scan(&storedJSON); err != nil {
		t.Fatal(err)
	}
	var stored works.Item
	if err := json.Unmarshal(storedJSON, &stored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored, got) {
		t.Fatalf("typed read differs from saved record: stored=%+v read=%+v", stored, got)
	}
	if got.ID != before.ID || got.Action != action || got.Title != title || got.Quantity == nil || *got.Quantity != "123456789012345678.123456789" || got.Target.Text != "Make safe" || !got.UserTouched || got.ReviewStatus != "accepted_for_planning" || got.Origin != "user" || got.Version != before.Version+1 {
		t.Fatalf("correction lost: %+v", got)
	}
	if string(got.Provenance.Sources) != string(before.Provenance.Sources) || got.Provenance.LastEditedBy != userA || got.Provenance.LastEditedAt == nil {
		t.Fatalf("audit lost: %+v", got.Provenance)
	}
	cleared, err := s.PatchWorkItem(ctx, orgA, projectA, got.ID, userA, works.Patch{Version: got.Version, Quantity: json.RawMessage(`null`)})
	if err != nil || cleared.Quantity != nil || cleared.Unit != nil {
		t.Fatalf("clear %+v %v", cleared, err)
	}
	if _, err := s.PatchWorkItem(ctx, orgB, projectA, got.ID, userA, patch); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org: %v", err)
	}
}
