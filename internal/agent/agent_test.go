package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/llm"
)

// scripted is a stand-in model: it replies with the next scripted turn and
// records what it was sent.
type scripted struct {
	turns []llm.Turn
	err   error
	sent  []llm.Conversation
}

func (s *scripted) Converse(_ context.Context, conv llm.Conversation, onText func(string)) (llm.Turn, error) {
	s.sent = append(s.sent, conv)
	if s.err != nil {
		return llm.Turn{}, s.err
	}
	if len(s.sent) > len(s.turns) {
		return llm.Turn{}, errors.New("the script ran out")
	}
	t := s.turns[len(s.sent)-1]
	if t.Text != "" {
		onText(t.Text)
	}
	t.Message = json.RawMessage(fmt.Sprintf(`{"role":"assistant","content":[{"type":"text","text":%q}]}`, t.Text))
	return t, nil
}

func call(id, name, input string) llm.ToolCall {
	return llm.ToolCall{ID: id, Name: name, Input: json.RawMessage(input)}
}

// box is a stand-in toolbox. "read" returns records, "passages" returns
// outside text, "apply" writes: it is refused when its input says so, and
// prepared when the turn has read outside text.
type box struct {
	mu   sync.Mutex
	ran  []string
	seen []bool // the turn's outside flag as each apply saw it
}

func (b *box) Tools() []llm.Tool {
	return []llm.Tool{{Name: "read"}, {Name: "passages"}, {Name: "apply"}}
}
func (b *box) Writes(name string) bool { return name == "apply" }
func (b *box) Run(_ context.Context, st *State, c llm.ToolCall) Output {
	b.mu.Lock()
	b.ran = append(b.ran, c.Name)
	b.mu.Unlock()
	switch c.Name {
	case "read":
		time.Sleep(20 * time.Millisecond)
		return Output{Text: "Mechanical services | budget 120000"}
	case "passages":
		return Output{Text: "p1 page 3: Tree 4 is to be retained.", Outside: "the Arborist report", Passages: []Passage{{ID: "p1", Document: "Arborist report", Page: 3, Text: "Tree 4 is to be retained."}}}
	}
	b.seen = append(b.seen, st.Outside)
	var in struct {
		Refuse bool `json:"refuse"`
		Final  bool `json:"final"`
	}
	_ = json.Unmarshal(c.Input, &in)
	switch {
	case in.Refuse:
		return Output{Text: "operation 1, amount: say the figure.", IsError: true, Refused: true}
	case st.Outside:
		return Output{Text: "prepared", Final: in.Final, Cards: []Card{{Kind: "prepared", Area: "cost_plan", Why: "this used text from " + st.Why}}}
	}
	return Output{Text: "applied", Final: in.Final, Cards: []Card{{Kind: "change", Area: "cost_plan", ChangeID: "c1", Lines: []string{"Added Joinery."}}}}
}

func run(m Model, b Toolbox, typed ...string) (Result, []Event) {
	var events []Event
	res := Run(context.Background(), m, b, Request{Model: "m", System: "sys", Typed: typed, Outline: "Cost sections\n Mechanical | budget 120000", Today: time.Now()}, func(e Event) { events = append(events, e) })
	return res, events
}

func kinds(events []Event) string {
	var k []string
	for _, e := range events {
		k = append(k, e.Type)
	}
	return strings.Join(k, " ")
}

func TestSimpleCommandIsOneModelRound(t *testing.T) {
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{call("a", "apply", `{"final":true}`)}}}}
	res, events := run(m, &box{}, "Add a line for joinery, $45,000")
	if res.Rounds != 1 || len(m.sent) != 1 || len(res.Cards) != 1 || res.Cards[0].Kind != "change" || res.Err != nil || res.Answer.Text != "" {
		t.Fatalf("rounds %d calls %d cards %+v err %v", res.Rounds, len(m.sent), res.Cards, res.Err)
	}
	if got := kinds(events); got != "tool change answer" {
		t.Fatalf("events: %s", got)
	}
	// The request: system text as given, the outline and message in the
	// user turn, the PM's figure in the book.
	sent := m.sent[0]
	if sent.System != "sys" || !strings.Contains(string(sent.History[0]), "project_outline") || !strings.Contains(string(sent.History[0]), "joinery") {
		t.Fatalf("request %+v", sent)
	}
	if !res.State.Book.HasNumber("45000") || res.State.Book.HasNumber("54000") || res.State.Book.HasNumber("120000") || !res.State.Seen.HasNumber("120000") {
		t.Fatal("the book should hold the PM's figure and no other")
	}
	// The turn's messages are kept for the store: user, assistant, results.
	if len(res.Messages) != 3 {
		t.Fatalf("%d messages kept", len(res.Messages))
	}
}

func TestApplyWithoutFinalGetsASecondRound(t *testing.T) {
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{call("a", "apply", `{}`)}}, {Text: "Done. Anything else?"}}}
	res, _ := run(m, &box{}, "Add a line")
	if res.Rounds != 2 || res.Answer.Text != "Done. Anything else?" || res.Answer.Question {
		// A question mark after a change is not a question back.
		t.Fatalf("result %+v", res)
	}
}

func TestReadsRunBeforeWritesAndOutsideTextPreparesTheChange(t *testing.T) {
	b := &box{}
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{call("a", "apply", `{"final":true}`), call("b", "passages", `{}`), call("c", "read", `{}`)}}}}
	res, events := run(m, b, "Read the arborist report and add what it says")
	if len(b.seen) != 1 || !b.seen[0] {
		t.Fatalf("the apply must see the outside flag the same round's read set: %v", b.seen)
	}
	if b.ran[len(b.ran)-1] != "apply" {
		t.Fatalf("writes run after reads: %v", b.ran)
	}
	if !res.Outside || len(res.Cards) != 1 || res.Cards[0].Kind != "prepared" || !strings.Contains(res.Cards[0].Why, "Arborist report") {
		t.Fatalf("result %+v", res.Cards)
	}
	if !strings.Contains(kinds(events), "prepared") || res.Rounds != 1 {
		t.Fatalf("events %s rounds %d", kinds(events), res.Rounds)
	}
	// All three results went back in one message.
	var msg struct {
		Content []map[string]any `json:"content"`
	}
	_ = json.Unmarshal(res.Messages[len(res.Messages)-1], &msg)
	if len(msg.Content) != 3 {
		t.Fatalf("%d tool results in the message", len(msg.Content))
	}
}

func TestTheSameWriteSentTwiceRunsOnce(t *testing.T) {
	b := &box{}
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{call("a", "apply", `{"final":true}`), call("b", "apply", `{"final":true}`)}}}}
	res, _ := run(m, b, "Add five rows")
	if len(b.ran) != 1 || len(res.Cards) != 1 || res.Rounds != 1 {
		t.Fatalf("ran %v, %d cards, %d rounds", b.ran, len(res.Cards), res.Rounds)
	}
}

func TestRefusalIsFedBackOnceThenReported(t *testing.T) {
	refuse := llm.Turn{Calls: []llm.ToolCall{call("a", "apply", `{"refuse":true}`)}}
	m := &scripted{turns: []llm.Turn{refuse, refuse, {Text: "never reached"}}}
	res, _ := run(m, &box{}, "Add joinery for a fair price")
	if res.Rounds != 2 || len(res.Cards) != 0 || !strings.Contains(res.Answer.Text, "say the figure") {
		t.Fatalf("result rounds %d text %q", res.Rounds, res.Answer.Text)
	}
	// One refusal, then a corrected batch, is an ordinary success.
	m = &scripted{turns: []llm.Turn{refuse, {Calls: []llm.ToolCall{call("b", "apply", `{"final":true}`)}}}}
	res, _ = run(m, &box{}, "Add joinery")
	if res.Rounds != 2 || len(res.Cards) != 1 {
		t.Fatalf("result %+v", res)
	}
}

func TestEightRoundsStops(t *testing.T) {
	var turns []llm.Turn
	for i := 0; i < 12; i++ {
		turns = append(turns, llm.Turn{Calls: []llm.ToolCall{call(fmt.Sprint(i), "read", `{}`)}})
	}
	m := &scripted{turns: turns}
	res, _ := run(m, &box{}, "Keep reading")
	if res.Rounds != MaxRounds || len(m.sent) != MaxRounds || !strings.Contains(res.Answer.Text, "stopped after eight") || !strings.Contains(res.Answer.Text, "Nothing was changed") {
		t.Fatalf("rounds %d text %q", res.Rounds, res.Answer.Text)
	}
}

func TestBadToolInputIsAnsweredNotRun(t *testing.T) {
	b := &box{}
	bad := llm.ToolCall{ID: "a", Name: "apply", Input: json.RawMessage(`{`), Err: "invalid input: not valid JSON"}
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{bad}}, {Text: "Which line do you mean?"}}}
	res, _ := run(m, b, "Do it")
	if len(b.ran) != 0 || res.Rounds != 2 || !res.Answer.Question {
		t.Fatalf("ran %v rounds %d answer %+v", b.ran, res.Rounds, res.Answer)
	}
	if !strings.Contains(string(res.Messages[2]), `"is_error":true`) {
		t.Fatalf("the bad call was not answered with an error: %s", res.Messages[2])
	}
}

func TestModelFailureChangesNothing(t *testing.T) {
	b := &box{}
	res, events := run(&scripted{err: llm.ErrBusy}, b, "Add a line")
	if res.Err == nil || len(b.ran) != 0 || len(events) != 0 {
		t.Fatalf("err %v ran %v events %v", res.Err, b.ran, events)
	}
	if Unavailable(res.Err, nil) != "Chat is unavailable. Nothing was changed." || Unavailable(llm.ErrRefused, nil) != "No change was made." {
		t.Fatal("the plain messages changed")
	}
	if got := Unavailable(llm.ErrBusy, []Card{{Kind: "change"}}); !strings.Contains(got, "were made") {
		t.Fatalf("a failure after a change must say the change stands: %q", got)
	}
}

func TestAnswerIsCheckedByGo(t *testing.T) {
	m := &scripted{turns: []llm.Turn{{Calls: []llm.ToolCall{call("a", "passages", `{}`)}}, {Text: `It says "Tree 4 is to be retained." [p1] Allow 14 days.`}}}
	res, events := run(m, &box{}, "Are trees involved?")
	if len(res.Answer.Cited) != 1 || res.Answer.Unchecked != 1 || strings.Contains(res.Answer.Text, "[p1]") {
		t.Fatalf("answer %+v", res.Answer)
	}
	if last := events[len(events)-1]; last.Type != "answer" || last.Answer == nil {
		t.Fatalf("last event %+v", last)
	}
}
