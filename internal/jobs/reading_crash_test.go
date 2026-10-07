package jobs_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

type readingCrashAsk struct{ forbid bool }

func (a readingCrashAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	if a.forbid {
		return jev.Result{}, errors.New("restart repeated an already cached judgment")
	}
	confidence := .99
	result := jev.Result{Answers: map[string]jev.Answer{}}
	for id, question := range call.Questions {
		answer := jev.Answer{Type: question.Type, Choice: "not_stated", Confidence: &confidence}
		switch id {
		case "source.category":
			answer.Choice = "requirement"
		case "source.scope":
			answer.Choice = "whole_project"
		case "hdr.work_type":
			answer.Choice = "refurb"
		case "system.hydraulic", "system.hydraulic.gas":
			answer.Noul = .99
		case "sys.hydraulic.gas.presence":
			answer.Choice = "included"
		case "sys.hydraulic.gas.action":
			answer.Choice = "replace"
		}
		result.Answers[id] = answer
	}
	return result, nil
}

func readingCrashThresholds() profile.Thresholds {
	return profile.Thresholds{Version: "crash-test", Amber: map[string]float64{"header": .6, "presence": .6, "action": .8, "location.n4": .8}}
}

func TestReadingCrashChild(t *testing.T) {
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
	s, err := store.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w := &jobs.Worker{Store: s, Lease: time.Second, Kinds: []string{os.Getenv("SITEWISE_READING_CRASH_KIND")}, Catalog: loadKnowledge(t), MinNoul: .6,
		Profile: readingCrashThresholds(), Ask: readingCrashAsk{forbid: os.Getenv("SITEWISE_READING_CRASH_RESTART") == "1"}}
	if err := w.Once(ctx, os.Getenv("SITEWISE_JOB_CRASH_ORG")); err != nil {
		t.Fatal(err)
	}
}

func TestReadingProcessCrashRecovery(t *testing.T) {
	for _, kind := range []string{store.JobKindLabel, store.JobKindEvidence} {
		t.Run(kind, func(t *testing.T) {
			s := openStore(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			org, project := newID(t), newID(t)
			seedOrg(t, s, org, project)
			if _, err := s.EnsureWholePart(ctx, org, project); err != nil {
				t.Fatal(err)
			}
			doc := seedDoc(t, s, org, project, store.StatusFiled)
			if err := s.ReplaceSource(ctx, org, doc, store.DocumentSource{
				Source: []store.SourcePage{{Text: "Refurbishment replaces gas throughout the project."}},
				Units: []store.SourceUnit{{Body: "Replace gas pipework throughout the project."},
					{Body: "Refurbish the project and replace all gas fittings."}},
			}); err != nil {
				t.Fatal(err)
			}
			passages, err := s.DocumentPassages(ctx, org, doc)
			if err != nil || len(passages) != 2 {
				t.Fatalf("source fixture: %+v %v", passages, err)
			}
			for _, passage := range passages {
				if err := s.SetPassageSystems(ctx, org, passage.ID, []string{"hydraulic.gas"}); err != nil {
					t.Fatal(err)
				}
			}
			cat := loadKnowledge(t)
			if kind == store.JobKindEvidence {
				// Use the real preceding label stage, including its durable cache,
				// rather than inventing the evidence stage's starting labels.
				w := &jobs.Worker{Store: s, Catalog: cat, MinNoul: .6, Profile: readingCrashThresholds(), Ask: readingCrashAsk{}}
				if err := w.Perform(ctx, store.ClaimedJob{OrgID: org, DocumentID: doc, Kind: store.JobKindLabel}); err != nil {
					t.Fatal(err)
				}
			}
			build := store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion, ThresholdsVersion: "crash-test",
				Compute: func(s store.ProfileSnapshot) []profile.Row {
					return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: readingCrashThresholds()}, cat)
				}}
			confidence := .99
			if err := s.WithProfile(build).ReplaceDocumentFacts(ctx, org, doc, []string{"hdr.", "sys."}, profile.QuestionVersion, []store.StoredFact{
				{PassageID: passages[0].ID, QuestionID: "hdr.work_type", Value: "new", Confidence: &confidence, DecidedBy: "jev"},
				{PassageID: passages[0].ID, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Confidence: &confidence, DecidedBy: "jev"},
				{PassageID: passages[0].ID, QuestionID: "sys.hydraulic.gas.action", Value: "repair", Confidence: &confidence, DecidedBy: "jev"},
			}); err != nil {
				t.Fatal(err)
			}
			if err := s.EnqueueStage(ctx, org, doc, kind); err != nil {
				t.Fatal(err)
			}
			beforeWorks, err := s.ReadWorks(ctx, org, project)
			if err != nil || len(beforeWorks) != 1 {
				t.Fatalf("baseline work projection: %+v %v", beforeWorks, err)
			}
			pool, err := pgxpool.New(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			snapshot := func() string {
				t.Helper()
				var raw string
				if err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
 'facts',(SELECT jsonb_agg(to_jsonb(f) ORDER BY id) FROM profile_facts f WHERE org_id=$1::uuid),
 'rows',(SELECT jsonb_agg(to_jsonb(r) ORDER BY part_id,key,scope) FROM profile_rows r WHERE org_id=$1::uuid),
 'build',(SELECT jsonb_agg(to_jsonb(b) ORDER BY project_id) FROM profile_builds b WHERE org_id=$1::uuid),
 'works',(SELECT jsonb_agg(to_jsonb(w) ORDER BY id) FROM work_items w WHERE org_id=$1::uuid),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_revisions r WHERE org_id=$1::uuid),
 'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM events e WHERE org_id=$1::uuid)
 )::text`, org).Scan(&raw); err != nil {
					t.Fatal(err)
				}
				return raw
			}
			cached := func() string {
				t.Helper()
				var raw string
				if err := pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(c) ORDER BY passage_id,stage),'[]'::jsonb)::text FROM passage_calls c WHERE org_id=$1::uuid`, org).Scan(&raw); err != nil {
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
			if _, err := barrier.Exec(ctx, "LOCK TABLE profile_rows IN SHARE MODE"); err != nil {
				t.Fatal(err)
			}
			app := fmt.Sprintf("reading-crash-%s-%d", kind, time.Now().UnixNano())
			child := startWorkerCrashProcess(t, org, app, "TestReadingCrashChild", "SITEWISE_READING_CRASH_KIND="+kind)
			for {
				var waiting bool
				if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND query LIKE 'DELETE FROM profile_rows%')`, app).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case <-child.done:
					t.Fatalf("reader exited before projection boundary: %v\n%s", child.err, &child.output)
				case <-ctx.Done():
					t.Fatal("reader did not reach projection boundary", ctx.Err())
				case <-time.After(20 * time.Millisecond):
				}
			}
			var jobID string
			if err := pool.QueryRow(ctx, `SELECT id::text FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind=$3 AND status='leased' AND attempts=1`, org, doc, kind).Scan(&jobID); err != nil {
				t.Fatal(err)
			}
			var calls int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM passage_calls WHERE org_id=$1::uuid AND stage=$2`, org, kind).Scan(&calls); err != nil || calls != 2 {
				t.Fatalf("readings not cached before crash: %d %v", calls, err)
			}
			cacheBefore := cached()
			if snapshot() != before {
				t.Fatal("partial reading changed the completed profile")
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
					t.Fatal("reader backend did not close", ctx.Err())
				case <-time.After(20 * time.Millisecond):
				}
			}
			if snapshot() != before || cached() != cacheBefore {
				t.Fatal("crash lost cached results or changed completed projection")
			}
			view, err := s.ReadProfile(ctx, org, project, nil)
			if err != nil || view.PendingDocuments != 1 {
				t.Fatalf("interrupted reading is not visible as pending: %+v %v", view, err)
			}
			for {
				orgs, err := s.OrgsWithJobKindsSince(ctx, []string{kind}, time.Time{})
				if err != nil {
					t.Fatal(err)
				}
				if contains(orgs, org) {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal("abandoned reading lease not discovered", ctx.Err())
				case <-time.After(20 * time.Millisecond):
				}
			}
			restarted := startWorkerCrashProcess(t, org, app+"-restart", "TestReadingCrashChild", "SITEWISE_READING_CRASH_KIND="+kind, "SITEWISE_READING_CRASH_RESTART=1")
			select {
			case <-restarted.done:
				if restarted.err != nil {
					t.Fatalf("reader restart failed: %v\n%s", restarted.err, &restarted.output)
				}
			case <-ctx.Done():
				t.Fatal("reader restart timed out", ctx.Err())
			}
			job, err := s.GetJob(ctx, org, jobID)
			if err != nil || job.Status != store.JobStatusDone || job.Attempts != 2 {
				t.Fatalf("reading did not recover in place: %+v %v", job, err)
			}
			if cached() != cacheBefore {
				t.Fatal("restart rewrote the cached judgments")
			}
			after, err := s.ReadProfile(ctx, org, project, nil)
			if err != nil || after.Revision != view.Revision+1 || after.InputFingerprint == view.InputFingerprint {
				t.Fatalf("recovered profile revision/fingerprint: %+v %v", after, err)
			}
			question, value := "hdr.work_type", "refurb"
			if kind == store.JobKindEvidence {
				question, value = "sys.hydraulic.gas.action", "replace"
				items, err := s.ReadWorks(ctx, org, project)
				if err != nil || len(items) != 1 || items[0].ID != beforeWorks[0].ID || items[0].Action != "replace" {
					t.Fatalf("recovery did not update the stable work item: %+v %v", items, err)
				}
			} else {
				found := false
				for _, row := range after.Rows {
					found = found || row.Key == question && row.Value == value
				}
				if !found {
					t.Fatal("recovery did not project the new work type")
				}
			}
			var facts, completions int
			if err := pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM profile_facts WHERE org_id=$1::uuid AND document_id=$2::uuid AND question_id=$3 AND value=$4),
 (SELECT count(*) FROM events WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='job' AND payload::jsonb->>'status'='done')`, org, doc, question, value).Scan(&facts, &completions); err != nil || facts != 2 || completions != 1 {
				t.Fatalf("incomplete/duplicate recovered readings: facts=%d completions=%d error=%v", facts, completions, err)
			}
		})
	}
}
