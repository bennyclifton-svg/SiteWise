package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/delivery"
	"strings"
)

type DeliveryDependencyInput struct {
	delivery.Dependency
	Version int64 `json:"version"`
}

func readDependencies(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, org, project string) ([]delivery.Dependency, error) {
	rows, err := q.Query(ctx, `SELECT predecessor_id::text,successor_id::text,type,lag_days FROM delivery_dependencies WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY predecessor_id,successor_id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []delivery.Dependency{}
	for rows.Next() {
		var e delivery.Dependency
		if err = rows.Scan(&e.PredecessorID, &e.SuccessorID, &e.Type, &e.LagDays); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) ReadDeliveryDependencies(ctx context.Context, org, project string) ([]delivery.Dependency, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	return readDependencies(ctx, s.pool, org, project)
}
func (s *Store) WriteDeliveryDependency(ctx context.Context, org, project, actor string, in DeliveryDependencyInput, remove bool) error {
	var id pgtype.UUID
	if id.Scan(in.PredecessorID) != nil || id.Scan(in.SuccessorID) != nil {
		return ErrInvalidDelivery
	}
	if in.Version < 1 || (in.Type != "" && in.Type != "finish_to_start") || in.LagDays < -36500 || in.LagDays > 36500 {
		return ErrInvalidDelivery
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err = packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	for _, id := range []string{in.PredecessorID, in.SuccessorID} {
		var version int64
		err = tx.QueryRow(ctx, `SELECT version FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL`, org, project, id).Scan(&version)
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if id == in.SuccessorID && version != in.Version {
			return ErrVersionConflict
		}
	}
	if remove {
		tag, e := tx.Exec(ctx, `DELETE FROM delivery_dependencies WHERE org_id=$1::uuid AND project_id=$2::uuid AND predecessor_id=$3::uuid AND successor_id=$4::uuid`, org, project, in.PredecessorID, in.SuccessorID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	} else {
		edges, e := readDependencies(ctx, tx, org, project)
		if e != nil {
			return e
		}
		if path := delivery.Cycle(edges, in.PredecessorID, in.SuccessorID); path != nil {
			return fmt.Errorf("%w: dependency cycle %s", ErrInvalidDelivery, strings.Join(path, " -> "))
		}
		_, err = tx.Exec(ctx, `INSERT INTO delivery_dependencies(org_id,project_id,predecessor_id,successor_id,type,lag_days) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'finish_to_start',$5) ON CONFLICT(org_id,project_id,predecessor_id,successor_id) DO UPDATE SET lag_days=EXCLUDED.lag_days`, org, project, in.PredecessorID, in.SuccessorID, in.LagDays)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE project_delivery_items SET version=version+1,provenance=provenance||jsonb_build_object('last_edited_by',$4::text,'last_edited_at',now()) WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, in.SuccessorID, actor)
	if err != nil {
		return err
	}
	return s.finishDeliveryWrite(ctx, tx, org, project)
}
