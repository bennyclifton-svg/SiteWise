package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

func readPackage(ctx context.Context, tx pgx.Tx, org, project, id string) (procurement.Package, error) {
	return readPackageRecord(ctx, tx, org, project, id, false)
}

func readPackageRecord(ctx context.Context, tx pgx.Tx, org, project, id string, includeRetired bool) (procurement.Package, error) {
	var raw []byte
	var p procurement.Package
	err := tx.QueryRow(ctx, `SELECT to_jsonb(p)||jsonb_build_object('stages',COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY ordinal,label,id) FROM package_stages s WHERE s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id AND s.retired_at IS NULL),'[]'::jsonb)) FROM packages p WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND ($4 OR retired_at IS NULL)`, org, project, id, includeRetired).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(raw, &p)
	return p, err
}

func (s *Store) PatchPackage(ctx context.Context, org, project, id, actor string, patch procurement.PackagePatch) (procurement.Package, error) {
	var p procurement.Package
	if patch.Version < 1 {
		return p, ErrInvalidPackage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return p, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, err
	}
	p, err = readPackage(ctx, tx, org, project, id)
	if err != nil {
		return p, err
	}
	if p.Version != patch.Version {
		return p, ErrVersionConflict
	}
	patch.Apply(&p)
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	for _, stage := range p.Stages {
		if err := procurement.ValidateStage(stage, p, s.workCatalog()); err != nil {
			return p, fmt.Errorf("%w: change stage phases before changing novation: %v", ErrInvalidPackage, err)
		}
	}
	retire := patch.Retired != nil && *patch.Retired
	var blocked bool
	if retire {
		used, err := liveCostReference(ctx, tx, org, project, "package_id", id)
		if err != nil {
			return p, err
		}
		if used {
			return p, fmt.Errorf("%w: reassign live costs before retiring this package", ErrInvalidPackage)
		}
		used, err = openDeliveryReference(ctx, tx, org, project, "package_id", id)
		if err != nil {
			return p, err
		}
		if used {
			return p, fmt.Errorf("%w: resolve open delivery records before retiring this package", ErrInvalidPackage)
		}
	}
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND retired_at IS NULL AND ($4 OR ($5='supply' AND role IS NOT NULL AND role<>'supply')))`, org, project, id, retire, p.Kind).Scan(&blocked); err != nil {
		return p, err
	}
	if blocked {
		return p, fmt.Errorf("%w: retire or reassign package scope before changing this package", ErrInvalidPackage)
	}
	var raw []byte
	err = tx.QueryRow(ctx, `UPDATE packages SET kind=$4,works_scope=NULLIF($5,''),discipline_id=NULLIF($6,''),title=$7,novation=$8,lifecycle_status=$9,version=version+1,origin='user',review_status='accepted_for_planning',verified_by=NULL,verified_at=NULL,verification_basis=NULL,provenance=provenance||jsonb_build_object('last_edited_by',$10::text),retired_at=CASE WHEN $11 THEN now() ELSE retired_at END WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid RETURNING to_jsonb(packages)`, org, project, id, p.Kind, p.WorksScope, p.DisciplineID, p.Title, p.Novation, p.LifecycleStatus, actor, retire).Scan(&raw)
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(raw, &p)
	if err != nil {
		return p, err
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return p, err
	}
	return p, nil
}

func (s *Store) CreatePackageStage(ctx context.Context, org, project, id, actor string, stage procurement.Stage) (procurement.Stage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return stage, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return stage, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return stage, err
	}
	p, err := readPackage(ctx, tx, org, project, id)
	if err != nil {
		return stage, err
	}
	if len(p.Stages) >= 100 {
		return stage, fmt.Errorf("%w: too many stages", ErrInvalidPackage)
	}
	stage.Origin = "user"
	stage, err = s.insertPackageStage(ctx, tx, org, p, stage)
	if err != nil {
		return stage, err
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return stage, err
	}
	return stage, nil
}

func (s *Store) PatchPackageStage(ctx context.Context, org, project, id, stageID, actor string, patch procurement.StagePatch) (procurement.Stage, error) {
	var stage procurement.Stage
	if patch.Version < 1 {
		return stage, ErrInvalidPackage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return stage, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return stage, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return stage, err
	}
	p, err := readPackage(ctx, tx, org, project, id)
	if err != nil {
		return stage, err
	}
	for _, existing := range p.Stages {
		if existing.ID == stageID {
			stage = existing
			break
		}
	}
	if stage.ID == "" {
		return stage, ErrNotFound
	}
	if stage.Version != patch.Version {
		return stage, ErrVersionConflict
	}
	patch.Apply(&stage)
	if err := procurement.ValidateStage(stage, p, s.workCatalog()); err != nil {
		return stage, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	retire := patch.Retired != nil && *patch.Retired
	if retire {
		used, err := liveCostReference(ctx, tx, org, project, "package_stage_id", stageID)
		if err != nil {
			return stage, err
		}
		if used {
			return stage, fmt.Errorf("%w: reassign live costs before retiring this stage", ErrInvalidPackage)
		}
		used, err = openDeliveryReference(ctx, tx, org, project, "stage_id", stageID)
		if err != nil {
			return stage, err
		}
		if used {
			return stage, fmt.Errorf("%w: resolve open delivery records before retiring this stage", ErrInvalidPackage)
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND stage_id=$4::uuid AND retired_at IS NULL)`, org, project, id, stageID).Scan(&used); err != nil {
			return stage, err
		}
		if used {
			return stage, fmt.Errorf("%w: reassign scope before retiring this stage", ErrInvalidPackage)
		}
	}
	var raw []byte
	err = tx.QueryRow(ctx, `UPDATE package_stages SET stage_id=$5,label=$6,ordinal=$7,novation_phase=$8,origin='user',version=version+1,retired_at=CASE WHEN $9 THEN now() ELSE retired_at END WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND id=$4::uuid RETURNING to_jsonb(package_stages)`, org, project, id, stageID, stage.StageID, stage.Label, stage.Ordinal, stage.NovationPhase, retire).Scan(&raw)
	if err != nil {
		return stage, err
	}
	if err := json.Unmarshal(raw, &stage); err != nil {
		return stage, err
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return stage, err
	}
	return stage, nil
}
