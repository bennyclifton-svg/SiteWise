package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing/fstest"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/auth"
	"sitewise/internal/chateval"
	"sitewise/internal/chatfix"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/llm"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

// pick keeps the cases whose id or group is named; an empty list keeps all.
func pick(cases []chateval.Case, only string) []chateval.Case {
	if strings.TrimSpace(only) == "" {
		return cases
	}
	want := strings.Split(only, ",")
	var out []chateval.Case
	for _, c := range cases {
		if slices.Contains(want, c.ID) || slices.Contains(want, c.Group) {
			out = append(out, c)
		}
	}
	return out
}

var areas = []string{"cost_plan", "profile", "program", "procurement"}

// live runs each case through the real server: a fresh org and project, the
// fixture seeded through the same routes and store a PM's work goes
// through, the message sent to the chat route, the areas read before and
// after. The database is a private one; no shared instance is touched.
type live struct {
	st    *store.Store
	url   string
	debug io.Writer
	warm  sync.Once
}

func uuid() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func liveAgent(model string) (chateval.Agent, func(), error) {
	dsn := os.Getenv("SITEWISE_CHAT_EVAL_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "sitewise_chat") {
		return nil, nil, errors.New("SITEWISE_CHAT_EVAL_DATABASE_URL must name a private database whose name starts sitewise_chat")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := st.Migrate(ctx); err != nil {
		return nil, nil, err
	}
	tmp, err := os.MkdirTemp("", "chat-eval")
	if err != nil {
		return nil, nil, err
	}
	blobs, err := files.Open(tmp, 10<<20)
	if err != nil {
		return nil, nil, err
	}
	cat, err := intake.LoadCatalog(filepath.Join("data", "intake"))
	if err != nil {
		return nil, nil, err
	}
	thresholds, err := intake.LoadThresholds(filepath.Join("data", "intake"))
	if err != nil {
		return nil, nil, err
	}
	building, err := knowledge.Load("knowledge")
	if err != nil {
		return nil, nil, err
	}
	reading, err := profile.LoadReadPolicy(filepath.Join("data", "profile", "reading.json"))
	if err != nil {
		return nil, nil, err
	}
	// Nothing in a chat turn calls Jev; the client points nowhere.
	jevClient, err := jev.New(jev.Options{BaseURL: "http://127.0.0.1:9", APIKey: "unused", Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		return nil, nil, err
	}
	logFile, err := os.Create(filepath.Join(os.TempDir(), "chat-eval-llm.log"))
	if err != nil {
		return nil, nil, err
	}
	client := llm.New(llm.Options{APIKey: os.Getenv("SITEWISE_ANTHROPIC_HAIKU_KEY"), Logger: slog.New(slog.NewJSONHandler(logFile, nil)), MaxConcurrent: 4})
	if !client.On() {
		return nil, nil, errors.New("SITEWISE_ANTHROPIC_HAIKU_KEY is not set")
	}
	prompt, err := httpapi.LoadChatPrompt(filepath.Join("knowledge", "chat"))
	if err != nil {
		return nil, nil, err
	}
	chat := &httpapi.Chat{Model: client, On: client.On, Name: model, Prompt: prompt, Think: os.Getenv("CHAT_EVAL_THINK") == "1"}
	if chat.Synonyms, chat.SystemTerms, err = httpapi.LoadChatWords("knowledge"); err != nil {
		return nil, nil, err
	}
	ts := httptest.NewUnstartedServer(nil)
	srv, err := httpapi.New(httpapi.Options{
		Store: st, Blobs: blobs, Jev: jevClient, Catalog: cat, Thresholds: thresholds,
		Static:       fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>chat-eval</title>")}},
		PublicOrigin: "http://" + ts.Listener.Addr().String(), MaxUploadBytes: 10 << 20, Log: log.New(io.Discard, "", 0),
		Knowledge: building, ProfileReading: reading,
		ProfileThresholds: profile.Thresholds{Version: "eval", Amber: map[string]float64{}, Green: map[string]*float64{}},
		Chat:              chat,
	})
	if err != nil {
		return nil, nil, err
	}
	ts.Config.Handler = srv
	ts.Start()
	l := &live{st: st, url: ts.URL, debug: io.Discard}
	if path := os.Getenv("CHAT_EVAL_DEBUG"); path != "" {
		if f, err := os.Create(path); err == nil {
			l.debug = f
		}
	}
	done := func() {
		srv.CloseStreams()
		ts.Close()
		st.Close()
		logFile.Close()
		_ = os.RemoveAll(tmp)
	}
	return l, done, nil
}

type session struct {
	l       *live
	client  *http.Client
	org     string
	project string
}

func (s *session) do(method, path string, body any) (int, []byte, error) {
	var raw []byte
	if body != nil {
		if b, ok := body.([]byte); ok {
			raw = b
		} else {
			raw, _ = json.Marshal(body)
		}
	}
	req, err := http.NewRequest(method, s.l.url+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Origin", s.l.url)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	return resp.StatusCode, out, err
}

// must is a call that has to succeed for the fixture to be right.
func (s *session) must(method, path string, body any, into any) error {
	code, raw, err := s.do(method, path, body)
	if err != nil {
		return err
	}
	if code < 200 || code > 299 {
		return fmt.Errorf("%s %s = %d: %s", method, path, code, raw)
	}
	if into != nil {
		return json.Unmarshal(raw, into)
	}
	return nil
}

func (s *session) ops(area string, calls ...map[string]any) error {
	return s.must(http.MethodPost, "/api/projects/"+s.project+"/ops", map[string]any{"area": area, "ops": calls}, nil)
}

func op(name string, fields map[string]any) map[string]any {
	return map[string]any{"op": name, "fields": fields}
}

func (l *live) open(ctx context.Context, name string) (*session, error) {
	token, org, err := auth.Bootstrap(ctx, l.st, &auth.Sink{}, uuid(), uuid()+"@example.test", nil)
	if err != nil {
		return nil, err
	}
	jar, _ := cookiejar.New(nil)
	s := &session{l: l, client: &http.Client{Jar: jar, Timeout: 3 * time.Minute}, org: org}
	if err := s.must(http.MethodPost, "/api/session", map[string]string{"token": token}, nil); err != nil {
		return nil, err
	}
	if err := s.must(http.MethodPut, "/api/account/model", map[string]bool{"enabled": true}, nil); err != nil {
		return nil, err
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := s.must(http.MethodPost, "/api/projects", map[string]string{"name": name}, &p); err != nil {
		return nil, err
	}
	s.project = p.ID
	return s, nil
}

// seed builds a fixture in the session's project.
func (s *session) seed(ctx context.Context, f fixture) error {
	base := "/api/projects/" + s.project
	for _, c := range f.Cost {
		fields := map[string]any{"label": c.Label, "section": c.Section}
		for k, v := range map[string]string{"system": c.System, "category": c.Category, "amount": c.Budget} {
			if v != "" {
				fields[k] = v
			}
		}
		if err := s.ops("cost_plan", op("cost_plan.add_line", fields)); err != nil {
			return err
		}
	}
	var plan struct {
		ID      string `json:"id"`
		Version int64  `json:"version"`
		Items   []struct {
			ID    string `json:"cost_item_id"`
			Label string `json:"label"`
		} `json:"items"`
	}
	for _, c := range f.Cost {
		if c.Contract == "" && c.Claimed == "" {
			continue
		}
		if err := s.must(http.MethodGet, base+"/cost-plan", nil, &plan); err != nil {
			return err
		}
		id := ""
		for _, i := range plan.Items {
			if i.Label == c.Label {
				id = i.ID
			}
		}
		if c.Contract != "" {
			if err := s.ops("cost_plan", op("cost_plan.set_amount", map[string]any{"line": id, "metric": "contract", "amount": c.Contract})); err != nil {
				return err
			}
		}
		if c.Claimed != "" {
			if err := s.must(http.MethodGet, base+"/cost-plan", nil, &plan); err != nil {
				return err
			}
			body := map[string]any{"plan_version_id": plan.ID, "plan_version": plan.Version, "amount": c.Claimed, "value_state": "known",
				"as_of": "2026-10-31", "origin": "user", "meaning": "stated"}
			if err := s.must(http.MethodPut, base+"/cost-plan/items/"+id+"/values/claimed_to_date", body, nil); err != nil {
				return err
			}
		}
	}
	for key, value := range f.Profile {
		if err := s.ops("profile", op("profile.set_value", map[string]any{"key": key, "value": value})); err != nil {
			return err
		}
	}
	if err := s.must(http.MethodPost, base+"/program/prepare", map[string]string{"surface": "program"}, nil); err != nil {
		return err
	}
	if f.SixDayWeek {
		if err := s.ops("program", op("program.edit", map[string]any{"op": "set_working_week", "working_week": 63})); err != nil {
			return err
		}
	}
	var program struct {
		Activities []struct {
			ID, Kind, Label string
		} `json:"activities"`
	}
	if err := s.must(http.MethodGet, base+"/program", nil, &program); err != nil {
		return err
	}
	stage := func(name string) string {
		first := ""
		for _, a := range program.Activities {
			if a.Kind != "stage" {
				continue
			}
			if first == "" {
				first = a.ID
			}
			if strings.Contains(strings.ToLower(a.Label), strings.ToLower(name)) {
				return a.ID
			}
		}
		return first
	}
	for _, a := range f.Activities {
		if err := s.ops("program", op("program.edit", map[string]any{"op": "add", "parent_id": stage(a.Stage), "kind": "activity", "label": a.Label, "duration": a.Duration})); err != nil {
			return err
		}
	}
	for _, p := range f.Packages {
		var row struct {
			ID string `json:"id"`
		}
		if err := s.must(http.MethodPost, base+"/procurement/rows", map[string]string{"discipline_id": p.Discipline}, &row); err != nil {
			return err
		}
		for _, firm := range p.Invited {
			if err := s.must(http.MethodPost, base+"/procurement/packages/"+row.ID+"/invitations", map[string]string{"name": firm}, nil); err != nil {
				return err
			}
		}
	}
	if _, err := chatfix.SeedDocuments(ctx, s.l.st, s.org, s.project, f.Documents); err != nil {
		return err
	}
	return nil
}

func (s *session) reads() (map[string]string, error) {
	out := map[string]string{}
	for _, area := range areas {
		code, raw, err := s.do(http.MethodGet, "/api/projects/"+s.project+"/chat/read?area="+area, nil)
		if err != nil || code != http.StatusOK {
			return nil, fmt.Errorf("read %s = %d %v", area, code, err)
		}
		out[area] = string(raw)
	}
	return out, nil
}

// Run is one case from a fresh fixture.
func (l *live) Run(ctx context.Context, c chateval.Case) (chateval.Outcome, error) {
	var out chateval.Outcome
	s, err := l.open(ctx, fixtures[c.Fixture].Name)
	if err != nil {
		return out, err
	}
	defer func() { _ = l.st.DeleteOrg(context.Background(), s.org) }()
	if err := s.seed(ctx, fixtures[c.Fixture]); err != nil {
		return out, fmt.Errorf("fixture %s: %w", c.Fixture, err)
	}
	before, err := s.reads()
	if err != nil {
		return out, err
	}
	// Focusing the strip warms the cached prefix; the eval does it once.
	l.warm.Do(func() {
		_, _, _ = s.do(http.MethodPost, "/api/projects/"+s.project+"/chat/warm", []byte(`{}`))
		time.Sleep(3 * time.Second)
	})
	raw, _ := json.Marshal(map[string]string{"text": c.Message})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, l.url+"/api/projects/"+s.project+"/chat", bytes.NewReader(raw))
	req.Header.Set("Origin", l.url)
	req.Header.Set("Content-Type", "application/json")
	began := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("chat = %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 4<<20), 4<<20)
	fmt.Fprintf(l.debug, "\n=== %s [%s] %s\n", c.ID, c.Fixture, c.Message)
	failed := ""
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var e struct {
			agent.Event
			Rounds     int   `json:"rounds"`
			InTokens   int64 `json:"in_tokens"`
			OutTokens  int64 `json:"out_tokens"`
			CacheRead  int64 `json:"cache_read_tokens"`
			FirstMilli int64 `json:"first_millis"`
		}
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return out, err
		}
		if e.Type != "text" {
			fmt.Fprintf(l.debug, "%s\n", line)
		}
		switch e.Type {
		case "prepared":
			out.Prepared++
		case "answer":
			if e.Answer != nil {
				out.Text, out.Question, out.Unchecked = e.Answer.Text, e.Answer.Question, e.Answer.Unchecked
				out.Cited, out.Coverage = e.Answer.Cited, e.Answer.Coverage != ""
			}
		case "error":
			failed = e.Text
		case "done":
			out.Rounds, out.InTokens, out.OutTokens, out.CacheRead = e.Rounds, e.InTokens, e.OutTokens, e.CacheRead
		}
	}
	out.Millis = time.Since(began).Milliseconds()
	if failed != "" {
		return out, errors.New(failed)
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
	return out, nil
}
