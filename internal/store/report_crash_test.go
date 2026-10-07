package store_test

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// This entry point runs in an owned child of this test executable. It opens
// existing fixtures only: running profileStore here would erase the parent's
// committed state and turn a restart test into a fresh installation test.
func TestReportCrashChild(t *testing.T) {
	if os.Getenv("SITEWISE_REPORT_CRASH_CHILD") != "1" {
		return
	}
	dsn, err := url.Parse(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	q := dsn.Query()
	q.Set("application_name", os.Getenv("SITEWISE_REPORT_CRASH_APP"))
	dsn.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := store.Open(ctx, dsn.String())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	s = s.WithProfile(store.ProfileBuild{Catalog: cat})
	if os.Getenv("SITEWISE_REPORT_CRASH_MODE") == "issue" {
		blobs, e := files.Open(os.Getenv("SITEWISE_REPORT_CRASH_BLOBS"), 20<<20)
		if e != nil {
			t.Fatal(e)
		}
		version, e := strconv.ParseInt(os.Getenv("SITEWISE_REPORT_CRASH_VERSION"), 10, 64)
		if e != nil {
			t.Fatal(e)
		}
		_, e = s.IssueReport(ctx, orgA, os.Getenv("SITEWISE_REPORT_CRASH_ID"), userA, "restart-test", store.IssueOptions{Version: version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Crash recovery fixture"}, blobs)
		if e != nil {
			t.Fatal(e)
		}
		return
	}
	if _, err := s.RefreshReport(ctx, orgA, os.Getenv("SITEWISE_REPORT_CRASH_ID"), userA, "restart-test", true); err != nil {
		t.Fatal(err)
	}
}

type reportCrashProcess struct {
	cmd    *exec.Cmd
	done   chan struct{}
	err    error
	output bytes.Buffer
}

func startReportCrashProcess(t *testing.T, reportID, app string, extraEnv ...string) *reportCrashProcess {
	t.Helper()
	// Execute and kill only the process we create, never a name or PID search.
	p := &reportCrashProcess{cmd: exec.Command(os.Args[0], "-test.run=^TestReportCrashChild$", "-test.count=1"), done: make(chan struct{})}
	p.cmd.Env = append(os.Environ(), "SITEWISE_REPORT_CRASH_CHILD=1", "SITEWISE_REPORT_CRASH_ID="+reportID, "SITEWISE_REPORT_CRASH_APP="+app)
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

func TestReportAssemblyProcessCrashRecovery(t *testing.T) {
	for _, existingDraft := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing_draft_%t", existingDraft), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s, _, _ := workStore(t)
			pool := rawPool(t)
			pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Crash recovery engineering", LifecycleStatus: "planned"})
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", pkg.ID)
			if err != nil {
				t.Fatal(err)
			}
			var previous store.ReportDraft
			if existingDraft {
				previous, err = s.RefreshReport(ctx, orgA, r.ID, userA, "before-crash", true)
				if err != nil {
					t.Fatal(err)
				}
				previous, err = s.EditReport(ctx, orgA, r.ID, userA, "package:"+pkg.ID, "Keep this protected wording", previous.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
			// Include all stored fields, not just the response: failed assembly must
			// not advance the report revision, publish events or lose protected edits.
			snapshot := func() string {
				t.Helper()
				var raw string
				err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
 'reports',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM reports r WHERE org_id=$1::uuid),
 'versions',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM report_versions v WHERE org_id=$1::uuid),
 'references',(SELECT jsonb_agg(to_jsonb(r) ORDER BY report_version_id,citation_id) FROM report_references r WHERE org_id=$1::uuid),
 'edits',(SELECT jsonb_agg(to_jsonb(e) ORDER BY report_version_id,target_id) FROM report_edits e WHERE org_id=$1::uuid),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_revisions r WHERE org_id=$1::uuid),
 'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM events e WHERE org_id=$1::uuid)
 )::text`, orgA).Scan(&raw)
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
			// SHARE allows reads but stops reference replacement. RefreshReport
			// reaches this lock only after writing its new sections/version.
			if _, err := barrier.Exec(ctx, "LOCK TABLE report_references IN SHARE MODE"); err != nil {
				t.Fatal(err)
			}
			app := fmt.Sprintf("report-crash-%d", time.Now().UnixNano())
			child := startReportCrashProcess(t, r.ID, app)
			for {
				var waiting bool
				if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND query LIKE 'DELETE FROM report_references%')`, app).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case <-child.done:
					t.Fatalf("child exited before crash point: %v\n%s", child.err, &child.output)
				case <-ctx.Done():
					t.Fatal("child did not reach reference replacement", ctx.Err())
				case <-time.After(20 * time.Millisecond):
				}
			}
			if got := snapshot(); got != before {
				t.Fatal("uncommitted assembly became visible")
			}
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
			// A server blocked in a statement can observe a closed socket only
			// after the lock clears. Wait for that exact backend to disappear.
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
					t.Fatal("crashed backend did not release transaction", ctx.Err())
				case <-time.After(20 * time.Millisecond):
				}
			}
			if got := snapshot(); got != before {
				t.Fatal("crash changed committed report state")
			}
			restarted := startReportCrashProcess(t, r.ID, app+"-restart")
			select {
			case <-restarted.done:
				if restarted.err != nil {
					t.Fatalf("restart failed: %v\n%s", restarted.err, &restarted.output)
				}
			case <-ctx.Done():
				t.Fatal("restart did not complete", ctx.Err())
			}
			view, err := s.ReadReport(ctx, orgA, r.ID, "restart-test")
			if err != nil || view.Draft == nil {
				t.Fatalf("missing recovered draft: %+v %v", view, err)
			}
			if len(view.Draft.References) == 0 || len(view.Draft.Sections) != 7 || view.Draft.SourceRevisions.AppBuild != "restart-test" {
				t.Fatal("restart did not publish a complete draft")
			}
			if existingDraft && (view.Draft.ID != previous.ID || view.Draft.Version != previous.Version+1 || len(view.Edits) != 1 || view.Edits[0].Text != "Keep this protected wording") {
				t.Fatal("restart lost draft identity, protected edits or version continuity")
			}
			var versions int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid`, orgA, r.ID).Scan(&versions); err != nil || versions != 1 {
				t.Fatalf("duplicate or missing draft after restart: %d %v", versions, err)
			}
		})
	}
}
