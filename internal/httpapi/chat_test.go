package httpapi_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/agent"
	"sitewise/internal/costs"
	"sitewise/internal/httpapi"
	"sitewise/internal/llm"
)

// scriptModel is a stand-in model. Each call is answered by the script,
// which sees the conversation it was sent.
type scriptModel struct {
	mu     sync.Mutex
	sent   []llm.Conversation
	script func(n int, conv llm.Conversation) llm.Turn
}

func (s *scriptModel) Converse(_ context.Context, conv llm.Conversation, onText func(string)) (llm.Turn, error) {
	s.mu.Lock()
	s.sent = append(s.sent, conv)
	n := len(s.sent)
	s.mu.Unlock()
	if conv.Warm {
		return llm.Turn{}, nil
	}
	t := s.script(n, conv)
	if t.Text != "" && onText != nil {
		onText(t.Text)
	}
	blocks := []map[string]any{}
	if t.Text != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": t.Text})
	}
	for _, c := range t.Calls {
		blocks = append(blocks, map[string]any{"type": "tool_use", "id": c.ID, "name": c.Name, "input": json.RawMessage(c.Input)})
	}
	t.Message, _ = json.Marshal(map[string]any{"role": "assistant", "content": blocks})
	t.CacheRead = 6000
	return t, nil
}

func (s *scriptModel) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

func withChat(t *testing.T, m agent.Model) func(*httpapi.Options) {
	prompt, err := httpapi.LoadChatPrompt(filepath.Join("..", "..", "knowledge", "chat"))
	if err != nil {
		t.Fatal(err)
	}
	return func(o *httpapi.Options) {
		o.Chat = &httpapi.Chat{Model: m, Name: "scripted", Prompt: prompt}
	}
}

// applyCall is a model turn that calls apply.
func applyCall(area string, final bool, ops ...map[string]any) llm.Turn {
	in, _ := json.Marshal(map[string]any{"area": area, "ops": ops, "final": final})
	return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: "apply", Input: in}}, StopReason: "tool_use"}
}

type chatEvent struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Tool string `json:"tool"`
	Card *struct {
		Kind     string   `json:"kind"`
		Area     string   `json:"area"`
		Lines    []string `json:"lines"`
		ChangeID string   `json:"change_id"`
		Undo     string   `json:"undo"`
		Prepared string   `json:"prepared_id"`
		Why      string   `json:"why"`
	} `json:"card"`
	Answer *struct {
		Text      string `json:"text"`
		Question  bool   `json:"question"`
		Unchecked int    `json:"unchecked"`
		Coverage  string `json:"coverage"`
	} `json:"answer"`
	ConversationID string `json:"conversation_id"`
	Rounds         int    `json:"rounds"`
	CacheRead      int64  `json:"cache_read_tokens"`
}

// chat sends one message and returns the events of the turn.
func (m *member) chat(t *testing.T, project, conversation, text string) []chatEvent {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"conversation_id": conversation, "text": text})
	resp := m.do(t, http.MethodPost, "/api/projects/"+project+"/chat", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("chat = %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var out []chatEvent
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var e chatEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("event %q: %v", line, err)
		}
		out = append(out, e)
	}
	if len(out) == 0 || out[len(out)-1].Type != "done" {
		t.Fatalf("the stream did not end with done: %+v", out)
	}
	return out
}

func eventKinds(events []chatEvent) string {
	var k []string
	for _, e := range events {
		k = append(k, e.Type)
	}
	return strings.Join(k, " ")
}

func cardOf(t *testing.T, events []chatEvent, kind string) chatEvent {
	t.Helper()
	for _, e := range events {
		if e.Type == kind && e.Card != nil {
			return e
		}
	}
	t.Fatalf("no %s card in %s", kind, eventKinds(events))
	return chatEvent{}
}

func chatApp(t *testing.T, script func(n int, conv llm.Conversation) llm.Turn) (*app, *member, string, *scriptModel) {
	t.Helper()
	model := &scriptModel{script: script}
	a := newApp(t, withProfile(t), withChat(t, model))
	m := a.member(t, newUUID(t))
	m.modelSwitch(t, true, http.StatusOK)
	return a, m, m.createProject(t, "Chat"), model
}

// Worked case 2: three cost lines, one with an amount, in one model round.
func TestChatSimpleCommandAppliesInOneRound(t *testing.T) {
	_, m, project, model := chatApp(t, func(int, llm.Conversation) llm.Turn {
		return applyCall("cost_plan", true,
			op("cost_plan.add_line", map[string]any{"label": "Joinery", "section": "construction", "system": "interiors", "amount": "45000"}),
			op("cost_plan.add_line", map[string]any{"label": "Stone", "section": "construction", "system": "interiors"}),
			op("cost_plan.add_line", map[string]any{"label": "Tapware", "section": "construction", "system": "interiors"}))
	})
	planPath := "/api/projects/" + project + "/cost-plan"
	before := withoutVersions(t, m, planPath)
	events := m.chat(t, project, "", "Add lines for joinery, stone and tapware to Fitout. Joinery is $45,000.")
	if got := eventKinds(events); got != "tool change answer done" {
		t.Fatalf("events: %s", got)
	}
	done := events[len(events)-1]
	if done.Rounds != 1 || model.calls() != 1 || done.ConversationID == "" || done.CacheRead != 6000 {
		t.Fatalf("done %+v, model calls %d", done, model.calls())
	}
	card := cardOf(t, events, "change")
	if len(card.Card.Lines) != 3 || card.Card.Undo != "change" || !strings.Contains(card.Card.Lines[0], "$45,000") {
		t.Fatalf("card %+v", card.Card)
	}
	p := planOf(t, m, project)
	if len(p.Items) != 3 || p.Items[0].SystemID != "interiors" {
		t.Fatalf("plan has %d lines", len(p.Items))
	}
	for _, i := range p.Items {
		if known := i.Values["budget"].Amount != nil; known != (i.Label == "Joinery") {
			t.Errorf("%s: budget known = %v", i.Label, known)
		}
	}
	// The request: the operation list is in the cached system text, the
	// outline and the message in the user turn, and the tools are fixed.
	sent := model.sent[0]
	if !strings.Contains(sent.System, "cost_plan.add_line") || !strings.Contains(sent.System, "Area guide: program") || len(sent.Tools) != 7 {
		t.Fatalf("system prompt or tools are wrong (%d tools)", len(sent.Tools))
	}
	if !strings.Contains(string(sent.History[0]), "project_outline") || !strings.Contains(string(sent.History[0]), "interiors (") {
		t.Fatalf("the outline is missing: %s", sent.History[0])
	}
	// The card's Undo puts everything back.
	m.undo(t, project, card.Card.ChangeID, http.StatusOK)
	if after := withoutVersions(t, m, planPath); !reflect.DeepEqual(before, after) {
		t.Fatal("undo did not restore the plan")
	}
	// The next turn carries what was said before, and the short ids.
	events = m.chat(t, project, done.ConversationID, "And a contingency line called Latent conditions")
	second := model.sent[1]
	if len(second.History) != 3 || !strings.Contains(string(second.History[1]), "Applied") {
		t.Fatalf("history of the second turn: %d messages", len(second.History))
	}
	var view struct {
		Available bool `json:"available"`
		Turns     []struct {
			Text string `json:"text"`
		} `json:"turns"`
	}
	m.getJSON(t, "/api/projects/"+project+"/chat", &view)
	if !view.Available || len(view.Turns) != 2 || view.Turns[0].Text == "" {
		t.Fatalf("chat view %+v", view)
	}
	_ = events
}

// Worked cases 3 and 1: profile values, and five Program rows of two weeks.
func TestChatProfileAndProgramCommands(t *testing.T) {
	stage := ""
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("profile", true,
				op("profile.set_value", map[string]any{"key": "plan.site_area", "value": "612"}),
				op("profile.set_value", map[string]any{"key": "proj.zone", "value": "R3"}))
		}
		var calls []map[string]any
		for i := 1; i <= 5; i++ {
			calls = append(calls, op("program.edit", map[string]any{"op": "add", "parent_id": stage[:8], "kind": "activity", "label": fmt.Sprint("DA drawings ", i), "duration": "2 weeks"}))
		}
		return applyCall("program", true, calls...)
	})
	events := m.chat(t, project, "", "Set the site area to 612 square metres and the zone to R3")
	if card := cardOf(t, events, "change"); len(card.Card.Lines) != 2 || !strings.Contains(card.Card.Lines[0], "Site area") {
		t.Fatalf("card %+v", card.Card)
	}
	base := "/api/projects/" + project + "/program"
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, nil)
	var view programView
	m.getJSON(t, base, &view)
	for _, a := range view.Activities {
		if a.Kind == "stage" && stage == "" {
			stage = a.ID
		}
	}
	events = m.chat(t, project, "", "Add five activities under Design for the DA drawings, two weeks each")
	card := cardOf(t, events, "change")
	if card.Card.Undo != "program" || len(card.Card.Lines) != 5 || !strings.Contains(card.Card.Lines[0], "10 working days") || model.calls() != 2 {
		t.Fatalf("card %+v, calls %d", card.Card, model.calls())
	}
	// The outline gave the model the stage's short id.
	if !strings.Contains(string(model.sent[1].History[0]), stage[:8]) {
		t.Fatal("the outline does not carry the stage id")
	}
}

// Review change 2: a figure nobody gave is refused, and the plan is untouched.
func TestChatRefusesAFigureNobodyGave(t *testing.T) {
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("cost_plan", true, op("cost_plan.add_line", map[string]any{"label": "Joinery", "section": "construction", "system": "interiors", "amount": "54000"}))
		}
		last := string(conv.History[len(conv.History)-1])
		if !strings.Contains(last, "Say the figure") || !strings.Contains(last, `"is_error":true`) {
			t.Errorf("the refusal was not fed back: %s", last)
		}
		return llm.Turn{Text: "What is the joinery amount?"}
	})
	events := m.chat(t, project, "", "Add a joinery line to Fitout for about what we discussed")
	if len(planOf(t, m, project).Items) != 0 || model.calls() != 2 {
		t.Fatalf("a line was added, or the model was not asked again (%d calls)", model.calls())
	}
	last := events[len(events)-2]
	if last.Type != "answer" || !last.Answer.Question {
		t.Fatalf("events %s, answer %+v", eventKinds(events), last.Answer)
	}
}

// Review change 4: what the PM vouches for is prepared and waits for a click.
func TestChatPreparesWhatNeedsAClick(t *testing.T) {
	a, m, project, _ := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("cost_plan", true, op("cost_plan.add_line", map[string]any{"label": "DA fees", "section": "authority", "amount": "8400"}))
		}
		return applyCall("cost_plan", true, op("cost_plan.freeze", map[string]any{}))
	})
	m.chat(t, project, "", "Add an authority line DA fees $8,400")
	events := m.chat(t, project, "", "Freeze the cost plan")
	card := cardOf(t, events, "prepared")
	if card.Card.Prepared == "" || card.Card.Why == "" || planOf(t, m, project).Status != "draft" {
		t.Fatalf("card %+v", card.Card)
	}
	path := "/api/projects/" + project + "/chat/prepared/" + card.Card.Prepared
	// Another member of another org cannot click it.
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodPost, path+"/apply", []byte(`{}`)); got != http.StatusNotFound {
		t.Fatalf("another org's click = %d", got)
	}
	var applied struct {
		Kind  string   `json:"kind"`
		Lines []string `json:"lines"`
	}
	m.call(t, http.MethodPost, path+"/apply", []byte(`{}`), http.StatusOK, &applied)
	var p costs.Plan
	m.getJSON(t, "/api/projects/"+project+"/cost-plan", &p)
	if applied.Kind != "change" || len(applied.Lines) != 1 || p.BaselineVersionID == "" {
		t.Fatalf("applied %+v, baseline %q", applied, p.BaselineVersionID)
	}
	if got := m.status(t, http.MethodPost, path+"/apply", []byte(`{}`)); got != http.StatusConflict {
		t.Fatalf("a second click = %d", got)
	}
}

func TestChatOffAndIsolation(t *testing.T) {
	model := &scriptModel{script: func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "hello"} }}
	a := newApp(t, withProfile(t), withChat(t, model))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Chat off")
	// The org's switch is off: a plain message, no model call, nothing changed.
	events := m.chat(t, project, "", "Add a line")
	if events[0].Type != "error" || !strings.Contains(events[0].Text, "switch it on") || model.calls() != 0 {
		t.Fatalf("events %+v calls %d", events, model.calls())
	}
	m.modelSwitch(t, true, http.StatusOK)
	// A message that is exactly a known read goes to the screen with no model call.
	events = m.chat(t, project, "", "Show the cost plan.")
	if events[0].Type != "go" || events[0].Text != "costs" || model.calls() != 0 {
		t.Fatalf("events %+v calls %d", events, model.calls())
	}
	events = m.chat(t, project, "", "Hello")
	conversation := events[len(events)-1].ConversationID
	if model.calls() != 1 || conversation == "" {
		t.Fatalf("calls %d", model.calls())
	}
	// Another org cannot post to the project; a made-up conversation is not found.
	other := a.member(t, newUUID(t))
	body, _ := json.Marshal(map[string]string{"text": "Hello"})
	if got := other.status(t, http.MethodPost, "/api/projects/"+project+"/chat", body); got != http.StatusNotFound {
		t.Fatalf("another org = %d", got)
	}
	theirs := other.createProject(t, "Theirs")
	body, _ = json.Marshal(map[string]string{"text": "Hello", "conversation_id": conversation})
	if got := other.status(t, http.MethodPost, "/api/projects/"+theirs+"/chat", body); got != http.StatusNotFound {
		t.Fatalf("another org using this conversation = %d", got)
	}
	// With no chat configured the route still answers plainly.
	bare := newApp(t, withProfile(t))
	bm := bare.member(t, newUUID(t))
	events = bm.chat(t, bm.createProject(t, "No chat"), "", "Hello")
	if events[0].Type != "error" {
		t.Fatalf("events %+v", events)
	}
}
