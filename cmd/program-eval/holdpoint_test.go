package main

import (
	"os"
	"path/filepath"
	"testing"

	"sitewise/internal/eval"
	"sitewise/internal/programread"
)

func TestReleaseFitWritesNoulCutOffAndKeepsOtherFamilies(t *testing.T) {
	dir := t.TempDir()
	shipped, err := os.ReadFile(filepath.Join("..", "..", "data", "program", "thresholds.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "thresholds.json"), shipped, 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := programread.LoadThresholds(dir)
	if err != nil {
		t.Fatal(err)
	}

	// No gold set: the family is written with no cut-off and proposes nothing.
	if err := writeReleaseThresholds(dir, nil); err != nil {
		t.Fatal(err)
	}
	th, err := programread.LoadThresholds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if th.ReleaseCalibrated() || len(th.Questions[programread.HoldPointFamily]) != 1 || th.Versions[programread.HoldPointFamily] != programread.HoldPointReleaseVersion {
		t.Fatalf("empty fit %+v", th)
	}
	if _, apply := th.BandRelease(0.99); apply {
		t.Fatal("uncalibrated release applied")
	}

	var answers []eval.Answer
	for i := 0; i < 40; i++ {
		answers = append(answers, eval.Answer{Case: eval.SplitCalibration + "/c", Field: programread.HoldPointFamily, Options: 2, Confidence: 0.9 + float64(i%10)/100, Correct: true})
	}
	if err := writeReleaseThresholds(dir, answers); err != nil {
		t.Fatal(err)
	}
	th, err = programread.LoadThresholds(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := th.Questions[programread.HoldPointFamily]
	if len(list) != 1 || list[0].Options != nil || list[0].N != 40 || !th.ReleaseCalibrated() {
		t.Fatalf("fit %+v", list)
	}
	// 40 right answers meet the amber bound; green needs more evidence.
	if band, apply := th.BandRelease(0.99); !apply || band == programread.BandBlank {
		t.Fatalf("band %s %v", band, apply)
	}
	if _, apply := th.BandRelease(0.5); apply {
		t.Fatal("an answer below every calibrated probability applied")
	}
	for family, v := range before.Versions {
		if th.Versions[family] != v {
			t.Fatalf("%s version lost", family)
		}
	}
	if th.QuestionVersion != programread.QuestionVersion {
		t.Fatal("top-level version changed")
	}
}

func TestShippedReleaseThresholdsAreUncalibrated(t *testing.T) {
	th, err := programread.LoadThresholds("../../data/program")
	if err != nil {
		t.Fatal(err)
	}
	if th.ReleaseCalibrated() {
		t.Fatal("hold_point_release ships calibrated without a gold set")
	}
}
