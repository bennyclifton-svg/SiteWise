package jobs_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

func TestExtractionCrashChild(t *testing.T) {
	if os.Getenv("SITEWISE_JOB_CRASH_CHILD") != "1" {
		return
	}
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if name, err := databaseName(dsn); err != nil || name != "sitewise_test" {
		t.Fatalf("refusing non-test database %q: %v", name, err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("application_name", os.Getenv("SITEWISE_JOB_CRASH_APP"))
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	w := &jobs.Worker{Store: st, Lease: time.Second, Kinds: []string{store.JobKindFullText}, Text: func(context.Context, string, string) (string, error) {
		return "Recovered first paragraph.\n\nRecovered second paragraph.", nil
	}}
	org := os.Getenv("SITEWISE_JOB_CRASH_ORG")
	if err := w.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	if err := w.Once(ctx, org); !errors.Is(err, store.ErrIdle) {
		t.Fatalf("completed extraction was delivered again: %v", err)
	}
}

type workerCrashProcess struct {
	cmd    *exec.Cmd
	done   chan struct{}
	err    error
	output bytes.Buffer
}

func startExtractionCrashProcess(t *testing.T, org, app string) *workerCrashProcess {
	return startWorkerCrashProcess(t, org, app, "TestExtractionCrashChild")
}

func startWorkerCrashProcess(t *testing.T, org, app, entry string, extraEnv ...string) *workerCrashProcess {
	t.Helper()
	p := &workerCrashProcess{cmd: exec.Command(os.Args[0], "-test.run=^"+entry+"$", "-test.count=1"), done: make(chan struct{})}
	p.cmd.Env = append(os.Environ(), "SITEWISE_JOB_CRASH_CHILD=1", "SITEWISE_JOB_CRASH_ORG="+org, "SITEWISE_JOB_CRASH_APP="+app)
	p.cmd.Env = append(p.cmd.Env, extraEnv...)
	p.cmd.Stdout, p.cmd.Stderr = &p.output, &p.output
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		p.err = p.cmd.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		_ = p.cmd.Process.Kill()
		<-p.done
	})
	return p
}

func TestExtractionProcessCrashRecovery(t *testing.T) {
	st := openStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	doc := seedDoc(t, st, org, project, store.StatusFiled)
	if err := st.ReplaceSource(ctx, org, doc, store.DocumentSource{
		Source: []store.SourcePage{{Text: "Previously saved source", Location: "Document"}},
		Units:  []store.SourceUnit{{Body: "Previously saved source"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	snapshot := func() string {
		t.Helper()
		var raw string
		err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
 'source',(SELECT jsonb_agg(to_jsonb(s) ORDER BY document_id) FROM document_sources s WHERE org_id=$1::uuid),
 'passages',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM passages p WHERE org_id=$1::uuid),
 'units',(SELECT jsonb_agg(to_jsonb(s) ORDER BY passage_id) FROM passage_sources s WHERE org_id=$1::uuid),
 'facts',(SELECT jsonb_agg(to_jsonb(f) ORDER BY id) FROM profile_facts f WHERE org_id=$1::uuid),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_revisions r WHERE org_id=$1::uuid),
 'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM events e WHERE org_id=$1::uuid)
 )::text`, org).Scan(&raw)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	barrier, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer barrier.Rollback(context.Background())
	// The worker has replaced passages inside its transaction before reaching
	// this INSERT. Stop at a real database boundary without a production hook.
	if _, err := barrier.Exec(ctx, "LOCK TABLE document_sources IN SHARE MODE"); err != nil {
		t.Fatal(err)
	}
	app := fmt.Sprintf("extraction-crash-%d", time.Now().UnixNano())
	child := startExtractionCrashProcess(t, org, app)
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND query LIKE 'INSERT INTO document_sources%')`, app).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case <-child.done:
			t.Fatalf("worker exited before crash point: %v\n%s", child.err, &child.output)
		case <-ctx.Done():
			t.Fatal("worker did not reach source persistence", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	var jobID, token string
	if err := pool.QueryRow(ctx, `SELECT id::text,lease_token::text FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='full_text' AND status='leased' AND attempts=1`, org, doc).Scan(&jobID, &token); err != nil {
		t.Fatal("worker did not durably claim the job", err)
	}
	if snapshot() != before {
		t.Fatal("partial extraction became visible")
	}
	// This handle belongs to the exact child above; no other worker is killed.
	if err := child.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-child.done
	if child.err == nil {
		t.Fatal("crash child exited successfully")
	}
	if err := barrier.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for {
		var active bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1)`, app).Scan(&active); err != nil {
			t.Fatal(err)
		}
		if !active {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("crashed worker did not release database connections", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	if snapshot() != before {
		t.Fatal("crash changed committed source, revision or events")
	}
	// Let the real lease expire. Explicitly expiring it would not demonstrate
	// that a process restart discovers an abandoned job without intervention.
	for {
		orgs, err := st.OrgsWithJobKindsSince(ctx, []string{store.JobKindFullText}, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		if contains(orgs, org) {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("abandoned lease was not discovered", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	restarted := startExtractionCrashProcess(t, org, app+"-restart")
	select {
	case <-restarted.done:
		if restarted.err != nil {
			t.Fatalf("restarted worker failed: %v\n%s", restarted.err, &restarted.output)
		}
	case <-ctx.Done():
		t.Fatal("worker restart did not finish", ctx.Err())
	}
	job, err := st.GetJob(ctx, org, jobID)
	if err != nil || job.Status != store.JobStatusDone || job.Attempts != 2 || job.DocumentID != doc {
		t.Fatalf("job was not recovered in place: %+v %v", job, err)
	}
	passages, err := st.DocumentPassages(ctx, org, doc)
	if err != nil || len(passages) != 2 || passages[0].Body != "Recovered first paragraph." || passages[1].Body != "Recovered second paragraph." {
		t.Fatalf("missing or duplicate extracted passages: %+v %v", passages, err)
	}
	units, err := st.SourceUnits(ctx, org, doc)
	if err != nil || len(units) != 2 {
		t.Fatalf("missing or duplicate source units: %+v %v", units, err)
	}
	var sources, completions, readings int
	if err := pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM document_sources WHERE org_id=$1::uuid AND document_id=$2::uuid),
 (SELECT count(*) FROM events WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='job' AND payload::jsonb->>'status'='done'),
 (SELECT count(*) FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind IN ('label','evidence'))`, org, doc).Scan(&sources, &completions, &readings); err != nil {
		t.Fatal(err)
	}
	if sources != 1 || completions != 1 || readings != 0 {
		t.Fatalf("unexpected recovery effects: sources=%d completions=%d reading jobs=%d", sources, completions, readings)
	}
	completed := snapshot()
	if err := st.CompleteJob(ctx, org, jobID, token, store.EventWrite{Kind: "job", DocumentID: doc, Payload: `{"status":"done"}`}); !errors.Is(err, store.ErrLeaseLost) {
		t.Fatalf("dead worker token still completes the job: %v", err)
	}
	if snapshot() != completed {
		t.Fatal("stale worker completion changed committed data")
	}
}
