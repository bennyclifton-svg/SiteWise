package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"sitewise/internal/llm"
)

// The chat agency plan, Task 5: history names ids, and "undo that" is
// answered by Go with no model call.

// Task 5: "undo that" is answered by Go with no model call; a change since
// is left alone; history names the rows a change touched.
func TestChatUndoThatWithoutTheModel(t *testing.T) {
	var ids []string
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		return applyCall("cost_plan", true, op("cost_plan.set_lines", map[string]any{"lines": []string{ids[0][:8], ids[1][:8]}, "excluded": true}))
	})
	ids = costLines(t, m, project, "Footings", "Slab", "Frame")
	events := m.chat(t, project, "", "Exclude the footings and the slab")
	conversation := events[len(events)-1].ConversationID
	if len(excludedLabels(t, m, project)) != 2 {
		t.Fatal("the change was not made")
	}
	calls := model.calls()
	events = m.chat(t, project, conversation, "Undo that.")
	answer := events[len(events)-2]
	if model.calls() != calls || answer.Type != "answer" || !strings.HasPrefix(answer.Answer.Text, "Undone:") || len(excludedLabels(t, m, project)) != 0 {
		t.Fatalf("calls %d→%d, events %s, answer %+v, excluded %v", calls, model.calls(), eventKinds(events), answer.Answer, excludedLabels(t, m, project))
	}
	// Nothing left to undo: said plainly, still no model call.
	events = m.chat(t, project, conversation, "undo")
	if a := events[len(events)-2].Answer; model.calls() != calls || a == nil || !strings.Contains(a.Text, "nothing to undo") {
		t.Fatalf("second undo: %+v", a)
	}

	// A change another screen has altered since is not undone.
	m.chat(t, project, conversation, "Exclude the footings and the slab again")
	history := string(model.sent[len(model.sent)-1].History[1])
	if !strings.Contains(history, "[Changed "+ids[0][:8]+" "+ids[1][:8]+":") {
		t.Errorf("the change's ids are not in the history: %s", history)
	}
	m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.set_lines", map[string]any{"lines": []string{ids[0][:8]}, "excluded": false}))
	calls = model.calls()
	events = m.chat(t, project, conversation, "undo the last change")
	if a := events[len(events)-2].Answer; model.calls() != calls || a == nil || a.Text != "Can't undo: this has changed since." {
		t.Fatalf("undo after a later edit: %+v", a)
	}
	if got := excludedLabels(t, m, project); len(got) != 1 || got[0] != "Slab" {
		t.Fatalf("the later edit was lost: %v", got)
	}
}

// A Program change is undone only while the Program is as it left it.
func TestChatUndoAProgramChange(t *testing.T) {
	stage := ""
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		return applyCall("program", true, op("program.edit", map[string]any{"op": "add", "parent_id": stage[:8], "kind": "activity", "label": fmt.Sprint("DA drawings ", n), "duration": "2 weeks"}))
	})
	base := "/api/projects/" + project + "/program"
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, nil)
	var view programView
	m.getJSON(t, base, &view)
	for _, a := range view.Activities {
		if a.Kind == "stage" && stage == "" {
			stage = a.ID
		}
	}
	rows := func() int {
		var v programView
		m.getJSON(t, base, &v)
		return len(v.Activities)
	}
	before := rows()
	events := m.chat(t, project, "", "Add DA drawings under the first stage, two weeks")
	conversation := events[len(events)-1].ConversationID
	if rows() != before+1 {
		t.Fatal("the row was not added")
	}
	calls := model.calls()
	events = m.chat(t, project, conversation, "undo that")
	if a := events[len(events)-2].Answer; model.calls() != calls || a == nil || !strings.HasPrefix(a.Text, "Undone:") || rows() != before {
		t.Fatalf("program undo: %+v, rows %d", a, rows())
	}
}
