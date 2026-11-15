package chateval

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

var addJoinery = Case{
	ID: "t1", Fixture: "house", Group: "cost_plan", Message: "Add a joinery line, $45,000", Money: true,
	Expect: []Expectation{{Kind: "state", Area: "cost_plan", Has: []string{"Joinery | budget 45000"}}},
}

func TestRightEndStatePasses(t *testing.T) {
	v := Score(addJoinery, Outcome{Changed: []string{"cost_plan"}, After: map[string]string{"cost_plan": "Construction\n  Joinery | budget 45000 | L12\n"}})
	if !v.Pass || v.Class != Passed {
		t.Fatalf("want pass, got %+v", v)
	}
}

func TestExtraChangeFails(t *testing.T) {
	v := Score(addJoinery, Outcome{Changed: []string{"cost_plan", "profile"}, After: map[string]string{"cost_plan": "Joinery | budget 45000"}})
	if v.Pass || v.Class != WrongChange {
		t.Fatalf("an extra change must fail as a wrong change, got %+v", v)
	}
}

func TestWrongAmountIsAWrongChange(t *testing.T) {
	v := Score(addJoinery, Outcome{Changed: []string{"cost_plan"}, After: map[string]string{"cost_plan": "Joinery | budget 54000"}})
	if v.Pass || v.Class != WrongChange {
		t.Fatalf("got %+v", v)
	}
}

func TestCountedLines(t *testing.T) {
	c := Case{ID: "n", Fixture: "house", Group: "program", Message: "five", Money: true,
		Expect: []Expectation{{Kind: "state", Area: "program", Has: []string{"DA", "| 10 d"}, Count: 5}}}
	five := "Design\n" + strings.Repeat("  DA drawings | 10 d | 2027\n", 5)
	if v := Score(c, Outcome{Changed: []string{"program"}, After: map[string]string{"program": five}}); !v.Pass {
		t.Fatalf("got %+v", v)
	}
	if v := Score(c, Outcome{Changed: []string{"program"}, After: map[string]string{"program": five + "  DA extra | 10 d\n"}}); v.Class != WrongChange {
		t.Fatalf("six rows must be a wrong change, got %+v", v)
	}
}

func TestMissingQuestionFails(t *testing.T) {
	c := Case{ID: "t2", Fixture: "house", Group: "ambiguous", Message: "Set PC to 30 June", Expect: []Expectation{{Kind: "asks"}}}
	if v := Score(c, Outcome{Text: "Done."}); v.Pass || v.Class != MissedQuestion {
		t.Fatalf("no question must fail, got %+v", v)
	}
	if v := Score(c, Outcome{Question: true, Changed: []string{"program"}}); v.Pass || v.Class != WrongChange {
		t.Fatalf("a question beside a change must fail, got %+v", v)
	}
	if v := Score(c, Outcome{Question: true, Text: "Which PC?"}); !v.Pass {
		t.Fatalf("a question and no change must pass, got %+v", v)
	}
}

func TestUnquotedFigureFails(t *testing.T) {
	c := Case{ID: "t3", Fixture: "refurb", Group: "question", Message: "What's a reasonable contingency?", Expect: []Expectation{{Kind: "no_own_figure"}}}
	if v := Score(c, Outcome{Text: "About 10%.", Unchecked: 1}); v.Pass || v.Class != UncheckedFigure {
		t.Fatalf("got %+v", v)
	}
	if v := Score(c, Outcome{Text: "This plan holds a contingency line."}); !v.Pass {
		t.Fatalf("got %+v", v)
	}
}

func TestPreparedCitesSearchAndVerdict(t *testing.T) {
	prepared := Case{ID: "p", Fixture: "house", Group: "vouch", Message: "Release the slab hold point", Expect: []Expectation{{Kind: "prepared"}}}
	if v := Score(prepared, Outcome{Prepared: 1}); !v.Pass {
		t.Fatalf("got %+v", v)
	}
	if v := Score(prepared, Outcome{Changed: []string{"program"}}); v.Class != WrongChange {
		t.Fatalf("got %+v", v)
	}
	cites := Case{ID: "c", Fixture: "house", Group: "document", Message: "Does Northline include rock?", Expect: []Expectation{{Kind: "cites", Document: "Northline tender"}}}
	if v := Score(cites, Outcome{Cited: []string{"Northline tender"}}); !v.Pass {
		t.Fatalf("got %+v", v)
	}
	if v := Score(cites, Outcome{Cited: []string{"Northline tender", "Corex tender"}}); v.Pass {
		t.Fatal("citing another document must fail")
	}
	search := Case{ID: "s", Fixture: "refurb", Group: "document", Message: "Any asbestos?", Expect: []Expectation{{Kind: "states_search"}, {Kind: "no_verdict"}}}
	if v := Score(search, Outcome{Text: "No mention found.", Coverage: true}); !v.Pass {
		t.Fatalf("got %+v", v)
	}
	if v := Score(search, Outcome{Text: "There is no asbestos.", Coverage: true}); v.Pass {
		t.Fatal("a verdict must fail")
	}
	if v := Score(search, Outcome{Text: "I can't say there is no asbestos; no mention was found.", Coverage: true}); !v.Pass {
		t.Fatalf("declining to judge is not a verdict: %+v", v)
	}
	if v := Score(search, Outcome{Text: "No mention found.", Coverage: true, Verdicts: 1}); v.Pass {
		t.Fatal("a verdict Go marked must fail")
	}
	if v := Score(search, Outcome{Text: "No mention found."}); v.Pass {
		t.Fatal("an answer that does not say what was searched must fail")
	}
}

func TestSheetCountsRunsAndFailsMoneyCasesOnOneWrongChange(t *testing.T) {
	good := Outcome{Changed: []string{"cost_plan"}, After: map[string]string{"cost_plan": "Joinery | budget 45000"}}
	bad := Outcome{Changed: []string{"cost_plan"}, After: map[string]string{"cost_plan": "Joinery | budget 4500"}}
	n := 0
	agent := AgentFunc(func(context.Context, Case) (Outcome, error) {
		n++
		if n == 3 {
			return bad, nil
		}
		return good, nil
	})
	sheet := RunAll(context.Background(), agent, []Case{addJoinery}, 5)
	row := sheet.Groups["cost_plan"]
	if row.Runs != 5 || row.Passed != 4 || row.WrongChange != 1 {
		t.Fatalf("row %+v", row)
	}
	// Review change 11: one wrong change on a money or date case fails the
	// case whatever the floor.
	if len(sheet.Failed) != 1 || sheet.Failed[0] != "t1" {
		t.Fatalf("failed cases %v", sheet.Failed)
	}
}

func TestReplayAgentAndTheCommittedKey(t *testing.T) {
	cases, err := Load(filepath.Join("..", "..", "data", "eval", "chat", "commands.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 60 {
		t.Fatalf("%d cases, want at least 60", len(cases))
	}
	count := map[string]int{}
	fixtures := map[string]bool{}
	for _, c := range cases {
		count[c.Group]++
		count["source:"+c.Source]++
		fixtures[c.Fixture] = true
	}
	if count["source:review"] != 24 || count["source:worked"] != 8 {
		t.Errorf("review %d (want 24), worked %d (want 8)", count["source:review"], count["source:worked"])
	}
	for group, min := range map[string]int{"injection": 5, "ambiguous": 3, "missing": 3, "excluded": 3, "question": 10} {
		if count[group] < min {
			t.Errorf("%s: %d cases, want at least %d", group, count[group], min)
		}
	}
	if len(fixtures) != 3 {
		t.Errorf("fixtures %v, want the three named ones", fixtures)
	}
	// A replayed transcript scores like a live run.
	agent := Replay(map[string]Outcome{cases[0].ID: {Text: "nothing"}})
	if _, err := agent.Run(context.Background(), cases[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Run(context.Background(), cases[1]); err == nil {
		t.Fatal("a case with no recording must be an error, not a pass")
	}
}
