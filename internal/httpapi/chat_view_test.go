package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/chatfix"
	"sitewise/internal/llm"
	psample "sitewise/internal/proposals/sample"
)

// The chat agency plan, Task 2: the view tool moves the screen.

// changeCounts is how many changes and prepared cards the org holds: a
// screen move must add neither.
func changeCounts(t *testing.T, org string) (int, int) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var changes, prepared int
	if err := conn.QueryRow(ctx, `SELECT (SELECT count(*) FROM project_changes WHERE org_id=$1::uuid), (SELECT count(*) FROM chat_prepared WHERE org_id=$1::uuid)`, org).Scan(&changes, &prepared); err != nil {
		t.Fatal(err)
	}
	return changes, prepared
}

// Task 2: the view tool checks every id, ends the turn in one round, is
// stored with the turn, and after a document read prepares nothing.
func TestChatViewMovesTheScreen(t *testing.T) {
	ta := newToolApp(t)
	docs, err := chatfix.SeedDocuments(context.Background(), ta.a.store, ta.m.orgID, ta.project, testDocs)
	if err != nil {
		t.Fatal(err)
	}
	ids := costLines(t, ta.m, ta.project, "Footings", "Slab", "Frame")
	viewOf := func(events []chatEvent) *chatEvent {
		for i := range events {
			if events[i].Type == "view" {
				return &events[i]
			}
		}
		return nil
	}

	// select by short ids, final: one round, full ids back, Go's own line.
	events := ta.run(t, "Select the footings and the slab", tool("view", map[string]any{"do": "select", "table": "cost_lines", "ids": []string{ids[0][:8], ids[1][:8]}, "final": true}))
	v := viewOf(events)
	if v == nil || !reflect.DeepEqual(v.View.IDs, ids[:2]) || v.View.Line != "Selected 2 cost lines." || events[len(events)-1].Rounds != 1 {
		t.Fatalf("events %s, view %+v", eventKinds(events), v)
	}
	conversation := events[len(events)-1].ConversationID

	// The next turn's history names the selected rows by id.
	ta.calls = nil
	ta.m.chat(t, ta.project, conversation, "Now show only those")

	// Another org's row, a made-up id, a bad table and a filter name are
	// refused, and nothing moves.
	other := ta.a.member(t, newUUID(t))
	theirs := other.createProject(t, "Theirs")
	foreign := costLines(t, other, theirs, "Their line")[0]
	for name, input := range map[string]map[string]any{
		"another org's row": {"do": "select", "table": "cost_lines", "ids": []string{foreign}},
		"a made-up id":      {"do": "only", "table": "cost_lines", "ids": []string{"deadbeef"}},
		"a bad table":       {"do": "select", "table": "invoices", "ids": []string{ids[0]}},
		"a filter":          {"do": "only", "table": "cost_lines", "filter": "no_budget"},
		"a bad dialog":      {"do": "open", "dialog": "send", "fields": map[string]string{}},
	} {
		events := ta.run(t, "Move the screen", tool("view", input))
		if viewOf(events) != nil || len(ta.results) != 1 {
			t.Errorf("%s: a view was sent: %s", name, eventKinds(events))
		}
		if name == "a filter" && !strings.Contains(ta.results[0], `no named filter "no_budget"`) {
			t.Errorf("filter refusal: %s", ta.results[0])
		}
		if strings.Contains(ta.results[0], foreign[:8]) && name != "another org's row" {
			t.Errorf("%s: %s", name, ta.results[0])
		}
	}

	// Done means 5: a turn that read a document and then only moves the
	// screen prepares nothing and changes no table.
	changes, prepared := changeCounts(t, ta.m.orgID)
	planBefore := withoutVersions(t, ta.m, "/api/projects/"+ta.project+"/cost-plan")
	var consent string
	for title, id := range docs {
		if strings.HasPrefix(title, "Development consent") {
			consent = id
		}
	}
	ta.reply = func([]string) llm.Turn {
		in := fmt.Sprintf(`{"do":"select","table":"documents","ids":[%q],"final":true}`, consent[:8])
		return llm.Turn{Calls: []llm.ToolCall{{ID: "call-view", Name: "view", Input: []byte(in)}}, StopReason: "tool_use"}
	}
	events = ta.run(t, "Find the consent and select it", tool("passages", map[string]any{"words": "consent"}))
	v = viewOf(events)
	if v == nil || len(v.View.IDs) != 1 || v.View.IDs[0] != consent {
		t.Fatalf("events %s, view %+v", eventKinds(events), v)
	}
	for _, e := range events {
		if e.Type == "prepared" || e.Type == "change" {
			t.Fatalf("a screen move made a %s card", e.Type)
		}
	}
	if c, p := changeCounts(t, ta.m.orgID); c != changes || p != prepared {
		t.Fatalf("changes %d→%d, prepared %d→%d", changes, c, prepared, p)
	}
	if after := withoutVersions(t, ta.m, "/api/projects/"+ta.project+"/cost-plan"); !reflect.DeepEqual(planBefore, after) {
		t.Fatal("the cost plan changed")
	}
	// The move is stored with the turn.
	var stored struct {
		Turns []struct {
			Answer struct {
				Events []struct {
					Type string `json:"type"`
				} `json:"events"`
			} `json:"answer"`
		} `json:"turns"`
	}
	ta.m.getJSON(t, "/api/projects/"+ta.project+"/chat", &stored)
	last := stored.Turns[len(stored.Turns)-1].Answer.Events
	kinds := ""
	for _, e := range last {
		kinds += e.Type + " "
	}
	if !strings.Contains(kinds, "view") {
		t.Fatalf("the view was not stored: %s", kinds)
	}
}

// The award dialog opens on a quoted firm's column; chat never awards.
func TestChatOpensTheAwardDialog(t *testing.T) {
	var pkg, sub string
	model := &scriptModel{script: func(n int, conv llm.Conversation) llm.Turn {
		if !strings.Contains(lastUser(conv), "tool_result") {
			in := fmt.Sprintf(`{"do":"open","dialog":"award","fields":{"package":%q,"column":%q},"final":true}`, pkg[:8], sub[:8])
			return llm.Turn{Calls: []llm.ToolCall{{ID: "call-award", Name: "view", Input: []byte(in)}}, StopReason: "tool_use"}
		}
		return llm.Turn{Text: "That firm has not quoted on that package."}
	}}
	a := newApp(t, withProfile(t), greenProposals, withChat(t, model))
	m := a.member(t, newUUID(t))
	m.modelSwitch(t, true, http.StatusOK)
	project := m.createProject(t, "Award from chat")
	var row struct{ ID string }
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &row)
	pkg = row.ID
	pdf := psample.PDF(psample.Proposal{Firm: "Design Practice Pty Ltd", Lines: []psample.Line{{Label: "Concept and documentation lump sum", Cents: 1200000}}})
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/packages/"+pkg+"/submissions?name="+url.QueryEscape("Design fee.pdf"), pdf, http.StatusCreated, &up)
	sub = up.SubmissionID

	events := m.chat(t, project, "", "Open the award for Design Practice")
	var view *chatEvent
	for i := range events {
		if events[i].Type == "view" {
			view = &events[i]
		}
	}
	if view == nil || view.View.Dialog != "award" || view.View.Fields["package"] != pkg || view.View.Fields["column"] != sub || !strings.Contains(view.View.Line, "Press Award there") {
		t.Fatalf("events %s, view %+v", eventKinds(events), view)
	}
	for _, e := range events {
		if e.Card != nil {
			t.Fatalf("opening the dialog made a %s card", e.Type)
		}
	}
	// A column that is not on the package is refused.
	sub = newUUID(t)
	events = m.chat(t, project, "", "Open the award for someone else")
	for _, e := range events {
		if e.Type == "view" {
			t.Fatalf("a made-up column opened the dialog: %+v", e.View)
		}
	}
}
