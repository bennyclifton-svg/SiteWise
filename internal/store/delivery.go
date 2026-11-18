package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/delivery"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
)

var ErrInvalidDelivery = errors.New("invalid delivery record")

type DeliveryInput struct {
	delivery.Content
	SourceRefs []profile.Source `json:"source_refs"`
}

func (s *Store) CreateDelivery(ctx context.Context, org, project, actor string, input DeliveryInput) (delivery.Item, error) {
	var item delivery.Item
	if input.Status == "" {
		input.Status = delivery.DefaultStatus(input.Kind)
	}
	if len(input.Details) == 0 {
		input.Details = json.RawMessage(`{}`)
	}
	if err := delivery.Validate(input.Content); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
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
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return item, err
	}
	if !exists {
		return item, ErrNotFound
	}
	if err := s.deliveryTargets(ctx, tx, org, project, input.Content); err != nil {
		return item, err
	}
	if err := s.registerFill(ctx, tx, org, project, &input.Content, nil, input.TargetDate != nil); err != nil {
		return item, err
	}
	number, err := nextNumber(ctx, tx, org, project, input.Kind)
	if err != nil {
		return item, err
	}
	sources, err := resolveScopeSources(ctx, tx, org, project, input.SourceRefs)
	if err != nil {
		return item, err
	}
	provenance, _ := json.Marshal(map[string]any{"actor": actor, "sources": sources})
	var raw []byte
	err = tx.QueryRow(ctx, `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,owner_text,owner_user_id,baseline_date,target_date,forecast_date,actual_date,status,as_of,package_id,work_item_id,stage_id,details,origin,review_status,provenance,number,system_id,determinant_id)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,NULLIF($7,'')::uuid,$8::text::date,$9::text::date,$10::text::date,$11::text::date,$12,$13::text::date,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,$17::jsonb,'user','accepted_for_planning',$18::jsonb||jsonb_build_object('at',now(),'initial_status',$12::text),$19,NULLIF($20,''),NULLIF($21,'')) RETURNING to_jsonb(project_delivery_items)`, org, newID(), project, input.Kind, input.Title, input.OwnerText, input.OwnerUserID, input.BaselineDate, input.TargetDate, input.ForecastDate, input.ActualDate, input.Status, input.AsOf, input.PackageID, input.WorkItemID, input.StageID, input.Details, provenance, number, input.SystemID, input.DeterminantID).Scan(&raw)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	return item, s.finishDeliveryWrite(ctx, tx, org, project)
}

func (s *Store) deliveryTargets(ctx context.Context, tx pgx.Tx, org, project string, c delivery.Content) error {
	var uuid pgtype.UUID
	for _, id := range []string{c.OwnerUserID, c.PackageID, c.WorkItemID, c.StageID} {
		if id != "" && uuid.Scan(id) != nil {
			return ErrInvalidDelivery
		}
	}
	if c.OwnerUserID != "" {
		if err := packageActor(ctx, tx, org, c.OwnerUserID); err != nil {
			return err
		}
	}
	if c.PackageID != "" {
		if _, err := readPackage(ctx, tx, org, project, c.PackageID); err != nil {
			return err
		}
	}
	if err := scopeTargets(ctx, tx, org, project, c.PackageID, procurement.ScopeContent{WorkItemID: c.WorkItemID, StageID: c.StageID}); err != nil {
		return err
	}
	return s.validateDeliveryRisk(c)
}

func (s *Store) validateDeliveryRisk(c delivery.Content) error {
	if c.Kind == "risk" {
		var d delivery.RiskDetails
		if err := json.Unmarshal(c.Details, &d); err != nil {
			return ErrInvalidDelivery
		}
		if d.UCRecordID != "" {
			found := false
			if cat := s.workCatalog(); cat != nil {
				for _, record := range cat.UnforeseenConditions() {
					if record.ID == d.UCRecordID {
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("%w: unknown unforeseen condition", ErrInvalidDelivery)
			}
		}
	}
	return nil
}

func (s *Store) finishDeliveryWrite(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := BumpRevision(ctx, tx, org, project, "delivery"); err != nil {
		return err
	}
	revisions, err := readRevisions(ctx, tx, org, project)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"project_id": project, "revision": revisions.Delivery})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), org, "delivery", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
