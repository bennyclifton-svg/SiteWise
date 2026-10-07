package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

// Package and optional assignment are explicit choices; catalogue prose cannot
// decide who is appointed or which existing work item carries a responsibility.
type ScopeAcceptance struct {
	PackageID  string `json:"package_id"`
	WorkItemID string `json:"work_item_id,omitempty"`
	Role       string `json:"role,omitempty"`
	StageID    string `json:"stage_id,omitempty"`
}

func readScopeRecord(ctx context.Context, tx pgx.Tx, org, project, id string) (ScopeItem, error) {
	var item ScopeItem
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(i) FROM package_scope_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	err = json.Unmarshal(raw, &item)
	return item, err
}

func (s *Store) AcceptScopeProposal(ctx context.Context, org, project, key, actor, fingerprint string, choice ScopeAcceptance) (ScopeItem, error) {
	var item ScopeItem
	var uuid pgtype.UUID
	if uuid.Scan(choice.PackageID) != nil {
		return item, ErrInvalidPackage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return item, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return item, err
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return item, err
	}
	if d, ok := decisions[key]; ok && d.Decision == "accepted" {
		if d.CreatedRecordType != "package_scope_item" {
			return item, ErrProposalUnavailable
		}
		if d.InputsFingerprint != fingerprint {
			return item, ErrVersionConflict
		}
		// Compare against the accepted assignment, not the subsequently edited row.
		var saved struct {
			Assignment ScopeAcceptance `json:"assignment"`
		}
		if err := json.Unmarshal(d.InputsSnapshot, &saved); err != nil {
			return item, err
		}
		if saved.Assignment != choice {
			return item, ErrVersionConflict
		}
		item, err = readScopeRecord(ctx, tx, org, project, d.CreatedRecordID)
		if err != nil {
			return item, err
		}
		item.Provisional = s.scopeProvisional(item)
		return item, tx.Commit(ctx)
	}
	pkg, err := readPackage(ctx, tx, org, project, choice.PackageID)
	if err != nil {
		return item, err
	}
	var raw []byte
	var proposal works.Proposal
	err = tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &proposal); err != nil {
		return item, err
	}
	if proposal.InputsFingerprint != fingerprint {
		return item, ErrVersionConflict
	}
	if proposal.Kind != "obligation" {
		return item, ErrProposalUnavailable
	}
	content := procurement.ScopeContent{ItemKind: "obligation", WorkItemID: choice.WorkItemID, Role: choice.Role, StageID: choice.StageID, UserText: proposal.Label, Inclusion: "included", InterfaceIDs: []string{}}
	if proposal.InterfaceID != "" {
		content.InterfaceIDs = append(content.InterfaceIDs, proposal.InterfaceID)
	}
	if err := procurement.ValidateScope(content, pkg, s.workCatalog()); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	if err := scopeTargets(ctx, tx, org, project, pkg.ID, content); err != nil {
		return item, err
	}
	// If a retained work item is selected, planning must name the works package
	// that maintains or protects it (D-09); no installer is invented.
	if choice.WorkItemID != "" {
		var action string
		if err := tx.QueryRow(ctx, `SELECT action FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, choice.WorkItemID).Scan(&action); err != nil {
			return item, err
		}
		if action == "retain" && (pkg.Kind != "works" || (choice.Role != "maintain_operation" && choice.Role != "protect")) {
			return item, ErrInvalidPackage
		}
	}
	id, version := newID(), int64(1)
	var oldID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND source_proposal_key=$3`, org, project, key).Scan(&oldID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return item, err
	}
	if oldID != "" {
		old, err := readScopeRecord(ctx, tx, org, project, oldID)
		if err != nil {
			return item, err
		}
		var eligible bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM proposal_decisions d CROSS JOIN LATERAL jsonb_array_elements(d.undo_history) h WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.proposal_key=$3 AND h->>'created_record_type'='package_scope_item' AND h->>'created_record_id'=$4 AND (h->>'created_record_version')::bigint+1=$5)`, org, project, key, oldID, old.Version).Scan(&eligible); err != nil {
			return item, err
		}
		if old.RetiredAt == nil || !eligible {
			return item, ErrProposalUndoBlocked
		}
		id, version = old.ID, old.Version+1
	}
	provenance, _ := json.Marshal(map[string]any{"actor": actor, "proposal": proposal})
	err = tx.QueryRow(ctx, `INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,user_text,stage_id,inclusion,interface_ids,origin,review_status,provenance,source_proposal_key)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'obligation',NULLIF($5,'')::uuid,NULLIF($6,''),$7,NULLIF($8,'')::uuid,'included',$9,'calculation','accepted_for_planning',$10::jsonb,$11)
 ON CONFLICT(org_id,id) DO UPDATE SET package_id=EXCLUDED.package_id,work_item_id=EXCLUDED.work_item_id,role=EXCLUDED.role,user_text=EXCLUDED.user_text,stage_id=EXCLUDED.stage_id,interface_ids=EXCLUDED.interface_ids,provenance=EXCLUDED.provenance,retired_at=NULL,version=package_scope_items.version+1 RETURNING to_jsonb(package_scope_items)`, org, id, project, pkg.ID, choice.WorkItemID, choice.Role, proposal.Label, choice.StageID, content.InterfaceIDs, provenance, key).Scan(&raw)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	snapshot, _ := json.Marshal(struct {
		works.Proposal
		Assignment ScopeAcceptance `json:"assignment"`
	}{proposal, choice})
	_, err = tx.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,created_record_type,created_record_id,created_record_version,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,(SELECT work_item_id FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$3::uuid AND proposal_key=$4 ORDER BY work_item_id LIMIT 1),'accepted',$6,$7::uuid,'package_scope_item',$8::uuid,$9,$10::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET trigger_work_item_id=EXCLUDED.trigger_work_item_id,decision='accepted',inputs_fingerprint=EXCLUDED.inputs_fingerprint,actor=EXCLUDED.actor,decided_at=now(),created_record_type=EXCLUDED.created_record_type,created_record_id=EXCLUDED.created_record_id,created_record_version=EXCLUDED.created_record_version,inputs_snapshot=EXCLUDED.inputs_snapshot,rationale='',version=proposal_decisions.version+1,undone_at=NULL`, org, newID(), project, key, proposal.RecordID, fingerprint, actor, id, version, snapshot)
	if err != nil {
		return item, err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='accepted',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return item, err
	}
	if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
		return item, err
	}
	if err := workEvent(ctx, tx, s, org, project); err != nil {
		return item, err
	}
	item.Provisional = s.scopeProvisional(item)
	return item, s.finishPackagesWrite(ctx, tx, org, project)
}

func (s *Store) scopeProvisional(item ScopeItem) bool {
	var p struct {
		Proposal struct {
			Draft bool `json:"draft"`
		} `json:"proposal"`
	}
	if json.Unmarshal(item.Provenance, &p) != nil {
		return true
	}
	return p.Proposal.Draft || procurement.ClauseProvisional(item.ScopeContent, s.workCatalog())
}
