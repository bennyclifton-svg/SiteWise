package profile_test

import (
	"fmt"
	"testing"

	"sitewise/internal/profile"
)

// votes returns n passages reading value for question q, each its own document.
func votes(q, value string, n int, prefix string) []profile.Fact {
	var out []profile.Fact
	for i := 0; i < n; i++ {
		out = append(out, jevFact(q, value, fmt.Sprintf("%s%d", prefix, i), 0.9))
	}
	return out
}

func reconcileVotes(t *testing.T, q string, groups ...[]profile.Fact) profile.Row {
	t.Helper()
	var facts []profile.Fact
	for _, g := range groups {
		facts = append(facts, g...)
	}
	in := input(facts...)
	in.Thresholds.Green = map[string]*float64{"header": conf(0.5), "determinant": conf(0.5)}
	return row(t, profile.Reconcile(in, repoCatalog(t)), whole, q)
}

// F15: a choice with one dissenting passage settles on its leading value when
// the evidence is lopsided, stays amber and keeps the dissent visible.
func TestLopsidedChoiceSettlesAmberWithDissentVisible(t *testing.T) {
	cases := []struct {
		q, lead, other string
		n, m           int
	}{
		{"hdr.work_type", "extend", "refurb", 42, 5},
		{"det.heritage_status", "conservation_area", "none", 14, 1},
		{"fact.contract_basis", "lump_sum", "schedule_of_rates", 13, 3},
		{"det.existing_building", "stated_true", "stated_false", 3, 1},
		{"hdr.work_type", "extend", "refurb", 3, 1}, // the floor: 3 sources, exactly 3x
	}
	for _, c := range cases {
		r := reconcileVotes(t, c.q, votes(c.q, c.lead, c.n, "a"), votes(c.q, c.other, c.m, "b"))
		if r.Value != c.lead || r.Band != "amber" {
			t.Fatalf("%s %d:%d: want amber %s, got %+v", c.q, c.n, c.m, c.lead, r)
		}
		if len(r.Alternatives) != 1 || r.Alternatives[0].Value != c.other || len(r.Alternatives[0].Sources) != c.m {
			t.Fatalf("%s: dissent not visible: %+v", c.q, r.Alternatives)
		}
	}
}

func TestCloseChoiceStaysConflict(t *testing.T) {
	cases := []struct {
		q, lead, other string
		n, m           int
	}{
		{"det.existing_building", "stated_true", "stated_false", 2, 1}, // fewer than 3 sources
		{"hdr.work_type", "extend", "refurb", 5, 2},                    // under 3x
		{"hdr.work_type", "extend", "refurb", 1, 1},
	}
	for _, c := range cases {
		r := reconcileVotes(t, c.q, votes(c.q, c.lead, c.n, "a"), votes(c.q, c.other, c.m, "b"))
		if r.Value != "" || r.Band != "red" || len(r.Alternatives) != 2 {
			t.Fatalf("%s %d:%d: want red conflict, got %+v", c.q, c.n, c.m, r)
		}
	}
}

// Three values: the leader must beat all the others combined, three times over.
func TestLopsidedRuleCountsAllOtherValuesTogether(t *testing.T) {
	q := "hdr.work_type"
	r := reconcileVotes(t, q, votes(q, "extend", 8, "a"), votes(q, "refurb", 2, "b"), votes(q, "new", 1, "c"))
	if r.Band != "red" {
		t.Fatalf("8 vs 2+1 must stay a conflict: %+v", r)
	}
	r = reconcileVotes(t, q, votes(q, "extend", 9, "a"), votes(q, "refurb", 2, "b"), votes(q, "new", 1, "c"))
	if r.Band != "amber" || r.Value != "extend" || len(r.Alternatives) != 2 {
		t.Fatalf("9 vs 2+1 settles: %+v", r)
	}
}

// Flags, presence and free-text details keep their own conflict semantics.
func TestLopsidedRuleIsChoiceFieldsOnly(t *testing.T) {
	for _, c := range []struct{ q, lead, other string }{
		{"proj.flag.basement", "yes", "no"},
		{"sys.hydraulic.gas.presence", "included", "not_included"},
	} {
		in := input(append(votes(c.q, c.lead, 12, "a"), votes(c.q, c.other, 1, "b")...)...)
		in.Thresholds.Amber["detail"] = 0.6
		r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, c.q)
		if r.Band != "red" || r.Value != "" {
			t.Fatalf("%s must stay a conflict: %+v", c.q, r)
		}
	}
}
