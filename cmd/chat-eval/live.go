package main

import (
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
	"sitewise/internal/config"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/llm"
	"sitewise/internal/profile"
	"sitewise/internal/proposals"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
	tsample "sitewise/internal/tender/sample"
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

var areas = []string{"cost_plan", "profile", "accommodation", "program", "procurement", "delivery", "report", "shares", "mail"}

// live runs each case through the real server: a fresh org and project, the
// fixture seeded through the same routes and store a PM's work goes
// through, the message sent to the chat route, the areas read before and
// after. The database is a private one; no shared instance is touched.
type live struct {
	st    *store.Store
	url   string
	debug io.Writer
	warm  sync.Once
	// warmFor is how long the first case waits after warming the prefix.
	warmFor time.Duration
	// onSeed, when set, is told each case's rows after its fixture is
	// built: the offline test's stand-in model needs their ids.
	onSeed func(c chateval.Case, n *names)
	// dsn is the database the runner's store uses; seeding a fixture's
	// issued RFP writes it directly (shares.go).
	dsn string
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
	logFile, err := os.Create(filepath.Join(os.TempDir(), "chat-eval-llm.log"))
	if err != nil {
		return nil, nil, err
	}
	client := llm.New(llm.Options{APIKey: os.Getenv("SITEWISE_ANTHROPIC_HAIKU_KEY"), Logger: slog.New(slog.NewJSONHandler(logFile, nil)), MaxConcurrent: 4})
	if !client.On() {
		return nil, nil, errors.New("SITEWISE_ANTHROPIC_HAIKU_KEY is not set")
	}
	l, done, err := newRunner(st, client, client.On, model, ".")
	if err != nil {
		return nil, nil, err
	}
	l.dsn = dsn
	l.debug = io.Discard
	if path := os.Getenv("CHAT_EVAL_DEBUG"); path != "" {
		if f, err := os.Create(path); err == nil {
			l.debug = f
		}
	}
	return l, func() {
		done()
		st.Close()
		logFile.Close()
	}, nil
}

// newRunner starts the real server on st with the given chat model. The
// live run passes the provider's client; the offline test passes a
// stand-in. repo is the repository root, for the knowledge and data files.
func newRunner(st *store.Store, model agent.Model, on func() bool, name, repo string) (*live, func(), error) {
	tmp, err := os.MkdirTemp("", "chat-eval")
	if err != nil {
		return nil, nil, err
	}
	blobs, err := files.Open(tmp, 10<<20)
	if err != nil {
		return nil, nil, err
	}
	cat, err := intake.LoadCatalog(filepath.Join(repo, "data", "intake"))
	if err != nil {
		return nil, nil, err
	}
	thresholds, err := intake.LoadThresholds(filepath.Join(repo, "data", "intake"))
	if err != nil {
		return nil, nil, err
	}
	building, err := knowledge.Load(filepath.Join(repo, "knowledge"))
	if err != nil {
		return nil, nil, err
	}
	reading, err := profile.LoadReadPolicy(filepath.Join(repo, "data", "profile", "reading.json"))
	if err != nil {
		return nil, nil, err
	}
	// Nothing in a chat turn calls Jev; the client points nowhere.
	// Filing a fixture's tender asks Jev its filing questions. A stand-in
	// answers them here, as in the test suites; no chat turn calls Jev.
	fake := httptest.NewServer(http.HandlerFunc(standInJev))
	jevClient, err := jev.New(jev.Options{BaseURL: fake.URL, APIKey: "unused", Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		return nil, nil, err
	}
	prompt, err := httpapi.LoadChatPrompt(filepath.Join(repo, "knowledge", "chat"))
	if err != nil {
		return nil, nil, err
	}
	chat := &httpapi.Chat{Model: model, On: on, Name: name, Prompt: prompt, Think: os.Getenv("CHAT_EVAL_THINK") == "1"}
	if chat.Synonyms, chat.SystemTerms, err = httpapi.LoadChatWords(filepath.Join(repo, "knowledge")); err != nil {
		return nil, nil, err
	}
	ts := httptest.NewUnstartedServer(nil)
	srv, err := httpapi.New(httpapi.Options{
		Store: st, Blobs: blobs, Jev: jevClient, Catalog: cat, Thresholds: thresholds,
		Static:       fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>chat-eval</title>")}},
		PublicOrigin: "http://" + ts.Listener.Addr().String(), MaxUploadBytes: 10 << 20, Log: log.New(io.Discard, "", 0),
		Knowledge: building, ProfileReading: reading,
		// A fixture's tenders are read by stand-ins, in this process.
		ProposalJev: psample.Jev{}, ProposalThresholds: greenProposals(),
		TenderLLM: tsample.Model{}, TenderModel: "claude-haiku-5-5", TenderTimeout: 20 * time.Second,
		TenderRender: func(_ context.Context, _ string, pages []int) (map[int]string, error) {
			out := map[int]string{}
			for _, p := range pages {
				out[p] = "/9j/4AAQSkZJRg=="
			}
			return out, nil
		},
		ProfileThresholds: profile.Thresholds{Version: "eval", Amber: map[string]float64{}, Green: map[string]*float64{}},
		Chat:              chat,
	})
	if err != nil {
		return nil, nil, err
	}
	ts.Config.Handler = srv
	ts.Start()
	l := &live{st: st, url: ts.URL, debug: io.Discard, warmFor: 3 * time.Second}
	done := func() {
		srv.CloseStreams()
		ts.Close()
		fake.Close()
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
func greenProposals() proposals.Thresholds {
	lo, hi, g := 2, 255, 0.5
	th := proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	for _, q := range []string{"price", "rate", "criterion", "firm", "kind", "package", "gst_basis", "stated_total"} {
		th.Questions[q] = []proposals.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &g}}
	}
	return th
}

// standInJev answers every choice question with its first option.
func standInJev(w http.ResponseWriter, r *http.Request) {
	var call struct {
		Questions map[string]struct {
			Criteria map[string]json.RawMessage `json:"criteria"`
		} `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	answers := map[string]any{}
	for id, q := range call.Questions {
		first := ""
		probs := map[string]float64{}
		for opt := range q.Criteria {
			if first == "" || opt < first {
				first = opt
			}
			probs[opt] = 0
		}
		probs[first] = 1
		answers[id] = map[string]any{"type": "choice", "choice": first, "probabilities": probs, "confidence": 0.5}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"model": config.PinnedJevModel, "answers": answers, "usage": map[string]int{"input_tokens": 10, "output_tokens": 2}})
}

// seedTenders drops each firm's synthetic tender on the package, waits for
// the stand-in tender pass, and checks the columns the fixture says the PM
// has checked.
func (s *session) seedTenders(base string, tenders []tenderFix) error {
	var table struct {
		Items []struct {
			Label string `json:"label"`
			Base  bool   `json:"base"`
		} `json:"items"`
		Tender struct {
			Version int64 `json:"version"`
			Columns []struct {
				SubmissionID string `json:"submission_id"`
				State        string `json:"state"`
			} `json:"columns"`
		} `json:"tender"`
	}
	if err := s.must(http.MethodGet, base+"/prices", nil, &table); err != nil {
		return err
	}
	var labels []string
	for _, it := range table.Items {
		if it.Base {
			labels = append(labels, it.Label)
		}
	}
	if len(labels) == 0 {
		labels = []string{"The works"}
	}
	subs := map[string]bool{}
	checked := []string{}
	for n, t := range tenders {
		var lines []psample.Line
		for _, label := range labels {
			lines = append(lines, psample.Line{Label: label, Cents: t.Total * 100 / int64(len(labels))})
		}
		pdf := psample.PDF(psample.Proposal{Firm: t.Firm, ABN: abn(n), Email: fmt.Sprintf("office%d@invented.test", n), Lines: lines})
		var up struct {
			SubmissionID string `json:"submission_id"`
		}
		if err := s.must(http.MethodPost, base+"/submissions?name="+url.QueryEscape("Tender ("+t.Firm+").pdf"), pdf, &up); err != nil {
			return err
		}
		subs[up.SubmissionID] = true
		if t.Checked {
			checked = append(checked, up.SubmissionID)
		}
	}
	for deadline := time.Now().Add(40 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		if err := s.must(http.MethodGet, base+"/prices", nil, &table); err != nil {
			return err
		}
		read := 0
		for _, c := range table.Tender.Columns {
			if subs[c.SubmissionID] && c.State == "read" {
				read++
			}
		}
		if read == len(subs) {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%d of %d tenders were read in time", read, len(subs))
		}
	}
	for _, sub := range checked {
		if err := s.must(http.MethodGet, base+"/prices", nil, &table); err != nil {
			return err
		}
		body := map[string]any{"version": table.Tender.Version, "ops": []map[string]any{{"op": "check_column", "submission": sub, "checked": true}}}
		if err := s.must(http.MethodPost, base+"/tender/ops", body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) seed(ctx context.Context, f fixture, needs []string) error {
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
	// The plan is ex GST at 10%, as the fixtures' description says.
	if err := s.must(http.MethodGet, base+"/cost-plan", nil, &plan); err != nil {
		return err
	}
	settings := map[string]any{"plan_version_id": plan.ID, "version": plan.Version, "currency": "AUD", "tax_basis": "ex_tax", "tax_rate": "0.1", "coverage": ""}
	if err := s.must(http.MethodPut, base+"/cost-plan", settings, nil); err != nil {
		return err
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
		add := map[string]any{"op": "add", "parent_id": stage(a.Stage), "kind": kindOf(a), "label": a.Label}
		if a.Kind == "" {
			add["duration"] = a.Duration // a hold point, like a milestone, has no length
		}
		if err := s.ops("program", op("program.edit", add)); err != nil {
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
		if slices.Contains(needs, "tenders") && len(p.Tenders) > 0 {
			if err := s.seedTenders(base+"/procurement/packages/"+row.ID, p.Tenders); err != nil {
				return fmt.Errorf("tenders: %w", err)
			}
		}
		for _, firm := range p.Invited {
			if err := s.must(http.MethodPost, base+"/procurement/packages/"+row.ID+"/invitations", map[string]string{"name": firm}, nil); err != nil {
				return err
			}
		}
	}
	docs, err := chatfix.SeedDocuments(ctx, s.l.st, s.org, s.project, f.Documents)
	if err != nil {
		return err
	}
	if slices.Contains(needs, "shares") {
		if err := s.seedShares(ctx, f); err != nil {
			return fmt.Errorf("shares: %w", err)
		}
	}
	if err := s.seedRegister(ctx, f.Register, docs); err != nil {
		return err
	}
	if err := s.seedMessages(ctx, f.Messages); err != nil {
		return fmt.Errorf("messages: %w", err)
	}
	return s.seedConditions(ctx, docs, f.Conditions)
}

// seedRegister adds the fixture's register rows through the screen's
// operation, then runs the register's reply match on the filed documents,
// as the post-filing job would: Go's rule only (no fixture document that
// the job reads is correspondence, so no Jev step runs).
func (s *session) seedRegister(ctx context.Context, rows []registerRow, docs map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	for _, r := range rows {
		fields := map[string]any{"kind": "rfi", "title": r.Title, "status": r.Status, "question": r.Question}
		for k, v := range map[string]string{"owner_text": r.With, "system": r.System, "raised_on": r.RaisedOn} {
			if v != "" {
				fields[k] = v
			}
		}
		if err := s.ops("delivery", op("delivery.add", fields)); err != nil {
			return err
		}
	}
	for _, id := range docs {
		if err := s.l.st.QueueRegister(ctx, s.org, id); err != nil {
			return err
		}
	}
	// The fixture's text is already stored, so extraction is done: its
	// jobs are finished as the worker would finish them, which lets the
	// register's job (held until extraction is done) run.
	for {
		job, err := s.l.st.ClaimJobSince(ctx, s.org, time.Minute, []string{store.JobKindFullText}, time.Time{})
		if errors.Is(err, store.ErrIdle) {
			break
		}
		if err != nil {
			return err
		}
		if err := s.l.st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"full_text","status":"done"}`}); err != nil {
			return err
		}
	}
	w := &jobs.Worker{Store: s.l.st}
	for {
		job, err := s.l.st.ClaimJobSince(ctx, s.org, time.Minute, []string{store.JobKindRegister}, time.Time{})
		if errors.Is(err, store.ErrIdle) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := w.Perform(ctx, job); err != nil {
			return err
		}
		if err := s.l.st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"register","status":"done"}`}); err != nil {
			return err
		}
	}
}

// abn is the n-th invented, checksum-valid ABN, so each fixture firm is a
// different firm to the reader.
func abn(n int) string {
	weights := []int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	for v, found := 51000000000, -1; ; v++ {
		digits := fmt.Sprint(v)
		sum := 0
		for i, w := range weights {
			d := int(digits[i] - '0')
			if i == 0 {
				d--
			}
			sum += d * w
		}
		if sum%89 == 0 {
			if found++; found == n {
				return digits
			}
		}
	}
}

func kindOf(a activity) string {
	if a.Kind != "" {
		return a.Kind
	}
	return "activity"
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
