package db

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

// Exercise the actual pre-015 schema, DDL, backfill and assertions in an
// isolated rollback-only schema. This never drops or rewrites public tables.
func TestWorkItemMigrationBackfillAndAtomicFailure(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			ctx := context.Background()
			cfg, err := pgx.ParseConfig(os.Getenv("SITEWISE_TEST_DATABASE_URL"))
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
			schema := fmt.Sprintf("wp20_migration_%d", time.Now().UnixNano())
			run := func(q pgx.Tx, sql string, args ...any) {
				t.Helper()
				if _, err := q.Exec(ctx, sql, args...); err != nil {
					t.Fatal(err)
				}
			}
			run(tx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
			run(tx, "SET LOCAL search_path TO "+pgx.Identifier{schema}.Sanitize()+",public")
			files, _ := fs.Glob(migrationFiles, "migrations/*.sql")
			sort.Strings(files)
			for _, name := range files {
				if strings.Compare(name, "migrations/015_work_items.sql") >= 0 {
					break
				}
				raw, _ := migrationFiles.ReadFile(name)
				run(tx, string(raw))
			}
			const org = "20000000-0000-4000-8000-000000000001"
			const site = "20000000-0000-4000-8000-000000000002"
			const project = "20000000-0000-4000-8000-000000000003"
			const part = "20000000-0000-4000-8000-000000000004"
			const actor = "20000000-0000-4000-8000-000000000005"
			run(tx, `INSERT INTO orgs(id,name) VALUES($1,'Migration')`, org)
			run(tx, `INSERT INTO users(org_id,id,email) VALUES($1,$2,'migration@example.test')`, org, actor)
			run(tx, `INSERT INTO sites(org_id,id,label) VALUES($1,$2,'Site')`, org, site)
			run(tx, `INSERT INTO projects(org_id,id,site_id,name) VALUES($1,$2,$3,'Project')`, org, project, site)
			run(tx, `INSERT INTO project_parts(org_id,id,site_id,created_by_project_id,label,kind) VALUES($1,$2,$3,$4,'Whole','whole')`, org, part, site, project)
			for key, value := range map[string]string{"hdr.work_type": "refurb", "scope.hydraulic.gas": "in", "scope.structure": "out"} {
				run(tx, `INSERT INTO profile_user_values(org_id,id,project_id,site_id,part_id,scope,key,value,user_id) VALUES($1,gen_random_uuid(),$2,$3,$4,'project',$5,$6,$7)`, org, project, site, part, key, value, actor)
			}
			if bad {
				run(tx, `UPDATE profile_user_values SET value=NULL,value_state='cleared' WHERE key='scope.structure'`)
			}
			save, err := tx.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := migrationFiles.ReadFile("migrations/015_work_items.sql")
			run(save, string(raw))
			err = backfillWorkItems(ctx, save)
			if bad {
				if err == nil {
					t.Fatal("unexpected legacy value was discarded")
				}
				if err := save.Rollback(ctx); err != nil {
					t.Fatal(err)
				}
				var count int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM profile_user_values WHERE key LIKE 'scope.%'`).Scan(&count); err != nil || count != 2 {
					t.Fatalf("rollback lost scope: %d %v", count, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := save.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM profile_user_values`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("non-scope values altered: %d %v", count, err)
			}
			var id, action, inclusion, review string
			var touched bool
			if err := tx.QueryRow(ctx, `SELECT id::text,action,inclusion,review_status,user_touched FROM work_items WHERE system_id='structure'`).Scan(&id, &action, &inclusion, &review, &touched); err != nil {
				t.Fatal(err)
			}
			if id != works.CoarseID(project, part, "structure") || action != "alter" || inclusion != "excluded" || review != "accepted_for_planning" || !touched {
				t.Fatalf("bad migrated choice: %s %s %s %s %v", id, action, inclusion, review, touched)
			}
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM work_items`).Scan(&count); err != nil || count != 2 {
				t.Fatalf("count mismatch %d %v", count, err)
			}
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM orgs`).Scan(&count); err != nil || count != 1 {
				t.Fatal("unrelated org rows changed")
			}
		})
	}
}
