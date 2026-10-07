package jobs_test

import (
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"strings"
	"testing"
)

func TestSignalsShareEvidenceFanout(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	call, ok := jobs.EvidenceCall(cat, jobs.Passage{Labels: []string{"fire-active", "fire-active.fire-water"}, Text: "The authority provided flow and pressure advice."})
	if !ok {
		t.Fatal("no evidence call")
	}
	key := "sig:sig.fire-water-authority-capacity-advice-stated"
	if q, ok := call.Questions[key]; !ok || q.Type != jev.TypeNoul {
		t.Fatalf("missing labelled signal: %+v", q)
	}
	if call.Questions["sys.fire-active.hydrants.action"].Type != jev.TypeChoice {
		t.Fatal("signal displaced action")
	}
	other, _ := jobs.EvidenceCall(cat, jobs.Passage{Labels: []string{"interiors"}, Text: "Paint walls"})
	if _, ok := other.Questions[key]; ok {
		t.Fatal("unrelated signal routed")
	}
	parentOnly, _ := jobs.EvidenceCall(cat, jobs.Passage{Labels: []string{"fire-active"}, Text: "Fire services"})
	if _, ok := parentOnly.Questions[key]; ok {
		t.Fatal("unlabelled child signal was speculated")
	}
	if _, ok := call.Questions["sig:sig.performance-solution-acceptance-stated"]; !ok {
		t.Fatal("ancestor signal not matched")
	}
	count := 0
	for id := range call.Questions {
		if strings.HasPrefix(id, "sig:") {
			count++
		}
	}
	t.Logf("fire-active evidence: %d total questions, %d signals", len(call.Questions), count)
}
