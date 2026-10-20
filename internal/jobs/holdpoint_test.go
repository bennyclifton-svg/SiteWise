package jobs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/programread"
	"sitewise/internal/store"
)

const (
	holdIface    = "if.envelope-frame-before-roof"
	holdEvidence = "Frame inspection record accepting the support construction."
	releaseID    = "hold:" + holdIface
)

// releaseKnowledge is the repository knowledge with one release question
// authored on a real hold point, in a temporary copy: release questions
// are owner content (WP-P15) and are not written into knowledge/.
func releaseKnowledge(t *testing.T) *knowledge.Catalog {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(file), "..", "..", "knowledge")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dst, "clusters", "envelope", "interfaces.yaml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	anchor := "    evidence: " + holdEvidence + "\n"
	if !bytes.Contains(body, []byte(anchor)) {
		t.Fatal("fixture hold point moved")
	}
	release := anchor + `    release:
      type: noul
      instructions: >-
        Using ` + "`text`" + `, does this passage state that the inspection described in
        ` + "`hold_point`" + ` was carried out and the work was released to proceed?
      criteria:
        true: It states the inspection was completed and the work may proceed.
        false: It only requires, schedules or requests the inspection.
      runs_on: [structure]
      status: draft
`
	if err := os.WriteFile(path, bytes.Replace(body, []byte(anchor), []byte(release), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := knowledge.Load(dst)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// replayAsk answers every call with one recorded System One response
// through the real client, so it is validated as a live one would be
// (plan section 2, Jev rule 15), and keeps the calls it was asked.
type replayAsk struct {
	mu     sync.Mutex
	client *jev.Client
	calls  []jev.Call
}

type replayTransport struct{ body []byte }

func (r replayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_, _ = io.Copy(io.Discard, req.Body)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(r.body)), Request: req}, nil
}

func recordedAsk(t *testing.T, name string) *replayAsk {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "jev", name))
	if err != nil {
		t.Fatal(err)
	}
	c, err := jev.New(jev.Options{BaseURL: "http://replay.invalid", APIKey: "test-key", Transport: replayTransport{body}, BreakerThreshold: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return &replayAsk{client: c}
}

func (r *replayAsk) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()
	return r.client.Ask(ctx, call)
}

type holdCase struct {
	kind    string
	labels  []string
	program bool
	answer  string // recorded response file
	th      programread.Thresholds
}

type holdRun struct {
	st                    *store.Store
	org, project, doc, id string // id is the one passage
	hold                  string // the hold point row, when a program exists
	calls                 []jev.Call
}

// runEvidence files one one-passage document of the given kind, labels its
// passage, optionally gives the project a program with the hold point,
// and runs the evidence stage once.
func runEvidence(t *testing.T, st *store.Store, cat *knowledge.Catalog, c holdCase) holdRun {
	t.Helper()
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	for field, value := range map[string]string{"kind": c.kind, "title": "Frame inspection certificate"} {
		if err := st.CreateDecision(ctx, org, store.Decision{ID: newID(t), DocumentID: doc, Field: field, Value: value, Band: "green", DecidedBy: "rule"}); err != nil {
			t.Fatal(err)
		}
	}
	text := "Frame inspected on site. The supporting frame is accepted and roof covering may proceed."
	if err := st.ReplaceSource(ctx, org, doc, store.DocumentSource{Source: []store.SourcePage{{Text: text}}, Units: []store.SourceUnit{{Body: text, Page: 2}}}); err != nil {
		t.Fatal(err)
	}
	passages, err := st.DocumentPassages(ctx, org, doc)
	if err != nil || len(passages) != 1 {
		t.Fatalf("passages %v %v", passages, err)
	}
	if err := st.SetPassageSystems(ctx, org, passages[0].ID, c.labels); err != nil {
		t.Fatal(err)
	}
	run := holdRun{st: st, org: org, project: project, doc: doc, id: passages[0].ID}
	if c.program {
		in := store.ProgramCreate{Surface: "program", LibraryVersion: "test", Rows: []store.ProgramRowInput{
			{Key: "construction", Kind: "stage", Label: "Construction", CatalogueStageID: "construction", ReviewStatus: "proposed", Origin: "library"},
			{Key: "hold", ParentKey: "construction", Kind: "hold_point", Label: "Hold point: frame inspection", InterfaceID: holdIface, SourceKey: "hold:" + holdIface,
				ReviewStatus: "proposed", Origin: "library", DurationOrigin: "calculation", DurationSource: json.RawMessage(`{"hold_point":"` + holdIface + `"}`)},
		}}
		if _, err := st.CreateProgram(ctx, org, project, in); err != nil {
			t.Fatal(err)
		}
		p, err := st.LoadProgram(ctx, org, project)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range p.Activities {
			if a.Kind == "hold_point" {
				run.hold = a.ID
			}
		}
	}
	ask := recordedAsk(t, c.answer)
	w := &jobs.Worker{Store: st, Ask: ask, Catalog: cat, MinNoul: 0.5, ProgramThresholds: c.th}
	if err := w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindEvidence}); err != nil {
		t.Fatal(err)
	}
	run.calls = ask.calls
	return run
}

func holdThresholds(t *testing.T) programread.Thresholds {
	t.Helper()
	th, err := programread.LoadThresholds(filepath.Join("testdata", "holdpoint"))
	if err != nil {
		t.Fatal(err)
	}
	return th
}

func callJSON(t *testing.T, c jev.Call) string {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// withoutRelease is the joined call with the release question, its state
// field and its version suffix taken out: what the passage asked before.
func withoutRelease(c jev.Call) jev.Call {
	state := map[string]any{}
	for k, v := range c.State.(map[string]any) {
		if k != programread.HoldPointStateKey {
			state[k] = v
		}
	}
	questions := map[string]jev.Question{}
	for k, v := range c.Questions {
		if k != releaseID {
			questions[k] = v
		}
	}
	c.State, c.Questions = state, questions
	c.QuestionVersion = strings.TrimSuffix(c.QuestionVersion, "+"+programread.HoldPointReleaseVersion)
	return c
}

func (r holdRun) readings(t *testing.T) []store.ProgramHoldPointView {
	t.Helper()
	p, err := r.st.LoadProgram(context.Background(), r.org, r.project)
	if err != nil {
		t.Fatal(err)
	}
	v, err := r.st.ProgramView(context.Background(), r.org, r.project, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range p.Activities {
		if a.Kind == "hold_point" && (strings.Contains(string(a.DurationSource), "released") || a.ReviewStatus != "proposed") {
			t.Fatalf("hold point changed without a click: %s %s", a.DurationSource, a.ReviewStatus)
		}
	}
	return v.HoldPoints
}

func TestHoldPointReleaseJoinsThePassageFanOut(t *testing.T) {
	st := openStore(t)
	cat := releaseKnowledge(t)
	th := holdThresholds(t)
	ctx := context.Background()

	// Before: no program. The call carries no release question and the
	// passage's own version.
	before := runEvidence(t, st, cat, holdCase{kind: "certificate", labels: []string{"structure"}, answer: "hold_point_release_green.json", th: th})
	if len(before.calls) != 1 {
		t.Fatalf("calls %d", len(before.calls))
	}
	base := before.calls[0]
	if _, asked := base.Questions[releaseID]; asked || strings.Contains(base.QuestionVersion, programread.HoldPointReleaseVersion) {
		t.Fatalf("release asked without a program: %s", base.QuestionVersion)
	}
	if _, ok := base.State.(map[string]any)[programread.HoldPointStateKey]; ok {
		t.Fatal("hold_point state without a program")
	}

	// With the hold point in the program: the same single fan-out plus one
	// question, one state field and the release version.
	joined := runEvidence(t, st, cat, holdCase{kind: "certificate", labels: []string{"structure"}, program: true, answer: "hold_point_release_green.json", th: th})
	if len(joined.calls) != 1 {
		t.Fatalf("the release made %d calls, want the passage's one", len(joined.calls))
	}
	call := joined.calls[0]
	if call.Questions[releaseID].Type != jev.TypeNoul || call.State.(map[string]any)[programread.HoldPointStateKey] != holdEvidence ||
		call.QuestionVersion != base.QuestionVersion+"+"+programread.HoldPointReleaseVersion || call.Priority != jev.PriorityBackground {
		t.Fatalf("joined call %s", callJSON(t, call))
	}
	if callJSON(t, withoutRelease(call)) != callJSON(t, base) {
		t.Fatalf("the passage's own questions changed:\n%s\n%s", callJSON(t, withoutRelease(call)), callJSON(t, base))
	}

	// Existing passage thresholds apply unchanged: the same evidence states,
	// and the release answer is not one of them.
	was, err := st.PassageEvidence(ctx, before.org, before.id)
	if err != nil {
		t.Fatal(err)
	}
	now, err := st.PassageEvidence(ctx, joined.org, joined.id)
	if err != nil {
		t.Fatal(err)
	}
	if len(was) == 0 || len(was) != len(now) || was["if:"+holdIface+"#0"] != knowledge.StateAddressed {
		t.Fatalf("evidence %v vs %v", was, now)
	}
	for k, v := range was {
		if now[k] != v {
			t.Fatalf("evidence %s: %s, was %s", k, now[k], v)
		}
	}
	if _, ok := now[releaseID]; ok {
		t.Fatal("release answer stored as passage evidence")
	}

	// A green answer is a proposal on the hold point, nothing more.
	hv := joined.readings(t)
	if len(hv) != 1 || hv[0].ActivityID != joined.hold || len(hv[0].Evidence) != 1 || hv[0].Released != nil {
		t.Fatalf("readings %+v", hv)
	}
	ev := hv[0].Evidence[0]
	if ev.Band != programread.BandGreen || ev.DocumentID != joined.doc || ev.PassageID != joined.id || ev.Page != 2 ||
		ev.Text != "Evidence found: Frame inspection certificate, page 2" || ev.DateLabel != "document dated" {
		t.Fatalf("evidence %+v", ev)
	}

	// Rerunning reads the cached answer and replaces, never duplicates.
	w := &jobs.Worker{Store: st, Ask: failAsk{t}, Catalog: cat, MinNoul: 0.5, ProgramThresholds: th}
	if err := w.Perform(ctx, store.ClaimedJob{OrgID: joined.org, DocumentID: joined.doc, Kind: store.JobKindEvidence}); err != nil {
		t.Fatal(err)
	}
	if hv := joined.readings(t); len(hv) != 1 || len(hv[0].Evidence) != 1 {
		t.Fatalf("rerun readings %+v", hv)
	}
}

type failAsk struct{ t *testing.T }

func (f failAsk) Ask(context.Context, jev.Call) (jev.Result, error) {
	f.t.Error("a cached passage asked Jev again")
	return jev.Result{}, errors.New("asked")
}

func TestHoldPointReleaseGating(t *testing.T) {
	st := openStore(t)
	cat := releaseKnowledge(t)
	th := holdThresholds(t)
	cases := map[string]holdCase{
		"drawing":         {kind: "drawing", labels: []string{"structure"}, program: true},
		"specification":   {kind: "specification", labels: []string{"structure"}, program: true},
		"outside runs_on": {kind: "report", labels: []string{"envelope"}, program: true},
		"no program":      {kind: "correspondence", labels: []string{"structure"}},
	}
	for name, c := range cases {
		c.answer, c.th = "hold_point_release_green.json", th
		r := runEvidence(t, st, cat, c)
		for _, call := range r.calls {
			if _, asked := call.Questions[releaseID]; asked || strings.Contains(call.QuestionVersion, programread.HoldPointReleaseVersion) {
				t.Fatalf("%s: release asked", name)
			}
			if _, ok := call.State.(map[string]any)[programread.HoldPointStateKey]; ok {
				t.Fatalf("%s: hold_point state sent", name)
			}
		}
		if c.program {
			if hv := r.readings(t); len(hv) != 0 {
				t.Fatalf("%s: readings %+v", name, hv)
			}
		}
	}
	for _, kind := range []string{"report", "correspondence"} {
		r := runEvidence(t, st, cat, holdCase{kind: kind, labels: []string{"structure"}, program: true, answer: "hold_point_release_green.json", th: th})
		if len(r.calls) != 1 || r.calls[0].Questions[releaseID].Type != jev.TypeNoul {
			t.Fatalf("%s: release not asked", kind)
		}
	}
}

// Noul banding on recorded responses: only an answer above a calibrated
// cut-off proposes; no cut-off proposes nothing.
func TestHoldPointReleaseBanding(t *testing.T) {
	st := openStore(t)
	cat := releaseKnowledge(t)
	th := holdThresholds(t)
	for _, c := range []struct {
		answer string
		th     programread.Thresholds
		band   string
	}{
		{"hold_point_release_green.json", th, programread.BandGreen},
		{"hold_point_release_amber.json", th, programread.BandAmber},
		{"hold_point_release_low.json", th, ""},
		{"hold_point_release_green.json", programread.Thresholds{QuestionVersion: programread.QuestionVersion}, ""},
	} {
		r := runEvidence(t, st, cat, holdCase{kind: "certificate", labels: []string{"structure"}, program: true, answer: c.answer, th: c.th})
		hv := r.readings(t)
		if c.band == "" {
			if len(hv) != 0 {
				t.Fatalf("%s: proposed %+v", c.answer, hv)
			}
			continue
		}
		if len(hv) != 1 || len(hv[0].Evidence) != 1 || hv[0].Evidence[0].Band != c.band {
			t.Fatalf("%s: %+v", c.answer, hv)
		}
	}
}

// With no hold points to read, WithRelease returns the call it was given.
func TestWithReleaseLeavesCallUnchanged(t *testing.T) {
	cat := releaseKnowledge(t)
	p := jobs.Passage{Text: "Frame accepted.", Kind: "certificate", Labels: []string{"structure"}}
	call, ok := jobs.EvidenceCall(cat, p)
	if !ok {
		t.Fatal("no evidence call")
	}
	target := jobs.ReleaseTarget{ProgramID: "p", ActivityID: "a", InterfaceID: holdIface, Evidence: holdEvidence}
	for name, c := range map[string]struct {
		p       jobs.Passage
		targets []jobs.ReleaseTarget
	}{
		"no targets": {p, nil},
		"other kind": {jobs.Passage{Text: p.Text, Kind: "drawing", Labels: p.Labels}, []jobs.ReleaseTarget{target}},
		"no release": {p, []jobs.ReleaseTarget{{InterfaceID: "if.envelope-sarking-before-roof", Evidence: "x"}}},
	} {
		got, _, joined := jobs.WithRelease(cat, call, c.p, c.targets)
		if joined || callJSON(t, got) != callJSON(t, call) {
			t.Fatalf("%s: call changed", name)
		}
	}
	got, tg, joined := jobs.WithRelease(cat, call, p, []jobs.ReleaseTarget{target})
	if !joined || tg != target || len(got.Questions) != len(call.Questions)+1 || len(call.Questions) == len(got.Questions) {
		t.Fatal("release not joined")
	}
	if _, leaked := call.Questions[releaseID]; leaked {
		t.Fatal("joining changed the caller's call")
	}
}
