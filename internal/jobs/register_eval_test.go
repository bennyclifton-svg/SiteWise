package jobs

import "testing"

func TestProposeRegisterCutoff(t *testing.T) {
	run := []LabeledAnswer{{0.97, true}, {0.93, true}, {0.9, false}, {0.85, true}, {0.6, true}, {0.4, false}, {0.2, false}}
	cut, p, recall, ok := ProposeRegisterCutoff(run, 0.75)
	if !ok || cut != 0.6 || p != 0.8 || recall != 1 {
		t.Fatalf("cut %v precision %v recall %v ok %v", cut, p, recall, ok)
	}
	if cut, _, _, ok := ProposeRegisterCutoff(run, 0.95); !ok || cut != 0.93 {
		t.Fatalf("strict cut %v %v", cut, ok)
	}
	if _, _, _, ok := ProposeRegisterCutoff([]LabeledAnswer{{0.9, false}}, 0.5); ok {
		t.Fatal("a run with no positive answer proposes nothing")
	}
}
