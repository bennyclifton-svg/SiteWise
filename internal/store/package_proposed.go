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
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return p, err
	}
	if p, err = s.acceptProposalTx(ctx, tx, org, project, actor, key, ""); err != nil {
		return p, err
	}
	return p, s.finishPackagesWrite(ctx, tx, org, project)
}

// acceptProposalTx builds the records for one proposal inside the caller's
// transaction and does not commit, so Procurement can create the report in
// the same transaction. disciplineID labels the new package only: an
// existing head-contract package keeps whatever discipline it has.
func (s *Store) acceptProposalTx(ctx context.Context, tx pgx.Tx, org, project, actor, key, disciplineID string) (procurement.Package, error) {
	var p procurement.Package
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return p, err
	}
	if !exists {
		return p, ErrNotFound
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
			p = procurement.Package{Kind: "works", WorksScope: "head_contract", DisciplineID: disciplineID, Title: w.Title, LifecycleStatus: "planned", Stages: []procurement.Stage{}}
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
		return p, nil
	}
	for _, sug := range o.Suggestions {
		if sug.Key != key {
			continue
		}
		p = procurement.Package{Kind: "services", DisciplineID: disciplineID, Title: sug.Title, Novation: sug.Novation, LifecycleStatus: "planned"}
		p.Stages = procurement.DefaultStages(p, s.workCatalog())
		if p, err = s.insertProposedPackage(ctx, tx, org, project, p, provenance); err != nil {
			return p, err
		}
		return p, nil
	}
	// Already accepted or no longer proposed: the inputs changed.
	return p, ErrNotFound
}

func (s *Store) insertProposedPackage(ctx context.Context, tx pgx.Tx, org, project string, p procurement.Package, provenance []byte) (procurement.Package, error) {
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	if p.DisciplineID != "" {
		taken, err := disciplineTaken(ctx, tx, org, project, p.DisciplineID)
		if err != nil {
			return p, err
		}
		if taken {
			return p, ErrDuplicatePackage
		}
	}
	if p.Origin == "" {
		p.Origin = "calculation"
	}
	p.ID, p.ProjectID, p.Version = newID(), project, 1
	p.ReviewStatus, p.Meaning, p.Provenance = "accepted_for_planning", "stated", provenance
	_, err := tx.Exec(ctx, `INSERT INTO packages(org_id,id,project_id,kind,works_scope,title,novation,lifecycle_status,origin,review_status,meaning,provenance,discipline_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),$6,$7,$8,$9,'accepted_for_planning','stated',$10::jsonb,NULLIF($11,''))`, org, p.ID, project, p.Kind, p.WorksScope, p.Title, p.Novation, p.LifecycleStatus, p.Origin, provenance, p.DisciplineID)
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

// disciplineTaken keeps the Procurement table one row per discipline: a
// live package already holds the discipline, or, for Main Works, a live head
// contract exists (legacy ones have no discipline). Checked under the
// project lock rather than by a unique index, so retired rows never block.
func disciplineTaken(ctx context.Context, tx pgx.Tx, org, project, disciplineID string) (bool, error) {
	var taken bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL AND (discipline_id=$3 OR ($3='trade.main_works' AND works_scope='head_contract')))`, org, project, disciplineID).Scan(&taken)
	return taken, err
}

// DismissPackageSuggestion records that the PM removed a suggested consultant
// row, so it is not proposed again on this project. The suggestion is
// recomputed under the project lock; a key that is not suggested now is
// refused unless it was already dismissed, so a retry is harmless.
func (s *Store) DismissPackageSuggestion(ctx context.Context, org, project, actor, key string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	if d, ok := decisions[key]; ok && d.Decision == "dismissed" {
		return nil
	}
	o, err := s.packageOverview(ctx, tx, org, project)
	if err != nil {
		return err
	}
	var sug *procurement.PackageSuggestion
	for i := range o.Suggestions {
		if o.Suggestions[i].Key == key {
			sug = &o.Suggestions[i]
		}
	}
	if sug == nil {
		return ErrNotFound
	}
	snapshot, err := json.Marshal(sug)
	if err != nil {
		return err
	}
	// The knowledge version is the fingerprint: the lists it came from.
	if _, err := tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,decision,inputs_fingerprint,actor,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$4,'dismissed',$5,$6::uuid,$7::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET decision='dismissed',inputs_fingerprint=EXCLUDED.inputs_fingerprint,actor=EXCLUDED.actor,decided_at=now(),inputs_snapshot=EXCLUDED.inputs_snapshot,rationale='',version=proposal_decisions.version+1,undone_at=NULL,created_record_type=NULL,created_record_id=NULL,created_record_version=NULL`,
		org, newID(), project, key, sug.KnowledgeVersion, actor, snapshot); err != nil {
		return err
	}
	return s.finishPackagesWrite(ctx, tx, org, project)
}
