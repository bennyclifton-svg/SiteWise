package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"
)

func TestPassageCallDocumentMigration(t *testing.T) {
	ctx := context.Background()
	cfg, e := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if e != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required")
	}
	c, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close(ctx)
	tx, e := c.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	run := func(sql string, args ...any) {
		t.Helper()
		if _, e := tx.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("call_document_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/021k_passage_call_document_coverage.sql"
	for _, name := range names {
		if name >= migration {
			break
		}
		raw, e := migrationFiles.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		run(string(raw))
	}
	const org = "42000000-0000-4000-8000-000000000001"
	run(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	run(`INSERT INTO sites(org_id,id,label) VALUES($1,md5('site')::uuid,'Site')`, org)
	run(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5('project')::uuid,'Project',md5('site')::uuid)`, org)
	run(`INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) SELECT $1,md5('file'||n)::uuid,md5('project')::uuid,decode(repeat(lpad(to_hex(n),2,'0'),32),'hex'),1,'text/plain' FROM generate_series(1,2)n`, org)
	run(`INSERT INTO documents(org_id,id,project_id,file_id,filename,status) SELECT $1,md5('doc'||n)::uuid,md5('project')::uuid,md5('file'||n)::uuid,'fixture','filed' FROM generate_series(1,2)n`, org)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body) SELECT $1,md5('passage'||n)::uuid,md5('doc1')::uuid,n,'Fixture' FROM generate_series(1,3)n`, org)
	run(`INSERT INTO passage_calls(org_id,passage_id,stage,fingerprint,result) VALUES($1,md5('passage1')::uuid,'evidence','frozen','{"retained":true}')`, org)
	raw, e := migrationFiles.ReadFile(migration)
	if e != nil {
		t.Fatal(e)
	}
	run(string(raw))
	var correct bool
	if e = tx.QueryRow(ctx, `SELECT document_id=md5('doc1')::uuid AND fingerprint='frozen' AND result='{"retained":true}'::jsonb FROM passage_calls WHERE org_id=$1`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("call backfill changed result", e)
	}
	run(`INSERT INTO passage_calls(org_id,passage_id,stage,fingerprint,result) VALUES($1,md5('passage2')::uuid,'label','legacy','{}'),($1,md5('passage2')::uuid,'evidence','legacy','{}')`, org)
	if e = tx.QueryRow(ctx, `SELECT count(*)=2 AND bool_and(document_id=md5('doc1')::uuid) FROM passage_calls WHERE org_id=$1 AND passage_id=md5('passage2')::uuid`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("legacy inserts lost stages", e)
	}
	for i, sql := range []string{
		`INSERT INTO passage_calls(org_id,passage_id,document_id,stage,fingerprint,result) VALUES($1,md5('passage3')::uuid,md5('doc2')::uuid,'evidence','wrong','{}')`,
		`INSERT INTO passage_calls(org_id,passage_id,stage,fingerprint,result) VALUES($1,md5('passage3')::uuid,'evidence','wrong','{}')`,
		`UPDATE passage_calls SET document_id=md5('doc2')::uuid WHERE org_id=$1`,
	} {
		sub, e := tx.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		input := org
		if i == 1 {
			input = "42000000-0000-4000-8000-000000000099"
		}
		_, e = sub.Exec(ctx, sql, input)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "23503" || pg.ConstraintName != "passage_calls_org_id_passage_id_fkey" {
			t.Fatal("wrong parent accepted", e)
		}
		sub.Rollback(ctx)
	}
	run(`DELETE FROM passages WHERE org_id=$1 AND id=md5('passage2')::uuid`, org)
	if e = tx.QueryRow(ctx, `SELECT count(*)=1 FROM passage_calls WHERE org_id=$1`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("passage cascade changed", e)
	}
	run(`DELETE FROM documents WHERE org_id=$1 AND id=md5('doc1')::uuid`, org)
	if e = tx.QueryRow(ctx, `SELECT count(*)=0 FROM passage_calls WHERE org_id=$1`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("document cascade changed", e)
	}
}
