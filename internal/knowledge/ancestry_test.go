package knowledge

import (
	"reflect"
	"testing"
)

func TestCompiledChildrenMatchLiveOrderedHierarchy(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	reference := *cat
	reference.children = nil
	parents := []string{"", "missing"}
	for id := range cat.systems {
		parents = append(parents, id)
	}
	for _, parent := range parents {
		got, want := cat.Children(parent), reference.Children(parent)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("children %s differ", parent)
		}
		if len(got) > 0 {
			got[0].ID = "mutated"
			if !reflect.DeepEqual(cat.Children(parent), want) {
				t.Fatalf("children %s returned mutable index", parent)
			}
		}
	}
}

func TestCompiledAncestryMatchesHierarchy(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	for id := range cat.systems {
		for target := range cat.systems {
			want := false
			for at := id; at != ""; at = cat.systems[at].Parent {
				if at == target {
					want = true
					break
				}
			}
			if got := cat.covers(id, target); got != want {
				t.Fatalf("%s under %s: %v want %v", id, target, got, want)
			}
		}
	}
	for _, pair := range [][2]string{{"", ""}, {"missing", "structure"}, {"structure", "missing"}} {
		if cat.covers(pair[0], pair[1]) {
			t.Fatalf("invalid ancestry %v", pair)
		}
	}
}
