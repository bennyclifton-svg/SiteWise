package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/chateval"
	"sitewise/internal/delivery"
	"sitewise/internal/store"
)

// names are a fixture's rows by id and by the name the key uses: a cost
// line's label, a document's title, a Program row's label, a package's
// discipline id, and a quoted firm's column by the firm's name.
type names struct {
	byID    map[string]map[string]string   // table to id to name
	byName  map[string]map[string][]string // table to name to ids
	columns map[string]string              // submission id to firm name
	firms   map[string]string              // firm name to submission id
}

func (n *names) add(table, id, name string) {
	if n.byID[table] == nil {
		n.byID[table], n.byName[table] = map[string]string{}, map[string][]string{}
	}
	n.byID[table][id] = name
	n.byName[table][name] = append(n.byName[table][name], id)
}

// name is a row's name, or a mark the scorer cannot mistake for a fixture
// row when the row is not in the project.
func (n *names) name(table, id string) string {
	if v, ok := n.byID[table][id]; ok {
		return v
	}
	return "unknown row " + id[:min(len(id), 8)]
}

// ids are the rows of the given names; every row of a name is meant.
func (n *names) ids(table string, rows []string) ([]string, error) {
	var out []string
	for _, r := range rows {
		ids := n.byName[table][r]
		if len(ids) == 0 {
			return nil, fmt.Errorf("the fixture has no %s row %q", table, r)
		}
		out = append(out, ids...)
	}
	return out, nil
}

// names reads the project's rows with the session's own org.
func (s *session) names(ctx context.Context) (*names, error) {
	n := &names{byID: map[string]map[string]string{}, byName: map[string]map[string][]string{}, columns: map[string]string{}, firms: map[string]string{}}
	plan, err := s.l.st.ReadCostPlan(ctx, s.org, s.project, "")
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	for _, i := range plan.Items {
		n.add("cost_lines", i.ID, i.Label)
	}
	docs, err := s.l.st.ProjectDocumentViews(ctx, s.org, s.project)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		title := d.Filename
		for _, f := range d.Fields {
			if f.Field == "title" && f.Value != "" {
				title = f.Value
			}
		}
		n.add("documents", d.ID, title)
	}
	program, err := s.l.st.LoadProgram(ctx, s.org, s.project)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	for _, a := range program.Activities {
		n.add("program", a.ID, a.Label)
	}
	pkgs, err := s.l.st.ReadPackages(ctx, s.org, s.project)
	if err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		if p.RetiredAt == nil {
			n.add("procurement", p.ID, p.DisciplineID)
		}
	}
	register, err := s.l.st.ReadDelivery(ctx, s.org, s.project)
	if err != nil {
		return nil, err
	}
	for _, it := range register {
		name := it.Title
		if it.Number != nil {
			name = fmt.Sprintf("RFI-%03d", *it.Number)
		}
		if it.Kind == "condition" {
			name = "Condition " + delivery.Condition(it.Content).Number
		}
		n.add("register", it.ID, name)
	}
	board, _, err := s.l.st.ProcurementBoard(ctx, s.org, s.project)
	if err != nil {
		return nil, err
	}
	for _, cards := range board {
		for _, c := range cards {
			if c.SubmissionID != "" {
				n.columns[c.SubmissionID], n.firms[c.FirmName] = c.FirmName, c.SubmissionID
			}
		}
	}
	return n, nil
}

// screen is the case's screen context with this fixture's ids, as the
// strip sends it.
func (n *names) screen(sc *chateval.Screen) (map[string]any, error) {
	if sc == nil {
		return nil, nil
	}
	out := map[string]any{"view": sc.View}
	if sc.Open != nil {
		ids, err := n.ids("documents", []string{sc.Open.Name})
		if err != nil {
			return nil, err
		}
		out["open"] = map[string]string{"kind": "document", "id": ids[0]}
	}
	if sel := sc.Selected; sel != nil {
		ids, err := n.ids(sel.Table, sel.Rows)
		if err != nil {
			return nil, err
		}
		out["selected"] = map[string]any{"table": sel.Table, "ids": ids, "count": len(ids)}
	}
	return out, nil
}

// goTables are the tables a go event shows.
var goTables = map[string]string{"costs": "cost_lines", "program": "program", "procurement": "procurement", "documents": "documents", "register": "register"}

// view is a screen move with its rows named as the key names them.
func (n *names) view(v agent.View) chateval.View {
	out := chateval.View{Do: v.Do, Table: v.Table, Dialog: v.Dialog}
	for _, id := range v.IDs {
		out.Rows = append(out.Rows, n.name(v.Table, id))
	}
	if v.Dialog != "" {
		out.Fields = map[string]string{"package": n.name("procurement", v.Fields["package"]), "column": n.columns[v.Fields["column"]]}
	}
	return out
}

// turn is what one chat turn sent back.
type turn struct {
	conversation string
	out          chateval.Outcome
	views        []agent.View
	goes         []string
}

// chat sends one message, with the screen when there is one, and reads the
// stream to its end.
func (s *session) chat(ctx context.Context, conversation, text string, screen map[string]any, debug io.Writer) (turn, error) {
	var t turn
	body := map[string]any{"text": text}
	if conversation != "" {
		body["conversation_id"] = conversation
	}
	if screen != nil {
		body["screen"] = screen
	}
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.l.url+"/api/projects/"+s.project+"/chat", bytes.NewReader(raw))
	req.Header.Set("Origin", s.l.url)
	req.Header.Set("Content-Type", "application/json")
	began := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return t, fmt.Errorf("chat = %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 4<<20), 4<<20)
	failed := ""
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var e struct {
			agent.Event
			ConversationID string `json:"conversation_id"`
			Rounds         int    `json:"rounds"`
			InTokens       int64  `json:"in_tokens"`
			OutTokens      int64  `json:"out_tokens"`
			CacheRead      int64  `json:"cache_read_tokens"`
		}
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return t, err
		}
		if e.Type != "text" {
			fmt.Fprintf(debug, "%s\n", line)
		}
		switch e.Type {
		case "prepared":
			t.out.Prepared++
		case "view":
			if e.View != nil {
				t.views = append(t.views, *e.View)
			}
		case "go":
			t.goes = append(t.goes, e.Text)
		case "answer":
			if e.Answer != nil {
				t.out.Text, t.out.Question, t.out.Unchecked = e.Answer.Text, e.Answer.Question, e.Answer.Unchecked
				t.out.Cited, t.out.Coverage, t.out.Verdicts = e.Answer.Cited, e.Answer.Coverage != "", e.Answer.Verdicts
			}
		case "error":
			failed = e.Text
		case "done":
			t.conversation = e.ConversationID
			t.out.Rounds, t.out.InTokens, t.out.OutTokens, t.out.CacheRead = e.Rounds, e.InTokens, e.OutTokens, e.CacheRead
		}
	}
	t.out.Millis = time.Since(began).Milliseconds()
	if failed != "" {
		return t, errors.New(failed)
	}
	return t, nil
}

// Run is one case from a fresh fixture: its earlier messages, then its
// message with its screen. The areas are read before the first message, so
// the case judges where the conversation leaves the project; the rest of
// the outcome is the last turn's.
func (l *live) Run(ctx context.Context, c chateval.Case) (chateval.Outcome, error) {
	var out chateval.Outcome
	s, err := l.open(ctx, fixtures[c.Fixture].Name)
	if err != nil {
		return out, err
	}
	defer func() { _ = l.st.DeleteOrg(context.Background(), s.org) }()
	if err := s.seed(ctx, fixtures[c.Fixture], c.Needs); err != nil {
		return out, fmt.Errorf("fixture %s: %w", c.Fixture, err)
	}
	before, err := s.reads()
	if err != nil {
		return out, err
	}
	seeded, err := s.names(ctx)
	if err != nil {
		return out, err
	}
	screen, err := seeded.screen(c.Screen)
	if err != nil {
		return out, err
	}
	if l.onSeed != nil {
		l.onSeed(c, seeded)
	}
	if _, outline, err := s.do(http.MethodGet, "/api/projects/"+s.project+"/chat/read?area=outline", nil); err == nil && os.Getenv("CHAT_EVAL_OUTLINE") != "" {
		_ = os.WriteFile(os.Getenv("CHAT_EVAL_OUTLINE"), outline, 0o644)
	}
	// Focusing the strip warms the cached prefix; the eval does it once.
	l.warm.Do(func() {
		_, _, _ = s.do(http.MethodPost, "/api/projects/"+s.project+"/chat/warm", []byte(`{}`))
		time.Sleep(l.warmFor)
	})
	fmt.Fprintf(l.debug, "\n=== %s [%s] %s\n", c.ID, c.Fixture, c.Message)
	conversation := ""
	for _, m := range c.Before {
		t, err := s.chat(ctx, conversation, m, nil, l.debug)
		if err != nil {
			return out, fmt.Errorf("earlier message %q: %w", m, err)
		}
		conversation = t.conversation
	}
	t, err := s.chat(ctx, conversation, c.Message, screen, l.debug)
	out = t.out
	if err != nil {
		return out, err
	}
	after, err := s.reads()
	if err != nil {
		return out, err
	}
	out.After = after
	for _, area := range areas {
		if before[area] != after[area] {
			out.Changed = append(out.Changed, area)
		}
	}
	// Rows are named as they were after seeding, so a row the turn deleted
	// keeps its name.
	for _, v := range t.views {
		out.Views = append(out.Views, seeded.view(v))
	}
	for _, g := range t.goes {
		if table, ok := goTables[g]; ok {
			out.Views = append(out.Views, chateval.View{Do: "show", Table: table})
		}
	}
	return out, nil
}
