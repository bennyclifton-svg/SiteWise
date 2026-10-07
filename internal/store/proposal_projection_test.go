package store

import (
	"sitewise/internal/works"
	"testing"
)

func TestProjectionHashIncludesDisplayDecisionAndTriggerIdentity(t *testing.T) {
	base := works.Proposal{Key: "p", Rank: 1, State: "open", Reason: works.ProposalReason{Triggers: []works.ProposalTrigger{{WorkItemID: "one", System: "structure", Part: "whole", Action: "repair"}}}}
	hash := func(p works.Proposal) string {
		t.Helper()
		h, err := proposalProjectionHash("site", p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	original := hash(base)
	for name, change := range map[string]func(*works.Proposal){
		"label": func(p *works.Proposal) { p.Label = "changed" }, "knowledge": func(p *works.Proposal) { p.KnowledgeVersion = "v2" }, "fingerprint": func(p *works.Proposal) { p.InputsFingerprint = "new" }, "state": func(p *works.Proposal) { p.State = "dismissed" }, "changed": func(p *works.Proposal) { p.InputsChanged = true }, "review": func(p *works.Proposal) { p.UnacceptedTriggers = true }, "trigger": func(p *works.Proposal) {
			p.Reason.Triggers = []works.ProposalTrigger{{WorkItemID: "two", System: "structure", Part: "whole", Action: "repair"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := base
			change(&p)
			if hash(p) == original {
				t.Fatal("projection change omitted")
			}
		})
	}
	p := base
	p.Rank++
	if hash(p) != original {
		t.Fatal("derived rank must not change the persisted payload hash")
	}
}
