package httpapi_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// projectReportID finds the project's project report through the screen's
// own list.
func projectReportID(t *testing.T, m *member, project string) string {
	t.Helper()
	var list []store.Report
	m.getJSON(t, "/api/projects/"+project+"/reports", &list)
	for _, r := range list {
		if r.Kind == "monthly" {
			return r.ID
		}
	}
	t.Fatalf("no project report in %+v", list)
	return ""
}

func commentBlock(t *testing.T, m *member, report string) (text string, edited bool) {
	t.Helper()
	var view store.ReportView
	m.getJSON(t, "/api/reports/"+report, &view)
	for _, s := range view.Draft.Sections {
		for _, b := range s.Blocks {
			if b.ID == "comment:cost" {
				return b.Text, b.Edited
			}
		}
	}
	t.Fatal("no cost comment")
	return "", false
}

// From chat: Update applies at once with nothing to undo; a comment applies
// at once and Undo puts it back; a cleared comment comes back on Undo; a
// report of another org is never found (monthly plan, Task 10).
func TestProjectReportOperations(t *testing.T) {
	a, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "ok"} })
	// No report yet: the read says how to make one, and opening it is refused.
	if read := m.plainRead(t, project, "report"); !strings.Contains(read, "There is no project report yet") {
		t.Fatalf("read before: %s", read)
	}
	updated := m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	if updated.Undo.Kind != "none" || len(updated.Results) != 1 || !strings.Contains(updated.Results[0].Summary, "draft 1") {
		t.Fatalf("update %+v", updated)
	}
	report := projectReportID(t, m, project)
	// A second update reuses the one report.
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	if again := projectReportID(t, m, project); again != report {
		t.Fatal("a second update made a second report")
	}
	read := m.plainRead(t, project, "report")
	for _, want := range []string{"Project report draft 1", "At a glance:", "Needs attention:", "No cost plan yet.", "the PM presses Issue"} {
		if !strings.Contains(read, want) {
			t.Fatalf("read lacks %q:\n%s", want, read)
		}
	}
	generated, _ := commentBlock(t, m, report)

	written := m.ops(t, project, "reports", http.StatusOK, op("reports.comment", map[string]any{"section": "cost", "text": "The cost plan starts next month."}))
	if written.ChangeID == "" || written.Undo.Kind != "change" {
		t.Fatalf("comment %+v", written)
	}
	if text, edited := commentBlock(t, m, report); text != "The cost plan starts next month." || !edited {
		t.Fatalf("after the comment: %q %v", text, edited)
	}
	if read := m.plainRead(t, project, "report"); !strings.Contains(read, "The cost plan starts next month. (PM)") {
		t.Fatalf("read after the comment:\n%s", read)
	}
	m.undo(t, project, written.ChangeID, http.StatusOK)
	if text, edited := commentBlock(t, m, report); text != generated || edited {
		t.Fatalf("after undo: %q %v, want %q", text, edited, generated)
	}

	m.ops(t, project, "reports", http.StatusOK, op("reports.comment", map[string]any{"section": "cost", "text": "Kept."}))
	cleared := m.ops(t, project, "reports", http.StatusOK, op("reports.comment", map[string]any{"section": "cost", "clear": true}))
	if text, edited := commentBlock(t, m, report); text != generated || edited {
		t.Fatalf("after clear: %q %v", text, edited)
	}
	m.undo(t, project, cleared.ChangeID, http.StatusOK)
	if text, edited := commentBlock(t, m, report); text != "Kept." || !edited {
		t.Fatalf("after undoing the clear: %q %v", text, edited)
	}
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.comment", map[string]any{"section": "glance", "text": "x"})); r.Field != "section" {
		t.Fatalf("a section with no comment: %+v", r)
	}

	// Another org's report, named by id, is not in this project.
	other := a.member(t, newUUID(t))
	otherProject := other.createProject(t, "Other")
	other.ops(t, otherProject, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	foreign := projectReportID(t, other, otherProject)
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.comment", map[string]any{"report": foreign, "section": "cost", "text": "x"})); !strings.Contains(r.Message, "no report with id") {
		t.Fatalf("a foreign report: %+v", r)
	}
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.update", map[string]any{"report": foreign})); !strings.Contains(r.Message, "no report with id") {
		t.Fatalf("a foreign update: %+v", r)
	}
	// Another org's member cannot reach this project's operations or report.
	other.ops(t, project, "reports", http.StatusNotFound, op("reports.update", map[string]any{}))
	if got := other.status(t, http.MethodGet, "/api/reports/"+report, nil); got != http.StatusNotFound {
		t.Fatalf("a foreign read = %d", got)
	}
	// A second project of the same org: its report is not this one's.
	second := m.createProject(t, "Second")
	if r := m.ops(t, second, "reports", http.StatusUnprocessableEntity, op("reports.comment", map[string]any{"report": report, "section": "cost", "text": "x"})); !strings.Contains(r.Message, "no report with id") {
		t.Fatalf("another project's report: %+v", r)
	}
}

// "Issue it" opens the reader's issue form; chat issues nothing.
func TestChatOpensTheProjectReportToIssue(t *testing.T) {
	model := func(n int, conv llm.Conversation) llm.Turn {
		if !strings.Contains(lastUser(conv), "tool_result") {
			return llm.Turn{Calls: []llm.ToolCall{{ID: "call-report", Name: "view", Input: []byte(`{"do":"open","dialog":"report","fields":{"issue":"yes"},"final":true}`)}}, StopReason: "tool_use"}
		}
		return llm.Turn{Text: "There is no project report yet."}
	}
	_, m, project, _ := chatApp(t, model)
	events := m.chat(t, project, "", "Issue the project report")
	for _, e := range events {
		if e.Type == "view" {
			t.Fatalf("opened a report that does not exist: %+v", e.View)
		}
	}
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	events = m.chat(t, project, "", "Issue the project report")
	var view *chatEvent
	for i := range events {
		if events[i].Type == "view" {
			view = &events[i]
		}
		if events[i].Card != nil {
			t.Fatalf("opening the report made a %s card", events[i].Type)
		}
	}
	if view == nil || view.View.Dialog != "report" || view.View.Fields["report"] != report || view.View.Fields["issue"] != "yes" || !strings.Contains(view.View.Line, "Press Freeze and issue PDF") {
		t.Fatalf("events %s, view %+v", eventKinds(events), view)
	}
	var after store.ReportView
	m.getJSON(t, "/api/reports/"+report, &after)
	if len(after.Issues) != 0 {
		t.Fatal("chat issued the report")
	}
}

var figureSigns = regexp.MustCompile(`[$,]`)

// The report's Needs attention and chat's attention read are one list
// (Task 5's test, with the report as the second reader).
func TestProjectReportAttentionEqualsChats(t *testing.T) {
	_, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "ok"} })
	m.ops(t, project, "cost_plan", http.StatusOK,
		op("cost_plan.add_line", map[string]any{"label": "Joinery", "section": "construction", "system": "interiors"}),
		op("cost_plan.add_line", map[string]any{"label": "Stone", "section": "construction", "system": "interiors", "amount": "45000"}))
	m.call(t, http.MethodPost, "/api/projects/"+project+"/delivery", []byte(`{"kind":"approval","title":"DA","status":"submitted","target_date":"2020-01-01","details":{}}`), http.StatusCreated, nil)
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	var view store.ReportView
	m.getJSON(t, "/api/reports/"+report, &view)
	var fromReport []string
	for _, s := range view.Draft.Sections {
		if s.ID != "attention" {
			continue
		}
		for _, b := range s.Blocks {
			fromReport = append(fromReport, strings.TrimSuffix(figureSigns.ReplaceAllString(b.Text, ""), "."))
		}
	}
	var fromChat []string
	line := regexp.MustCompile(`^\s+\d+\. (.*) \(source: [^)]*\)$`)
	for _, l := range strings.Split(m.plainRead(t, project, "attention"), "\n") {
		if g := line.FindStringSubmatch(l); g != nil {
			fromChat = append(fromChat, figureSigns.ReplaceAllString(g[1], ""))
		}
	}
	if len(fromChat) < 2 || fmt.Sprint(fromChat) != fmt.Sprint(fromReport) {
		t.Fatalf("chat %q\nreport %q", fromChat, fromReport)
	}
}

// A comment drafted in chat carrying a figure nobody gave is refused: the
// pencil guard holds for report comments as for every other figure.
func TestChatRefusesAReportCommentWithItsOwnFigure(t *testing.T) {
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("reports", true, op("reports.comment", map[string]any{"section": "cost", "text": "Forecast is tracking at $99,000."}))
		}
		if last := string(conv.History[len(conv.History)-1]); !strings.Contains(last, "Say the figure") {
			t.Errorf("the refusal was not fed back: %s", last)
		}
		return llm.Turn{Text: "Which figure should the comment use?"}
	})
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	m.chat(t, project, "", "Draft the cost comment")
	if _, edited := commentBlock(t, m, report); edited || model.calls() != 2 {
		t.Fatalf("a comment with the model's own figure was written (%d calls)", model.calls())
	}
}
