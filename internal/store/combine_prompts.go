package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
)

// Combined engagements, slice 5 (docs/plans/2026-11-07-combined-engagements-slice5.md):
// prompts to combine rows made before groups, and Combine for this proposal.
// No Jev call here.

// DismissCombinePrompt hides a combine prompt on this project for good. It
// is kept with the dismissed package suggestions, as a dismissed decision.
func (s *Store) DismissCombinePrompt(ctx context.Context, org, project, actor, key string) error {
	if !procurement.IsCombinePrompt(key) {
		return fmt.Errorf("%w: not a combine prompt", ErrInvalidPackage)
	}
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
	sum := sha256.Sum256([]byte(key))
	snapshot, _ := json.Marshal(map[string]string{"prompt": key})
	if _, err := tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,decision,inputs_fingerprint,actor,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$4,'dismissed',$5,$6::uuid,$7::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET decision='dismissed',actor=EXCLUDED.actor,decided_at=now(),version=proposal_decisions.version+1,undone_at=NULL`,
		org, newID(), project, key, hex.EncodeToString(sum[:]), actor, snapshot); err != nil {
		return err
	}
	return s.finishPackagesWrite(ctx, tx, org, project)
}

// CombineForProposal combines the rows the PM ticked and, in the same
// transaction, places an unplaced proposal on the new group as the PM's
// choice. Nothing is combined when the proposal cannot be placed.
func (s *Store) CombineForProposal(ctx context.Context, org, project, actor string, ids []string, title, submission string) (procurement.Package, error) {
	if !validUUID(submission) {
		return procurement.Package{}, ErrNotFound
	}
	return s.combine(ctx, org, project, actor, ids, title, func(tx pgx.Tx, group procurement.Package) error {
		sub, err := lockSubmission(ctx, tx, org, project, submission)
		if err != nil {
			return err
		}
		// Only a proposal waiting for a package: one already placed is moved
		// on its own row, not by a combine.
		if sub.PackageID != "" || sub.Status == SubmissionAwarded {
			return ErrConflict
		}
		if err := placeSubmission(ctx, tx, org, project, sub, group.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET updated_at=now(),version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, submission); err != nil {
			return err
		}
		return procurementEvent(ctx, tx, s, org, project, submission, "edited")
	})
}
