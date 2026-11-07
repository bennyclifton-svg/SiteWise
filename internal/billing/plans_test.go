package billing

import "testing"

func TestLookupUnknownIsFree(t *testing.T) {
	if got := Lookup("gold").ID; got != Free {
		t.Fatalf("unknown plan = %q, want free", got)
	}
	if got := Lookup(Team).Users; got != 5 {
		t.Fatalf("team users = %d, want 5", got)
	}
}

func TestEffectivePlan(t *testing.T) {
	cases := []struct {
		plan, status, want string
	}{
		{Starter, "active", Starter},
		{Team, "trialing", Team},
		// Stripe is still retrying the card: the plan stays.
		{Starter, "past_due", Starter},
		{Starter, "canceled", Free},
		{Starter, "unpaid", Free},
		{Starter, "incomplete", Free},
		{Starter, "incomplete_expired", Free},
		{Starter, "paused", Free},
		// An operator-set plan has no Stripe status.
		{Team, "", Team},
		{Free, "active", Free},
		{"gold", "active", Free},
	}
	for _, c := range cases {
		if got := Effective(c.plan, c.status); got != c.want {
			t.Errorf("Effective(%q, %q) = %q, want %q", c.plan, c.status, got, c.want)
		}
	}
}

func TestPricesAreTenMonthsForAYear(t *testing.T) {
	for _, p := range []Plan{Lookup(Starter), Lookup(Team)} {
		if p.AnnualCents != p.MonthlyCents*10 {
			t.Errorf("%s annual = %d, want ten months (%d)", p.ID, p.AnnualCents, p.MonthlyCents*10)
		}
	}
}

func TestProjectAllowance(t *testing.T) {
	if Lookup(Free).AllowsProjects(1) {
		t.Fatal("free plan allowed a second project")
	}
	if !Lookup(Free).AllowsProjects(0) {
		t.Fatal("free plan refused its first project")
	}
	if !Lookup(Team).AllowsProjects(10_000) {
		t.Fatal("team projects are unlimited")
	}
}
