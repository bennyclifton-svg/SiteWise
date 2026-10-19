// Command genfixture writes the seeded 2,000-activity, 4,000-link performance
// fixture that the Go benchmark and the TypeScript engine both load (plan
// 6.7). Run it from the repository root:
//
//	go run ./internal/program/cmd/genfixture -seed 1
//
// The program is acyclic by construction: rows sit in outline order (each
// stage followed by its activities) and every link runs from an earlier row
// to a later one, never between a stage and its own activity. Its expected
// finish is computed by the Go engine, so it is a regression and a
// cross-language check, not a hand-worked answer.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"sitewise/internal/program"
)

const (
	stages      = 40
	perStage    = 49 // 40 stages + 40 × 49 activities = 2,000 rows
	targetLinks = 4000
	reach       = 150 // links join rows at most this far apart in outline order
)

func main() {
	seed := flag.Int64("seed", 1, "random seed")
	out := flag.String("out", "internal/program/testdata/engine/perf-2000.json", "output file")
	flag.Parse()
	in := generate(rand.New(rand.NewSource(*seed)))
	if err := write(*out, *seed, in); err != nil {
		log.Fatal(err)
	}
}

func ptr(n int) *int { return &n }

func generate(r *rand.Rand) program.Input {
	in := program.Input{Calendar: program.Calendar{Start: "2027-03-01", WorkingWeek: program.DefaultWorkingWeek}}
	// Invented non-working days for coverage only; real calendars come from
	// the library once sources are named (owner answer Q6).
	for y := 2027; y <= 2031; y++ {
		for _, md := range []string{"01-26", "04-25", "06-14", "10-04"} {
			in.Calendar.Days = append(in.Calendar.Days, program.CalendarDay{Day: fmt.Sprintf("%d-%s", y, md), Kind: program.DayPublicHoliday, Label: "Fixture holiday"})
		}
		for d := 22; d <= 31; d++ {
			in.Calendar.Days = append(in.Calendar.Days, program.CalendarDay{Day: fmt.Sprintf("%d-12-%02d", y, d), Kind: program.DayShutdown, Label: "Fixture shutdown"})
		}
	}

	var rows []program.Activity
	for s := 0; s < stages; s++ {
		sid := fmt.Sprintf("s%02d", s)
		rows = append(rows, program.Activity{ID: sid, Ordinal: s, Kind: program.KindStage, Label: fmt.Sprintf("Stage %d", s+1)})
		for c := 0; c < perStage; c++ {
			a := program.Activity{ID: fmt.Sprintf("s%02d-%02d", s, c), ParentID: sid, Ordinal: c, Kind: program.KindActivity,
				Label: fmt.Sprintf("Activity %d.%d", s+1, c+1), Duration: ptr(1 + r.Intn(5))}
			switch r.Intn(20) {
			case 0:
				a.Kind, a.Duration = program.KindMilestone, ptr(0)
			case 1:
				a.Kind, a.Duration = program.KindHoldPoint, ptr(0)
			}
			if a.Kind == program.KindActivity && r.Intn(3) == 0 {
				lo, hi := max(1, *a.Duration-r.Intn(2)), *a.Duration+r.Intn(4)
				a.Low, a.High = &lo, &hi
			}
			if r.Intn(50) == 0 {
				a.Constraint, a.ConstraintDate = program.ConstraintSNET, fmt.Sprintf("2027-%02d-%02d", 3+r.Intn(9), 1+r.Intn(28))
			}
			rows = append(rows, a)
		}
	}
	in.Activities = rows

	seen := map[[2]int]bool{}
	link := func(i, j int, typ string, lag int, unit string) {
		seen[[2]int{i, j}] = true
		in.Links = append(in.Links, program.Link{ID: fmt.Sprintf("l%04d", len(in.Links)), Pred: rows[i].ID, Succ: rows[j].ID, Type: typ, Lag: lag, LagUnit: unit})
	}
	// Chain links, as generation writes them: stage to stage, and activity to
	// activity inside each stage.
	stageRow := func(s int) int { return s * (perStage + 1) }
	for s := 0; s < stages; s++ {
		if s > 0 {
			link(stageRow(s-1), stageRow(s), program.LinkFS, 0, program.LagWorking)
		}
		for c := 1; c < perStage; c++ {
			link(stageRow(s)+c, stageRow(s)+c+1, program.LinkFS, 0, program.LagWorking)
		}
	}
	for len(in.Links) < targetLinks {
		i := r.Intn(len(rows) - 1)
		j := i + 1 + r.Intn(reach)
		if j >= len(rows) || seen[[2]int{i, j}] || rows[j].ParentID == rows[i].ID {
			continue
		}
		typ := []string{program.LinkFS, program.LinkFS, program.LinkFS, program.LinkFS, program.LinkSS, program.LinkSS, program.LinkFF, program.LinkSF}[r.Intn(8)]
		// FF and SF into a stage with activities are refused (plan 6.5).
		if rows[j].Kind == program.KindStage && (typ == program.LinkFF || typ == program.LinkSF) {
			typ = program.LinkFS
		}
		lag := 0
		if r.Intn(3) == 0 {
			lag = r.Intn(9) - 3
		}
		unit := program.LagWorking
		if r.Intn(10) == 0 {
			unit = program.LagCalendar
		}
		link(i, j, typ, lag, unit)
	}
	return in
}

// write keeps one activity or link per line so the committed file diffs and
// loads in both languages without a pretty-printer.
func write(path string, seed int64, in program.Input) error {
	outp := program.Compute(in)
	if outp.Cycle != nil {
		return fmt.Errorf("generated program has a cycle: %v", outp.Cycle)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	line := func(v any, sep string) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "    %s%s\n", b, sep)
	}
	cal, _ := json.Marshal(in.Calendar)
	fmt.Fprintf(w, "{\n  \"name\": \"perf-2000\",\n  \"why\": \"Seeded performance fixture (go run ./internal/program/cmd/genfixture -seed %d): 2,000 activities, 4,000 links, acyclic. Expected finish is the Go engine's own answer, a regression and cross-language check, not hand-worked.\",\n", seed)
	fmt.Fprintf(w, "  \"calendar\": %s,\n  \"activities\": [\n", cal)
	for k, a := range in.Activities {
		line(a, sep(k, len(in.Activities)))
	}
	fmt.Fprintf(w, "  ],\n  \"links\": [\n")
	for k, l := range in.Links {
		line(l, sep(k, len(in.Links)))
	}
	expect, _ := json.Marshal(map[string]any{
		"finish": outp.Finish.TypicalIndex, "finish_low": outp.Finish.LowIndex, "finish_high": outp.Finish.HighIndex,
		"finish_dates": map[string]any{"typical": outp.Finish.Typical, "low": outp.Finish.Low, "high": outp.Finish.High},
		"status":       outp.Status,
	})
	fmt.Fprintf(w, "  ],\n  \"expect\": %s\n}\n", expect)
	return w.Flush()
}

func sep(k, n int) string {
	if k == n-1 {
		return ""
	}
	return ","
}
