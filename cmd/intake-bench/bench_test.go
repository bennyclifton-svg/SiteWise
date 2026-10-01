package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
)

type stubTransport struct{ calls int }

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

func TestDegradingStallsEveryNthUntilDeadline(t *testing.T) {
	next := &stubTransport{}
	d := &degrading{next: next, every: 3}
	for i := 1; i <= 6; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://x", nil)
		_, err := d.RoundTrip(req)
		cancel()
		if stalled := i%3 == 0; stalled != (err != nil) {
			t.Fatalf("request %d err %v", i, err)
		}
	}
	if next.calls != 4 || d.stall.Load() != 2 {
		t.Fatalf("calls %d stalled %d", next.calls, d.stall.Load())
	}
}

func TestLastByteMarksTheFinalRead(t *testing.T) {
	lb := newLastByte([]byte("abcdef"))
	buf := make([]byte, 4)
	if _, err := lb.Read(buf); err != nil {
		t.Fatal(err)
	}
	select {
	case <-lb.done:
		t.Fatal("marked before the final byte")
	default:
	}
	if _, err := lb.Read(buf); err != nil {
		t.Fatal(err)
	}
	select {
	case <-lb.done:
	default:
		t.Fatal("final byte not marked")
	}
}

func TestArrivalsKeepTheFirstTerminalEvent(t *testing.T) {
	a := newArrivals()
	stream := "retry: 2000\n\nid: 1\nevent: correction\ndata: {\"id\":1,\"kind\":\"correction\",\"document_id\":\"d\"}\n\n" +
		"id: 2\nevent: filing\ndata: {\"id\":2,\"kind\":\"filing\",\"document_id\":\"d\"}\n\n" +
		"id: 3\nevent: not_filed\ndata: {\"id\":3,\"kind\":\"not_filed\",\"document_id\":\"d\"}\n\n"
	if err := readEvents(strings.NewReader(stream), a.record); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, kind, err := a.await(ctx, "d")
	if err != nil || kind != "filing" || a.cursor() != 3 {
		t.Fatalf("kind %s err %v cursor %d", kind, err, a.cursor())
	}
}

func TestBenchDatabaseMustBeDedicated(t *testing.T) {
	env := func(dsn string) func(string) string {
		return func(k string) string {
			if k == "SITEWISE_TEST_DATABASE_URL" {
				return dsn
			}
			return ""
		}
	}
	if _, err := benchDSN(env("postgres://u@h/sitewise")); err == nil {
		t.Fatal("the production database name must be refused")
	}
	if _, err := benchDSN(env("postgres://u@h/sitewise_test")); err != nil {
		t.Fatal(err)
	}
}

func docx(t *testing.T, lines ...string) []byte {
	t.Helper()
	var body strings.Builder
	body.WriteString(`<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, l := range lines {
		fmt.Fprintf(&body, `<w:p><w:r><w:t>%s</w:t></w:r></w:p>`, l)
	}
	body.WriteString(`</w:body></w:document>`)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	io.WriteString(w, body.String())
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// fakeJev picks the first option of every question after a short delay.
func fakeJev(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Questions map[string]struct {
				Criteria map[string]json.RawMessage `json:"criteria"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
			t.Error(err)
			return
		}
		time.Sleep(15 * time.Millisecond)
		answers := map[string]any{}
		for id, q := range call.Questions {
			opts := make([]string, 0, len(q.Criteria))
			for o := range q.Criteria {
				opts = append(opts, o)
			}
			sort.Strings(opts)
			probs := map[string]float64{}
			for _, o := range opts {
				probs[o] = 0
			}
			probs[opts[0]] = 1
			answers[id] = map[string]any{"type": "choice", "choice": opts[0], "probabilities": probs, "confidence": 0.9}
		}
		json.NewEncoder(w).Encode(map[string]any{"model": config.PinnedJevModel, "answers": answers, "usage": map[string]int{}})
	}))
}

// TestBenchEndToEnd records a small corpus against a fake provider, then
// benches it in replay: uploads, SSE arrival, stage samples and the gate.
func TestBenchEndToEnd(t *testing.T) {
	if os.Getenv("SITEWISE_TEST_DATABASE_URL") == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "corpus", "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	var key strings.Builder
	key.WriteString("schema_version: 1\ncorpus: demo\nlabel_source: award-doc\nadjudication: unreviewed\nentries:\n")
	for n := 1; n <= 6; n++ {
		name := fmt.Sprintf("sheet-%d.docx", n)
		body := docx(t, fmt.Sprintf("Drawing No E-%03d", n), fmt.Sprintf("Ref Drawing No X-%03d", n), "Rev C1", "Title Lighting")
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&key, "- id: demo-%d\n  corpus_path: %q\n  page: null\n  doc_type: drawing\n  expected:\n    number: \"E-%03d\"\n  flags: []\n  sha256: %s\n", n, name, n, eval.SHA256Hex(body))
	}
	keys := filepath.Join(dir, "keys")
	os.MkdirAll(keys, 0o755)
	if err := os.WriteFile(filepath.Join(keys, "demo.yaml"), []byte(key.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := eval.Manifest{
		SchemaVersion: 1, QuestionVersion: intake.QuestionVersion, Model: config.PinnedJevModel, Privacy: "test",
		CorpusRoot: filepath.Join(dir, "corpus"), KeysRoot: keys, PrivateDir: filepath.Join(dir, "private"),
		Corpora:              []eval.Corpus{{Name: "demo", Key: "demo.yaml", KeySHA256: eval.SHA256Hex([]byte(key.String())), Root: "demo"}},
		AuthoritativeSources: []string{"award-doc"}, AuthoritativeFields: []string{"number"},
		Split: eval.SplitPolicy{Salt: "t", CalibrationPercent: 50}, Gate: eval.GatePolicy{Z: 1.645},
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	writeTestJSON(t, manifestPath, manifest)

	set, err := eval.BuildCases(manifest)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := intake.LoadCatalog(filepath.Join("..", "..", "data", "intake"))
	if err != nil {
		t.Fatal(err)
	}
	fake := fakeJev(t)
	defer fake.Close()
	var log bytes.Buffer
	rec := eval.NewRecorder(http.DefaultTransport, &log)
	client, err := jev.New(jev.Options{BaseURL: fake.URL, APIKey: "k", Transport: rec})
	if err != nil {
		t.Fatal(err)
	}
	ev := eval.Evaluator{Catalog: cat, Thresholds: intake.Thresholds{QuestionVersion: intake.QuestionVersion}, Asker: client, Labeler: rec}
	if _, err := ev.Run(context.Background(), set); err != nil {
		t.Fatal(err)
	}
	recordings := filepath.Join(dir, "recordings.jsonl")
	if err := os.WriteFile(recordings, log.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	budgets := filepath.Join(dir, "budgets.json")
	writeTestJSON(t, budgets, map[string]any{
		"min_samples": 2,
		"paths": []map[string]any{
			{"name": "whole_intake", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "identity_text_extraction", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "jev_admission_request", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "commit_sse_enqueue", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "project_document_list", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "field_correction", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "project_invite_auth", "p50_us": 30_000_000, "p90_us": 30_000_000},
			{"name": "sse_reconnect", "p50_us": 30_000_000, "p90_us": 30_000_000},
		},
	})
	out := filepath.Join(dir, "results.json")
	samplesOut := filepath.Join(dir, "samples.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"-manifest", manifestPath, "-budgets", budgets, "-data", filepath.Join("..", "..", "data", "intake"),
		"-recordings", recordings, "-out", out, "-samples-out", samplesOut,
		"-rounds", "2", "-concurrency", "3", "-background", "1", "-degrade", "50", "-api-samples", "3",
	}, os.Getenv, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	t.Log(stdout.String())
	var res Results
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatal(err)
	}
	if res.Paths["whole_intake"].N != 12 {
		t.Fatalf("every upload of both rounds is a whole-intake sample: %+v", res.Paths["whole_intake"])
	}
	if res.Workload.StalledRequests == 0 || res.Outcomes["grey_filings"] == 0 {
		t.Fatalf("degraded filings must be in the samples: %+v %+v", res.Workload, res.Outcomes)
	}
	if res.Jev != "replayed-recorded-latency" || res.Host.CPUs == 0 {
		t.Fatalf("%+v", res)
	}
	// A stall lasts the full interactive deadline, so the degraded share
	// shows in the tail rather than being dropped.
	if res.Paths["whole_intake"].MaxUS < jev.InteractiveDeadline.Microseconds() {
		t.Fatalf("degraded filings missing from the tail: %+v", res.Paths["whole_intake"])
	}
}

func writeTestJSON(t *testing.T, file string, v any) {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
