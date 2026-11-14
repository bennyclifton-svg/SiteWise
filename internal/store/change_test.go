package store_test

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"sitewise/internal/store"
)

// The registry of captured tables must name exactly the tables that carry
// the capture trigger, with their primary keys and the trigger's arguments.
func TestCapturedTablesMatchDatabase(t *testing.T) {
	ctx := context.Background()
	workStore(t)
	pool := rawPool(t)
	rows, err := pool.Query(ctx, `SELECT c.relname, t.tgargs,
 (SELECT array_agg(a.attname ORDER BY k.ord) FROM pg_index i
  CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY k(attnum, ord)
  JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum WHERE i.indrelid = c.oid AND i.indisprimary)
FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE t.tgname = 'capture_change' AND n.nspname = current_schema()`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	registry := store.CapturedTables()
	for rows.Next() {
		var table string
		var args []byte
		var key []string
		if err := rows.Scan(&table, &args, &key); err != nil {
			t.Fatal(err)
		}
		seen[table] = true
		reg, ok := registry[table]
		if !ok {
			t.Errorf("%s carries the trigger but is not registered", table)
			continue
		}
		var ignore []string
		for _, a := range strings.Split(string(args), "\x00") {
			if a != "" {
				ignore = append(ignore, a)
			}
		}
		sort.Strings(ignore)
		want := append([]string{}, reg.Ignore...)
		sort.Strings(want)
		if !reflect.DeepEqual(key, reg.Key) {
			t.Errorf("%s: primary key %v, registered %v", table, key, reg.Key)
		}
		if len(ignore)+len(want) > 0 && !reflect.DeepEqual(ignore, want) {
			t.Errorf("%s: trigger ignores %v, registered %v", table, ignore, want)
		}
	}
	for table := range registry {
		if !seen[table] {
			t.Errorf("%s is registered but carries no trigger", table)
		}
	}
}

// A write is captured only inside a transaction marked with a change, and a
// change cannot write into another project.
func TestCaptureRecordsMarkedWritesOnly(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 51)
	commit(t, s, sub, tenderReadingOf(t, s, sub, rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")))
	pool := rawPool(t)
	count := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM project_change_rows r JOIN project_changes c ON c.org_id=r.org_id AND c.id=r.change_id WHERE c.org_id=$1 AND c.project_id=$2`, orgA, projectA).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if count() != 0 {
		t.Fatal("an unmarked reading was captured")
	}
	mark := func(project string) string {
		return `{"id":"11111111-1111-4111-8111-00000000c0de","org":"` + orgA + `","project":"` + project + `","actor":"` + userA + `","via":"screen","turn":"","area":"procurement"}`
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('sitewise.change', $1, true)`, mark(projectA)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE tender_lines SET label='Advice, renamed' WHERE org_id=$1 AND submission_id=$2 AND label='Advice'`, orgA, sub); err != nil {
		t.Fatal(err)
	}
	// A bump of an ignored column alone is not a change.
	if _, err := tx.Exec(ctx, `UPDATE tender_lines SET updated_at=now() WHERE org_id=$1 AND submission_id=$2`, orgA, sub); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if count() != 1 {
		t.Fatalf("%d rows captured, want the one label change", count())
	}
	tx, _ = pool.Begin(ctx)
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('sitewise.change', $1, true)`, mark(projectB)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE tender_lines SET label='Crossed' WHERE org_id=$1 AND submission_id=$2`, orgA, sub); err == nil || !strings.Contains(err.Error(), "23514") {
		t.Fatalf("a change wrote into another project: %v", err)
	}
}
