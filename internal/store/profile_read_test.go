package store_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/store"
)

func TestProfileActiveWorkExcludesQueuedAndExpiredLeases(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindEvidence); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='failed',attempts=max_attempts WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	v, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || v.PendingDocuments != 1 || v.FailedDocuments != 1 || v.ActiveDocuments != 0 {
		t.Fatalf("blocked work: %+v %v", v, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='leased',locked_until=now()+interval '1 minute' WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	v, err = st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || v.ActiveDocuments != 1 {
		t.Fatalf("active lease: %+v %v", v, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET locked_until=now()-interval '1 second' WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	v, err = st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || v.ActiveDocuments != 0 || v.PendingDocuments != 1 {
		t.Fatalf("expired lease: %+v %v", v, err)
	}
}

func TestProfileRequestDuringExtractionWaitsThenReads(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		n, err := st.RequestProfileRead(ctx, orgA, projectA, nil)
		if err != nil || n != int64(1-i) {
			t.Fatalf("click %d queued=%d err=%v", i, n, err)
		}
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindLabel}); err != store.ErrIdle {
		t.Fatalf("reading started before extraction: %v", err)
	}
	extraction, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindFullText})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteJob(ctx, orgA, extraction.ID, extraction.Token, store.EventWrite{Kind: "job", DocumentID: docA, Payload: `{}`}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindLabel}); err != nil {
		t.Fatalf("remembered click did not start reading after extraction: %v", err)
	}
}

func TestDocumentViewsExposeTextPreparation(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"queued", "leased", "done", "failed"} {
		if _, err := pool.Exec(ctx, `UPDATE jobs SET status=$1 WHERE org_id=$2 AND document_id=$3 AND kind='full_text'`, status, orgA, docA); err != nil {
			t.Fatal(err)
		}
		v, err := st.DocumentView(ctx, orgA, docA)
		if err != nil || v.TextStatus != status {
			t.Fatalf("single document: %q %v", v.TextStatus, err)
		}
		list, err := st.ProjectDocumentViews(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range list {
			if d.ID == docA {
				found = d.TextStatus == status
			}
		}
		if !found {
			t.Fatalf("register did not expose %s", status)
		}
	}
}

func TestProfileFailureIsVisibleAndRetryable(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='failed', attempts=max_attempts, last_error='jev response rejected: status 402' WHERE org_id=$1 AND document_id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	v, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || v.FailedDocuments != 1 || !v.PaymentRequired {
		t.Fatalf("missing payment failure: %+v %v", v, err)
	}
	if _, err := st.RequestProfileRead(ctx, orgA, projectA, nil); err != nil {
		t.Fatal(err)
	}
	v, err = st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || v.FailedDocuments != 0 || v.PaymentRequired || v.PendingDocuments != 1 {
		t.Fatalf("retry state: %+v %v", v, err)
	}
	if _, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindLabel}); err != nil {
		t.Fatal(err)
	}
}

func TestRequestProfileReadQueuesUnreadDocuments(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)

	view, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	if view.UnreadDocuments != 0 {
		t.Fatalf("unread before any text = %d", view.UnreadDocuments)
	}

	// Filing has split the text; nothing has asked Jev to read it yet.
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status = 'done' WHERE org_id = $1`, orgA); err != nil {
		t.Fatal(err)
	}
	if view, _ = st.ReadProfile(ctx, orgA, projectA, nil); view.UnreadDocuments != 1 {
		t.Fatalf("unread = %d, want 1", view.UnreadDocuments)
	}

	for i := 0; i < 2; i++ { // a second click queues nothing new
		n, err := st.RequestProfileRead(ctx, orgA, projectA, nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := int64(1 - i); n != want {
			t.Fatalf("click %d queued %d, want %d", i, n, want)
		}
	}
	var labels int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE org_id = $1 AND kind = 'label' AND status = 'queued'`, orgA).Scan(&labels); err != nil {
		t.Fatal(err)
	}
	if labels != 1 {
		t.Fatalf("label jobs = %d", labels)
	}
	if view, _ = st.ReadProfile(ctx, orgA, projectA, nil); view.UnreadDocuments != 0 || view.PendingDocuments != 1 {
		t.Fatalf("after click unread=%d pending=%d", view.UnreadDocuments, view.PendingDocuments)
	}

	// Another org's project is out of reach.
	if n, err := st.RequestProfileRead(ctx, orgB, projectA, nil); err == nil && n != 0 {
		t.Fatalf("cross-org click queued %d", n)
	}
}

func TestRequestProfileReadRestampsOldQueuedJobs(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)

	// A reading job queued before the server started is skipped by a worker
	// limited to new jobs; the click makes it current.
	if err := st.EnqueueStage(ctx, orgA, docA, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET created_at = now() - interval '1 hour' WHERE org_id = $1`, orgA); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RequestProfileRead(ctx, orgA, projectA, nil); err != nil {
		t.Fatal(err)
	}
	var fresh bool
	if err := pool.QueryRow(ctx, `SELECT created_at > now() - interval '1 minute' FROM jobs WHERE org_id = $1 AND kind = 'full_text'`, orgA).Scan(&fresh); err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("queued full_text job was not restamped")
	}
}
