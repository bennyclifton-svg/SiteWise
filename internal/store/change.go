package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Change marks every transaction begun with its context, so the capture
// triggers record the rows it writes under one undoable change
// (docs/plans/2026-11-14-change-log-undo-operations.md, tasks 1 to 4).
type Change struct {
	ID, OrgID, ProjectID, Actor string
	Via                         string // "chat" or "screen"
	TurnID                      string // groups one chat turn's changes; may be empty
	Area                        string // "cost_plan", "profile" or "procurement"
}

type changeKey struct{}

// WithChange returns a context whose store writes are captured under c.
func WithChange(ctx context.Context, c Change) context.Context {
	return context.WithValue(ctx, changeKey{}, c)
}

// NewChangeID is a fresh id for a Change.
func NewChangeID() string { return newID() }

// markedPool is the pool every store function uses. Begin sets the
// transaction-local mark when the context carries a change. A write made
// outside a transaction is not marked and is not captured.
type markedPool struct{ *pgxpool.Pool }

func (p markedPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.BeginTx(ctx, pgx.TxOptions{})
}

func (p markedPool) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, err := p.Pool.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	c, ok := ctx.Value(changeKey{}).(Change)
	if !ok || o.AccessMode == pgx.ReadOnly {
		return tx, nil
	}
	mark, _ := json.Marshal(map[string]string{"id": c.ID, "org": c.OrgID, "project": c.ProjectID,
		"actor": c.Actor, "via": c.Via, "turn": c.TurnID, "area": c.Area})
	if _, err := tx.Exec(ctx, `SELECT set_config('sitewise.change',$1,true)`, string(mark)); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
