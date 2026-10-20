package main

import (
	"strings"
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
	if err := writeThresholds(dir, programread.Family, answers); err != nil {
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

func TestStageWeeksFitKeepsProgramRowAndVersionsItself(t *testing.T) {
	dir := t.TempDir()
	var rows, stages []eval.Answer
	for i := 0; i < 40; i++ {
		rows = append(rows, eval.Answer{Case: eval.SplitCalibration + "/c", Field: programread.Family, Options: 5, Confidence: 0.9 + float64(i%10)/100, Correct: true})
		stages = append(stages, eval.Answer{Case: eval.SplitCalibration + "/s", Field: programread.StageWeeksFamily, Options: 3, Confidence: 0.9 + float64(i%10)/100, Correct: true})
	}
	if err := writeThresholds(dir, programread.Family, rows); err != nil {
		t.Fatal(err)
	}
	if err := writeThresholds(dir, programread.StageWeeksFamily, stages); err != nil {
		t.Fatal(err)
	}
	th, err := programread.LoadThresholds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !th.Calibrated(programread.Family) || !th.Calibrated(programread.StageWeeksFamily) || th.Versions[programread.StageWeeksFamily] != "proposal-program-1" {
		t.Fatalf("%+v", th)
	}
	c := 0.99
	if _, apply := th.Band(programread.StageWeeksFamily, &c, 3); !apply {
		t.Fatal("fitted stage weeks cut-off not applied")
	}
}

func TestShippedStageWeeksThresholdsAreUncalibrated(t *testing.T) {
	th, err := programread.LoadThresholds("../../data/program")
	if err != nil {
		t.Fatal(err)
	}
	if th.Calibrated(programread.StageWeeksFamily) {
		t.Fatal("program_stage_weeks ships calibrated without a gold set")
	}
	var out strings.Builder
	if code := run([]string{"-family", programread.StageWeeksFamily, "-key", t.TempDir() + "/missing.json"}, func(string) string { return "" }, &out, &out); code != 1 || !strings.Contains(out.String(), "no gold set") {
		t.Fatalf("%d %s", code, out.String())
	}
}
