package profile

import "testing"

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
		c, w := PackageBaselineFacts([]Row{class, work, bad})
		if c != "industrial" || len(w) != 1 || w[0] != "new" {
			t.Fatalf("ineligible value applied: %s %v", c, w)
		}
	}
}
