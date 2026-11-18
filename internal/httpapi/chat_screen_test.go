package httpapi_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"sitewise/internal/llm"
)

// The chat agency plan (docs/plans/2026-11-18-chat-agency.md), Task 1: chat
// sees the selection on screen, and only this project's rows reach it.

// lastUser is the turn's own message as the model was sent it, as text.
func lastUser(conv llm.Conversation) string {
	raw := conv.History[len(conv.History)-1]
	var msg struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &msg) != nil || len(msg.Content) == 0 || msg.Content[0].Type != "text" {
		return string(raw)
	}
	return msg.Content[0].Text
}

// onScreenOf is the screen line of the turn's message.
func onScreenOf(conv llm.Conversation) string {
	text := lastUser(conv)
	_, after, _ := strings.Cut(text, "<on_screen>")
	line, _, _ := strings.Cut(after, "</on_screen>")
	return line
}

var selectedLine = regexp.MustCompile(`Selected: \d+ cost lines \(([0-9a-f, ]+)\)`)

// costLines adds labelled construction lines and returns their full ids.
func costLines(t *testing.T, m *member, project string, labels ...string) []string {
	t.Helper()
	m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"labels": labels, "section": "construction", "system": "structure"}))
	p := planOf(t, m, project)
	ids := make([]string, len(labels))
	for i, l := range labels {
		ids[i] = lineID(t, p, l)
	}
	return ids
}

func excludedLabels(t *testing.T, m *member, project string) []string {
	t.Helper()
	var out []string
	for _, i := range planOf(t, m, project).Items {
		if i.Excluded {
			out = append(out, i.Label)
		}
	}
	return out
}

// Done means 2 and 6: "exclude these" on three lines selected by hand
// excludes exactly those three, with Undo; an id from another org never
// reaches the model; the cached prefix does not move with the screen.
func TestChatSeesTheSelection(t *testing.T) {
	a, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		found := selectedLine.FindStringSubmatch(lastUser(conv))
		if found == nil {
			return llm.Turn{Text: "Which lines?"}
		}
		return applyCall("cost_plan", true, op("cost_plan.set_lines", map[string]any{"lines": strings.Split(found[1], ", "), "excluded": true}))
	})
	ids := costLines(t, m, project, "Footings", "Slab", "Frame", "Roof")
	other := a.member(t, newUUID(t))
	theirs := other.createProject(t, "Theirs")
	foreign := costLines(t, other, theirs, "Their line")[0]

	screen := map[string]any{"view": "costs", "selected": map[string]any{"table": "cost_lines", "ids": []string{ids[0], ids[1], ids[3], foreign}, "count": 4}}
	events := m.chatOn(t, project, "", "Exclude these", screen)
	card := cardOf(t, events, "change")
	if got := excludedLabels(t, m, project); strings.Contains(strings.Join(got, ","), "Frame") || len(got) != 3 {
		t.Fatalf("excluded %v; card %+v", got, card.Card)
	}
	sent := lastUser(model.sent[0])
	for _, want := range []string{"<on_screen>", "On screen: Cost Plan.", "Selected: 3 cost lines (", "1 of the 4 is no longer there."} {
		if !strings.Contains(sent, want) {
			t.Errorf("the turn lacks %q:\n%s", want, sent)
		}
	}
	if strings.Contains(sent, foreign) || strings.Contains(sent, foreign[:8]) || strings.Contains(sent, "Their line") {
		t.Fatalf("another org's line reached the model:\n%s", sent)
	}
	if strings.Contains(model.sent[0].System, "On screen") {
		t.Fatal("the screen line is in the cached prefix")
	}
	// A different screen leaves the prefix as it was.
	m.chatOn(t, project, "", "Hello", map[string]any{"view": "program"})
	if model.sent[1].System != model.sent[0].System || !reflect.DeepEqual(model.sent[1].Tools, model.sent[0].Tools) {
		t.Fatal("the cached prefix changed with the screen")
	}
	if !strings.Contains(onScreenOf(model.sent[1]), "On screen: Program. Nothing is selected.") {
		t.Errorf("second screen line:\n%s", lastUser(model.sent[1]))
	}
	m.undo(t, project, card.Card.ChangeID, http.StatusOK)
	if got := excludedLabels(t, m, project); len(got) != 0 {
		t.Fatalf("undo left %v excluded", got)
	}
}

// More rows selected than one turn carries: the ids are not sent, and an
// operation on "these" is refused rather than applied to a part.
func TestChatRefusesAPartOfALargeSelection(t *testing.T) {
	var ids []string
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("cost_plan", true, op("cost_plan.set_lines", map[string]any{"lines": []string{ids[0][:8], ids[1][:8]}, "excluded": true}))
		}
		if !strings.Contains(lastUser(conv), "Narrow the selection") {
			t.Errorf("the refusal was not fed back: %s", lastUser(conv))
		}
		return llm.Turn{Text: "Narrow the selection to 200 or fewer."}
	})
	ids = costLines(t, m, project, "Footings", "Slab")
	m.chatOn(t, project, "", "Exclude these", map[string]any{"view": "costs", "selected": map[string]any{"table": "cost_lines", "ids": ids, "count": 600}})
	sent := onScreenOf(model.sent[0])
	if !strings.Contains(sent, "600 cost lines, more than chat can act on") || strings.Contains(sent, ids[0][:8]) {
		t.Fatalf("the screen line:\n%s", sent)
	}
	if got := excludedLabels(t, m, project); len(got) != 0 {
		t.Fatalf("a part of the selection was changed: %v", got)
	}
}
