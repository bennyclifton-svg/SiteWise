package intake_test

import (
	"testing"

	"sitewise/internal/intake"
)

func TestRevisionP10AfterP2(t *testing.T) {
	cmp, comparable := intakeCompare(t, "P10", "P2")
	if !comparable || cmp != 1 {
		t.Fatalf("P10 vs P2: cmp=%d comparable=%v", cmp, comparable)
	}
	cmp, comparable = intakeCompare(t, "P2", "P10")
	if !comparable || cmp != -1 {
		t.Fatalf("P2 vs P10: cmp=%d comparable=%v", cmp, comparable)
	}
	cmp, comparable = intakeCompare(t, "P02", "P2")
	if !comparable || cmp != 0 {
		t.Fatalf("P02 vs P2: cmp=%d comparable=%v", cmp, comparable)
	}
	cmp, comparable = intakeCompare(t, "10", "2")
	if !comparable || cmp != 1 {
		t.Fatalf("10 vs 2: cmp=%d comparable=%v", cmp, comparable)
	}
	cmp, comparable = intakeCompare(t, "B", "A")
	if !comparable || cmp != 1 {
		t.Fatalf("B vs A: cmp=%d comparable=%v", cmp, comparable)
	}
	ordered, ok := intake.Sequence([]string{"P10", "P2", "P1"})
	if !ok || len(ordered) != 3 || ordered[0] != "P1" || ordered[1] != "P2" || ordered[2] != "P10" {
		t.Fatalf("sequence: %v %v", ordered, ok)
	}
}

func TestRevisionIncompatibleSeries(t *testing.T) {
	for _, pair := range [][2]string{{"P2", "C"}, {"A", "01"}, {"C1", "P10"}, {"C", "C1"}} {
		_, comparable := intakeCompare(t, pair[0], pair[1])
		if comparable {
			t.Fatalf("%s and %s compared as one series", pair[0], pair[1])
		}
	}
	ordered, ok := intake.Sequence([]string{"P10", "C", "P2"})
	if ok || ordered != nil {
		t.Fatalf("mixed series ordered: %v %v", ordered, ok)
	}
}

func intakeCompare(t *testing.T, a, b string) (int, bool) {
	t.Helper()
	return intake.Compare(a, b)
}
