package works

import (
	"encoding/json"
	"reflect"
	"sitewise/internal/knowledge"
	"sync"
	"testing"
	"time"
)

func TestProposalInputFingerprintInvalidatesSemanticInputs(t *testing.T) {
	e, err := NewEvaluator(proposalCatalogue(t))
	if err != nil {
		t.Fatal(err)
	}
	hash := func(in ProposalInput) string {
		t.Helper()
		h, err := e.InputFingerprint(in)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	base := hash(capexProposalInput())
	cases := map[string]func(*ProposalInput){
		"action":   func(in *ProposalInput) { in.Items[0].Action = "replace" },
		"layout":   func(in *ProposalInput) { in.Items[0].LayoutChange = "yes" },
		"target":   func(in *ProposalInput) { in.Items[0].Target.Text = "Changed target" },
		"identity": func(in *ProposalInput) { in.Items[0].ID = "another" },
		"review":   func(in *ProposalInput) { in.Items[0].ReviewStatus = "dismissed" },
		"group":    func(in *ProposalInput) { in.Items[0].IsGroup = true },
		"sources":  func(in *ProposalInput) { in.Items[0].Provenance.Sources = json.RawMessage(`[{"document_id":"new"}]`) },
		"determinant": func(in *ProposalInput) {
			in.Parts["C"].Values["existing_building"] = ProposalValue{Value: "false", Origin: "user"}
		},
		"signal": func(in *ProposalInput) {
			p := in.Parts["C"]
			p.Signals = map[string]ProposalSignal{"sig.test": {State: "true", Sources: json.RawMessage(`["new"]`)}}
			in.Parts["C"] = p
		},
		"existing": func(in *ProposalInput) { in.Existing = func(string, string) knowledge.Truth { return knowledge.False } },
		"present": func(in *ProposalInput) {
			p := in.Parts["C"]
			p.PresentState = func(string) knowledge.Truth { return knowledge.True }
			in.Parts["C"] = p
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			in := capexProposalInput()
			change(&in)
			if hash(in) == base {
				t.Fatal("changed input reused fingerprint")
			}
		})
	}
	in := capexProposalInput()
	in.Items[0].Version++
	now := time.Now()
	in.Items[0].Provenance.LastEditedAt = &now
	if hash(in) != base {
		t.Fatal("audit-only edit invalidated evaluation")
	}
	old := e.inputVersion
	e.inputVersion = "changed-evaluator-version"
	if hash(capexProposalInput()) == base {
		t.Fatal("evaluator version omitted")
	}
	e.inputVersion = old
}

func TestMaterializedProposalInputSharesExactTriStateSnapshot(t *testing.T) {
	cat := proposalCatalogue(t)
	e, err := NewEvaluator(cat)
	if err != nil {
		t.Fatal(err)
	}
	in := capexProposalInput()
	state := knowledge.Unknown
	calls := 0
	in.Existing = func(string, string) knowledge.Truth { calls++; return state }
	for id, p := range in.Parts {
		p.PresentState = func(string) knowledge.Truth { calls++; return state }
		in.Parts[id] = p
	}
	want, err := e.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	stable, _, err := e.MaterializeInput(in)
	if err != nil {
		t.Fatal(err)
	}
	before := calls
	got, err := e.Evaluate(stable)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("materialization changed complete proposal output")
	}
	if calls != before {
		t.Fatal("evaluation rescanned materialized system evidence")
	}
	state = knowledge.True
	if stable.Existing("C", "structure") != knowledge.Unknown || stable.Parts["C"].PresentState("structure") != knowledge.Unknown {
		t.Fatal("snapshot retained mutable callbacks")
	}
	if in.Existing("C", "structure") != knowledge.True {
		t.Fatal("materialization changed caller")
	}
}

func TestMaterializedEvaluationConcurrentOutputMatchesSequential(t *testing.T) {
	e, err := NewEvaluator(proposalCatalogue(t))
	if err != nil {
		t.Fatal(err)
	}
	in := capexProposalInput()
	want, err := e.Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	stable, _, err := e.MaterializeInput(in)
	if err != nil {
		t.Fatal(err)
	}
	var pending sync.WaitGroup
	for range 8 {
		pending.Go(func() {
			got, err := e.Evaluate(stable)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("concurrent evaluation changed complete output: %v", err)
			}
		})
	}
	pending.Wait()
}
