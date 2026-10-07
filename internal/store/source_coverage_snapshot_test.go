package store

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSourceCoverageWithStaleStatistics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated sitewise_test database required")
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	run := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"documents", "document_sources", "passages", "passage_sources", "passage_calls", "supersessions"} {
		name := pgx.Identifier{table}.Sanitize()
		run("CREATE TEMP TABLE " + name + " (LIKE public." + name + " INCLUDING ALL) ON COMMIT DROP; ANALYZE " + name)
	}
	const org = "14000000-0000-4000-8000-000000000001"
	const project = "14000000-0000-4000-8000-000000000002"
	run("SET LOCAL plan_cache_mode = force_generic_plan")
	if _, err := sourceCoverage(ctx, tx, org, project); err != nil {
		t.Fatal(err)
	}
	run(`INSERT INTO documents(org_id,id,project_id,file_id,filename,status)
SELECT $1,md5('doc'||n)::uuid,$2,md5('file'||n)::uuid,'fixture.pdf','filed' FROM generate_series(1,38) n`, org, project)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body)
SELECT $1,md5('passage'||n)::uuid,md5('doc1')::uuid,n,'Fixture' FROM generate_series(1,6000) n`, org)
	run(`INSERT INTO passage_sources(org_id,document_id,passage_id,outcome)
SELECT $1,md5('doc1')::uuid,md5('passage'||n)::uuid,(ARRAY['pending','mapped','background','needs_mapping'])[1+n%4] FROM generate_series(2,6000) n`, org)
	run(`INSERT INTO document_sources(org_id,document_id,version,pages,source) VALUES($1,md5('doc1')::uuid,$2,10,'[]')`, org, SourceVersion)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result)
SELECT $1,md5('doc1')::uuid,md5('passage'||n)::uuid,'evidence','fixture','{}' FROM generate_series(2,6000,2) n`, org)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result)
SELECT '14000000-0000-4000-8000-000000000099',md5('doc1')::uuid,md5('passage'||n)::uuid,'evidence','foreign','{}' FROM generate_series(1,5999,2) n`)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result)
SELECT $1,md5('doc1')::uuid,md5('passage'||n)::uuid,'label','fixture','{}' FROM generate_series(2,6000,2) n`, org)
	run(`INSERT INTO passage_sources(org_id,document_id,passage_id,outcome)
SELECT '14000000-0000-4000-8000-000000000099',document_id,passage_id,'mapped' FROM passage_sources WHERE org_id=$1`, org)
	run(`INSERT INTO supersessions(org_id,document_id,prior_document_id) VALUES($1,md5('doc3')::uuid,md5('doc2')::uuid)`, org)
	samples := []time.Duration{}
	for i := 0; i < 20; i++ {
		start := time.Now()
		coverage, err := sourceCoverage(ctx, tx, org, project)
		samples = append(samples, time.Since(start))
		if err != nil {
			t.Fatal(err)
		}
		if len(coverage) != 37 {
			t.Fatalf("superseded coverage count %d", len(coverage))
		}
		found := false
		for _, c := range coverage {
			if c.Units == 0 {
				continue
			}
			found = true
			if c.Units != 6000 || c.Labelled != 4499 || c.Evidence != 3000 || c.Mapped != 1499 || c.Background != 1500 || c.NeedsMapping != 1500 || !c.Current || c.Pages != 10 {
				t.Fatalf("wrong coverage %+v", c)
			}
		}
		if !found {
			t.Fatal("lost source")
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	t.Logf("coverage p50=%s p90=%s", samples[9], samples[17])
	if samples[9] > 50*time.Millisecond || samples[17] > 150*time.Millisecond {
		t.Fatal("coverage exceeds profile read budget")
	}
	// Entering and leaving the partial index must not leave stale counts.
	for _, state := range []struct {
		outcome                      string
		labelled, mapped, background int
	}{
		{"pending", 4498, 1499, 1499},
		{"mapped", 4499, 1500, 1499},
		{"background", 4499, 1499, 1500},
	} {
		run(`UPDATE passage_sources SET outcome=$2 WHERE org_id=$1 AND passage_id=md5('passage2')::uuid`, org, state.outcome)
		coverage, err := sourceCoverage(ctx, tx, org, project)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range coverage {
			if c.Units == 0 {
				continue
			}
			if c.Units != 6000 || c.Evidence != 3000 || c.NeedsMapping != 1500 || c.Labelled != state.labelled || c.Mapped != state.mapped || c.Background != state.background {
				t.Fatalf("after %s: %+v", state.outcome, c)
			}
		}
	}
}
