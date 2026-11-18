package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/delivery"
	"sitewise/internal/profile"
)

type DeliveryPatch struct {
	delivery.Patch
	SourceRefs *[]profile.Source `json:"source_refs"`
	// AcceptReading accepts one possible answer: the RFI becomes Answered,
	// the document joins its answer documents and the date answered is the
	// document's upload date, all set here in Go. It is the same edit the
	// PM could type field by field.
	AcceptReading string `json:"accept_reading"`
	// DismissReadings sets possible answers aside. On its own it leaves the
	// row and its version as they are.
	DismissReadings []string `json:"dismiss_readings"`
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
	if len(patch.DismissReadings) > 100 {
		return item, ErrInvalidDelivery
	}
	if patch.Empty() && patch.SourceRefs == nil && patch.AcceptReading == "" && len(patch.DismissReadings) > 0 {
		if err := dismissReadings(ctx, tx, org, project, id, patch.DismissReadings); err != nil {
			return item, err
		}
		return item, s.finishDeliveryWrite(ctx, tx, org, project)
	}
	if item.Version != patch.Version {
		return item, ErrVersionConflict
	}
	oldStatus, oldKind := item.Status, item.Kind
	before := item.Content
	if patch.AcceptReading != "" {
		if err := acceptReading(ctx, tx, org, project, &item, patch.AcceptReading, &patch.Patch); err != nil {
			return item, err
		}
	}
	patch.Apply(&item.Content)
	if err := delivery.CheckKindChange(oldKind, item.Kind); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
	}
	if err := delivery.Validate(item.Content); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
	}
	if err := s.registerFill(ctx, tx, org, project, &item.Content, &before, patch.TargetDate.Present); err != nil {
		return item, err
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
	err = tx.QueryRow(ctx, `UPDATE project_delivery_items SET kind=$4,title=$5,owner_text=$6,owner_user_id=NULLIF($7,'')::uuid,baseline_date=$8::text::date,target_date=$9::text::date,forecast_date=$10::text::date,actual_date=$11::text::date,status=$12,as_of=$13::text::date,package_id=NULLIF($14,'')::uuid,work_item_id=NULLIF($15,'')::uuid,stage_id=NULLIF($16,'')::uuid,details=$17::jsonb,system_id=NULLIF($23,''),determinant_id=NULLIF($24,''),origin='user',review_status='accepted_for_planning',verified_by=NULL,verified_at=NULL,verification_basis=NULL,
 provenance=provenance||$18::jsonb||jsonb_build_object('last_edited_at',now())||CASE WHEN $19<>$12 OR $20<>$4 THEN jsonb_build_object('status_history',COALESCE(provenance->'status_history','[]'::jsonb)||jsonb_build_array(jsonb_build_object('from_kind',$20::text,'from',$19::text,'kind',$4::text,'status',$12::text,'actor',$21::text,'at',now()))) ELSE '{}'::jsonb END,
 retired_at=CASE WHEN $22 THEN now() ELSE retired_at END,version=version+1 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid RETURNING to_jsonb(project_delivery_items)`, org, project, id, item.Kind, item.Title, item.OwnerText, item.OwnerUserID, item.BaselineDate, item.TargetDate, item.ForecastDate, item.ActualDate, item.Status, item.AsOf, item.PackageID, item.WorkItemID, item.StageID, item.Details, changes, oldStatus, oldKind, actor, retire, item.SystemID, item.DeterminantID).Scan(&raw)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	if err := settleReadings(ctx, tx, org, project, item); err != nil {
		return item, err
	}
	if err := dismissReadings(ctx, tx, org, project, id, patch.DismissReadings); err != nil {
		return item, err
	}
	return item, s.finishDeliveryWrite(ctx, tx, org, project)
}

// acceptReading turns one proposed reading into the fields an accepting
// edit sets. The values come from the reading and the document, never
// from the caller.
func acceptReading(ctx context.Context, tx pgx.Tx, org, project string, item *delivery.Item, readingID string, p *delivery.Patch) error {
	var uuid pgtype.UUID
	if uuid.Scan(readingID) != nil {
		return ErrInvalidDelivery
	}
	var doc, uploaded string
	err := tx.QueryRow(ctx, `SELECT r.document_id::text, to_char(d.created_at AT TIME ZONE 'Australia/Sydney','YYYY-MM-DD')
 FROM delivery_readings r JOIN documents d ON d.org_id=r.org_id AND d.id=r.document_id
 WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.item_id=$3::uuid AND r.id=$4::uuid AND r.status='proposed'`, org, project, item.ID, readingID).Scan(&doc, &uploaded)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if item.Kind != "rfi" {
		return fmt.Errorf("%w: only an RFI takes an answer", ErrInvalidDelivery)
	}
	d := delivery.RFI(item.Content)
	if !slices.Contains(d.AnswerDocuments, doc) {
		d.AnswerDocuments = append(d.AnswerDocuments, doc)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	answered := "answered"
	p.Status, p.Details = &answered, raw
	if !p.ActualDate.Present && item.ActualDate == nil {
		p.ActualDate = delivery.DateChange{Present: true, Value: &uploaded}
	}
	return nil
}

// settleReadings marks accepted every proposed reading whose document an
// answered RFI now names as an answer, in the same transaction as the edit.
func settleReadings(ctx context.Context, tx pgx.Tx, org, project string, item delivery.Item) error {
	if item.Kind != "rfi" || item.Status != "answered" {
		return nil
	}
	docs := delivery.RFI(item.Content).AnswerDocuments
	if len(docs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `UPDATE delivery_readings SET status='accepted' WHERE org_id=$1::uuid AND project_id=$2::uuid AND item_id=$3::uuid AND status='proposed' AND document_id::text = ANY($4::text[])`, org, project, item.ID, docs)
	return err
}

func dismissReadings(ctx context.Context, tx pgx.Tx, org, project, item string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var uuid pgtype.UUID
	for _, id := range ids {
		if uuid.Scan(id) != nil {
			return ErrInvalidDelivery
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE delivery_readings SET status='dismissed' WHERE org_id=$1::uuid AND project_id=$2::uuid AND item_id=$3::uuid AND status='proposed' AND id::text = ANY($4::text[])`, org, project, item, ids)
	if err == nil && tag.RowsAffected() != int64(len(ids)) {
		return ErrNotFound
	}
	return err
}
