package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

// AcceptProposedPackage turns one proposal on the Packages tab into records.
// The proposal is recomputed under the project lock from the key alone; client
// content is never trusted. Accepting the head contract again adds only the
// lines still missing, so a retry cannot duplicate responsibilities.
func (s *Store) AcceptProposedPackage(ctx context.Context, org, project, actor, key string) (procurement.Package, error) {
	var p procurement.Package
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return p, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return p, err
	}
	if !exists {
		return p, ErrNotFound
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, err
	}
	o, err := s.packageOverview(ctx, tx, org, project)
	if err != nil {
		return p, err
	}
	provenance, _ := json.Marshal(map[string]any{"actor": actor, "proposal": key, "procurement_route": o.Route.Value})
	if key == procurement.WorksProposalKey {
		w := o.ProposedWork
		if w == nil {
			return p, ErrNotFound
		}
		if w.PackageID != "" {
			if p, err = readPackage(ctx, tx, org, project, w.PackageID); err != nil {
				return p, err
			}
		} else {
			p = procurement.Package{Kind: "works", WorksScope: "head_contract", Title: w.Title, LifecycleStatus: "planned", Stages: []procurement.Stage{}}
			if p, err = s.insertProposedPackage(ctx, tx, org, project, p, provenance); err != nil {
				return p, err
			}
		}
		for _, line := range w.Lines {
			content := procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: line.WorkItemID, Role: line.Role, UserText: line.Text, Inclusion: "included", InterfaceIDs: []string{}}
			if err := procurement.ValidateScope(content, p, s.workCatalog()); err != nil {
				return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
			}
			_, err := tx.Exec(ctx, `INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,user_text,inclusion,interface_ids,source_refs,origin,review_status,provenance)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'responsibility',$5::uuid,$6,$7,'included','{}','[]'::jsonb,'calculation','accepted_for_planning',$8::jsonb)`, org, newID(), project, p.ID, line.WorkItemID, line.Role, line.Text, provenance)
			if err != nil {
				return p, err
			}
		}
		return p, s.finishPackagesWrite(ctx, tx, org, project)
	}
	for _, sug := range o.Suggestions {
		if sug.Key != key {
			continue
		}
		p = procurement.Package{Kind: "services", Title: sug.Title, Novation: sug.Novation, LifecycleStatus: "planned"}
		p.Stages = procurement.DefaultStages(p, s.workCatalog())
		if p, err = s.insertProposedPackage(ctx, tx, org, project, p, provenance); err != nil {
			return p, err
		}
		return p, s.finishPackagesWrite(ctx, tx, org, project)
	}
	// Already accepted or no longer proposed: the inputs changed.
	return p, ErrNotFound
}

func (s *Store) insertProposedPackage(ctx context.Context, tx pgx.Tx, org, project string, p procurement.Package, provenance []byte) (procurement.Package, error) {
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	p.ID, p.ProjectID, p.Version = newID(), project, 1
	p.Origin, p.ReviewStatus, p.Meaning, p.Provenance = "calculation", "accepted_for_planning", "stated", provenance
	_, err := tx.Exec(ctx, `INSERT INTO packages(org_id,id,project_id,kind,works_scope,title,novation,lifecycle_status,origin,review_status,meaning,provenance,discipline_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6,$7,$8,'calculation','accepted_for_planning','stated',$9::jsonb,NULLIF($10,''))`, org, p.ID, project, p.Kind, p.WorksScope, p.Title, p.Novation, p.LifecycleStatus, provenance, p.DisciplineID)
	if err != nil {
		return p, err
	}
	for i := range p.Stages {
		stage, err := s.insertPackageStage(ctx, tx, org, p, p.Stages[i])
		if err != nil {
			return p, err
		}
		p.Stages[i] = stage
	}
	return p, nil
}
