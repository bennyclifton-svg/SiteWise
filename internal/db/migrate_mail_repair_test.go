package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Review LOW-6: 077 repairs a database that ran either version of 075/076,
// so it must run again over its own result without error and leave the
// same columns and tables.
func TestMailRepairMigrationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	sql, err := migrationFiles.ReadFile("migrations/077_change_areas.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for i := 0; i < 2; i++ {
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='mail_messages'
AND column_name IN ('reply_to','send_tried_at','raw_sha256','send_pool')`).Scan(&n); err != nil || n != 4 {
		t.Fatalf("columns after two runs: %d %v", n, err)
	}
	var stages, old bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('mail_blob_stages') IS NOT NULL, to_regclass('mail_staged_blobs') IS NOT NULL`).Scan(&stages, &old); err != nil || !stages || old {
		t.Fatalf("staging tables: new %v old %v %v", stages, old, err)
	}
}
