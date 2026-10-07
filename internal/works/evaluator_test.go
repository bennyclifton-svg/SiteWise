package works

import (
	"reflect"
	"testing"

	"sitewise/internal/knowledge"
)

func TestCompiledEvaluatorPreservesCompleteOutput(t *testing.T) {
	cat := proposalCatalogue(t)
	e, err := NewEvaluator(cat)
	if err != nil {
		t.Fatal(err)
	}
	in := capexProposalInput()
	for i := 0; i < 3; i++ {
		if i == 1 {
			in.Parts["C"].Values["existing_building"] = ProposalValue{Value: "false", Origin: "user"}
		}
		if i == 2 {
			in.Items[0].Action = "replace"
		}
		want, err := EvaluateProposals(cat, in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := e.Evaluate(in)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cached evaluation %d differs", i)
		}
	}
}

func TestUnconditionalInterfaceKeepsRecordFingerprint(t *testing.T) {
	// Adding condition support must not reopen decisions for unchanged edges.
	// This is the previous persisted hash shape, before AppliesWhen was loaded.
	legacy := struct {
		ID           string               `yaml:"id"`
		Type         string               `yaml:"type"`
		From         []string             `yaml:"from"`
		To           []string             `yaml:"to"`
		Summary      string               `yaml:"summary"`
		Status       string               `yaml:"status"`
		ResolvedWhen []knowledge.Question `yaml:"resolved_when"`
	}{ID: "if.example", Type: "supplies", From: []string{"mechanical.central-plant"}, To: []string{"mechanical.air-conditioning"}, Status: "draft"}
	edge := knowledge.Interface{ID: legacy.ID, Type: legacy.Type, From: legacy.From, To: legacy.To, Status: legacy.Status}
	want, err := proposalRecordHash(legacy)
	if err != nil {
		t.Fatal(err)
	}
	got, err := proposalRecordHash(edge)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("unconditional interface fingerprint changed")
	}
	edge.AppliesWhen = map[string]any{"det": "existing_building", "is": true}
	got, err = proposalRecordHash(edge)
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("condition missing from fingerprint")
	}
}

func TestCompiledEvaluatorRefreshesPartEvidence(t *testing.T) {
	cat := proposalCatalogue(t)
	e, err := NewEvaluator(cat)
	if err != nil {
		t.Fatal(err)
	}
	in := capexProposalInput()
	state := knowledge.Unknown
	in.Existing = func(part, _ string) knowledge.Truth {
		if part == "A" {
			return knowledge.False
		}
		return state
	}
	for id, part := range in.Parts {
		part.PresentState = func(string) knowledge.Truth {
			if id == "A" {
				return knowledge.True
			}
			return state
		}
		in.Parts[id] = part
	}
	for _, next := range []knowledge.Truth{knowledge.Unknown, knowledge.True, knowledge.False, knowledge.Unknown} {
		state = next
		want, err := EvaluateProposals(cat, in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := e.Evaluate(in)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("part evidence differs for %s", state.String())
		}
		if in.Parts["C"].PresentState("structure") != state {
			t.Fatal("evaluation changed the caller's evidence callback")
		}
	}
}

func BenchmarkCompiledProposals(b *testing.B) {
	cat := proposalCatalogue(b)
	e, err := NewEvaluator(cat)
	if err != nil {
		b.Fatal(err)
	}
	in := capexProposalInput()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := e.Evaluate(in); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCompiledPrimitiveMemoPreservesLayoutAndUnknownTraces(t *testing.T) {
	cat := proposalCatalogue(t)
	e, err := NewEvaluator(cat)
	if err != nil {
		t.Fatal(err)
	}
	for _, layout := range []string{"unknown", "yes", "no"} {
		for _, action := range []string{"alter", ""} {
			in := capexProposalInput()
			in.Items[0].SystemID = "interiors.walls-linings"
			in.Items[0].Action = action
			in.Items[0].LayoutChange = layout
			expected, err := EvaluateProposals(cat, in)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := e.Evaluate(in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, actual) {
				t.Fatalf("memo changed full traces action=%q layout=%s", action, layout)
			}
		}
	}
}

func BenchmarkProposalStartup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cat := proposalCatalogue(b)
		if _, err := NewEvaluator(cat); err != nil {
			b.Fatal(err)
		}
	}
}
