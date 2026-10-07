package works

import "testing"

func TestMechanicalCommissioningReviewFollowsAffectedWork(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, mode := range []string{"new", "alter", "replace", "upgrade", "repair", "remove", "retain", "excluded", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			item := Item{ID: "scope", PartID: "plant-room", SystemID: "mechanical.air-conditioning", Action: mode, Inclusion: "included"}
			if mode == "excluded" {
				item.Action, item.Inclusion = "alter", "excluded"
			}
			if mode == "unrelated" {
				item.Action, item.SystemID = "new", "interiors.floor-finishes"
			}
			ps, err := EvaluateProposals(cat, ProposalInput{Items: []Item{item}, Parts: map[string]ProposalPart{"plant-room": {}}})
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, p := range ps {
				if p.RecordID != "cq.mechanical-commissioning-plan-review" {
					continue
				}
				count++
				if p.Kind != "hold_point" || !p.Draft || p.TargetPartID != item.PartID || p.TargetSystemID != "" || p.Action != "" {
					t.Fatal(p)
				}
				if len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != item.ID {
					t.Fatal(p.Reason)
				}
			}
			want := 0
			if mode == "new" || mode == "alter" || mode == "replace" || mode == "upgrade" {
				want = 1
			}
			if count != want {
				t.Fatalf("review proposals %d, want %d", count, want)
			}
		})
	}
}
