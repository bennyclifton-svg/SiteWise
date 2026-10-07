package works

import (
	"testing"

	"sitewise/internal/knowledge"
)

func capexProposalInput() ProposalInput {
	return ProposalInput{Items: []Item{{ID: "hydrants", PartID: "C", SystemID: "fire-active.hydrants", Action: "upgrade", Inclusion: "included"}, {ID: "sprinklers-a", PartID: "A", SystemID: "fire-active.sprinklers", Action: "replace", Inclusion: "included"}, {ID: "sprinklers-b", PartID: "B", SystemID: "fire-active.sprinklers", Action: "replace", Inclusion: "included"}}, Parts: map[string]ProposalPart{
		"A": {Values: map[string]ProposalValue{"existing_building": {Value: "true", Origin: "stated"}}, WorkTypes: []string{"refurb"}},
		"B": {Values: map[string]ProposalValue{"existing_building": {Value: "true", Origin: "stated"}}, WorkTypes: []string{"refurb"}},
		"C": {Values: map[string]ProposalValue{"existing_building": {Value: "true", Origin: "stated"}}, WorkTypes: []string{"refurb"}},
	}, Existing: func(string, string) knowledge.Truth { return knowledge.Unknown }}
}

func findProposal(t *testing.T, proposals []Proposal, record, part string) Proposal {
	t.Helper()
	for _, p := range proposals {
		if p.RecordID == record && p.TargetPartID == part {
			return p
		}
	}
	t.Fatalf("missing %s for %s", record, part)
	return Proposal{}
}

func TestCapexRaisesWaterSupplyInvestigation(t *testing.T) {
	cat := proposalCatalogue(t)
	in := capexProposalInput()
	ps, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	p := findProposal(t, ps, "uc.existing-fire-water-fails-flow-test", "C")
	if p.Kind != "investigation" || p.TargetSystemID != "fire-active.fire-water" || !p.Draft || p.Severity != "life-safety" || p.Specificity != 3 {
		t.Fatal(p)
	}
	if !p.UnacceptedTriggers {
		t.Fatal("unaccepted trigger not labelled")
	}
	in.Items[0].ReviewStatus = "accepted_for_planning"
	accepted, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	acceptedP := findProposal(t, accepted, p.RecordID, "C")
	if acceptedP.UnacceptedTriggers || acceptedP.InputsFingerprint != p.InputsFingerprint {
		t.Fatal("acceptance changed semantic identity or stayed unlabelled")
	}
	if len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != "hydrants" || len(p.Reason.Signals) != 1 || p.Reason.Signals[0].State != "unknown" {
		t.Fatal(p.Reason)
	}
	for _, p := range ps {
		if p.RecordID == "uc.existing-fire-water-fails-flow-test" && p.TargetPartID != "C" {
			t.Fatal("replacement invented upgrade trigger", p)
		}
	}
}

func TestConsequenceFingerprintsRelevantOriginsOnly(t *testing.T) {
	cat := proposalCatalogue(t)
	in := capexProposalInput()
	in.Parts["C"].Values["existing_building_year"] = ProposalValue{Value: "2000", Origin: "stated"}
	before, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	p := findProposal(t, before, "cq.pre-2004-fabric-hazardous-materials-survey", "C")
	if p.Specificity != 1 || len(p.Reason.Determinants) != 1 {
		t.Fatal(p)
	}
	in.Parts["C"].Values["unrelated"] = ProposalValue{Value: "changed", Origin: "user"}
	after, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	q := findProposal(t, after, p.RecordID, "C")
	if p.InputsFingerprint != q.InputsFingerprint {
		t.Fatal("unrelated value reopened proposal")
	}
	in.Parts["C"].Values["existing_building_year"] = ProposalValue{Value: "2000", Origin: "assumed"}
	after, err = EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	q = findProposal(t, after, p.RecordID, "C")
	if p.InputsFingerprint == q.InputsFingerprint {
		t.Fatal("origin change lost")
	}
	in.Parts["C"].Values["existing_building_year"] = ProposalValue{Value: "2020", Origin: "stated"}
	after, err = EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range after {
		if q.Key == p.Key {
			t.Fatal("false predicate retained proposal")
		}
	}
}

func TestFullProposalEngineSplitStability(t *testing.T) {
	cat := proposalCatalogue(t)
	in := capexProposalInput()
	before, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	child := in.Items[0]
	child.ID = "child"
	in.Items[0].ID = "other-child"
	in.Items = append(in.Items, child)
	after, err := EvaluateProposals(cat, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("%d %d", len(before), len(after))
	}
	for i, p := range before {
		if p.Key != after[i].Key || p.InputsFingerprint != after[i].InputsFingerprint {
			t.Fatalf("split changed %s", p.Key)
		}
	}
}

func BenchmarkAllProposals(b *testing.B) {
	cat := proposalCatalogue(b)
	in := capexProposalInput()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := EvaluateProposals(cat, in); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(cat.InterfaceConsequences())+len(cat.Consequences())+len(cat.UnforeseenConditions())), "records")
}
