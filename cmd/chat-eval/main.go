// Command chat-eval scores the chat agent against the command key
// (data/eval/chat/commands.json).
//
//	chat-eval -replay transcript.json     score a recorded run (no model)
//	chat-eval -live -model claude-haiku-5-5 -runs 5 -out sheet.json
//
// The live run builds each case's fixture in a private database, sends the
// message through the real chat route, and reads the areas before and after.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"sitewise/internal/chateval"
)

func main() {
	key := flag.String("key", "data/eval/chat/commands.json", "the command key")
	replay := flag.String("replay", "", "a recorded transcript: case id to outcome")
	live := flag.Bool("live", false, "run against the live model")
	model := flag.String("model", "claude-haiku-5-5", "model id for -live")
	runs := flag.Int("runs", 5, "runs per case for -live")
	only := flag.String("only", "", "comma-separated case ids or groups to run")
	out := flag.String("out", "", "write the score sheet as JSON here")
	flag.Parse()
	cases, err := chateval.Load(*key)
	if err != nil {
		fail(err)
	}
	cases = pick(cases, *only)
	var sheet chateval.Sheet
	switch {
	case *live:
		agent, done, err := liveAgent(*model)
		if err != nil {
			fail(err)
		}
		sheet = chateval.RunAll(context.Background(), agent, cases, *runs)
		done()
	case *replay != "":
		raw, err := os.ReadFile(*replay)
		if err != nil {
			fail(err)
		}
		var recorded map[string]chateval.Outcome
		if err := json.Unmarshal(raw, &recorded); err != nil {
			fail(err)
		}
		sheet = chateval.RunAll(context.Background(), chateval.Replay(recorded), cases, 1)
	default:
		fmt.Printf("%d cases in %s; give -replay or -live to score\n", len(cases), *key)
		return
	}
	print(sheet)
	if *out != "" {
		raw, _ := json.MarshalIndent(sheet, "", " ")
		if err := os.WriteFile(*out, raw, 0o644); err != nil {
			fail(err)
		}
	}
}

func print(s chateval.Sheet) {
	groups := make([]string, 0, len(s.Groups))
	for g := range s.Groups {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	fmt.Println("| Kind of case | Runs | Passed | Wrong change | Missed question | Unchecked figure | Other fail | Errors | Ended prepared | Rounds per turn | p50 ms | p90 ms |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, g := range groups {
		r := s.Groups[g]
		rounds := 0.0
		if n := len(r.Millis); n > 0 {
			rounds = float64(r.Rounds) / float64(n)
		}
		fmt.Printf("| %s | %d | %d | %d | %d | %d | %d | %d | %d | %.1f | %d | %d |\n", g, r.Runs, r.Passed, r.WrongChange,
			r.MissedQuestion, r.UncheckedFigure, r.Failed, r.Errors, r.Prepared, rounds, chateval.Percentile(r.Millis, 50), chateval.Percentile(r.Millis, 90))
	}
	if len(s.Failed) > 0 {
		fmt.Println("\nMoney or date cases with a wrong change in any run (each fails whatever the floor):", s.Failed)
	}
	ids := make([]string, 0, len(s.Notes))
	for id := range s.Notes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, n := range s.Notes[id] {
			fmt.Printf("%s: %s\n", id, n)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "chat-eval:", err)
	os.Exit(1)
}
