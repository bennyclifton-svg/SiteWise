package reports

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
)

func money(s string) *costs.Money { m := costs.Money(s); return &m }

// total is a known total of amount, or an unknown one when amount is "".
func total(amount string, unknown int) costs.Total {
	if amount == "" {
		return costs.Total{KnownSubtotal: "0.00", Unknown: unknown, Lines: unknown}
	}
	return costs.Total{Amount: money(amount), KnownSubtotal: costs.Money(amount), Complete: unknown == 0, Unknown: unknown, Lines: 1 + unknown}
}

func totals(sections map[string]map[string]costs.Total, overall map[string]costs.Total, noBudget ...string) *costs.Totals {
	return &costs.Totals{Sections: sections, Overall: overall, NoBudget: noBudget}
}

func ptr(n int) *int { return &n }

func cents(n int64) *int64 { return &n }

// rulesOf is each item as "rule: text".
func rulesOf(items []AttentionItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Rule+": "+it.Text())
	}
	return out
}

func TestAttentionRules(t *testing.T) {
	for _, c := range []struct {
		name string
		in   AttentionInput
		want []string
		ids  [][]string
	}{
		{name: "nothing loaded", in: AttentionInput{}, want: nil},
		{
			name: "section contract over its budget",
			in: AttentionInput{Costs: totals(map[string]map[string]costs.Total{
				"construction": {"budget": total("1000.00", 0), "commitment": total("1200.00", 0)},
				"consultants":  {"budget": total("500.00", 0), "commitment": total("500.00", 0)},
			}, nil)},
			want: []string{RuleSectionContractOver + ": Construction: contract 1200 is over budget 1000 by 200"},
			ids:  [][]string{{"construction"}},
		},
		{
			name: "a section with no known budget is never over it",
			in: AttentionInput{Costs: totals(map[string]map[string]costs.Total{
				"construction": {"budget": total("", 3), "commitment": total("1200.00", 0)},
			}, nil)},
			want: nil,
		},
		{
			name: "forecast over a complete budget",
			in:   AttentionInput{Costs: totals(nil, map[string]costs.Total{"budget": total("1000.00", 0), "forecast": total("1100.00", 0)})},
			want: []string{RuleForecastOver + ": Forecast final cost 1100 is over the total budget 1000 by 100"},
			ids:  [][]string{nil},
		},
		{
			name: "forecast under budget",
			in:   AttentionInput{Costs: totals(nil, map[string]costs.Total{"budget": total("1000.00", 0), "forecast": total("900.00", 0)})},
			want: nil,
		},
		{
			name: "an unknown budget line makes the forecast say nothing",
			in:   AttentionInput{Costs: totals(nil, map[string]costs.Total{"budget": total("1000.00", 1), "forecast": total("5000.00", 0)}, "x")},
			want: []string{RuleCostNoBudget + ": 1 cost line has no budget amount"},
			ids:  [][]string{{"x"}},
		},
		{
			name: "an unknown forecast says nothing",
			in:   AttentionInput{Costs: totals(nil, map[string]costs.Total{"budget": total("1000.00", 0), "forecast": total("", 2)})},
			want: nil,
		},
		{
			name: "no budget, plural",
			in:   AttentionInput{Costs: totals(nil, nil, "a", "b")},
			want: []string{RuleCostNoBudget + ": 2 cost lines have no budget amount"},
			ids:  [][]string{{"a", "b"}},
		},
		{
			name: "packages: invited with no quote, and quotes to check",
			in: AttentionInput{Packages: []AttentionPackage{
				{ID: "p1", Title: "Architect", Firms: []AttentionFirm{{}, {}}},
				{ID: "p2", Title: "Planner 100%", Firms: []AttentionFirm{{Quoted: true, NeedsReview: true}, {}}},
				{ID: "p3", Title: "Quoted", Firms: []AttentionFirm{{Quoted: true}}},
			}},
			want: []string{
				RulePackageInvitedNoQuote + ": Architect: 2 firms invited and no quote received",
				RulePackageQuotesToCheck + ": Planner 100%: 1 quote has readings to check",
			},
			ids: [][]string{{"p1"}, {"p2"}},
		},
		{
			name: "documents not filed; pending and filed are not",
			in:   AttentionInput{Documents: []AttentionDocument{{ID: "d1", Status: "filed"}, {ID: "d2", Status: "not_filed"}, {ID: "d3", Status: "pending"}, {ID: "d4", Status: "split"}}},
			want: []string{RuleDocumentNotFiled + ": 1 document could not be filed or read"},
			ids:  [][]string{{"d2"}},
		},
		{
			name: "Program rows with no duration; stages and milestones are not",
			in: AttentionInput{Program: []AttentionActivity{
				{ID: "s", Kind: "stage"}, {ID: "a1", Kind: "activity"}, {ID: "a2", Kind: "activity", DurationDays: ptr(5)}, {ID: "a3", Kind: "activity"}, {ID: "m", Kind: "milestone"},
			}},
			want: []string{RuleProgramNoDuration + ": 2 Program rows have no duration, so the finish date is a lower bound"},
			ids:  [][]string{{"a1", "a3"}},
		},
		{
			name: "approvals submitted and past their target date",
			in: AttentionInput{Today: "2026-11-18", Delivery: []AttentionRecord{
				{ID: "late", Kind: "approval", Status: "submitted", TargetDate: "2026-11-17"},
				{ID: "today", Kind: "approval", Status: "submitted", TargetDate: "2026-11-18"},
				{ID: "no date", Kind: "approval", Status: "submitted"},
				{ID: "approved", Kind: "approval", Status: "approved", TargetDate: "2026-01-01"},
				{ID: "risk", Kind: "risk", Status: "open", TargetDate: "2026-01-01"},
			}},
			want: []string{RuleApprovalPastTarget + ": 1 approval is submitted, not determined and past the target date"},
			ids:  [][]string{{"late"}},
		},
		{
			name: "invoices to review and on no line",
			in: AttentionInput{Ledger: &AttentionLedger{
				Invoices: []invoices.Invoice{
					{ID: "i1", Status: invoices.StatusFiled, Kind: "invoice", Ex: cents(10000), Allocations: []invoices.Allocation{{LineID: "l1"}}},
					{ID: "i2", Status: invoices.StatusFiled, Kind: "invoice", Ex: cents(25050), Allocations: []invoices.Allocation{{LineID: ""}}},
					{ID: "i3", Status: invoices.StatusFiled, Kind: "invoice", Ex: cents(30000), Allocations: []invoices.Allocation{{LineID: "l1", Amount: cents(10000)}, {LineID: "", Amount: cents(20000)}}},
					{ID: "void", Status: invoices.StatusVoid, Kind: "invoice", Ex: cents(99900)},
				},
				Status:  map[string]string{"i1": invoices.StatusNeedsReview, "i2": invoices.StatusFiled, "i3": invoices.StatusNeedsReview, "void": invoices.StatusVoid},
				Summary: invoices.Summary{Lines: []invoices.LineSummary{{ID: "l1"}}, Unallocated: invoices.LineSummary{Invoiced: 45050}},
				Lines:   []invoices.BudgetLine{{ID: "l1"}},
			}},
			want: []string{
				RuleInvoiceNeedsReview + ": 2 invoices need review",
				RuleInvoiceUnallocated + ": 2 invoices have amounts on no cost line; the Unallocated row holds 450.5",
			},
			ids: [][]string{{"i1", "i3"}, {"i2", "i3"}},
		},
		{
			name: "claims over a line's budget: posting lines by their flag, groups by the group's",
			in: AttentionInput{Ledger: &AttentionLedger{
				Summary: invoices.Summary{Lines: []invoices.LineSummary{{ID: "l1", OverBudget: true}, {ID: "g", OverBudget: false}, {ID: "l2"}, {ID: "old", OverBudget: true, Archived: true}}},
				Lines:   []invoices.BudgetLine{{ID: "l1"}, {ID: "g", Group: true}, {ID: "l2"}, {ID: "old", Archived: true}},
				Groups:  map[string]invoices.GroupSummary{"g": {LineSummary: invoices.LineSummary{ID: "g", OverBudget: true}}},
			}},
			want: []string{RuleLineClaimedOver + ": 2 cost lines have claims to date above the line's budget"},
			ids:  [][]string{{"l1", "g"}},
		},
		{
			name: "implications: open critical or high only, by rank",
			in: AttentionInput{Implications: []AttentionImplication{
				{Key: "b", Label: "Bushfire", Rank: 2, Severity: "high"},
				{Key: "a", Label: "Asbestos", Rank: 1, Critical: true},
				{Key: "c", Label: "Decided", Rank: 0, Critical: true, Decided: true},
				{Key: "d", Label: "Low", Rank: 3, Severity: "low"},
			}},
			want: []string{RuleImplicationOpen + ": Open implication, critical: Asbestos", RuleImplicationOpen + ": Open implication, severity high: Bushfire"},
			ids:  [][]string{{"a"}, {"b"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			items := Attention(c.in)
			if got := rulesOf(items); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
			for i, it := range items {
				if i < len(c.ids) && !reflect.DeepEqual(it.IDs, c.ids[i]) {
					t.Errorf("%s ids %v, want %v", it.Rule, it.IDs, c.ids[i])
				}
				if len(it.IDs) > 0 && it.Count != len(it.IDs) && it.Rule != RulePackageInvitedNoQuote && it.Rule != RulePackageQuotesToCheck {
					t.Errorf("%s count %d for %d ids", it.Rule, it.Count, len(it.IDs))
				}
			}
		})
	}
}

// The order is fixed and the same whatever order the parts arrive in.
func TestAttentionOrder(t *testing.T) {
	in := AttentionInput{
		Today:        "2026-11-18",
		Delivery:     []AttentionRecord{{ID: "r", Kind: "approval", Status: "submitted", TargetDate: "2026-01-01"}},
		Program:      []AttentionActivity{{ID: "a", Kind: "activity"}},
		Documents:    []AttentionDocument{{ID: "d", Status: "not_filed"}},
		Packages:     []AttentionPackage{{ID: "p", Title: "P", Firms: []AttentionFirm{{}}}},
		Costs:        totals(map[string]map[string]costs.Total{"consultants": {"budget": total("1.00", 0), "commitment": total("2.00", 0)}, "construction": {"budget": total("1.00", 0), "commitment": total("2.00", 0)}}, nil, "x"),
		Implications: []AttentionImplication{{Key: "i", Label: "I", Critical: true}},
	}
	var got []string
	for _, it := range Attention(in) {
		got = append(got, it.Rule+" "+strings.Join(it.IDs, ","))
	}
	want := []string{RuleImplicationOpen + " i", RuleSectionContractOver + " construction", RuleSectionContractOver + " consultants", RuleCostNoBudget + " x",
		RulePackageInvitedNoQuote + " p", RuleDocumentNotFiled + " d", RuleProgramNoDuration + " a", RuleApprovalPastTarget + " r"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order\n%s", strings.Join(got, "\n"))
	}
}

// At most eight implications are listed, and the list is cut at ten with
// the rest counted.
func TestAttentionCaps(t *testing.T) {
	var in AttentionInput
	for i := 0; i < 12; i++ {
		in.Implications = append(in.Implications, AttentionImplication{Key: fmt.Sprint(i), Label: fmt.Sprint("Item ", i), Rank: i, Critical: true})
	}
	in.Documents = []AttentionDocument{{ID: "d", Status: "not_filed"}}
	in.Program = []AttentionActivity{{ID: "a", Kind: "activity"}}
	in.Packages = []AttentionPackage{{ID: "p", Title: "P", Firms: []AttentionFirm{{}}}}
	items := Attention(in)
	if len(items) != 11 {
		t.Fatalf("%d items: %v", len(items), rulesOf(items))
	}
	shown, more := AttentionShown(items)
	if len(shown) != MaxAttentionShown || more != 1 || shown[9].Rule != RuleDocumentNotFiled {
		t.Fatalf("shown %d, more %d, last %s", len(shown), more, shown[len(shown)-1].Rule)
	}
	if shown, more := AttentionShown(items[:3]); len(shown) != 3 || more != 0 {
		t.Fatalf("short list cut: %d, %d", len(shown), more)
	}
}

// Figures reach the caller as figures, so chat can tag each one; a record's
// own "%" never breaks the sentence.
func TestAttentionSentence(t *testing.T) {
	it := Attention(AttentionInput{Costs: totals(map[string]map[string]costs.Total{
		"construction": {"budget": total("1000.00", 0), "commitment": total("1500.50", 0)},
	}, nil)})[0]
	var labels []string
	got := it.Sentence(func(f AttentionFigure) string {
		labels = append(labels, f.Label)
		return "$" + string(f.Amount)
	})
	if got != "Construction: contract $1500.50 is over budget $1000.00 by $500.50" {
		t.Errorf("sentence %q", got)
	}
	if !reflect.DeepEqual(labels, []string{"Construction, contract", "Construction, budget", "Construction, contract less budget"}) {
		t.Errorf("labels %v", labels)
	}
	if c := centsMoney(-150); c != "-1.50" {
		t.Errorf("negative cents %s", c)
	}
}

// Plan 3's rules: payment schedules past due, due or to check, and
// variations waiting for a decision. A served schedule says nothing.
func TestAttentionClaimRules(t *testing.T) {
	l := &AttentionLedger{
		Claims: []AttentionClaim{{ID: "c1", Number: "4", Alert: invoices.AlertOverdue}, {ID: "c2", Alert: invoices.AlertSoon}, {ID: "c3", Alert: invoices.AlertCheck}, {ID: "c4", Alert: ""}},
		Variations: []invoices.Variation{{ID: "v1", Status: invoices.VariationSubmitted}, {ID: "v2", Status: invoices.VariationApproved}, {ID: "v3", Status: invoices.VariationProposed}},
	}
	items := Attention(AttentionInput{Ledger: l})
	want := map[string][]string{RuleScheduleOverdue: {"c1"}, RuleScheduleDue: {"c2", "c3"}, RuleVariationsUndecided: {"v1", "v3"}}
	for rule, ids := range want {
		it, ok := AttentionFor(items, rule)
		if !ok || !reflect.DeepEqual(it.IDs, ids) {
			t.Errorf("%s: %+v", rule, it)
		}
	}
	got := strings.Join(rulesOf(items), "\n")
	for _, s := range []string{"1 payment schedule is past its due date and not served: the claimed amount counts", "2 payment schedules are due within two business days, or with dates to check", "2 variations are waiting for a decision"} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q in\n%s", s, got)
		}
	}
	if items := Attention(AttentionInput{Ledger: &AttentionLedger{Claims: []AttentionClaim{{ID: "c4"}}}}); len(items) != 0 {
		t.Errorf("nothing due: %v", rulesOf(items))
	}
}
