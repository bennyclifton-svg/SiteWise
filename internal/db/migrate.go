package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrationLockID is the transaction advisory lock that serialises migration.
const migrationLockID int64 = 741932745

// Migrate applies embedded SQL migrations in one transaction under an advisory lock.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}

	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimPrefix(name, "migrations/")
		var applied bool
		err := tx.QueryRow(ctx, `SELECT true FROM schema_migrations WHERE version = $1`, version).Scan(&applied)
		if err == nil && applied {
			continue
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		body, err := migrationFiles.ReadFile(name)
		if err != nil {
			return err
		}
		// With no bind arguments pgx uses PostgreSQL's simple protocol. Let the
		// database parse function bodies, quoted semicolons and comments.
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("%s: %w", version, err)
		}
		if version == "015_work_items.sql" {
			if err := backfillWorkItems(ctx, tx); err != nil {
				return fmt.Errorf("%s: %w", version, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
