package programread

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
)

func releaseQuestion() knowledge.Question {
	// YAML reads `true:` and `false:` as boolean keys.
	return knowledge.Question{Type: "noul", Instructions: "Using `text`, does this passage state that the inspection described in `hold_point` was carried out and the work was released to proceed?",
		Criteria: map[any]any{true: "It states the inspection was completed and the work may proceed.", false: "It only requires, schedules or requests the inspection."}}
}

func TestReleaseQuestionHasStringCriteria(t *testing.T) {
	q := ReleaseQuestion(releaseQuestion())
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := q.Criteria.(map[string]string)
	if q.Type != jev.TypeNoul || !ok || c["true"] == "" || c["false"] == "" || len(c) != 2 {
		t.Fatalf("question %s", raw)
	}
}

// Recorded answers through the real client, banded on probability alone:
// a noul has no confidence (https://docs.typesafe.ai/confidence).
func TestReleaseBandingWithRecordedResponses(t *testing.T) {
	client, _ := recorded(t, "hold_point_release.json")
	q := ReleaseQuestion(releaseQuestion())
	res, err := client.Ask(context.Background(), jev.Call{
		State:     map[string]any{"text": "Frame inspected and accepted; roofing may proceed.", HoldPointStateKey: "Frame inspection record accepting the support construction."},
		Questions: map[string]jev.Question{"hold:if.green": q, "hold:if.amber": q, "hold:if.low": q}, Priority: jev.PriorityBackground, QuestionVersion: HoldPointReleaseVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	th, err := LoadThresholds(filepath.Join("testdata", "holdpoint"))
	if err != nil {
		t.Fatal(err)
	}
	if !th.ReleaseCalibrated() {
		t.Fatal("fixture cut-off not read")
	}
	want := map[string]struct {
		band  string
		apply bool
	}{"hold:if.green": {BandGreen, true}, "hold:if.amber": {BandAmber, true}, "hold:if.low": {BandBlank, false}}
	for id, w := range want {
		a := res.Answers[id]
		if a.Type != jev.TypeNoul || a.Confidence != nil {
			t.Fatalf("%s answer %+v", id, a)
		}
		if band, apply := th.BandRelease(a.Noul); band != w.band || apply != w.apply {
			t.Fatalf("%s: band %s %v, want %s %v", id, band, apply, w.band, w.apply)
		}
	}

	// No cut-off, the shipped file, or another version: nothing applies.
	shipped, err := LoadThresholds(filepath.Join("..", "..", "data", "program"))
	if err != nil {
		t.Fatal(err)
	}
	for name, t2 := range map[string]Thresholds{
		"empty":   {QuestionVersion: QuestionVersion},
		"shipped": shipped,
		"version": {QuestionVersion: QuestionVersion, Versions: map[string]string{HoldPointFamily: "hold-point-release-0"}, Questions: th.Questions},
		"shaped":  {QuestionVersion: QuestionVersion, Versions: th.Versions, Questions: map[string][]Threshold{HoldPointFamily: {{Options: holdIntp(2), Green: holdFp(0.9), Amber: holdFp(0.6), N: 40}}}},
		"invalid": {QuestionVersion: QuestionVersion, Versions: th.Versions, Questions: map[string][]Threshold{HoldPointFamily: {{Green: holdFp(0.5), Amber: holdFp(0.6), N: 40}}}},
	} {
		if _, apply := t2.BandRelease(0.99); apply || t2.ReleaseCalibrated() {
			t.Fatalf("%s: an answer applied without a release cut-off", name)
		}
	}
}

func holdIntp(v int) *int       { return &v }
func holdFp(v float64) *float64 { return &v }
