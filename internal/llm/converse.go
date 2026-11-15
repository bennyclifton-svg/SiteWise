package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/packages/param"
)

// Tool is one tool the model may call. Schema is its JSON Schema, an object.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
}

// Conversation is one model call in a tool loop. It has no field for a
// user, an org or a project, so none can be sent.
type Conversation struct {
	Name  string // for the log, e.g. "chat"
	Model string
	// System and Tools are the stable prefix: one cache breakpoint sits
	// after them. Nothing that changes per request belongs in either.
	System string
	Tools  []Tool
	// History is the messages so far, each as the provider's own JSON
	// ({"role":…,"content":[…]}), kept whole so tool-use blocks survive.
	History   []json.RawMessage
	MaxTokens int64
	Timeout   time.Duration // zero means one minute
	// Warm sends the prefix with max_tokens 0 so the next real call reads
	// it from the cache. Nothing is generated.
	Warm bool
}

// ToolCall is one call the model asked for. Err is set when the input is
// not valid for the tool; such a call must be answered with a tool error
// and never run.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
	Err   string
}

// Turn is the model's reply to one Conversation.
type Turn struct {
	// Message is the assistant message as the provider returned it, to
	// append to History unchanged.
	Message    json.RawMessage
	Text       string
	Calls      []ToolCall
	StopReason string
	Truncated  bool // stop_reason was max_tokens
	InTokens   int64
	OutTokens  int64
	CacheRead  int64
	CacheWrite int64
	FirstToken time.Duration // call start to the first streamed text or tool input
	Latency    time.Duration
}

// Talker is what the agent loop depends on, so tests and benches stand in.
type Talker interface {
	On() bool
	Converse(ctx context.Context, conv Conversation, onText func(string)) (Turn, error)
}

func newWithTalk(opts Options, talk func(context.Context, Conversation, func(string)) (Turn, error)) *Client {
	c := newWithSend(opts, nil)
	c.talk = talk
	return c
}

// NewScripted returns a client whose model is the given function. Benches,
// browser tests and the eval's replay use it; it shares the real client's
// slots, breaker, timeout and log line.
func NewScripted(opts Options, talk func(context.Context, Conversation, func(string)) (Turn, error)) *Client {
	return newWithTalk(opts, talk)
}

// Converse sends one round of a tool conversation. Text is passed to onText
// as it streams. It shares Ask's slots, breaker, timeout and log line.
func (c *Client) Converse(ctx context.Context, conv Conversation, onText func(string)) (Turn, error) {
	if c == nil || c.talk == nil {
		return Turn{}, ErrOff
	}
	if onText == nil {
		onText = func(string) {}
	}
	release, err := c.enter(ctx)
	if err != nil {
		return Turn{}, err
	}
	defer release()
	timeout := conv.Timeout
	if timeout <= 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	began := c.now()
	turn, err := c.talk(ctx, conv, onText)
	turn.Latency = c.now().Sub(began)
	if err == nil && turn.StopReason == "refusal" {
		err = ErrRefused
	}
	c.settle(err)
	class := "ok"
	if err != nil {
		class = errorClass(err)
	}
	// Never log the conversation: it holds the project's figures and text.
	c.log.Info("llm call", "name", conv.Name, "model", conv.Model, "class", class, "stop", turn.StopReason,
		"latency_ms", turn.Latency.Milliseconds(), "first_token_ms", turn.FirstToken.Milliseconds(),
		"in_tokens", turn.InTokens, "out_tokens", turn.OutTokens,
		"cache_read_tokens", turn.CacheRead, "cache_write_tokens", turn.CacheWrite, "tool_calls", len(turn.Calls))
	if err != nil {
		return turn, err
	}
	turn.Truncated = turn.StopReason == "max_tokens"
	schemas := map[string]map[string]any{}
	for _, t := range conv.Tools {
		schemas[t.Name] = t.Schema
	}
	for i := range turn.Calls {
		call := &turn.Calls[i]
		schema, known := schemas[call.Name]
		switch {
		case !known:
			call.Err = "there is no tool named " + call.Name
		case turn.Truncated:
			// A cut-off input can still parse; it is never run.
			call.Err = "the call was cut off before it was complete; send it again, shorter"
		default:
			if err := CheckInput(schema, call.Input); err != nil {
				call.Err = "invalid input: " + err.Error()
			}
		}
	}
	return turn, nil
}

// CheckInput checks a tool input against the subset of JSON Schema the
// tools use: type, properties, required, additionalProperties false, items
// and enum. Streamed tool inputs are not validated by the provider, so Go
// checks every one before it is run.
func CheckInput(schema map[string]any, input json.RawMessage) error {
	var v any
	dec := json.NewDecoder(strings.NewReader(string(input)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return errors.New("not valid JSON")
	}
	if dec.More() {
		return errors.New("not valid JSON")
	}
	return checkValue(schema, v, "input")
}

func checkValue(schema map[string]any, v any, at string) error {
	if enum := list(schema["enum"]); len(enum) > 0 {
		for _, e := range enum {
			if e == v {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of %v", at, enum)
	}
	switch schema["type"] {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s must be a string", at)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s must be true or false", at)
		}
	case "number", "integer":
		n, ok := v.(json.Number)
		if !ok {
			return fmt.Errorf("%s must be a number", at)
		}
		if f, err := n.Float64(); err != nil || (schema["type"] == "integer" && f != math.Trunc(f)) {
			return fmt.Errorf("%s must be a whole number", at)
		}
	case "array":
		items, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s must be a list", at)
		}
		if each, ok := schema["items"].(map[string]any); ok {
			for i, item := range items {
				if err := checkValue(each, item, fmt.Sprintf("%s[%d]", at, i)); err != nil {
					return err
				}
			}
		}
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", at)
		}
		props, _ := schema["properties"].(map[string]any)
		for _, name := range list(schema["required"]) {
			if _, ok := obj[name.(string)]; !ok {
				return fmt.Errorf("%s.%s is required", at, name)
			}
		}
		for name, value := range obj {
			sub, ok := props[name].(map[string]any)
			if !ok {
				if schema["additionalProperties"] == false {
					return fmt.Errorf("%s.%s is not a field", at, name)
				}
				continue
			}
			if err := checkValue(sub, value, at+"."+name); err != nil {
				return err
			}
		}
	}
	return nil
}

// list reads a schema list written either as []any or as []string.
func list(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case []string:
		out := make([]any, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	}
	return nil
}

// thinkingOff is the request setting that turns thinking off. Sonnet 5.5
// rejects "disabled" and takes "between_tools"; the others take "disabled"
// at low effort.
func thinkingOff(model string) anthropic.ThinkingConfigParamUnion {
	if strings.HasPrefix(model, "claude-sonnet-5-5") {
		return anthropic.ThinkingConfigParamUnion{OfBetweenTools: &anthropic.ThinkingConfigBetweenToolsParam{}}
	}
	return anthropic.ThinkingConfigParamUnion{OfDisabled: &anthropic.ThinkingConfigDisabledParam{}}
}

func conversationParams(conv Conversation) anthropic.MessageNewParams {
	tools := make([]anthropic.ToolUnionParam, 0, len(conv.Tools))
	for _, t := range conv.Tools {
		extra := map[string]any{}
		for k, v := range t.Schema {
			if k != "type" && k != "properties" && k != "required" {
				extra[k] = v
			}
		}
		required := make([]string, 0)
		for _, r := range list(t.Schema["required"]) {
			required = append(required, r.(string))
		}
		tool := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: t.Schema["properties"], Required: required, ExtraFields: extra},
		}
		if !conv.Warm {
			// Inputs stream as they are written; Converse validates them.
			tool.EagerInputStreaming = anthropic.Bool(true)
		}
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &tool})
	}
	messages := make([]anthropic.MessageParam, 0, len(conv.History))
	for _, m := range conv.History {
		// The history goes back exactly as the provider wrote it.
		messages = append(messages, param.Override[anthropic.MessageParam](m))
	}
	return anthropic.MessageNewParams{
		Model:     anthropic.Model(conv.Model),
		MaxTokens: conv.MaxTokens,
		// One breakpoint after the tools and the system text: the stable
		// prefix every turn of every project shares.
		System:   []anthropic.TextBlockParam{{Text: conv.System, CacheControl: anthropic.NewCacheControlEphemeralParam()}},
		Tools:    tools,
		Messages: messages,
		// Thinking off and low effort: the fastest set-up (plan, speed rule 5).
		Thinking:     thinkingOff(conv.Model),
		OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow},
	}
}

func sdkTalk(client anthropic.Client) func(context.Context, Conversation, func(string)) (Turn, error) {
	return func(ctx context.Context, conv Conversation, onText func(string)) (Turn, error) {
		params := conversationParams(conv)
		began := time.Now()
		if conv.Warm {
			params.MaxTokens = 0
			msg, err := client.Messages.New(ctx, params)
			if err != nil {
				return Turn{}, err
			}
			return usage(Turn{StopReason: string(msg.StopReason)}, *msg), nil
		}
		stream := client.Messages.NewStreaming(ctx, params)
		msg := anthropic.Message{}
		var out Turn
		for stream.Next() {
			event := stream.Current()
			if err := msg.Accumulate(event); err != nil {
				return Turn{}, err
			}
			if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if out.FirstToken == 0 {
					out.FirstToken = time.Since(began)
				}
				if text, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
					onText(text.Text)
				}
			}
		}
		if err := stream.Err(); err != nil {
			return Turn{}, err
		}
		out = usage(out, msg)
		out.StopReason = string(msg.StopReason)
		for _, block := range msg.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				out.Text += b.Text
			case anthropic.ToolUseBlock:
				out.Calls = append(out.Calls, ToolCall{ID: b.ID, Name: b.Name, Input: json.RawMessage(b.JSON.Input.Raw())})
			}
		}
		raw, err := json.Marshal(msg.ToParam())
		if err != nil {
			return Turn{}, err
		}
		out.Message = raw
		return out, nil
	}
}

func usage(t Turn, msg anthropic.Message) Turn {
	t.InTokens, t.OutTokens = msg.Usage.InputTokens, msg.Usage.OutputTokens
	t.CacheRead, t.CacheWrite = msg.Usage.CacheReadInputTokens, msg.Usage.CacheCreationInputTokens
	return t
}

// UserText is a user message holding one text block, in the provider's JSON.
func UserText(text string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"role": "user", "content": []map[string]any{{"type": "text", "text": text}}})
	return raw
}

// ToolResult is the answer to one tool call.
type ToolResult struct {
	CallID  string
	Text    string
	IsError bool
}

// ToolResults is the one user message that answers every call of a turn.
// Splitting results across messages teaches the model to stop calling tools
// in parallel, so they always travel together.
func ToolResults(results []ToolResult) json.RawMessage {
	blocks := make([]map[string]any, 0, len(results))
	for _, r := range results {
		b := map[string]any{"type": "tool_result", "tool_use_id": r.CallID, "content": r.Text}
		if r.IsError {
			b["is_error"] = true
		}
		blocks = append(blocks, b)
	}
	raw, _ := json.Marshal(map[string]any{"role": "user", "content": blocks})
	return raw
}
