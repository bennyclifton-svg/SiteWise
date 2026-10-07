package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

// AcceptProposal creates planning scope, never verified evidence. The project
// lock serialises acceptance with rebuilds and concurrent retries.
func (s *Store) AcceptProposal(ctx context.Context, org, project, key, actor, fingerprint string) (works.Item, error) {
	return s.AcceptProposalWithAction(ctx, org, project, key, actor, fingerprint, "")
}

// An actionless physical proposal requires the user's explicit selection.
// The selection is preserved separately from the evidence fingerprint.
func (s *Store) AcceptProposalWithAction(ctx context.Context, org, project, key, actor, fingerprint, chosenAction string) (works.Item, error) {
	var item works.Item
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return item, err
	}
	var actorExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE org_id=$1::uuid AND id=$2::uuid)`, org, actor).Scan(&actorExists); err != nil {
		return item, err
	}
	if !actorExists {
		return item, ErrNotFound
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return item, err
	}
	// A retry uses the fingerprint originally accepted, even if a subsequent
	// rebuild changed or removed the projection.
	if d, ok := decisions[key]; ok && d.Decision == "accepted" {
		if d.CreatedRecordType != "work_item" {
			return item, ErrProposalUnavailable
		}
		if d.InputsFingerprint != fingerprint {
			return item, ErrVersionConflict
		}
		var raw []byte
		if err := tx.QueryRow(ctx, `SELECT to_jsonb(w)||jsonb_build_object('quantity',quantity::text) FROM work_items w WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, d.CreatedRecordID).Scan(&raw); err != nil {
			return item, err
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return item, err
		}
		if chosenAction != "" && chosenAction != item.Action {
			return works.Item{}, ErrVersionConflict
		}
		return item, tx.Commit(ctx)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM ranked_proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return item, ErrNotFound
		}
		return item, err
	}
	var p works.Proposal
	if err := json.Unmarshal(raw, &p); err != nil {
		return item, err
	}
	if p.InputsFingerprint != fingerprint {
		return item, ErrVersionConflict
	}
	if p.Kind != "investigation" && p.Kind != "work_item" {
		return item, ErrProposalUnavailable
	}
	action := p.Action
	if chosenAction != "" {
		if p.Kind != "work_item" || !works.ValidAction(chosenAction) || chosenAction == "investigate" || action != "" && action != chosenAction {
			return item, ErrInvalidWork
		}
		action = chosenAction
		var inputs map[string]json.RawMessage
		if err := json.Unmarshal(raw, &inputs); err != nil {
			return item, err
		}
		inputs["user_selected_action"], _ = json.Marshal(chosenAction)
		raw, err = json.Marshal(inputs)
		if err != nil {
			return item, err
		}
	}
	if p.Kind == "investigation" {
		action = "investigate"
	}
	// An ambiguous target must be resolved in knowledge, not guessed here.
	if p.TargetSystemID == "" || p.TargetPartID == "" || !works.ValidAction(action) {
		return item, ErrProposalUnavailable
	}
	site, err := userValueOwner(ctx, tx, org, project, p.TargetPartID)
	if err != nil {
		return item, err
	}
	item = works.Item{ID: newID(), ProjectID: project, SiteID: site, PartID: p.TargetPartID, SystemID: p.TargetSystemID, Action: action, Inclusion: "included", Title: p.Label, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated", UserTouched: true, SourceProposalKey: key, Version: 1, Provenance: works.Provenance{Actor: actor, Band: "user", Sources: raw}}
	if err := works.Validate(item, s.workCatalog()); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidWork, err)
	}
	prov, err := json.Marshal(item.Provenance)
	if err != nil {
		return item, err
	}
	oldID, oldVersion, err := undoneProposalWork(ctx, tx, org, project, key)
	if err != nil {
		return item, err
	}
	if oldID == "" {
		_, err = tx.Exec(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,meaning,provenance,user_touched,source_proposal_key) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'included',$8,'calculation','accepted_for_planning','stated',$9::jsonb,true,$10)`, org, item.ID, project, site, item.PartID, item.SystemID, action, item.Title, prov, key)
	} else {
		item.ID, item.Version = oldID, oldVersion+1
		_, err = tx.Exec(ctx, `UPDATE work_items SET retired_at=NULL,retired_by=NULL,version=version+1,action=$4,title=$5,provenance=$6::jsonb WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, item.ID, action, item.Title, prov)
	}
	if err != nil {
		return item, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id,inputs_snapshot,created_record_version)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,(SELECT work_item_id FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$3::uuid AND proposal_key=$4 ORDER BY work_item_id LIMIT 1),'accepted',$6,$7::uuid,'work_item',$8::uuid,$9::jsonb,$10)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET trigger_work_item_id=EXCLUDED.trigger_work_item_id,decision='accepted',inputs_fingerprint=EXCLUDED.inputs_fingerprint,actor=EXCLUDED.actor,decided_at=now(),created_record_type=EXCLUDED.created_record_type,created_record_id=EXCLUDED.created_record_id,created_record_version=EXCLUDED.created_record_version,inputs_snapshot=EXCLUDED.inputs_snapshot,rationale='',version=proposal_decisions.version+1,undone_at=NULL`, org, newID(), project, key, p.RecordID, fingerprint, actor, item.ID, raw, item.Version); err != nil {
		return item, err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='accepted',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return item, err
	}
	if err := s.finishWorksWrite(ctx, tx, org, project); err != nil {
		return item, err
	}
	return item, nil
}
