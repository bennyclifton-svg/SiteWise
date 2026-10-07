package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This is an upgrade rehearsal, distinct from restoring an already upgraded
// database. Both databases are newly created and exclusively owned by this test.
func TestPopulatedPreM2MigrationRehearsal(t *testing.T) {
	bin := os.Getenv("SITEWISE_RESTORE_PG_BIN")
	if bin == "" {
		t.Skip("set SITEWISE_RESTORE_PG_BIN for populated upgrade rehearsal")
	}
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated sitewise_test connection required", err)
	}
	base, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	adminURL := *base
	adminURL.Path = "/postgres"
	admin, err := pgx.Connect(ctx, adminURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(context.Background())
	create := func(suffix string) string {
		t.Helper()
		name := fmt.Sprintf("sitewise_upgrade_%d_%s", time.Now().UnixNano(), suffix)
		quoted := pgx.Identifier{name}.Sanitize()
		if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
			t.Fatal(err)
		}
		// Register cleanup only after CREATE succeeds. Never drop a supplied name.
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			connection, e := pgx.Connect(cleanup, adminURL.String())
			if e != nil {
				t.Error(e)
				return
			}
			defer connection.Close(cleanup)
			if _, e = connection.Exec(cleanup, "DROP DATABASE "+quoted+" WITH (FORCE)"); e != nil {
				t.Error("owned scratch cleanup", e)
			}
		})
		u := *base
		u.Path = "/" + name
		return u.String()
	}
	sourceURL, targetURL := create("before"), create("after")
	source, err := pgx.Connect(ctx, sourceURL)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close(context.Background())
	tx, err := source.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `CREATE TABLE schema_migrations(version text PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	baselineMigrations := 0
	for _, name := range names {
		if name >= "migrations/021" {
			break
		}
		raw, e := migrationFiles.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, string(raw)); e != nil {
			t.Fatalf("baseline %s: %v", name, e)
		}
		if name == "migrations/015_work_items.sql" {
			if e = backfillWorkItems(ctx, tx); e != nil {
				t.Fatal(e)
			}
		}
		if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, strings.TrimPrefix(name, "migrations/")); e != nil {
			t.Fatal(e)
		}
		baselineMigrations++
	}
	fixture, err := os.ReadFile("testdata/pre_m2_upgrade_fixture.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(fixture)); err != nil {
		t.Fatal("populate old schema", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	tables, err := upgradeTables(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	before, err := upgradeDigests(ctx, source, tables)
	if err != nil {
		t.Fatal(err)
	}
	dump := filepath.Join(t.TempDir(), "pre-m2.dump")
	command := func(tool string, args ...string) {
		t.Helper()
		exe := filepath.Join(bin, tool)
		if _, e := os.Stat(exe + ".exe"); e == nil {
			exe += ".exe"
		}
		if out, e := exec.CommandContext(ctx, exe, args...).CombinedOutput(); e != nil {
			t.Fatalf("%s: %v %s", tool, e, out)
		}
	}
	command("pg_dump", "--format=custom", "--file", dump, "--dbname", sourceURL)
	command("pg_restore", "--exit-on-error", "--no-owner", "--dbname", targetURL, dump)
	target, err := pgx.Connect(ctx, targetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close(context.Background())
	restored, err := upgradeDigests(ctx, target, tables)
	if err != nil || !reflect.DeepEqual(before, restored) {
		t.Fatal("pre-upgrade backup restore changed rows", err)
	}
	pool, err := pgxpool.New(ctx, targetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = Migrate(ctx, pool); err != nil {
		t.Fatal("upgrade restored old schema", err)
	}
	command("psql", "-X", "--quiet", "--tuples-only", "--csv", "--file", "migrations/checks/pre_m2_preservation.sql", "--dbname", targetURL)
	after, err := upgradeDigests(ctx, target, tables)
	if err != nil {
		t.Fatal(err)
	}
	for table, expected := range before {
		if after[table] != expected {
			t.Errorf("old data changed in %s: before=%+v after=%+v", table, expected, after[table])
		}
	}
	var valid bool
	err = target.QueryRow(ctx, `SELECT
 (SELECT count(*)=3 AND bool_and(layout_change='unknown') FROM work_items) AND
 (SELECT count(*)=3 AND bool_and(ps.document_id=p.document_id) FROM passage_sources ps JOIN passages p ON p.org_id=ps.org_id AND p.id=ps.passage_id) AND
 (SELECT count(*)=3 AND bool_and(c.document_id=p.document_id) FROM passage_calls c JOIN passages p ON p.org_id=c.org_id AND p.id=c.passage_id) AND
 (SELECT count(*)=3 AND bool_and(projection_hash='') FROM profile_rows) AND
 (SELECT count(*)=3 AND bool_and(projection_hash='') FROM proposals) AND
 (SELECT bool_and(rank=CASE severity WHEN 'life-safety' THEN 1 ELSE 2 END) FROM ranked_proposals WHERE org_id=md5('upgrade-org1')::uuid) AND
 (SELECT rank=1 FROM ranked_proposals WHERE org_id=md5('upgrade-org2')::uuid) AND
 NOT EXISTS(SELECT 1 FROM cost_plan_versions) AND
 NOT EXISTS(SELECT 1 FROM pg_constraint WHERE connamespace='public'::regnamespace AND NOT convalidated) AND
 (SELECT count(*)=$1 FROM schema_migrations)`, len(names)).Scan(&valid)
	if err != nil || !valid {
		t.Fatal("new defaults/backfill/rank/constraints/migration inventory failed", valid, err)
	}
	if err = Migrate(ctx, pool); err != nil {
		t.Fatal("migration rerun", err)
	}
	again, err := upgradeDigests(ctx, target, tables)
	if err != nil || !reflect.DeepEqual(after, again) {
		t.Fatal("idempotent migration changed data", err)
	}
	t.Logf("synthetic old-schema backup/restore/upgrade: %d baseline migrations, %d new migrations, %d old tables retain counts and normalized SHA-256; source document/layout/hash/rank backfills and all constraints validated", baselineMigrations, len(names)-baselineMigrations, len(tables))
}

type upgradeDigest struct {
	Rows   int64
	SHA256 string
}

func upgradeTables(ctx context.Context, c *pgx.Conn) ([]string, error) {
	r, err := c.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations' ORDER BY tablename`)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var name string
		if err = r.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, r.Err()
}

// Normalize only declared migration changes. Every other old column, including
// provenance, timestamps, money strings and explicit nulls, participates.
func upgradeDigests(ctx context.Context, c *pgx.Conn, tables []string) (map[string]upgradeDigest, error) {
	if _, err := c.Exec(ctx, `SET TIME ZONE 'UTC'`); err != nil {
		return nil, err
	}
	out := map[string]upgradeDigest{}
	for _, table := range tables {
		exclude := []string{}
		switch table {
		case "work_items":
			exclude = []string{"layout_change"}
		case "proposals":
			exclude = []string{"rank", "projection_hash"}
		case "profile_rows":
			exclude = []string{"projection_hash"}
		case "passage_sources", "passage_calls":
			exclude = []string{"document_id"}
		}
		rows, err := c.Query(ctx, `SELECT content FROM (SELECT (to_jsonb(r)-$1::text[])::text AS content FROM `+pgx.Identifier{"public", table}.Sanitize()+` r) saved_rows ORDER BY content COLLATE "C"`, exclude)
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		var count int64
		for rows.Next() {
			var content string
			if err = rows.Scan(&content); err != nil {
				rows.Close()
				return nil, err
			}
			hash.Write([]byte(content))
			hash.Write([]byte{'\n'})
			count++
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		out[table] = upgradeDigest{count, hex.EncodeToString(hash.Sum(nil))}
	}
	return out, nil
}
