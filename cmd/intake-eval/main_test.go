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
	"strings"
	"testing"

	"sitewise/internal/config"
	"sitewise/internal/eval"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
)

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

type workspace struct {
	manifest, data, recordings, out string
	m                               eval.Manifest
}

// setup writes a corpus of sheets with two drawing numbers each, so every
// number goes to Jev, and copies the intake data so -fit can write to it.
func setup(t *testing.T) workspace {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "corpus", "demo")
	var key strings.Builder
	key.WriteString("schema_version: 1\ncorpus: demo\nlabel_source: award-doc\nadjudication: unreviewed\nentries:\n")
	for n := 1; n <= 12; n++ {
		rel := fmt.Sprintf("SET%d/sheet-%d.docx", n, n)
		body := docx(t, fmt.Sprintf("Drawing No E-%03d", n), fmt.Sprintf("Ref Drawing No X-%03d", n), "Rev C1", "Title Lighting", "Electrical", "Construction")
		if err := os.MkdirAll(filepath.Join(root, fmt.Sprintf("SET%d", n)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), body, 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&key, "- id: demo-%d\n  corpus_path: %q\n  page: null\n  doc_type: drawing\n  expected:\n    number: \"E-%03d\"\n  flags: []\n  sha256: %s\n", n, rel, n, eval.SHA256Hex(body))
	}
	keys := filepath.Join(dir, "keys")
	os.MkdirAll(keys, 0o755)
	if err := os.WriteFile(filepath.Join(keys, "demo.yaml"), []byte(key.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "intake")
	os.MkdirAll(data, 0o755)
	for _, f := range []string{"disciplines.json", "kinds.json", "lifecycle.json", "thresholds.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "data", "intake", f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(data, f), body, 0o644)
	}
	m := eval.Manifest{
		SchemaVersion: 1, QuestionVersion: intake.QuestionVersion, Model: config.PinnedJevModel, Privacy: "test",
		CorpusRoot: filepath.Join(dir, "corpus"), KeysRoot: keys, PrivateDir: filepath.Join(dir, "private"),
		Corpora:              []eval.Corpus{{Name: "demo", Key: "demo.yaml", KeySHA256: eval.SHA256Hex([]byte(key.String())), Root: "demo"}},
		AuthoritativeSources: []string{"award-doc"}, AuthoritativeFields: []string{"number"},
		Split:    eval.SplitPolicy{Salt: "cmd", CalibrationPercent: 50},
		Fit:      map[string]eval.FitPolicy{"number": {Z: 1.645, MinSamples: 2, MinBandSamples: 1, GreenLower: 0.1, AmberLower: 0.05, MaxErrors: -1}},
		Gate:     eval.GatePolicy{Z: 1.645, MinCases: 1, MinHeldout: map[string]int{"number": 1}},
		Baseline: filepath.Join(dir, "baseline.json"),
	}
	manifest := filepath.Join(dir, "manifest.json")
	body, _ := json.Marshal(m)
	os.WriteFile(manifest, body, 0o644)
	return workspace{manifest: manifest, data: data, recordings: filepath.Join(dir, "private", "recordings.jsonl"), out: filepath.Join(dir, "report.json"), m: m}
}

// recordFake makes the recording a -live run would, against a fake provider
// that picks the E- number at 0.97.
func recordFake(t *testing.T, w workspace) {
	t.Helper()
	fake := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var call struct {
			Questions map[string]struct {
				Criteria map[string]string `json:"criteria"`
			} `json:"questions"`
		}
		json.NewDecoder(r.Body).Decode(&call)
		answers := map[string]any{}
		for id, q := range call.Questions {
			for opt, label := range q.Criteria {
				if id == intake.FieldNumber && strings.HasPrefix(label, "E-") {
					probs := map[string]float64{}
					for o := range q.Criteria {
						probs[o] = 0
					}
					probs[opt] = 1
					answers[id] = map[string]any{"type": "choice", "choice": opt, "probabilities": probs, "confidence": 0.97}
				}
			}
		}
		json.NewEncoder(rw).Encode(map[string]any{"model": config.PinnedJevModel, "answers": answers, "usage": map[string]int{}})
	}))
	defer fake.Close()
	set, err := eval.BuildCases(w.m)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := intake.LoadCatalog(w.data)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(w.recordings), 0o700)
	f, err := os.Create(w.recordings)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rec := eval.NewRecorder(http.DefaultTransport, f)
	client, err := jev.New(jev.Options{BaseURL: fake.URL, APIKey: "k", Transport: rec})
	if err != nil {
		t.Fatal(err)
	}
	ev := eval.Evaluator{Catalog: cat, Thresholds: intake.Thresholds{QuestionVersion: intake.QuestionVersion}, Asker: client, Labeler: rec}
	if _, err := ev.Run(context.Background(), set); err != nil {
		t.Fatal(err)
	}
}

func evalRun(t *testing.T, w workspace, extra ...string) (int, string) {
	t.Helper()
	args := append([]string{"-manifest", w.manifest, "-data", w.data, "-out", w.out}, extra...)
	var out bytes.Buffer
	code := run(args, func(string) string { return "" }, &out, &out)
	return code, out.String()
}

func TestEvalWorkflowFailsClosedThenAcceptsAReviewedBaseline(t *testing.T) {
	w := setup(t)
	if code, out := evalRun(t, w, "-replay"); code == 0 || !strings.Contains(out, "no recordings") {
		t.Fatalf("replay without recordings must fail: %d\n%s", code, out)
	}
	if code, out := evalRun(t, w, "-live"); code == 0 || !strings.Contains(out, "SITEWISE_JEV_API_KEY") {
		t.Fatalf("live without a key must fail: %d\n%s", code, out)
	}
	if code, out := evalRun(t, w, "-check"); code != 0 || !strings.Contains(out, "verified 12 cases") {
		t.Fatalf("check: %d\n%s", code, out)
	}

	recordFake(t, w)
	code, out := evalRun(t, w, "-replay", "-fit")
	if code == 0 || !strings.Contains(out, "no accepted baseline") {
		t.Fatalf("first replay must fail on the missing baseline: %d\n%s", code, out)
	}
	thresholds, err := intake.LoadThresholds(w.data)
	if err != nil {
		t.Fatal(err)
	}
	if got := thresholds.Questions[intake.FieldNumber]; len(got) != 1 || *got[0].Green != 0.97 {
		t.Fatalf("-fit must write the calibration cut-off: %+v\n%s", thresholds.Questions, out)
	}
	if len(thresholds.Fit) == 0 {
		t.Fatal("the fit must record how it was made")
	}
	if code, out := evalRun(t, w, "-replay", "-accept"); code != 0 {
		t.Fatalf("accept: %d\n%s", code, out)
	}
	if code, out := evalRun(t, w, "-replay"); code != 0 || !strings.Contains(out, "gate passed") {
		t.Fatalf("replay against the accepted baseline: %d\n%s", code, out)
	}
	report, err := os.ReadFile(w.out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(report), "Lighting") || strings.Contains(string(report), "E-001") {
		t.Fatal("the committed report must not carry document text or labels")
	}

	// An edited recording fails its own request-hash check.
	recs, _ := os.ReadFile(w.recordings)
	os.WriteFile(w.recordings, bytes.ReplaceAll(recs, []byte(`\"Which candidate`), []byte(`\"Changed`)), 0o600)
	if code, out := evalRun(t, w, "-replay"); code == 0 {
		t.Fatalf("a stale recording must fail replay: %d\n%s", code, out)
	}
}
