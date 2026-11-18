// Package agent is the chat loop: it calls the model with a fixed set of
// tools, runs the tools the model asks for, and decides in Go what a turn
// may do (docs/plans/2026-11-15-chat-agent.md). It knows nothing about the
// store or HTTP: the tools are behind Toolbox, the model behind Model.
//
// Only the chat route may import this package. No screen read or write
// reaches the model (AGENTS.md rule 3); TestOnlyChatReachesTheModel in
// internal/httpapi holds that.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"sitewise/internal/figures"
	"sitewise/internal/llm"
)

// Model is the one call the loop makes.
type Model interface {
	Converse(ctx context.Context, conv llm.Conversation, onText func(string)) (llm.Turn, error)
}

// Limits of one turn (plan Task 7, step 5).
const (
	MaxRounds   = 8
	TurnTimeout = 60 * time.Second
	// maxTokens caps one reply. It bounds a runaway answer; it does not
	// slow a short one, so one value serves a command and an answer.
	maxTokens = 1200
)

// State is what Go knows about the turn so far. Tools read it; the loop
// alone writes it, between rounds.
type State struct {
	// Book holds every figure the PM typed in this conversation and every
	// figure a tool returned this turn. An operation's amount, date or
	// duration must be in it (review change 2).
	Book figures.Book
	// Seen is the book plus the outline: what an answer may repeat without
	// it being the model's own figure.
	Seen figures.Book
	// Outside is true once a tool result carried text that came from
	// outside: a document, a message, a firm's tender. From then on this
	// turn's changes wait for the PM's click. Why says where it came from.
	Outside bool
	Why     string
	Today   time.Time
	// Asked is the PM's message of this turn. Tools read it to keep to the
	// document or the firm the PM named.
	Asked string
	// Passages are the passages read this turn, by id, for the quote check.
	Passages map[string]Passage
	// Facts are the figures tools returned this turn, by tag (Task 8).
	Facts map[string]Fact
	// Coverage is Go's line saying what a document search covered.
	Coverage string
}

// Passage is one stored passage a tool returned.
type Passage struct {
	ID       string `json:"id"`
	Document string `json:"document"` // title
	DocID    string `json:"document_id"`
	Page     int    `json:"page"`
	Text     string `json:"-"`
	Note     string `json:"note,omitempty"` // "rev B", "superseded by rev B", a firm's name
}

// Fact is one figure Go returned, with the row it belongs to.
type Fact struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Source string `json:"source"` // "Cost Plan", "calc", …
	// Plain is the figure as a bare number ("120000"), when it is one. An
	// answer that types it without its tag is tied back to this row by Go.
	Plain string `json:"-"`
}

// Output is what one tool call returned.
type Output struct {
	Text    string
	IsError bool
	// Outside, when not empty, says the result carried outside text and
	// where from ("the Arborist report").
	Outside  string
	Passages []Passage
	Facts    map[string]Fact
	Coverage string
	Cards    []Card
	// Views are screen moves for the strip: they change no record, so they
	// carry no Undo and never wait for a click.
	Views []View
	// Final: the tool did everything the PM asked and its card says so, so
	// no further model call is needed (speed rule 1).
	Final bool
	// Refused: an apply was refused. One refusal is fed back; a second ends
	// the turn with the refusal in plain words.
	Refused bool
}

// Card is what the strip shows for a change: applied, with Undo, or
// prepared, with Apply.
type Card struct {
	Kind     string   `json:"kind"` // change | prepared
	Area     string   `json:"area"`
	Lines    []string `json:"lines"`
	ChangeID string   `json:"change_id,omitempty"`
	Undo     string   `json:"undo,omitempty"` // change | program | none
	Prepared string   `json:"prepared_id,omitempty"`
	Why      string   `json:"why,omitempty"`
	// Refs are the short row ids the card's operations named, so a later
	// turn can say "those" (chat agency plan, Task 5).
	Refs []string `json:"refs,omitempty"`
	// ProgramVersion is the Program's version just after a Program change,
	// so an undo typed later can tell whether anything has moved since.
	ProgramVersion int64 `json:"program_version,omitempty"`
}

// View is one move of the screen: show a table, select or show only some of
// its rows, clear that, or open a row or a dialog. Ids are full ids Go has
// checked against the project. Line says what was done, in plain words.
type View struct {
	Do     string            `json:"do"` // show | open | select | only | clear
	Table  string            `json:"table,omitempty"`
	IDs    []string          `json:"ids,omitempty"`
	Dialog string            `json:"dialog,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	Line   string            `json:"line"`
	// Filter names the rule that chose the ids ("no_budget"), and Shown is
	// its words for the strip's "Showing N cost lines with no budget" line.
	// Both are empty for rows picked by id.
	Filter string `json:"filter,omitempty"`
	Shown  string `json:"shown,omitempty"`
}

// Toolbox is the fixed set of tools.
type Toolbox interface {
	Tools() []llm.Tool
	// Writes reports whether the tool changes records. Such calls run after
	// the round's reads, one at a time.
	Writes(name string) bool
	Run(ctx context.Context, st *State, call llm.ToolCall) Output
}

// Request is one turn.
type Request struct {
	Model  string
	System string // the cached prefix: rules, operation list, area guides
	// History is earlier turns as provider messages (plain text only).
	History []json.RawMessage
	// Typed is every message the PM typed in this conversation, this one
	// last. Figures in any of them may be used in an operation.
	Typed   []string
	Outline string // Go's compact outline of the project, with ids
	// Screen is Go's one line about what is on the PM's screen: the view,
	// the open row and the selection, as ids and counts only.
	Screen string
	Think  bool // let the model use hidden thinking (see llm.Conversation)
	Today  time.Time
	// Prefetch is reads Go runs before the model is asked, because the
	// PM's words already say which area a question is about. Their results
	// go in the first turn, so the answer takes one model call, not two.
	Prefetch []Prefetch
}

// Prefetch is one read done ahead of the model.
type Prefetch struct {
	Area string // named in the turn so the model knows what it holds
	Call llm.ToolCall
}

// Event is one thing sent to the strip while a turn runs.
type Event struct {
	Type string `json:"type"` // text | tool | change | prepared | view | answer | done | error
	Text string `json:"text,omitempty"`
	Tool string `json:"tool,omitempty"`
	Card *Card  `json:"card,omitempty"`
	View *View  `json:"view,omitempty"`
	// On an answer: the checked text and whether it is a question.
	Answer *Answer `json:"answer,omitempty"`
}

// Result is how a turn ended.
type Result struct {
	Answer   Answer
	Cards    []Card
	Messages []json.RawMessage // this turn's provider messages, for the store
	Rounds   int
	Tools    []string
	Outside  bool
	Stopped  string // why the loop stopped early, or ""
	Err      error  // the model failed; nothing more was done
	// Usage summed over the rounds.
	InTokens, OutTokens, CacheRead, CacheWrite int64
	FirstToken                                 time.Duration // of the first round
	ModelTime                                  time.Duration
	State                                      *State
}

// UserMessage is the text of the PM's turn as the model sees it: Go's
// outline first, then the message. Both sit after the cache breakpoint.
func UserMessage(outline, typed string) string {
	return userMessage(outline, "", typed)
}

// userMessage puts what Go read ahead between the outline and the message.
func userMessage(outline, read, typed string) string {
	return "<project_outline>\n" + strings.TrimSpace(outline) + "\n</project_outline>\n\n" + read + "<pm_message>\n" + strings.TrimSpace(typed) + "\n</pm_message>"
}

// onScreen wraps the screen line. It sits in the turn's own message, after
// the cache breakpoint, so a change of selection never changes the prefix.
func onScreen(line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	return "<on_screen>\n" + strings.TrimSpace(line) + "\n</on_screen>\n\n"
}

// prefetch runs the reads Go was asked to do ahead and returns them as the
// model will see them. A result is merged into the state exactly as if the
// model had asked for it. A failed read is left out: the model can still
// ask. A write is never run here.
func prefetch(ctx context.Context, box Toolbox, st *State, reads []Prefetch) string {
	outs := make([]Output, len(reads))
	var wg sync.WaitGroup
	for i, p := range reads {
		if box.Writes(p.Call.Name) {
			outs[i].IsError = true
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			outs[i] = box.Run(ctx, st, p.Call)
		}()
	}
	wg.Wait()
	var b strings.Builder
	for i, out := range outs {
		if out.IsError || strings.TrimSpace(out.Text) == "" {
			continue
		}
		merge(st, out)
		b.WriteString("<already_read area=\"" + reads[i].Area + "\">\n" + strings.TrimSpace(out.Text) + "\n</already_read>\n\n")
	}
	return b.String()
}

// Run carries out one turn. emit is called from this goroutine only.
func Run(ctx context.Context, m Model, box Toolbox, req Request, emit func(Event)) Result {
	if emit == nil {
		emit = func(Event) {}
	}
	ctx, cancel := context.WithTimeout(ctx, TurnTimeout)
	defer cancel()
	st := &State{Today: req.Today, Passages: map[string]Passage{}, Facts: map[string]Fact{}}
	typed := ""
	for _, t := range req.Typed {
		st.Book.Add(t)
		st.Seen.Add(t)
		typed = t
	}
	// The outline is SiteWise's own records: a figure from it may be
	// repeated in an answer. It does not go in the book for operations,
	// because it is full of small numbers (dates, counts) that would make
	// any small figure look given.
	st.Seen.Add(req.Outline)
	st.Asked = typed
	res := Result{State: st}
	history := append([]json.RawMessage{}, req.History...)
	add := func(msg json.RawMessage) {
		history = append(history, msg)
		res.Messages = append(res.Messages, msg)
	}
	add(llm.UserText(userMessage(req.Outline, onScreen(req.Screen)+prefetch(ctx, box, st, req.Prefetch), typed)))
	tools := box.Tools()
	refusals := 0
	var said strings.Builder
	for {
		if res.Rounds == MaxRounds {
			res.Stopped = "I stopped after eight steps."
			break
		}
		res.Rounds++
		said.Reset()
		turn, err := m.Converse(ctx, llm.Conversation{Name: "chat", Model: req.Model, System: req.System, Tools: tools,
			History: history, MaxTokens: maxTokens, Timeout: TurnTimeout, Think: req.Think}, func(s string) {
			said.WriteString(s)
			emit(Event{Type: "text", Text: s})
		})
		res.InTokens, res.OutTokens = res.InTokens+turn.InTokens, res.OutTokens+turn.OutTokens
		res.CacheRead, res.CacheWrite = res.CacheRead+turn.CacheRead, res.CacheWrite+turn.CacheWrite
		res.ModelTime += turn.Latency
		if res.Rounds == 1 {
			res.FirstToken = turn.FirstToken
		}
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && res.Rounds > 1 {
				res.Stopped = "I ran out of time."
				break
			}
			res.Err = err
			break
		}
		if len(turn.Message) > 0 {
			add(turn.Message)
		}
		if len(turn.Calls) == 0 {
			res.Answer.Text = strings.TrimSpace(turn.Text)
			if turn.Truncated {
				res.Stopped = "The answer was cut short."
			}
			break
		}
		outs := runRound(ctx, box, st, turn.Calls, emit, &res)
		results := make([]llm.ToolResult, len(turn.Calls))
		final, failed := false, false
		lastRefusal := ""
		for i, out := range outs {
			results[i] = llm.ToolResult{CallID: turn.Calls[i].ID, Text: out.Text, IsError: out.IsError}
			final = final || out.Final
			failed = failed || out.IsError
			if out.Refused {
				refusals++
				lastRefusal = out.Text
			}
			for j := range out.Cards {
				card := out.Cards[j]
				res.Cards = append(res.Cards, card)
				emit(Event{Type: card.Kind, Card: &card})
			}
			for j := range out.Views {
				v := out.Views[j]
				emit(Event{Type: "view", View: &v})
			}
		}
		add(llm.ToolResults(results))
		if final && !failed {
			// Go's cards are the record of what was done. Words the model
			// wrote before the call are its working, not an answer: they
			// are dropped.
			res.Answer.Text = ""
			break
		}
		if refusals >= 2 {
			res.Answer.Text = "I couldn't make that change. " + lastRefusal
			break
		}
		if ctx.Err() != nil {
			res.Stopped = "I ran out of time."
			break
		}
	}
	res.Outside = st.Outside
	if res.Stopped != "" {
		res.Answer.Text = strings.TrimSpace(res.Stopped + " " + doneSoFar(res.Cards) + " " + res.Answer.Text)
	}
	if res.Err == nil {
		res.Answer = Check(res.Answer.Text, st)
		// A reply that asks something and changed nothing is a question
		// back, wherever in the reply the question sits.
		res.Answer.Question = len(res.Cards) == 0 && strings.Contains(res.Answer.Text, "?")
		emit(Event{Type: "answer", Answer: &res.Answer})
	}
	return res
}

// runRound runs one round's tool calls: the reads together, then the writes
// in order. A call whose input failed its schema is answered with the
// error and never run.
func runRound(ctx context.Context, box Toolbox, st *State, calls []llm.ToolCall, emit func(Event), res *Result) []Output {
	outs := make([]Output, len(calls))
	var wg sync.WaitGroup
	for i, call := range calls {
		res.Tools = append(res.Tools, call.Name)
		emit(Event{Type: "tool", Tool: call.Name})
		if call.Err != "" {
			outs[i] = Output{Text: call.Err, IsError: true}
			continue
		}
		if box.Writes(call.Name) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			outs[i] = box.Run(ctx, st, call)
		}()
	}
	wg.Wait()
	// The reads' results are in the book before any write is judged.
	for i, call := range calls {
		if call.Err == "" && !box.Writes(call.Name) {
			merge(st, outs[i])
		}
	}
	// The model sometimes sends the same write twice in one round. Running
	// both would make every row twice, so a write whose input is the same
	// as an earlier one in the round is answered, not run.
	ran := map[string]bool{}
	for i, call := range calls {
		if call.Err == "" && box.Writes(call.Name) {
			key := call.Name + "\x00" + string(call.Input)
			if ran[key] {
				outs[i] = Output{Text: "This is the same call as an earlier one in this step. It was run once; nothing was done twice."}
				continue
			}
			ran[key] = true
			outs[i] = box.Run(ctx, st, call)
			merge(st, outs[i])
		}
	}
	return outs
}

func merge(st *State, out Output) {
	if out.IsError {
		return
	}
	st.Book.Add(out.Text)
	st.Seen.Add(out.Text)
	if out.Outside != "" && !st.Outside {
		st.Outside, st.Why = true, out.Outside
	}
	for _, p := range out.Passages {
		st.Passages[p.ID] = p
	}
	for tag, f := range out.Facts {
		st.Facts[tag] = f
	}
	if out.Coverage != "" {
		st.Coverage = out.Coverage
	}
}

func doneSoFar(cards []Card) string {
	applied, prepared := 0, 0
	for _, c := range cards {
		if c.Kind == "change" {
			applied++
		} else {
			prepared++
		}
	}
	switch {
	case applied == 0 && prepared == 0:
		return "Nothing was changed."
	case prepared == 0:
		return "The changes shown above were made; anything else was not."
	case applied == 0:
		return "The changes shown above wait for your click; nothing was changed."
	}
	return "The changes shown above were made or wait for your click; anything else was not."
}

// Unavailable is the plain message for a model failure (design 6.4). A
// failure after a change was made says the change stands.
func Unavailable(err error, cards []Card) string {
	if len(cards) > 0 {
		return "Chat stopped part way. " + doneSoFar(cards)
	}
	switch {
	case errors.Is(err, llm.ErrRefused):
		return "No change was made."
	case strings.Contains(err.Error(), "429"):
		return "Chat is busy, try again in a minute. Nothing was changed."
	}
	return "Chat is unavailable. Nothing was changed."
}
