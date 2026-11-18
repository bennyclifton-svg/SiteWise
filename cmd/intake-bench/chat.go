package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"sitewise/internal/chatfix"
	"sitewise/internal/httpapi"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// The chat rows of bench/budgets.json, sampled through the real server with
// a stand-in model that answers at once. So these are Go's share of each
// path: the outline, the tools, the apply, the checks and storing the turn.
// The live model's time is measured by cmd/chat-eval and recorded in
// docs/evidence/2026-11-15-chat-agent.md.

type benchChatModel struct{}

func (benchChatModel) Converse(_ context.Context, conv llm.Conversation, onText func(string)) (llm.Turn, error) {
	if conv.Warm {
		return llm.Turn{}, nil
	}
	last := string(conv.History[len(conv.History)-1])
	text := func(s string) (llm.Turn, error) {
		if onText != nil {
			onText(s)
		}
		msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": s}}})
		return llm.Turn{Text: s, StopReason: "end_turn", Message: msg}, nil
	}
	call := func(name string, input map[string]any) (llm.Turn, error) {
		raw, _ := json.Marshal(input)
		msg, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "tool_use", "id": "call-1", "name": name, "input": json.RawMessage(raw)}}})
		return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: name, Input: raw}}, StopReason: "tool_use", Message: msg}, nil
	}
	switch {
	case strings.Contains(last, `"tool_result"`):
		return text("No mention found.")
	case strings.Contains(last, "joinery"):
		return call("apply", map[string]any{"area": "cost_plan", "final": true, "ops": []map[string]any{{"op": "cost_plan.add_line",
			"fields": map[string]any{"label": "Joinery", "section": "construction", "system": "interiors", "amount": "45000"}}}})
	case strings.Contains(last, "trees"):
		return call("passages", map[string]any{"words": "tree, arborist"})
	// The screen commands (agency plan, Task 8): each ends in one round.
	case strings.Contains(last, "Show the cost lines with no budget"):
		return call("view", map[string]any{"do": "only", "table": "cost_lines", "filter": "no_budget", "final": true})
	case strings.Contains(last, "Select the basement documents"):
		return call("documents", map[string]any{"about": "basement", "select": true})
	case strings.Contains(last, "Open document "):
		_, typed, _ := strings.Cut(last, "Open document ")
		return call("view", map[string]any{"do": "open", "table": "documents", "ids": []string{typed[:8]}, "final": true})
	case strings.Contains(last, "Show the packages"):
		return call("view", map[string]any{"do": "show", "table": "procurement", "final": true})
	}
	return text("Which line do you mean?")
}

func benchChat(repo string) (*httpapi.Chat, error) {
	prompt, err := httpapi.LoadChatPrompt(filepath.Join(repo, "knowledge", "chat"))
	if err != nil {
		return nil, err
	}
	return &httpapi.Chat{Model: benchChatModel{}, Name: "stand-in", Prompt: prompt}, nil
}

func (a *apiClient) chat(ctx context.Context, st *store.Store, n int) error {
	// The org's switch, on: not a sampled path.
	resp, err := a.do(ctx, http.MethodPut, "/account/model", strings.NewReader(`{"enabled":true}`), 16)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("switching the model on = %d", resp.StatusCode)
	}
	project, err := a.createProject(ctx, "Chat benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project
	ops := func(calls ...map[string]any) (string, error) {
		body, _ := json.Marshal(map[string]any{"area": "cost_plan", "ops": calls})
		raw, err := a.timed(ctx, "ops_apply", http.MethodPost, base+"/ops", body, http.StatusOK)
		if err != nil {
			return "", err
		}
		var out struct {
			ChangeID string `json:"change_id"`
		}
		return out.ChangeID, json.Unmarshal(raw, &out)
	}
	line := func(label string) map[string]any {
		return map[string]any{"op": "cost_plan.add_line", "fields": map[string]any{"label": label, "section": "contingency", "amount": "1000"}}
	}
	// A plan of sixty lines, so the outline and the reads are of real size.
	for i := 0; i < 12; i++ {
		var five []map[string]any
		for j := 0; j < 5; j++ {
			five = append(five, line(fmt.Sprintf("Line %d-%d", i, j)))
		}
		if _, err := ops(five...); err != nil {
			return err
		}
	}
	// Documents for the screen commands: a basement set among forty others,
	// filed straight into the store as the chat tests file them.
	docs := []chatfix.Document{
		{Title: "Basement excavation plan", Kind: "drawing", Pages: []string{"Set out and levels for the bulk dig."}},
		{Title: "Geotechnical report", Kind: "report", Pages: []string{"Groundwater was met above the basement floor level."}},
		{Title: "Shoring design", Kind: "report", Pages: []string{"Soldier piles with shotcrete infill."}, Systems: []string{"substructure.basement-retention"}},
		{Title: "Basement survey", Kind: "drawing"},
	}
	for i := 0; i < 40; i++ {
		docs = append(docs, chatfix.Document{Title: fmt.Sprintf("Drawing A-%03d", i), Kind: "drawing", Pages: []string{fmt.Sprintf("General arrangement, level %d.", i)}})
	}
	ids, err := chatfix.SeedDocuments(ctx, st, benchOrg, project, docs)
	if err != nil {
		return err
	}
	screen := []string{"Show the cost lines with no budget", "Select the basement documents", "Open document " + ids["Shoring design"][:8], "Show the packages"}
	var rounds []int
	say := func(path, text string) ([]byte, error) {
		body, _ := json.Marshal(map[string]string{"text": text})
		return a.timed(ctx, path, http.MethodPost, base+"/chat", body, http.StatusOK)
	}
	for i := 0; i < n; i++ {
		var five []map[string]any
		for j := 0; j < 5; j++ {
			five = append(five, line(fmt.Sprintf("Bench %d-%d", i, j)))
		}
		change, err := ops(five...)
		if err != nil {
			return err
		}
		if _, err := a.timed(ctx, "change_undo", http.MethodPost, base+"/changes/"+change+"/undo", []byte(`{}`), http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "chat_tool_read", http.MethodGet, base+"/chat/read?area=cost_plan", nil, http.StatusOK); err != nil {
			return err
		}
		raw, err := say("chat_simple_command", "Add a joinery line to Fitout, $45,000")
		if err != nil {
			return err
		}
		// One model round, or the row has failed whatever its time.
		if !strings.Contains(string(raw), `"rounds":1,`) || !strings.Contains(string(raw), `"type":"change"`) {
			return fmt.Errorf("a simple command did not apply in one model round")
		}
		if _, err := say("chat_first_response", "Which line?"); err != nil {
			return err
		}
		if _, err := say("chat_document_answer", "Are trees involved in this project?"); err != nil {
			return err
		}
		for _, text := range screen {
			raw, err := say("chat_screen_command", text)
			if err != nil {
				return err
			}
			var done struct {
				Rounds int `json:"rounds"`
			}
			moved := false
			for _, line := range strings.Split(string(raw), "\n") {
				data, ok := strings.CutPrefix(line, "data: ")
				switch {
				case !ok:
				case strings.Contains(data, `"type":"view"`):
					moved = true
				case strings.Contains(data, `"type":"done"`):
					_ = json.Unmarshal([]byte(data), &done)
				}
			}
			if !moved {
				return fmt.Errorf("screen command %q moved nothing", text)
			}
			rounds = append(rounds, done.Rounds)
		}
	}
	// The median screen command takes one model round, or the row has
	// failed whatever its time.
	sort.Ints(rounds)
	if len(rounds) > 0 && rounds[len(rounds)/2] != 1 {
		return fmt.Errorf("the median screen command took %d model rounds; it must take one", rounds[len(rounds)/2])
	}
	return nil
}
