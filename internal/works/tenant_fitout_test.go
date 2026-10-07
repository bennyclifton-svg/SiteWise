package works

import (
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func TestTenantAlterationInvestigatesBaseBuildingCapacity(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, scenario := range []string{"alter", "upgrade", "retain", "repair", "excluded", "absent", "unknown", "plant-replaced"} {
		t.Run(scenario, func(t *testing.T) {
			item := Item{ID: "tenancy-work", PartID: "tenancy", SystemID: "mechanical.air-conditioning", Action: "alter", Inclusion: "included"}
			state := knowledge.True
			switch scenario {
			case "upgrade", "retain", "repair":
				item.Action = scenario
			case "excluded":
				item.Inclusion = "excluded"
			case "absent":
				state = knowledge.False
			case "unknown":
				state = knowledge.Unknown
			}
			items := []Item{item}
			if scenario == "plant-replaced" {
				items = append(items, Item{ID: "replacement", PartID: "tenancy", SystemID: "mechanical.central-plant", Action: "replace", Inclusion: "included"})
			}
			proposals, err := InterfaceProposals(cat, items, func(part, system string) knowledge.Truth {
				if part != "tenancy" {
					t.Fatalf("wrong part: %s", part)
				}
				return state
			})
			if err != nil {
				t.Fatal(err)
			}
			targets := map[string]bool{}
			for _, p := range proposals {
				if p.InterfaceID != "if.tenant-fitout-base-building-hvac" || p.RecordID != "ic.supplies-investigate-supply" {
					continue
				}
				targets[p.TargetSystemID] = true
				if p.Kind != "investigation" || !p.Draft || p.TargetPartID != "tenancy" || !strings.Contains(p.Label, "cooling") || strings.Contains(p.Label, "structure or ground") {
					t.Fatal(p)
				}
				if len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != item.ID || p.Reason.OtherSideExistence != state.String() {
					t.Fatal(p.Reason)
				}
			}
			want := 2
			if scenario == "retain" || scenario == "repair" || scenario == "excluded" || scenario == "absent" {
				want = 0
			}
			if scenario == "plant-replaced" {
				want = 1
			}
			if len(targets) != want {
				t.Fatalf("capacity targets: %+v, want %d", targets, want)
			}
			if want > 0 && !targets["mechanical.ventilation"] {
				t.Fatal("missing base ventilation")
			}
			if want == 2 && !targets["mechanical.central-plant"] {
				t.Fatal("missing base plant")
			}
		})
	}
}

func TestTenantInterfaceApplicabilityUsesPartEvidence(t *testing.T) {
	cat := proposalCatalogue(t)
	in := ProposalInput{Parts: map[string]ProposalPart{}, Existing: func(string, string) knowledge.Truth { return knowledge.True }}
	for _, part := range []string{"existing", "new", "unknown"} {
		in.Items = append(in.Items, Item{ID: part, PartID: part, SystemID: "mechanical.air-conditioning", Action: "alter", Inclusion: "included"})
		in.Parts[part] = ProposalPart{Values: map[string]ProposalValue{}}
	}
	in.Parts["existing"].Values["existing_building"] = ProposalValue{Value: "true", Origin: "stated"}
	in.Parts["new"].Values["existing_building"] = ProposalValue{Value: "false", Origin: "user"}
	evaluator, err := NewEvaluator(cat)
	if err != nil {
		t.Fatal(err)
	}
	check := func(ps []Proposal) map[string]Proposal {
		t.Helper()
		found := map[string]Proposal{}
		for _, p := range ps {
			if p.InterfaceID != "if.tenant-fitout-base-building-hvac" || p.RecordID != "ic.supplies-investigate-supply" || p.TargetSystemID != "mechanical.central-plant" {
				continue
			}
			found[p.TargetPartID] = p
		}
		if len(found) != 2 || found["existing"].Key == "" || found["unknown"].Key == "" {
			t.Fatalf("applicability targets: %+v", found)
		}
		for part, p := range found {
			if len(p.Reason.Determinants) != 1 || p.Reason.Determinants[0].Key != "existing_building" {
				t.Fatal(p.Reason)
			}
			want := in.Parts[part].Values["existing_building"]
			if p.Reason.Determinants[0].Value != want.Value || p.Reason.Determinants[0].Origin != want.Origin {
				t.Fatal(p.Reason)
			}
		}
		return found
	}
	ps, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	before := check(ps)
	ps, err = evaluator.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	for part, p := range check(ps) {
		if p.InputsFingerprint != before[part].InputsFingerprint {
			t.Fatal("compiled evaluator differs")
		}
	}
	in.Parts["existing"].Values["existing_building"] = ProposalValue{Value: "true", Origin: "user"}
	ps, err = evaluator.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	after := check(ps)
	if after["existing"].InputsFingerprint == before["existing"].InputsFingerprint || after["unknown"].InputsFingerprint != before["unknown"].InputsFingerprint {
		t.Fatal("fingerprint lost relevant origin or leaked across parts")
	}
	in.Parts["existing"].Values["unrelated"] = ProposalValue{Value: "changed", Origin: "user"}
	ps, err = evaluator.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	if check(ps)["existing"].InputsFingerprint != after["existing"].InputsFingerprint {
		t.Fatal("unrelated input changed fingerprint")
	}
	in.Parts["existing"].Values["existing_building"] = ProposalValue{Value: "false", Origin: "user"}
	ps, err = evaluator.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if p.InterfaceID == "if.tenant-fitout-base-building-hvac" && p.TargetPartID == "existing" {
			t.Fatal("cached applicability survived edit", p)
		}
	}
}

func TestInterfaceApplicabilityTracesSystemPresence(t *testing.T) {
	cat := proposalCatalogue(t)
	for i := range cat.Interfaces {
		if cat.Interfaces[i].ID == "if.tenant-fitout-base-building-hvac" {
			cat.Interfaces[i].AppliesWhen = map[string]any{"system_present": "mechanical.central-plant"}
		}
	}
	for _, state := range []knowledge.Truth{knowledge.True, knowledge.Unknown, knowledge.False} {
		in := ProposalInput{
			Items:    []Item{{ID: "work", PartID: "tenancy", SystemID: "mechanical.air-conditioning", Action: "alter", Inclusion: "included"}},
			Parts:    map[string]ProposalPart{"tenancy": {PresentState: func(string) knowledge.Truth { return state }}},
			Existing: func(string, string) knowledge.Truth { return knowledge.True },
		}
		ps, err := EvaluateProposals(cat, in)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range ps {
			if p.InterfaceID != "if.tenant-fitout-base-building-hvac" || p.RecordKind != "ic" {
				continue
			}
			found = true
			if len(p.Reason.SystemStates) != 1 || p.Reason.SystemStates[0] != (ProposalSystemState{"system_present", "mechanical.central-plant", state.String()}) {
				t.Fatal(p.Reason)
			}
		}
		if found != (state != knowledge.False) {
			t.Fatalf("presence %s: found=%v", state.String(), found)
		}
	}
}
