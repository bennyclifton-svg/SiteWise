package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

var applyTool = Tool{Name: "apply", Description: "apply operations", Schema: map[string]any{
	"type": "object", "additionalProperties": false, "required": []string{"area", "ops"},
	"properties": map[string]any{
		"area":  map[string]any{"type": "string", "enum": []string{"cost_plan", "profile"}},
		"ops":   map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []string{"op"}, "properties": map[string]any{"op": map[string]any{"type": "string"}}}},
		"final": map[string]any{"type": "boolean"},
	},
}}

func talker(turn Turn, err error, said ...string) func(context.Context, Conversation, func(string)) (Turn, error) {
	return func(_ context.Context, _ Conversation, onText func(string)) (Turn, error) {
		for _, s := range said {
			onText(s)
		}
		return turn, err
	}
}

func TestConverseTextOnly(t *testing.T) {
	c := newWithTalk(Options{}, talker(Turn{Text: "Hello there", StopReason: "end_turn"}, nil, "Hello ", "there"))
	var heard []string
	turn, err := c.Converse(context.Background(), Conversation{Model: "m", Tools: []Tool{applyTool}}, func(s string) { heard = append(heard, s) })
	if err != nil || turn.Text != "Hello there" || len(turn.Calls) != 0 || strings.Join(heard, "") != "Hello there" {
		t.Fatalf("turn %+v err %v heard %v", turn, err, heard)
	}
}

func TestConverseOneAndTwoToolCalls(t *testing.T) {
	one := ToolCall{ID: "a", Name: "apply", Input: json.RawMessage(`{"area":"cost_plan","ops":[{"op":"cost_plan.add_line"}],"final":true}`)}
	two := ToolCall{ID: "b", Name: "apply", Input: json.RawMessage(`{"area":"profile","ops":[]}`)}
	c := newWithTalk(Options{}, talker(Turn{Calls: []ToolCall{one}, StopReason: "tool_use"}, nil))
	turn, err := c.Converse(context.Background(), Conversation{Tools: []Tool{applyTool}}, nil)
	if err != nil || len(turn.Calls) != 1 || turn.Calls[0].Err != "" {
		t.Fatalf("turn %+v err %v", turn, err)
	}
	c = newWithTalk(Options{}, talker(Turn{Calls: []ToolCall{one, two}, StopReason: "tool_use"}, nil))
	turn, err = c.Converse(context.Background(), Conversation{Tools: []Tool{applyTool}}, nil)
	if err != nil || len(turn.Calls) != 2 || turn.Calls[0].Err != "" || turn.Calls[1].Err != "" {
		t.Fatalf("turn %+v err %v", turn, err)
	}
}

// A bad input is not run: it comes back marked, so the loop answers it with a
// tool error.
func TestConverseMalformedToolInput(t *testing.T) {
	bad := []ToolCall{
		{ID: "1", Name: "apply", Input: json.RawMessage(`{"area":"cost_plan","ops":[{"op":`)},         // cut off
		{ID: "2", Name: "apply", Input: json.RawMessage(`{"area":"billing","ops":[]}`)},               // not in the enum
		{ID: "3", Name: "apply", Input: json.RawMessage(`{"area":"profile"}`)},                        // a required field is missing
		{ID: "4", Name: "apply", Input: json.RawMessage(`{"area":"profile","ops":[],"extra":1}`)},     // an unknown field
		{ID: "5", Name: "apply", Input: json.RawMessage(`{"area":"profile","ops":[{"op":7}]}`)},       // a wrong type inside the list
		{ID: "6", Name: "launch", Input: json.RawMessage(`{}`)},                                       // not a tool
		{ID: "7", Name: "apply", Input: json.RawMessage(`{"area":"profile","ops":[],"final":"yes"}`)}, // a wrong type
	}
	c := newWithTalk(Options{}, talker(Turn{Calls: bad, StopReason: "tool_use"}, nil))
	turn, err := c.Converse(context.Background(), Conversation{Tools: []Tool{applyTool}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range turn.Calls {
		if call.Err == "" {
			t.Errorf("call %s was accepted: %s", call.ID, call.Input)
		}
	}
}

func TestConverseRefusalAndMaxTokens(t *testing.T) {
	c := newWithTalk(Options{}, talker(Turn{StopReason: "refusal"}, nil))
	if _, err := c.Converse(context.Background(), Conversation{}, nil); !errors.Is(err, ErrRefused) {
		t.Fatalf("err %v", err)
	}
	// A tool call cut off by max_tokens is never run.
	cut := ToolCall{ID: "a", Name: "apply", Input: json.RawMessage(`{"area":"profile","ops":[]}`)}
	c = newWithTalk(Options{}, talker(Turn{Calls: []ToolCall{cut}, StopReason: "max_tokens"}, nil))
	turn, err := c.Converse(context.Background(), Conversation{Tools: []Tool{applyTool}}, nil)
	if err != nil || !turn.Truncated || turn.Calls[0].Err == "" {
		t.Fatalf("turn %+v err %v", turn, err)
	}
}

func TestConverseReportsCacheReadsAndLogsNoContent(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	c := newWithTalk(Options{Logger: log}, talker(Turn{Text: "SECRET-ANSWER", StopReason: "end_turn", InTokens: 40, OutTokens: 9, CacheRead: 6100, CacheWrite: 12}, nil))
	turn, err := c.Converse(context.Background(), Conversation{Name: "chat", Model: "m", System: "SECRET-SYSTEM"}, nil)
	if err != nil || turn.CacheRead != 6100 || turn.CacheWrite != 12 {
		t.Fatalf("turn %+v err %v", turn, err)
	}
	line := buf.String()
	if !strings.Contains(line, "cache_read_tokens=6100") || strings.Contains(line, "SECRET") {
		t.Fatalf("log line: %s", line)
	}
}

func TestConverseSharesOffAndBreaker(t *testing.T) {
	if _, err := New(Options{}).Converse(context.Background(), Conversation{}, nil); !errors.Is(err, ErrOff) {
		t.Fatalf("err %v", err)
	}
	c := newWithTalk(Options{}, talker(Turn{}, errors.New("boom")))
	for i := 0; i < 5; i++ {
		_, _ = c.Converse(context.Background(), Conversation{}, nil)
	}
	if _, err := c.Converse(context.Background(), Conversation{}, nil); !errors.Is(err, ErrBusy) {
		t.Fatalf("after five failures the breaker is open; err %v", err)
	}
}

func TestCheckInputTypes(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{
		"n": map[string]any{"type": "integer"}, "x": map[string]any{"type": "number"}, "o": map[string]any{"type": "object"},
	}}
	for input, ok := range map[string]bool{`{"n":3}`: true, `{"n":3.5}`: false, `{"x":3.5}`: true, `{"o":{"a":1}}`: true, `{"o":[]}`: false, `[]`: false, `{"n":null}`: false} {
		if err := CheckInput(schema, json.RawMessage(input)); (err == nil) != ok {
			t.Errorf("%s: err %v, want ok=%v", input, err, ok)
		}
	}
}
