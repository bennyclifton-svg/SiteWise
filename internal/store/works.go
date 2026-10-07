package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/knowledge"
	"sitewise/internal/works"
)

var ErrWorkConflict = errors.New("a coarse work item already exists")
var ErrInvalidWork = errors.New("invalid work item")

func readWorkItems(ctx context.Context, q rowQuerier, org, project string) ([]works.Item, error) {
	rows, err := q.Query(ctx, `SELECT id::text,project_id::text,site_id::text,part_id::text,system_id,
 action,inclusion,COALESCE(parent_id::text,''),is_group,title,existing_condition_note,target,
 quantity::text,unit,origin,review_status,meaning,provenance,user_touched,
 COALESCE(coarse_key,''),COALESCE(source_proposal_key,''),retired_at,version
 FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY part_id,system_id,id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []works.Item{}
	for rows.Next() {
		var target, provenance []byte
		var item works.Item
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.SiteID, &item.PartID, &item.SystemID,
			&item.Action, &item.Inclusion, &item.ParentID, &item.IsGroup, &item.Title, &item.ExistingConditionNote, &target,
			&item.Quantity, &item.Unit, &item.Origin, &item.ReviewStatus, &item.Meaning, &provenance, &item.UserTouched,
			&item.CoarseKey, &item.SourceProposalKey, &item.RetiredAt, &item.Version); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(target, &item.Target); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(provenance, &item.Provenance); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ReadWorks(ctx context.Context, org, project string) ([]works.Item, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	items, err := readWorkItems(ctx, s.pool, org, project)
	if err != nil {
		return nil, err
	}
	// Read through the site's projected condition. It is not copied into the
	// authoritative item, and rebuilds never use profile_rows as an input.
	rows, err := s.pool.Query(ctx, `SELECT part_id::text,key,value FROM profile_rows WHERE org_id=$1::uuid AND project_id=$2::uuid AND scope='site' AND key LIKE 'sys.%.condition'`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conditions := map[string]string{}
	for rows.Next() {
		var part, key, value string
		if err := rows.Scan(&part, &key, &value); err != nil {
			return nil, err
		}
		conditions[part+"/"+key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := []works.Item{}
	for _, item := range items {
		if item.RetiredAt != nil {
			continue
		}
		item.ExistingCondition = conditions[item.PartID+"/sys."+item.SystemID+".condition"]
		out = append(out, item)
	}
	return out, nil
}

func (s *Store) workCatalog() *knowledge.Catalog {
	if s.profileBuild != nil {
		return s.profileBuild.Catalog
	}
	return nil
}

// CreateWorkItem creates a user-owned coarse item; a duplicate is a conflict,
// never an implicit overwrite of a suggestion or another person's decision.
func (s *Store) CreateWorkItem(ctx context.Context, org, project, actor string, item works.Item) (works.Item, error) {
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
	site, err := userValueOwner(ctx, tx, org, project, item.PartID)
	if err != nil {
		return item, err
	}
	if err := works.Validate(item, s.workCatalog()); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidWork, err)
	}
	item.ID = works.CoarseID(project, item.PartID, item.SystemID)
	item.ProjectID = project
	item.SiteID = site
	item.ParentID = ""
	item.IsGroup = false
	item.CoarseKey = item.PartID + "|" + item.SystemID
	item.Origin = "user"
	item.ReviewStatus = "accepted_for_planning"
	item.Meaning = "stated"
	item.Inclusion = "included"
	item.UserTouched = true
	item.Provenance = works.Provenance{Actor: actor, Band: "user"}
	item.Version = 1
	if strings.TrimSpace(item.Title) == "" {
		sys, _ := s.workCatalog().System(item.SystemID)
		item.Title = sys.Label
	}
	if err := insertWork(ctx, tx, org, item); err != nil {
		return item, workError(err)
	}
	if item.ExistingCondition != "" {
		key := "sys." + item.SystemID + ".condition"
		tag, err := tx.Exec(ctx, `INSERT INTO profile_user_values(org_id,id,site_id,part_id,scope,key,value,user_id)
 VALUES($1::uuid,gen_random_uuid(),$2::uuid,$3::uuid,'site',$4,$5,$6::uuid)
 ON CONFLICT(org_id,site_id,part_id,key) WHERE scope='site' DO NOTHING`, org, site, item.PartID, key, item.ExistingCondition, actor)
		if err != nil {
			return item, err
		}
		if tag.RowsAffected() == 0 {
			var old *string
			if err := tx.QueryRow(ctx, `SELECT value FROM profile_user_values WHERE org_id=$1::uuid AND site_id=$2::uuid AND part_id=$3::uuid AND scope='site' AND key=$4`, org, site, item.PartID, key).Scan(&old); err != nil {
				return item, err
			}
			if old == nil || *old != item.ExistingCondition {
				return item, fmt.Errorf("%w: existing site condition differs; edit it with its version first", ErrInvalidWork)
			}
		} else if err := BumpRevision(ctx, tx, org, project, "profile_inputs"); err != nil {
			return item, err
		}
	}
	if err := s.finishWorksWrite(ctx, tx, org, project); err != nil {
		return item, err
	}
	return item, nil
}

func workError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return ErrWorkConflict
	}
	return err
}

func insertWork(ctx context.Context, tx pgx.Tx, org string, item works.Item) error {
	target, _ := json.Marshal(item.Target)
	prov, _ := json.Marshal(item.Provenance)
	_, err := tx.Exec(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,existing_condition_note,target,quantity,unit,origin,review_status,meaning,provenance,user_touched,coarse_key)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11,$12::numeric,$13,$14,$15,$16,$17,$18,$19)`,
		org, item.ID, item.ProjectID, item.SiteID, item.PartID, item.SystemID, item.Action, item.Inclusion, item.Title, item.ExistingConditionNote, target, item.Quantity, item.Unit, item.Origin, item.ReviewStatus, item.Meaning, prov, item.UserTouched, item.CoarseKey)
	return err
}

func workEvent(ctx context.Context, tx pgx.Tx, s *Store, org, project string) error {
	revisions, err := readRevisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"project_id": project, "revision": revisions.Works})
	_, err = appendEvent(ctx, s.q.WithTx(tx), org, "works", "", string(payload))
	return err
}

func (s *Store) finishWorksWrite(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
		return err
	}
	if err := workEvent(ctx, tx, s, org, project); err != nil {
		return err
	}
	if s.profileBuild != nil {
		if err := s.rebuildProfileTx(ctx, tx, org, project, *s.profileBuild); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// syncProposedWorks owns only untouched proposals. Retirement keeps identity
// and provenance; reappearing support revives that same coarse ID.
func (s *Store) syncProposedWorks(ctx context.Context, tx pgx.Tx, org, project, site string, desired []works.Item, cat *knowledge.Catalog) error {
	ids := make([]string, 0, len(desired))
	batch := &pgx.Batch{}
	type proposedRow struct {
		ID         string           `json:"id"`
		Part       string           `json:"part"`
		System     string           `json:"system"`
		Action     string           `json:"action"`
		Title      string           `json:"title"`
		Origin     string           `json:"origin"`
		Provenance works.Provenance `json:"provenance"`
		CoarseKey  string           `json:"coarse_key"`
	}
	proposed := make([]proposedRow, 0, len(desired))
	for _, item := range desired {
		ids = append(ids, item.ID)
		proposed = append(proposed, proposedRow{item.ID, item.PartID, item.SystemID, item.Action, item.Title, item.Origin, item.Provenance, item.CoarseKey})
	}
	if len(proposed) > 0 {
		raw, err := json.Marshal(proposed)
		if err != nil {
			return err
		}
		// One statement avoids per-item parse/bind and execution overhead on
		// every edit. The same conflict guard protects accepted/user work.
		batch.Queue(`INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,meaning,provenance,coarse_key)
 SELECT $1::uuid,p.id,$2::uuid,$3::uuid,p.part,p.system,p.action,'included',p.title,p.origin,'proposed','stated',p.provenance,p.coarse_key
 FROM jsonb_to_recordset($4::jsonb) AS p(id uuid,part uuid,system text,action text,title text,origin text,provenance jsonb,coarse_key text)
 ON CONFLICT(org_id,id) DO UPDATE SET action=EXCLUDED.action,title=EXCLUDED.title,origin=EXCLUDED.origin,provenance=EXCLUDED.provenance,retired_at=NULL,version=work_items.version+1
 WHERE work_items.review_status='proposed' AND NOT work_items.user_touched
 AND (work_items.action,work_items.title,work_items.origin,work_items.provenance,work_items.retired_at)
 IS DISTINCT FROM (EXCLUDED.action,EXCLUDED.title,EXCLUDED.origin,EXCLUDED.provenance,NULL::timestamptz)`, org, project, site, raw)
	}
	// A removed/deprecated catalogue ID is a review problem, not permission to
	// silently retire previously recorded scope.
	activeSystems := []string{}
	pending := cat.TopSystems()
	for len(pending) > 0 {
		sys := pending[0]
		pending = pending[1:]
		if sys.Status != "deprecated" {
			activeSystems = append(activeSystems, sys.ID)
		}
		pending = append(pending, cat.Children(sys.ID)...)
	}
	batch.Queue(`UPDATE work_items SET retired_at=now(),version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND parent_id IS NULL AND coarse_key IS NOT NULL AND review_status='proposed' AND NOT user_touched AND retired_at IS NULL AND NOT(id::text=ANY($3::text[])) AND system_id=ANY($4::text[]) AND NOT EXISTS(SELECT 1 FROM package_scope_items si WHERE si.org_id=work_items.org_id AND si.project_id=work_items.project_id AND si.work_item_id=work_items.id AND si.retired_at IS NULL) AND NOT EXISTS(SELECT 1 FROM project_delivery_items di WHERE di.org_id=work_items.org_id AND di.project_id=work_items.project_id AND di.work_item_id=work_items.id AND di.retired_at IS NULL AND `+deliveryOpenSQL+`)`, org, project, ids, activeSystems)
	result := tx.SendBatch(ctx, batch)
	changed := false
	for range batch.Len() {
		tag, err := result.Exec()
		if err != nil {
			result.Close()
			return err
		}
		changed = changed || tag.RowsAffected() > 0
	}
	if err := result.Close(); err != nil {
		return err
	}
	if changed {
		if err := BumpRevision(ctx, tx, org, project, "works"); err != nil {
			return err
		}
		return workEvent(ctx, tx, s, org, project)
	}
	return nil
}

func workFingerprint(items []works.Item) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		item.Version = 0
		// Edit time is audit metadata, not changed evidence or planning scope.
		item.Provenance.LastEditedAt = nil
		retired := item.RetiredAt != nil
		item.RetiredAt = nil
		raw, _ := json.Marshal(struct {
			Item    works.Item `json:"item"`
			Retired bool       `json:"retired"`
		}{item, retired})
		out = append(out, raw)
	}
	return out
}

// setWorkScope is the old scope-picker write, now backed solely by work_items.
func (s *Store) setWorkScope(ctx context.Context, org, project, part, actor string, choices map[string]*string) error {
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
	site, err := userValueOwner(ctx, tx, org, project, part)
	if err != nil {
		return err
	}
	var kind string
	err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT value FROM profile_user_values WHERE org_id=$1::uuid AND project_id=$2::uuid AND part_id=$3::uuid AND key='hdr.work_type'),
 (SELECT v.value FROM profile_user_values v JOIN project_parts p ON p.org_id=v.org_id AND p.id=v.part_id WHERE v.org_id=$1::uuid AND v.project_id=$2::uuid AND p.kind='whole' AND v.key='hdr.work_type'),'')`, org, project, part).Scan(&kind)
	if err != nil {
		return err
	}
	if kind == "" && s.profileBuild != nil {
		// A document may supply the work type without a user override. Read its
		// authoritative inputs rather than borrowing a stale projection row.
		snap, err := readSnapshot(ctx, tx, org, project)
		if err != nil {
			return err
		}
		rows := s.profileBuild.Compute(snap)
		whole := ""
		for _, p := range snap.Parts {
			if p.Kind == "whole" {
				whole = p.ID
			}
		}
		for _, wanted := range []string{whole, part} {
			for _, row := range rows {
				if row.PartID == wanted && row.Key == "hdr.work_type" && row.Value != "" && (row.Band == "user" || row.Band == "amber" || row.Band == "green") {
					kind = row.Value
				}
			}
		}
	}
	for key, value := range choices {
		system, ok := strings.CutPrefix(key, "scope.")
		if !ok || system == "" || (value != nil && *value != "in" && *value != "out") {
			return ErrInvalidWork
		}
		title := system
		if cat := s.workCatalog(); cat != nil {
			sys, ok := cat.System(system)
			if !ok || sys.Status == "deprecated" {
				return ErrInvalidWork
			}
			title = sys.Label
		}
		id := works.CoarseID(project, part, system)
		if value == nil {
			deliveryUsed, err := openDeliveryReference(ctx, tx, org, project, "work_item_id", id)
			if err != nil {
				return err
			}
			if deliveryUsed {
				return fmt.Errorf("%w: resolve open delivery records before resetting this work item", ErrInvalidWork)
			}
			var used bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND work_item_id=$3::uuid AND retired_at IS NULL)`, org, project, id).Scan(&used); err != nil {
				return err
			}
			if used {
				return fmt.Errorf("%w: reassign package scope before resetting this work item", ErrInvalidWork)
			}
			// Reset explicitly hands ownership back to the rebuild. Keep the ID for
			// future references instead of deleting the row.
			_, err = tx.Exec(ctx, `UPDATE work_items SET user_touched=false,review_status='proposed',origin='calculation',inclusion='included',retired_at=now(),version=version+1,provenance='{}' WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id)
		} else {
			inclusion := "included"
			if *value == "out" {
				inclusion = "excluded"
			}
			_, err = tx.Exec(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,user_touched,coarse_key,provenance)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,'user','accepted_for_planning',true,$5||'|'||$6,jsonb_build_object('actor',$10::text,'band','user'))
 ON CONFLICT(org_id,id) DO UPDATE SET inclusion=EXCLUDED.inclusion,origin='user',review_status='accepted_for_planning',user_touched=true,retired_at=NULL,version=work_items.version+1,provenance=work_items.provenance||EXCLUDED.provenance`,
				org, id, project, site, part, system, works.DefaultAction(s.workCatalog(), kind, ""), inclusion, title, actor)
		}
		if err != nil {
			return workError(err)
		}
	}
	return s.finishWorksWrite(ctx, tx, org, project)
}
