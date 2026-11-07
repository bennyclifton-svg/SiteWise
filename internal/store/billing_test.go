package store_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"sitewise/internal/billing"
	"sitewise/internal/store"
)

// billingAccount makes a fresh password account and removes it afterwards.
func billingAccount(t *testing.T, st *store.Store) store.Session {
	t.Helper()
	email := fmt.Sprintf("billing-%d@example.test", time.Now().UnixNano())
	s, err := st.CreateAccount(context.Background(), email, []byte("hash"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), s.OrgID) })
	return s
}

func newID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func TestUsageCountsPerOrgAndMonth(t *testing.T) {
	ctx := context.Background()
	st := openHealthStore(t)
	a, b := billingAccount(t, st), billingAccount(t, st)
	now := time.Date(2026, 11, 30, 13, 0, 0, 0, time.UTC) // 1 Dec in Sydney
	for range 3 {
		if err := st.CountDocument(ctx, a.OrgID, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.CreateProject(ctx, a.OrgID, newID(t), "One"); err != nil {
		t.Fatal(err)
	}
	ua, err := st.Usage(ctx, a.OrgID, now)
	if err != nil {
		t.Fatal(err)
	}
	if ua.DocumentsThisMonth != 3 || ua.Projects != 1 || ua.Seats != 1 {
		t.Fatalf("usage a %+v", ua)
	}
	// The previous Sydney month is a different allowance.
	if u, _ := st.Usage(ctx, a.OrgID, now.Add(-24*time.Hour)); u.DocumentsThisMonth != 0 {
		t.Fatalf("november count %d", u.DocumentsThisMonth)
	}
	if ub, _ := st.Usage(ctx, b.OrgID, now); ub.DocumentsThisMonth != 0 || ub.Projects != 0 {
		t.Fatalf("other org usage %+v", ub)
	}
}

func TestApplyStripeLifecycle(t *testing.T) {
	ctx := context.Background()
	st := openHealthStore(t)
	a, other := billingAccount(t, st), billingAccount(t, st)
	n := time.Now().UnixNano()
	ev := func(id, typ string, at int64) billing.Event {
		return billing.Event{ID: fmt.Sprintf("evt_%d_%s", n, id), Type: typ, Created: time.Unix(at, 0), OrgID: a.OrgID,
			CustomerID: fmt.Sprintf("cus_%d", n), SubscriptionID: "sub_1"}
	}
	apply := func(e billing.Event, plan string, want store.StripeOutcome) {
		t.Helper()
		got, err := st.ApplyStripe(ctx, e, plan)
		if err != nil || got != want {
			t.Fatalf("%s: outcome %q err %v, want %q", e.ID, got, err, want)
		}
	}
	checkout := ev("checkout", "checkout.session.completed", 100)
	apply(checkout, "", store.StripeApplied)
	apply(checkout, "", store.StripeDuplicate)

	created := ev("created", "customer.subscription.created", 100)
	created.Status, created.Interval, created.PeriodEnd = "active", "month", time.Unix(5000, 0)
	created.OrgID = "" // found by customer id instead
	apply(created, billing.Starter, store.StripeApplied)
	b, _ := st.OrgBilling(ctx, a.OrgID)
	if b.Plan != billing.Starter || b.Status != "active" || b.Interval != "month" || b.CustomerID == "" || b.Effective().ID != billing.Starter {
		t.Fatalf("after created %+v", b)
	}
	if ob, _ := st.OrgBilling(ctx, other.OrgID); ob.Plan != billing.Free {
		t.Fatalf("other org changed: %+v", ob)
	}

	upgraded := ev("upgraded", "customer.subscription.updated", 300)
	upgraded.Status, upgraded.Interval = "past_due", "year"
	apply(upgraded, billing.Team, store.StripeApplied)
	// An older event arriving late changes nothing.
	late := ev("late", "customer.subscription.updated", 200)
	late.Status = "active"
	apply(late, billing.Starter, store.StripeStale)
	b, _ = st.OrgBilling(ctx, a.OrgID)
	if b.Plan != billing.Team || b.Status != "past_due" || b.Effective().ID != billing.Team {
		t.Fatalf("after late %+v", b)
	}

	deleted := ev("deleted", "customer.subscription.deleted", 400)
	deleted.Status = "canceled"
	apply(deleted, "", store.StripeApplied)
	b, _ = st.OrgBilling(ctx, a.OrgID)
	if b.Plan != billing.Free || b.SubscriptionID != "" || b.Effective().ID != billing.Free {
		t.Fatalf("after deleted %+v", b)
	}

	stray := billing.Event{ID: fmt.Sprintf("evt_%d_stray", n), Type: "customer.subscription.updated", OrgID: "not-a-uuid", CustomerID: "cus_unknown"}
	apply(stray, billing.Team, store.StripeNoOrg)
	apply(billing.Event{ID: fmt.Sprintf("evt_%d_inv", n), Type: "invoice.paid"}, "", store.StripeIgnored)
}

func TestSetPlan(t *testing.T) {
	ctx := context.Background()
	st := openHealthStore(t)
	a := billingAccount(t, st)
	if err := st.SetPlan(ctx, a.OrgID, billing.Team); err != nil {
		t.Fatal(err)
	}
	if b, _ := st.OrgBilling(ctx, a.OrgID); b.Effective().ID != billing.Team {
		t.Fatalf("set plan %+v", b)
	}
	if err := st.SetPlan(ctx, a.OrgID, "gold"); err == nil {
		t.Fatal("unknown plan accepted")
	}
}
