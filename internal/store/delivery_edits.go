package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
	"sitewise/internal/profile"
)

type DeliveryPatch struct {
	delivery.Patch
	SourceRefs *[]profile.Source `json:"source_refs"`
}

func (s *Store) PatchDelivery(ctx context.Context, org, project, id, actor string, patch DeliveryPatch) (delivery.Item, error) {
	var item delivery.Item
	if patch.Version < 1 {
		return item, ErrInvalidDelivery
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
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(i) FROM project_delivery_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL`, org, project, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	if item.Version != patch.Version {
		return item, ErrVersionConflict
	}
	oldStatus, oldKind := item.Status, item.Kind
	patch.Apply(&item.Content)
	if err := delivery.Validate(item.Content); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
	}
	// Closed historical records may retain retired targets. Reopening or choosing
	// a new target must resolve every reference against live project records.
	if delivery.Open(item.Kind, item.Status) || patch.PackageID != nil || patch.WorkItemID != nil || patch.StageID != nil || patch.OwnerUserID != nil {
		if err := s.deliveryTargets(ctx, tx, org, project, item.Content); err != nil {
			return item, err
		}
	} else if err := s.validateDeliveryRisk(item.Content); err != nil {
		return item, err
	}
	provenance := map[string]any{}
	if patch.SourceRefs != nil {
		sources, err := resolveScopeSources(ctx, tx, org, project, *patch.SourceRefs)
		if err != nil {
			return item, err
		}
		provenance["sources"] = sources
	}
	provenance["last_edited_by"] = actor
	changes, _ := json.Marshal(provenance)
	retire := patch.Retired != nil && *patch.Retired
	if retire {
		var linked bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM delivery_dependencies WHERE org_id=$1::uuid AND project_id=$2::uuid AND (predecessor_id=$3::uuid OR successor_id=$3::uuid))`, org, project, id).Scan(&linked); err != nil {
			return item, err
		}
		if linked {
			return item, fmt.Errorf("%w: remove dependencies before retiring this record", ErrInvalidDelivery)
		}
	}
	err = tx.QueryRow(ctx, `UPDATE project_delivery_items SET kind=$4,title=$5,owner_text=$6,owner_user_id=NULLIF($7,'')::uuid,baseline_date=$8::text::date,target_date=$9::text::date,forecast_date=$10::text::date,actual_date=$11::text::date,status=$12,as_of=$13::text::date,package_id=NULLIF($14,'')::uuid,work_item_id=NULLIF($15,'')::uuid,stage_id=NULLIF($16,'')::uuid,details=$17::jsonb,origin='user',review_status='accepted_for_planning',verified_by=NULL,verified_at=NULL,verification_basis=NULL,
 provenance=provenance||$18::jsonb||jsonb_build_object('last_edited_at',now())||CASE WHEN $19<>$12 OR $20<>$4 THEN jsonb_build_object('status_history',COALESCE(provenance->'status_history','[]'::jsonb)||jsonb_build_array(jsonb_build_object('from_kind',$20::text,'from',$19::text,'kind',$4::text,'status',$12::text,'actor',$21::text,'at',now()))) ELSE '{}'::jsonb END,
 retired_at=CASE WHEN $22 THEN now() ELSE retired_at END,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid RETURNING to_jsonb(project_delivery_items)`, org, project, id, item.Kind, item.Title, item.OwnerText, item.OwnerUserID, item.BaselineDate, item.TargetDate, item.ForecastDate, item.ActualDate, item.Status, item.AsOf, item.PackageID, item.WorkItemID, item.StageID, item.Details, changes, oldStatus, oldKind, actor, retire).Scan(&raw)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	return item, s.finishDeliveryWrite(ctx, tx, org, project)
}
