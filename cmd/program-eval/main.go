// Command program-eval calibrates Reading A (plan
// docs/plans/2026-10-18-program.md 10.2 and 10.5): it reads every program
// file of a hand-labelled gold set the way the server does (harvest, rules,
// one program_row fan-out) and, with -fit, writes data/program/thresholds.json
// in the shape of data/proposal/thresholds.json, recording n for every
// cut-off.
//
// Gold set format (data/eval/program/private/answer-key.json; private
// because programs carry real projects):
//
//	{
//	  "question_version": "program-row-1",
//	  "corpus_root": "D:/corpora",          // files are read relative to it
//	  "cases": [{
//	    "id": "hale-01",
//	    "file": "Hale/Programme Rev C.xlsx",
//	    "targets": [{"id": "t1", "label": "Slab", "kind": "activity",
//	                 "stage_label": "Construction", "tier": 1}],
//	    "expected": {"t1": "Pour ground slab", "t2": null}  // a row label, or null for none
//	  }]
//	}
//
// No gold set exists yet. Until one does, data/program/thresholds.json holds
// no cut-off and every Jev answer stays blank ("program_row: not
// calibrated" on the dev status page). Replay is the default; -live calls
// System One and records every exchange (nightly or owner-run only, never CI).
package main

import (
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
	"strconv"
	"strings"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/programread"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type options struct {
	key, recording, data, corpusRoot string
	live, fit                        bool
}

// policy is the proposal read's: the accuracy a cut-off must show on the
// calibration split (Wilson lower bounds). Proposed; the owner has not
// signed it off for programs.
var policy = eval.FitPolicy{Z: 1.645, MinSamples: 20, MinBandSamples: 10, GreenLower: 0.95, AmberLower: 0.70, MaxErrors: -1}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("program-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.key, "key", "data/eval/program/private/answer-key.json", "gold set answer key")
	fs.StringVar(&o.recording, "recording", "data/eval/program/private/recording.jsonl", "Jev recording")
	fs.StringVar(&o.data, "data", "data/program", "thresholds directory")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the answer key corpus_root")
	fs.BoolVar(&o.live, "live", false, "call System One and record every exchange")
	fs.BoolVar(&o.fit, "fit", false, "write cut-offs fitted on the calibration split")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := evaluate(context.Background(), o, getenv, stdout); err != nil {
		fmt.Fprintln(stderr, "program-eval:", err)
		return 1
	}
	return 0
}

type keyTarget struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	StageLabel string `json:"stage_label"`
	Tier       int    `json:"tier"`
}

type keyCase struct {
	ID       string             `json:"id"`
	File     string             `json:"file"`
	Targets  []keyTarget        `json:"targets"`
	Expected map[string]*string `json:"expected"`
}

type answerKey struct {
	QuestionVersion string    `json:"question_version"`
	CorpusRoot      string    `json:"corpus_root"`
	Cases           []keyCase `json:"cases"`
}

func evaluate(ctx context.Context, o options, getenv func(string) string, stdout io.Writer) error {
	body, err := os.ReadFile(o.key)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no gold set at %s: see the format at the top of cmd/program-eval/main.go; until one exists data/program/thresholds.json stays without cut-offs", o.key)
	}
	if err != nil {
		return err
	}
	var key answerKey
	if err := json.Unmarshal(body, &key); err != nil {
		return err
	}
	if key.QuestionVersion != programread.QuestionVersion {
		return fmt.Errorf("gold set is for %q, this build asks %q", key.QuestionVersion, programread.QuestionVersion)
	}
	root := key.CorpusRoot
	if o.corpusRoot != "" {
		root = o.corpusRoot
	}
	ask, finish, err := client(ctx, o, getenv)
	if err != nil {
		return err
	}
	var answers []eval.Answer
	right, wrong, blank := 0, 0, 0
	for _, c := range key.Cases {
		got, err := evaluateCase(ctx, ask, root, c)
		if err != nil {
			return fmt.Errorf("%s: %w", c.ID, err)
		}
		for _, a := range got {
			if a.Correct {
				right++
			} else {
				wrong++
			}
		}
		if len(got) == 0 {
			blank++
		}
		answers = append(answers, got...)
	}
	if err := finish(); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%d cases, %d answers: %d right, %d wrong; %d cases asked nothing\n", len(key.Cases), len(answers), right, wrong, blank)
	if !o.fit {
		return nil
	}
	return writeThresholds(o.data, answers)
}

// evaluateCase reads one file as the server does and scores each asked
// target's raw answer against the label the owner gave.
func evaluateCase(ctx context.Context, ask programread.Asker, root string, c keyCase) ([]eval.Answer, error) {
	path := filepath.Join(root, filepath.FromSlash(c.File))
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	text, err := identity.Extract(ctx, format, f, info.Size(), programread.ExtractLimits())
	if err != nil {
		return nil, err
	}
	h := programread.HarvestText(filepath.Base(path), text, 5)
	targets := make([]programread.Target, len(c.Targets))
	for i, t := range c.Targets {
		targets[i] = programread.Target{ID: t.ID, Label: t.Label, Kind: t.Kind, StageLabel: t.StageLabel, Tier: t.Tier}
	}
	d := programread.NewDraft(h, targets)
	call, ok := d.Call()
	if !ok {
		return nil, nil
	}
	res, err := ask.Ask(ctx, call)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	for _, r := range h.Rows {
		labels[r.ID] = r.Label
	}
	split := eval.SplitOf(c.ID, programread.QuestionVersion, 60)
	var out []eval.Answer
	for _, id := range d.Asked() {
		row, conf, answered := d.Raw(res, id)
		if !answered || conf == nil {
			continue
		}
		want, scored := c.Expected[id]
		if !scored {
			continue
		}
		correct := row == "" && want == nil || row != "" && want != nil && labels[row] == *want
		out = append(out, eval.Answer{Case: split + "/" + c.ID, Field: programread.Family, Options: d.Options(id), Confidence: *conf, Correct: correct})
	}
	return out, nil
}

func client(ctx context.Context, o options, getenv func(string) string) (programread.Asker, func() error, error) {
	if o.live {
		k := strings.TrimSpace(getenv("SITEWISE_JEV_API_KEY"))
		if k == "" {
			return nil, nil, errors.New("SITEWISE_JEV_API_KEY is required for -live")
		}
		if err := os.MkdirAll(filepath.Dir(o.recording), 0o700); err != nil {
			return nil, nil, err
		}
		f, err := os.OpenFile(o.recording, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, nil, err
		}
		rec := eval.NewRecorder(jev.NewTransport(), f)
		// Pinned model, never jev-latest (https://docs.typesafe.ai/api).
		c, err := jev.New(jev.Options{APIKey: k, Model: config.PinnedJevModel, Transport: rec, Logger: slog.New(slog.DiscardHandler), BreakerThreshold: 1 << 20})
		if err != nil {
			return nil, nil, err
		}
		if err := c.Warm(ctx); err != nil {
			return nil, nil, fmt.Errorf("jev warm-up: %w", err)
		}
		return c, func() error {
			if err := rec.Err(); err != nil {
				return err
			}
			return f.Close()
		}, nil
	}
	f, err := os.Open(o.recording)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("no recording at %s: run with -live and SITEWISE_JEV_API_KEY first", o.recording)
	}
	if err != nil {
		return nil, nil, err
	}
	records, err := eval.ReadRecordings(f)
	f.Close()
	if err != nil {
		return nil, nil, err
	}
	c, err := jev.New(jev.Options{APIKey: "replay", Model: config.PinnedJevModel, Transport: eval.NewReplayer(records, false), Logger: slog.New(slog.DiscardHandler), BreakerThreshold: 1 << 20})
	if err != nil {
		return nil, nil, err
	}
	return c, func() error { return nil }, nil
}

// shapeOf buckets option counts as the proposal eval does: confidence is
// never compared between a two-option and a many-option question.
// https://docs.typesafe.ai/confidence
func shapeOf(options int) eval.Shape {
	switch {
	case options < 2:
		return eval.Shape{}
	case options <= 3:
		return eval.Shape{Min: 2, Max: 3}
	case options <= 12:
		return eval.Shape{Min: 4, Max: 12}
	default:
		return eval.Shape{Min: 13, Max: 255}
	}
}

// writeThresholds fits on the calibration split only and writes every
// bucket with its n, a nil cut-off where the evidence is too thin.
func writeThresholds(dir string, answers []eval.Answer) error {
	groups := map[string][]eval.Answer{}
	n := 0
	for _, a := range answers {
		if !strings.HasPrefix(a.Case, eval.SplitCalibration+"/") {
			continue
		}
		s := shapeOf(a.Options)
		if s == (eval.Shape{}) {
			continue
		}
		k := a.Field + "@" + strconv.Itoa(s.Min)
		a.Field, a.Options = k, 2 // the whole bucket is one shape for Fit
		groups[k] = append(groups[k], a)
		n++
	}
	th := programread.Thresholds{QuestionVersion: programread.QuestionVersion, Questions: map[string][]programread.Threshold{}}
	th.Calibration = fmt.Sprintf("cmd/program-eval -fit, %d calibration answers, %s", n, time.Now().UTC().Format("2006-01-02"))
	th.Policy, _ = json.Marshal(policy)
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		family, lo, _ := strings.Cut(k, "@")
		min, _ := strconv.Atoi(lo)
		s := shapeOf(min)
		strict := policy
		strict.MaxErrors = 0
		sf := eval.Fit(groups[k], map[string]eval.FitPolicy{k: strict})
		for i, f := range eval.Fit(groups[k], map[string]eval.FitPolicy{k: policy}) {
			lo, hi := s.Min, s.Max
			t := programread.Threshold{Options: &lo, MaxOptions: &hi, Amber: f.Amber, N: f.N, Reason: f.Reason}
			if i < len(sf) && sf[i].Green != nil {
				t.Green = sf[i].Green
			}
			if t.Green != nil && (t.Amber == nil || *t.Amber > *t.Green) {
				t.Amber = t.Green
			}
			th.Questions[family] = append(th.Questions[family], t)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(th, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "thresholds.json"), append(out, '\n'), 0o644)
}
