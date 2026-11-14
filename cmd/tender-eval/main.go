// Command tender-eval runs the tender pass over real tenders and scores it
// against a private answer key.
//
//	go run ./cmd/tender-eval                 # replay every case, print the marks, write the aggregate
//	go run ./cmd/tender-eval -live           # call the model for cases with no recording of this request
//	go run ./cmd/tender-eval -live -fresh    # call it for every case
//	go run ./cmd/tender-eval -split hold     # only the hold-out cases (default: tune; "all" for both)
//	go run ./cmd/tender-eval -show           # also print, to the terminal, each case's misses
//	go run ./cmd/tender-eval -dump <case>    # print a draft key for one case from its reading
//	go run ./cmd/tender-eval -one <file> -schedule <id>   # run one file and print its lines
//
// The corpus, the answer key and the recording hold real firms and prices
// and stay in git-ignored folders. -one, -show and -dump print real prices
// to the terminal only. The aggregate written to -out holds no case id,
// label, firm name or amount. It exits nonzero on a changed corpus file, a
// replay miss or a tender sum shown wrong with no flag.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/invoices"
	"sitewise/internal/knowledge"
	"sitewise/internal/llm"
	"sitewise/internal/tender"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type options struct {
	one, schedule, key, recording, out, corpusRoot, knowledge, split, dump, model string
	live, fresh, show                                                             bool
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("tender-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.one, "one", "", "run one file through the pass and print its lines (real prices; terminal only)")
	fs.StringVar(&o.schedule, "schedule", "main-works", "RFP ids for -one, comma separated")
	fs.StringVar(&o.key, "key", "data/eval/tender/private/answer-key.json", "answer key")
	fs.StringVar(&o.recording, "recording", "data/eval/tender/private/recording.jsonl", "recorded model answers")
	fs.StringVar(&o.out, "out", "data/eval/tender/results/latest.json", "aggregate report (no case ids, names or amounts)")
	fs.StringVar(&o.corpusRoot, "corpus-root", "", "override the answer key's corpus_root")
	fs.StringVar(&o.knowledge, "knowledge", "knowledge", "knowledge directory")
	fs.StringVar(&o.split, "split", "tune", "tune, hold or all")
	fs.StringVar(&o.dump, "dump", "", "print a draft key for this case id from its reading")
	fs.StringVar(&o.model, "model", "", "model id (default: SITEWISE_TENDER_MODEL, else "+config.DefaultTenderModel+")")
	fs.BoolVar(&o.live, "live", false, "call the model where no answer is recorded")
	fs.BoolVar(&o.fresh, "fresh", false, "with -live: call the model for every case")
	fs.BoolVar(&o.show, "show", false, "print each case's misses (private; terminal only)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if o.model == "" {
		o.model = strings.TrimSpace(getenv("SITEWISE_TENDER_MODEL"))
	}
	if o.model == "" {
		o.model = config.DefaultTenderModel
	}
	ctx := context.Background()
	var err error
	if o.one != "" {
		err = runOne(ctx, o, getenv, stdout)
	} else {
		err = evaluate(ctx, o, getenv, stdout)
	}
	if err != nil {
		fmt.Fprintln(stderr, "tender-eval:", err)
		return 1
	}
	return 0
}

// liveClient is the real model client, or nil when no key is set. The two
// settings are read directly: config.Load demands a database.
func liveClient(getenv func(string) string) llm.Asker {
	c := llm.New(llm.Options{APIKey: strings.TrimSpace(getenv("SITEWISE_ANTHROPIC_HAIKU_KEY")), MaxConcurrent: 4})
	if !c.On() {
		return nil
	}
	return c
}

func runOne(ctx context.Context, o options, getenv func(string) string, w io.Writer) error {
	ask := liveClient(getenv)
	if ask == nil {
		return fmt.Errorf("SITEWISE_ANTHROPIC_HAIKU_KEY is not set")
	}
	cat, err := knowledge.Load(o.knowledge)
	if err != nil {
		return err
	}
	rows, err := rowsFor(cat, strings.Split(o.schedule, ","))
	if err != nil {
		return err
	}
	b, err := readFile(o.one)
	if err != nil {
		return err
	}
	rd := newReader(getenv)
	pages, paged, err := rd.pages(ctx, o.one, b)
	if err != nil {
		return err
	}
	raw, info, err := tender.Pass(ctx, ask, rd.render(), o.model, o.one, paged, pages, rows)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%.1f s, %d tokens in, %d out, stop %s, %d pages (%d as images, %d as text), %d lines, %d dropped\n",
		info.Latency.Seconds(), info.InTokens, info.OutTokens, info.Stop, len(pages), len(info.Sel.Images), len(info.Sel.Texts), len(raw.Lines), info.Dropped)
	for _, l := range raw.Lines {
		fmt.Fprintf(w, "%3d | %-8s | %-7s | %-28s | %14s | %s\n", l.P, l.Kind, l.Where, l.Row, l.Amt, l.Label)
	}
	r := tender.Reconcile(raw, pages, rows, nil)
	sum := "none"
	if r.TenderSum != nil {
		sum = invoices.FormatCents(*r.TenderSum)
	}
	fmt.Fprintf(w, "tender sum %s (%s), gap %s, why %q\n", sum, r.SumHow, invoices.FormatCents(r.Gap), r.Why)
	for _, f := range r.Flags {
		fmt.Fprintln(w, "  flag:", f.Text)
	}
	return nil
}

type caseResult struct {
	score            caseScore
	large, scan      bool
	latencyMS        int64
	inTokens, outTok int64
	lines            int
}

func evaluate(ctx context.Context, o options, getenv func(string) string, w io.Writer) error {
	key, err := loadKey(o.key, o.corpusRoot)
	if err != nil {
		return err
	}
	cat, err := knowledge.Load(o.knowledge)
	if err != nil {
		return err
	}
	var live llm.Asker
	if o.live {
		if live = liveClient(getenv); live == nil {
			return fmt.Errorf("SITEWISE_ANTHROPIC_HAIKU_KEY is not set")
		}
	}
	rep, err := newReplayer(o.recording, live, o.fresh)
	if err != nil {
		return err
	}
	rd := newReader(getenv)
	var results []caseResult
	silent, empties, failed := 0, 0, 0
	for _, c := range key.Cases {
		if o.dump != "" && c.ID != o.dump {
			continue
		}
		if o.dump == "" && o.split != "all" && c.Split != o.split {
			continue
		}
		rows, err := rowsFor(cat, c.Schedules)
		if err != nil {
			return fmt.Errorf("case %s: %w", c.ID, err)
		}
		path := filepath.Join(key.CorpusRoot, filepath.FromSlash(c.Path))
		raw, info, pages, err := runCase(ctx, rep, rd, o.model, path, c.ID, rows, live != nil, &empties)
		if errors.Is(err, errReplayMiss) {
			return fmt.Errorf("case %s: %w (run with -live)", c.ID, err)
		}
		if err != nil {
			// One case that cannot be read does not hide the rest.
			failed++
			fmt.Fprintf(w, "%-16s NOT READ: %v\n", c.ID, err)
			continue
		}
		reading := tender.Reconcile(raw, pages, rows, nil)
		if info.Cut || info.Dropped > 0 {
			reading.Flags = append(reading.Flags, tender.Flag{Code: tender.FlagIncomplete, Text: "The reading was cut off. Some lines may be missing."})
		}
		if o.dump != "" {
			return dumpKey(w, c, reading)
		}
		s := scoreCase(c, reading)
		if s.Sum == "silent" {
			silent++
		}
		results = append(results, caseResult{score: s, large: len(c.Lines) >= 100, scan: c.Scan, latencyMS: info.Latency.Milliseconds(),
			inTokens: info.InTokens, outTok: info.OutTokens, lines: len(raw.Lines)})
		if o.show {
			share := 0.0
			if s.KeyDollars > 0 {
				share = float64(s.RowDollars) / float64(s.KeyDollars)
			}
			fmt.Fprintf(w, "%-16s sum %-7s rows %5.1f%% lines %3d key %3d  %.1fs  why %q gap %s\n", c.ID, s.Sum, 100*share, s.Lines, s.KeyLines,
				info.Latency.Seconds(), reading.Why, invoices.FormatCents(reading.Gap))
			for _, f := range reading.Flags {
				fmt.Fprintln(w, "    flag:", f.Text)
			}
			for _, m := range s.Misses {
				fmt.Fprintln(w, "    "+m)
			}
		}
	}
	if o.dump != "" {
		return fmt.Errorf("no case %q in the key", o.dump)
	}
	out := summarize(results, key.Reviewed, o)
	out.EmptyAnswers, out.NotRead = empties, failed
	printReport(w, out)
	if err := writeReport(o.out, out); err != nil {
		return err
	}
	if silent > 0 {
		return fmt.Errorf("%d tender sums are shown wrong with no flag", silent)
	}
	return nil
}

// runCase reads one file and runs the pass, within four minutes. An empty
// answer is asked again, as the product does.
func runCase(ctx context.Context, rep *replayer, rd reader, model, path, id string, rows []tender.Row, live bool, empties *int) (tender.RawReading, tender.PassInfo, []tender.Page, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	b, err := readFile(path)
	if err != nil {
		return tender.RawReading{}, tender.PassInfo{}, nil, errors.New("its file cannot be read")
	}
	pages, paged, err := rd.pages(ctx, path, b)
	if err != nil {
		return tender.RawReading{}, tender.PassInfo{}, nil, errors.New("its text cannot be extracted")
	}
	rep.caseID = id
	raw, info, err := tender.Pass(ctx, rep, rd.render(), model, path, paged, pages, rows)
	for try := 0; errors.Is(err, tender.ErrEmpty) && live && try < 3; try++ {
		rep.forgetLast()
		*empties++
		raw, info, err = tender.Pass(ctx, rep, rd.render(), model, path, paged, pages, rows)
	}
	return raw, info, pages, err
}

// dumpKey prints a draft key for one case: a labelling aid, nothing else.
func dumpKey(w io.Writer, c keyCase, r tender.Reading) error {
	c.TenderSum, c.Lines = "", nil
	if r.TenderSum != nil {
		c.TenderSum = invoices.FormatCents(*r.TenderSum)
	}
	for _, l := range r.Lines {
		switch l.Kind {
		case "base", "ps", "pc", "option", "saving":
			if l.Cents != nil {
				c.Lines = append(c.Lines, keyLine{Label: l.Label, Amount: invoices.FormatCents(*l.Cents), Row: l.Row, Counted: l.Counted, Kind: l.Kind})
			}
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(c)
}

// report is the aggregate. It holds no case id, label, firm name or amount.
type report struct {
	At              time.Time `json:"at"`
	Prompt          string    `json:"prompt"`
	Model           string    `json:"model"`
	Split           string    `json:"split"`
	KeyReviewed     bool      `json:"key_reviewed"`
	Cases           int       `json:"cases"`
	SumSilent       int       `json:"sum_silent"`
	SumFlagged      int       `json:"sum_flagged"`
	SumRight        int       `json:"sum_right"`
	SumRightShare   float64   `json:"sum_right_share"`
	RowShare        float64   `json:"row_dollars_share"`
	RowShareLarge   float64   `json:"row_dollars_share_large"`
	LargeCases      int       `json:"large_cases"`
	FoundShare      float64   `json:"found_dollars_share"`
	OutsideCounted  int       `json:"outside_counted"`
	AmountsKept     float64   `json:"amounts_kept_share"`
	AmountsKeptScan float64   `json:"amounts_kept_share_scans"`
	ScanCases       int       `json:"scan_cases"`
	ShortP50S       float64   `json:"seconds_p50_up_to_100_lines"`
	ShortP90S       float64   `json:"seconds_p90_up_to_100_lines"`
	LongP50S        float64   `json:"seconds_p50_over_100_lines"`
	LongP90S        float64   `json:"seconds_p90_over_100_lines"`
	ShortCases      int       `json:"cases_up_to_100_lines"`
	LongCases       int       `json:"cases_over_100_lines"`
	InTokens        int64     `json:"in_tokens"`
	OutTokens       int64     `json:"out_tokens"`
	TokensPerLine   float64   `json:"out_tokens_per_line"`
	CostUSD         float64   `json:"cost_usd"`
	EmptyAnswers    int       `json:"empty_answers_asked_again"`
	NotRead         int       `json:"cases_not_read"`
}

// Claude Haiku 5.5, US dollars per million tokens, prompts up to 100,000
// tokens (the pass's ceiling). Another model's cost is not worked out here.
const haikuInUSD, haikuOutUSD = 0.10, 0.50

func percentile(sorted []int64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p*float64(len(sorted))+0.999999) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return float64(sorted[i]) / 1000
}

func summarize(results []caseResult, reviewed bool, o options) report {
	rep := report{At: time.Now().UTC(), Prompt: tender.PromptVersion, Model: o.model, Split: o.split, KeyReviewed: reviewed, Cases: len(results)}
	var rowSum, rowLarge, foundSum float64
	var rowN, asked, kept, askedScan, keptScan, lines int
	var short, long []int64
	for _, r := range results {
		s := r.score
		switch s.Sum {
		case "right":
			rep.SumRight++
		case "flagged":
			rep.SumFlagged++
		default:
			rep.SumSilent++
		}
		if s.KeyDollars > 0 {
			share := float64(s.RowDollars) / float64(s.KeyDollars)
			rowSum, foundSum, rowN = rowSum+share, foundSum+float64(s.FoundDollars)/float64(s.KeyDollars), rowN+1
			if r.large {
				rowLarge += share
				rep.LargeCases++
			}
		}
		rep.OutsideCounted += s.OutsideCounted
		asked, kept = asked+s.AmountsAsked, kept+s.AmountsKept
		if r.scan {
			rep.ScanCases++
			askedScan, keptScan = askedScan+s.AmountsAsked, keptScan+s.AmountsKept
		}
		if r.lines > 100 {
			long = append(long, r.latencyMS)
		} else {
			short = append(short, r.latencyMS)
		}
		rep.InTokens, rep.OutTokens, lines = rep.InTokens+r.inTokens, rep.OutTokens+r.outTok, lines+r.lines
	}
	div := func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	}
	rep.SumRightShare = div(float64(rep.SumRight), float64(len(results)))
	rep.RowShare, rep.FoundShare = div(rowSum, float64(rowN)), div(foundSum, float64(rowN))
	rep.RowShareLarge = div(rowLarge, float64(rep.LargeCases))
	rep.AmountsKept, rep.AmountsKeptScan = div(float64(kept), float64(asked)), div(float64(keptScan), float64(askedScan))
	sort.Slice(short, func(i, j int) bool { return short[i] < short[j] })
	sort.Slice(long, func(i, j int) bool { return long[i] < long[j] })
	rep.ShortCases, rep.LongCases = len(short), len(long)
	rep.ShortP50S, rep.ShortP90S = percentile(short, 0.5), percentile(short, 0.9)
	rep.LongP50S, rep.LongP90S = percentile(long, 0.5), percentile(long, 0.9)
	rep.TokensPerLine = div(float64(rep.OutTokens), float64(lines))
	if o.model == config.DefaultTenderModel {
		rep.CostUSD = (float64(rep.InTokens)*haikuInUSD + float64(rep.OutTokens)*haikuOutUSD) / 1e6
	}
	return rep
}

func printReport(w io.Writer, r report) {
	mark := func(ok bool) string {
		if ok {
			return "pass"
		}
		return "UNDER"
	}
	reviewed := "the key is NOT yet reviewed by the owner"
	if r.KeyReviewed {
		reviewed = "the key is reviewed"
	}
	fmt.Fprintf(w, "\nprompt %s, model %s, split %s, %d cases; %s\n", r.Prompt, r.Model, r.Split, r.Cases, reviewed)
	fmt.Fprintf(w, "  sum shown wrong with no flag   %d            (pass: 0)  %s\n", r.SumSilent, mark(r.SumSilent == 0))
	fmt.Fprintf(w, "  sum right                      %d of %d, %.0f%% (pass: 90%%) %s; %d flagged\n", r.SumRight, r.Cases, 100*r.SumRightShare, mark(r.SumRightShare >= 0.9), r.SumFlagged)
	fmt.Fprintf(w, "  dollars in the right row       %.1f%%        (pass: 80%%) %s; found on any row %.1f%%\n", 100*r.RowShare, mark(r.RowShare >= 0.8), 100*r.FoundShare)
	fmt.Fprintf(w, "  the same, large tenders only   %.1f%% over %d cases\n", 100*r.RowShareLarge, r.LargeCases)
	fmt.Fprintf(w, "  outside money counted          %d            (pass: 0)  %s\n", r.OutsideCounted, mark(r.OutsideCounted == 0))
	fmt.Fprintf(w, "  amounts kept by R3             %.1f%%        (under 97%% is a finding); scans %.1f%% over %d cases\n", 100*r.AmountsKept, 100*r.AmountsKeptScan, r.ScanCases)
	fmt.Fprintf(w, "  seconds, up to 100 lines       p50 %.1f p90 %.1f over %d (budget 15 and 30) %s\n", r.ShortP50S, r.ShortP90S, r.ShortCases, mark(r.ShortP50S <= 15 && r.ShortP90S <= 30))
	fmt.Fprintf(w, "  seconds, over 100 lines        p50 %.1f p90 %.1f over %d (budget 60 and 90) %s\n", r.LongP50S, r.LongP90S, r.LongCases, mark(r.LongP50S <= 60 && r.LongP90S <= 90))
	fmt.Fprintf(w, "  tokens                         %d in, %d out, %.0f out per line; US$%.3f\n", r.InTokens, r.OutTokens, r.TokensPerLine, r.CostUSD)
}

func writeReport(path string, r report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(r, "", " ")
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
