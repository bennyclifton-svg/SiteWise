package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

func (s *Store) AcceptPackageProposal(ctx context.Context, org, project, key, actor, fingerprint string) (procurement.Package, error) {
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
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return p, err
	}
	if d, ok := decisions[key]; ok && d.Decision == "accepted" {
		if d.CreatedRecordType != "package" {
			return p, ErrProposalUnavailable
		}
		if d.InputsFingerprint != fingerprint {
			return p, ErrVersionConflict
		}
		p, err = readPackageRecord(ctx, tx, org, project, d.CreatedRecordID, true)
		if err != nil {
			return p, err
		}
		return p, tx.Commit(ctx)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p, ErrNotFound
		}
		return p, err
	}
	var proposal works.Proposal
	if err := json.Unmarshal(raw, &proposal); err != nil {
		return p, err
	}
	if proposal.InputsFingerprint != fingerprint {
		return p, ErrVersionConflict
	}
	if proposal.Kind != "discipline" {
		return p, ErrProposalUnavailable
	}
	// The knowledge label describes the proposed service. No discipline ID is
	// inferred from prose; it remains editable when an explicit mapping exists.
	p = procurement.Package{ID: newID(), ProjectID: project, Kind: "services", Title: proposal.Label, LifecycleStatus: "planned", Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated", SourceProposalKey: key, Version: 1}
	if err := procurement.ValidatePackage(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	p.Provenance, _ = json.Marshal(map[string]any{"actor": actor, "proposal": proposal})
	old, err := undoneProposalPackage(ctx, tx, org, project, key)
	if err != nil {
		return p, err
	}
	if old.ID != "" {
		p.ID, p.Version, p.Stages = old.ID, old.Version+1, old.Stages
		_, err = tx.Exec(ctx, `UPDATE packages SET title=$4,provenance=$5::jsonb,retired_at=NULL,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, p.ID, p.Title, p.Provenance)
		if err != nil {
			return p, err
		}
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status,meaning,provenance,source_proposal_key) VALUES($1::uuid,$2::uuid,$3::uuid,'services',$4,'planned','calculation','accepted_for_planning','stated',$5::jsonb,$6)`, org, p.ID, project, p.Title, p.Provenance, key)
		if err != nil {
			return p, err
		}
		p.Stages = procurement.DefaultStages(p, s.workCatalog())
		for i := range p.Stages {
			stage, err := s.insertPackageStage(ctx, tx, org, p, p.Stages[i])
			if err != nil {
				return p, err
			}
			p.Stages[i] = stage
		}
	}
	raw, err = json.Marshal(struct {
		works.Proposal
		CreatedStages []procurement.Stage `json:"created_stages"`
	}{proposal, p.Stages})
	if err != nil {
		return p, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id,created_record_version,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,(SELECT work_item_id FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$3::uuid AND proposal_key=$4 ORDER BY work_item_id LIMIT 1),'accepted',$6,$7::uuid,'package',$8::uuid,$10,$9::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET trigger_work_item_id=EXCLUDED.trigger_work_item_id,decision='accepted',inputs_fingerprint=EXCLUDED.inputs_fingerprint,actor=EXCLUDED.actor,decided_at=now(),created_record_type='package',created_record_id=EXCLUDED.created_record_id,created_record_version=EXCLUDED.created_record_version,inputs_snapshot=EXCLUDED.inputs_snapshot,rationale='',version=proposal_decisions.version+1,undone_at=NULL`, org, newID(), project, key, proposal.RecordID, fingerprint, actor, p.ID, raw, p.Version)
	if err != nil {
		return p, err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='accepted',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return p, err
	}
	if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
		return p, err
	}
	if err := workEvent(ctx, tx, s, org, project); err != nil {
		return p, err
	}
	if err := s.finishPackagesWrite(ctx, tx, org, project); err != nil {
		return p, err
	}
	return p, nil
}
