package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestWorkLayoutChangePersistenceAndSplit(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	item, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "interiors.walls-linings", Action: "alter", Title: "Partitions"})
	if e != nil {
		t.Fatal(e)
	}
	if item.LayoutChange != "unknown" {
		t.Fatal("new work asserted layout", item)
	}
	yes, no, invalid := "yes", "no", "true"
	if _, e = s.PatchWorkItem(ctx, orgB, projectA, item.ID, userB, works.Patch{Version: item.Version, LayoutChange: &yes}); !errors.Is(e, store.ErrNotFound) {
		t.Fatal(e)
	}
	if _, e = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: item.Version, LayoutChange: &invalid}); !errors.Is(e, store.ErrInvalidWork) {
		t.Fatal(e)
	}
	original := item.Version
	item, e = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: item.Version, LayoutChange: &yes})
	if e != nil || item.LayoutChange != "yes" || item.Provenance.LastEditedBy != userA {
		t.Fatal(item, e)
	}
	if _, e = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: original, LayoutChange: &no}); !errors.Is(e, store.ErrVersionConflict) {
		t.Fatal(e)
	}
	var patch works.Patch
	if e = json.Unmarshal([]byte(`{"version":2,"title":"Partitions retained correction","layout_change":null}`), &patch); e != nil {
		t.Fatal(e)
	}
	patch.Version = item.Version
	item, e = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, patch)
	if e != nil || item.LayoutChange != "yes" {
		t.Fatal("null must omit", item, e)
	}
	repair := "repair"
	if _, e = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: item.Version, Action: &repair}); !errors.Is(e, store.ErrInvalidWork) {
		t.Fatal("incompatible retained context", e)
	}
	children, e := s.SplitWorkItem(ctx, orgA, projectA, item.ID, userA, works.SplitRequest{Version: item.Version, Children: []works.SplitChild{{Title: "Layout to confirm"}, {Title: "Lining only", LayoutChange: "no"}}})
	if e != nil {
		t.Fatal(e)
	}
	if children[0].LayoutChange != "unknown" || children[1].LayoutChange != "no" {
		t.Fatal("split inherited layout assertion", children)
	}
	saved, e := s.ReadWorks(ctx, orgA, projectA)
	if e != nil {
		t.Fatal(e)
	}
	for _, w := range saved {
		if w.ID == children[1].ID && w.LayoutChange != "no" {
			t.Fatal("layout not saved", w)
		}
	}
}
