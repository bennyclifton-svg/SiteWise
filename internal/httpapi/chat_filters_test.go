package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/chatfix"
	"sitewise/internal/httpapi"
	"sitewise/internal/llm"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/reports"
)

// The chat agency plan, Task 4: named filters. The model names the set and
// Go computes the rows, in one round.

// viewEvent is the turn's one view event, or fails the test.
func viewEvent(t *testing.T, events []chatEvent) chatEvent {
	t.Helper()
	for _, e := range events {
		if e.Type == "view" && e.View != nil {
			return e
		}
	}
	t.Fatalf("no view event: %s", eventKinds(events))
	return chatEvent{}
}

// filter runs one view call with a filter and returns its view, checking
// it took one round and made no card.
func (ta *toolApp) filter(t *testing.T, do, table, name string) chatEvent {
	t.Helper()
	events := ta.run(t, "Filter the "+table, tool("view", map[string]any{"do": do, "table": table, "filter": name, "final": true}))
	if last := events[len(events)-1]; last.Rounds != 1 {
		t.Errorf("%s %s took %d rounds", table, name, last.Rounds)
	}
	for _, e := range events {
		if e.Card != nil {
			t.Errorf("%s %s made a %s card", table, name, e.Type)
		}
	}
	return viewEvent(t, events)
}

func sqlExec(t *testing.T, query string, args ...any) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func sorted(ids []string) []string {
	out := append([]string{}, ids...)
	sort.Strings(out)
	return out
}

func TestChatNamedFilters(t *testing.T) {
	ta := newToolApp(t)
	ctx := context.Background()

	// Rows in another project of the same org, and in another org, that
	// every filter would match: they must never be in a set.
	elsewhere := ta.m.createProject(t, "Elsewhere")
	theirLine := costLines(t, ta.m, elsewhere, "Elsewhere line")[0]
	other := ta.a.member(t, newUUID(t))
	theirs := other.createProject(t, "Theirs")
	foreignLine := costLines(t, other, theirs, "Their line")[0]
	foreignDocs, err := chatfix.SeedDocuments(ctx, ta.a.store, other.orgID, theirs, []chatfix.Document{{Title: "Their survey", Kind: "drawing"}})
	if err != nil {
		t.Fatal(err)
	}
	strangers := []string{theirLine, foreignLine, foreignDocs["Their survey"]}
	noStrangers := func(name string, ids []string) {
		t.Helper()
		for _, s := range strangers {
			for _, id := range ids {
				if id == s {
					t.Errorf("%s: a row from another project or org is in the set", name)
				}
			}
		}
	}

	t.Run("cost lines", func(t *testing.T) {
		ta.m.ops(t, ta.project, "cost_plan", http.StatusOK,
			op("cost_plan.add_line", map[string]any{"label": "Footings", "section": "construction", "system": "structure", "amount": "1000"}),
			op("cost_plan.add_line", map[string]any{"label": "Frame", "section": "construction", "system": "structure", "amount": "90000"}))
		ids := costLines(t, ta.m, ta.project, "Slab", "Roof", "Walls")
		p := planOf(t, ta.m, ta.project)
		footings, frame, slab, roof := lineID(t, p, "Footings"), lineID(t, p, "Frame"), ids[0], ids[1]
		ta.m.ops(t, ta.project, "cost_plan", http.StatusOK, op("cost_plan.set_lines", map[string]any{"lines": []string{roof[:8]}, "excluded": true}))

		// no budget: the set is the one the report counts.
		v := ta.filter(t, "only", "cost_lines", "no_budget")
		got := v.View.IDs
		noStrangers("no_budget", got)
		if !containsString(got, slab) || containsString(got, footings) || containsString(got, frame) || containsString(got, roof) {
			t.Fatalf("no_budget %v (slab %s)", got, slab)
		}
		ta.run(t, "What needs attention?", tool("read", map[string]any{"area": "attention"}))
		attention := strings.Join(ta.results, "\n")
		if len(got) < 2 || !strings.Contains(attention, fmt.Sprintf("%d cost lines have no budget amount", len(got))) {
			t.Errorf("the filter has %d lines; attention says:\n%s", len(got), attention)
		}
		if v.View.Do != "only" || v.View.Filter != "no_budget" || v.View.Shown != "with no budget" || v.View.Line != fmt.Sprintf("Showing %d cost lines with no budget.", len(got)) {
			t.Errorf("view %+v", v.View)
		}

		// excluded, selected.
		v = ta.filter(t, "select", "cost_lines", "excluded")
		if !reflect.DeepEqual(v.View.IDs, []string{roof}) || v.View.Line != "Selected 1 cost line excluded from the totals." {
			t.Errorf("excluded %+v", v.View)
		}

		// over budget: nothing claimed yet, so nothing is over, whatever the
		// budgets; the table is shown with a plain line.
		v = ta.filter(t, "only", "cost_lines", "over_budget")
		if v.View.Do != "show" || len(v.View.IDs) != 0 || v.View.Line != "No cost line has claimed more than its budget." {
			t.Errorf("over_budget with no invoices %+v", v.View)
		}
		// An invoice of $24,000 on Footings (budget $1,000) and on Slab (no
		// budget): Footings is over; Slab's budget is unknown, so it is not.
		raw, err := os.ReadFile("../invoices/testdata/quoin.md")
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range []string{footings, slab} {
			var created struct{ ID string }
			body := append(append([]byte{}, raw...), []byte(strings.Repeat("\n", i+1))...)
			ta.m.call(t, http.MethodPost, "/api/projects/"+ta.project+"/invoices?name="+url.QueryEscape(fmt.Sprintf("quoin-%d.md", i)), body, http.StatusCreated, &created)
			ta.m.waitInvoices(t, ta.project, i+1)
			edit := fmt.Sprintf(`{"fields":{"line":%q}}`, line)
			ta.m.call(t, http.MethodPatch, "/api/projects/"+ta.project+"/invoices/"+created.ID, []byte(edit), http.StatusOK, nil)
		}
		v = ta.filter(t, "only", "cost_lines", "over_budget")
		if !reflect.DeepEqual(v.View.IDs, []string{footings}) || v.View.Line != "Showing 1 cost line over budget on claims to date." {
			t.Errorf("over_budget %+v", v.View)
		}
	})

	t.Run("documents", func(t *testing.T) {
		docs, err := chatfix.SeedDocuments(ctx, ta.a.store, ta.m.orgID, ta.project, testDocs)
		if err != nil {
			t.Fatal(err)
		}
		sqlExec(t, `UPDATE documents SET status='not_filed', reason='test: unreadable' WHERE org_id=$1::uuid AND id=$2::uuid`, ta.m.orgID, docs["Arborist report"])
		v := ta.filter(t, "only", "documents", "not_filed")
		if !reflect.DeepEqual(v.View.IDs, []string{docs["Arborist report"]}) || v.View.Line != "Showing 1 document not filed." {
			t.Errorf("not_filed %+v", v.View)
		}
		// The profile reads reports and specifications, not drawings, and
		// never a document set to skip.
		sqlExec(t, `UPDATE documents SET profile_read='skip' WHERE org_id=$1::uuid AND id=$2::uuid`, ta.m.orgID, docs["Mechanical specification rev A"])
		v = ta.filter(t, "select", "documents", "not_read")
		want := sorted([]string{docs["Site survey"], docs["Mechanical specification rev A"]})
		noStrangers("not_read", v.View.IDs)
		if !reflect.DeepEqual(sorted(v.View.IDs), want) || v.View.Line != "Selected 2 documents the profile does not read." {
			t.Errorf("not_read %+v, want %v", v.View, want)
		}
		v = ta.filter(t, "only", "documents", "superseded")
		if !reflect.DeepEqual(v.View.IDs, []string{docs["Mechanical specification rev A"]}) || v.View.Line != "Showing 1 document superseded by a newer revision." {
			t.Errorf("superseded %+v", v.View)
		}

		// A filter after a document read still moves the screen at once and
		// prepares nothing.
		changes, prepared := changeCounts(t, ta.m.orgID)
		ta.reply = func([]string) llm.Turn {
			return llm.Turn{Calls: []llm.ToolCall{tool("view", map[string]any{"do": "only", "table": "documents", "filter": "superseded", "final": true})}, StopReason: "tool_use"}
		}
		events := ta.run(t, "Read the consent, then show the superseded documents", tool("passages", map[string]any{"words": "consent"}))
		ta.reply = func([]string) llm.Turn { return llm.Turn{Text: "ok"} }
		if v := viewEvent(t, events); !reflect.DeepEqual(v.View.IDs, []string{docs["Mechanical specification rev A"]}) {
			t.Errorf("after a read %+v", v.View)
		}
		for _, e := range events {
			if e.Type == "prepared" || e.Type == "change" {
				t.Fatalf("a screen move after a read made a %s card", e.Type)
			}
		}
		if c, p := changeCounts(t, ta.m.orgID); c != changes || p != prepared {
			t.Fatalf("changes %d→%d, prepared %d→%d", changes, c, prepared, p)
		}
	})

	t.Run("program", func(t *testing.T) {
		base := "/api/projects/" + ta.project + "/program"
		ta.m.call(t, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, nil)
		var view programView
		ta.m.getJSON(t, base, &view)
		stage := ""
		for _, act := range view.Activities {
			if act.Kind == "stage" && stage == "" {
				stage = act.ID
			}
		}
		ta.m.ops(t, ta.project, "program", http.StatusOK, op("program.edit", map[string]any{"op": "add", "parent_id": stage, "kind": "activity", "label": "Undated task"}))
		var full struct {
			Activities []struct {
				ID, Label, Kind string
				DurationDays    *int `json:"duration_days"`
			} `json:"activities"`
			Computed struct {
				Activities map[string]struct {
					Critical bool `json:"critical"`
					Unknown  bool `json:"unknown"`
				} `json:"activities"`
			} `json:"computed"`
		}
		ta.m.getJSON(t, base, &full)
		var undated, critical []string
		for _, a := range full.Activities {
			if a.Kind == "activity" && a.DurationDays == nil {
				undated = append(undated, a.ID)
			}
			if c := full.Computed.Activities[a.ID]; c.Critical && !c.Unknown {
				critical = append(critical, a.ID)
			}
		}
		if len(undated) == 0 || len(critical) == 0 {
			t.Fatalf("fixture: %d undated, %d critical", len(undated), len(critical))
		}
		v := ta.filter(t, "only", "program", "no_duration")
		if !reflect.DeepEqual(sorted(v.View.IDs), sorted(undated)) || v.View.Shown != "with no duration" {
			t.Errorf("no_duration %+v, want %v", v.View, undated)
		}
		v = ta.filter(t, "select", "program", "critical")
		if !reflect.DeepEqual(sorted(v.View.IDs), sorted(critical)) || !strings.HasSuffix(v.View.Line, " on the critical path.") {
			t.Errorf("critical %+v, want %d rows", v.View, len(critical))
		}
	})

	t.Run("refusals", func(t *testing.T) {
		for name, input := range map[string]map[string]any{
			"unknown":      {"do": "only", "table": "cost_lines", "filter": "overdue"},
			"wrong table":  {"do": "only", "table": "documents", "filter": "no_budget"},
			"with ids":     {"do": "only", "table": "cost_lines", "filter": "no_budget", "ids": []string{"abc"}},
			"with show":    {"do": "show", "table": "cost_lines", "filter": "no_budget"},
			"no such verb": {"do": "open", "table": "program", "filter": "critical"},
		} {
			events := ta.run(t, "Filter", tool("view", input))
			for _, e := range events {
				if e.Type == "view" {
					t.Errorf("%s: a view was sent: %+v", name, e.View)
				}
			}
			if len(ta.results) != 1 {
				t.Fatalf("%s: results %v", name, ta.results)
			}
			switch name {
			case "unknown":
				if !strings.Contains(ta.results[0], `no filter "overdue" for cost lines`) || !strings.Contains(ta.results[0], "excluded, no_budget, over_budget") {
					t.Errorf("unknown: %s", ta.results[0])
				}
			case "wrong table":
				if !strings.Contains(ta.results[0], "not_filed, not_read, superseded") {
					t.Errorf("wrong table: %s", ta.results[0])
				}
			}
		}
	})

	// One fixture, two readers (the subtests above built it, so this runs
	// after them): each attention rule a filter shares gives
	// the filter's rows exactly (monthly report plan, Task 5).
	t.Run("attention and filters agree", func(t *testing.T) {
		items, err := httpapi.AttentionForTest(ctx, ta.a.store, ta.m.orgID, ta.project)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct{ rule, table, filter string }{
			{reports.RuleCostNoBudget, "cost_lines", "no_budget"},
			{reports.RuleLineClaimedOver, "cost_lines", "over_budget"},
			{reports.RuleDocumentNotFiled, "documents", "not_filed"},
			{reports.RuleProgramNoDuration, "program", "no_duration"},
		} {
			it, ok := reports.AttentionFor(items, c.rule)
			rows, err := httpapi.FilterRowsForTest(ctx, ta.a.store, ta.m.orgID, ta.project, c.table, c.filter)
			if err != nil {
				t.Fatal(err)
			}
			if !ok || len(rows) == 0 || it.Count != len(rows) || !reflect.DeepEqual(sorted(it.IDs), sorted(rows)) {
				t.Errorf("%s: attention %v (found %v, count %d), filter %s %v", c.rule, it.IDs, ok, it.Count, c.filter, rows)
			}
			if c.rule != reports.RuleLineClaimedOver && !reflect.DeepEqual(it.IDs, rows) {
				t.Errorf("%s: order differs: attention %v, filter %v", c.rule, it.IDs, rows)
			}
		}
	})

	// The tool description lists every filter, so the model can name one.
	var desc string
	for _, tl := range ta.model.sent[len(ta.model.sent)-1].Tools {
		if tl.Name == "view" {
			raw, _ := json.Marshal(tl.Schema)
			desc = tl.Description + string(raw)
		}
	}
	for _, name := range []string{"no_budget", "over_budget", "excluded", "not_filed", "not_read", "superseded", "no_duration", "critical", "no_quote"} {
		if !strings.Contains(desc, name) {
			t.Errorf("the view tool does not list %s: %s", name, desc)
		}
	}
}

// Packages with no quote, beside one that has a quote, in one round.
func TestChatNoQuoteFilter(t *testing.T) {
	model := &scriptModel{script: func(n int, conv llm.Conversation) llm.Turn {
		return llm.Turn{Calls: []llm.ToolCall{{ID: "call-filter", Name: "view", Input: []byte(`{"do":"only","table":"procurement","filter":"no_quote","final":true}`)}}, StopReason: "tool_use"}
	}}
	a := newApp(t, withProfile(t), greenProposals, withChat(t, model))
	m := a.member(t, newUUID(t))
	m.modelSwitch(t, true, http.StatusOK)
	project := m.createProject(t, "No quote")
	var quoted, empty struct{ ID string }
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &quoted)
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &empty)
	pdf := psample.PDF(psample.Proposal{Firm: "Design Practice Pty Ltd", Lines: []psample.Line{{Label: "Concept and documentation lump sum", Cents: 1200000}}})
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/packages/"+quoted.ID+"/submissions?name="+url.QueryEscape("Design fee.pdf"), pdf, http.StatusCreated, nil)

	events := m.chat(t, project, "", "Show the packages with no quote")
	v := viewEvent(t, events)
	if containsString(v.View.IDs, quoted.ID) || !containsString(v.View.IDs, empty.ID) || events[len(events)-1].Rounds != 1 {
		t.Fatalf("no_quote %+v (quoted %s, empty %s)", v.View, quoted.ID, empty.ID)
	}
	if !regexp.MustCompile(`^Showing \d+ packages? with no quote\.$`).MatchString(v.View.Line) {
		t.Errorf("line %q", v.View.Line)
	}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
