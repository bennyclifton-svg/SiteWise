package store

import (
	"context"
	"github.com/jackc/pgx/v5"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// Temporary tables isolate the stale-statistics fixture. ANALYZE runs before
// the upload; PostgreSQL never auto-analyzes temp tables.
func TestProfileSnapshotAfterUploadWithStaleStatistics(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database != "sitewise_test" {
		t.Fatalf("refusing database %q", cfg.Database)
	}
	st, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	run := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"projects", "project_parts", "profile_facts", "documents", "files", "decisions", "supersessions", "passages", "passage_sources", "profile_user_values", "profile_planning_values"} {
		name := pgx.Identifier{table}.Sanitize()
		run("CREATE TEMP TABLE " + name + " (LIKE public." + name + " INCLUDING ALL) ON COMMIT DROP; ANALYZE " + name)
	}
	// A pooled connection may cache its generic plan before any upload.
	// Empty-table statistics made the old per-fact lateral lookup choose a
	// full passage_sources scan for every fact after a large import.
	run("SET LOCAL plan_cache_mode = force_generic_plan")
	const org = "f3100000-0000-4000-8000-000000000001"
	const project = "f3100000-0000-4000-8000-000000000002"
	const site = "f3100000-0000-4000-8000-000000000003"
	if _, err := readSnapshot(ctx, tx, org, project); err != nil {
		t.Fatal(err)
	}
	run(`INSERT INTO projects(org_id,id,name,site_id) VALUES ($1,$2,'Snapshot fixture',$3)`, org, project, site)
	run(`INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind) VALUES ($1,gen_random_uuid(),$3,$2,'Whole project','whole')`, org, project, site)
	run(`INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type)
 SELECT $1,md5('file'||n)::uuid,$2,decode(md5(n::text)||md5(n::text),'hex'),8,'application/pdf' FROM generate_series(1,1000) n`, org, project)
	run(`INSERT INTO documents(org_id,id,project_id,file_id,filename,status,document_number,revision,profile_read)
 SELECT $1,md5('doc'||n)::uuid,$2,md5('file'||n)::uuid,'plan.pdf','filed','A-'||n,'P1','skip' FROM generate_series(1,1000) n`, org, project)
	run(`INSERT INTO profile_facts(org_id,id,project_id,document_id,passage_id,question_id,value,decided_by,question_version)
 SELECT $1,gen_random_uuid(),$2,md5('doc1')::uuid,md5('passage'||n)::uuid,'det.test.'||n,'included','rule','test' FROM generate_series(1,1000) n`, org, project)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body)
 SELECT $1,md5('passage'||n)::uuid,md5('doc1')::uuid,n,'Source text' FROM generate_series(1,1000) n`, org)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body)
 SELECT $1,md5('passage'||n)::uuid,md5('doc2')::uuid,n,'Other source text' FROM generate_series(2001,8000) n`, org)
	run(`INSERT INTO passage_sources(org_id,passage_id,page,location,section,start_offset,end_offset)
 SELECT $1,md5('passage'||n)::uuid,7,'Sheet A','Fire',10,20 FROM generate_series(1,999) n`, org)
	run(`INSERT INTO passage_sources(org_id,passage_id,page,location,section,start_offset,end_offset)
 SELECT $1,md5('passage'||n)::uuid,7,'Sheet A','Fire',10,20 FROM generate_series(2001,8000) n`, org)
	run(`INSERT INTO decisions(org_id,id,document_id,field,value,band,decided_by) VALUES ($1,gen_random_uuid(),md5('doc1')::uuid,'kind','drawing','green','rule')`, org)
	run(`INSERT INTO supersessions(org_id,document_id,prior_document_id) VALUES ($1,md5('doc2')::uuid,md5('doc1')::uuid)`, org)
	// The same source IDs may exist in another org; ID alone is not authority.
	run(`INSERT INTO passage_sources(org_id,passage_id,page,section)
SELECT 'f3100000-0000-4000-8000-000000000099',passage_id,99,'Foreign' FROM passage_sources`)
	// Autovacuum can refresh location statistics before passage statistics.
	// The ownership join must remain bounded in that mixed-statistics state.
	run("ANALYZE passage_sources")
	var samples []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		snap, err := readSnapshot(ctx, tx, org, project)
		samples = append(samples, time.Since(start))
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Facts) != 1000 || len(snap.Parts) != 1 {
			t.Fatalf("counts: %d facts, %d parts", len(snap.Facts), len(snap.Parts))
		}
		missing := 0
		for _, f := range snap.Facts {
			if f.Filename != "plan.pdf" || f.DocumentNumber != "A-1" || f.Revision != "P1" || f.DocumentKind != "drawing" || f.ReadSetting != "skip" || !f.Superseded || len(f.FileSHA256) != 64 {
				t.Fatalf("source metadata lost: %+v", f)
			}
			if f.Page == 0 {
				missing++
				continue
			}
			if f.Page != 7 || f.Location != "Sheet A" || f.Section != "Fire" || f.StartOffset != 10 || f.EndOffset != 20 {
				t.Fatalf("location lost: %+v", f)
			}
		}
		if missing != 1 {
			t.Fatalf("missing source must retain its fact: %d", missing)
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	t.Logf("stale-statistics snapshot p50 %s / p90 %s", samples[9], samples[17])
	if samples[9] > 50*time.Millisecond || samples[17] > 150*time.Millisecond {
		t.Fatal("snapshot alone exceeds profile_edit 50/150 ms budget")
	}
	run(`INSERT INTO profile_facts(org_id,id,project_id,document_id,question_id,value,decided_by,question_version)
VALUES ($1,gen_random_uuid(),$2,md5('doc2')::uuid,'det.no_passage','unknown','rule','test')`, org, project)
	snap, err := readSnapshot(ctx, tx, org, project)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range snap.Facts {
		if f.QuestionID != "det.no_passage" {
			continue
		}
		found = true
		if f.DocumentNumber != "A-2" || f.DocumentKind != "" || f.Superseded || f.PassageID != "" || f.Page != 0 || f.Value != "unknown" {
			t.Fatalf("document metadata mixed or absent source changed: %+v", f)
		}
	}
	if !found {
		t.Fatal("fact without passage was dropped")
	}
	snap, err = readSnapshot(ctx, tx, "f3100000-0000-4000-8000-000000000099", project)
	if err != nil || len(snap.Facts) != 0 || len(snap.Parts) != 0 {
		t.Fatalf("wrong-org snapshot: %+v %v", snap, err)
	}
	// Model a replacement becoming visible immediately after the fact query.
	// Splitting provenance into later queries would retain the old facts but
	// lose their locations. One statement must return the complete old state.
	interleaved := &snapshotInterleave{tx: tx, after: func() {
		run("DELETE FROM profile_facts; DELETE FROM passage_sources")
	}}
	snap, err = readSnapshot(ctx, interleaved, org, project)
	if err != nil {
		t.Fatal(err)
	}
	located := 0
	for _, f := range snap.Facts {
		if f.Page == 7 {
			located++
		}
	}
	if len(snap.Facts) != 1001 || located != 999 {
		t.Fatalf("replacement tore fact provenance: %d facts, %d locations", len(snap.Facts), located)
	}

}

// Release the query's connection before simulating another committed writer.
// The callback is once-only because rows can be closed more than once.
type snapshotInterleave struct {
	tx    pgx.Tx
	after func()
}

func (q *snapshotInterleave) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := q.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if strings.Contains(sql, "FROM profile_facts") && q.after != nil {
		after := q.after
		q.after = nil
		return &snapshotInterleaveRows{Rows: rows, after: after}, nil
	}
	return rows, nil
}

type snapshotInterleaveRows struct {
	pgx.Rows
	after func()
}

func (r *snapshotInterleaveRows) Close() {
	r.Rows.Close()
	if r.after != nil {
		after := r.after
		r.after = nil
		after()
	}
}
