package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
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
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/config"
	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/latency"
	"sitewise/internal/store"
)

type document struct {
	ID       string  `json:"id"`
	Filename string  `json:"filename"`
	Status   string  `json:"status"`
	Reason   string  `json:"reason"`
	Fields   []field `json:"fields"`
}

type field struct {
	Field     string `json:"field"`
	Value     string `json:"value"`
	Band      string `json:"band"`
	DecidedBy string `json:"decided_by"`
}

type listBody struct {
	Cursor    int64      `json:"cursor"`
	Documents []document `json:"documents"`
}

type sseEvent struct {
	ID      string
	Kind    string
	Payload struct {
		DocumentID string  `json:"document_id"`
		Status     string  `json:"status"`
		Fields     []field `json:"fields"`
	}
}

func TestUploadFileCorrectReconnect(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	project := a.createProject(t, "Hale")

	list := a.list(t, project)
	stream := a.events(t, list.Cursor)

	doc := a.upload(t, project, "identity-page.pdf", http.StatusCreated)
	if doc.Status != store.StatusPending {
		t.Fatalf("upload status %s", doc.Status)
	}
	filed := stream.next(t, "filing", doc.ID)
	if filed.Payload.Status != store.StatusFiled || len(filed.Payload.Fields) == 0 {
		t.Fatalf("filing event %+v", filed.Payload)
	}

	corrected := a.correct(t, doc.ID, intake.FieldTitle, "Ground Floor Plan", http.StatusOK)
	title := fieldOf(t, corrected.Fields, intake.FieldTitle)
	if title.Value != "Ground Floor Plan" || title.DecidedBy != intake.DecidedByUser {
		t.Fatalf("correction %+v", title)
	}
	ev := stream.next(t, "correction", doc.ID)
	if got := fieldOf(t, ev.Payload.Fields, intake.FieldTitle); got.DecidedBy != intake.DecidedByUser {
		t.Fatalf("correction event %+v", got)
	}
	stream.close()

	// Reconnect: a fresh list and a stream resumed from the old cursor both
	// carry the correction.
	again := a.list(t, project)
	if got := fieldOf(t, findDoc(t, again.Documents, doc.ID).Fields, intake.FieldTitle); got.Value != "Ground Floor Plan" {
		t.Fatalf("after reconnect %+v", got)
	}
	replay := a.events(t, list.Cursor)
	replay.next(t, "filing", doc.ID)
	replay.next(t, "correction", doc.ID)
	replay.close()

	// Re-dropping stored bytes returns the same filing.
	same := a.upload(t, project, "identity-page.pdf", http.StatusOK)
	if same.ID != doc.ID {
		t.Fatalf("re-drop made %s, want %s", same.ID, doc.ID)
	}
}

func TestEachFormatAndNotFiled(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	project := a.createProject(t, "Formats")
	stream := a.events(t, a.list(t, project).Cursor)
	for _, name := range []string{"docx-table.docx", "merged-cells.xlsx"} {
		doc := a.upload(t, project, name, http.StatusCreated)
		stream.next(t, "filing", doc.ID)
	}
	scan := a.upload(t, project, "scanned-empty.pdf", http.StatusCreated)
	if ev := stream.next(t, "not_filed", scan.ID); ev.Payload.Status != store.StatusNotFiled {
		t.Fatalf("scan event %+v", ev.Payload)
	}
	txt := a.uploadBytes(t, project, "notes.txt", []byte("plain text"), http.StatusCreated)
	if txt.Status != store.StatusNotFiled || txt.Reason != intake.ReasonUnsupported {
		t.Fatalf("txt %+v", txt)
	}
	list := a.list(t, project)
	if got := findDoc(t, list.Documents, scan.ID); got.Status != store.StatusNotFiled || got.Reason != intake.ReasonNoText {
		t.Fatalf("scan %+v", got)
	}
}

func TestNoAccessFromAnotherOrg(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	b := app.member(t, "Org B")
	project := a.createProject(t, "Private")
	stream := a.events(t, 0)
	doc := a.upload(t, project, "identity-page.pdf", http.StatusCreated)
	stream.next(t, "filing", doc.ID)
	stream.close()

	checks := []struct {
		method, path string
		body         []byte
	}{
		{http.MethodGet, "/api/projects/" + project, nil},
		{http.MethodGet, "/api/projects/" + project + "/documents", nil},
		{http.MethodPost, "/api/projects/" + project + "/files?name=x.pdf", []byte("%PDF-1.4")},
		{http.MethodGet, "/api/documents/" + doc.ID, nil},
		{http.MethodPost, "/api/documents/" + doc.ID + "/filing", nil},
		{http.MethodPost, "/api/documents/" + doc.ID + "/details/reprocess", nil},
		{http.MethodPut, "/api/documents/" + doc.ID + "/fields/title", []byte(`{"value":"stolen"}`)},
	}
	for _, c := range checks {
		if got := b.status(t, c.method, c.path, c.body); got != http.StatusNotFound {
			t.Errorf("org B %s %s = %d, want 404", c.method, c.path, got)
		}
	}
	var projects []store.Project
	b.getJSON(t, "/api/projects", &projects)
	if len(projects) != 0 {
		t.Fatalf("org B sees %+v", projects)
	}
	// Org B's stream has its own cursor and none of org A's events.
	bs := b.events(t, 0)
	b.createProject(t, "Own")
	bs.expectNone(t, doc.ID, 300*time.Millisecond)
	bs.close()
	if got := fieldOf(t, a.document(t, doc.ID).Fields, intake.FieldTitle); got.Value == "stolen" {
		t.Fatal("org B corrected org A's document")
	}
}

func TestMutationsNeedOriginAndSession(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	project := a.createProject(t, "Origin")
	req, err := http.NewRequest(http.MethodPost, app.url+"/api/projects/"+project+"/files?name=a.pdf", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("no origin = %d", resp.StatusCode)
	}
	anon := &member{app: app, client: &http.Client{}}
	if got := anon.status(t, http.MethodGet, "/api/projects", nil); got != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", got)
	}
	if got := a.status(t, http.MethodPut, "/api/documents/"+newUUID(t)+"/fields/supersedes", []byte(`{"value":"x"}`)); got != http.StatusBadRequest {
		t.Fatalf("supersedes correction = %d", got)
	}
	if got := a.status(t, http.MethodGet, "/api/documents/not-a-uuid", nil); got != http.StatusNotFound {
		t.Fatalf("bad id = %d", got)
	}
}

func TestSPAFallbackAndHeaders(t *testing.T) {
	app := newApp(t)
	resp, err := http.Get(app.url + "/projects/abc")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "sitewise-test") {
		t.Fatalf("fallback %d %q", resp.StatusCode, body)
	}
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Fatalf("csp %q", csp)
	}
	resp, err = http.Get(app.url + "/assets/missing.js")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset %d", resp.StatusCode)
	}
}

// The list and correction paths have p50/p90 budgets in bench/budgets.json.
// This measures them against the local test database; Task 11 runs the same
// gate on the target host.
func TestListAndCorrectionBudgets(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	project := a.createProject(t, "Budget")
	stream := a.events(t, 0)
	var ids []string
	for _, name := range []string{"identity-page.pdf", "docx-table.docx", "merged-cells.xlsx"} {
		doc := a.upload(t, project, name, http.StatusCreated)
		stream.next(t, "filing", doc.ID)
		ids = append(ids, doc.ID)
	}
	stream.close()
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string][]int64{}
	for i := 0; i < budgets.MinSamples; i++ {
		start := time.Now()
		a.list(t, project)
		samples["project_document_list"] = append(samples["project_document_list"], time.Since(start).Microseconds())
		start = time.Now()
		a.correct(t, ids[i%len(ids)], intake.FieldTitle, fmt.Sprintf("Title %d", i), http.StatusOK)
		samples["field_correction"] = append(samples["field_correction"], time.Since(start).Microseconds())
	}
	var scoped latency.Budgets
	scoped.MinSamples = budgets.MinSamples
	for _, p := range budgets.Paths {
		if _, ok := samples[p.Name]; ok {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	if code, report := latency.Gate(scoped, samples); code != 0 {
		t.Fatal(report)
	}
}

// --- harness ---

type app struct {
	url   string
	store *store.Store
	// jevHits counts evaluation calls that reached the fake provider.
	jevHits atomic.Int64
}

func newApp(t *testing.T, adjust ...func(*httpapi.Options)) *app {
	t.Helper()
	st := openStore(t)
	blobs, err := files.Open(t.TempDir(), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join("..", "..", "data", "intake")
	cat, err := intake.LoadCatalog(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	thresholds, err := intake.LoadThresholds(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{store: st}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			a.jevHits.Add(1)
		}
		recordedJev(w, r)
	}))
	t.Cleanup(fake.Close)
	client, err := jev.New(jev.Options{
		BaseURL: fake.URL,
		APIKey:  "test-key",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(nil)
	origin := "http://" + ts.Listener.Addr().String()
	opts := httpapi.Options{
		Store:          st,
		Blobs:          blobs,
		Jev:            client,
		Catalog:        cat,
		Thresholds:     thresholds,
		Static:         fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>sitewise-test</title>")}},
		PublicOrigin:   origin,
		MaxUploadBytes: 10 << 20,
		Log:            log.New(io.Discard, "", 0),
	}
	for _, f := range adjust {
		f(&opts)
	}
	srv, err := httpapi.New(opts)
	if err != nil {
		t.Fatal(err)
	}
	ts.Config.Handler = srv
	ts.Start()
	t.Cleanup(func() {
		srv.CloseStreams()
		ts.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Wait(ctx)
	})
	a.url = ts.URL
	return a
}

// recordedJev answers every choice question with its first option at a
// middling confidence. The committed thresholds are unknown, so these answers
// stay blank; the test proves the flow, not calibration.
func recordedJev(w http.ResponseWriter, r *http.Request) {
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
		var first string
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":   config.PinnedJevModel,
		"answers": answers,
		"usage":   map[string]int{"input_tokens": 10, "output_tokens": 2},
	})
}

type member struct {
	app    *app
	client *http.Client
	orgID  string
}

func (a *app) member(t *testing.T, org string) *member {
	t.Helper()
	ctx := context.Background()
	raw, orgID, err := auth.Bootstrap(ctx, a.store, &auth.Sink{}, org, newUUID(t)+"@example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.store.DeleteOrg(context.Background(), orgID) })
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := &member{app: a, client: &http.Client{Jar: jar}, orgID: orgID}
	body, _ := json.Marshal(map[string]string{"token": raw})
	if got := m.status(t, http.MethodPost, "/api/session", body); got != http.StatusNoContent {
		t.Fatalf("sign in = %d", got)
	}
	return m
}

func (m *member) do(t *testing.T, method, path string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, m.app.url+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", m.app.url)
	if body != nil && !strings.Contains(path, "/files") {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (m *member) status(t *testing.T, method, path string, body []byte) int {
	t.Helper()
	resp := m.do(t, method, path, body)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode
}

func (m *member) call(t *testing.T, method, path string, body []byte, want int, out any) {
	t.Helper()
	resp := m.do(t, method, path, body)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func (m *member) getJSON(t *testing.T, path string, out any) {
	t.Helper()
	m.call(t, http.MethodGet, path, nil, http.StatusOK, out)
}

func (m *member) createProject(t *testing.T, name string) string {
	t.Helper()
	var out struct {
		ID string `json:"id"`
	}
	body, _ := json.Marshal(map[string]string{"name": name})
	m.call(t, http.MethodPost, "/api/projects", body, http.StatusCreated, &out)
	return out.ID
}

func (m *member) list(t *testing.T, project string) listBody {
	t.Helper()
	var out listBody
	m.getJSON(t, "/api/projects/"+project+"/documents", &out)
	return out
}

func (m *member) document(t *testing.T, id string) document {
	t.Helper()
	var out document
	m.getJSON(t, "/api/documents/"+id, &out)
	return out
}

func (m *member) upload(t *testing.T, project, fixture string, want int) document {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "identity", fixture))
	if err != nil {
		t.Fatal(err)
	}
	return m.uploadBytes(t, project, fixture, body, want)
}

func (m *member) uploadBytes(t *testing.T, project, name string, body []byte, want int) document {
	t.Helper()
	var out document
	m.call(t, http.MethodPost, "/api/projects/"+project+"/files?name="+url.QueryEscape(name), body, want, &out)
	return out
}

func (m *member) correct(t *testing.T, docID, fieldName, value string, want int) document {
	t.Helper()
	var out document
	body, _ := json.Marshal(map[string]string{"value": value})
	m.call(t, http.MethodPut, "/api/documents/"+docID+"/fields/"+fieldName, body, want, &out)
	return out
}

type stream struct {
	events chan sseEvent
	cancel context.CancelFunc
}

func (m *member) events(t *testing.T, after int64) *stream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/events?after=%d", m.app.url, after), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("events = %d", resp.StatusCode)
	}
	s := &stream{events: make(chan sseEvent, 64), cancel: cancel}
	t.Cleanup(s.close)
	go func() {
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		var ev sseEvent
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, "id: "):
				ev.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				ev.Kind = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				var wire struct {
					Payload json.RawMessage `json:"payload"`
				}
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &wire) == nil {
					_ = json.Unmarshal(wire.Payload, &ev.Payload)
				}
			case line == "" && ev.Kind != "":
				s.events <- ev
				ev = sseEvent{}
			}
		}
	}()
	return s
}

func (s *stream) next(t *testing.T, kind, docID string) sseEvent {
	t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case ev := <-s.events:
			if ev.Kind == kind && ev.Payload.DocumentID == docID {
				return ev
			}
		case <-timeout:
			t.Fatalf("no %s event for %s", kind, docID)
		}
	}
}

func (s *stream) expectNone(t *testing.T, docID string, wait time.Duration) {
	t.Helper()
	deadline := time.After(wait)
	for {
		select {
		case ev := <-s.events:
			if ev.Payload.DocumentID == docID {
				t.Fatalf("foreign event %+v", ev)
			}
		case <-deadline:
			return
		}
	}
}

func (s *stream) close() { s.cancel() }

func fieldOf(t *testing.T, list []field, name string) field {
	t.Helper()
	for _, f := range list {
		if f.Field == name {
			return f
		}
	}
	t.Fatalf("no %s field in %+v", name, list)
	return field{}
}

func findDoc(t *testing.T, list []document, id string) document {
	t.Helper()
	for _, d := range list {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("document %s not listed", id)
	return document{}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name := strings.TrimPrefix(u.Path, "/"); name != "sitewise_test" {
		t.Fatalf("refusing to use database %q", name)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st
}

func newUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
