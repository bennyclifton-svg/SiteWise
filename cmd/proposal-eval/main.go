// Command proposal-eval measures fee-proposal reading against an answer key
// of real proposals and fits the cut-offs Jev's answers need.
//
//	go run ./cmd/proposal-eval -live      # call System One, record every exchange
//	go run ./cmd/proposal-eval [-fit]     # replay; -fit writes data/proposal/thresholds.json
//	go run ./cmd/proposal-eval -schedule town-planning   # print a discipline's default schedule
//
// The answer key, the corpus and the recording hold real fees and stay in
// data/eval/proposal/private/ (git-ignored). Each case is read into its
// discipline's default schedule, as a proposal that arrives before an RFP
// is issued would be. Scoring applies the production order: rule, then Jev,
// then code. It exits nonzero on a changed corpus file, a replay miss or any
// wrong value marked green.
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

	"gopkg.in/yaml.v3"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/invoices"
	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type options struct {
	key, recording, data, out, corpusRoot, knowledge, schedule string
	live, fit, show, dump                                      bool
	deadline                                                   time.Duration
}

// policy is the accuracy a cut-off must show on the calibration split
// (Wilson lower bounds, 95 % one-sided), the invoice policy; green also
// needs no calibration error at or above its cut. Proposed by the
// agent on 16 October 2026; the owner has not signed it off.
var policy = eval.FitPolicy{Z: 1.645, MinSamples: 20, MinBandSamples: 10, GreenLower: 0.95, AmberLower: 0.70, MaxErrors: -1}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("proposal-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.key, "key", "data/eval/proposal/private/answer-key.json", "answer key")
	fs.StringVar(&o.recording, "recording", "data/eval/proposal/private/recording.jsonl", "Jev recording")
	fs.StringVar(&o.data, "data", "data/proposal", "thresholds directory")
	fs.StringVar(&o.out, "out", "data/eval/proposal/results/latest.json", "aggregate report (no case names)")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the answer key corpus_root")
	fs.StringVar(&o.knowledge, "knowledge", "knowledge", "knowledge directory")
	fs.StringVar(&o.schedule, "schedule", "", "print this RFP discipline's default schedule and stop")
	fs.BoolVar(&o.live, "live", false, "call System One and record every exchange")
	fs.BoolVar(&o.fit, "fit", false, "write cut-offs fitted on the calibration split")
	fs.BoolVar(&o.show, "show", false, "print each wrong or blank field (private values; terminal only)")
	fs.BoolVar(&o.dump, "dump", false, "print each case's schedule and harvested amounts for labelling, without asking Jev")
	fs.DurationVar(&o.deadline, "deadline", 20*time.Second, "per-call deadline")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := evaluate(context.Background(), o, getenv, stdout); err != nil {
		fmt.Fprintln(stderr, "proposal-eval:", err)
		return 1
	}
	return 0
}

type answerKey struct {
	Version         int    `json:"version"`
	Reviewed        bool   `json:"reviewed"`
	LabelSource     string `json:"label_source"`
	QuestionVersion string `json:"question_version"`
	CorpusRoot      string `json:"corpus_root"`
	// Projects name the RFP scenario (data/eval/rfp/answer-keys) whose
	// facts and settings a case's schedule is resolved with.
	Projects map[string]struct {
		Scenario string `json:"scenario"`
	} `json:"projects"`
	Cases []keyCase `json:"cases"`
}

// keyCase is one proposal. Prices map a schedule key to "cents:<n>",
// "not_priced" or "included"; a key left out is not scored. Firm values list
// every acceptable spelling.
type keyCase struct {
	ID          string            `json:"id"`
	Path        string            `json:"path"`
	SHA256      string            `json:"sha256"`
	RFP         string            `json:"rfp"`
	Project     string            `json:"project"`
	Discipline  string            `json:"discipline"`
	Family      string            `json:"family"`
	Kind        string            `json:"kind"`
	FirmName    []string          `json:"firm_name"`
	FirmABN     string            `json:"firm_abn"`
	FirmEmail   []string          `json:"firm_email"`
	StatedTotal string            `json:"stated_total"`
	GSTBasis    string            `json:"gst_basis"`
	Prices      map[string]string `json:"prices"`
	Settings    map[string]string `json:"settings"`
}

func evaluate(ctx context.Context, o options, getenv func(string) string, stdout io.Writer) error {
	cat, err := knowledge.Load(o.knowledge)
	if err != nil {
		return err
	}
	if o.schedule != "" {
		items, err := schedule(cat, o.schedule, scenario{}, nil)
		if err != nil {
			return err
		}
		for _, it := range items {
			fmt.Fprintf(stdout, "%-40s %-12s base=%-5v %s\n", it.Key, it.Kind, it.Base, it.Label)
		}
		return nil
	}
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
	if key.QuestionVersion != proposals.QuestionVersion {
		return fmt.Errorf("answer key is for %q, this build asks %q", key.QuestionVersion, proposals.QuestionVersion)
	}
	root := key.CorpusRoot
	if o.corpusRoot != "" {
		root = o.corpusRoot
	}
	th, err := proposals.LoadThresholds(o.data)
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
		sc, err := loadScenario(key.Projects[c.Project].Scenario)
		if err != nil {
			return fmt.Errorf("case %s: %w", c.ID, err)
		}
		items, err := schedule(cat, c.RFP, sc, c.Settings)
		if err != nil {
			return fmt.Errorf("case %s: %w", c.ID, err)
		}
		if o.dump {
			if err := dump(ctx, stdout, root, c, items); err != nil {
				return fmt.Errorf("case %s: %w", c.ID, err)
			}
			continue
		}
		r, raw, err := evaluateCase(ctx, o, root, c, items, asker, th)
		if err != nil {
			return fmt.Errorf("case %s: %w", c.ID, err)
		}
		results = append(results, r)
		answers = append(answers, raw...)
	}
	if o.dump {
		return nil
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
					fmt.Fprintf(stdout, "%-14s %-44s %-7s %s/%s got=%q want=%q conf=%s\n", r.ID, f.Field, f.Outcome, f.By, f.Band, f.Got, f.Want, confText(f.Conf))
				}
			}
		}
	}
	printReport(stdout, rep)
	if o.fit {
		if err := writeThresholds(o, answers); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s; re-run without -fit to score with it\n", filepath.Join(o.data, "thresholds.json"))
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

// scenario is the part of an RFP answer-key scenario a schedule needs.
type scenario struct {
	Facts    map[string]yaml.Node            `yaml:"facts"`
	Flags    map[string]string               `yaml:"flags"`
	Address  string                          `yaml:"address"`
	Settings map[string]map[string]yaml.Node `yaml:"settings"`
	Parts    []struct {
		ID      string   `yaml:"id"`
		Label   string   `yaml:"label"`
		Kind    string   `yaml:"kind"`
		Classes []string `yaml:"classes"`
		GFA     string   `yaml:"gfa"`
	} `yaml:"parts"`
}

func loadScenario(path string) (scenario, error) {
	var sc scenario
	if path == "" {
		return sc, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return sc, err
	}
	return sc, yaml.Unmarshal(b, &sc)
}

// drivers are the scenario's facts as the RFP reads them; a fact a scenario
// leaves out is unknown.
func (sc scenario) drivers() rfp.DriverSet {
	d := rfp.DriverSet{Facts: map[string]rfp.Fact{}, Text: map[string]string{"address": sc.Address}}
	for k, n := range sc.Facts {
		if n.Kind == yaml.ScalarNode {
			d.Facts[k] = rfp.Fact{Values: []string{n.Value}, State: rfp.Known, Source: "profile"}
			continue
		}
		var f struct {
			State  string   `yaml:"state"`
			Values []string `yaml:"values"`
		}
		if n.Decode(&f) == nil {
			d.Facts[k] = rfp.Fact{Values: f.Values, State: f.State, Source: "profile"}
		}
	}
	for k, v := range sc.Flags {
		d.Facts["flag."+k] = rfp.Fact{Values: []string{v}, State: rfp.Known, Source: "profile"}
	}
	for _, p := range sc.Parts {
		kind := p.Kind
		if kind == "" {
			kind = "building"
		}
		d.Parts = append(d.Parts, rfp.Part{ID: p.ID, Label: p.Label, Kind: kind, Classes: p.Classes, GFA: p.GFA})
	}
	return d
}

// schedule is the discipline's RFP resolved with the project's facts and
// settings, and the case's own stage settings: what a proposal is read into.
func schedule(cat *knowledge.Catalog, id string, sc scenario, settings map[string]string) ([]rfp.ScheduleItem, error) {
	d, ok := cat.RFP(id)
	sk, skOK := cat.RFPSkeleton()
	if !ok || !skOK {
		return nil, fmt.Errorf("no RFP knowledge for %q", id)
	}
	s := rfp.Settings{}
	for k, n := range sc.Settings[id] {
		if strings.HasPrefix(k, "driver:") {
			continue
		}
		var v any
		if n.Decode(&v) == nil {
			raw, _ := json.Marshal(v)
			s[k] = rfp.Setting{Value: raw}
		}
	}
	for k, v := range settings {
		raw, _ := json.Marshal(v)
		s[k] = rfp.Setting{Value: raw}
	}
	return rfp.Schedule(rfp.Resolve(d, sk, sc.drivers(), s)), nil
}

func client(ctx context.Context, o options, getenv func(string) string) (proposals.Asker, func() error, func() int, error) {
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

type fieldResult struct {
	Conf    *float64
	Got     string
	Want    []string
	Field   string
	Family  string
	Outcome string // correct, wrong, blank
	Band    string
	By      string
}

type caseResult struct {
	ID     string
	Split  string
	Fields []fieldResult
}

const evalPackage = "00000000-0000-4000-8000-000000000001"

func evaluateCase(ctx context.Context, o options, root string, c keyCase, items []rfp.ScheduleItem, ask proposals.Asker, th proposals.Thresholds) (caseResult, []eval.Answer, error) {
	path := filepath.Join(root, filepath.FromSlash(c.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		return caseResult{}, nil, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != c.SHA256 {
		return caseResult{}, nil, errors.New("file changed since it was labelled")
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	text, err := identity.Extract(ctx, format, strings.NewReader(string(b)), int64(len(b)), proposals.ExtractLimits())
	if err != nil {
		return caseResult{}, nil, err
	}
	family := c.Family
	if family == "" {
		family = c.ID
	}
	res := caseResult{ID: c.ID, Split: eval.SplitOf(family, proposals.QuestionVersion, 60)}
	ev := proposals.Harvest(filepath.Base(path), text)
	pc := proposals.Context{
		Packages:   []proposals.Package{{ID: evalPackage, Title: c.RFP, Discipline: c.Discipline}},
		Discipline: c.Discipline,
		Schedules:  map[string][]rfp.ScheduleItem{evalPackage: items},
		OwnDomains: []string{"engineroomvm.com", "enginerommvm.com", "invoke.net.au"},
	}
	d := proposals.NewDraft(ev, pc, nil)
	var raw []eval.Answer
	var result jev.Result
	var askErr error
	if call, asked := d.Call(); asked {
		if size, err := jev.MeasureRequest(call); err == nil && o.show {
			fmt.Fprintf(os.Stderr, "%-14s %d questions, %d bytes\n", c.ID, size.Questions, size.Bytes)
		}
		call.Deadline = o.deadline
		cctx, cancel := context.WithTimeout(ctx, o.deadline+time.Second)
		result, askErr = ask.Ask(cctx, call)
		cancel()
		if askErr != nil && errors.Is(askErr, eval.ErrReplayMiss) {
			return caseResult{}, nil, askErr
		}
		if askErr != nil {
			fmt.Fprintf(os.Stderr, "case %s: jev: %v\n", c.ID, askErr)
		}
		for _, f := range d.Questions() {
			want, ok := expected(c, f)
			if !ok {
				continue
			}
			v, conf, got := d.Raw(result, f)
			if !got || conf == nil {
				continue
			}
			fam, n := d.Family(f)
			right := matches(d, f, want, v)
			if o.show && !right {
				fmt.Fprintf(os.Stderr, "raw miss %-14s %-44s got=%q want=%q conf=%s\n", c.ID, f, shown(d, v), want, confText(conf))
			}
			raw = append(raw, eval.Answer{Case: c.ID, Field: fam, Options: n, Confidence: *conf, Correct: right})
		}
	}
	d.Apply(result, askErr, th)
	for _, f := range scoredFields(c) {
		want, _ := expected(c, f)
		dec, _ := d.Decided(f)
		out := "blank"
		switch {
		case dec.Value == "" && contains(want, ""):
			out = "correct"
		case dec.Value == "":
		case matches(d, f, want, dec.Value):
			out = "correct"
		default:
			out = "wrong"
		}
		fam, _ := d.Family(f)
		if fam == "" {
			fam = familyOf(f)
		}
		res.Fields = append(res.Fields, fieldResult{Field: f, Family: fam, Outcome: out, Band: dec.Band, By: dec.DecidedBy, Got: shown(d, dec.Value), Want: want, Conf: dec.Confidence})
	}
	for i := range raw {
		raw[i].Case = res.Split + "/" + raw[i].Case
	}
	return res, raw, nil
}

// dump prints what a labeller needs: the file's hash, the schedule's price
// and rate rows, and every harvested amount with its words.
func dump(ctx context.Context, w io.Writer, root string, c keyCase, items []rfp.ScheduleItem) error {
	path := filepath.Join(root, filepath.FromSlash(c.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	text, err := identity.Extract(ctx, format, strings.NewReader(string(b)), int64(len(b)), proposals.ExtractLimits())
	if err != nil {
		return err
	}
	ev := proposals.Harvest(filepath.Base(path), text)
	fmt.Fprintf(w, "=== %s  %s  sha256 %s\n", c.ID, c.Path, hex.EncodeToString(sum[:]))
	fmt.Fprintf(w, "firm: names=%q abns=%v emails=%v gst ex/inc=%d/%d\n", ev.Names[:min(len(ev.Names), 6)], ev.ABNs, ev.Emails, ev.GSTEx, ev.GSTInc)
	for _, it := range items {
		if it.Price() || it.Kind == rfp.ItemRate {
			fmt.Fprintf(w, "  item %-44s %s\n", it.Key, clip(it.Label, 70))
		}
	}
	for _, a := range ev.Amounts {
		flag := ""
		if a.Unit || a.GSTPart || a.IncTwin {
			flag = " (dropped)"
		}
		fmt.Fprintf(w, "  %-4s %-13s hr=%-5v tot=%-5v gst=%-3s p%-2d label=%q col=%q%s | %s\n", a.ID, a.Display, a.PerHour, a.Total, a.GST, a.Page, clip(a.Label, 40), a.Column, flag, clip(a.Context, 90))
	}
	return nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func familyOf(field string) string {
	switch {
	case strings.HasPrefix(field, proposals.PricePrefix+"rate:"):
		return "rate"
	case strings.HasPrefix(field, proposals.PricePrefix):
		return "price"
	case strings.HasPrefix(field, "firm_"):
		return "firm"
	}
	return field
}

func scoredFields(c keyCase) []string {
	out := []string{proposals.FieldKind}
	if len(c.FirmName) > 0 {
		out = append(out, proposals.FieldFirmName)
	}
	if c.FirmABN != "" {
		out = append(out, proposals.FieldFirmABN)
	}
	if len(c.FirmEmail) > 0 {
		out = append(out, proposals.FieldFirmEmail)
	}
	if c.StatedTotal != "" {
		out = append(out, proposals.FieldStatedTotal)
	}
	if c.GSTBasis != "" {
		out = append(out, proposals.FieldGSTBasis)
	}
	keys := make([]string, 0, len(c.Prices))
	for k := range c.Prices {
		keys = append(keys, proposals.PricePrefix+k)
	}
	sort.Strings(keys)
	return append(out, keys...)
}

func expected(c keyCase, field string) ([]string, bool) {
	switch field {
	case proposals.FieldKind:
		return []string{c.Kind}, c.Kind != ""
	case proposals.FieldFirmName:
		return c.FirmName, len(c.FirmName) > 0
	case proposals.FieldFirmABN:
		return []string{c.FirmABN}, c.FirmABN != ""
	case proposals.FieldFirmEmail:
		return c.FirmEmail, len(c.FirmEmail) > 0
	case proposals.FieldStatedTotal:
		if c.StatedTotal == "" {
			return nil, false
		}
		var out []string
		for _, alt := range strings.Split(c.StatedTotal, "|") {
			if alt == "none" {
				out = append(out, "")
				continue
			}
			out = append(out, cents(alt))
		}
		return out, true
	case proposals.FieldGSTBasis:
		return []string{c.GSTBasis}, c.GSTBasis != ""
	}
	if k, ok := strings.CutPrefix(field, proposals.PricePrefix); ok {
		v, ok := c.Prices[k]
		if !ok {
			return nil, false
		}
		// "a|b" accepts either: one fee printed for several schedule rows
		// may sit on any of them, the others included.
		var out []string
		for _, alt := range strings.Split(v, "|") {
			if alt == "not_priced" || alt == "included" {
				out = append(out, alt)
				continue
			}
			out = append(out, "cents:"+cents(alt))
		}
		return out, true
	}
	return nil, false
}

func cents(v string) string {
	v = strings.TrimPrefix(v, "cents:")
	n, ok, err := invoices.ParseCents(v)
	if err != nil || !ok {
		return v
	}
	return strconv.FormatInt(n, 10)
}

// matches compares an answer (an amount id, a choice token or a value) with
// the labels. An amount id matches by its figure.
func matches(d *proposals.Draft, field string, want []string, got string) bool {
	g := normal(d, field, got)
	for _, w := range want {
		if field == proposals.FieldFirmName {
			// One spelling inside the other is the same firm: "LID" and
			// "Low Impact Development (LID) Consulting Pty Ltd".
			a, b := proposals.NameKey(w), proposals.NameKey(g)
			if len(a) >= 3 && len(b) >= 3 && (strings.Contains(a, b) || strings.Contains(b, a)) {
				return true
			}
			continue
		}
		if strings.EqualFold(w, g) {
			return true
		}
	}
	return false
}

func normal(d *proposals.Draft, field, got string) string {
	if a, ok := d.Amount(got); ok {
		if field == proposals.FieldStatedTotal {
			return strconv.FormatInt(a.Cents, 10)
		}
		return "cents:" + strconv.FormatInt(a.Cents, 10)
	}
	return got
}

func shown(d *proposals.Draft, v string) string {
	if a, ok := d.Amount(v); ok {
		return v + "=" + a.Display
	}
	return v
}

type fieldStats struct {
	Family     string `json:"family"`
	Scored     int    `json:"scored"`
	Correct    int    `json:"correct"`
	Wrong      int    `json:"wrong"`
	Blank      int    `json:"blank"`
	RuleOK     int    `json:"rule_correct"`
	RuleWrong  int    `json:"rule_wrong"`
	GreenWrong int    `json:"green_wrong"`
	AmberWrong int    `json:"amber_wrong"`
	RawAnswers int    `json:"raw_answers"`
	RawCorrect int    `json:"raw_correct"`
	HeldOut    int    `json:"held_out_scored"`
	HeldOutOK  int    `json:"held_out_correct"`
}

type report struct {
	QuestionVersion string         `json:"question_version"`
	Cases           int            `json:"cases"`
	FalseGreen      int            `json:"false_green"`
	Families        []*fieldStats  `json:"families"`
	Policy          eval.FitPolicy `json:"fit_policy"`
}

func summarize(results []caseResult, answers []eval.Answer) report {
	by := map[string]*fieldStats{}
	get := func(f string) *fieldStats {
		if by[f] == nil {
			by[f] = &fieldStats{Family: f}
		}
		return by[f]
	}
	rep := report{QuestionVersion: proposals.QuestionVersion, Cases: len(results), Policy: policy}
	for _, r := range results {
		for _, f := range r.Fields {
			s := get(f.Family)
			s.Scored++
			switch f.Outcome {
			case "correct":
				s.Correct++
				if f.By == proposals.ByRule {
					s.RuleOK++
				}
			case "wrong":
				s.Wrong++
				if f.By == proposals.ByRule {
					s.RuleWrong++
				}
				if f.Band == proposals.BandGreen {
					s.GreenWrong++
					rep.FalseGreen++
				}
				if f.Band == proposals.BandAmber {
					s.AmberWrong++
				}
			default:
				s.Blank++
			}
			if r.Split == eval.SplitHeldout {
				s.HeldOut++
				if f.Outcome == "correct" {
					s.HeldOutOK++
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
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rep.Families = append(rep.Families, by[k])
	}
	return rep
}

func printReport(w io.Writer, rep report) {
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "family\tscored\tcorrect\twrong\tblank\trule ok/wrong\tgreen wrong\tamber wrong\tjev raw ok/n\theld-out ok/n")
	for _, s := range rep.Families {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d/%d\t%d\t%d\t%d/%d\t%d/%d\n", s.Family, s.Scored, s.Correct, s.Wrong, s.Blank, s.RuleOK, s.RuleWrong, s.GreenWrong, s.AmberWrong, s.RawCorrect, s.RawAnswers, s.HeldOutOK, s.HeldOut)
	}
	tw.Flush()
	fmt.Fprintf(w, "%d cases; %d wrong values marked green\n", rep.Cases, rep.FalseGreen)
}

// shapeOf buckets option counts: confidence is never compared between a
// two-option and a many-option question. https://docs.typesafe.ai/confidence
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

func writeThresholds(o options, answers []eval.Answer) error {
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
		a.Field = k
		a.Options = 2 // present the whole bucket to Fit as one shape
		groups[k] = append(groups[k], a)
		n++
	}
	th := proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	th.Calibration = fmt.Sprintf("cmd/proposal-eval -fit, %d calibration answers, %s", n, time.Now().UTC().Format("2006-01-02"))
	th.Policy, _ = json.Marshal(policy)
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		field, lo, _ := strings.Cut(k, "@")
		min, _ := strconv.Atoi(lo)
		s := shapeOf(min)
		// Green comes from a strict fit (no calibration error at or above
		// the cut); amber from the ordinary one, below green. A fee shown
		// green is one a PM will not re-read.
		strict := policy
		strict.MaxErrors = 0
		sf := eval.Fit(groups[k], map[string]eval.FitPolicy{k: strict})
		for i, f := range eval.Fit(groups[k], map[string]eval.FitPolicy{k: policy}) {
			lo, hi := s.Min, s.Max
			t := proposals.Threshold{Options: &lo, MaxOptions: &hi, Amber: f.Amber, N: f.N, Reason: f.Reason}
			if i < len(sf) && sf[i].Green != nil {
				t.Green = sf[i].Green
			}
			if t.Green == nil && t.Amber != nil {
				t.Reason = "review only; no cut-off is free of calibration errors"
			}
			if t.Green != nil && (t.Amber == nil || *t.Amber > *t.Green) {
				t.Amber = t.Green
			}
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
