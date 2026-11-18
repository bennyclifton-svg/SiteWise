package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/chatfix"
	"sitewise/internal/llm"
)

// TestChatScreenCommandBudget holds chat_screen_command (agency plan, Task
// 8): a whole turn that only moves the screen, with the model answering at
// once, so the figure is the server's own time. The four commands are the
// plan's: a named filter, a select by about, an open and a show. Each must
// take one model round; the test fails when the median takes more.
func TestChatScreenCommandBudget(t *testing.T) {
	shortID := regexp.MustCompile(`\b[0-9a-f]{8}\b`)
	call := func(name string, in map[string]any) llm.Turn {
		raw, _ := json.Marshal(in)
		return llm.Turn{Calls: []llm.ToolCall{{ID: "call-1", Name: name, Input: raw}}, StopReason: "tool_use"}
	}
	a, m, project, model := chatApp(t, func(_ int, conv llm.Conversation) llm.Turn {
		asked := lastUser(conv)
		switch {
		case strings.Contains(asked, "tool_result"):
			return llm.Turn{Text: "Done.", StopReason: "end_turn"}
		case strings.Contains(asked, "no budget"):
			return call("view", map[string]any{"do": "only", "table": "cost_lines", "filter": "no_budget", "final": true})
		case strings.Contains(asked, "basement"):
			return call("documents", map[string]any{"about": "basement", "select": true})
		case strings.Contains(asked, "Open document"):
			_, typed, _ := strings.Cut(asked, "<pm_message>")
			return call("view", map[string]any{"do": "open", "table": "documents", "ids": []string{shortID.FindString(typed)}, "final": true})
		}
		return call("view", map[string]any{"do": "show", "table": "procurement", "final": true})
	})
	// A plan of sixty lines with no budget yet, and the basement set among
	// forty other documents, so the outline and the reads are of real size.
	for i := 0; i < 6; i++ {
		labels := make([]string, 10)
		for j := range labels {
			labels[j] = fmt.Sprintf("Line %d-%d", i, j)
		}
		m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"labels": labels, "section": "construction", "system": "structure"}))
	}
	docs := append([]chatfix.Document{}, basementDocs...)
	for i := 0; i < 40; i++ {
		docs = append(docs, chatfix.Document{Title: fmt.Sprintf("Drawing A-%03d", i), Kind: "drawing", Pages: []string{fmt.Sprintf("General arrangement, level %d.", i)}})
	}
	ids, err := chatfix.SeedDocuments(context.Background(), a.store, m.orgID, project, docs)
	if err != nil {
		t.Fatal(err)
	}
	open := ids["Shoring design"][:8]
	commands := []string{"Show the cost lines with no budget", "Select the basement documents", "Open document " + open, "Show the packages"}
	var samples []int64
	var rounds []int
	for i := 0; i < 6; i++ {
		for _, text := range commands {
			before := model.calls()
			start := time.Now()
			events := m.chat(t, project, "", text)
			samples = append(samples, time.Since(start).Microseconds())
			rounds = append(rounds, model.calls()-before)
			moved := false
			for _, e := range events {
				if e.Card != nil {
					t.Fatalf("%q made a %s card", text, e.Type)
				}
				moved = moved || e.Type == "view"
			}
			if !moved {
				t.Fatalf("%q moved nothing: %s", text, eventKinds(events))
			}
		}
	}
	sort.Ints(rounds)
	if median := rounds[len(rounds)/2]; median != 1 {
		t.Fatalf("the median screen command took %d model rounds; it must take one", median)
	}
	gateTender(t, []string{"chat_screen_command"}, map[string][]int64{"chat_screen_command": samples})
}
