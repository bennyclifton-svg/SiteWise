package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/store"
)

func TestRawChoicesDoNotCountAsAppliedAccuracy(t *testing.T) {
	s := summary{Metrics: map[string]counts{}}
	tr := trace{Entry: entry{Corpus: "fixture", Split: "heldout"}, Text: identity.Text{TextLayer: true},
		Choices:   []intake.Choice{{Field: "discipline", Value: "consultant.architect"}},
		Decisions: []intake.Decision{{Field: "discipline", Band: intake.BandBlank}}}
	score(&s, tr, goldCase{Fields: map[string]label{"discipline": {"consultant.architect", "copyright author"}}})
	for _, key := range []string{"discipline", "discipline:consultant.architect/discipline"} {
		c := s.Metrics[key]
		if c.RawJevCorrect != 1 || c.RawJevScored != 1 || c.Correct != 0 || c.Blank != 1 {
			t.Fatalf("%s: %+v", key, c)
		}
	}
}

func TestInventorySelectionAndMissingRoot(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"a.pdf": "one", "b.PDF": "two", "copy.pdf": "one", "._a.pdf": "metadata", "ignore.txt": "text"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	all, err := inventory([]root{{"fixture", dir}}, "seed")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("inventory %d", len(all))
	}
	selected := selectEntries(all, 0, "seed")
	if len(selected) != 2 {
		t.Fatalf("selected %+v", selected)
	}
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if !reflect.DeepEqual(selected, selectEntries(all, 0, "seed")) {
		t.Fatal("selection depends on input order")
	}
	if len(selectEntries(all, 1, "seed")) != 1 {
		t.Fatal("per-family cap ignored")
	}
	if _, err := inventory([]root{{"missing", filepath.Join(dir, "missing")}}, "seed"); err == nil {
		t.Fatal("missing corpus silently accepted")
	}
}

func TestHeldoutAnswersNeverFitThresholds(t *testing.T) {
	confidence := .99
	tr := trace{Entry: entry{Split: eval.SplitHeldout}, Choices: []intake.Choice{{Field: "number", Value: "A101", Options: 3, Confidence: &confidence}}}
	g := goldCase{Fields: map[string]label{"number": {"A101", "page 1"}}}
	if len(calibrationAnswers(tr, g)) != 0 {
		t.Fatal("heldout leaked into calibration")
	}
	tr.Entry.Split = eval.SplitCalibration
	got := calibrationAnswers(tr, g)
	if len(got) != 1 || !got[0].Correct {
		t.Fatalf("%+v", got)
	}
	tr.Error = "provider deadline"
	if len(calibrationAnswers(tr, g)) != 0 {
		t.Fatal("failed call entered calibration")
	}
}

func TestSourceSupportedAlternativesRetainPrimaryLabel(t *testing.T) {
	g := goldCase{Fields: map[string]label{"title": {"PLAN AND DETAILS", "register"}}, Alternatives: map[string][]label{"title": {{"PLAN & DETAILS", "explicit PDF title cell"}}}}
	if !matchesGold(g, "title", g.Fields["title"], "PLAN & DETAILS") || !matchesGold(g, "title", g.Fields["title"], "PLAN AND DETAILS") {
		t.Fatal("source alternative missing")
	}
	if matchesGold(g, "title", g.Fields["title"], "OTHER PLAN") {
		t.Fatal("unrecorded alternative accepted")
	}
}

func TestProvisionalLabelsExcludedOnlyFromCalibration(t *testing.T) {
	confidence := .99
	tr := trace{Entry: entry{Split: eval.SplitCalibration}, Choices: []intake.Choice{
		{Field: "kind", Value: "unknown", Options: 12, Confidence: &confidence},
		{Field: "number", Value: "A101", Options: 3, Confidence: &confidence},
	}}
	g := goldCase{Fields: map[string]label{"kind": {"unknown", "no manual category"}, "number": {"A101", "page 1"}}, CalibrationExcluded: map[string]string{"kind": "manual category absent"}}
	got := calibrationAnswers(tr, g)
	if len(got) != 1 || got[0].Field != "number" {
		t.Fatalf("provisional label entered fit: %+v", got)
	}
	if !matchesGold(g, "kind", g.Fields["kind"], "unknown") {
		t.Fatal("calibration exclusion changed scoring")
	}
}

func TestAppScoringUsesPersistedFields(t *testing.T) {
	r := appResult{Document: store.DocumentView{Status: store.StatusFiled, Fields: []store.FieldView{{Field: "number", Value: "S101", Band: "green"}}}}
	tr := appTrace(r)
	if !tr.Text.TextLayer || len(tr.Decisions) != 1 || tr.Decisions[0].Value != "S101" {
		t.Fatalf("%+v", tr)
	}
	r.Document.Status = store.StatusNotFiled
	r.Document.Reason = intake.ReasonNoText
	if got := appTrace(r); got.Error != "no text layer on identity page" || got.Text.TextLayer {
		t.Fatalf("%+v", got)
	}
}

func TestBaselineRejectsLostCorrectValues(t *testing.T) {
	b := summary{Files: 2, Metrics: map[string]counts{"number": {Scored: 2, Correct: 2}}}
	a := summary{Files: 2, Metrics: map[string]counts{"number": {Scored: 2, Correct: 1, Blank: 1}}}
	if len(regressions(b, a)) == 0 {
		t.Fatal("blank regression passed")
	}
	if len(regressions(b, b)) != 0 {
		t.Fatal("unchanged baseline failed")
	}
}

func TestP90DoesNotHideSlowSmallSample(t *testing.T) {
	if percentile([]float64{1, 3000}, 90) != 3000 {
		t.Fatal("p90 ignored the slower file")
	}
}

func TestScoringBlanksFalseGreenAndCandidateRecall(t *testing.T) {
	s := summary{Metrics: map[string]counts{}}
	tr := trace{Text: identity.Text{TextLayer: true}, Candidates: []intake.Candidate{}, Decisions: []intake.Decision{{Field: "number", Value: "S999", Band: "green"}}}
	score(&s, tr, goldCase{Fields: map[string]label{"number": {"S101", "page 1"}, "revision": {"A", "page 1"}}})
	if got := s.Metrics["number"]; got.Correct != 0 || got.Wrong != 1 || got.FalseGreen != 1 || got.CandidateMissing != 1 {
		t.Fatalf("%+v", got)
	}
	if got := s.Metrics["revision"]; got.Correct != 0 || got.Blank != 1 {
		t.Fatalf("%+v", got)
	}
}
