package works

import "testing"

func TestK2BoundedPlanningReviews(t *testing.T) {
	cat := proposalCatalogue(t)
	cases := []struct {
		id, system            string
		actions               []string
		existing, remediation bool
	}{
		{"cq.existing-structural-intervention-sequence-review", "structure.concrete", []string{"alter", "replace", "upgrade", "remove"}, true, false},
		{"cq.hydraulic-testing-plan-review", "hydraulic.sanitary", []string{"new", "alter", "replace", "upgrade"}, false, false},
		{"cq.existing-electrical-shutdown-plan-review", "electrical.switchboards", []string{"alter", "replace", "upgrade", "remove"}, true, false},
		{"cq.building-remediation-investigation-basis-review", "envelope.roof-coverings", []string{"alter", "replace", "upgrade", "repair"}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			for _, action := range tc.actions {
				t.Run(action, func(t *testing.T) {
					for _, mode := range []string{"match", "excluded", "retain", "unrelated", "other_context"} {
						item := Item{ID: "work", PartID: "part", SystemID: tc.system, Action: action, Inclusion: "included"}
						part := ProposalPart{Values: map[string]ProposalValue{"existing_building": {Value: "true", Origin: "document"}}, WorkTypes: []string{"remediation"}}
						want := true
						switch mode {
						case "excluded":
							item.Inclusion = "excluded"
							want = false
						case "retain":
							item.Action = "retain"
							want = false
						case "unrelated":
							item.SystemID = "comms-security.structured-cabling"
							want = false
						case "other_context":
							if tc.existing {
								part.Values["existing_building"] = ProposalValue{Value: "false", Origin: "document"}
								want = false
							}
							if tc.remediation {
								part.WorkTypes = []string{"new"}
								want = false
							}
						}
						proposals, err := EvaluateProposals(cat, ProposalInput{Items: []Item{item}, Parts: map[string]ProposalPart{"part": part}})
						if err != nil {
							t.Fatal(err)
						}
						count := 0
						for _, p := range proposals {
							if p.RecordID != tc.id {
								continue
							}
							count++
							if !p.Draft || p.Kind != "hold_point" || p.TargetSystemID != "" || p.Action != "" || p.TargetPartID != "part" || len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != "work" {
								t.Fatalf("invented target or missing provenance: %+v", p)
							}
						}
						if (count == 1) != want || count > 1 {
							t.Fatalf("%s: count=%d want=%v", mode, count, want)
						}
					}
				})
			}
		})
	}
}
