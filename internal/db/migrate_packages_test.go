package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPackageMigrationIsolationAndConstraints(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("dedicated test database required")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required")
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
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
	schema := pgx.Identifier{fmt.Sprintf("wp30_%d", time.Now().UnixNano())}.Sanitize()
	run("CREATE SCHEMA " + schema)
	run("SET LOCAL search_path TO " + schema + ",public")
	names, _ := fs.Glob(migrationFiles, "migrations/*.sql")
	sort.Strings(names)
	for _, name := range names {
		if name > "migrations/020_reports.sql" {
			break
		}
		body, _ := migrationFiles.ReadFile(name)
		run(string(body))
	}
	id := func(n int) string { return fmt.Sprintf("30000000-0000-4000-8000-%012d", n) }
	for i := 1; i <= 2; i++ {
		run(`INSERT INTO orgs(id,name) VALUES($1,'Packages')`, id(i))
		run(`INSERT INTO sites(org_id,id,label) VALUES($1,$2,'Site')`, id(i), id(i+2))
		run(`INSERT INTO projects(org_id,id,site_id,name) VALUES($1,$2,$3,'Project')`, id(i), id(i+4), id(i+2))
	}
	insert := `INSERT INTO packages(org_id,id,project_id,kind,works_scope,title,novation,lifecycle_status,origin,review_status) VALUES($1,$2,$3,$4,$5,'Package',$6,'proposed','calculation','proposed')`
	run(insert, id(1), id(7), id(5), "services", nil, true)
	bad := func(code, sql string, args ...any) {
		t.Helper()
		save, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer save.Rollback(ctx)
		_, err = save.Exec(ctx, sql, args...)
		if err == nil {
			_, err = save.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != code {
			t.Fatalf("want %s, got %v", code, err)
		}
	}
	bad("23503", insert, id(2), id(8), id(5), "services", nil, false)
	bad("23514", insert, id(1), id(8), id(5), "works", "trade", true)
	bad("23514", insert, id(1), id(8), id(5), "works", nil, false)
	bad("23514", insert, id(1), id(8), id(5), "supply", "trade", false)
	bad("23514", `UPDATE packages SET review_status='verified' WHERE id=$1`, id(7))
	stage := `INSERT INTO package_stages(org_id,id,project_id,package_id,stage_id,label,ordinal,novation_phase,origin) VALUES($1,$2,$3,$4,'design','Design',0,'pre','calculation')`
	run(stage, id(1), id(9), id(5), id(7))
	bad("23503", stage, id(2), id(10), id(6), id(7))
	bad("23503", `UPDATE package_stages SET project_id=$1 WHERE org_id=$2 AND id=$3`, id(6), id(1), id(9))
	bad("23505", stage, id(1), id(10), id(5), id(7))
	run(`INSERT INTO users(org_id,id,email) VALUES($1,$2,'package-reviewer@example.test')`, id(1), id(11))
	run(insert, id(2), id(8), id(6), "services", nil, false)
	decision := `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id) VALUES($1,$2,$3,'ic.package||||0','ic.package','accepted',repeat('a',64),$4,$5,$6)`
	bad("23503", decision, id(1), id(12), id(5), id(11), "package", id(8))
	bad("23503", decision, id(1), id(12), id(5), id(11), "work_item", id(7))
	run(decision, id(1), id(12), id(5), id(11), "package", id(7))
	run(`INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind) VALUES($1,$2,$3,$4,'Whole','whole')`, id(1), id(13), id(3), id(5))
	run(`INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status) VALUES($1,$2,$3,$4,$5,'sys.fire','retain','included','Retain fire system','user','accepted_for_planning')`, id(1), id(14), id(5), id(3), id(13))
	scope := `INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,user_text,stage_id,inclusion,origin,review_status) VALUES($1,$2,$3,$4,'responsibility',$5,$6,'Maintain the system',$7,'included','user','accepted_for_planning')`
	run(scope, id(1), id(15), id(5), id(7), id(14), "maintain_operation", id(9))
	bad("23505", scope, id(1), id(16), id(5), id(7), id(14), "maintain_operation", id(9))
	bad("23503", scope, id(2), id(16), id(6), id(8), id(14), "maintain_operation", nil)
	bad("23514", scope, id(1), id(16), id(5), id(7), nil, "protect", nil)
	bad("23514", `UPDATE package_scope_items SET clause_id='cl.example',clause_version=1 WHERE id=$1`, id(15))
	bad("23514", `UPDATE package_scope_items SET clause_version=1 WHERE id=$1`, id(15))
	bad("23514", `UPDATE package_scope_items SET review_status='verified' WHERE id=$1`, id(15))
	run(insert, id(1), id(17), id(5), "works", "trade", false)
	bad("23503", scope, id(1), id(16), id(5), id(17), id(14), "protect", id(9))
	// The same role across packages is an overlap for the gap checker, not a
	// database error. Excluded or retired rows can also preserve prior wording.
	run(scope, id(1), id(16), id(5), id(17), id(14), "maintain_operation", nil)
	bad("23503", `DELETE FROM work_items WHERE org_id=$1 AND id=$2`, id(1), id(14))
	bad("23503", `DELETE FROM package_stages WHERE org_id=$1 AND id=$2`, id(1), id(9))
	delivery := `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,status,package_id,work_item_id,stage_id,origin,review_status) VALUES($1,$2,$3,'approval','Authority approval','not_submitted',$4,$5,$6,'user','accepted_for_planning')`
	run(delivery, id(1), id(20), id(5), id(7), id(14), id(9))
	bad("23503", delivery, id(2), id(21), id(6), id(7), nil, nil)
	bad("23503", delivery, id(1), id(21), id(5), id(17), nil, id(9))
	bad("23514", delivery, id(1), id(21), id(5), nil, nil, id(9))
	bad("23514", `UPDATE project_delivery_items SET review_status='verified' WHERE id=$1`, id(20))
	run(delivery, id(1), id(21), id(5), nil, nil, nil)
	dependency := `INSERT INTO delivery_dependencies(org_id,project_id,predecessor_id,successor_id) VALUES($1,$2,$3,$4)`
	run(dependency, id(1), id(5), id(20), id(21))
	bad("23514", dependency, id(1), id(5), id(20), id(20))
	bad("23503", dependency, id(2), id(6), id(20), id(21))
	report := `INSERT INTO reports(org_id,id,project_id,kind,package_id,title) VALUES($1,$2,$3,'rfp',$4,'Draft RFP')`
	run(report, id(1), id(30), id(5), id(7))
	bad("23503", report, id(2), id(31), id(6), id(7))
	versionSQL := `INSERT INTO report_versions(org_id,id,project_id,report_id,number,status,reporting_date,source_revisions,template_id,template_version,sections) VALUES($1,$2,$3,$4,1,'draft','2026-10-05','{}','tpl.rfp-capex',1,'[]')`
	run(versionSQL, id(1), id(31), id(5), id(30))
	bad("23503", versionSQL, id(2), id(32), id(6), id(30))
	run(`UPDATE reports SET current_draft_version_id=$2 WHERE id=$1`, id(30), id(31))
	run(`INSERT INTO report_edits(org_id,report_version_id,target_id,text,base_content_sha256,user_id) VALUES($1,$2,'brief','Protected; wording',repeat('a',64),$3)`, id(1), id(31), id(11))
	run(`INSERT INTO report_references(org_id,report_version_id,citation_id,label,anchor_id,basis) VALUES($1,$2,'U1','U','brief','{}')`, id(1), id(31))
	bad("23514", `UPDATE report_versions SET status='issued' WHERE id=$1`, id(31))
	run("SAVEPOINT report_issue_test")
	run(`UPDATE report_versions SET status='issued',snapshot='{}',snapshot_sha256=repeat('a',64),issued_at=now(),issued_by=$2 WHERE id=$1`, id(31), id(11))
	bad("23514", `UPDATE report_versions SET sections='[]' WHERE id=$1`, id(31))
	bad("23514", `DELETE FROM report_versions WHERE id=$1`, id(31))
	bad("23514", `UPDATE report_edits SET text='Changed' WHERE report_version_id=$1`, id(31))
	bad("23514", `DELETE FROM report_references WHERE report_version_id=$1`, id(31))
	bad("23514", `INSERT INTO report_references(org_id,report_version_id,citation_id,label,anchor_id,basis) VALUES($1,$2,'U2','U','brief','{}')`, id(1), id(31))
	run("ROLLBACK TO SAVEPOINT report_issue_test")
	run(`DELETE FROM orgs WHERE id=$1`, id(1))
	run("SET CONSTRAINTS ALL IMMEDIATE")
	for _, table := range []string{"packages", "package_stages", "package_scope_items", "project_delivery_items", "delivery_dependencies", "reports", "report_versions", "report_edits", "report_references"} {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE org_id=$1", id(1)).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s survived delete: %d %v", table, n, err)
		}
	}
}
