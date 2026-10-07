package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
)

var ErrInvalidPackage = errors.New("invalid package")

func (s *Store) ReadPackageOverview(ctx context.Context, org, project string) ([]procurement.Package, []procurement.PackageSuggestion, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrNotFound
	}
	items, err := readPackages(ctx, tx, org, project)
	if err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(ctx, `SELECT r.key,COALESCE(r.value,''),r.band,r.origin,r.meaning,r.review_status,r.value_state FROM profile_rows r JOIN project_parts p ON p.org_id=r.org_id AND p.site_id=r.site_id AND p.id=r.part_id WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND (r.key='hdr.work_type' OR (p.kind='whole' AND (r.key='hdr.building_class' OR r.key LIKE 'det.%' OR r.key LIKE 'hdr.cond.%')))`, org, project)
	if err != nil {
		return nil, nil, err
	}
	facts := []profile.Row{}
	for rows.Next() {
		var row profile.Row
		if err := rows.Scan(&row.Key, &row.Value, &row.Band, &row.Origin, &row.Meaning, &row.ReviewStatus, &row.ValueState); err != nil {
			rows.Close()
			return nil, nil, err
		}
		facts = append(facts, row)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	class, workTypes := profile.PackageBaselineFacts(facts)
	suggestions := procurement.SuggestedPackages(s.workCatalog(), class, workTypes, profile.PackageComplexityFacts(facts), items)
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return items, suggestions, nil
}

func readPackages(ctx context.Context, q rowQuerier, org, project string) ([]procurement.Package, error) {
	rows, err := q.Query(ctx, `SELECT to_jsonb(p)||jsonb_build_object('stages',COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY ordinal,label,id) FROM package_stages s WHERE s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id AND s.retired_at IS NULL),'[]'::jsonb)) FROM packages p WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL ORDER BY title,id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []procurement.Package{}
	for rows.Next() {
		var raw []byte
		var p procurement.Package
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ReadPackages(ctx context.Context, org, project string) ([]procurement.Package, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	return readPackages(ctx, s.pool, org, project)
}

func (s *Store) CreatePackage(ctx context.Context, org, project, actor string, p procurement.Package) (procurement.Package, error) {
	if len(p.Stages) > 100 {
		return p, fmt.Errorf("%w: too many stages", ErrInvalidPackage)
	}
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return p, err
	}
	var projectExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&projectExists); err != nil {
		return p, err
	}
	if !projectExists {
		return p, ErrNotFound
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, err
	}
	p.ID, p.ProjectID, p.Version = newID(), project, 1
	p.Origin, p.ReviewStatus, p.Meaning = "user", "accepted_for_planning", "stated"
	p.SourceProposalKey, p.RetiredAt = "", nil
	p.Provenance, _ = json.Marshal(map[string]string{"actor": actor})
	_, err = tx.Exec(ctx, `INSERT INTO packages(org_id,id,project_id,kind,works_scope,discipline_id,title,novation,lifecycle_status,origin,review_status,meaning,provenance) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),NULLIF($6,''),$7,$8,$9,'user','accepted_for_planning','stated',$10::jsonb)`, org, p.ID, project, p.Kind, p.WorksScope, p.DisciplineID, p.Title, p.Novation, p.LifecycleStatus, p.Provenance)
	if err != nil {
		return p, err
	}
	if p.Stages == nil {
		p.Stages = procurement.DefaultStages(p, s.workCatalog())
	} else {
		for i := range p.Stages {
			p.Stages[i].Origin = "user"
		}
	}
	for i := range p.Stages {
		stage, err := s.insertPackageStage(ctx, tx, org, p, p.Stages[i])
		if err != nil {
			return p, err
		}
		p.Stages[i] = stage
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return p, err
	}
	return p, nil
}

func packageActor(ctx context.Context, tx pgx.Tx, org, actor string) error {
	if actor == "" {
		return ErrNotFound
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE org_id=$1::uuid AND id=$2::uuid)`, org, actor).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (s *Store) insertPackageStage(ctx context.Context, tx pgx.Tx, org string, p procurement.Package, stage procurement.Stage) (procurement.Stage, error) {
	if err := procurement.ValidateStage(stage, p, s.workCatalog()); err != nil {
		return stage, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	stage.ID, stage.ProjectID, stage.PackageID, stage.Version, stage.RetiredAt = newID(), p.ProjectID, p.ID, 1, nil
	// Caller-supplied stages are user choices; only the service creates defaults.
	if stage.Origin != "calculation" {
		stage.Origin = "user"
	}
	_, err := tx.Exec(ctx, `INSERT INTO package_stages(org_id,id,project_id,package_id,stage_id,label,ordinal,novation_phase,origin) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9)`, org, stage.ID, p.ProjectID, p.ID, stage.StageID, stage.Label, stage.Ordinal, stage.NovationPhase, stage.Origin)
	return stage, err
}

func (s *Store) finishPackagesWrite(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
		return err
	}
	revisions, err := readRevisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"project_id": project, "revision": revisions.Packages})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), org, "packages", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
