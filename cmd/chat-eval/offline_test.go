package main

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/chateval"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// The screen group of the command key, scored offline (chat agency plan,
// Task 7). A stand-in model makes the call a right answer makes; the real
// server does the rest, on the real fixture, and the real scorer judges the
// end state. So this scores Go's half of each case: the set a filter or a
// subject finds, the ids a selection or the history carries, the undo, the
// dialog's fields, and that a move after a document read prepares nothing.
// Whether the model makes that call is the live run's to score
// (cmd/chat-eval -live), and so are the cases that are judgement alone.

// modelOnly are the screen cases the stand-in cannot score: what is right
// there is the model's own choice of words or of question.
var modelOnly = []string{
	"s9",  // "these" with nothing selected: the model must ask
	"s22", // the messages table: the model must say the fixed words
	"s23", // the RFI table: the same
	"s27", // "this document": the model must quote the open document
	"ga1", // who has not opened: the answer's words
	"ga3", // who can see a document: the answer's words
	"ga8", // reminding firms waits on correspondence: the fixed words
}

// standIn answers each round with the call its script gives for the PM's
// message, and counts the calls.
type standIn struct {
	mu     sync.Mutex
	calls  int
	script map[string]func(conv llm.Conversation, round int) llm.Turn
}

var pmMessage = regexp.MustCompile(`(?s)<pm_message>\n(.*?)\n</pm_message>`)

func (m *standIn) Converse(_ context.Context, conv llm.Conversation, _ func(string)) (llm.Turn, error) {
	if conv.Warm {
		return llm.Turn{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	// The PM's message is in the turn's first user message; later rounds
	// add tool results after it.
	asked, round := "", 0
	for _, raw := range conv.History {
		if found := pmMessage.FindStringSubmatch(textOf(raw)); found != nil {
			asked, round = found[1], 0
		}
		if strings.Contains(string(raw), `"tool_result"`) {
			round++
		}
	}
	play, ok := m.script[asked]
	if !ok {
		return say("The stand-in has no script for: " + asked), nil
	}
	return play(conv, round), nil
}

func textOf(raw json.RawMessage) string {
	var msg struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(raw, &msg)
	var b strings.Builder
	for _, c := range msg.Content {
		b.WriteString(c.Text)
	}
	return b.String()
}

func say(text string) llm.Turn {
	msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": text}}})
	return llm.Turn{Text: text, StopReason: "end_turn", Message: msg}
}

func call(name string, input map[string]any) llm.Turn {
	raw, _ := json.Marshal(input)
	msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "tool_use", "id": "call-1", "name": name, "input": json.RawMessage(raw)}}})
	return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: name, Input: raw}}, StopReason: "tool_use", Message: msg}
}

// once is a script of one call; a later round, if any, answers in words.
func once(name string, input map[string]any) func(llm.Conversation, int) llm.Turn {
	return func(_ llm.Conversation, round int) llm.Turn {
		if round > 0 {
			return say("Done.")
		}
		return call(name, input)
	}
}

// readThen reads a document first, then moves the screen: the move must
// still prepare nothing.
func readThen(words string, view map[string]any) func(llm.Conversation, int) llm.Turn {
	return func(_ llm.Conversation, round int) llm.Turn {
		if round == 0 {
			return call("passages", map[string]any{"words": words})
		}
		if round == 1 {
			return call("view", view)
		}
		return say("Done.")
	}
}

// fromHistory moves the rows an earlier turn's line named ("[Selected 2
// cost lines: a b]"), as a model reading the history would.
func fromHistory(do, table string) func(llm.Conversation, int) llm.Turn {
	line := regexp.MustCompile(`\[Selected [^:\]]*:((?: [0-9a-f]{8})+)\]`)
	return func(conv llm.Conversation, round int) llm.Turn {
		if round > 0 {
			return say("Done.")
		}
		var ids []string
		for _, raw := range conv.History {
			if m := line.FindStringSubmatch(textOf(raw)); m != nil {
				ids = strings.Fields(m[1])
			}
		}
		if len(ids) == 0 {
			return say("Which rows?")
		}
		return call("view", map[string]any{"do": do, "table": table, "ids": ids, "final": true})
	}
}

// scripts are each case's calls, with this run's ids.
func scripts(n *names) map[string]func(llm.Conversation, int) llm.Turn {
	short := func(table string, rows ...string) []string {
		ids, _ := n.ids(table, rows)
		for i, id := range ids {
			ids[i] = id[:8]
		}
		return ids
	}
	view := func(in map[string]any) func(llm.Conversation, int) llm.Turn {
		in["final"] = true
		return once("view", in)
	}
	filter := func(do, table, name string) func(llm.Conversation, int) llm.Turn {
		return view(map[string]any{"do": do, "table": table, "filter": name})
	}
	about := func(subject string) func(llm.Conversation, int) llm.Turn {
		return once("documents", map[string]any{"about": subject, "select": true})
	}
	apply := func(area, op string, fields map[string]any) func(llm.Conversation, int) llm.Turn {
		return once("apply", map[string]any{"area": area, "final": true, "ops": []map[string]any{{"op": op, "fields": fields}}})
	}
	award := func(discipline, firm string) func(llm.Conversation, int) llm.Turn {
		pkg := short("procurement", discipline)
		col := n.firms[firm]
		if len(pkg) == 0 || col == "" {
			return once("view", map[string]any{"do": "open", "dialog": "award", "fields": map[string]string{}})
		}
		return view(map[string]any{"do": "open", "dialog": "award", "fields": map[string]string{"package": pkg[0], "column": col[:8]}})
	}
	mechanical := short("procurement", "trade.mechanical")
	return map[string]func(llm.Conversation, int) llm.Turn{
		"Show the cost lines with no budget":                                    filter("only", "cost_lines", "no_budget"),
		"Select the cost lines that are over budget":                            filter("select", "cost_lines", "over_budget"),
		"Show only the superseded documents":                                    filter("only", "documents", "superseded"),
		"Select the Program rows with no duration":                              filter("select", "program", "no_duration"),
		"Show the packages with no quote":                                       filter("only", "procurement", "no_quote"),
		"Select the basement documents":                                         about("basement"),
		"Select the documents about trees":                                      about("trees"),
		"Exclude these from the totals":                                         apply("cost_plan", "cost_plan.set_lines", map[string]any{"lines": short("cost_lines", "Landscaping", "Escalation", "DA fees"), "excluded": true}),
		"Show only these":                                                       view(map[string]any{"do": "only", "table": "documents", "ids": short("documents", "Finishes specification", "Carpet data sheet")}),
		"Select the landscaping and escalation lines":                           view(map[string]any{"do": "select", "table": "cost_lines", "ids": short("cost_lines", "Landscaping", "Escalation")}),
		"Exclude escalation from the totals":                                    apply("cost_plan", "cost_plan.set_lines", map[string]any{"lines": short("cost_lines", "Escalation"), "excluded": true}),
		"Duplicate the make good line":                                          apply("cost_plan", "cost_plan.duplicate_lines", map[string]any{"lines": short("cost_lines", "Make good")}),
		"Open the award dialog for Polar Air on the mechanical package":         award("trade.mechanical", "Polar Air"),
		"Open the award dialog for Northline on the structural package":         award("consultant.structural", "Northline Structural"),
		"Find what the consent says about working hours, then open the consent": readThen("working hours, construction hours", map[string]any{"do": "open", "table": "documents", "ids": short("documents", "Development consent DA-2026-0412"), "final": true}),
		"Read the arborist report and select it":                                readThen("tree, arborist", map[string]any{"do": "select", "table": "documents", "ids": short("documents", "Arborist report"), "final": true}),
		"Update the profile":                                                    apply("profile", "profile.update", map[string]any{}),
		"Read Polar Air's tender again":                                         apply("procurement", "procurement.read_tender_again", map[string]any{"package": strings.Join(mechanical, ""), "firm": "Polar Air"}),
		"Select the structural package":                                         view(map[string]any{"do": "select", "table": "procurement", "ids": short("procurement", "consultant.structural")}),
		"Show all the cost lines again":                                         view(map[string]any{"do": "clear", "table": "cost_lines"}),
		// Guest links (docs/plans/2026-11-17-guest-access.md, "From chat").
		"Show the packages whose invited firm has not opened its link": filter("only", "procurement", "link_not_opened"),
		"Share the structural RFP with Hartwell":                       view(map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "package", "package": strings.Join(short("procurement", "consultant.structural"), ""), "firm": "Hartwell"}}),
		"Share the arborist report with the client":                    view(map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "documents", "documents": strings.Join(short("documents", "Arborist report"), ",")}}),
		"Revoke Hartwell's link":                                       apply("shares", "shares.revoke", map[string]any{"share": strings.Join(short("shares", "hartwell@example.test"), "")}),
		"Share the cost plan with the client": func(_ llm.Conversation, round int) llm.Turn {
			if round > 0 {
				return say("Only documents, an issued report or an issued RFP or RFT can be shared.")
			}
			return call("view", map[string]any{"do": "open", "dialog": "share", "fields": map[string]string{"kind": "cost_plan"}})
		},
	}
}

func TestScreenCasesScoredOffline(t *testing.T) {
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	model := &standIn{}
	l, done, err := newRunner(st, model, nil, "stand-in", "../..")
	if err != nil {
		t.Fatal(err)
	}
	l.dsn = dsn
	defer done()
	l.warmFor = 0
	l.onSeed = func(c chateval.Case, n *names) {
		s := scripts(n)
		switch c.ID {
		case "s11":
			s["Now show only those"] = fromHistory("only", "documents")
		case "s12":
			s["Now show only those"] = fromHistory("only", "cost_lines")
		}
		model.mu.Lock()
		model.script = s
		model.mu.Unlock()
	}
	cases, err := chateval.Load("../../data/eval/chat/commands.json")
	if err != nil {
		t.Fatal(err)
	}
	scored := 0
	for _, c := range cases {
		if (c.Group != "screen" && c.Group != "shares") || slices.Contains(modelOnly, c.ID) {
			continue
		}
		scored++
		t.Run(c.ID, func(t *testing.T) {
			o, err := l.Run(ctx, c)
			if err != nil {
				t.Fatal(err)
			}
			if v := chateval.Score(c, o); !v.Pass {
				t.Fatalf("%s %q: %s %v\nanswer: %s\nviews: %+v", c.ID, c.Message, v.Class, v.Reasons, o.Text, o.Views)
			}
		})
	}
	if scored < 25 {
		t.Fatalf("%d screen and shares cases scored offline, want at least 25", scored)
	}
}
