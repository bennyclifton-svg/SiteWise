package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
)

// DeliveryTerms are the project's optional terms (correspondence plan,
// section 18): each is nil until the PM sets it. Nothing needs one to raise
// an item.
type DeliveryTerms struct {
	RFIDays          *int
	EOTDays          *int
	AgreedCompletion *string
}

// Periods are the response periods in force, each with where it came from.
type Periods struct {
	RFIDays    int  // working days; also a letter's
	RFITypical bool // SiteWise's default, not the project's
	EOTDays    int  // calendar days
	EOTTypical bool
}

// Resolve is the one place an unset term falls back to SiteWise's default,
// so no date calculation carries a branch of its own.
func (t DeliveryTerms) Resolve() Periods {
	p := Periods{RFIDays: delivery.DefaultResponseDays, RFITypical: true, EOTDays: delivery.DefaultEOTResponseDays, EOTTypical: true}
	if t.RFIDays != nil {
		p.RFIDays, p.RFITypical = *t.RFIDays, false
	}
	if t.EOTDays != nil {
		p.EOTDays, p.EOTTypical = *t.EOTDays, false
	}
	return p
}

// deliveryTerms reads the project's terms; all nil when none are set.
func deliveryTerms(ctx context.Context, tx pgx.Tx, org, project string) (DeliveryTerms, error) {
	var t DeliveryTerms
	var rfi, eot *int16
	var agreed *time.Time
	err := tx.QueryRow(ctx, `SELECT rfi_response_days, eot_response_days, agreed_completion FROM delivery_settings WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project).Scan(&rfi, &eot, &agreed)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	if rfi != nil {
		n := int(*rfi)
		t.RFIDays = &n
	}
	if eot != nil {
		n := int(*eot)
		t.EOTDays = &n
	}
	if agreed != nil {
		d := agreed.Format("2006-01-02")
		t.AgreedCompletion = &d
	}
	return t, nil
}

// TermsPatch is a change to the terms. A field left out keeps its value; a
// field present with a nil value clears it (back to the default, or to
// unknown for the completion date).
type TermsPatch struct {
	RFIDays          Optional[int]
	EOTDays          Optional[int]
	AgreedCompletion Optional[string]
}

// Optional is a field that may be absent, present and null, or set.
type Optional[T any] struct {
	Present bool
	Value   *T
}

// SetDeliveryTerms applies a change to the project's terms in one undoable
// write. Open items whose due date the PM did not type keep the date they
// have: a new period applies to items received or re-dated after it.
func (s *Store) SetDeliveryTerms(ctx context.Context, org, project, actor string, p TermsPatch) error {
	if !p.RFIDays.Present && !p.EOTDays.Present && !p.AgreedCompletion.Present {
		return fmt.Errorf("%w: nothing to set", ErrInvalidDelivery)
	}
	if v := p.RFIDays.Value; v != nil && (*v < 1 || *v > 60) {
		return fmt.Errorf("%w: the response time is 1 to 60 working days", ErrInvalidDelivery)
	}
	if v := p.EOTDays.Value; v != nil && (*v < 1 || *v > 120) {
		return fmt.Errorf("%w: the EOT response time is 1 to 120 days", ErrInvalidDelivery)
	}
	if v := p.AgreedCompletion.Value; v != nil {
		if _, err := time.Parse("2006-01-02", *v); err != nil {
			return fmt.Errorf("%w: the agreed completion date uses YYYY-MM-DD", ErrInvalidDelivery)
		}
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
	t, err := deliveryTerms(ctx, tx, org, project)
	if err != nil {
		return err
	}
	if p.RFIDays.Present {
		t.RFIDays = p.RFIDays.Value
	}
	if p.EOTDays.Present {
		t.EOTDays = p.EOTDays.Value
	}
	if p.AgreedCompletion.Present {
		t.AgreedCompletion = p.AgreedCompletion.Value
	}
	// IS DISTINCT FROM keeps a no-op from counting as a change for Undo.
	_, err = tx.Exec(ctx, `INSERT INTO delivery_settings(org_id,project_id,rfi_response_days,eot_response_days,agreed_completion)
 SELECT org_id,id,$3,$4,$5::text::date FROM projects WHERE org_id=$1::uuid AND id=$2::uuid
 ON CONFLICT (org_id,project_id) DO UPDATE SET rfi_response_days=EXCLUDED.rfi_response_days,eot_response_days=EXCLUDED.eot_response_days,agreed_completion=EXCLUDED.agreed_completion
 WHERE (delivery_settings.rfi_response_days,delivery_settings.eot_response_days,delivery_settings.agreed_completion) IS DISTINCT FROM (EXCLUDED.rfi_response_days,EXCLUDED.eot_response_days,EXCLUDED.agreed_completion)`,
		org, project, t.RFIDays, t.EOTDays, t.AgreedCompletion)
	if err != nil {
		return err
	}
	return s.finishDeliveryWrite(ctx, tx, org, project)
}

// Completion is the agreed completion date and the date after the days
// granted on EOT claims, computed at read. Unknown stays unknown: with no
// agreed date the days granted are still totalled and nothing moves.
type Completion struct {
	Agreed      *string `json:"agreed,omitempty"`
	DaysGranted int     `json:"days_granted"`
	// Adjusted is Agreed plus DaysGranted calendar days; nil with no Agreed.
	Adjusted *string `json:"adjusted,omitempty"`
	// Claims is how many EOT claims carry a standing grant.
	Claims int `json:"claims"`
}

// completionOf totals the standing grants over the register's live rows.
func completionOf(items []delivery.Item, agreed *string) Completion {
	c := Completion{Agreed: agreed}
	for _, it := range items {
		if n := delivery.DaysGranted(it.Content); n > 0 {
			c.DaysGranted += n
			c.Claims++
		}
	}
	c.Adjusted = delivery.AddDays(agreed, c.DaysGranted)
	return c
}
