package jobs_test

import (
	"strings"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
)

func TestFullDocumentDoesNotStopAt500Passages(t *testing.T) {
	text := strings.Repeat("A specified requirement.\n\n", 700) + "Final handover requirement."
	got := jobs.SplitPassages(text)
	if len(got) != 701 || got[700] != "Final handover requirement." {
		t.Fatalf("lost end of document: %d passages", len(got))
	}
}

func TestMultipleLeavesInSameFamily(t *testing.T) {
	cat := loadKnowledge(t)
	call, ok := jobs.EvidenceCall(cat, jobs.Passage{Text: "The contractor shall provide hot and cold water services.", Labels: []string{"hydraulic"}})
	if !ok {
		t.Fatal("missing evidence fan-out")
	}
	answers := map[string]jev.Answer{}
	for _, leaf := range []string{"hydraulic.hot-water", "hydraulic.cold-water"} {
		id := "system." + leaf
		if _, ok := call.Questions[id]; !ok {
			t.Fatalf("missing independent question %s", id)
		}
		answers[id] = jev.Answer{Type: jev.TypeNoul, Noul: 0.98}
	}
	got := jobs.AcceptLabels(cat, jev.Result{Answers: answers}, 0.5)
	for _, want := range []string{"hydraulic.hot-water", "hydraulic.cold-water"} {
		if !contains(got, want) {
			t.Fatalf("lost %s: %v", want, got)
		}
	}
}
