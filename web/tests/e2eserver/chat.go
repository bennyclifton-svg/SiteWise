package main

import (
	"context"
	"encoding/json"
	"errors"
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
	said := ""
	for _, c := range last.Content {
		if c.Type == "tool_result" {
			return say("Done.", onText), nil
		}
		if _, after, ok := strings.Cut(c.Text, "<pm_message>"); ok {
			said = strings.ToLower(after)
		}
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
