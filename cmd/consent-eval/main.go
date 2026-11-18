// Command consent-eval scores the consent conditions read against the
// owner's answer key (docs/plans/2026-11-17-consent-conditions.md, Tasks 3
// and 10). It only reports: it never writes a threshold or marks a key
// reviewed.
//
//	consent-eval -pages notice.pdf -name petersham-2016
//	    writes data/eval/consent/private/petersham-2016.pages.json (no Jev)
//	consent-eval
//	    splits every keyed notice and scores the split; with a recording,
//	    replays Jev's answers and scores the picks
//	consent-eval -live
//	    records Jev's answers and times the read (needs SITEWISE_JEV_API_KEY;
//	    measures the consent_read row: 16 in flight, job start to last row)
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
	"strings"
	"sync"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/consent"
	"sitewise/internal/eval"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "consent-eval:", err)
		os.Exit(1)
	}
}

type options struct {
	keys, recording, pages, name, knowledge string
	live, list                              bool
}

func run(ctx context.Context, args []string, out io.Writer, getenv func(string) string) error {
	fs := flag.NewFlagSet("consent-eval", flag.ContinueOnError)
	var o options
	fs.StringVar(&o.keys, "key", "data/eval/consent/private", "the answer key directory")
	fs.StringVar(&o.recording, "recording", "data/eval/consent/private/jev.jsonl", "recorded Jev answers")
	fs.StringVar(&o.pages, "pages", "", "a notice file (.pdf, .md, .txt) to write as pages beside the key")
	fs.StringVar(&o.name, "name", "", "the key document name for -pages")
	fs.StringVar(&o.knowledge, "knowledge", "knowledge", "the knowledge directory (systems for the system question)")
	fs.BoolVar(&o.live, "live", false, "ask Jev and record the answers")
	fs.BoolVar(&o.list, "list", false, "with -pages: list each condition's number, page, gate and heading (for drafting the key)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.pages != "" {
		return writePages(ctx, o, out)
	}
	keys, err := consent.LoadKeys(o.keys)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		fmt.Fprintf(out, "No answer key in %s. The owner supplies it; nothing is scored.\n", o.keys)
		return nil
	}
	notices := map[string]consent.Notice{}
	for _, k := range keys {
		pages, err := consent.PagesFile(filepath.Join(o.keys, k.Document+".pages.json"))
		if err != nil {
			fmt.Fprintf(out, "%s: no pages (%v); run -pages first\n", k.Document, err)
			continue
		}
		n := consent.Split(pages)
		exact, problems := consent.SplitScore(k, n)
		fmt.Fprintf(out, "%s: split %d conditions, key %d, exact %v %s; gaps %v\n", k.Document, len(n.Conditions), len(k.Conditions), exact, strings.Join(problems, "; "), n.Gaps)
		if exact {
			notices[k.Document] = n
		}
	}
	if len(notices) == 0 {
		fmt.Fprintln(out, "No split is exact, so no pick is scored.")
		return nil
	}
	cat, err := knowledge.Load(o.knowledge)
	if err != nil {
		return err
	}
	var systems []consent.System
	for _, s := range cat.BuildingOrderSystems() {
		systems = append(systems, consent.System{ID: s.ID, Label: s.Label})
	}
	ask, done, err := client(ctx, o, getenv)
	if err != nil {
		return err
	}
	defer done()
	for _, k := range keys {
		n, ok := notices[k.Document]
		if !ok {
			continue
		}
		if err := score(ctx, out, ask, k, n, systems); err != nil {
			return err
		}
	}
	return done()
}

// score asks (or replays) one fan-out per condition, 16 in flight as the
// job does, and counts agreement per field and band. It reports only.
func score(ctx context.Context, out io.Writer, ask jobs.Asker, k consent.Key, n consent.Notice, systems []consent.System) error {
	readings := make([]consent.Reading, len(n.Conditions))
	start := time.Now()
	var mu sync.Mutex
	var first error
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	for i, c := range n.Conditions {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c consent.Condition) {
			defer wg.Done()
			defer func() { <-sem }()
			call := consent.Call(c, systems)
			result, err := ask.Ask(ctx, call)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && first == nil {
				first = err
			}
			readings[i] = consent.Apply(c, call, result, err != nil)
		}(i, c)
	}
	wg.Wait()
	elapsed := time.Since(start)
	if first != nil {
		return first
	}
	want := map[string]consent.KeyCondition{}
	for _, c := range k.Conditions {
		want[c.Number] = c
	}
	type tally struct{ amber, right, blank, grey int }
	tallies := map[string]*tally{}
	for i, c := range n.Conditions {
		key := want[c.Number]
		for field, f := range readings[i].Fields {
			t := tallies[field]
			if t == nil {
				t = &tally{}
				tallies[field] = t
			}
			switch f.Band {
			case "grey":
				t.grey++
			case "blank":
				t.blank++
			case "amber", "rule":
				t.amber++
				if f.Value == keyValue(key, field) {
					t.right++
				}
			}
		}
	}
	fields := make([]string, 0, len(tallies))
	for f := range tallies {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	fmt.Fprintf(out, "%s: %d conditions read in %v (reviewed key: %v)\n", k.Document, len(n.Conditions), elapsed.Round(time.Millisecond), k.Reviewed)
	for _, f := range fields {
		t := tallies[f]
		fmt.Fprintf(out, "  %-8s shown %d, right %d, blank %d, grey %d\n", f, t.amber, t.right, t.blank, t.grey)
	}
	return nil
}

func keyValue(k consent.KeyCondition, field string) string {
	switch field {
	case "timing":
		return k.Stage
	case "party":
		return k.Party
	case "system":
		return k.System
	case "evidence":
		if k.Evidence == nil {
			return ""
		}
		return fmt.Sprint(*k.Evidence)
	}
	return ""
}

// writePages extracts a notice's text page by page, as filing's full_text
// job does, and writes it beside the key. No Jev.
func writePages(ctx context.Context, o options, out io.Writer) error {
	if o.name == "" {
		return errors.New("-name is needed with -pages")
	}
	var pages []consent.Page
	switch ext := strings.ToLower(filepath.Ext(o.pages)); ext {
	case ".md", ".txt":
		body, err := os.ReadFile(o.pages)
		if err != nil {
			return err
		}
		pages = []consent.Page{{Page: 1, Text: string(body)}}
	default:
		f, err := os.Open(o.pages)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		limits := identity.DefaultLimits()
		limits.MaxPages, limits.MaxRuns, limits.ReadControlPage = 200, 400000, false
		text, err := identity.Extract(ctx, strings.TrimPrefix(ext, "."), f, info.Size(), limits)
		if err != nil {
			return err
		}
		for _, p := range jobs.SourceFromText(text).Source {
			pages = append(pages, consent.Page{Page: p.Page, Text: p.Text})
		}
	}
	n := consent.Split(pages)
	raw := make([]map[string]any, len(pages))
	for i, p := range pages {
		raw[i] = map[string]any{"page": p.Page, "text": p.Text}
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(o.keys, 0o700); err != nil {
		return err
	}
	path := filepath.Join(o.keys, o.name+".pages.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return err
	}
	if o.list {
		for _, c := range n.Conditions {
			fmt.Fprintf(out, "%s | p%d | %s | %s | %s | %s\n", c.Number, c.Page, c.Stage, c.Marker, c.StageHeading, c.Heading)
		}
	}
	fmt.Fprintf(out, "%s: %d pages, detected %v, %d conditions, reference %q, determined %q, gaps %v\n", path, len(pages), consent.Detect(pages), len(n.Conditions), n.ConsentRef, n.Determined, n.Gaps)
	return nil
}

func client(ctx context.Context, o options, getenv func(string) string) (jobs.Asker, func() error, error) {
	if o.live {
		k := strings.TrimSpace(getenv("SITEWISE_JEV_API_KEY"))
		if k == "" {
			return nil, nil, errors.New("SITEWISE_JEV_API_KEY is required for -live")
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
		closed := false
		return c, func() error {
			if closed {
				return nil
			}
			closed = true
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
