package reports

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/costs"
	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
)

var monthlyTitles = map[string]string{"glance": "At a glance", "attention": "Needs attention", "changes": "Changes since the last report", "cost": "Cost",
	"program": "Program", "procurement": "Procurement", "registers": "Registers", "next": "Next period"}

func assemble(t *testing.T, in MonthlyInput, edits ...Edit) []Section {
	t.Helper()
	if in.AsAt == "" {
		in.AsAt = "2026-11-17"
	}
	out, err := AssembleMonthly(in, monthlyTitles, edits)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func textOf(sections []Section, section string) string {
	var out []string
	for _, s := range sections {
		if s.ID != section {
			continue
		}
		for _, b := range s.Blocks {
			out = append(out, b.Text)
			if b.Table != nil {
				for _, r := range b.Table.Rows {
					out = append(out, strings.Join(r, " | "))
				}
			}
		}
	}
	return strings.Join(out, "\n")
}

func plan(sections map[string]map[string]costs.Total, taxBasis string) *costs.Totals {
	overall := map[string]costs.Total{}
	for _, metric := range []string{"budget", "commitment", "forecast"} {
		sum := costs.Total{KnownSubtotal: "0.00"}
		for _, s := range sections {
			t := s[metric]
			sum.Lines += t.Lines
			sum.Unknown += t.Unknown
			if t.KnownSubtotal != "" {
				sum.KnownSubtotal = costs.Add(sum.KnownSubtotal, t.KnownSubtotal)
			}
		}
		if sum.Unknown == 0 && sum.Lines > 0 {
			a := sum.KnownSubtotal
			sum.Amount, sum.Complete = &a, true
		}
		overall[metric] = sum
	}
	full := map[string]map[string]costs.Total{}
	for _, s := range CostSectionOrder {
		full[s] = map[string]costs.Total{"budget": {KnownSubtotal: "0.00"}, "commitment": {KnownSubtotal: "0.00"}, "forecast": {KnownSubtotal: "0.00"}}
		for m, v := range sections[s] {
			full[s][m] = v
		}
	}
	return &costs.Totals{Sections: full, Overall: overall, TaxBasis: taxBasis}
}

func TestTotalTextNeverPrintsZeroForUnknown(t *testing.T) {
	r1, r2 := costs.Money("10.00"), costs.Money("20.00")
	for _, c := range []struct {
		in   costs.Total
		want string
	}{
		{costs.Total{}, "—"},
		{costs.Total{Lines: 2, Unknown: 2, KnownSubtotal: "0.00"}, "Not known"},
		{total("1200.00", 0), "$1,200"},
		{costs.Total{Lines: 3, Unknown: 1, KnownSubtotal: "500.00"}, "$500 known; 1 unknown"},
		{costs.Total{Lines: 1, KnownSubtotal: "0.00", Low: &r1, High: &r2}, "$10–$20"},
	} {
		if got := TotalText(c.in); got != c.want {
			t.Errorf("%+v: %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMonthlyCostSection(t *testing.T) {
	ledger := &AttentionLedger{Summary: invoices.Summary{Total: 330000, Comparable: true, Sections: map[string]int64{"consultants": 300000},
		SectionsContractRemaining: map[string]int64{"consultants": 900000}, Unallocated: invoices.LineSummary{Invoiced: 30000}}}
	remaining := int64(900000)
	ledger.Summary.ContractRemaining = &remaining
	in := MonthlyInput{Costs: plan(map[string]map[string]costs.Total{
		"consultants": {"budget": total("20000.00", 0), "commitment": total("12000.00", 0), "forecast": total("12000.00", 0)},
		"construction": {"budget": total("1000000.00", 0), "commitment": {Lines: 2, Unknown: 2, KnownSubtotal: "0.00"},
			"forecast": {Lines: 2, Unknown: 1, KnownSubtotal: "400000.00"}},
	}, "ex_tax"), Ledger: ledger}
	got := textOf(assemble(t, in), "cost")
	for _, want := range []string{
		"Consultants | $20,000 | $12,000 | $12,000 | $3,000 | $9,000",
		"Construction | $1,000,000 | Not known | $400,000 known; 1 unknown | $0 | Not known",
		"Unallocated invoices | — | — | — | $300 | —",
		"Total | $1,020,000 | $12,000 known; 2 unknown | $412,000 known; 1 unknown | $3,300 | $9,000",
		"Forecast final cost is not yet known: 1 line has no forecast, and the known part is $412,000 ex GST.",
		"Variations are not yet recorded in SiteWise.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("cost lacks %q:\n%s", want, got)
		}
	}
	// A plan in inc GST: remaining stays unknown, as on the grid.
	ledger.Summary.Comparable = false
	in.Costs.TaxBasis = "inc_tax"
	got = textOf(assemble(t, in), "cost")
	if !strings.Contains(got, "Consultants | $20,000 | $12,000 | $12,000 | $3,000 | Not known") || !strings.Contains(got, "include GST, so remaining is not known") {
		t.Errorf("inc GST:\n%s", got)
	}
	// Complete totals give the status words.
	in = MonthlyInput{Costs: plan(map[string]map[string]costs.Total{"consultants": {"budget": total("20000.00", 0), "commitment": total("12000.00", 0), "forecast": total("15000.00", 0)}}, "ex_tax")}
	if got := textOf(assemble(t, in), "cost"); !strings.Contains(got, "Forecast final cost is $15,000 ex GST, $5,000 under budget.") {
		t.Errorf("under budget:\n%s", got)
	}
}

func relativeProgram(finish int) *MonthlyProgram {
	return &MonthlyProgram{Relative: true, Complete: true, TypicalIndex: &finish, TypicalLabel: fmt.Sprintf("Week %d", finish/5+1), Known: 2, Total: 2,
		Activities: []MonthlyActivity{{ID: "s1", Kind: "stage", Label: "Design", Critical: true}, {ID: "a1", StageID: "s1", Kind: "activity", Label: "Concept", StartLabel: "Week 1", FinishLabel: "Week 2", Critical: true}}}
}

func TestMonthlyProgramSection(t *testing.T) {
	got := textOf(assemble(t, MonthlyInput{Program: relativeProgram(52)}), "program")
	for _, want := range []string{"Planned completion Week 11.", "counted in weeks", "The critical path runs through Design.", "SiteWise records no construction progress."} {
		if !strings.Contains(got, want) {
			t.Errorf("relative program lacks %q:\n%s", want, got)
		}
	}
	if next := textOf(assemble(t, MonthlyInput{Program: relativeProgram(52)}), "next"); !strings.Contains(next, "none of its rows is dated") {
		t.Errorf("next with a relative program: %s", next)
	}
	p := &MonthlyProgram{Complete: false, Typical: "2028-03-14", TypicalLabel: "14 March 2028", Known: 31, Total: 44}
	got = textOf(assemble(t, MonthlyInput{Program: p}), "program")
	if !strings.Contains(got, "Planned completion at least 14 March 2028. 31 of 44 activities have a duration") {
		t.Errorf("durations missing:\n%s", got)
	}
}

func TestMonthlyProcurementIsCapped(t *testing.T) {
	var pkgs []MonthlyPackage
	for i := 0; i < 40; i++ {
		pkgs = append(pkgs, MonthlyPackage{ID: fmt.Sprintf("p%02d", i), Title: fmt.Sprintf("Package %02d", i), Lifecycle: "planned", Invited: i % 3})
	}
	pkgs[0].AwardedTo, pkgs[0].Quoted, pkgs[0].Invited = "Northline", 2, 0
	sections := assemble(t, MonthlyInput{Packages: pkgs})
	b := findBlock(sections, "proc:table")
	if b == nil || len(b.Table.Rows) != maxPackageRows || !strings.Contains(b.Text, "And 25 more packages in SiteWise.") {
		t.Fatalf("capped table %+v", b)
	}
	var basis packagesBasis
	if err := json.Unmarshal(b.Basis, &basis); err != nil || len(basis.Packages) != 40 {
		t.Fatalf("every package is kept for the next comparison: %d %v", len(basis.Packages), err)
	}
	if b.Table.Rows[0][4] != "Northline" || b.Table.Rows[0][5] != "Not known" {
		t.Errorf("awarded row %v: a contract the cost plan does not hold is not known", b.Table.Rows[0])
	}
	if got := textOf(sections, "glance"); !strings.Contains(got, "Of 40 packages, 1 is awarded, 0 have quotes and 26 are waiting for quotes.") {
		t.Errorf("glance %s", got)
	}
}

func TestMonthlyFirstReportAndMovement(t *testing.T) {
	first := MonthlyInput{AsAt: "2026-10-17", Number: 1,
		Costs:    plan(map[string]map[string]costs.Total{"consultants": {"budget": total("20000.00", 0), "commitment": total("12000.00", 0), "forecast": total("12000.00", 0)}}, "ex_tax"),
		Program:  &MonthlyProgram{Complete: true, Typical: "2027-06-01", TypicalLabel: "1 June 2027", Activities: []MonthlyActivity{{ID: "m1", Kind: "milestone", Label: "DA lodged", Start: "2026-12-01", StartLabel: "1 December 2026"}, {ID: "m2", Kind: "milestone", Label: "CC issued", Start: "2027-02-01", StartLabel: "1 February 2027"}}},
		Packages: []MonthlyPackage{{ID: "p1", Title: "Builder", Invited: 3}, {ID: "p2", Title: "Surveyor", Lifecycle: "planned"}},
		Delivery: []delivery.Item{{ID: "r1", Content: delivery.Content{Kind: "risk", Title: "Ground", Status: "open"}}},
	}
	one := assemble(t, first)
	if got := textOf(one, "changes"); got != "This is the first issued report; changes are measured from the next one." {
		t.Fatalf("first report changes %q", got)
	}
	second := first
	second.AsAt, second.Number = "2026-11-17", 2
	second.Previous = &MonthlyPrevious{Number: 1, Date: "2026-10-17", Sections: one}
	second.Costs = plan(map[string]map[string]costs.Total{"consultants": {"budget": total("20000.00", 0), "commitment": total("52000.00", 0), "forecast": total("52000.00", 0)}}, "ex_tax")
	// The milestone is renamed and moved: it is matched by id and its current
	// label printed. The other is removed.
	second.Program = &MonthlyProgram{Complete: true, Typical: "2027-06-10", TypicalLabel: "10 June 2027", Activities: []MonthlyActivity{{ID: "m1", Kind: "milestone", Label: "DA lodged with council", Start: "2026-12-10", StartLabel: "10 December 2026"}}}
	second.Packages = []MonthlyPackage{{ID: "p1", Title: "Builder", AwardedTo: "Northline", Quoted: 3, AwardedOn: "2026-11-02"}}
	second.Delivery = []delivery.Item{{ID: "r1", Content: delivery.Content{Kind: "risk", Title: "Ground", Status: "closed"}}}
	second.Documents = []MonthlyDocument{{ID: "d1", Received: "2026-11-01", Discipline: "architectural"}, {ID: "d0", Received: "2026-10-01", Discipline: "architectural"}}
	two := assemble(t, second)
	got := textOf(two, "changes")
	for _, want := range []string{
		"Forecast final cost up $40,000, from $12,000 to $52,000.",
		"Planned completion 9 days later, from 1 June 2027 to 10 June 2027.",
		"DA lodged with council 9 days later, from 1 December 2026 to 10 December 2026.",
		"CC issued was removed from the Program.",
		"Builder: Invited → awarded to Northline.",
		"Surveyor was removed from procurement.",
		"1 risk closed: Ground.",
		"1 document filed since 17 October 2026: 1 architectural.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("changes lack %q:\n%s", want, got)
		}
	}
	// Cost moves come first, largest first.
	if !strings.HasPrefix(got, "Forecast final cost up $40,000") {
		t.Errorf("order:\n%s", got)
	}
	if glance := textOf(two, "glance"); !strings.Contains(glance, "That is 9 days later than report 1.") {
		t.Errorf("glance %s", glance)
	}
	// A previous issue built another way is not compared.
	for i := range one {
		for j := range one[i].Blocks {
			if one[i].Blocks[j].ID == "cost:table" {
				one[i].Blocks[j].Basis = json.RawMessage(`{"method":"older"}`)
			}
		}
	}
	if got := textOf(assemble(t, second), "changes"); !strings.Contains(got, "Cost is not compared: the previous report was built another way.") {
		t.Errorf("method change:\n%s", got)
	}
}

// A comment survives Update; a new period's draft flags it.
func TestMonthlyCommentIsFlaggedByANewPeriod(t *testing.T) {
	in := MonthlyInput{AsAt: "2026-10-17"}
	sections := assemble(t, in)
	b := findBlock(sections, "comment:cost")
	edit := Edit{TargetID: "comment:cost", Text: "On track.", BaseContentSHA256: b.ContentSHA256}
	again := assemble(t, in, edit)
	if c := findBlock(again, "comment:cost"); c.Text != "On track." || c.Conflict {
		t.Fatalf("same period %+v", c)
	}
	in.Previous = &MonthlyPrevious{Number: 1, Date: "2026-10-17", Sections: sections}
	next := assemble(t, in, edit)
	if c := findBlock(next, "comment:cost"); c.Text != "On track." || !c.Conflict {
		t.Fatalf("new period %+v", c)
	}
}

func TestCompletionLaterRule(t *testing.T) {
	idx := func(n int) *int { return &n }
	for _, c := range []struct {
		name      string
		now, last *AttentionCompletion
		want      string
	}{
		{"no report yet", &AttentionCompletion{Date: "2027-06-10", Label: "10 June 2027"}, nil, ""},
		{"later", &AttentionCompletion{Date: "2027-06-10", Label: "10 June 2027"}, &AttentionCompletion{Number: 3, Date: "2027-06-01", Label: "1 June 2027"},
			RuleCompletionLater + ": Planned completion 10 June 2027 is 9 days later than report 3 printed (1 June 2027)"},
		{"earlier says nothing", &AttentionCompletion{Date: "2027-05-10"}, &AttentionCompletion{Number: 3, Date: "2027-06-01"}, ""},
		{"relative in working days", &AttentionCompletion{Relative: true, Index: idx(60), Label: "Week 13"}, &AttentionCompletion{Number: 1, Relative: true, Index: idx(55), Label: "Week 12"},
			RuleCompletionLater + ": Planned completion Week 13 is 5 working days later than report 1 printed (Week 12)"},
		{"dated then relative says nothing", &AttentionCompletion{Relative: true, Index: idx(60)}, &AttentionCompletion{Number: 1, Date: "2027-06-01"}, ""},
	} {
		got := strings.Join(rulesOf(Attention(AttentionInput{Completion: c.now, LastReport: c.last})), "; ")
		if got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
