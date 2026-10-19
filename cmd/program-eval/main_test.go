package main

import (
	"testing"

	"sitewise/internal/eval"
	"sitewise/internal/programread"
)

func TestFitWritesLoadableThresholdsWithN(t *testing.T) {
	var answers []eval.Answer
	for i := 0; i < 40; i++ {
		answers = append(answers, eval.Answer{Case: eval.SplitCalibration + "/c", Field: programread.Family, Options: 5, Confidence: 0.9 + float64(i%10)/100, Correct: true})
	}
	answers = append(answers, eval.Answer{Case: eval.SplitCalibration + "/c", Field: programread.Family, Options: 41, Confidence: 0.99, Correct: true})
	dir := t.TempDir()
	if err := writeThresholds(dir, answers); err != nil {
		t.Fatal(err)
	}
	th, err := programread.LoadThresholds(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := th.Questions[programread.Family]
	if len(list) != 2 {
		t.Fatalf("%+v", list)
	}
	for _, q := range list {
		if q.N == 0 {
			t.Fatalf("n not recorded: %+v", q)
		}
	}
	if !th.Calibrated(programread.Family) {
		t.Fatal("40 right answers fitted nothing")
	}
	// One answer in the 13–255 bucket is too thin: that shape stays blank.
	c := 0.99
	if _, apply := th.Band(programread.Family, &c, 41); apply {
		t.Fatal("thin bucket applied")
	}
}

func TestShippedThresholdsAreUncalibrated(t *testing.T) {
	th, err := programread.LoadThresholds("../../data/program")
	if err != nil {
		t.Fatal(err)
	}
	if th.Calibrated(programread.Family) {
		t.Fatal("program_row ships calibrated without a gold set")
	}
}
