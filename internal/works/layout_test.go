package works

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestLayoutChangePredicateUsesSameWorkAndPreservesUnknown(t *testing.T) {
	cat := proposalCatalogue(t)
	predicate := map[string]any{"works": map[string]any{"system": []any{"interiors.walls-linings"}, "action": []any{"alter"}, "layout_change": "yes"}}
	for _, tc := range []struct {
		name    string
		items   []knowledge.WorkItem
		want    knowledge.Truth
		indices int
	}{
		{"yes", []knowledge.WorkItem{{System: "interiors.walls-linings", Action: "alter", LayoutChange: "yes"}}, knowledge.True, 1},
		{"no", []knowledge.WorkItem{{System: "interiors.walls-linings", Action: "alter", LayoutChange: "no"}}, knowledge.False, 0},
		{"unknown", []knowledge.WorkItem{{System: "interiors.walls-linings", Action: "alter"}}, knowledge.Unknown, 1},
		{"sibling action cannot supply layout", []knowledge.WorkItem{{System: "interiors.walls-linings", Action: "alter", LayoutChange: "no"}, {System: "interiors.walls-linings", Action: "remove", LayoutChange: "yes"}}, knowledge.False, 0},
		{"other system cannot supply layout", []knowledge.WorkItem{{System: "interiors.walls-linings", Action: "alter", LayoutChange: "no"}, {System: "electrical", Action: "alter", LayoutChange: "yes"}}, knowledge.False, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trace := cat.TraceRelevantWorks(predicate, knowledge.WorksEnv{Items: tc.items})
			if trace.Truth != tc.want || len(trace.ItemIndexes) != tc.indices {
				t.Fatal(trace)
			}
		})
	}
}
func TestLayoutChangeReviewTraceAndFingerprint(t *testing.T) {
	cat := proposalCatalogue(t)
	in := ProposalInput{Items: []Item{{ID: "wall", PartID: "A", SystemID: "interiors.walls-linings", Action: "alter", Inclusion: "included", LayoutChange: "yes"}}, Parts: map[string]ProposalPart{"A": {}}, Existing: func(string, string) knowledge.Truth { return knowledge.True }}
	read := func() *Proposal {
		ps, e := EvaluateProposals(cat, in)
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range ps {
			if p.RecordID == "cq.room-layout-air-distribution-review" {
				return &p
			}
		}
		return nil
	}
	yes := read()
	if yes == nil || !yes.Draft || yes.Kind != "hold_point" || len(yes.Reason.Triggers) != 1 || yes.Reason.Triggers[0].WorkItemID != "wall" {
		t.Fatal(yes)
	}
	in.Items[0].LayoutChange = "unknown"
	unknown := read()
	if unknown == nil || unknown.InputsFingerprint == yes.InputsFingerprint {
		t.Fatal("layout change not fingerprinted", unknown)
	}
	in.Items[0].LayoutChange = "no"
	if read() != nil {
		t.Fatal("lining-only work proposed layout review")
	}
	in.Items[0].LayoutChange = "yes"
	in.Items[0].Inclusion = "excluded"
	if read() != nil {
		t.Fatal("excluded layout work proposed")
	}
	in.Items[0].Inclusion = "included"
	in.Existing = func(string, string) knowledge.Truth { return knowledge.False }
	if read() != nil {
		t.Fatal("absent air distribution proposed")
	}
}
