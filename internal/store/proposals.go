package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/profile"
	"sitewise/internal/works"
)

var ErrProposalUnavailable = errors.New("proposal evaluator is not configured")
var ErrInvalidProposalDecision = errors.New("invalid proposal decision")

func writeProposals(ctx context.Context, tx pgx.Tx, org, project, site string, proposals []works.Proposal) error {
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	projection := make([]map[string]any, 0, len(proposals))
	type triggerRow struct {
		Key        string `json:"proposal_key"`
		WorkItemID string `json:"work_item_id"`
	}
	triggers := []triggerRow{}
	keys := make([]string, 0, len(proposals))
	seenKeys := map[string]bool{}
	null := func(value string) any {
		if value == "" {
			return nil
		}
		return value
	}
	for _, p := range proposals {
		if seenKeys[p.Key] {
			return fmt.Errorf("duplicate proposal key")
		}
		seenKeys[p.Key] = true
		keys = append(keys, p.Key)
		if d, ok := decisions[p.Key]; ok {
			p = works.ApplyProposalDecision(p, &works.ProposalDecision{Decision: d.Decision, InputsFingerprint: d.InputsFingerprint})
		}
		projection = append(projection, map[string]any{
			"key": p.Key, "record_kind": p.RecordKind, "record_id": p.RecordID, "interface_id": null(p.InterfaceID),
			"proposal_index": p.ProposalIndex, "target_system_id": null(p.TargetSystemID), "target_part_id": null(p.TargetPartID),
			"kind": p.Kind, "label": p.Label, "action": null(p.Action), "reason": p.Reason, "severity": p.Severity,
			"specificity": p.Specificity, "rank": p.Rank, "critical": p.Critical, "draft": p.Draft,
			"unaccepted_triggers": p.UnacceptedTriggers, "inputs_fingerprint": p.InputsFingerprint,
			"knowledge_version": p.KnowledgeVersion, "state": p.State, "inputs_changed": p.InputsChanged,
		})
		seen := map[string]bool{}
		for _, trigger := range p.Reason.Triggers {
			if trigger.WorkItemID == "" {
				return fmt.Errorf("empty proposal trigger")
			}
			if !seen[trigger.WorkItemID] {
				triggers = append(triggers, triggerRow{p.Key, trigger.WorkItemID})
				seen[trigger.WorkItemID] = true
			}
		}
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		return err
	}
	relations, err := json.Marshal(triggers)
	if err != nil {
		return err
	}
	// Ranking changes must not delete and recreate unchanged trigger links.
	// Compare every mutable column, including provenance and display metadata;
	// semantic fingerprints alone intentionally ignore split work-item IDs.
	batch := &pgx.Batch{}
	batch.Queue(`INSERT INTO proposals(org_id,project_id,site_id,key,record_kind,record_id,interface_id,proposal_index,
 target_system_id,target_part_id,kind,label,action,reason,severity,specificity,rank,critical,draft,
 unaccepted_triggers,inputs_fingerprint,knowledge_version,state,inputs_changed)
 SELECT $1::uuid,$2::uuid,$3::uuid,p.key,p.record_kind,p.record_id,p.interface_id,p.proposal_index,
 p.target_system_id,p.target_part_id,p.kind,p.label,p.action,p.reason,p.severity,p.specificity,p.rank,p.critical,p.draft,
 p.unaccepted_triggers,p.inputs_fingerprint,p.knowledge_version,p.state,p.inputs_changed
 FROM jsonb_to_recordset($4::jsonb) AS p(key text,record_kind text,record_id text,interface_id text,proposal_index integer,
 target_system_id text,target_part_id uuid,kind text,label text,action text,reason jsonb,severity text,specificity integer,
 rank integer,critical boolean,draft boolean,unaccepted_triggers boolean,inputs_fingerprint text,knowledge_version text,state text,inputs_changed boolean)
 ON CONFLICT(org_id,project_id,key) DO UPDATE SET
 (site_id,record_kind,record_id,interface_id,proposal_index,target_system_id,target_part_id,kind,label,action,reason,
 severity,specificity,rank,critical,draft,unaccepted_triggers,inputs_fingerprint,knowledge_version,state,inputs_changed)=
 (EXCLUDED.site_id,EXCLUDED.record_kind,EXCLUDED.record_id,EXCLUDED.interface_id,EXCLUDED.proposal_index,
 EXCLUDED.target_system_id,EXCLUDED.target_part_id,EXCLUDED.kind,EXCLUDED.label,EXCLUDED.action,EXCLUDED.reason,
 EXCLUDED.severity,EXCLUDED.specificity,EXCLUDED.rank,EXCLUDED.critical,EXCLUDED.draft,EXCLUDED.unaccepted_triggers,
 EXCLUDED.inputs_fingerprint,EXCLUDED.knowledge_version,EXCLUDED.state,EXCLUDED.inputs_changed)
 WHERE (proposals.site_id,proposals.record_kind,proposals.record_id,proposals.interface_id,proposals.proposal_index,
 proposals.target_system_id,proposals.target_part_id,proposals.kind,proposals.label,proposals.action,proposals.reason,
 proposals.severity,proposals.specificity,proposals.rank,proposals.critical,proposals.draft,proposals.unaccepted_triggers,
 proposals.inputs_fingerprint,proposals.knowledge_version,proposals.state,proposals.inputs_changed)
 IS DISTINCT FROM (EXCLUDED.site_id,EXCLUDED.record_kind,EXCLUDED.record_id,EXCLUDED.interface_id,EXCLUDED.proposal_index,
 EXCLUDED.target_system_id,EXCLUDED.target_part_id,EXCLUDED.kind,EXCLUDED.label,EXCLUDED.action,EXCLUDED.reason,
 EXCLUDED.severity,EXCLUDED.specificity,EXCLUDED.rank,EXCLUDED.critical,EXCLUDED.draft,EXCLUDED.unaccepted_triggers,
 EXCLUDED.inputs_fingerprint,EXCLUDED.knowledge_version,EXCLUDED.state,EXCLUDED.inputs_changed)`, org, project, site, string(raw))
	batch.Queue(`WITH desired AS MATERIALIZED (
 SELECT * FROM jsonb_to_recordset($3::jsonb) AS r(proposal_key text,work_item_id uuid)
 ) DELETE FROM proposal_triggers t WHERE t.org_id=$1::uuid AND t.project_id=$2::uuid
 AND NOT EXISTS(SELECT 1 FROM desired d WHERE d.proposal_key=t.proposal_key AND d.work_item_id=t.work_item_id)`, org, project, string(relations))
	batch.Queue(`INSERT INTO proposal_triggers(org_id,project_id,proposal_key,work_item_id)
 SELECT $1::uuid,$2::uuid,r.proposal_key,r.work_item_id FROM jsonb_to_recordset($3::jsonb) AS r(proposal_key text,work_item_id uuid)
 ON CONFLICT(org_id,project_id,proposal_key,work_item_id) DO NOTHING`, org, project, string(relations))
	batch.Queue(`DELETE FROM proposals WHERE org_id=$1::uuid AND project_id=$2::uuid AND NOT(key=ANY($3::text[]))`, org, project, keys)
	return tx.SendBatch(ctx, batch).Close()
}

type ProposalDecisionView struct {
	ID                   string          `json:"id"`
	ProposalKey          string          `json:"proposal_key"`
	RecordID             string          `json:"record_id"`
	TriggerWorkItemID    string          `json:"trigger_work_item_id,omitempty"`
	Decision             string          `json:"decision"`
	InputsFingerprint    string          `json:"inputs_fingerprint"`
	Rationale            string          `json:"rationale"`
	Actor                string          `json:"actor"`
	DecidedAt            time.Time       `json:"decided_at"`
	CreatedRecordType    string          `json:"created_record_type,omitempty"`
	CreatedRecordID      string          `json:"created_record_id,omitempty"`
	CreatedRecordVersion int64           `json:"created_record_version,omitempty"`
	Version              int64           `json:"version"`
	InputsSnapshot       json.RawMessage `json:"inputs_snapshot"`
}

type ProposalView struct {
	works.Proposal
	TriggerWorkItemIDs []string              `json:"trigger_work_item_ids"`
	Decision           *ProposalDecisionView `json:"decision,omitempty"`
}

func readProposalDecisions(ctx context.Context, q rowQuerier, org, project string) (map[string]ProposalDecisionView, error) {
	rows, err := q.Query(ctx, `SELECT to_jsonb(d) FROM proposal_decisions d WHERE org_id=$1::uuid AND project_id=$2::uuid AND undone_at IS NULL`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ProposalDecisionView{}
	for rows.Next() {
		var raw []byte
		var d ProposalDecisionView
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		out[d.ProposalKey] = d
	}
	return out, rows.Err()
}

func readProposals(ctx context.Context, q rowQuerier, org, project string) ([]ProposalView, error) {
	rows, err := q.Query(ctx, `SELECT to_jsonb(p)||jsonb_build_object('trigger_work_item_ids',COALESCE((SELECT jsonb_agg(t.work_item_id ORDER BY t.work_item_id) FROM proposal_triggers t WHERE t.org_id=p.org_id AND t.project_id=p.project_id AND t.proposal_key=p.key),'[]'::jsonb)) FROM proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY rank,key`, org, project)
	if err != nil {
		return nil, err
	}
	out := []ProposalView{}
	for rows.Next() {
		var raw []byte
		var p ProposalView
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	decisions, err := readProposalDecisions(ctx, q, org, project)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if d, ok := decisions[out[i].Key]; ok {
			out[i].Decision = &d
		}
	}
	return out, nil
}

func (s *Store) ReadProposals(ctx context.Context, org, project string) ([]ProposalView, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	out, err := readProposals(ctx, tx, org, project)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// RebuildProfileWithProposals measures the complete single-transaction rebuild.
// It is explicit diagnostic wiring, not called by ordinary profile edits.
func (s *Store) RebuildProfileWithProposals(ctx context.Context, org, project string, evaluator *works.Evaluator) error {
	if s.profileBuild == nil || s.profileBuild.Catalog == nil || s.profileBuild.Compute == nil || evaluator == nil {
		return ErrProposalUnavailable
	}
	if evaluator.KnowledgeVersion() != s.profileBuild.Catalog.Version() {
		return ErrProposalUnavailable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.rebuildProfileProjectionTx(ctx, tx, org, project, *s.profileBuild, evaluator); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RebuildProposals is an explicit code-only operation for measurement and
// integration. Ordinary profile edits do not call it until timing gates pass.
func (s *Store) RebuildProposals(ctx context.Context, org, project string, evaluator *works.Evaluator) error {
	if s.profileBuild == nil || s.profileBuild.Catalog == nil || s.profileBuild.Compute == nil || evaluator == nil {
		return ErrProposalUnavailable
	}
	if evaluator.KnowledgeVersion() != s.profileBuild.Catalog.Version() {
		return ErrProposalUnavailable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	snapshot, err := readSnapshot(ctx, tx, org, project)
	if err != nil {
		return err
	}
	items, err := readWorkItems(ctx, tx, org, project)
	if err != nil {
		return err
	}
	build := s.profileBuild
	rows := build.Compute(snapshot)
	inputs, err := profile.ProposalInputs(rows, snapshot.Parts, items, build.Catalog)
	if err != nil {
		return err
	}
	proposals, err := evaluator.Evaluate(inputs)
	if err != nil {
		return err
	}
	var siteID string
	if err := tx.QueryRow(ctx, `SELECT site_id::text FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, project).Scan(&siteID); err != nil {
		return err
	}
	if err := writeProposals(ctx, tx, org, project, siteID, proposals); err != nil {
		return err
	}
	err = tx.Commit(ctx)
	return err
}

// DismissProposal serialises with rebuilds and records the exact projection
// the person saw. Retrying the same dismissal does not advance its version.
func (s *Store) DismissProposal(ctx context.Context, org, project, key, actor, fingerprint, rationale string) (ProposalDecisionView, error) {
	var result ProposalDecisionView
	if !utf8.ValidString(rationale) || utf8.RuneCountInString(rationale) > 200 {
		return result, ErrInvalidProposalDecision
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return result, err
	}
	var actorExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE org_id=$1::uuid AND id=$2::uuid)`, org, actor).Scan(&actorExists); err != nil {
		return result, err
	}
	if !actorExists {
		return result, ErrNotFound
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT to_jsonb(p) FROM proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, ErrNotFound
		}
		return result, err
	}
	var p works.Proposal
	if err := json.Unmarshal(raw, &p); err != nil {
		return result, err
	}
	if p.InputsFingerprint != fingerprint {
		return result, ErrVersionConflict
	}
	decisions, err := readProposalDecisions(ctx, tx, org, project)
	if err != nil {
		return result, err
	}
	if old, ok := decisions[key]; ok {
		if old.Decision == "accepted" {
			return old, ErrVersionConflict
		}
		if old.InputsFingerprint == fingerprint && old.Rationale == rationale {
			return old, tx.Commit(ctx)
		}
	}
	snapshot, err := json.Marshal(p)
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,rationale,actor,inputs_snapshot)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,(SELECT work_item_id FROM proposal_triggers WHERE org_id=$1::uuid AND project_id=$3::uuid AND proposal_key=$4 ORDER BY work_item_id LIMIT 1),'dismissed',$6,$7,$8::uuid,$9::jsonb)
 ON CONFLICT(org_id,project_id,proposal_key) DO UPDATE SET trigger_work_item_id=EXCLUDED.trigger_work_item_id,decision='dismissed',inputs_fingerprint=EXCLUDED.inputs_fingerprint,rationale=EXCLUDED.rationale,actor=EXCLUDED.actor,decided_at=now(),inputs_snapshot=EXCLUDED.inputs_snapshot,version=proposal_decisions.version+1,undone_at=NULL,created_record_type=NULL,created_record_id=NULL,created_record_version=NULL
 RETURNING to_jsonb(proposal_decisions)`, org, newID(), project, key, p.RecordID, fingerprint, rationale, actor, snapshot).Scan(&raw)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `UPDATE proposals SET state='dismissed',inputs_changed=false WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, org, project, key); err != nil {
		return result, err
	}
	if err := s.finishWorksWrite(ctx, tx, org, project); err != nil {
		return result, err
	}
	return result, nil
}
