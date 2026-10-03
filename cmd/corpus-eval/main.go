// corpus-eval inventories every PDF, chooses reproducible folder-stratified
// samples, and exposes the production intake path for evidence-based diagnosis.
// Private documents and traces belong under data/eval/intake/private only.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

type root struct{ Name, Path string }
type entry struct {
	Corpus, Path, Relative, SHA256, Family, Split string
	Bytes                                         int64
	DuplicateOf                                   string `json:",omitempty"`
	Excluded                                      string `json:",omitempty"`
}
type trace struct {
	Entry                 entry
	Error                 string `json:",omitempty"`
	FailureStage          string `json:",omitempty"`
	Text                  identity.Text
	Candidates            []intake.Candidate
	Decisions             []intake.Decision
	Choices               []intake.Choice
	ExtractionMS, TotalMS float64
}
type appResult struct {
	Entry            entry
	Document         store.DocumentView
	ObservedFilingMS float64
	Error            string
}
type label struct{ Value, Evidence string }
type goldCase struct {
	SHA256 string
	// For a scan/unreadable input, record the expected extraction failure instead
	// of demanding seven invented identity fields. Evidence must cite the page.
	NotFiled string `json:",omitempty"`
	Evidence string `json:",omitempty"`
	// Evidence is transcribed from a source page independently of predictions.
	Fields       map[string]label
	Alternatives map[string][]label `json:",omitempty"`
	// Provisional taxonomy mappings remain scored but must not train thresholds.
	CalibrationExcluded map[string]string `json:",omitempty"`
}
type counts struct {
	Scored, Correct, Blank, Wrong, FalseGreen, CandidateMissing int
	ExpectedAbsent, CorrectAbsent                               int
	RawJevScored, RawJevCorrect                                 int
}
type reviewItem struct {
	SHA256, Corpus, Field, Reason string
	Expected                      *label           `json:",omitempty"`
	Actual                        *intake.Decision `json:",omitempty"`
}
type summary struct {
	Mode                                                                          string
	Files, Errors, Unlabelled                                                     int
	ProviderErrors                                                                int
	Metrics                                                                       map[string]counts
	P50MS, P90MS                                                                  float64
	Passed                                                                        bool
	Failures                                                                      []string
	ExpectedNotFiled                                                              int
	ExtractionP50MS, ExtractionP90MS                                              float64
	FullyLabelledDocuments, FullyCorrectDocuments, FullyPopulatedCorrectDocuments int
}
type unavailable struct{}

func (unavailable) Ask(context.Context, jev.Call) (jev.Result, error) {
	return jev.Result{}, errors.New("rules-only: Jev not called")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	rootsFile := flag.String("roots", "data/eval/intake/corpus-roots.json", "named PDF roots")
	out := flag.String("out", "data/eval/intake/private/corpus", "private output directory")
	mode := flag.String("mode", "scan", "scan, rules, live, replay, score, or app-score")
	tracesDir := flag.String("traces", "", "existing trace directory for score mode")
	appFile := flag.String("app-results", "", "results.json from corpus-upload.py for app-score")
	perFolder := flag.Int("per-folder", 2, "stable samples per leaf folder; 0 selects all unique PDFs")
	seed := flag.String("seed", "sitewise-corpus-1", "fixed selection/split salt")
	goldFile := flag.String("gold", "", "independent evidence-backed labels, keyed by SHA256")
	baselineFile := flag.String("baseline", "", "previous summary.json; fail on fewer correct values or more false green")
	data := flag.String("data", "data/intake", "catalog and thresholds")
	recordsFile := flag.String("recordings", "", "exact request recordings for live/replay")
	minScore := flag.Float64("min-success", .95, "minimum correct/scored per field (blanks fail)")
	gate := flag.Bool("gate", false, "fail on errors, unlabelled files/fields, low success or false green")
	fit := flag.Bool("fit", false, "fit experimental thresholds on calibration labels only; writes <out>/calibrated")
	deadline := flag.Duration("deadline", jev.InteractiveDeadline, "Jev deadline; default matches production")
	flag.Parse()
	if *mode != "scan" && *mode != "rules" && *mode != "live" && *mode != "replay" && *mode != "score" && *mode != "app-score" {
		return errors.New("unknown mode")
	}
	if *perFolder < 0 || *minScore <= 0 || *minScore > 1 || *deadline <= 0 {
		return errors.New("invalid sample, success or deadline setting")
	}
	if *mode == "score" && *tracesDir == "" {
		return errors.New("score requires -traces")
	}
	appTraces := map[string]trace{}
	if *mode == "app-score" {
		var results []appResult
		if err := readJSON(*appFile, &results); err != nil {
			return err
		}
		for _, r := range results {
			if _, ok := appTraces[r.Entry.SHA256]; ok {
				return errors.New("duplicate app result hash")
			}
			appTraces[r.Entry.SHA256] = appTrace(r)
		}
	}
	if *fit && *mode != "live" && *mode != "replay" {
		return errors.New("fit requires current live/replay evidence")
	}
	var roots []root
	if err := readJSON(*rootsFile, &roots); err != nil {
		return err
	}
	entries, err := inventory(roots, *seed)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0700); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(*out, "inventory.json"), entries); err != nil {
		return err
	}
	selected := selectEntries(entries, *perFolder, *seed)
	if err := writeJSON(filepath.Join(*out, "selected.json"), selected); err != nil {
		return err
	}
	fmt.Printf("inventoried %d PDFs; selected %d unique PDFs\n", len(entries), len(selected))
	if *mode == "scan" {
		return nil
	}
	var gold []goldCase
	if *goldFile != "" {
		if err := readJSON(*goldFile, &gold); err != nil {
			return err
		}
	}
	labels := map[string]goldCase{}
	for _, g := range gold {
		if _, exists := labels[g.SHA256]; exists {
			return fmt.Errorf("duplicate gold hash %s", g.SHA256)
		}
		for f, l := range g.Fields {
			if !knownField(f) || strings.TrimSpace(l.Evidence) == "" {
				return fmt.Errorf("gold %s/%s needs valid field and source evidence", g.SHA256, f)
			}
		}
		for f, reason := range g.CalibrationExcluded {
			if _, ok := g.Fields[f]; !ok || strings.TrimSpace(reason) == "" {
				return errors.New("calibration exclusion requires a labelled field and reason")
			}
		}
		for f, alternatives := range g.Alternatives {
			if _, ok := g.Fields[f]; !ok {
				return errors.New("gold alternatives require a primary field label")
			}
			for _, l := range alternatives {
				if strings.TrimSpace(l.Evidence) == "" {
					return errors.New("gold alternative needs independent source evidence")
				}
			}
		}
		if g.NotFiled != "" && (g.Evidence == "" || len(g.Fields) > 0) {
			return errors.New("not-filed gold requires evidence and no identity labels")
		}
		labels[g.SHA256] = g
	}
	cat, err := intake.LoadCatalog(*data)
	if err != nil {
		return err
	}
	thresholds, err := intake.LoadThresholds(*data)
	if err != nil {
		return err
	}
	// Freeze the exact draft labels/config used by this run. A later annotation
	// revision must not silently change the meaning of an earlier report.
	if err := writeJSON(filepath.Join(*out, "gold-snapshot.json"), gold); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(*out, "run.json"), map[string]any{
		"mode": *mode, "started_utc": time.Now().UTC(), "runner_question_version": intake.QuestionVersion,
		"model": config.PinnedJevModel, "seed": *seed, "per_folder": *perFolder,
		"gold_sha256": jsonHash(gold), "selection_sha256": jsonHash(selected),
		"thresholds": thresholds, "deadline": deadline.String(),
	}); err != nil {
		return err
	}
	ctx := context.Background()
	if *mode != "score" && *mode != "app-score" {
		if err := identity.Warm(ctx); err != nil {
			return err
		}
	}
	var asker intake.Asker = unavailable{}
	var rec *eval.Recorder
	var rep *eval.Replayer
	if *recordsFile == "" {
		*recordsFile = filepath.Join(*out, "recordings.jsonl")
	}
	if *mode == "live" || *mode == "replay" {
		var transport http.RoundTripper
		key := "replay"
		if *mode == "live" {
			key = os.Getenv("SITEWISE_JEV_API_KEY")
			if key == "" {
				key = os.Getenv("TYPESAFE_API_KEY")
			}
			if key == "" {
				return errors.New("SITEWISE_JEV_API_KEY required")
			}
			f, err := os.OpenFile(*recordsFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("use a new recording path for each live experiment: %w", err)
			}
			defer f.Close()
			rec = eval.NewRecorder(jev.NewTransport(), f)
			transport = rec
		} else {
			f, err := os.Open(*recordsFile)
			if err != nil {
				return err
			}
			records, err := eval.ReadRecordings(f)
			f.Close()
			if err != nil {
				return err
			}
			rep = eval.NewReplayer(records, false)
			transport = rep
		}
		client, err := jev.New(jev.Options{APIKey: key, Model: config.PinnedJevModel, Transport: transport, Logger: slog.New(slog.DiscardHandler), BreakerThreshold: 1 << 20})
		if err != nil {
			return err
		}
		if *mode == "live" {
			if err := client.Warm(ctx); err != nil {
				return err
			}
		}
		asker = client
	}
	report := summary{Mode: *mode, Files: len(selected), Metrics: map[string]counts{}}
	var times []float64
	var extractionTimes []float64
	var calibration []eval.Answer
	var review []reviewItem
	for i, e := range selected {
		if rec != nil {
			rec.SetCase(e.SHA256)
		}
		var tr trace
		if *mode == "app-score" {
			var ok bool
			tr, ok = appTraces[e.SHA256]
			if !ok {
				return fmt.Errorf("missing app result for %s", e.SHA256)
			}
		} else if *mode == "score" {
			if err := readJSON(filepath.Join(*tracesDir, e.SHA256+".json"), &tr); err != nil {
				return err
			}
			if tr.Entry.SHA256 != e.SHA256 {
				return errors.New("trace hash does not match selected PDF")
			}
		} else {
			tr = classify(ctx, e, cat, thresholds, asker, *deadline)
		}
		g, ok := labels[e.SHA256]
		review = append(review, reviewFailures(tr, g, ok)...)
		if tr.FailureStage == "provider" {
			report.ProviderErrors++
		}
		if ok && g.NotFiled != "" {
			report.ExpectedNotFiled++
			if tr.Error != g.NotFiled {
				report.Failures = append(report.Failures, "unexpected filing status for "+e.SHA256)
			}
		} else if tr.Error != "" && tr.Error != "rules-only: Jev not called" {
			report.Errors++
		}
		times = append(times, tr.TotalMS)
		extractionTimes = append(extractionTimes, tr.ExtractionMS)
		if err := writeJSON(filepath.Join(*out, "traces", e.SHA256+".json"), tr); err != nil {
			return err
		}
		if !ok {
			report.Unlabelled++
		} else {
			score(&report, tr, g)
			calibration = append(calibration, calibrationAnswers(tr, g)...)
		}
		fmt.Printf("%d/%d %s: %d runs, %d candidates, %.0fms %s\n", i+1, len(selected), e.Corpus, len(tr.Text.Runs), len(tr.Candidates), tr.TotalMS, tr.Error)
	}
	if rec != nil && rec.Err() != nil {
		return rec.Err()
	}
	if err := writeJSON(filepath.Join(*out, "review.json"), review); err != nil {
		return err
	}
	if rep != nil && rep.Misses() > 0 {
		report.Failures = append(report.Failures, fmt.Sprintf("%d replay misses", rep.Misses()))
	}
	if *fit {
		if report.ProviderErrors > 0 || (rep != nil && rep.Misses() > 0) {
			return errors.New("cannot fit with provider errors or replay misses")
		}
		policies := map[string]eval.FitPolicy{}
		for _, f := range fields {
			policies[f] = eval.FitPolicy{Z: 1.645, MinSamples: 30, MinBandSamples: 20, GreenLower: .95, AmberLower: .8, MaxErrors: -1}
		}
		fits := eval.Fit(calibration, policies)
		thresholds := eval.ThresholdsFrom(fits, "Experimental corpus calibration only; independent labels, calibration families only. Promote only after held-out and real-app gates pass.", map[string]any{"seed": *seed, "calibration_answers": len(calibration), "fits": fits})
		configDir := filepath.Join(*out, "calibrated")
		if err := writeJSON(filepath.Join(configDir, "thresholds.json"), thresholds); err != nil {
			return err
		}
		for _, name := range []string{"kinds.json", "disciplines.json", "lifecycle.json"} {
			b, err := os.ReadFile(filepath.Join(*data, name))
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(configDir, name), b, 0600); err != nil {
				return err
			}
		}
	}
	sort.Float64s(times)
	sort.Float64s(extractionTimes)
	if len(times) > 0 {
		report.P50MS = percentile(times, 50)
		report.P90MS = percentile(times, 90)
		report.ExtractionP50MS = percentile(extractionTimes, 50)
		report.ExtractionP90MS = percentile(extractionTimes, 90)
	}
	if report.Errors > 0 {
		report.Failures = append(report.Failures, fmt.Sprintf("%d filing/extraction errors", report.Errors))
	}
	if report.Unlabelled > 0 {
		report.Failures = append(report.Failures, fmt.Sprintf("%d selected PDFs have no gold labels", report.Unlabelled))
	}
	for _, field := range fields {
		c := report.Metrics[field]
		if c.Scored < len(selected)-report.ExpectedNotFiled {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: %d/%d labelled", field, c.Scored, len(selected)-report.ExpectedNotFiled))
		}
		if c.Scored == 0 || float64(c.Correct)/float64(c.Scored) < *minScore || c.FalseGreen > 0 {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: %d/%d correct, %d false green", field, c.Correct, c.Scored, c.FalseGreen))
		}
	}
	groups := make([]string, 0, len(report.Metrics))
	for name := range report.Metrics {
		if strings.Contains(name, "/") {
			groups = append(groups, name)
		}
	}
	sort.Strings(groups)
	for _, name := range groups {
		c := report.Metrics[name]
		if c.Scored > 0 && float64(c.Correct)/float64(c.Scored) < *minScore {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: %d/%d correct", name, c.Correct, c.Scored))
		}
	}
	if *baselineFile != "" {
		var baseline summary
		if err := readJSON(*baselineFile, &baseline); err != nil {
			return err
		}
		report.Failures = append(report.Failures, regressions(baseline, report)...)
	}
	if report.ExtractionP50MS > 80 || report.ExtractionP90MS > 250 {
		report.Failures = append(report.Failures, "identity extraction exceeded 80/250ms p50/p90 budget")
	}
	if *mode == "live" && (report.P50MS > 1000 || report.P90MS > 2000) {
		report.Failures = append(report.Failures, "filing draft exceeded 1000/2000ms p50/p90 budget (excludes commit)")
	}
	if *mode == "app-score" && (report.P50MS > 1000 || report.P90MS > 2000) {
		report.Failures = append(report.Failures, "observed app filing exceeded 1000/2000ms p50/p90 budget")
	}
	report.Passed = len(report.Failures) == 0 && *mode != "rules"
	if err := writeJSON(filepath.Join(*out, "summary.json"), report); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
	if *gate && !report.Passed {
		return errors.New("corpus accuracy gate failed; see summary.json")
	}
	return nil
}

// Nearest rank: a two-file smoke test's p90 must include its slower file.
func percentile(sorted []float64, percent int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[(len(sorted)*percent+99)/100-1]
}

func regressions(before, after summary) []string {
	var out []string
	if before.Files != after.Files {
		out = append(out, "baseline population changed")
	}
	keys := make([]string, 0, len(before.Metrics))
	for k := range before.Metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := before.Metrics[k]
		a := after.Metrics[k]
		if a.Scored != b.Scored {
			out = append(out, k+": baseline label count changed")
		}
		if a.Correct < b.Correct || a.FalseGreen > b.FalseGreen {
			out = append(out, k+": accuracy regression against baseline")
		}
	}
	return out
}

func inventory(roots []root, seed string) ([]entry, error) {
	if len(roots) == 0 {
		return nil, errors.New("no corpus roots")
	}
	var out []entry
	names := map[string]bool{}
	for _, r := range roots {
		if r.Name == "" || names[r.Name] {
			return nil, errors.New("root names must be unique and nonempty")
		}
		names[r.Name] = true
		base, err := filepath.Abs(r.Path)
		if err != nil {
			return nil, err
		}
		n := len(out)
		err = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".pdf") {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("PDF symlink unsupported: %s", p)
			}
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			h := sha256.New()
			size, err := io.Copy(h, f)
			f.Close()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			family := r.Name + "/" + filepath.ToSlash(filepath.Dir(rel))
			out = append(out, entry{Corpus: r.Name, Path: p, Relative: rel, SHA256: hex.EncodeToString(h.Sum(nil)), Bytes: size, Family: family, Split: eval.SplitOf(family, seed, 60)})
			if strings.HasPrefix(d.Name(), "._") {
				out[len(out)-1].Excluded = "AppleDouble metadata sidecar"
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if len(out) == n {
			return nil, fmt.Errorf("no PDFs in %s", base)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	seen := map[string]string{}
	for i := range out {
		e := &out[i]
		if path, ok := seen[e.SHA256]; ok {
			e.DuplicateOf = path
		} else {
			seen[e.SHA256] = e.Path
		}
	}
	return out, nil
}
func selectEntries(entries []entry, n int, seed string) []entry {
	unique := []entry{}
	for _, e := range entries {
		if e.DuplicateOf == "" && e.Excluded == "" {
			unique = append(unique, e)
		}
	}
	sort.Slice(unique, func(i, j int) bool {
		a := eval.SHA256Hex([]byte(seed + unique[i].SHA256))
		b := eval.SHA256Hex([]byte(seed + unique[j].SHA256))
		return a < b
	})
	counts := map[string]int{}
	out := []entry{}
	for _, e := range unique {
		if n == 0 || counts[e.Family] < n {
			out = append(out, e)
			counts[e.Family]++
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
func classify(ctx context.Context, e entry, cat intake.Catalog, t intake.Thresholds, a intake.Asker, deadline time.Duration) trace {
	tr := trace{Entry: e}
	start := time.Now()
	f, err := os.Open(e.Path)
	if err != nil {
		tr.Error = err.Error()
		tr.FailureStage = "open"
		return tr
	}
	tr.Text, err = identity.Extract(ctx, "pdf", f, e.Bytes, identity.DefaultLimits())
	f.Close()
	tr.ExtractionMS = float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		tr.Error = err.Error()
		tr.FailureStage = "extract"
	} else if !tr.Text.TextLayer {
		tr.Error = "no text layer on identity page"
		tr.FailureStage = "extract"
	} else {
		tr.Candidates = intake.Harvest(filepath.Base(e.Path), tr.Text)
		d := intake.NewDraft(cat, filepath.Base(e.Path), tr.Text, tr.Candidates, nil)
		d.Plan(nil, e.SHA256)
		if call, ok := d.Call(); ok {
			call.Deadline = deadline
			result, err := a.Ask(ctx, call)
			if err != nil {
				tr.Error = err.Error()
				tr.FailureStage = "provider"
			}
			tr.Choices = d.Choices(result)
			d.Apply(result, err, t)
		}
		tr.Decisions = d.Decisions()
	}
	tr.TotalMS = float64(time.Since(start).Microseconds()) / 1000
	return tr
}

var fields = []string{"kind", "discipline", "lifecycle", "number", "revision", "title", "date"}

func calibrationAnswers(tr trace, g goldCase) []eval.Answer {
	if tr.Entry.Split != eval.SplitCalibration || tr.Error != "" {
		return nil
	}
	var out []eval.Answer
	for _, choice := range tr.Choices {
		l, ok := g.Fields[choice.Field]
		if !ok || choice.Confidence == nil || g.CalibrationExcluded[choice.Field] != "" {
			continue
		}
		out = append(out, eval.Answer{Case: tr.Entry.SHA256, Field: choice.Field, Options: choice.Options, Confidence: *choice.Confidence, Correct: matchesGold(g, choice.Field, l, choice.Value)})
	}
	return out
}

func appTrace(r appResult) trace {
	tr := trace{Entry: r.Entry, TotalMS: r.ObservedFilingMS, Text: identity.Text{TextLayer: r.Document.Status == store.StatusFiled}}
	if r.Error != "" {
		tr.Error = r.Error
		return tr
	}
	if r.Document.Status != store.StatusFiled {
		switch r.Document.Reason {
		case intake.ReasonNoText:
			tr.Error = "no text layer on identity page"
		case intake.ReasonTooLarge:
			tr.Error = identity.ErrTooLarge.Error()
		case intake.ReasonUnreadable:
			tr.Error = identity.ErrMalformed.Error()
		default:
			tr.Error = "unexpected document status: " + r.Document.Status
		}
	}
	for _, d := range r.Document.Fields {
		tr.Decisions = append(tr.Decisions, intake.Decision{Field: d.Field, Value: d.Value, Band: d.Band, DecidedBy: d.DecidedBy, Confidence: d.Confidence})
	}
	return tr
}

func knownField(f string) bool {
	for _, v := range fields {
		if f == v {
			return true
		}
	}
	return false
}
func score(s *summary, tr trace, g goldCase) {
	by := map[string]intake.Decision{}
	for _, d := range tr.Decisions {
		by[d.Field] = d
	}
	labelled, correct, populated := true, tr.Text.TextLayer && tr.Error == "", true
	for _, field := range []string{"number", "revision", "title", "date", "kind", "discipline"} {
		l, ok := g.Fields[field]
		labelled = labelled && ok
		correct = correct && ok && matchesGold(g, field, l, by[field].Value)
		populated = populated && by[field].Value != ""
	}
	if labelled {
		s.FullyLabelledDocuments++
	}
	if labelled && correct {
		s.FullyCorrectDocuments++
		if populated {
			s.FullyPopulatedCorrectDocuments++
		}
	}
	chosen := map[string]intake.Choice{}
	for _, c := range tr.Choices {
		chosen[c.Field] = c
	}
	for field, l := range g.Fields {
		keys := []string{field, tr.Entry.Corpus + "/" + field, tr.Entry.Split + "/" + field}
		if d, ok := g.Fields[intake.FieldDiscipline]; ok && d.Value != "" {
			keys = append(keys, "discipline:"+d.Value+"/"+field)
		}
		for _, key := range keys {
			c := s.Metrics[key]
			c.Scored++
			if l.Value == "" {
				c.ExpectedAbsent++
				if d := by[field]; d.Value == "" && tr.Text.TextLayer && tr.Error == "" {
					c.CorrectAbsent++
				}
			}
			if raw, ok := chosen[field]; ok {
				c.RawJevScored++
				if matchesGold(g, field, l, raw.Value) {
					c.RawJevCorrect++
				}
			}
			d := by[field]
			if matchesGold(g, field, l, d.Value) && tr.Text.TextLayer {
				c.Correct++
			} else if d.Value == "" {
				c.Blank++
			} else {
				c.Wrong++
				if d.Band == intake.BandGreen {
					c.FalseGreen++
				}
			}
			if tr.Candidates != nil && l.Value != "" && (field == "number" || field == "revision" || field == "title" || field == "date") {
				found := false
				for _, v := range tr.Candidates {
					if v.Field == field && matchesGold(g, field, l, v.Display) {
						found = true
					}
				}
				if !found {
					c.CandidateMissing++
				}
			}
			s.Metrics[key] = c
		}
	}
}

func matchesGold(g goldCase, field string, primary label, value string) bool {
	if eval.SameValue(field, primary.Value, value) {
		return true
	}
	for _, alternative := range g.Alternatives[field] {
		if eval.SameValue(field, alternative.Value, value) {
			return true
		}
	}
	return false
}

// The private review queue points back to immutable traces, keeping unknown
// labels separate from errors rather than treating missing truth as success.
func reviewFailures(tr trace, g goldCase, labelled bool) []reviewItem {
	item := reviewItem{SHA256: tr.Entry.SHA256, Corpus: tr.Entry.Corpus}
	if !labelled {
		item.Reason = "unlabelled"
		return []reviewItem{item}
	}
	if g.NotFiled != "" {
		if tr.Error == g.NotFiled {
			return nil
		}
		item.Reason = "unexpected_filing_status"
		return []reviewItem{item}
	}
	by := map[string]intake.Decision{}
	for _, d := range tr.Decisions {
		by[d.Field] = d
	}
	var out []reviewItem
	for _, field := range fields {
		want, ok := g.Fields[field]
		if !ok {
			continue
		}
		d := by[field]
		if tr.Text.TextLayer && matchesGold(g, field, want, d.Value) {
			continue
		}
		r := item
		r.Field, r.Expected, r.Actual = field, &want, &d
		r.Reason = "wrong_value"
		if d.Value == "" {
			r.Reason = "blank"
		} else if d.Band == intake.BandGreen {
			r.Reason = "wrong_green"
		}
		out = append(out, r)
	}
	return out
}
func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func jsonHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}
