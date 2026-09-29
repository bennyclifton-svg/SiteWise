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
		for _, stmt := range splitSQL(string(body)) {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("%s: %w", version, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func splitSQL(sql string) []string {
	var lines []string
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		lines = append(lines, line)
	}
	var out []string
	for _, part := range strings.Split(strings.Join(lines, "\n"), ";") {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
