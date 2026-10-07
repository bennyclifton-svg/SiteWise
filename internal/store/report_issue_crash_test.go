package store_test

import (
	"context"
	"fmt"
	"sitewise/internal/files"
	"testing"
	"time"
)

func TestReportIssueProcessCrashRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	s, b, _ := workStore(t)
	pool := rawPool(t)
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "restart-test", true)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture is intentionally brief: the crash test must reach commit,
	// independently of the expanding draft knowledge catalogue and page limit.
	_, err = pool.Exec(ctx, `UPDATE report_versions SET sections='[{"id":"brief","title":"Brief","blocks":[{"id":"project","text":"Crash recovery fixture","label":"U","basis":{}}]}]'::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	blobRoot := t.TempDir()
	extra := []string{"SITEWISE_REPORT_CRASH_MODE=issue", "SITEWISE_REPORT_CRASH_BLOBS=" + blobRoot, fmt.Sprintf("SITEWISE_REPORT_CRASH_VERSION=%d", d.Version)}
	snapshot := func() string {
		t.Helper()
		var raw string
		err := pool.QueryRow(ctx, `SELECT jsonb_build_object('reports',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM reports r WHERE org_id=$1::uuid),'versions',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM report_versions v WHERE org_id=$1::uuid),'references',(SELECT jsonb_agg(to_jsonb(x) ORDER BY report_version_id,citation_id) FROM report_references x WHERE org_id=$1::uuid),'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM events e WHERE org_id=$1::uuid),'files',(SELECT jsonb_agg(to_jsonb(f) ORDER BY id) FROM files f WHERE org_id=$1::uuid))::text`, orgA).Scan(&raw)
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
	// The issue transaction updates this row only after rendering, blob writing,
	// reference replacement and the draft-to-issued snapshot transition.
	if _, err = barrier.Exec(ctx, `SELECT id FROM reports WHERE org_id=$1::uuid AND id=$2::uuid FOR UPDATE`, orgA, r.ID); err != nil {
		t.Fatal(err)
	}
	app := fmt.Sprintf("issue-crash-%d", time.Now().UnixNano())
	child := startReportCrashProcess(t, r.ID, app, extra...)
	for {
		var waiting bool
		err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND query LIKE 'UPDATE reports SET current_draft_version_id=NULL%')`, app).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case <-child.done:
			t.Fatalf("child exited early: %v %s", child.err, &child.output)
		case <-ctx.Done():
			t.Fatal("issue did not reach crash point", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	if snapshot() != before {
		t.Fatal("partial issue visible before commit")
	}
	if err = child.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-child.done
	if err = barrier.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for {
		var active bool
		err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1)`, app).Scan(&active)
		if err != nil {
			t.Fatal(err)
		}
		if !active {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("crashed issue retained transaction")
		case <-time.After(20 * time.Millisecond):
		}
	}
	if snapshot() != before {
		t.Fatal("crash changed committed issue state")
	}
	restarted := startReportCrashProcess(t, r.ID, app+"-restart", extra...)
	select {
	case <-restarted.done:
		if restarted.err != nil {
			t.Fatalf("retry failed: %v %s", restarted.err, &restarted.output)
		}
	case <-ctx.Done():
		t.Fatal("issue retry timeout")
	}
	issued, err := s.ReadIssuedReport(ctx, orgA, r.ID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(blobRoot, 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	f, err := blobs.Open(issued.ExportSHA256)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND status='issued'`, orgA, r.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("retry duplicated or lost issue", count, err)
	}
}
