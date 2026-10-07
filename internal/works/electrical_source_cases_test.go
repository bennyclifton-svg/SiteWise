package works

import "testing"

func TestElectricalInvestigationsFollowSourceWorkSystems(t *testing.T) {
	cat := proposalCatalogue(t)
	cases := []struct{ system, record, target string }{
		{"electrical.ev-charging", "uc.existing-supply-or-main-capacity-below-added-load", "electrical.switchboards"},
		{"site.vehicle-access", "uc.in-ground-supply-asset-clashes-with-new-works", "site.services-connections"},
	}
	for _, c := range cases {
		for _, scenario := range []string{"new", "alter", "retain", "excluded", "other-system", "existing-false", "existing-unknown"} {
			t.Run(c.system+"/"+scenario, func(t *testing.T) {
				item := Item{ID: "work", PartID: "part", SystemID: c.system, Action: "new", Inclusion: "included"}
				switch scenario {
				case "alter", "retain":
					item.Action = scenario
				case "excluded":
					item.Inclusion = "excluded"
				case "other-system":
					item.SystemID = "interiors.floor-finishes"
				}
				existing := "true"
				if scenario == "existing-false" {
					existing = "false"
				}
				if scenario == "existing-unknown" {
					existing = ""
				}
				ps, err := EvaluateProposals(cat, ProposalInput{Items: []Item{item}, Parts: map[string]ProposalPart{"part": {Values: map[string]ProposalValue{"existing_building": {Value: existing, Origin: "user"}}}}})
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, p := range ps {
					if p.RecordID != c.record {
						continue
					}
					found = true
					if !p.Draft || p.Kind != "investigation" || p.TargetSystemID != c.target || p.TargetPartID != item.PartID || len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != item.ID {
						t.Fatal(p)
					}
				}
				want := scenario == "new" || scenario == "alter" || scenario == "existing-unknown" || (scenario == "existing-false" && c.system == "site.vehicle-access")
				if found != want {
					t.Fatalf("found=%v", found)
				}
			})
		}
	}
}
