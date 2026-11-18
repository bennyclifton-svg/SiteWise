package httpapi_test

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/llm"
)

// The Go work behind chat holds its budgets with a stand-in model, so the
// figures are the server's own time and not the provider's:
//
//	ops_apply            five cost lines in one operations call
//	change_undo          undoing that change, by its route and by typing
//	                     "undo that" (answered by Go, no model call)
//	chat_tool_read       one area rendered as chat reads it, on a 60-line plan
//	chat_simple_command  a whole turn (outline, one apply, the card, storing
//	                     the turn) with the model answering at once
//
// The same test fails when the median simple command takes more than one
// model round (speed rule 8). The live model's rounds and time are measured
// by cmd/chat-eval.
func TestChatGoWorkBudgets(t *testing.T) {
	n := 0
	_, m, project, model := chatApp(t, func(int, llm.Conversation) llm.Turn {
		n++
		return applyCall("cost_plan", true, op("cost_plan.add_line", map[string]any{"label": fmt.Sprint("Chat line ", n), "section": "construction", "system": "interiors", "amount": "45000"}))
	})
	// A plan of sixty lines, so the outline and the reads are of real size.
	for i := 0; i < 6; i++ {
		labels := make([]string, 10)
		for j := range labels {
			labels[j] = fmt.Sprintf("Line %d-%d", i, j)
		}
		m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"labels": labels, "section": "construction", "system": "structure"}))
	}
	samples := map[string][]int64{}
	var rounds []int
	for i := 0; i < 24; i++ {
		var calls []map[string]any
		for j := 0; j < 5; j++ {
			calls = append(calls, op("cost_plan.add_line", map[string]any{"label": fmt.Sprintf("Bench %d-%d", i, j), "section": "contingency", "amount": "1000"}))
		}
		start := time.Now()
		out := m.ops(t, project, "cost_plan", http.StatusOK, calls...)
		samples["ops_apply"] = append(samples["ops_apply"], time.Since(start).Microseconds())
		start = time.Now()
		m.undo(t, project, out.ChangeID, http.StatusOK)
		samples["change_undo"] = append(samples["change_undo"], time.Since(start).Microseconds())

		start = time.Now()
		if code, _ := m.text(t, http.MethodGet, "/api/projects/"+project+"/chat/read?area=cost_plan", nil); code != http.StatusOK {
			t.Fatalf("read = %d", code)
		}
		samples["chat_tool_read"] = append(samples["chat_tool_read"], time.Since(start).Microseconds())

		before := model.calls()
		start = time.Now()
		events := m.chat(t, project, "", "Add a joinery line to Fitout, $45,000")
		samples["chat_simple_command"] = append(samples["chat_simple_command"], time.Since(start).Microseconds())
		rounds = append(rounds, model.calls()-before)
		cardOf(t, events, "change")
		// "Undo that" is answered by Go with no model call; it is held to
		// the undo route's own budget (agency plan, Task 5).
		before = model.calls()
		start = time.Now()
		undone := m.chat(t, project, events[len(events)-1].ConversationID, "Undo that")
		samples["change_undo"] = append(samples["change_undo"], time.Since(start).Microseconds())
		if a := undone[len(undone)-2].Answer; model.calls() != before || a == nil || !strings.HasPrefix(a.Text, "Undone:") {
			t.Fatalf("undo that: %+v, model calls %d", a, model.calls()-before)
		}
	}
	sort.Ints(rounds)
	if median := rounds[len(rounds)/2]; median != 1 {
		t.Fatalf("the median simple command took %d model rounds; it must take one", median)
	}
	gateTender(t, []string{"ops_apply", "change_undo", "chat_tool_read", "chat_simple_command"}, samples)
}
