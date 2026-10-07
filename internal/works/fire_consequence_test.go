package works

import "testing"

func TestExistingFireApprovalReviewScope(t *testing.T) {
	cat := proposalCatalogue(t)
	const record = "cq.existing-fire-measures-approval-basis-review"
	for _, tc := range []struct {
		name, system, action, existing, inclusion string
		want                                      bool
	}{
		{"active upgrade", "fire-active.hydrants", "upgrade", "true", "included", true},
		{"passive alteration", "fire-passive.rated-elements", "alter", "true", "included", true},
		{"new building", "fire-active.hydrants", "upgrade", "false", "included", false},
		{"retain", "fire-active.hydrants", "retain", "true", "included", false},
		{"repair", "fire-active.hydrants", "repair", "true", "included", false},
		{"excluded", "fire-active.hydrants", "upgrade", "true", "excluded", false},
		{"unrelated work", "mechanical.air-conditioning", "upgrade", "true", "included", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := ProposalInput{Items: []Item{{ID: "work", PartID: "part", SystemID: tc.system, Action: tc.action, Inclusion: tc.inclusion}}, Parts: map[string]ProposalPart{
				"part": {Values: map[string]ProposalValue{"existing_building": {Value: tc.existing, Origin: "document"}}},
			}}
			proposals, err := EvaluateProposals(cat, in)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, p := range proposals {
				if p.RecordID != record {
					continue
				}
				count++
				if !p.Draft || p.Kind != "approval" || p.TargetSystemID != "" || p.TargetPartID != "part" || len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != "work" {
					t.Fatalf("review lost scope or acquired an inferred target: %+v", p)
				}
			}
			if (count == 1) != tc.want || count > 1 {
				t.Fatalf("proposal count %d, want present %v", count, tc.want)
			}
		})
	}
}
