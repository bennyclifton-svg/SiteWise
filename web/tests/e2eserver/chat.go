package main

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/llm"
)

// chatScript is the stand-in chat model for the browser tests. It answers
// from the PM's words alone, so no text leaves this process:
//
//	"joinery"       one apply that adds a Joinery line for $45,000, final
//	"freeze"        an apply of cost_plan.freeze, which waits for a click
//	"chat is down"  the model fails
//	"screen DO TABLE NAME[, NAME]"
//	                a view call on the rows the outline names (documents
//	                are listed first with the documents tool)
//	"screen award FIRM"  the award dialog on that firm's column
//	"exclude these" set_lines on the cost lines the screen line selects
//	anything else   a question back
type chatScript struct{}

func (chatScript) Converse(_ context.Context, conv llm.Conversation, onText func(string)) (llm.Turn, error) {
	if conv.Warm {
		return llm.Turn{}, nil
	}
	time.Sleep(100 * time.Millisecond)
	var last struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(conv.History[len(conv.History)-1], &last)
	said, text := "", ""
	for _, c := range last.Content {
		if c.Type == "tool_result" {
			// A documents list asked for by "screen … documents …": select from it.
			if pm := pmMessage(conv); strings.HasPrefix(pm, "screen ") && strings.Contains(toolText(conv), "Filed documents (id |") {
				return screenCall(pm, toolText(conv), "documents")
			}
			return say("Done.", onText), nil
		}
		if _, after, ok := strings.Cut(c.Text, "<pm_message>"); ok {
			said, text = strings.ToLower(after), c.Text
		}
	}
	if pm := pmMessage(conv); strings.HasPrefix(pm, "screen ") {
		if strings.HasPrefix(pm, "screen award ") {
			return awardCall(pm, text)
		}
		if strings.Contains(pm, " documents ") {
			return call("documents", map[string]any{})
		}
		return screenCall(pm, text, "")
	}
	if strings.Contains(said, "exclude these") {
		m := regexp.MustCompile(`Selected: \d+ cost lines? \(([0-9a-f, ]+)\)`).FindStringSubmatch(text)
		if m == nil {
			return say("Which lines do you mean?", onText), nil
		}
		in, _ := json.Marshal(map[string]any{"area": "cost_plan", "final": true, "ops": []map[string]any{{"op": "cost_plan.set_lines", "fields": map[string]any{"lines": strings.Split(m[1], ", "), "excluded": true}}}})
		return turnOf("apply", in), nil
	}
	apply := func(ops ...map[string]any) (llm.Turn, error) {
		in, _ := json.Marshal(map[string]any{"area": "cost_plan", "final": true, "ops": ops})
		msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "tool_use", "id": "call-1", "name": "apply", "input": json.RawMessage(in)}}})
		return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: "apply", Input: in}}, StopReason: "tool_use", Message: msg}, nil
	}
	switch {
	case strings.Contains(said, "chat is down"):
		return llm.Turn{}, errors.New("the stand-in model is down")
	case strings.Contains(said, "joinery"):
		return apply(map[string]any{"op": "cost_plan.add_line", "fields": map[string]any{"label": "Joinery", "section": "construction", "system": "interiors", "amount": "45000"}})
	case strings.Contains(said, "freeze"):
		return apply(map[string]any{"op": "cost_plan.freeze", "fields": map[string]any{}})
	}
	return say("Which line do you mean?", onText), nil
}

// pmMessage is the PM's words of this turn, from the newest user message
// that carries them.
func pmMessage(conv llm.Conversation) string {
	for i := len(conv.History) - 1; i >= 0; i-- {
		var msg struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		_ = json.Unmarshal(conv.History[i], &msg)
		for _, c := range msg.Content {
			if _, after, ok := strings.Cut(c.Text, "<pm_message>"); ok {
				before, _, _ := strings.Cut(after, "</pm_message>")
				return strings.ToLower(strings.TrimSpace(before))
			}
		}
	}
	return ""
}

// toolText is the newest tool result's text.
func toolText(conv llm.Conversation) string {
	var last struct {
		Content []struct {
			Content string `json:"content"`
		} `json:"content"`
	}
	_ = json.Unmarshal(conv.History[len(conv.History)-1], &last)
	out := ""
	for _, c := range last.Content {
		out += c.Content + "\n"
	}
	return out
}

func turnOf(name string, in json.RawMessage) llm.Turn {
	msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "tool_use", "id": "call-1", "name": name, "input": in}}})
	return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: name, Input: in}}, StopReason: "tool_use", Message: msg}
}

func call(name string, input map[string]any) (llm.Turn, error) {
	in, _ := json.Marshal(input)
	return turnOf(name, in), nil
}

// idsNamed finds each name's short id in text: an outline line "id Name…"
// or a documents line "id | Name…".
func idsNamed(text string, names []string) []string {
	var ids []string
	for _, n := range names {
		re := regexp.MustCompile(`(?mi)^\s*([0-9a-f]{8})(?: \|)? ` + regexp.QuoteMeta(strings.TrimSpace(n)) + `\b`)
		if m := re.FindStringSubmatch(text); m != nil {
			ids = append(ids, m[1])
		}
	}
	return ids
}

// screenCall is "screen DO TABLE NAME, NAME" as one final view call.
func screenCall(pm, text, table string) (llm.Turn, error) {
	parts := strings.SplitN(strings.TrimPrefix(pm, "screen "), " ", 3)
	in := map[string]any{"do": parts[0], "final": true}
	if len(parts) > 1 {
		in["table"] = parts[1]
	}
	if table != "" {
		in["table"] = table
	}
	if len(parts) > 2 {
		in["ids"] = idsNamed(text, strings.Split(parts[2], ","))
	}
	return call("view", in)
}

// awardCall finds "id Title: …Firm (column id)" in the outline.
func awardCall(pm, text string) (llm.Turn, error) {
	firm := strings.TrimSpace(strings.TrimPrefix(pm, "screen award "))
	re := regexp.MustCompile(`(?mi)^([0-9a-f]{8}) [^\n]*` + regexp.QuoteMeta(firm) + `[^\n(]*\(column ([0-9a-f]{8})\)`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		return say("That firm has not quoted.", nil), nil
	}
	return call("view", map[string]any{"do": "open", "dialog": "award", "fields": map[string]string{"package": m[1], "column": m[2]}, "final": true})
}

func say(text string, onText func(string)) llm.Turn {
	if onText != nil {
		onText(text)
	}
	msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": text}}})
	return llm.Turn{Text: text, StopReason: "end_turn", Message: msg}
}

func e2eChat() (*httpapi.Chat, error) {
	prompt, err := httpapi.LoadChatPrompt("knowledge/chat")
	if err != nil {
		return nil, err
	}
	return &httpapi.Chat{Model: chatScript{}, Name: "scripted", Prompt: prompt}, nil
}
