package store

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

func (s *Store) ReadGaps(ctx context.Context, org, project string) ([]procurement.Gap, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	items, err := readWorkItems(ctx, tx, org, project)
	if err != nil {
		return nil, err
	}
	packages, err := readPackages(ctx, tx, org, project)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT to_jsonb(i) FROM package_scope_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL AND inclusion='included' AND work_item_id IS NOT NULL`, org, project)
	if err != nil {
		return nil, err
	}
	assignments := []procurement.Assignment{}
	for rows.Next() {
		var raw []byte
		var a procurement.Assignment
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			rows.Close()
			return nil, err
		}
		assignments = append(assignments, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	gaps, err := procurement.CheckGaps(items, packages, assignments, s.workCatalog())
	if err != nil {
		return nil, err
	}
	return gaps, tx.Commit(ctx)
}
