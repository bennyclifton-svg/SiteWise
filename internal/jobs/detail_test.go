package jobs_test

import (
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"strings"
	"testing"
)

// The Specification.docx failure: a clause tagged with several families
// asked every leaf's detail questions at once and exceeded the size budget.
// The first pass asks only which leaves are present.
func TestEvidenceCallDefersLeafDetail(t *testing.T) {
	call, ok := jobs.EvidenceCall(loadKnowledge(t), jobs.Passage{Text: "Natural gas work.", Labels: []string{"hydraulic"}})
	if !ok {
		t.Fatal("no evidence call")
	}
	if call.Questions["system.hydraulic.gas"].Type != jev.TypeNoul {
		t.Fatal("leaf presence missing from first pass")
	}
	for id := range call.Questions {
		if strings.HasPrefix(id, "sys.") {
			t.Fatalf("detail question %s asked before its leaf was found", id)
		}
	}
}

func TestMultiFamilyClauseFitsFirstPass(t *testing.T) {
	call, _ := jobs.EvidenceCall(loadKnowledge(t), jobs.Passage{Text: "x", Labels: []string{"site", "structure", "comms-security", "mechanical"}})
	if _, err := jev.CheckRequestSize(call); err != nil {
		t.Fatal(err)
	}
}

func TestDetailCallAsksOnlyFoundLeaves(t *testing.T) {
	cat := loadKnowledge(t)
	p := jobs.Passage{Text: "Natural gas work.", Labels: []string{"hydraulic"}}
	call, ok := jobs.DetailCall(cat, p, []string{"hydraulic.gas"})
	if !ok || call.Questions["sys.hydraulic.gas.action"].Type != jev.TypeChoice || call.Questions["sys.hydraulic.gas.presence"].Type != jev.TypeChoice {
		t.Fatalf("gas detail missing: %v", ok)
	}
	if _, asked := call.Questions["sys.hydraulic.hot-water.action"]; asked {
		t.Fatal("detail asked for a leaf that was not found")
	}
	if _, ok := jobs.DetailCall(cat, p, nil); ok {
		t.Fatal("detail call with no leaves")
	}
}

// "NON POTABLE WATER SERVICES / Not used." must reach the detail pass, where
// its exclusion is read; the presence question must count exclusions.
func TestLeafPresenceCountsExclusions(t *testing.T) {
	call, _ := jobs.EvidenceCall(loadKnowledge(t), jobs.Passage{Text: "Not used.", Labels: []string{"hydraulic"}})
	q := call.Questions["system.hydraulic.non-potable-water"]
	if s, _ := q.Instructions.(string); !strings.Contains(s, "excluded") {
		t.Fatalf("leaf presence ignores exclusions: %s", q.Instructions)
	}
}
