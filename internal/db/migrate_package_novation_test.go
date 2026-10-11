package db

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// No client had elected to novate before this migration, so a novated
// package becomes an option, an unnovated one stays no.
func TestPackageNovationMigrationMakesOptions(t *testing.T) {
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
	run := func(sql string, args ...any) error {
		_, err := tx.Exec(ctx, sql, args...)
		return err
	}
	must := func(sql string, args ...any) {
		t.Helper()
		if err := run(sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("novation_%d", time.Now().UnixNano())}.Sanitize()
	must("CREATE SCHEMA " + schema)
	must("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/021q_package_novation_state.sql"
	for _, name := range names {
		if name >= migration {
			break
		}
		raw, _ := migrationFiles.ReadFile(name)
		must(string(raw))
	}
	const org = "46000000-0000-4000-8000-000000000001"
	must(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	must(`INSERT INTO sites(org_id,id,label) VALUES($1,md5('site')::uuid,'Site')`, org)
	must(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5('project')::uuid,'Project',md5('site')::uuid)`, org)
	for _, p := range []struct {
		id, kind string
		novation bool
	}{{"novated", "services", true}, {"plain", "services", false}, {"works", "works", false}} {
		scope := "NULL"
		if p.kind == "works" {
			scope = "'trade'"
		}
		must(`INSERT INTO packages(org_id,id,project_id,kind,works_scope,title,novation,lifecycle_status,origin,review_status,meaning,provenance) VALUES($1,md5($2)::uuid,md5('project')::uuid,$3,`+scope+`,$2,$4,'planned','user','accepted_for_planning','stated','{}')`, org, p.id, p.kind, p.novation)
	}
	must("SET CONSTRAINTS ALL IMMEDIATE")
	raw, _ := migrationFiles.ReadFile(migration)
	must(string(raw))
	for id, want := range map[string]string{"novated": "option", "plain": "no", "works": "no"} {
		var got string
		if e := tx.QueryRow(ctx, `SELECT novation FROM packages WHERE org_id=$1 AND id=md5($2)::uuid`, org, id).Scan(&got); e != nil || got != want {
			t.Fatalf("%s: %q %v", id, got, e)
		}
	}
	must("SAVEPOINT bad")
	if run(`UPDATE packages SET novation='maybe' WHERE org_id=$1 AND id=md5('plain')::uuid`, org) == nil {
		t.Fatal("unknown novation state stored")
	}
	must("ROLLBACK TO SAVEPOINT bad")
	if run(`UPDATE packages SET novation='yes' WHERE org_id=$1 AND id=md5('works')::uuid`, org) == nil {
		t.Fatal("works package novated")
	}
}
