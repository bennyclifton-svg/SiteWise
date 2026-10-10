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

// Retiring estimate must not lose a draft figure or touch a frozen baseline.
func TestCostEstimateFoldsIntoDraftBudget(t *testing.T) {
	ctx := context.Background()
	cfg, e := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if e != nil || cfg.Database != "sitewise_test" {
		t.Skip("dedicated test database required")
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
			t.Fatal(sql, e)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("cost_estimate_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, e := fs.Glob(migrationFiles, "migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(names)
	const migration = "migrations/021o_cost_retire_estimate.sql"
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
	const org = "43000000-0000-4000-8000-000000000001"
	run(`INSERT INTO orgs(id,name) VALUES($1,'Fixture')`, org)
	run(`INSERT INTO sites(org_id,id,label) VALUES($1,md5('site')::uuid,'Site')`, org)
	run(`INSERT INTO projects(org_id,id,name,site_id) VALUES($1,md5('project')::uuid,'Project',md5('site')::uuid)`, org)
	item := func(version, name string) {
		t.Helper()
		run(`INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1,md5('project')::uuid,md5($2)::uuid,md5('base')::uuid) ON CONFLICT DO NOTHING`, org, name)
		run(`INSERT INTO cost_item_revisions(org_id,project_id,plan_version_id,cost_item_id,label,line_kind,category,origin,meaning) VALUES($1,md5('project')::uuid,md5($2)::uuid,md5($3)::uuid,$3,'project_wide','other','user','allowance')`, org, version, name)
	}
	value := func(version, name, metric, state string, amount any) {
		t.Helper()
		run(`INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,origin,meaning) VALUES($1,md5('project')::uuid,md5($2)::uuid,md5($3)::uuid,$4,$5,$6::numeric,'user','allowance')`, org, version, name, metric, state, amount)
	}
	// The baseline is written as a draft, then frozen, as the app does.
	run(`INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,tax_basis) VALUES($1,md5('project')::uuid,md5('base')::uuid,1,'draft','ex_tax')`, org)
	item("base", "frozen")
	value("base", "frozen", "estimate", "known", "120.00")
	run(`UPDATE cost_plan_versions SET status='baseline',frozen_at=now() WHERE org_id=$1 AND id=md5('base')::uuid`, org)
	run(`INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,tax_basis) VALUES($1,md5('project')::uuid,md5('draft')::uuid,2,'draft','ex_tax')`, org)
	for _, name := range []string{"unknown-budget", "no-budget", "both", "unknown-estimate"} {
		item("draft", name)
	}
	value("draft", "unknown-budget", "estimate", "known", "120.00")
	value("draft", "unknown-budget", "budget", "unknown", nil)
	value("draft", "no-budget", "estimate", "known", "130.00")
	value("draft", "both", "estimate", "known", "120.00")
	value("draft", "both", "budget", "known", "100.00")
	value("draft", "unknown-estimate", "estimate", "unknown", nil)

	raw, e := migrationFiles.ReadFile(migration)
	if e != nil {
		t.Fatal(e)
	}
	run(string(raw))

	budget := func(name string) (amount *string, moved bool, found bool) {
		t.Helper()
		e := tx.QueryRow(ctx, `SELECT amount::text, coalesce(provenance->>'moved_from'='estimate',false) FROM cost_values WHERE org_id=$1 AND plan_version_id=md5('draft')::uuid AND cost_item_id=md5($2)::uuid AND metric='budget'`, org, name).Scan(&amount, &moved)
		if e == pgx.ErrNoRows {
			return nil, false, false
		}
		if e != nil {
			t.Fatal(e)
		}
		return amount, moved, true
	}
	for name, want := range map[string]string{"unknown-budget": "120.00", "no-budget": "130.00"} {
		if a, moved, ok := budget(name); !ok || a == nil || *a != want || !moved {
			t.Fatal(name, "estimate not moved into budget", a, moved, ok)
		}
	}
	if a, moved, _ := budget("both"); a == nil || *a != "100.00" || moved {
		t.Fatal("a known budget was overwritten", a, moved)
	}
	if _, _, ok := budget("unknown-estimate"); ok {
		t.Fatal("an unknown estimate invented a budget")
	}
	var drafts, frozen int
	if e = tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE plan_version_id=md5('draft')::uuid), count(*) FILTER (WHERE plan_version_id=md5('base')::uuid AND amount=120.00) FROM cost_values WHERE org_id=$1 AND metric='estimate'`, org).Scan(&drafts, &frozen); e != nil {
		t.Fatal(e)
	}
	if drafts != 0 || frozen != 1 {
		t.Fatal("draft estimates left or baseline changed", drafts, frozen)
	}
}
