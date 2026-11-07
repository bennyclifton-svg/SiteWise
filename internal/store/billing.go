package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/billing"
)

// OrgBilling is the org's copy of its Stripe state.
type OrgBilling struct {
	Plan              string
	Status            string
	Interval          string
	PaidUntil         *time.Time
	CancelAtPeriodEnd bool
	CustomerID        string
	SubscriptionID    string
}

// Effective is the plan whose limits apply now.
func (b OrgBilling) Effective() billing.Plan {
	return billing.Lookup(billing.Effective(b.Plan, b.Status))
}

// Usage is what an org has used against its plan.
type Usage struct {
	DocumentsThisMonth int   `json:"documents_this_month"`
	StorageBytes       int64 `json:"storage_bytes"`
	Projects           int   `json:"projects"`
	// Seats counts members plus invites still open.
	Seats int `json:"seats"`
}

// OrgBilling reads the org's plan and Stripe state.
func (s *Store) OrgBilling(ctx context.Context, orgID string) (OrgBilling, error) {
	var b OrgBilling
	var customer, sub *string
	err := s.pool.QueryRow(ctx, `
		SELECT plan, billing_status, billing_interval, paid_until, cancel_at_period_end,
		       stripe_customer_id, stripe_subscription_id
		FROM orgs WHERE id = $1`, orgID).
		Scan(&b.Plan, &b.Status, &b.Interval, &b.PaidUntil, &b.CancelAtPeriodEnd, &customer, &sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrgBilling{}, ErrNotFound
	}
	if customer != nil {
		b.CustomerID = *customer
	}
	if sub != nil {
		b.SubscriptionID = *sub
	}
	return b, err
}

// Usage counts the org's documents this month, stored bytes, projects and
// seats in one round trip.
func (s *Store) Usage(ctx context.Context, orgID string, now time.Time) (Usage, error) {
	var u Usage
	err := s.pool.QueryRow(ctx, `
		SELECT
		  coalesce((SELECT documents FROM org_usage WHERE org_id = $1 AND month = $2), 0),
		  coalesce((SELECT sum(byte_size) FROM files WHERE org_id = $1), 0)::bigint,
		  (SELECT count(*) FROM projects WHERE org_id = $1),
		  (SELECT count(*) FROM memberships WHERE org_id = $1)
		  + (SELECT count(*) FROM invites WHERE org_id = $1 AND consumed_at IS NULL AND expires_at > $3)`,
		orgID, billing.MonthStart(now), now).
		Scan(&u.DocumentsThisMonth, &u.StorageBytes, &u.Projects, &u.Seats)
	return u, err
}

// CountDocument adds one read document to this month's count.
func (s *Store) CountDocument(ctx context.Context, orgID string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO org_usage (org_id, month, documents) VALUES ($1, $2, 1)
		ON CONFLICT (org_id, month) DO UPDATE SET documents = org_usage.documents + 1`,
		orgID, billing.MonthStart(now))
	return err
}

// SetPlan is the operator's override for a comped or test org. It carries no
// Stripe status, so the plan applies until changed again or a webhook lands.
func (s *Store) SetPlan(ctx context.Context, orgID, plan string) error {
	if plan != billing.Free && !billing.Paid(plan) {
		return errors.New("unknown plan")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE orgs SET plan = $2, billing_status = '' WHERE id = $1`, orgID, plan)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// StripeOutcome says what ApplyStripe did, for the log.
type StripeOutcome string

const (
	StripeApplied   StripeOutcome = "applied"
	StripeDuplicate StripeOutcome = "duplicate"
	StripeStale     StripeOutcome = "stale"
	StripeNoOrg     StripeOutcome = "no_org"
	StripeIgnored   StripeOutcome = "ignored"
)

// ApplyStripe records a verified event once and writes its effect on the org.
// plan and interval come from the event's price, mapped by the caller; an
// empty plan on a live subscription leaves the plan as it was. Events older
// than the newest applied one change nothing, so late delivery is harmless.
func (s *Store) ApplyStripe(ctx context.Context, e billing.Event, plan string) (StripeOutcome, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `INSERT INTO stripe_events (id, type) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, e.ID, e.Type)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return StripeDuplicate, nil
	}
	outcome, err := applyStripe(ctx, tx, e, plan)
	if err != nil {
		return "", err
	}
	return outcome, tx.Commit(ctx)
}

func applyStripe(ctx context.Context, tx pgx.Tx, e billing.Event, plan string) (StripeOutcome, error) {
	if !e.Handled() {
		return StripeIgnored, nil
	}
	var orgID string
	var lastEvent *time.Time
	err := tx.QueryRow(ctx, `
		SELECT id::text, billing_event_at FROM orgs
		WHERE ($1 <> '' AND id::text = $1) OR ($1 = '' AND $2 <> '' AND stripe_customer_id = $2)
		FOR UPDATE`, uuidOrEmpty(e.OrgID), e.CustomerID).Scan(&orgID, &lastEvent)
	if errors.Is(err, pgx.ErrNoRows) {
		return StripeNoOrg, nil
	}
	if err != nil {
		return "", err
	}
	if e.Type == "checkout.session.completed" {
		// Links the customer to the org; the subscription event sets the plan.
		_, err := tx.Exec(ctx, `
			UPDATE orgs SET stripe_customer_id = coalesce(stripe_customer_id, nullif($2, '')),
			                stripe_subscription_id = coalesce(nullif($3, ''), stripe_subscription_id)
			WHERE id = $1`, orgID, e.CustomerID, e.SubscriptionID)
		return StripeApplied, err
	}
	if lastEvent != nil && e.Created.Before(*lastEvent) {
		return StripeStale, nil
	}
	var periodEnd *time.Time
	if !e.PeriodEnd.IsZero() {
		periodEnd = &e.PeriodEnd
	}
	if e.Type == "customer.subscription.deleted" {
		_, err := tx.Exec(ctx, `
			UPDATE orgs SET plan = 'free', billing_status = 'canceled', billing_interval = '',
			                cancel_at_period_end = false, stripe_subscription_id = NULL,
			                paid_until = $2, billing_event_at = $3
			WHERE id = $1`, orgID, periodEnd, e.Created)
		return StripeApplied, err
	}
	interval := e.Interval
	if interval != "month" && interval != "year" {
		interval = ""
	}
	_, err = tx.Exec(ctx, `
		UPDATE orgs SET plan = coalesce(nullif($2, ''), plan), billing_status = $3,
		                billing_interval = $4, paid_until = $5, cancel_at_period_end = $6,
		                stripe_customer_id = coalesce(stripe_customer_id, nullif($7, '')),
		                stripe_subscription_id = nullif($8, ''), billing_event_at = $9
		WHERE id = $1`, orgID, plan, e.Status, interval, periodEnd, e.CancelAtPeriodEnd, e.CustomerID, e.SubscriptionID, e.Created)
	return StripeApplied, err
}

// uuidOrEmpty keeps a malformed metadata value from reaching a uuid cast.
func uuidOrEmpty(s string) string {
	if len(s) != 36 {
		return ""
	}
	for i, c := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return ""
			}
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'):
		default:
			return ""
		}
	}
	return s
}
