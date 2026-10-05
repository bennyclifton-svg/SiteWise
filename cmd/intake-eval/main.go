// Command intake-eval measures filing accuracy against the answer keys and
// fits the green/amber cut-offs on the calibration split.
//
//	go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -live
//	go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay [-fit] [-accept]
//
// -live calls System One and records every exact request and response to the
// private directory. -replay serves those recordings and never touches the
// network; a request without a recording fails the run. A mock or replay
// proves behaviour, not the provider's latency: cmd/intake-bench measures that.
//
// It exits nonzero on a missing corpus or key, a missing recording, too few
// samples, a field regression, an incorrect automatic supersession, or no
// accepted baseline.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type options struct {
	manifest   string
	live       bool
	replay     bool
	check      bool
	rulesOnly  bool
	fit        bool
	accept     bool
	data       string
	recordings string
	out        string
	corpusRoot string
	keysRoot   string
	deadline   time.Duration
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("intake-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.manifest, "manifest", "data/eval/intake/manifest.json", "evaluation manifest")
	fs.BoolVar(&o.live, "live", false, "call System One and record every exchange")
	fs.BoolVar(&o.replay, "replay", false, "serve recorded exchanges only")
	fs.BoolVar(&o.check, "check", false, "verify the corpus and keys and print label counts per split; no Jev call")
	fs.BoolVar(&o.rulesOnly, "rules-only", false, "file with deterministic rules only: every Jev question goes grey; diagnostic, not a gate")
	fs.BoolVar(&o.fit, "fit", false, "write cut-offs fitted on the calibration split to <data>/thresholds.json")
	fs.BoolVar(&o.accept, "accept", false, "accept this held-out result as the baseline after review")
	fs.StringVar(&o.data, "data", "data/intake", "intake vocabulary and thresholds directory")
	fs.StringVar(&o.recordings, "recordings", "", "recording file (default <private_dir>/recordings.jsonl)")
	fs.StringVar(&o.out, "out", "", "report file (default data/eval/intake/results/<mode>-latest.json)")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the manifest corpus_root")
	fs.StringVar(&o.keysRoot, "keys-root", "", "override the manifest keys_root")
	fs.DurationVar(&o.deadline, "deadline", 10*time.Second, "per-call deadline while recording; accuracy is not truncated by the interactive budget")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if n := btoi(o.live) + btoi(o.replay) + btoi(o.check) + btoi(o.rulesOnly); n != 1 {
		fmt.Fprintln(stderr, "choose exactly one of -live, -replay, -check or -rules-only")
		return 2
	}
	if err := evaluate(context.Background(), o, getenv, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, "intake-eval:", err)
		return 1
	}
	return 0
}

func evaluate(ctx context.Context, o options, getenv func(string) string, stdout, stderr io.Writer) error {
	m, manifestBody, err := eval.LoadManifest(o.manifest)
	if err != nil {
		return err
	}
	if o.corpusRoot != "" {
		m.CorpusRoot = o.corpusRoot
	}
	if o.keysRoot != "" {
		m.KeysRoot = o.keysRoot
	}
	if o.recordings == "" {
		o.recordings = filepath.Join(m.PrivateDir, "recordings.jsonl")
	}
	mode := "replay"
	switch {
	case o.live:
		mode = "live"
	case o.rulesOnly:
		mode = "rules-only"
	}
	if o.out == "" {
		o.out = filepath.Join("data", "eval", "intake", "results", mode+"-latest.json")
	}

	set, err := eval.BuildCases(m)
	if err != nil {
		return fmt.Errorf("corpus or answer keys failed verification:\n%w", err)
	}
	fmt.Fprintf(stdout, "verified %d cases (%d calibration, %d held-out); excluded %v\n",
		len(set.Cases), count(set, eval.SplitCalibration), count(set, eval.SplitHeldout), set.Excluded)
	if o.check {
		printLabels(stdout, set)
		return nil
	}

	cat, err := intake.LoadCatalog(o.data)
	if err != nil {
		return err
	}
	thresholds, err := intake.LoadThresholds(o.data)
	if err != nil {
		return err
	}
	if err := identity.Warm(ctx); err != nil {
		return err
	}

	if o.rulesOnly {
		return rulesOnly(ctx, o, m, manifestBody, set, cat, thresholds, stdout)
	}
	if o.live {
		if err := record(ctx, o, m, set, cat, thresholds, getenv, stdout); err != nil {
			return err
		}
	}
	records, err := loadRecordings(o.recordings)
	if err != nil {
		return err
	}
	result, err := replay(ctx, set, cat, thresholds, records)
	if err != nil {
		return err
	}

	fits := eval.Fit(result.Answers(eval.SplitCalibration), m.Fit)
	if o.fit {
		thresholds, err = writeThresholds(o, m, manifestBody, set, records, fits)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s from %d calibration answers\n", filepath.Join(o.data, "thresholds.json"), len(result.Answers(eval.SplitCalibration)))
		if result, err = replay(ctx, set, cat, thresholds, records); err != nil {
			return err
		}
	}

	report := eval.NewReport(m, set, result, mode)
	report.Fit = fits
	report.SetRecordedLatency(records)
	report.Inputs["manifest"] = sha(manifestBody)
	if body, err := os.ReadFile(filepath.Join(o.data, "thresholds.json")); err == nil {
		report.Inputs["thresholds"] = sha(body)
	}
	if body, err := os.ReadFile(o.recordings); err == nil {
		report.Inputs["recordings"] = sha(body)
	}

	base, err := loadBaseline(m.Baseline)
	if err != nil {
		return err
	}
	failures := report.Gate(m, base)
	if o.accept {
		if blocking := withoutBaseline(failures); len(blocking) > 0 {
			return fmt.Errorf("cannot accept a baseline while other gates fail:\n  %s", strings.Join(blocking, "\n  "))
		}
		accepted := report.Baseline()
		if err := writeJSON(m.Baseline, accepted); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "accepted baseline %s\n", m.Baseline)
		failures = report.Gate(m, &accepted)
	}
	report.Verdict = eval.GateResult{Passed: len(failures) == 0, Failures: failures}
	if report.Verdict.Failures == nil {
		report.Verdict.Failures = []string{}
	}
	if err := writeJSON(o.out, report); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(m.PrivateDir, "cases-"+mode+".json"), result.Cases); err != nil {
		return err
	}
	printSummary(stdout, report)
	if len(failures) > 0 {
		return fmt.Errorf("gate failed (%s):\n  %s", o.out, strings.Join(failures, "\n  "))
	}
	fmt.Fprintf(stdout, "gate passed (%s)\n", o.out)
	return nil
}

// record files every case against System One, writing each exact exchange.
// Thresholds do not change what is asked, so one live pass serves any fit.
func record(ctx context.Context, o options, m eval.Manifest, set eval.CaseSet, cat intake.Catalog, thresholds intake.Thresholds, getenv func(string) string, stdout io.Writer) error {
	key := strings.TrimSpace(getenv("SITEWISE_JEV_API_KEY"))
	if key == "" {
		return errors.New("SITEWISE_JEV_API_KEY is required for -live")
	}
	if err := os.MkdirAll(filepath.Dir(o.recordings), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(o.recordings, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	rec := eval.NewRecorder(jev.NewTransport(), f)
	client, err := jev.New(jev.Options{
		APIKey:    key,
		Model:     config.PinnedJevModel,
		Transport: rec,
		Logger:    slog.New(slog.DiscardHandler),
		// Each case is measured on its own; an open circuit would turn later
		// cases into errors that say nothing about their answers.
		BreakerThreshold: 1 << 20,
	})
	if err != nil {
		return err
	}
	if err := client.Warm(ctx); err != nil {
		return fmt.Errorf("jev warm-up: %w", err)
	}
	ev := eval.Evaluator{Catalog: cat, Thresholds: thresholds, Asker: client, Labeler: rec, Deadline: o.deadline}
	result, err := ev.Run(ctx, set)
	if err != nil {
		return err
	}
	if err := rec.Err(); err != nil {
		return fmt.Errorf("recording: %w", err)
	}
	fmt.Fprintf(stdout, "recorded %d live calls (%d errors) to %s\n", result.Calls, result.Errors, o.recordings)
	return f.Sync()
}

// rulesOnly measures what code decides with Jev unavailable: rule values are
// kept and every open question goes grey, as on the degraded path. It shows
// rule accuracy and false confident rule actions on the real corpus; it is
// not the release gate.
func rulesOnly(ctx context.Context, o options, m eval.Manifest, manifestBody []byte, set eval.CaseSet, cat intake.Catalog, thresholds intake.Thresholds, stdout io.Writer) error {
	ev := eval.Evaluator{Catalog: cat, Thresholds: thresholds, Asker: unavailable{}}
	result, err := ev.Run(ctx, set)
	if err != nil {
		return err
	}
	report := eval.NewReport(m, set, result, "rules-only")
	report.Inputs["manifest"] = sha(manifestBody)
	if body, err := os.ReadFile(filepath.Join(o.data, "thresholds.json")); err == nil {
		report.Inputs["thresholds"] = sha(body)
	}
	report.Verdict = eval.GateResult{Passed: false, Failures: []string{"rules-only is diagnostic; the gate needs -replay of a live recording"}}
	if err := writeJSON(o.out, report); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(m.PrivateDir, "cases-rules-only.json"), result.Cases); err != nil {
		return err
	}
	printSummary(stdout, report)
	fmt.Fprintf(stdout, "wrote %s (diagnostic, not a gate)\n", o.out)
	return nil
}

// unavailable is a Jev that never answers.
type unavailable struct{}

func (unavailable) Ask(context.Context, jev.Call) (jev.Result, error) {
	return jev.Result{}, errors.New("rules-only: Jev not called")
}

func replay(ctx context.Context, set eval.CaseSet, cat intake.Catalog, thresholds intake.Thresholds, records []eval.Record) (*eval.RunResult, error) {
	rep := eval.NewReplayer(records, false)
	client, err := jev.New(jev.Options{
		APIKey:           "replay",
		Model:            config.PinnedJevModel,
		Transport:        rep,
		Logger:           slog.New(slog.DiscardHandler),
		BreakerThreshold: 1 << 20,
	})
	if err != nil {
		return nil, err
	}
	ev := eval.Evaluator{Catalog: cat, Thresholds: thresholds, Asker: client, Misses: rep.Misses, Deadline: time.Minute}
	return ev.Run(ctx, set)
}

func loadRecordings(file string) ([]eval.Record, error) {
	f, err := os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no recordings at %s: run with -live and SITEWISE_JEV_API_KEY first", file)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return eval.ReadRecordings(f)
}

func writeThresholds(o options, m eval.Manifest, manifestBody []byte, set eval.CaseSet, records []eval.Record, fits []eval.FitResult) (intake.Thresholds, error) {
	var body bytes.Buffer
	for _, r := range records {
		body.WriteString(r.RequestSHA256)
	}
	meta := map[string]any{
		"fitted_at":         time.Now().UTC(),
		"split":             eval.SplitCalibration,
		"manifest_sha256":   sha(manifestBody),
		"recordings_digest": sha(body.Bytes()),
		"calibration_cases": count(set, eval.SplitCalibration),
		"policy":            m.Fit,
		"results":           fits,
	}
	t := eval.ThresholdsFrom(fits, "Fitted by cmd/intake-eval -fit on the calibration split only, per question and power-of-two option shape. A question or shape with no entry is unknown and disables automatic Jev action. No probability from the TypeSafe docs is a default. Labels are unreviewed register/transmittal transcriptions; see fit.results for sample sizes.", meta)
	if err := writeJSON(filepath.Join(o.data, "thresholds.json"), t); err != nil {
		return intake.Thresholds{}, err
	}
	return intake.LoadThresholds(o.data)
}

func loadBaseline(file string) (*eval.Baseline, error) {
	body, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var b eval.Baseline
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}
	return &b, nil
}

func withoutBaseline(failures []string) []string {
	var out []string
	for _, f := range failures {
		if strings.Contains(f, "baseline") {
			continue
		}
		out = append(out, f)
	}
	return out
}

func printSummary(w io.Writer, r eval.Report) {
	for _, split := range []string{eval.SplitCalibration, eval.SplitHeldout} {
		for _, tier := range []string{eval.TierAuthoritative, eval.TierDiagnostic} {
			metrics := r.Metrics[split][tier]
			if len(metrics) == 0 {
				continue
			}
			fmt.Fprintf(w, "\n%s / %s labels\n", split, tier)
			tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "field\tscored\tapplied\tcorrect\taccuracy [90% CI]\tcoverage\tamber\tgrey\tfalse confident")
			fields := make([]string, 0, len(metrics))
			for f := range metrics {
				fields = append(fields, f)
			}
			sort.Strings(fields)
			for _, f := range fields {
				m := metrics[f]
				fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%.2f [%.2f, %.2f]\t%.2f\t%d\t%d\t%d\n",
					f, m.Scored, m.Applied, m.Correct, m.Accuracy, m.AccuracyLow, m.AccuracyHigh, m.Coverage, m.Amber, m.Grey, m.FalseConfident)
			}
			tw.Flush()
		}
	}
	fmt.Fprintf(w, "\njev calls %d, errors %d, grey filings %d, replay misses %d; recorded p50 %dus p90 %dus, %d over the %dus interactive deadline\n",
		r.Jev.Calls, r.Jev.Errors, r.Jev.GreyFilings, r.Jev.ReplayMisses, r.Jev.RecordedP50US, r.Jev.RecordedP90US, r.Jev.OverDeadline, r.Jev.InteractiveDeadUS)
	for split, s := range r.Supersession {
		fmt.Fprintf(w, "supersession %s: asked %d, scored %d, automatic %d, incorrect %d, unverified %d\n", split, s.Asked, s.Scored, s.Automatic, s.Incorrect, s.Unverified)
	}
	fmt.Fprintf(w, "not filed %v\n", r.Cases.NotFiled)
	fmt.Fprintf(w, "drawing sets %v, sheet calls %d\n", r.Cases.Sheets, r.Cases.SheetCalls)
}

// printLabels shows how many labels each split holds per field and tier, so
// sample sizes are known before a paid live run.
func printLabels(w io.Writer, set eval.CaseSet) {
	type key struct{ split, tier, field string }
	counts := map[key]int{}
	families := map[string]map[string]bool{}
	for _, c := range set.Cases {
		if families[c.Split] == nil {
			families[c.Split] = map[string]bool{}
		}
		families[c.Split][c.Family] = true
		for field, l := range c.Labels {
			tier := eval.TierDiagnostic
			if l.Authoritative {
				tier = eval.TierAuthoritative
			}
			counts[key{c.Split, tier, field}]++
		}
	}
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "split	families	tier	kind	discipline	number	revision	title	date")
	for _, split := range []string{eval.SplitCalibration, eval.SplitHeldout} {
		for _, tier := range []string{eval.TierAuthoritative, eval.TierDiagnostic} {
			fmt.Fprintf(tw, "%s	%d	%s", split, len(families[split]), tier)
			for _, f := range []string{intake.FieldKind, intake.FieldDiscipline, intake.FieldNumber, intake.FieldRevision, intake.FieldTitle, intake.FieldDate} {
				fmt.Fprintf(tw, "	%d", counts[key{split, tier, f}])
			}
			fmt.Fprintln(tw)
		}
	}
	tw.Flush()
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func count(set eval.CaseSet, split string) int {
	n := 0
	for _, c := range set.Cases {
		if c.Split == split {
			n++
		}
	}
	return n
}

func sha(b []byte) string {
	s := eval.SHA256Hex(b)
	return s
}

func writeJSON(file string, v any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}
