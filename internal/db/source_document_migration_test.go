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

func TestSourceDocumentCoverageMigration(t *testing.T) {
	ctx := context.Background()
	cfg, e := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if e != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required")
	}
	conn, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close(ctx)
	tx, e := conn.Begin(ctx)
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
	schema := pgx.Identifier{fmt.Sprintf("source_document_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/021h_source_document_coverage.sql"
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
	const org = "41000000-0000-4000-8000-000000000001"
	run(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	run(`INSERT INTO sites(org_id,id,label) VALUES($1,md5('site')::uuid,'Site')`, org)
	run(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5('project')::uuid,'Project',md5('site')::uuid)`, org)
	run(`INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) SELECT $1,md5('file'||n)::uuid,md5('project')::uuid,decode(repeat(lpad(to_hex(n),2,'0'),32),'hex'),1,'text/plain' FROM generate_series(1,2)n`, org)
	run(`INSERT INTO documents(org_id,id,project_id,file_id,filename,status) SELECT $1,md5('doc'||n)::uuid,md5('project')::uuid,md5('file'||n)::uuid,'fixture','filed' FROM generate_series(1,2)n`, org)
	run(`INSERT INTO passages(org_id,id,document_id,ordinal,body) SELECT $1,md5('passage'||n)::uuid,md5('doc1')::uuid,n,'Fixture' FROM generate_series(1,3)n`, org)
	run(`INSERT INTO passage_sources(org_id,passage_id,outcome) VALUES($1,md5('passage1')::uuid,'mapped')`, org)
	raw, e := migrationFiles.ReadFile(migration)
	if e != nil {
		t.Fatal(e)
	}
	run(string(raw))
	var correct bool
	if e = tx.QueryRow(ctx, `SELECT document_id=md5('doc1')::uuid AND outcome='mapped' FROM passage_sources WHERE org_id=$1`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("backfill changed or missed source", e)
	}
	run(`INSERT INTO passage_sources(org_id,passage_id,outcome) VALUES($1,md5('passage2')::uuid,'pending')`, org)
	if e = tx.QueryRow(ctx, `SELECT document_id=md5('doc1')::uuid FROM passage_sources WHERE org_id=$1 AND passage_id=md5('passage2')::uuid`, org).Scan(&correct); e != nil || !correct {
		t.Fatal("compatibility fill failed", e)
	}
	for i, sql := range []string{
		`INSERT INTO passage_sources(org_id,passage_id,document_id) VALUES($1,md5('passage3')::uuid,md5('doc2')::uuid)`,
		`INSERT INTO passage_sources(org_id,passage_id) VALUES($1,md5('passage3')::uuid)`,
	} {
		sub, e := tx.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		inputOrg := org
		if i == 1 {
			inputOrg = "41000000-0000-4000-8000-000000000099"
		}
		_, e = sub.Exec(ctx, sql, inputOrg)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "23503" || pg.ConstraintName != "passage_sources_org_id_passage_id_fkey" {
			t.Fatal("wrong parent not rejected by FK", e)
		}
		sub.Rollback(ctx)
	}
}
