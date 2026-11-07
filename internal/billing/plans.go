// Package billing holds the plans, their limits and the Stripe client.
// Nothing here calls Jev. See docs/design/2026-11-07-billing.md.
package billing

// Plan identifiers, stored on orgs.plan.
const (
	Free    = "free"
	Starter = "starter"
	Team    = "team"
)

// Plan is what an org may use. Prices are AUD cents excluding GST.
// Projects 0 means unlimited; every other limit is a hard cap.
type Plan struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	MonthlyCents      int64  `json:"monthly_cents"`
	AnnualCents       int64  `json:"annual_cents"`
	Projects          int    `json:"projects"`
	DocumentsPerMonth int    `json:"documents_per_month"`
	StorageBytes      int64  `json:"storage_bytes"`
	Users             int    `json:"users"`
}

const gb = int64(1) << 30

// The caps are provisional until per-document Jev cost is measured
// (design note, Plans). Annual is ten months: two months free.
var plans = map[string]Plan{
	Free:    {ID: Free, Name: "Free", Projects: 1, DocumentsPerMonth: 50, StorageBytes: gb / 2, Users: 1},
	Starter: {ID: Starter, Name: "Starter", MonthlyCents: 5000, AnnualCents: 50000, Projects: 5, DocumentsPerMonth: 500, StorageBytes: 10 * gb, Users: 1},
	Team:    {ID: Team, Name: "Team", MonthlyCents: 20000, AnnualCents: 200000, Projects: 0, DocumentsPerMonth: 3000, StorageBytes: 50 * gb, Users: 5},
}

// Plans lists the plans in price order, for the account page.
func Plans() []Plan { return []Plan{plans[Free], plans[Starter], plans[Team]} }

// Lookup returns the plan, or Free for anything unknown.
func Lookup(id string) Plan {
	if p, ok := plans[id]; ok {
		return p
	}
	return plans[Free]
}

// Paid reports whether id is a plan sold through Stripe.
func Paid(id string) bool { return id == Starter || id == Team }

// Effective is the plan whose limits apply. A Stripe subscription that is
// active, trialing or past due (Stripe still retrying) keeps its plan; any
// other Stripe status falls back to Free. An empty status is an operator-set
// plan with no subscription.
func Effective(plan, status string) string {
	if _, ok := plans[plan]; !ok {
		return Free
	}
	switch status {
	case "", "active", "trialing", "past_due":
		return plan
	}
	return Free
}

// AllowsProjects reports whether an org with existing projects may add one.
func (p Plan) AllowsProjects(existing int) bool {
	return p.Projects == 0 || existing < p.Projects
}
