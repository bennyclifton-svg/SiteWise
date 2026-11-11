package store

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestProfileFindsReadsRecentPassagesOfOneProject(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	for _, table := range []string{"documents", "passages", "passage_sources", "passage_calls", "passage_systems"} {
		name := pgx.Identifier{table}.Sanitize()
		run("CREATE TEMP TABLE " + name + " (LIKE public." + name + " INCLUDING ALL) ON COMMIT DROP")
	}
	const org = "15000000-0000-4000-8000-000000000001"
	const project = "15000000-0000-4000-8000-000000000002"
	const otherProject = "15000000-0000-4000-8000-000000000003"
	const foreignOrg = "15000000-0000-4000-8000-000000000099"
	base := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	run(`INSERT INTO documents(org_id,id,project_id,file_id,filename,status) VALUES
 ($1,md5('doc1')::uuid,$2,md5('f1')::uuid,'design-statement.pdf','filed'),
 ($1,md5('doc3')::uuid,$3,md5('f3')::uuid,'other-project.pdf','filed')`, org, project, otherProject)
	// A long document: 6000 passages, each labelled, every other one read for evidence.
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body)
SELECT $1,md5('p'||n)::uuid,md5('doc1')::uuid,n,'Passage '||n||' about a five storey mixed use development' FROM generate_series(1,6000) n`, org)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body) VALUES ($1,md5('q1')::uuid,md5('doc3')::uuid,1,'Another project')`, org)
	run(`INSERT INTO passage_sources(org_id,document_id,passage_id,page,outcome,mapped_keys)
SELECT $1,md5('doc1')::uuid,md5('p'||n)::uuid,1+n/40,'mapped',CASE WHEN n%3=0 THEN ARRAY['hdr.scale.storeys'] ELSE '{}' END FROM generate_series(1,6000) n`, org)
	run(`INSERT INTO passage_systems(org_id,passage_id,system_id) SELECT $1,md5('p'||n)::uuid,'substructure' FROM generate_series(5,6000,5) n`, org)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result,created_at)
SELECT $1,md5('doc1')::uuid,md5('p'||n)::uuid,'label','f','{}',$2::timestamptz + n * interval '1 millisecond' FROM generate_series(1,6000) n`, org, base)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result,created_at)
SELECT $1,md5('doc1')::uuid,md5('p'||n)::uuid,'evidence','f','{}',$2::timestamptz + interval '10 seconds' + n * interval '1 millisecond' FROM generate_series(2,6000,2) n`, org, base)
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result,created_at)
VALUES ($1,md5('doc3')::uuid,md5('q1')::uuid,'label','f','{}',$2::timestamptz + interval '1 millisecond')`, org, base)
	// The same passage ids under another org never leak in.
	run(`INSERT INTO passage_calls(org_id,document_id,passage_id,stage,fingerprint,result,created_at)
SELECT $1,md5('doc1')::uuid,md5('p'||n)::uuid,'label','f','{}',$2::timestamptz FROM generate_series(1,100) n`, foreignOrg, base.Add(time.Hour))
	run("ANALYZE passages; ANALYZE passage_calls; ANALYZE passage_sources")

	docs := []string{md5UUID("doc1"), md5UUID("doc3")}
	finds, err := profileFinds(ctx, tx, org, project, docs, base.Add(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != maxFinds {
		t.Fatalf("finds are capped at %d: %d", maxFinds, len(finds))
	}
	first := finds[0]
	if first.Stage != "label" || first.DocumentID != md5UUID("doc1") || first.Page != 3 || len(first.Systems) != 0 || len(first.Keys) != 0 || first.Excerpt == "" {
		t.Fatalf("first find %+v", first)
	}
	for i, f := range finds {
		if !f.At.After(base.Add(100 * time.Millisecond)) {
			t.Fatalf("find %d before the cursor: %+v", i, f)
		}
		if i > 0 && f.At.Before(finds[i-1].At) {
			t.Fatalf("finds out of order at %d", i)
		}
		if f.DocumentID != md5UUID("doc1") {
			t.Fatalf("a document of another project leaked in: %+v", f)
		}
	}
	// The evidence pass reads later; a cursor past the label pass sees only evidence.
	late, err := profileFinds(ctx, tx, org, project, docs, base.Add(9*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(late) == 0 || late[0].Stage != "evidence" || late[0].Keys == nil || late[0].Systems == nil {
		t.Fatalf("evidence finds %+v", late[:1])
	}
	none, err := profileFinds(ctx, tx, org, project, docs, base.Add(time.Minute))
	if err != nil || len(none) != 0 {
		t.Fatalf("nothing after the last call: %d %v", len(none), err)
	}

	samples := []time.Duration{}
	for i := 0; i < 20; i++ {
		start := time.Now()
		if _, err := profileFinds(ctx, tx, org, project, docs, base.Add(time.Duration(i)*400*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	if samples[10] > 50*time.Millisecond || samples[18] > 150*time.Millisecond {
		t.Fatalf("finds read exceeded p50/p90 budget: %s/%s", samples[10], samples[18])
	}
}

// md5UUID is the fixture id md5(text)::uuid, formatted in Go so the
// assertions do not ask the database.
func md5UUID(s string) string {
	sum := md5.Sum([]byte(s))
	h := hex.EncodeToString(sum[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
