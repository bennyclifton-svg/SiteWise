package profile

import (
	"slices"
	"testing"

	"sitewise/internal/knowledge"
)

func TestPackageComplexityFactsExcludeUnappliedAndConflictingValues(t *testing.T) {
	valid := Row{Key: "det.heritage_status", Value: "local_item", Band: "user", Meaning: "stated", ValueState: "set"}
	for _, bad := range []Row{
		{Key: "det.bal", Value: "BAL-40", Band: "suggested"},
		{Key: "det.bal", Value: "BAL-40", Band: "user", Origin: "assumption"},
		{Key: "det.bal", Value: "BAL-40", Band: "user", Meaning: "allowance"},
		{Key: "det.bal", Value: "BAL-40", Band: "user", ValueState: StateUnknown},
		{Key: "det.bal", Value: "BAL-40", Band: "user", ReviewStatus: ReviewSupersede},
	} {
		got := PackageComplexityFacts([]Row{valid, bad})
		if len(got) != 1 || got["heritage_status"] != "local_item" {
			t.Fatalf("ineligible fact: %+v", got)
		}
	}
	conflict := valid
	conflict.Key, conflict.Value = "hdr.cond.heritage_status", "state_register"
	if got := PackageComplexityFacts([]Row{valid, conflict, valid}); len(got) != 0 {
		t.Fatalf("conflict picked a value: %+v", got)
	}
}

func TestPackageBaselineEligibility(t *testing.T) {
	class := Row{Key: "hdr.building_class", Value: "industrial", Band: "user", Origin: "user", Meaning: "stated", ValueState: "set"}
	work := Row{Key: "hdr.work_type", Value: "new", Band: "amber", Origin: "document", Meaning: "stated", ValueState: "set"}
	for _, bad := range []Row{
		{Key: "hdr.work_type", Value: "extend", Band: "suggested"},
		{Key: "hdr.work_type", Value: "extend", Band: "user", Origin: "assumption"},
		{Key: "hdr.work_type", Value: "extend", Band: "user", Meaning: "allowance"},
		{Key: "hdr.work_type", Value: "extend", Band: "user", ValueState: StateUnknown},
		{Key: "hdr.work_type", Value: "extend", Band: "user", ReviewStatus: ReviewSupersede},
	} {
		c, w, _ := PackageBaselineFacts(nil, []Row{class, work, bad})
		if !slices.Equal(c, []string{"industrial"}) || len(w) != 1 || w[0] != "new" {
			t.Fatalf("ineligible value applied: %s %v", c, w)
		}
	}
}

// Several classes are normal: the header category, a second reading of it,
// and every NCC class all add their category; scope rows marked in count
// whatever their band, as the profile shows them.
func TestPackageBaselineFactsUnionClassesAndReadScope(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	stated := func(key, value, band string) Row {
		return Row{Key: key, Value: value, Band: band, Meaning: "stated", ValueState: "set"}
	}
	rows := []Row{
		stated("hdr.building_class", "commercial", "green"),
		stated("hdr.building_class", "industrial", "amber"),
		stated("det.ncc_class", "2,6", "user"),
		stated("hdr.work_type", "new", "user"),
		{Key: "scope.vertical-transport.passenger-lifts", Value: "in", Band: "suggested"},
		{Key: "scope.mechanical.ventilation", Value: "out", Band: "user"},
		stated("det.ncc_class", "9a", "suggested"),
	}
	classes, types, scope := PackageBaselineFacts(cat, rows)
	if !slices.Contains(classes, "commercial") || !slices.Contains(classes, "industrial") || !slices.Contains(classes, "residential") {
		t.Fatalf("classes %v", classes)
	}
	if slices.Contains(classes, "institution") {
		t.Fatalf("a suggested NCC class applied: %v", classes)
	}
	if !slices.IsSorted(classes) || len(slices.Compact(slices.Clone(classes))) != len(classes) {
		t.Fatalf("classes not sorted and unique: %v", classes)
	}
	if !slices.Equal(types, []string{"new"}) || !slices.Equal(scope, []string{"vertical-transport.passenger-lifts"}) {
		t.Fatalf("types %v scope %v", types, scope)
	}
}
