package eval

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/config"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// docx is a minimal Word file whose body is one paragraph per line.
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
	if _, err := io.WriteString(w, body.String()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hexSum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type fixtureCorpus struct {
	dir      string
	manifest Manifest
	keyPath  string
}

type fixtureEntry struct {
	id, path, number, revision, title, labelSource string
	page                                           int
	flags                                          []string
	body                                           []byte
	sha                                            string
}

// writeCorpus lays out files and an answer key the way the clerk keys are
// shaped, and a manifest that pins the key's hash.
func writeCorpus(t *testing.T, entries []fixtureEntry) fixtureCorpus {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "corpus", "demo")
	var key strings.Builder
	key.WriteString("schema_version: 1\ncorpus: demo\nlabel_source: award-doc\nadjudication: unreviewed\nentries:\n")
	for _, e := range entries {
		full := filepath.Join(root, filepath.FromSlash(e.path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if e.body != nil {
			if err := os.WriteFile(full, e.body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		sha := e.sha
		if sha == "" {
			sha = hexSum(e.body)
		}
		page := "null"
		if e.page > 0 {
			page = fmt.Sprint(e.page)
		}
		fmt.Fprintf(&key, "- id: %s\n  corpus_path: %q\n  page: %s\n  doc_type: drawing\n", e.id, e.path, page)
		if e.labelSource != "" {
			fmt.Fprintf(&key, "  label_source: %s\n", e.labelSource)
		}
		key.WriteString("  expected:\n    category: consultant.electrical\n")
		for _, kv := range [][2]string{{"number", e.number}, {"revision", e.revision}, {"title", e.title}} {
			if kv[1] == "" {
				fmt.Fprintf(&key, "    %s: null\n", kv[0])
				continue
			}
			fmt.Fprintf(&key, "    %s: %q\n", kv[0], kv[1])
		}
		key.WriteString("    date: null\n")
		fmt.Fprintf(&key, "  flags: [%s]\n  sha256: %s\n", strings.Join(e.flags, ", "), sha)
	}
	keysDir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(keysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(keysDir, "demo-design-docs.yaml")
	if err := os.WriteFile(keyPath, []byte(key.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		SchemaVersion:        1,
		QuestionVersion:      intake.QuestionVersion,
		Model:                config.PinnedJevModel,
		Privacy:              "test",
		CorpusRoot:           filepath.Join(dir, "corpus"),
		KeysRoot:             keysDir,
		Corpora:              []Corpus{{Name: "demo", Key: "demo-design-docs.yaml", KeySHA256: hexSum([]byte(key.String())), Root: "demo"}},
		AuthoritativeSources: []string{"award-doc"},
		AuthoritativeFields:  []string{"number", "revision", "title", "date"},
		KindMap:              map[string]string{"drawing": "drawing"},
		Flags:                map[string][]string{"register_title_typo": {"title"}, "windows_short_filename": {}},
		Split:                SplitPolicy{Salt: "test", CalibrationPercent: 50},
		Fit:                  map[string]FitPolicy{"number": {Z: 1.645, MinSamples: 2, MinBandSamples: 1, GreenLower: 0.1, AmberLower: 0.05, MaxErrors: -1}},
		Gate:                 GatePolicy{Z: 1.645, MinCases: 1, MinHeldout: map[string]int{"number": 1}},
	}
	return fixtureCorpus{dir: dir, manifest: m, keyPath: keyPath}
}

func sheet(n int) fixtureEntry {
	num := fmt.Sprintf("E-%03d", n)
	return fixtureEntry{
		id:       fmt.Sprintf("demo-%d", n),
		path:     fmt.Sprintf("ELEC%d/%s Lighting [C1].docx", n, num),
		number:   num,
		revision: "C1",
		title:    "Lighting",
		body:     nil,
	}
}

func withBody(t *testing.T, e fixtureEntry) fixtureEntry {
	e.body = docx(t, "Drawing No "+e.number, "Kind: Drawing", "Rev "+e.revision, "Title "+e.title, "Discipline: Electrical", "Construction")
	return e
}

func TestBuildCasesChecksKeyAndCorpusHashes(t *testing.T) {
	fc := writeCorpus(t, []fixtureEntry{withBody(t, sheet(1))})
	if _, err := BuildCases(fc.manifest); err != nil {
		t.Fatal(err)
	}
	bad := fc.manifest
	bad.Corpora = append([]Corpus(nil), bad.Corpora...)
	bad.Corpora[0].KeySHA256 = strings.Repeat("0", 64)
	if _, err := BuildCases(bad); err == nil || !strings.Contains(err.Error(), "answer key") {
		t.Fatalf("a changed answer key must fail: %v", err)
	}

	missing := sheet(2)
	missing.sha = strings.Repeat("a", 64)
	fc = writeCorpus(t, []fixtureEntry{withBody(t, sheet(1)), missing})
	if _, err := BuildCases(fc.manifest); err == nil || !strings.Contains(err.Error(), "corpus") {
		t.Fatalf("a missing corpus file must fail: %v", err)
	}

	changed := withBody(t, sheet(3))
	changed.sha = strings.Repeat("b", 64)
	fc = writeCorpus(t, []fixtureEntry{changed})
	if _, err := BuildCases(fc.manifest); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("a changed corpus file must fail: %v", err)
	}
}

func TestBuildCasesRejectsUnclassifiedFlagsAndDuplicateIDs(t *testing.T) {
	e := withBody(t, sheet(1))
	e.flags = []string{"never_seen_before"}
	fc := writeCorpus(t, []fixtureEntry{e})
	if _, err := BuildCases(fc.manifest); err == nil || !strings.Contains(err.Error(), "never_seen_before") {
		t.Fatalf("an unclassified flag must fail: %v", err)
	}
	a, b := withBody(t, sheet(1)), withBody(t, sheet(2))
	b.id = a.id
	fc = writeCorpus(t, []fixtureEntry{a, b})
	if _, err := BuildCases(fc.manifest); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate ids must fail: %v", err)
	}
}

func TestBuildCasesScopesLabels(t *testing.T) {
	typo := withBody(t, sheet(2))
	typo.flags = []string{"register_title_typo"}
	model := withBody(t, sheet(3))
	model.labelSource = "model"
	later := withBody(t, sheet(4))
	later.page = 2
	first := withBody(t, sheet(5))
	first.page = 1
	dup := withBody(t, sheet(6))
	dup.id = "demo-dup"
	dup.body = withBody(t, sheet(1)).body
	dup.path = "ELEC1/copy.docx"
	nullNumber := withBody(t, sheet(7))
	nullNumber.number = ""

	fc := writeCorpus(t, []fixtureEntry{withBody(t, sheet(1)), typo, model, later, first, dup, nullNumber})
	set, err := BuildCases(fc.manifest)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Case{}
	for _, c := range set.Cases {
		byID[c.ID] = c
	}
	if _, ok := byID["demo-4"]; ok {
		t.Fatal("a later page is not the identity page of its file")
	}
	if _, ok := byID["demo-5"]; !ok {
		t.Fatal("page 1 is the identity page")
	}
	if _, ok := byID["demo-dup"]; ok {
		t.Fatal("byte-identical files are one filing")
	}
	if set.Excluded["later_page"] != 1 || set.Excluded["duplicate_file"] != 1 {
		t.Fatalf("exclusions must be counted: %v", set.Excluded)
	}
	if _, ok := byID["demo-2"].Labels[intake.FieldTitle]; ok {
		t.Fatal("a flagged register title is not scored")
	}
	if l := byID["demo-3"].Labels[intake.FieldNumber]; l.Authoritative {
		t.Fatal("model labels are diagnostic")
	}
	if l := byID["demo-1"].Labels[intake.FieldNumber]; !l.Authoritative || l.Value != "E-001" {
		t.Fatalf("%+v", l)
	}
	if l := byID["demo-1"].Labels[intake.FieldDiscipline]; l.Authoritative {
		t.Fatal("category is not an authoritative field")
	}
	if _, ok := byID["demo-7"].Labels[intake.FieldNumber]; ok {
		t.Fatal("a null label is not scored")
	}
	if _, ok := byID["demo-1"].Labels[intake.FieldDate]; ok {
		t.Fatal("a null date is not scored")
	}
	for _, c := range set.Cases {
		if c.Split != SplitCalibration && c.Split != SplitHeldout {
			t.Fatalf("%s split %q", c.ID, c.Split)
		}
	}
}

func TestRecordThenReplayServesExactBytes(t *testing.T) {
	var hits int
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			return
		}
		hits++
		if r.Header.Get("Authorization") == "" {
			t.Error("live call without credentials")
		}
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"jev-1.13.0","answers":{"q":{"type":"noul","noul":0.8}},"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer live.Close()

	var log bytes.Buffer
	rec := NewRecorder(http.DefaultTransport, &log)
	rec.SetCase("case-1")
	client := newClient(t, live.URL, rec)
	if err := client.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	call := jev.Call{State: "s", Questions: map[string]jev.Question{"q": {Type: jev.TypeNoul, Instructions: "q?"}}}
	first, err := client.Ask(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(log.String(), "Bearer") || strings.Contains(log.String(), "test-key") {
		t.Fatal("a recording must not hold the credential")
	}
	records, err := ReadRecordings(&log)
	if err != nil || len(records) != 1 || records[0].Case != "case-1" || records[0].Status != 200 {
		t.Fatalf("%+v %v", records, err)
	}

	rep := NewReplayer(records, false)
	replayed, err := newClient(t, "https://replay.invalid", rep).Ask(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("replay reached the network: %d", hits)
	}
	if replayed.Answers["q"].Noul != first.Answers["q"].Noul {
		t.Fatalf("%+v %+v", replayed, first)
	}
	other := call
	other.State = "changed"
	if _, err := newClient(t, "https://replay.invalid", rep).Ask(context.Background(), other); err == nil {
		t.Fatal("a changed request must miss")
	}
	if rep.Misses() != 1 {
		t.Fatalf("misses %d", rep.Misses())
	}
}

func newClient(t *testing.T, base string, rt http.RoundTripper) *jev.Client {
	t.Helper()
	c, err := jev.New(jev.Options{BaseURL: base, APIKey: "test-key", Transport: rt, BreakerThreshold: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// ambiguous carries two drawing numbers, so the number goes to Jev.
func ambiguous(t *testing.T, n int) fixtureEntry {
	e := sheet(n)
	e.path = fmt.Sprintf("SET%d/sheet-%d.docx", n, n)
	e.body = docx(t, "Drawing No "+e.number, "Ref Drawing No X-"+fmt.Sprint(900+n), "Rev C1", "Title Lighting", "Discipline: Electrical", "Construction")
	return e
}

// pickNumber answers every number question with the option whose label
// starts with E-, at a fixed confidence.
func pickNumber(t *testing.T, confidence float64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Questions map[string]struct {
				Criteria map[string]json.RawMessage `json:"criteria"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
			t.Error(err)
			return
		}
		answers := map[string]any{}
		for id, q := range call.Questions {
			for opt, raw := range q.Criteria {
				label := optionValue(raw)
				if id == intake.FieldNumber && strings.HasPrefix(label, "E-") || id != intake.FieldNumber && opt == "none" {
					probs := map[string]float64{}
					for o := range q.Criteria {
						probs[o] = 0
					}
					probs[opt] = 1
					answers[id] = map[string]any{"type": "choice", "choice": opt, "probabilities": probs, "confidence": confidence}
				}
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"model": config.PinnedJevModel, "answers": answers, "usage": map[string]int{}})
	})
}

func TestEvaluateScoresFitsAndGates(t *testing.T) {
	var entries []fixtureEntry
	for n := 1; n <= 8; n++ {
		entries = append(entries, ambiguous(t, n))
	}
	entries = append(entries, withBody(t, sheet(20)))
	fc := writeCorpus(t, entries)
	set, err := BuildCases(fc.manifest)
	if err != nil {
		t.Fatal(err)
	}
	live := httptest.NewServer(pickNumber(t, 0.97))
	defer live.Close()
	var log bytes.Buffer
	rec := NewRecorder(http.DefaultTransport, &log)
	cat, err := intake.LoadCatalog(filepath.Join("..", "..", "data", "intake"))
	if err != nil {
		t.Fatal(err)
	}
	unknown := intake.Thresholds{QuestionVersion: intake.QuestionVersion, Reconciliation: "test"}
	ev := Evaluator{Catalog: cat, Thresholds: unknown, Asker: newClient(t, live.URL, rec), Labeler: rec}
	run, err := ev.Run(context.Background(), set)
	if err != nil {
		t.Fatal(err)
	}
	if run.Calls != 8 {
		t.Fatalf("one fan-out per ambiguous filing, none for the settled one: %d", run.Calls)
	}

	// Nothing is applied by Jev with unknown thresholds; the settled sheet
	// is still filed by rules.
	all := run.Outcomes(SplitCalibration, true)
	all = append(all, run.Outcomes(SplitHeldout, true)...)
	m := Summarize(all, 1.645)[intake.FieldNumber]
	if m.Scored != 9 || m.Applied != 1 || m.Correct != 1 {
		t.Fatalf("%+v", m)
	}

	fits := Fit(run.Answers(SplitCalibration), fc.manifest.Fit)
	if n := len(run.Answers(SplitCalibration)); n < 2 {
		t.Fatalf("fixture needs calibration answers on both splits; got %d calibration answers", n)
	}
	if len(run.Answers(SplitHeldout)) == 0 {
		t.Fatal("fixture needs held-out answers")
	}
	thresholds := ThresholdsFrom(fits, "fitted in test", nil)
	if got := thresholds.Questions[intake.FieldNumber]; len(got) != 1 || *got[0].Green != 0.97 {
		t.Fatalf("%+v", thresholds.Questions)
	}

	// Replaying with the fitted cut-offs now applies the number.
	records, err := ReadRecordings(&log)
	if err != nil {
		t.Fatal(err)
	}
	rep := NewReplayer(records, false)
	ev = Evaluator{Catalog: cat, Thresholds: thresholds, Asker: newClient(t, "https://replay.invalid", rep)}
	again, err := ev.Run(context.Background(), set)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Misses() != 0 {
		t.Fatalf("replay misses %d", rep.Misses())
	}
	report := NewReport(fc.manifest, set, again, "replay")
	if report.Metrics[SplitHeldout][TierAuthoritative][intake.FieldNumber].Applied == 0 && report.Metrics[SplitCalibration][TierAuthoritative][intake.FieldNumber].Applied < 2 {
		t.Fatalf("fitted cut-off did not apply: %+v", report.Metrics)
	}

	// The gate fails without a baseline, then passes against its own.
	if failures := report.Gate(fc.manifest, nil); !containsText(failures, "baseline") {
		t.Fatalf("missing baseline must fail: %v", failures)
	}
	base := report.Baseline()
	if failures := report.Gate(fc.manifest, &base); len(failures) != 0 {
		t.Fatalf("%v", failures)
	}
	strict := fc.manifest
	strict.Gate.MinHeldout = map[string]int{"number": 1000}
	if failures := report.Gate(strict, &base); !containsText(failures, "too few") {
		t.Fatalf("too few samples must fail: %v", failures)
	}
	worse := base
	worse.Metrics = map[string]FieldMetrics{intake.FieldNumber: {Correct: 1000}}
	if failures := report.Gate(fc.manifest, &worse); !containsText(failures, "below baseline") {
		t.Fatalf("a regression must fail: %v", failures)
	}
}

func TestReplayMissFailsTheRun(t *testing.T) {
	fc := writeCorpus(t, []fixtureEntry{ambiguous(t, 1)})
	set, err := BuildCases(fc.manifest)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := intake.LoadCatalog(filepath.Join("..", "..", "data", "intake"))
	if err != nil {
		t.Fatal(err)
	}
	rep := NewReplayer(nil, false)
	ev := Evaluator{Catalog: cat, Thresholds: intake.Thresholds{QuestionVersion: intake.QuestionVersion}, Asker: newClient(t, "https://replay.invalid", rep), Misses: rep.Misses}
	if _, err := ev.Run(context.Background(), set); !errors.Is(err, ErrReplayMiss) {
		t.Fatalf("want replay miss, got %v", err)
	}
}

func TestSameValueComparesLikeIntake(t *testing.T) {
	cases := []struct {
		field, a, b string
		want        bool
	}{
		{intake.FieldNumber, "e-01", "E-01", true},
		{intake.FieldNumber, "E 01", "E-01", false},
		{intake.FieldTitle, "LEVEL C1 CAR PARK – LIGHTING", "Level C1 Car Park - Lighting", true},
		{intake.FieldRevision, "c1", "C1", true},
		{intake.FieldDate, "2023-11-06", "06/11/2023", true},
		{intake.FieldDate, "2023-11-06", "11/06/2023", false},
		{intake.FieldDate, "2023-12-06", "6 DEC 2023", true},
		{intake.FieldDate, "2023-04-06", "6/04/2023", true},
		{intake.FieldDate, "2023-12-05", "05-12-2023", true},
		{intake.FieldDate, "2023-12-15", "15 December 2023", true},
		{intake.FieldDate, "2023-12-05", "90/90/90", false},
		{intake.FieldDiscipline, "consultant.electrical", "consultant.electrical", true},
	}
	for _, c := range cases {
		if got := SameValue(c.field, c.a, c.b); got != c.want {
			t.Errorf("%s %q %q got %v", c.field, c.a, c.b, got)
		}
	}
}

func containsText(list []string, s string) bool {
	for _, l := range list {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

func TestSupersessionTruthIsTheLatestEarlierRevision(t *testing.T) {
	auth := func(v string) Label { return Label{Value: v, Authoritative: true} }
	c := Case{ID: "c", Labels: map[string]Label{intake.FieldNumber: auth("E-01"), intake.FieldRevision: auth("C3")}}
	prior := func(id, number, rev string) filedPrior {
		return filedPrior{doc: store.NumberedDocument{ID: id, Number: number, Revision: rev}, auth: true}
	}
	got, ok := supersessionTruth(c, []filedPrior{prior("p1", "E-01", "C1"), prior("p2", "e-01", "C2"), prior("x", "E-02", "C9")})
	if !ok || got.Value != "p2" || !got.Authoritative {
		t.Fatalf("%+v %v", got, ok)
	}
	got, ok = supersessionTruth(c, []filedPrior{prior("x", "E-02", "C1")})
	if !ok || got.Value != choiceNew {
		t.Fatalf("no same-number prior means new: %+v %v", got, ok)
	}
	got, ok = supersessionTruth(c, []filedPrior{prior("later", "E-01", "C4")})
	if !ok || got.Value != choiceNew {
		t.Fatalf("a later prior is not replaced: %+v %v", got, ok)
	}
	if _, ok := supersessionTruth(c, []filedPrior{prior("odd", "E-01", "P1")}); ok {
		t.Fatal("an unrelated revision series is not ordered")
	}
	noRev := Case{ID: "n", Labels: map[string]Label{intake.FieldNumber: auth("E-01")}}
	if _, ok := supersessionTruth(noRev, []filedPrior{prior("p1", "E-01", "C1")}); ok {
		t.Fatal("an unlabeled revision leaves supersession unknown")
	}
}

// optionValue is an option's literal: the description string, or its value
// field when the description is structured.
func optionValue(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Value string `json:"value"`
	}
	_ = json.Unmarshal(raw, &obj)
	return obj.Value
}

func TestOrdinalDateComparisonPreservesCalendarDay(t *testing.T) {
	if !SameValue("date", "15 August 2014", "15th August, 2014") {
		t.Fatal("same ordinal date rejected")
	}
	if SameValue("date", "16 August 2014", "15th August, 2014") {
		t.Fatal("different day accepted")
	}
}
