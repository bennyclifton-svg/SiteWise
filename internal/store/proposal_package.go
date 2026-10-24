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

// EnsureProposalPackage gives a filed proposal a Procurement row to land in.
// When the document's filed discipline is a consultant or trade discipline
// and no live package holds it, the discipline's package is created with
// its default stages, origin document. The proposal read that follows then
// links it by the existing discipline rule and prices its fee schedule.
//
// Nothing is created when the discipline is unknown or a supplier, when a
// live package already holds it (the read links that one), or when the PM
// removed that discipline's package: their removal stands. label names the
// discipline as the Procurement table does. created reports a new package.
func (s *Store) EnsureProposalPackage(ctx context.Context, org, project, documentID string, label func(id string) string) (id string, created bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return "", false, err
	}
	var discipline, band string
	err = tx.QueryRow(ctx, `SELECT COALESCE(c.value,''), c.band FROM decisions c JOIN documents d ON d.org_id=c.org_id AND d.id=c.document_id
WHERE c.org_id=$1::uuid AND d.project_id=$2::uuid AND c.document_id::text=$3 AND c.field='discipline'`, org, project, documentID).Scan(&discipline, &band)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	// A blank or grey filing is not a discipline anyone stated.
	if discipline == "" || (band != "green" && band != "amber" && band != "user") {
		return "", false, nil
	}
	p, ok := procurement.DisciplinePackage(discipline, label(discipline))
	if !ok || p.Title == "" {
		return "", false, nil
	}
	taken, err := disciplineTaken(ctx, tx, org, project, discipline)
	if err != nil || taken {
		return "", false, err
	}
	var removed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND discipline_id=$3 AND retired_at IS NOT NULL)`, org, project, discipline).Scan(&removed); err != nil || removed {
		return "", false, err
	}
	facts, err := readPackageFacts(ctx, tx, org, project)
	if err != nil {
		return "", false, err
	}
	p.Novation = procurement.NovationNo
	if p.Kind == "services" {
		p.Novation = procurement.DefaultNovation(profile.PackageComplexityFacts(facts)["procurement_route"])
	}
	if err := procurement.ValidatePackage(p); err != nil {
		return "", false, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	plan, err := readStagePlan(ctx, tx, org, project, false)
	if err != nil {
		return "", false, err
	}
	p.Stages = s.defaultStages(p, plan)
	p.Origin = "document"
	provenance, _ := json.Marshal(map[string]string{"document_id": documentID, "discipline": discipline, "reason": "proposal received"})
	if p, err = s.insertProposedPackage(ctx, tx, org, project, p, provenance, plan); err != nil {
		if errors.Is(err, ErrDuplicatePackage) {
			return "", false, nil
		}
		return "", false, err
	}
	return p.ID, true, s.finishPackagesWrite(ctx, tx, org, project)
}
