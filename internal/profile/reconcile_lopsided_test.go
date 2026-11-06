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

// Scope presence keeps its own conflict semantics; a flag settles under R12.
func TestLopsidedRuleSkipsScopePresence(t *testing.T) {
	for _, c := range []struct {
		q, lead, other string
		settles        bool
	}{
		{"proj.flag.basement", "yes", "no", true},
		{"sys.hydraulic.gas.presence", "included", "not_included", false},
	} {
		in := input(append(votes(c.q, c.lead, 12, "a"), votes(c.q, c.other, 1, "b")...)...)
		in.Thresholds.Amber["detail"] = 0.6
		r := row(t, profile.Reconcile(in, repoCatalog(t)), whole, c.q)
		if c.settles && (r.Band != "amber" || r.Value != c.lead) {
			t.Fatalf("%s must settle on %s: %+v", c.q, c.lead, r)
		}
		if !c.settles && (r.Band != "red" || r.Value != "") {
			t.Fatalf("%s must stay a conflict: %+v", c.q, r)
		}
	}
}

// passages returns n passages reading value for question q, all from one document.
func passages(q, value, doc string, n int) []profile.Fact {
	var out []profile.Fact
	for i := 0; i < n; i++ {
		f := jevFact(q, value, doc, 0.9)
		f.PassageID = fmt.Sprintf("%s-p%d", doc, i)
		out = append(out, f)
	}
	return out
}

func reconcileDetail(t *testing.T, q string, groups ...[]profile.Fact) profile.Row {
	t.Helper()
	var facts []profile.Fact
	for _, g := range groups {
		facts = append(facts, g...)
	}
	in := input(facts...)
	in.Thresholds.Amber["detail"] = 0.6
	return row(t, profile.Reconcile(in, repoCatalog(t)), whole, q)
}

// F27 (R12): one document is one vote. T10's access constraints rested on five
// passages from two geotechnical reports against one survey; two documents
// to one is a conflict, not a settled value.
func TestLopsidedRuleCountsDocumentsNotPassages(t *testing.T) {
	q := "hdr.cond.access_constraints"
	r := reconcileVotes(t, q,
		passages(q, "urban_constrained", "geotech-1", 3),
		passages(q, "urban_constrained", "geotech-2", 2),
		votes(q, "unrestricted", 1, "survey"))
	if r.Band != "red" || r.Value != "" || len(r.Alternatives) != 2 {
		t.Fatalf("5 passages from 2 documents vs 1 document must stay a conflict: %+v", r)
	}
	if len(r.Alternatives[0].Sources) != 5 {
		t.Fatalf("every passage stays listed as a source: %+v", r.Alternatives)
	}
}

// F35 (R12): free text settles on a strong majority of documents. T17: eleven
// documents name the owners corporation one way, the audit another.
func TestLopsidedFreeTextSettlesByDocuments(t *testing.T) {
	q := "proj.client"
	lead, other := "The Owners - Strata Plan No. 87654", "Strata Plan 87654"
	r := reconcileDetail(t, q, votes(q, lead, 11, "d"), votes(q, other, 1, "audit"))
	if r.Value != lead || r.Band != "amber" {
		t.Fatalf("11 documents vs 1 must settle amber on %q: %+v", lead, r)
	}
	if len(r.Alternatives) != 1 || r.Alternatives[0].Value != other {
		t.Fatalf("the other spelling must stay visible: %+v", r.Alternatives)
	}
	// The audit is one of the eleven too: a document counts once per value.
	audit := jevFact(q, lead, "audit0", 0.9)
	audit.PassageID = "audit0-p2"
	r = reconcileDetail(t, q, votes(q, lead, 10, "d"), []profile.Fact{audit}, votes(q, other, 1, "audit"))
	if r.Value != lead || r.Band != "amber" {
		t.Fatalf("11 documents (one also dissenting) vs 1 must settle: %+v", r)
	}
}

// F36 (R12): a flag settles on a strong majority, but two documents against
// one is still a conflict (T17's existing basement car park).
func TestLopsidedFlag(t *testing.T) {
	q := "proj.flag.basement"
	r := reconcileDetail(t, q, votes(q, "no", 2, "a"), votes(q, "yes", 1, "b"))
	if r.Band != "red" || r.Value != "" || len(r.Alternatives) != 2 {
		t.Fatalf("no x2 vs yes x1 must stay a conflict: %+v", r)
	}
	r = reconcileDetail(t, q, votes(q, "no", 3, "a"), votes(q, "yes", 1, "b"))
	if r.Band != "amber" || r.Value != "no" || len(r.Alternatives) != 1 {
		t.Fatalf("no x3 vs yes x1 settles amber: %+v", r)
	}
}

// Numbers and money keep today's rule: any disagreement is a conflict.
func TestLopsidedRuleLeavesMoneyAndNumbers(t *testing.T) {
	for _, q := range []string{"proj.budget_stated", "proj.contingency_stated", "proj.preliminary_estimate",
		"proj.max_construction_months", "proj.preliminary_construction_months", "proj.owner_supply_stated"} {
		r := reconcileDetail(t, q, votes(q, "$1,200,000", 12, "a"), votes(q, "$1,300,000", 1, "b"))
		if r.Band != "red" || r.Value != "" {
			t.Fatalf("%s must stay a conflict: %+v", q, r)
		}
	}
}
