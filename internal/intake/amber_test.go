package intake

import "testing"

func TestCalibratedAmberOnlyNeverTurnsGreen(t *testing.T) {
	n, hi, amber := 9, 16, .8
	q := Threshold{Options: &n, MaxOptions: &hi, Amber: &amber, N: 93}
	if err := validateThreshold(FieldKind, q); err != nil {
		t.Fatal(err)
	}
	ts := Thresholds{QuestionVersion: QuestionVersion, Questions: map[string][]Threshold{FieldKind: {q}}}
	for _, confidence := range []float64{.8, .95, 1} {
		if band, apply := ts.Band(FieldKind, &confidence, 12); band != BandAmber || !apply {
			t.Fatalf("amber-only confidence %v: %s %v", confidence, band, apply)
		}
	}
	low := .79
	if _, apply := ts.Band(FieldKind, &low, 12); apply {
		t.Fatal("below-cutoff answer applied")
	}
	if _, apply := ts.Band(FieldKind, &amber, 17); apply {
		t.Fatal("unknown shape applied")
	}
	q.N = 0
	if validateThreshold(FieldKind, q) == nil {
		t.Fatal("uncalibrated partial threshold accepted")
	}
}
