package jobs_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

type blockedProfile struct{ started chan struct{} }

func (b *blockedProfile) Ask(ctx context.Context, _ jev.Call) (jev.Result, error) {
	select {
	case b.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return jev.Result{}, ctx.Err()
}

func TestExtractionDoesNotWaitForProfile(t *testing.T) {
	st := openStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	old := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.ReplacePassages(ctx, org, old, []string{"Sprinklers included."}); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueStage(ctx, org, old, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	ask := &blockedProfile{started: make(chan struct{}, 1)}
	w := &jobs.Worker{Store: st, Ask: ask, Catalog: loadKnowledge(t), Text: func(context.Context, string, string) (string, error) {
		return "First paragraph.\n\nSecond paragraph.", nil
	}}
	done := make(chan struct{})
	go func() { defer close(done); jobs.RunBackground(ctx, w, 10*time.Millisecond, func(string, ...any) {}) }()
	defer func() { cancel(); <-done }()
	select {
	case <-ask.started:
	case <-time.After(3 * time.Second):
		t.Fatal("profile did not start")
	}
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		passages, err := st.DocumentPassages(ctx, org, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(passages) == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("upload extraction blocked behind profile reading")
}

func TestBackgroundDiscoversInterruptedExtraction(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	j, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindFullText})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ExpireLease(ctx, org, j.ID); err != nil {
		t.Fatal(err)
	}
	orgs, err := st.OrgsWithBackgroundJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(orgs, org) {
		t.Fatal("interrupted extraction invisible after restart")
	}
}

func TestExtractionResumesWithDevelopmentBacklogDisabled(t *testing.T) {
	st := openStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	ask := &blockedProfile{started: make(chan struct{}, 1)}
	w := &jobs.Worker{Store: st, Ask: ask, Since: time.Now().Add(time.Hour), Text: func(context.Context, string, string) (string, error) {
		return "Prepared text.", nil
	}}
	done := make(chan struct{})
	go func() { defer close(done); jobs.RunBackground(ctx, w, 10*time.Millisecond, func(string, ...any) {}) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		passages, err := st.DocumentPassages(ctx, org, doc)
		if err != nil {
			t.Fatal(err)
		}
		if len(passages) == 1 {
			// The upload must stop at chunks: no label job may have been queued.
			if _, err := st.ClaimJob(ctx, org, time.Minute, []string{store.JobKindLabel, store.JobKindEvidence}); err != store.ErrIdle {
				t.Fatalf("upload queued profile reading: %v", err)
			}
			select {
			case <-ask.started:
				t.Fatal("extraction called Jev")
			default:
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("development restart skipped old extraction")
}
