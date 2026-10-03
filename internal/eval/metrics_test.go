package eval

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-3 }

func TestWilsonBounds(t *testing.T) {
	lo, hi := Wilson(0, 0, 1.645)
	if lo != 0 || hi != 1 {
		t.Fatalf("empty sample must be uninformative, got %v %v", lo, hi)
	}
	// 20 of 20 correct is not certainty: the lower bound stays below 1.
	lo, hi = Wilson(20, 20, 1.645)
	if !near(lo, 0.8808) || hi != 1 {
		t.Fatalf("20/20 got %v %v", lo, hi)
	}
	lo, hi = Wilson(5, 10, 1.96)
	if !near(lo, 0.2366) || !near(hi, 0.7634) {
		t.Fatalf("5/10 got %v %v", lo, hi)
	}
}

func TestSummarizeCountsAppliedAmberAndFalseConfident(t *testing.T) {
	got := Summarize([]Outcome{
		{Field: "number", Expected: "A1", Value: "A1", Band: "green", Correct: true},
		{Field: "number", Expected: "A2", Value: "A3", Band: "green"},
		{Field: "number", Expected: "A4", Value: "A4", Band: "amber", Correct: true},
		{Field: "number", Expected: "A5", Band: "blank"},
		{Field: "title", Expected: "PLAN", Band: "grey"},
	}, 1.645)
	n := got["number"]
	if n.Scored != 4 || n.Applied != 3 || n.Correct != 2 || n.Amber != 1 || n.FalseConfident != 1 {
		t.Fatalf("%+v", n)
	}
	if !near(n.Coverage, 0.75) || !near(n.Accuracy, 2.0/3) || !near(n.AmberRate, 0.25) {
		t.Fatalf("%+v", n)
	}
	if n.AccuracyLow >= n.Accuracy || n.AccuracyHigh <= n.Accuracy {
		t.Fatalf("interval must bracket accuracy: %+v", n)
	}
	title := got["title"]
	if title.Scored != 1 || title.Applied != 0 || title.Accuracy != 0 || title.Grey != 1 {
		t.Fatalf("an unapplied field has no accuracy claim: %+v", title)
	}
}

func TestShapeBucketsArePowersOfTwo(t *testing.T) {
	cases := map[int]Shape{2: {2, 2}, 3: {3, 4}, 4: {3, 4}, 5: {5, 8}, 12: {9, 16}, 57: {33, 64}, 255: {129, 255}}
	for options, want := range cases {
		if got := ShapeOf(options); got != want {
			t.Fatalf("%d: got %+v want %+v", options, got, want)
		}
	}
	if got := ShapeOf(1); got != (Shape{}) {
		t.Fatalf("one option is not a question: %+v", got)
	}
}

func answers(field string, options int, conf float64, correct bool, n int) []Answer {
	out := make([]Answer, n)
	for i := range out {
		out[i] = Answer{Field: field, Options: options, Confidence: conf, Correct: correct}
	}
	return out
}

var policy = FitPolicy{Z: 1.645, MinSamples: 30, MinBandSamples: 10, GreenLower: 0.9, AmberLower: 0.6, MaxErrors: -1}

func TestFitFindsLowestGreenCutMeetingLowerBound(t *testing.T) {
	var obs []Answer
	obs = append(obs, answers("number", 3, 0.99, true, 30)...)
	obs = append(obs, answers("number", 4, 0.80, true, 20)...)
	obs = append(obs, answers("number", 3, 0.80, false, 3)...)
	obs = append(obs, answers("number", 3, 0.40, false, 10)...)
	fits := Fit(obs, map[string]FitPolicy{"number": policy})
	if len(fits) != 1 {
		t.Fatalf("%+v", fits)
	}
	f := fits[0]
	if f.Field != "number" || f.Shape != (Shape{3, 4}) || f.N != 63 {
		t.Fatalf("%+v", f)
	}
	// At 0.80 the set is 50/53 correct: Wilson lower about 0.87, below 0.9.
	if f.Green == nil || *f.Green != 0.99 {
		t.Fatalf("green %+v", f)
	}
	// The amber band [0.80, 0.99) is 20/23 correct: lower bound about 0.71.
	if f.Amber == nil || *f.Amber != 0.80 {
		t.Fatalf("amber %+v", f)
	}
}

func TestFitLeavesUnknownWithTooFewSamples(t *testing.T) {
	fits := Fit(answers("title", 2, 0.99, true, 29), map[string]FitPolicy{"title": policy})
	if len(fits) != 1 || fits[0].Green != nil || fits[0].Amber != nil || fits[0].Reason == "" {
		t.Fatalf("%+v", fits)
	}
}

func TestFitAmberDoesNotRequireAutomaticGreen(t *testing.T) {
	p := FitPolicy{Z: 1.645, MinSamples: 30, MinBandSamples: 20, GreenLower: .95, AmberLower: .8, MaxErrors: -1}
	obs := answers("title", 3, .94, true, 30)
	obs = append(obs, answers("title", 3, .5, false, 10)...)
	f := Fit(obs, map[string]FitPolicy{"title": p})[0]
	// 30/30 supports an amber lower bound of .8, but not green .95.
	if f.Green != nil || f.Amber == nil || *f.Amber != .94 {
		t.Fatalf("independent review band: %+v", f)
	}
	thresholds := ThresholdsFrom([]FitResult{f}, "test", nil)
	if got := thresholds.Questions["title"]; len(got) != 1 || got[0].Green != nil || got[0].Amber == nil || *got[0].Amber != .94 {
		t.Fatalf("review-only fit lost on export: %+v", got)
	}
	p.MaxErrors = 0
	obs = append(obs, answers("title", 3, .94, false, 1)...)
	f = Fit(obs, map[string]FitPolicy{"title": p})[0]
	if f.Green != nil || f.Amber != nil {
		t.Fatalf("amber must still respect error policy: %+v", f)
	}
}

func TestFitWithoutPolicyIsUnknown(t *testing.T) {
	fits := Fit(answers("kind", 12, 0.99, true, 100), nil)
	if len(fits) != 1 || fits[0].Green != nil {
		t.Fatalf("a field with no policy must not be calibrated: %+v", fits)
	}
}

func TestFitEmptyAmberBandCollapsesToGreen(t *testing.T) {
	obs := answers("number", 3, 0.95, true, 40)
	obs = append(obs, answers("number", 3, 0.5, false, 20)...)
	fits := Fit(obs, map[string]FitPolicy{"number": policy})
	f := fits[0]
	if f.Green == nil || *f.Green != 0.95 || f.Amber == nil || *f.Amber != 0.95 {
		t.Fatalf("%+v", f)
	}
}

func TestStrictPolicyRejectsAnyErrorAboveTheCut(t *testing.T) {
	strict := policy
	strict.MaxErrors = 0
	obs := answers("supersedes", 2, 0.99, true, 200)
	obs = append(obs, answers("supersedes", 2, 0.99, false, 1)...)
	fits := Fit(obs, map[string]FitPolicy{"supersedes": strict})
	if fits[0].Green != nil {
		t.Fatalf("one wrong answer at the cut must leave supersession unknown: %+v", fits[0])
	}
}

func TestFamiliesJoinSharedDirectoryOrNumber(t *testing.T) {
	fam := Families([]Member{
		{ID: "a", Keys: []string{"dir:p/ELEC", "number:p/E01"}},
		{ID: "b", Keys: []string{"dir:p/ELEC-OLD", "number:p/E01"}},
		{ID: "c", Keys: []string{"dir:p/STRUCT"}},
		{ID: "d", Keys: []string{"dir:p/STRUCT", "number:p/S01"}},
		{ID: "e", Keys: []string{"dir:q/ELEC"}},
	})
	if fam["a"] != fam["b"] {
		t.Fatalf("revisions of one number must share a family: %v", fam)
	}
	if fam["c"] != fam["d"] || fam["c"] == fam["a"] || fam["e"] == fam["a"] {
		t.Fatalf("%v", fam)
	}
	if fam["a"] != "dir:p/ELEC" {
		t.Fatalf("family id is the smallest key: %v", fam["a"])
	}
}

func TestSplitIsStableAndBounded(t *testing.T) {
	if SplitOf("dir:p/ELEC", "s", 0) != SplitHeldout || SplitOf("dir:p/ELEC", "s", 100) != SplitCalibration {
		t.Fatal("bounds")
	}
	first := SplitOf("dir:p/ELEC", "s", 60)
	for i := 0; i < 5; i++ {
		if SplitOf("dir:p/ELEC", "s", 60) != first {
			t.Fatal("split must be deterministic")
		}
	}
	cal := 0
	for i := 0; i < 1000; i++ {
		if SplitOf(string(rune('a'+i%26))+string(rune(i)), "s", 60) == SplitCalibration {
			cal++
		}
	}
	if cal < 520 || cal > 680 {
		t.Fatalf("hash split far from 60%%: %d/1000", cal)
	}
}

func TestRegressionsFlagLostCorrectAndNewFalseConfident(t *testing.T) {
	base := map[string]FieldMetrics{
		"number":   {Scored: 40, Correct: 30, FalseConfident: 1},
		"revision": {Scored: 40, Correct: 20, FalseConfident: 0},
		"title":    {Scored: 40, Correct: 10},
	}
	cur := map[string]FieldMetrics{
		"number":   {Scored: 40, Correct: 29, FalseConfident: 1},
		"revision": {Scored: 40, Correct: 25, FalseConfident: 1},
	}
	got := Regressions(base, cur)
	if len(got) != 3 {
		t.Fatalf("want lost correct, new false confident and missing field: %v", got)
	}
	if len(Regressions(base, base)) != 0 {
		t.Fatal("identical metrics are not a regression")
	}
}
