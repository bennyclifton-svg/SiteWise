package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"
)

func TestCostSourceMigrationPreservesFrozenWorkDimensions(t *testing.T) {
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
	schema := pgx.Identifier{fmt.Sprintf("cost_sources_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/021p_cost_line_sources.sql"
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
	const org = "45000000-0000-4000-8000-000000000001"
	run(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	run(`INSERT INTO sites(org_id,id,label) VALUES($1,md5('site')::uuid,'Site')`, org)
	run(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5('project')::uuid,'Project',md5('site')::uuid)`, org)
	run(`INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind) VALUES($1,md5('part')::uuid,md5('site')::uuid,md5('project')::uuid,'Building','whole')`, org)
	run(`INSERT INTO work_items(org_id,project_id,id,site_id,part_id,system_id,action,title,origin,meaning,inclusion,review_status) VALUES($1,md5('project')::uuid,md5('work')::uuid,md5('site')::uuid,md5('part')::uuid,'structure','new','Frame','user','requirement','included','accepted_for_planning')`, org)
	run(`INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,tax_basis) VALUES($1,md5('project')::uuid,md5('base')::uuid,1,'draft','ex_tax')`, org)
	run(`INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1,md5('project')::uuid,md5('item')::uuid,md5('base')::uuid)`, org)
	run(`INSERT INTO cost_item_revisions(org_id,project_id,plan_version_id,cost_item_id,label,line_kind,work_item_id,system_id,part_id,origin,meaning) VALUES($1,md5('project')::uuid,md5('base')::uuid,md5('item')::uuid,'Frame','works',md5('work')::uuid,'structure',md5('part')::uuid,'user','allowance')`, org)
	run(`UPDATE cost_plan_versions SET status='baseline',frozen_at=now() WHERE org_id=$1`, org)
	run("SET CONSTRAINTS ALL IMMEDIATE")
	raw, e := migrationFiles.ReadFile(migration)
	if e != nil {
		t.Fatal(e)
	}
	run(string(raw))
	var system, work string
	if e = tx.QueryRow(ctx, `SELECT system_id,work_item_id::text FROM cost_item_revisions WHERE org_id=$1`, org).Scan(&system, &work); e != nil || system != "structure" || work == "" {
		t.Fatal("snapshot lost", system, work, e)
	}
	run(`INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,tax_basis) VALUES($1,md5('project')::uuid,md5('draft')::uuid,2,'draft','ex_tax')`, org)
	run(`INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1,md5('project')::uuid,md5('system-item')::uuid,md5('draft')::uuid)`, org)
	run(`INSERT INTO cost_item_revisions(org_id,project_id,plan_version_id,cost_item_id,label,line_kind,system_id,source_key,origin,meaning) VALUES($1,md5('project')::uuid,md5('draft')::uuid,md5('system-item')::uuid,'Frame','works','structure','works:structure','assumption','allowance')`, org)
}
