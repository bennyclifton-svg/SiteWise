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

func TestProposalMigrationIsolationAndDecisionRetention(t *testing.T) {
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
	run := func(q pgx.Tx, sql string, args ...any) {
		t.Helper()
		if _, err := q.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("wp26_migration_%d", time.Now().UnixNano())}.Sanitize()
	run(tx, "CREATE SCHEMA "+schema)
	run(tx, "SET LOCAL search_path TO "+schema+",public")
	names, _ := fs.Glob(migrationFiles, "migrations/*.sql")
	sort.Strings(names)
	for _, name := range names {
		if name > "migrations/016b_proposal_cascade.sql" {
			break
		}
		body, _ := migrationFiles.ReadFile(name)
		run(tx, string(body))
	}
	id := func(n int) string { return fmt.Sprintf("26000000-0000-4000-8000-%012d", n) }
	for _, n := range []int{1, 2} {
		run(tx, `INSERT INTO orgs(id,name) VALUES($1,'test')`, id(n))
		run(tx, `INSERT INTO users(org_id,id,email) VALUES($1,$2,'test@example.test')`, id(n), id(n+2))
	}
	for i := 0; i < 3; i++ {
		org := id(1)
		if i == 1 {
			org = id(2)
		}
		run(tx, `INSERT INTO sites(org_id,id,label) VALUES($1,$2,'Site')`, org, id(5+i))
		run(tx, `INSERT INTO projects(org_id,id,site_id,name) VALUES($1,$2,$3,'Project')`, org, id(8+i), id(5+i))
		run(tx, `INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind) VALUES($1,$2,$3,$4,'Whole project','whole')`, org, id(11+i), id(5+i), id(8+i))
		run(tx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status) VALUES($1,$2,$3,$4,$5,'structure','alter','included','Work','user','accepted_for_planning')`, org, id(14+i), id(8+i), id(5+i), id(11+i))
	}
	key := "ic.test||structure|" + id(11) + "|0"
	projection := `INSERT INTO proposals(org_id,project_id,site_id,key,record_kind,record_id,proposal_index,target_system_id,target_part_id,kind,label,reason,specificity,rank,draft,unaccepted_triggers,inputs_fingerprint,knowledge_version,state) VALUES($1,$2,$3,$4,'ic','ic.test',0,'structure',$5,'investigation','Check','{}',3,1,true,false,repeat('a',64),'knowledge','open')`
	run(tx, projection, id(1), id(8), id(5), key, id(11))
	bad := func(name, sql string, args ...any) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
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
			want := "23503"
			if name == "accepted-missing-type" || name == "dismissed-created-record" {
				want = "23514"
			}
			if !errors.As(err, &pgErr) || pgErr.Code != want {
				t.Fatalf("expected constraint %s, got %v", want, err)
			}
		})
	}
	bad("foreign-project", projection, id(2), id(8), id(5), key, id(11))
	bad("wrong-site", projection, id(1), id(10), id(5), key, id(11))
	bad("wrong-part", projection, id(1), id(8), id(5), "ic.test||structure|"+id(13)+"|0", id(13))
	trigger := `INSERT INTO proposal_triggers(org_id,project_id,proposal_key,work_item_id) VALUES($1,$2,$3,$4)`
	bad("foreign-trigger", trigger, id(1), id(8), key, id(15))
	bad("other-project-trigger", trigger, id(1), id(8), key, id(16))
	run(tx, trigger, id(1), id(8), key, id(14))
	decision := `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id) VALUES($1,$2,$3,$4,'ic.test',$5,$6,repeat('a',64),$7,$8,$9)`
	bad("foreign-actor", decision, id(1), id(20), id(8), key, id(14), "dismissed", id(4), nil, nil)
	bad("other-project-created-record", decision, id(1), id(20), id(8), key, id(14), "accepted", id(3), "work_item", id(16))
	bad("accepted-missing-type", decision, id(1), id(20), id(8), key, id(14), "accepted", id(3), nil, id(14))
	bad("dismissed-created-record", decision, id(1), id(20), id(8), key, id(14), "dismissed", id(3), "work_item", id(14))
	run(tx, decision, id(1), id(20), id(8), key, id(14), "dismissed", id(3), nil, nil)
	run(tx, `DELETE FROM proposals WHERE org_id=$1 AND project_id=$2`, id(1), id(8))
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM proposal_decisions WHERE org_id=$1 AND project_id=$2`, id(1), id(8)).Scan(&count); err != nil || count != 1 {
		t.Fatalf("decision lost: %d %v", count, err)
	}
	bad("referenced-work-delete", `DELETE FROM work_items WHERE org_id=$1 AND id=$2`, id(1), id(14))
	bad("referenced-actor-delete", `DELETE FROM users WHERE org_id=$1 AND id=$2`, id(1), id(3))
	// Recreate the projection and its trigger so deletion exercises all three
	// relationships, including the retained decision's independent lifetime.
	run(tx, projection, id(1), id(8), id(5), key, id(11))
	run(tx, trigger, id(1), id(8), key, id(14))
	run(tx, `DELETE FROM orgs WHERE id=$1`, id(1))
	run(tx, "SET CONSTRAINTS ALL IMMEDIATE")
	for _, table := range []string{"proposals", "proposal_triggers", "proposal_decisions"} {
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE org_id=$1", id(1)).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s survived organisation deletion: %d %v", table, count, err)
		}
	}
}
