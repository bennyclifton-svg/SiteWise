package program

import "testing"

// F54 (owner ruling R20): on design and construct with a CDC the contractor
// is usually engaged first. The contractor's designer completes the
// documents, the CDC is obtained on them, and then construction starts. The
// generic fallback put documentation and approval before the tender.
func TestNewDesignConstructCDCOrder(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := GenInput{
		WorkTypes:  []string{"new"},
		Conditions: map[string]string{"planning": "cdc", "procurement_route": "design_construct"},
		Values:     map[string]string{"ncc_class": "7b", "rise_in_storeys": "1", "state": "NSW"},
		WorkItems: []WorkItem{
			{ID: "slab", System: "substructure.slab-on-ground", Action: "new", Title: "Ground slab"},
		},
	}
	p := mustGenerate(t, cat, in)
	const sk = "sk.new-dc-cdc"
	if p.SkeletonID != sk {
		t.Fatalf("skeleton %s, want %s", p.SkeletonID, sk)
	}
	times := timesByID(t, p)
	stage := func(id string) Times {
		key := "skeleton:" + sk + ":" + id
		if _, ok := p.Row(key); !ok {
			t.Fatalf("no %s stage", id)
		}
		return times[key]
	}
	order := []string{"procurement", "construction_documentation", "approvals", "construction"}
	for i := 1; i < len(order); i++ {
		before, after := stage(order[i-1]), stage(order[i])
		if after.ES < before.EF {
			t.Errorf("%s starts day %d, before %s finishes day %d", order[i], after.ES, order[i-1], before.EF)
		}
	}
	if _, ok := p.Row("skeleton:" + sk + ":approval_documentation"); ok {
		t.Error("a CDC job has no development application documentation stage")
	}
}
