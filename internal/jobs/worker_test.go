package jobs_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/store"
)

func TestBackgroundFanOutIsOneCallPerState(t *testing.T) {
	cat := loadKnowledge(t)
	one := jobs.BackgroundCalls(cat, []jobs.Passage{{
		Ordinal: 1, Text: "Sprinkler pump and tank.", Section: "Fire",
		Kind: "specification", Discipline: "fire", Title: "Fire services",
	}})
	if len(one) != 1 {
		t.Fatalf("unlabelled passages make %d calls", len(one))
	}
	labelled := one[0]
	if labelled.Priority != jev.PriorityBackground {
		t.Fatalf("priority %s", labelled.Priority)
	}
	var nouls, choices int
	for id, q := range labelled.Questions {
		switch q.Type {
		case jev.TypeNoul:
			nouls++
			if !strings.HasPrefix(id, "system.") {
				t.Fatalf("noul id %s", id)
			}
		case jev.TypeChoice:
			choices++
			// Profile questions read the same passage and join this call.
			if !strings.HasPrefix(id, "source.") && !strings.HasPrefix(id, "hdr.") &&
				!strings.HasPrefix(id, "det.") && !strings.HasPrefix(id, "fact.") {
				t.Fatalf("choice id %s", id)
			}
		default:
			t.Fatalf("type %s", q.Type)
		}
	}
	if nouls != len(cat.TopSystems()) || choices == 0 {
		t.Fatalf("nouls %d choices %d tops %d", nouls, choices, len(cat.TopSystems()))
	}
	if _, ok := labelled.Questions["kind"]; ok {
		t.Fatal("label state includes a filing question")
	}

	both := jobs.BackgroundCalls(cat, []jobs.Passage{{
		Ordinal: 1, Text: "Sprinkler pump and tank.",
		Labels: []string{"fire-active", "fire-active.sprinklers"},
	}})
	if len(both) != 2 {
		t.Fatalf("calls %d", len(both))
	}
	if both[1].Priority != jev.PriorityBackground {
		t.Fatalf("evidence priority %s", both[1].Priority)
	}
	if len(both[1].Questions) < 2 {
		t.Fatalf("evidence fan-out %d", len(both[1].Questions))
	}
	two := jobs.BackgroundCalls(cat, []jobs.Passage{{Ordinal: 1, Text: "a"}, {Ordinal: 2, Text: "b"}})
	if len(two) != 2 {
		t.Fatalf("two passages produced %d calls", len(two))
	}
	if len(Accept(cat)) != 0 {
		t.Fatal("missing threshold accepted a label")
	}
}

func Accept(cat *knowledge.Catalog) []string {
	return jobs.AcceptLabels(cat, jev.Result{Answers: map[string]jev.Answer{
		"system.fire-active": {Type: jev.TypeNoul, Noul: 0.99},
	}}, 0)
}

func TestStagesLeaseAndPriority(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	cat := loadKnowledge(t)
	asker := &fakeAsk{}
	worker := &jobs.Worker{
		Store: st, Ask: asker, Catalog: cat, MinNoul: 0.9, Backoff: 0,
		Text: func(context.Context, string, string) (string, error) {
			return "Sprinkler tank on the roof.", nil
		},
	}
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	if err := worker.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	passages, err := st.DocumentPassages(ctx, org, doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(passages) != 1 {
		t.Fatalf("passages %+v", passages)
	}
	hits, err := st.SearchPassages(ctx, org, project, "sprinkler", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].DocumentID != doc {
		t.Fatalf("hits %+v", hits)
	}
	other, otherProject := newID(t), newID(t)
	seedOrg(t, st, other, otherProject)
	otherDoc := seedDoc(t, st, other, otherProject, store.StatusFiled)
	if err := st.ReplacePassages(ctx, other, otherDoc, []string{"Sprinkler in another org."}); err != nil {
		t.Fatal(err)
	}
	hits, err = st.SearchPassages(ctx, org, project, "sprinkler", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("cross-org hits %+v", hits)
	}

	// Splitting text queues no reading; the user's profile update does.
	if err := worker.Once(ctx, org); !errors.Is(err, store.ErrIdle) {
		t.Fatalf("reading ran unasked: %v", err)
	}
	if _, err := st.RequestProfileRead(ctx, org, project, nil); err != nil {
		t.Fatal(err)
	}
	if err := worker.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	labels, err := st.PassageSystems(ctx, org, passages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(labels, "fire-active") || !contains(labels, "fire-active.sprinklers") {
		t.Fatalf("labels %v", labels)
	}
	if err := worker.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	if len(asker.calls) != 2 {
		t.Fatalf("calls %d", len(asker.calls))
	}
	for _, call := range asker.calls {
		if call.Priority != jev.PriorityBackground {
			t.Fatalf("foreground call %+v", call.Priority)
		}
	}
	states, err := st.PassageEvidence(ctx, org, passages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) == 0 {
		t.Fatal("no evidence")
	}
	for _, state := range states {
		if state != knowledge.StateAddressed && state != knowledge.StateUnknown {
			t.Fatalf("state %s", state)
		}
	}
}

func TestDuplicateDeliveryAndExpiredLease(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	worker := &jobs.Worker{
		Store: st,
		Text: func(context.Context, string, string) (string, error) {
			return "First passage.\n\nSecond passage.", nil
		},
	}
	first, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Perform(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := worker.Perform(ctx, first); err != nil {
		t.Fatal(err)
	}
	passages, err := st.DocumentPassages(ctx, org, doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(passages) != 2 {
		t.Fatalf("passages %d", len(passages))
	}
	if err := st.ExpireLease(ctx, org, first.ID); err != nil {
		t.Fatal(err)
	}
	second, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Token == first.Token || second.Attempts != 2 {
		t.Fatalf("first %+v second %+v", first, second)
	}
	if err := st.CompleteJob(ctx, org, first.ID, first.Token, store.EventWrite{}); !errors.Is(err, store.ErrLeaseLost) {
		t.Fatalf("old token %v", err)
	}
	if err := st.CompleteJob(ctx, org, second.ID, second.Token, store.EventWrite{Kind: "job", DocumentID: doc, Payload: `{"kind":"full_text","status":"done"}`}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText}); !errors.Is(err, store.ErrIdle) {
		t.Fatalf("after complete %v", err)
	}
	record, err := st.GetJob(ctx, org, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != store.JobStatusDone {
		t.Fatalf("status %s", record.Status)
	}
}

func TestAttemptsFailAndInteractivePriority(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)

	var exhausted string
	for i := 0; i < 8; i++ {
		doc := seedDoc(t, st, org, project, store.StatusFiled)
		if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			job, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
			if err != nil {
				t.Fatal(err)
			}
			exhausted = job.ID
			if err := st.ExpireLease(ctx, org, job.ID); err != nil {
				t.Fatal(err)
			}
			for attempt := 1; attempt < 5; attempt++ {
				again, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
				if err != nil {
					t.Fatal(err)
				}
				if again.ID != exhausted {
					t.Fatalf("claim %d got %s", attempt, again.ID)
				}
				if err := st.ExpireLease(ctx, org, again.ID); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if _, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText}); err != nil {
		t.Fatal(err)
	}
	record, err := st.GetJob(ctx, org, exhausted)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != store.JobStatusFailed {
		t.Fatalf("exhausted %+v", record)
	}

	retryDoc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueJob(ctx, org, newID(t), retryDoc, store.JobKindJevRetry); err != nil {
		t.Fatal(err)
	}
	next, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindJevRetry, store.JobKindFullText})
	if err != nil {
		t.Fatal(err)
	}
	if next.Kind != store.JobKindJevRetry || next.DocumentID != retryDoc {
		t.Fatalf("priority %+v", next)
	}
}

func TestSkipLockedClaimsDifferentJobs(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	for i := 0; i < 2; i++ {
		doc := seedDoc(t, st, org, project, store.StatusFiled)
		if err := st.EnqueueJob(ctx, org, newID(t), doc, store.JobKindFullText); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	ids := make([]string, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			job, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
			errs[i] = err
			ids[i] = job.ID
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if ids[0] == "" || ids[0] == ids[1] {
		t.Fatalf("ids %v", ids)
	}
}

type fakeAsk struct {
	mu    sync.Mutex
	calls []jev.Call
}

func (f *fakeAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	answers := map[string]jev.Answer{}
	if _, ok := call.Questions["system.fire-active"]; ok {
		answers["system.fire-active"] = jev.Answer{Type: jev.TypeNoul, Noul: 0.99}
		answers["system.fire-active.sprinklers"] = jev.Answer{Type: jev.TypeNoul, Noul: 0.99}
	}
	for id, q := range call.Questions {
		if q.Type == jev.TypeNoul && !strings.HasPrefix(id, "system.") {
			answers[id] = jev.Answer{Type: jev.TypeNoul, Noul: 0.99}
		}
	}
	return jev.Result{Answers: answers}, nil
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func loadKnowledge(t *testing.T) *knowledge.Catalog {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	cat, err := knowledge.Load(filepath.Join(filepath.Dir(file), "..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	name, err := databaseName(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sitewise_test" {
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

func seedOrg(t *testing.T, st *store.Store, org, project string) {
	t.Helper()
	ctx := context.Background()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), org) })
	if err := st.CreateOrg(ctx, org, "org"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, org, project, "project"); err != nil {
		t.Fatal(err)
	}
}

func seedDoc(t *testing.T, st *store.Store, org, project, status string) string {
	t.Helper()
	ctx := context.Background()
	fileID, docID := newID(t), newID(t)
	sum := sha256.Sum256([]byte(docID))
	if err := st.CreateFile(ctx, org, store.File{
		ID: fileID, ProjectID: project, SHA256: sum[:], ByteSize: 4, MediaType: "text/plain",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDocument(ctx, org, store.Document{
		ID: docID, ProjectID: project, FileID: fileID, Filename: "a.txt", Status: status,
	}); err != nil {
		t.Fatal(err)
	}
	return docID
}

func newID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func databaseName(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", errors.New("database name missing")
	}
	return name, nil
}
