// Command invoice-eval measures invoice reading against the answer key of
// real invoices and fits the green/amber cut-offs Jev's answers need.
//
//	go run ./cmd/invoice-eval -live          # call System One, record every exchange
//	go run ./cmd/invoice-eval [-fit]         # replay the recording; -fit writes data/invoice/thresholds.json
//
// The answer key, the corpus paths and the recording hold private invoices
// and stay in data/eval/invoice/private/ (git-ignored). Every question with
// options is asked on every case so rule-settled questions are calibrated
// too; scoring applies the production order (rule, then Jev, then code).
//
// It exits nonzero on a missing or changed corpus file, a replay miss, or any
// wrong value marked green, by a rule or by Jev.
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
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/invoices"
	"sitewise/internal/jev"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type options struct {
	key, recording, data, out, corpusRoot string
	live, fit, show                       bool
	deadline                              time.Duration
}

// policy is the accuracy a cut-off must show on the calibration split
// (Wilson lower bounds, 95 % one-sided). Proposed by the agent on
// 16 October 2026; the owner has not signed it off.
var policy = eval.FitPolicy{Z: 1.645, MinSamples: 20, MinBandSamples: 10, GreenLower: 0.95, AmberLower: 0.70, MaxErrors: -1}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("invoice-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.key, "key", "data/eval/invoice/private/answer-key.json", "answer key")
	fs.StringVar(&o.recording, "recording", "data/eval/invoice/private/recording.jsonl", "Jev recording")
	fs.StringVar(&o.data, "data", "data/invoice", "thresholds directory")
	fs.StringVar(&o.out, "out", "data/eval/invoice/results/latest.json", "aggregate report (no case names)")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the answer key corpus_root")
	fs.BoolVar(&o.live, "live", false, "call System One and record every exchange")
	fs.BoolVar(&o.fit, "fit", false, "write cut-offs fitted on the calibration split")
	fs.BoolVar(&o.show, "show", false, "print each wrong or blank field (private values; terminal only)")
	fs.DurationVar(&o.deadline, "deadline", 10*time.Second, "per-call deadline; accuracy is not truncated by the interactive budget")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := evaluate(context.Background(), o, getenv, stdout); err != nil {
		fmt.Fprintln(stderr, "invoice-eval:", err)
		return 1
	}
	return 0
}

type answerKey struct {
	Version         int                       `json:"version"`
	Reviewed        bool                      `json:"reviewed"`
	QuestionVersion string                    `json:"question_version"`
	CorpusRoot      string                    `json:"corpus_root"`
	Contexts        map[string]projectContext `json:"contexts"`
	Cases           []keyCase                 `json:"cases"`
}

type projectContext struct {
	Packages [][4]string `json:"packages"` // id, title, kind, abn
	Lines    [][4]string `json:"lines"`    // id, label, section, package id
}

type keyCase struct {
	ID       string   `json:"id"`
	Path     string   `json:"path"`
	SHA256   string   `json:"sha256"`
	Project  string   `json:"project"`
	Kind     []string `json:"kind"`
	Number   *string  `json:"number"`
	Date     *string  `json:"date"`
	ABN      *string  `json:"issuer_abn"`
	Total    *string  `json:"total"`
	GST      *string  `json:"gst"`
	Supplier *string  `json:"supplier"`
	Line     *string  `json:"line"`
}

func evaluate(ctx context.Context, o options, getenv func(string) string, stdout io.Writer) error {
	body, err := os.ReadFile(o.key)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no answer key at %s", o.key)
	}
	if err != nil {
		return err
	}
	var key answerKey
	if err := json.Unmarshal(body, &key); err != nil {
		return err
	}
	if key.QuestionVersion != invoices.QuestionVersion {
		return fmt.Errorf("answer key is for %q, this build asks %q", key.QuestionVersion, invoices.QuestionVersion)
	}
	root := key.CorpusRoot
	if o.corpusRoot != "" {
		root = o.corpusRoot
	}
	th, err := invoices.LoadThresholds(o.data)
	if err != nil {
		return err
	}
	asker, finish, misses, err := client(ctx, o, getenv)
	if err != nil {
		return err
	}
	var answers []eval.Answer
	var results []caseResult
	for _, c := range key.Cases {
		r, raw, err := evaluateCase(ctx, o, root, key.Contexts[c.Project], c, asker, th)
		if err != nil {
			return fmt.Errorf("case %s: %w", c.ID, err)
		}
		results = append(results, r)
		answers = append(answers, raw...)
	}
	if err := finish(); err != nil {
		return err
	}
	if n := misses(); n > 0 {
		return fmt.Errorf("%d requests had no recording: re-record with -live", n)
	}
	rep := summarize(results, answers)
	if o.show {
		for _, r := range results {
			for _, f := range r.Fields {
				if f.Outcome != "correct" {
					fmt.Fprintf(stdout, "%s %-10s %-7s %s/%s got=%q want=%q conf=%s\n", r.ID, f.Field, f.Outcome, f.By, f.Band, f.Got, f.Want, confText(f.Conf))
				}
			}
		}
	}
	printReport(stdout, rep)
	if o.fit {
		if err := writeThresholds(o, answers); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s from %d calibration answers; re-run without -fit to score with it\n", filepath.Join(o.data, "thresholds.json"), countSplit(answers, eval.SplitCalibration))
	}
	if err := os.MkdirAll(filepath.Dir(o.out), 0o755); err != nil {
		return err
	}
	out, _ := json.MarshalIndent(rep, "", " ")
	if err := os.WriteFile(o.out, append(out, '\n'), 0o644); err != nil {
		return err
	}
	if rep.FalseGreen > 0 {
		return fmt.Errorf("%d wrong values were marked green", rep.FalseGreen)
	}
	return nil
}

func client(ctx context.Context, o options, getenv func(string) string) (invoices.Asker, func() error, func() int, error) {
	if o.live {
		k := strings.TrimSpace(getenv("SITEWISE_JEV_API_KEY"))
		if k == "" {
			return nil, nil, nil, errors.New("SITEWISE_JEV_API_KEY is required for -live")
		}
		if err := os.MkdirAll(filepath.Dir(o.recording), 0o700); err != nil {
			return nil, nil, nil, err
		}
		f, err := os.OpenFile(o.recording, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, nil, nil, err
		}
		rec := eval.NewRecorder(jev.NewTransport(), f)
		c, err := jev.New(jev.Options{APIKey: k, Model: config.PinnedJevModel, Transport: rec, Logger: slog.New(slog.DiscardHandler), BreakerThreshold: 1 << 20})
		if err != nil {
			return nil, nil, nil, err
		}
		if err := c.Warm(ctx); err != nil {
			return nil, nil, nil, fmt.Errorf("jev warm-up: %w", err)
		}
		return c, func() error {
			if err := rec.Err(); err != nil {
				return err
			}
			return f.Close()
		}, func() int { return 0 }, nil
	}
	f, err := os.Open(o.recording)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil, fmt.Errorf("no recording at %s: run with -live and SITEWISE_JEV_API_KEY first", o.recording)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	records, err := eval.ReadRecordings(f)
	f.Close()
	if err != nil {
		return nil, nil, nil, err
	}
	rep := eval.NewReplayer(records, false)
	c, err := jev.New(jev.Options{APIKey: "replay", Model: config.PinnedJevModel, Transport: rep, Logger: slog.New(slog.DiscardHandler), BreakerThreshold: 1 << 20})
	if err != nil {
		return nil, nil, nil, err
	}
	return c, func() error { return nil }, rep.Misses, nil
}

// field outcome for one case
type fieldResult struct {
	Conf    *float64
	Got     string
	Want    []string
	Field   string
	Outcome string // correct, wrong, blank
	Band    string
	By      string
}

type caseResult struct {
	ID     string
	Split  string
	Fields []fieldResult
}

var scored = []string{invoices.FieldKind, invoices.FieldNumber, invoices.FieldDate, invoices.FieldABN, invoices.FieldTotal, invoices.FieldGST, invoices.FieldSupplier, invoices.FieldLine}

func evaluateCase(ctx context.Context, o options, root string, pc projectContext, c keyCase, ask invoices.Asker, th invoices.Thresholds) (caseResult, []eval.Answer, error) {
	path := filepath.Join(root, filepath.FromSlash(c.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		return caseResult{}, nil, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != c.SHA256 {
		return caseResult{}, nil, errors.New("file changed since it was labelled")
	}
	text, err := identity.Extract(ctx, "pdf", strings.NewReader(string(b)), int64(len(b)), invoices.ExtractLimits())
	if err != nil {
		return caseResult{}, nil, err
	}
	res := caseResult{ID: c.ID, Split: eval.SplitOf(family(c), invoices.QuestionVersion, 60)}
	ev := invoices.Harvest(filepath.Base(c.Path), text)
	if len(c.Kind) == 1 && c.Kind[0] == "unreadable" {
		out := "correct"
		if text.TextLayer && ev.Readable {
			out = "wrong"
		}
		res.Fields = append(res.Fields, fieldResult{Field: "readable", Outcome: out, Band: invoices.BandGreen, By: invoices.DecidedByRule})
		return res, nil, nil
	}
	if !text.TextLayer || !ev.Readable {
		res.Fields = append(res.Fields, fieldResult{Field: "readable", Outcome: "wrong", Band: invoices.BandGreen, By: invoices.DecidedByRule})
		return res, nil, nil
	}
	d := invoices.NewDraft(ev, contextOf(pc), nil)
	var raw []eval.Answer
	call, asked := d.CallAll()
	var result jev.Result
	var askErr error
	if asked {
		call.Deadline = o.deadline
		cctx, cancel := context.WithTimeout(ctx, o.deadline+time.Second)
		result, askErr = ask.Ask(cctx, call)
		cancel()
		if askErr != nil && errors.Is(askErr, eval.ErrReplayMiss) {
			return caseResult{}, nil, askErr
		}
		for _, f := range scored {
			want, ok := expected(c, f)
			if !ok {
				continue
			}
			v, conf, got := d.Raw(result, f)
			if !got || conf == nil {
				continue
			}
			raw = append(raw, eval.Answer{Case: c.ID, Field: f, Options: d.Options(f), Confidence: *conf, Correct: matches(f, want, v)})
		}
	}
	d.Apply(result, askErr, th)
	r := d.Finish(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC))
	by := map[string]invoices.Decision{}
	for _, dec := range r.Decisions {
		by[dec.Field] = dec
	}
	for _, f := range scored {
		want, ok := expected(c, f)
		if !ok {
			continue
		}
		dec := by[f]
		out := "blank"
		switch {
		case dec.Value == "" && len(want) == 1 && want[0] == "":
			out = "correct"
		case dec.Value == "":
		case matches(f, want, dec.Value):
			out = "correct"
		default:
			out = "wrong"
		}
		res.Fields = append(res.Fields, fieldResult{Field: f, Outcome: out, Band: dec.Band, By: dec.DecidedBy, Got: dec.Value, Want: want, Conf: dec.Confidence})
	}
	for i := range raw {
		raw[i].Case = res.Split + "/" + raw[i].Case
	}
	return res, raw, nil
}

// family keeps one supplier's invoices in one split.
func family(c keyCase) string {
	if c.ABN != nil && *c.ABN != "" {
		return *c.ABN
	}
	return c.Project + "/" + c.ID
}

func expected(c keyCase, field string) ([]string, bool) {
	invoice := len(c.Kind) == 1 && (c.Kind[0] == "invoice" || c.Kind[0] == "credit_note")
	if field == invoices.FieldKind {
		return c.Kind, true
	}
	if !invoice && !(len(c.Kind) > 1 && c.Kind[0] == "invoice") {
		return nil, false
	}
	var p *string
	switch field {
	case invoices.FieldNumber:
		p = c.Number
	case invoices.FieldDate:
		p = c.Date
	case invoices.FieldABN:
		p = c.ABN
	case invoices.FieldTotal:
		p = c.Total
	case invoices.FieldGST:
		p = c.GST
	case invoices.FieldSupplier:
		p = c.Supplier
	case invoices.FieldLine:
		p = c.Line
	}
	if p == nil {
		return nil, false
	}
	v := *p
	switch field {
	case invoices.FieldTotal, invoices.FieldGST:
		if v != "" {
			n, ok, err := invoices.ParseCents(v)
			if err != nil || !ok {
				return nil, false
			}
			v = strconv.FormatInt(n, 10)
		}
	case invoices.FieldNumber:
		v = invoices.NormalizeNumber(v)
	}
	return []string{v}, true
}

func matches(field string, want []string, got string) bool {
	if field == invoices.FieldNumber {
		got = invoices.NormalizeNumber(got)
	}
	for _, w := range want {
		if w == got {
			return true
		}
	}
	return false
}

func contextOf(pc projectContext) invoices.Context {
	out := invoices.Context{History: map[string][]string{}}
	titles := map[string]string{}
	for _, p := range pc.Packages {
		out.Packages = append(out.Packages, invoices.Package{ID: p[0], Title: p[1], Kind: p[2], ABN: p[3]})
		titles[p[0]] = p[1]
	}
	for _, l := range pc.Lines {
		out.Lines = append(out.Lines, invoices.Line{ID: l[0], Label: l[1], Section: l[2], PackageID: l[3], Category: titles[l[3]]})
	}
	return out
}

type fieldStats struct {
	Field        string `json:"field"`
	Scored       int    `json:"scored"`
	Correct      int    `json:"correct"`
	Wrong        int    `json:"wrong"`
	Blank        int    `json:"blank"`
	RuleCorrect  int    `json:"rule_correct"`
	RuleWrong    int    `json:"rule_wrong"`
	GreenWrong   int    `json:"green_wrong"`
	AmberWrong   int    `json:"amber_wrong"`
	RawAnswers   int    `json:"raw_answers"`
	RawCorrect   int    `json:"raw_correct"`
	HeldOut      int    `json:"held_out_scored"`
	HeldOutRight int    `json:"held_out_correct"`
}

type report struct {
	QuestionVersion string         `json:"question_version"`
	Cases           int            `json:"cases"`
	FalseGreen      int            `json:"false_green"`
	Fields          []*fieldStats  `json:"fields"`
	Policy          eval.FitPolicy `json:"fit_policy"`
}

func summarize(results []caseResult, answers []eval.Answer) report {
	by := map[string]*fieldStats{}
	get := func(f string) *fieldStats {
		if by[f] == nil {
			by[f] = &fieldStats{Field: f}
		}
		return by[f]
	}
	rep := report{QuestionVersion: invoices.QuestionVersion, Cases: len(results), Policy: policy}
	for _, r := range results {
		for _, f := range r.Fields {
			s := get(f.Field)
			s.Scored++
			switch f.Outcome {
			case "correct":
				s.Correct++
				if f.By == invoices.DecidedByRule && f.Band == invoices.BandGreen {
					s.RuleCorrect++
				}
			case "wrong":
				s.Wrong++
				if f.By == invoices.DecidedByRule {
					s.RuleWrong++
				}
				if f.Band == invoices.BandGreen {
					s.GreenWrong++
					rep.FalseGreen++
				}
				if f.Band == invoices.BandAmber {
					s.AmberWrong++
				}
			default:
				s.Blank++
			}
			if r.Split == eval.SplitHeldout {
				s.HeldOut++
				if f.Outcome == "correct" {
					s.HeldOutRight++
				}
			}
		}
	}
	for _, a := range answers {
		s := get(a.Field)
		s.RawAnswers++
		if a.Correct {
			s.RawCorrect++
		}
	}
	for _, f := range append([]string{"readable"}, scored...) {
		if by[f] != nil {
			rep.Fields = append(rep.Fields, by[f])
		}
	}
	return rep
}

func printReport(w io.Writer, rep report) {
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "field\tscored\tcorrect\twrong\tblank\trule ok/wrong\tgreen wrong\tamber wrong\tjev raw ok/n\theld-out ok/n")
	for _, s := range rep.Fields {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d/%d\t%d\t%d\t%d/%d\t%d/%d\n", s.Field, s.Scored, s.Correct, s.Wrong, s.Blank, s.RuleCorrect, s.RuleWrong, s.GreenWrong, s.AmberWrong, s.RawCorrect, s.RawAnswers, s.HeldOutRight, s.HeldOut)
	}
	tw.Flush()
	fmt.Fprintf(w, "%d cases; %d wrong values marked green\n", rep.Cases, rep.FalseGreen)
}

func countSplit(answers []eval.Answer, split string) int {
	n := 0
	for _, a := range answers {
		if strings.HasPrefix(a.Case, split+"/") {
			n++
		}
	}
	return n
}

// shapeOf buckets option counts more coarsely than filing: an invoice has
// far fewer answers per question, and confidence is still never compared
// between a two-option and a many-option question.
// https://docs.typesafe.ai/confidence
func shapeOf(options int) eval.Shape {
	switch {
	case options < 2:
		return eval.Shape{}
	case options == 2:
		return eval.Shape{Min: 2, Max: 2}
	case options <= 8:
		return eval.Shape{Min: 3, Max: 8}
	default:
		return eval.Shape{Min: 9, Max: 255}
	}
}

func writeThresholds(o options, answers []eval.Answer) error {
	var cal []eval.Answer
	for _, a := range answers {
		if !strings.HasPrefix(a.Case, eval.SplitCalibration+"/") {
			continue
		}
		// eval.Fit buckets by its own shapes; map each answer to the bucket's
		// lower bound so Fit groups by ours.
		s := shapeOf(a.Options)
		if s == (eval.Shape{}) {
			continue
		}
		a.Field = a.Field + "@" + strconv.Itoa(s.Min)
		cal = append(cal, a)
	}
	policies := map[string]eval.FitPolicy{}
	for _, a := range cal {
		policies[a.Field] = policy
	}
	th := invoices.Thresholds{QuestionVersion: invoices.QuestionVersion, Questions: map[string][]invoices.Threshold{}}
	th.Calibration = fmt.Sprintf("cmd/invoice-eval -fit, %d calibration answers, %s", len(cal), time.Now().UTC().Format("2006-01-02"))
	th.Policy, _ = json.Marshal(policy)
	groups := map[string][]eval.Answer{}
	for _, a := range cal {
		groups[a.Field] = append(groups[a.Field], a)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		field, lo, _ := strings.Cut(k, "@")
		min, _ := strconv.Atoi(lo)
		s := shapeOf(min)
		group := groups[k]
		// Present the whole bucket to Fit as one shape.
		for i := range group {
			group[i].Options = 2
		}
		fits := eval.Fit(group, policies)
		for _, f := range fits {
			lo, hi := s.Min, s.Max
			t := invoices.Threshold{Options: &lo, MaxOptions: &hi, Green: f.Green, Amber: f.Amber, N: f.N, Reason: f.Reason}
			th.Questions[field] = append(th.Questions[field], t)
		}
	}
	if err := os.MkdirAll(o.data, 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(th, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(o.data, "thresholds.json"), append(out, '\n'), 0o644)
}

func confText(c *float64) string {
	if c == nil {
		return "-"
	}
	return strconv.FormatFloat(*c, 'f', 3, 64)
}
