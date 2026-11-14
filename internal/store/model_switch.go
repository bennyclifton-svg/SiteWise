package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ModelEnabled reports whether the org has agreed that its project text
// may be sent to the model provider. One switch covers every model use
// (docs/design/2026-11-14-tender-comparison.md, section 17).
func (s *Store) ModelEnabled(ctx context.Context, org string) (bool, error) {
	at, err := s.ModelEnabledAt(ctx, org)
	return at != nil, err
}

// ModelEnabledAt is when the org's owner switched the model on, or nil.
func (s *Store) ModelEnabledAt(ctx context.Context, org string) (*time.Time, error) {
	var at *time.Time
	err := s.pool.QueryRow(ctx, `SELECT model_enabled_at FROM orgs WHERE id::text = $1`, org).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return at, err
}

// SetModelEnabled records the owner's choice. The caller has checked that
// actor is the org's owner.
func (s *Store) SetModelEnabled(ctx context.Context, org, actor string, on bool, now time.Time) error {
	sql := `UPDATE orgs SET model_enabled_at = NULL, model_enabled_by = NULL WHERE id::text = $1`
	args := []any{org}
	if on {
		// Switching on again keeps the first date: it is when the org agreed.
		sql = `UPDATE orgs SET model_enabled_at = COALESCE(model_enabled_at, $2), model_enabled_by = COALESCE(model_enabled_by, $3::uuid) WHERE id::text = $1`
		args = append(args, now, actor)
	}
	tag, err := s.pool.Exec(ctx, sql, args...)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
