package works

import "testing"

func TestK3JurisdictionReviewsPreserveUnknownAndPartScope(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, id := range []string{"cq.nsw-alteration-approval-basis-review", "cq.nsw-class2-remedial-design-applicability-review", "cq.nsw-state-heritage-pathway-review"} {
		for _, state := range []string{"NSW", "VIC", ""} {
			t.Run(id+"/"+state, func(t *testing.T) {
				part := ProposalPart{Values: map[string]ProposalValue{"state": {Value: state, Origin: "user"}, "existing_building": {Value: "true", Origin: "user"}, "ncc_class": {Value: "2", Origin: "user"}, "heritage_status": {Value: "state_register", Origin: "user"}}, WorkTypes: []string{"remediation"}}
				other := ProposalPart{Values: map[string]ProposalValue{"state": {Value: "VIC", Origin: "user"}}, WorkTypes: []string{"remediation"}}
				items := []Item{{ID: "A-work", PartID: "A", SystemID: "envelope.roof-coverings", Action: "alter", Inclusion: "included", ReviewStatus: "accepted_for_planning"}, {ID: "B-work", PartID: "B", SystemID: "envelope.roof-coverings", Action: "alter", Inclusion: "included", ReviewStatus: "accepted_for_planning"}}
				ps, err := EvaluateProposals(cat, ProposalInput{Items: items, Parts: map[string]ProposalPart{"A": part, "B": other}})
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, p := range ps {
					if p.RecordID != id {
						continue
					}
					count++
					if p.TargetPartID != "A" || !p.Draft || p.Kind != "approval" || p.TargetSystemID != "" || p.Action != "" {
						t.Fatalf("incorrect scope or promoted legal action: %+v", p)
					}
					found := false
					for _, d := range p.Reason.Determinants {
						if d.Key == "state" {
							found = true
							if d.Value != state || d.Origin != "user" {
								t.Fatal(d)
							}
						}
					}
					if !found {
						t.Fatal("state provenance missing")
					}
				}
				want := 1
				if state == "VIC" {
					want = 0
				}
				if count != want {
					t.Fatalf("count=%d want=%d", count, want)
				}
			})
		}
	}
}

func TestK3RemainingTopicsAreBoundedReviews(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, tc := range []struct {
		id, system      string
		existing, class bool
	}{
		{"cq.nsw-essential-measures-continuity-review", "fire-active.sprinklers", true, false},
		{"cq.nsw-disturbance-information-review", "envelope.roof-coverings", true, false},
		{"cq.nsw-residential-contracting-applicability-review", "envelope.roof-coverings", false, true},
	} {
		for _, mode := range []string{"match", "wrong-state", "unknown-state", "wrong-class", "unknown-class", "not-existing", "unknown-existing", "retain", "unknown-action", "excluded", "unrelated-system"} {
			t.Run(tc.id+"/"+mode, func(t *testing.T) {
				part := ProposalPart{Values: map[string]ProposalValue{"state": {Value: "NSW", Origin: "user"}, "existing_building": {Value: "true", Origin: "user"}, "ncc_class": {Value: "2", Origin: "user"}}}
				item := Item{ID: "A-work", PartID: "A", SystemID: tc.system, Action: "alter", Inclusion: "included", ReviewStatus: "accepted_for_planning"}
				want := 1
				switch mode {
				case "wrong-state":
					part.Values["state"] = ProposalValue{Value: "VIC", Origin: "user"}
					want = 0
				case "unknown-state":
					part.Values["state"] = ProposalValue{Origin: "user"}
				case "wrong-class":
					part.Values["ncc_class"] = ProposalValue{Value: "8", Origin: "user"}
					if tc.class {
						want = 0
					}
				case "unknown-class":
					part.Values["ncc_class"] = ProposalValue{Origin: "user"}
				case "not-existing":
					part.Values["existing_building"] = ProposalValue{Value: "false", Origin: "user"}
					if tc.existing {
						want = 0
					}
				case "unknown-existing":
					part.Values["existing_building"] = ProposalValue{Origin: "user"}
				case "retain":
					item.Action = "retain"
					want = 0
				case "unknown-action":
					item.Action = ""
				case "excluded":
					item.Inclusion = "excluded"
					want = 0
				case "unrelated-system":
					item.SystemID = "comms-security.structured-cabling"
					want = 0
				}
				ps, err := EvaluateProposals(cat, ProposalInput{Items: []Item{item}, Parts: map[string]ProposalPart{"A": part}})
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, p := range ps {
					if p.RecordID != tc.id {
						continue
					}
					count++
					if !p.Draft || p.Kind != "hold_point" || p.TargetSystemID != "" || p.Action != "" || p.TargetPartID != "A" {
						t.Fatal(p)
					}
				}
				if count != want {
					t.Fatalf("got%d want%d", count, want)
				}
			})
		}
	}
}
