package jobs_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

// profileAsk labels fire-active.sprinklers and answers the profile questions
// it is offered, as a recorded Jev response would.
type profileAsk struct {
	mu    sync.Mutex
	calls []jev.Call
}

func (f *profileAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	c := 0.9
	answers := map[string]jev.Answer{}
	for id := range call.Questions {
		switch {
		case id == "system.fire-active":
			answers[id] = jev.Answer{Type: jev.TypeNoul, Noul: 0.99}
		case id == "system.fire-active.sprinklers":
			answers[id] = jev.Answer{Type: jev.TypeNoul, Noul: 0.99}
		case id == "det.bal":
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "BAL-40", Confidence: &c}
		case id == "det.bal.assertion":
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "stated", Confidence: &c}
		case strings.HasSuffix(id, ".presence"):
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "included", Confidence: &c}
		case strings.HasSuffix(id, ".provider"):
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "not_stated", Confidence: &c}
		}
	}
	return jev.Result{Answers: answers}, nil
}

func TestStagesStoreProfileFactsAndRebuild(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	asker := &profileAsk{}
	worker := &jobs.Worker{
		Store: st, Ask: asker, Catalog: loadKnowledge(t), MinNoul: 0.9,
		Text: func(context.Context, string, string) (string, error) {
			return "FIRE SERVICES\n\nSprinkler tank on the roof. Windows with BAL 40 compliance.", nil
		},
		Profile: profile.Thresholds{Version: "profile-1",
			Amber: map[string]float64{"presence": 0.6, "provider": 0.6, "assertion": 0.6, "header": 0.6, "determinant": 0.6},
			Green: map[string]*float64{}},
	}
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ { // full text, label, evidence
		if i == 1 { // the user asks for the profile update
			if _, err := st.RequestProfileRead(ctx, org, project, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := worker.Once(ctx, org); err != nil {
			t.Fatalf("stage %d: %v", i, err)
		}
	}
	if len(asker.calls) < 2 {
		t.Fatalf("calls %d", len(asker.calls))
	}
	view, err := st.ReadProfile(ctx, org, project, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]profile.Row{}
	for _, r := range view.Rows {
		got[r.Key] = r
	}
	if r := got["det.bal"]; r.Value != "BAL-40" || r.Band != "amber" || r.Assertion != "stated" {
		t.Fatalf("det.bal %+v (rows %v)", r, view.Rows)
	}
	if r := got["sys.fire-active.sprinklers.presence"]; r.Value != "included" || r.Band != "amber" || r.Note == "" {
		t.Fatalf("sprinklers %+v", r)
	}
	if view.BuiltAt == nil || len(view.Parts) != 1 {
		t.Fatalf("build %+v", view)
	}
}

func TestBackgroundOrgsListsQueuedWork(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	orgs, err := st.OrgsWithBackgroundJobs(ctx)
	if err != nil || !contains(orgs, org) {
		t.Fatalf("orgs %v %v", orgs, err)
	}
}

// Every evidence question must encode as the API expects: a JSON object of
// criteria. A boolean-keyed YAML map would make the client reject the call.
func TestEvidenceQuestionsEncode(t *testing.T) {
	cat := loadKnowledge(t)
	var labels []string
	for _, s := range cat.Leaves() {
		labels = append(labels, s.ID)
	}
	for _, s := range cat.TopSystems() {
		labels = append(labels, s.ID)
	}
	call, ok := jobs.EvidenceCall(cat, jobs.Passage{Text: "x", Labels: labels})
	if !ok {
		t.Fatal("no evidence call")
	}
	for id, q := range call.Questions {
		raw, err := json.Marshal(q.Criteria)
		if err != nil || len(raw) == 0 || raw[0] != '{' {
			t.Fatalf("%s criteria do not encode: %v %s", id, err, raw)
		}
	}
}

func TestClaimSinceLeavesOlderBacklogQueued(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if _, err := st.ClaimJobSince(ctx, org, time.Minute, []string{store.JobKindFullText}, later); err != store.ErrIdle {
		t.Fatalf("backlog must stay queued, got %v", err)
	}
	if orgs, _ := st.OrgsWithBackgroundJobsSince(ctx, later); contains(orgs, org) {
		t.Fatal("org listed for backlog only")
	}
	if _, err := st.ClaimJobSince(ctx, org, time.Minute, []string{store.JobKindFullText}, time.Time{}); err != nil {
		t.Fatalf("zero since claims everything: %v", err)
	}
}

// detailAsk picks the first candidate of every detail question and answers
// yes to every flag it is asked, like a confident recorded Jev.
type detailAsk struct{}

func (detailAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	c := 0.9
	answers := map[string]jev.Answer{}
	for id := range call.Questions {
		switch {
		case strings.HasPrefix(id, "proj.flag."):
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "yes", Confidence: &c}
		case strings.HasPrefix(id, "proj.") || strings.HasPrefix(id, "site."):
			answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "c1", Confidence: &c}
		}
	}
	return jev.Result{Answers: answers}, nil
}

// The Petersham cover page through the real stages: reading fills the client,
// address and LGA with their source, and the document is stamped as read at
// the current revision so the next Update does not read it again.
func TestStagesReadProjectDetailsFromADocument(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	worker := &jobs.Worker{
		Store: st, Ask: detailAsk{}, Catalog: loadKnowledge(t), MinNoul: 0.9,
		Text: func(context.Context, string, string) (string, error) {
			return "PRINCIPAL REQUIREMENTS JW building: 568-572 Parramatta Road Petersham Principal: Joins Win Pty Ltd ABN 20 098 348 031\n\n" +
				"The site is located within the Inner West local government area. Excavate the basement.", nil
		},
		Profile: profile.Thresholds{Version: "profile-1",
			Amber: map[string]float64{"detail": 0.6, "header": 0.6, "determinant": 0.6}, Green: map[string]*float64{}},
	}
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if i == 1 {
			if _, err := st.RequestProfileRead(ctx, org, project, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := worker.Once(ctx, org); err != nil {
			t.Fatalf("stage %d: %v", i, err)
		}
	}
	view, err := st.ReadProfile(ctx, org, project, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]profile.Row{}
	for _, r := range view.Rows {
		got[r.Key] = r
	}
	for key, want := range map[string]string{
		"proj.client": "Joins Win Pty Ltd", "site.address": "568-572 Parramatta Road Petersham",
		"proj.lga": "Inner West", "proj.flag.basement": "yes",
	} {
		r := got[key]
		if r.Value != want || r.Band != "amber" || len(r.Sources) != 1 || r.Sources[0].DocumentID != doc {
			t.Errorf("%s = %+v, want %q", key, r, want)
		}
	}
	if n, err := st.RequestProfileRead(ctx, org, project, nil); err != nil || n != 0 {
		t.Fatalf("a document read at the current revision was queued again: n=%d err=%v", n, err)
	}
}
