package proposals

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/rfp"
)

// page builds one PDF page of text: each row is a line of cells, printed
// left to right at fixed columns so table headings line up.
func page(n int, rows ...[]string) []identity.Run {
	var runs []identity.Run
	y := 800.0
	for _, row := range rows {
		x := 40.0
		for _, cell := range row {
			runs = append(runs, identity.Run{Text: cell, Source: identity.Source{Page: n, X: x, Y: y, Width: 90, Height: 10}})
			x += 120
		}
		y -= 14
	}
	return runs
}

func text(runs ...[]identity.Run) identity.Text {
	t := identity.Text{TextLayer: true, Format: "pdf"}
	for _, r := range runs {
		t.Runs = append(t.Runs, r...)
	}
	return t
}

func find(ev Evidence, display string) (Amount, bool) {
	for _, a := range ev.Amounts {
		if a.Display == display {
			return a, true
		}
	}
	return Amount{}, false
}

func TestHarvestReadsFeesRatesAndTotals(t *testing.T) {
	ev := Harvest("Fee Town Planning (Mecone).pdf", text(page(1,
		[]string{"Mecone Pty Ltd"},
		[]string{"4. FEE PROPOSAL"},
		[]string{"Task 1 - Preliminary advice", "$3,000"},
		[]string{"Task 2 - Report", "$ 11, 950"},
		[]string{"TOTAL COST (excl. GST)", "$14,950"},
		[]string{"Director", "$480/hour"},
		[]string{"Professional indemnity", "$10,000,000"},
		[]string{"Projects valued at $90 million"},
		[]string{"info@mecone.com.au", "02 8073 4677"},
	)))
	if a, ok := find(ev, "$3,000"); !ok || a.Cents != 300000 || a.Label != "Task 1 - Preliminary advice" || a.PerHour {
		t.Fatalf("task fee %+v %v", a, ok)
	}
	if a, ok := find(ev, "$ 11, 950"); !ok || a.Cents != 1195000 {
		t.Fatalf("spaced thousands %+v %v", a, ok)
	}
	if a, _ := find(ev, "$14,950"); !a.Total || a.GST != "ex" {
		t.Fatalf("total %+v", a)
	}
	if a, _ := find(ev, "$480"); !a.PerHour {
		t.Fatalf("rate %+v", a)
	}
	for _, a := range ev.Amounts {
		if a.Cents >= 1_000_000_00 || strings.Contains(a.Context, "valued") {
			t.Fatalf("insurance or project value kept: %+v", a)
		}
	}
	if len(ev.Emails) != 1 || ev.Phones[0] != "02 8073 4677" || ev.GSTEx == 0 {
		t.Fatalf("firm details %+v %+v", ev.Emails, ev.Phones)
	}
}

func TestHarvestDropsGSTCopiesAndUnitRates(t *testing.T) {
	ev := Harvest("quote.pdf", text(page(1,
		[]string{"Arboricultural Impact Assessment", "1", "$1,100.00", "$1,100.00"},
		[]string{"Subtotal", "$1,100.00"},
		[]string{"GST (10%)", "$110.00"},
		[]string{"Total", "$1,210.00"},
		[]string{"For drill rig hire", "30", "hour", "$330", "$9,900.00"},
		[]string{"Report", "$10,000 + GST = $11,000"},
	)))
	if a, _ := find(ev, "$1,210.00"); !a.IncTwin || a.GST != "inc" {
		t.Fatalf("inclusive total not marked: %+v", a)
	}
	if a, _ := find(ev, "$110.00"); !a.GSTPart {
		t.Fatalf("GST figure not marked: %+v", a)
	}
	if a, _ := find(ev, "$330"); !a.Unit {
		t.Fatalf("unit rate not marked: %+v", a)
	}
	if a, _ := find(ev, "$9,900.00"); a.Unit {
		t.Fatalf("extended amount marked unit: %+v", a)
	}
	if a, _ := find(ev, "$11,000"); !a.IncTwin {
		t.Fatalf("same-line inclusive twin not marked: %+v", a)
	}
}

func TestHarvestNamesTheColumn(t *testing.T) {
	ev := Harvest("services.pdf", text(page(1,
		[]string{"Stage", "Mechanical", "Electrical", "Total"},
		[]string{"Stage 1 - Schematic", "$3,000", "$4,000", "$7,000"},
	)))
	if a, _ := find(ev, "$4,000"); a.Column != "Electrical" {
		t.Fatalf("column %+v", a)
	}
}

var planning = []rfp.ScheduleItem{
	{Key: "stage:advice", Kind: rfp.ItemStage, Label: "Preliminary advice", Base: true},
	{Key: "stage:report", Kind: rfp.ItemStage, Label: "Planning report", Base: true},
	{Key: rfp.ReimbursablesKey, Kind: rfp.ItemReimbursable, Label: "Reimbursables", Base: true},
	{Key: "option:panel", Kind: rfp.ItemOption, Label: "Panel attendance"},
	{Key: "rate:director", Kind: rfp.ItemRate, Label: "Director"},
	{Key: "criterion:r6", Kind: rfp.ItemCriterion, Label: "Departures"},
}

func proposalText() identity.Text {
	return text(page(1,
		[]string{"Varga Traffic Planning Pty Ltd"},
		[]string{"ABN 88 071 762 537"},
		[]string{"1. FEE PROPOSAL"},
		[]string{"Preliminary advice", "$3,000"},
		[]string{"Planning report", "$9,000"},
		[]string{"Heritage report", "$1,200"},
		[]string{"Total (excl. GST)", "$12,000"},
		[]string{"Director", "$480 per hour"},
		[]string{"2. EXCLUSIONS AND ASSUMPTIONS"},
		[]string{"Council fees are excluded."},
	))
}

const pkg = "11111111-1111-4111-8111-111111111111"

func context1() Context {
	return Context{
		Packages:   []Package{{ID: pkg, Title: "Town planning", Discipline: "consultant.town_planner"}},
		Discipline: "consultant.town_planner",
		Schedules:  map[string][]rfp.ScheduleItem{pkg: planning},
	}
}

type fakeJev struct {
	calls   int
	answers map[string]string
	conf    float64
	err     error
}

func (f *fakeJev) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	f.calls++
	if f.err != nil {
		return jev.Result{}, f.err
	}
	res := jev.Result{Answers: map[string]jev.Answer{}}
	for field := range c.Questions {
		if choice, ok := f.answers[field]; ok {
			conf := f.conf
			res.Answers[field] = jev.Answer{Type: jev.TypeChoice, Choice: choice, Confidence: &conf}
		}
	}
	return res, nil
}

func greenAt(v float64) Thresholds {
	lo, hi := 2, 255
	g, a := v, v
	th := Thresholds{QuestionVersion: QuestionVersion, Questions: map[string][]Threshold{}}
	for _, f := range []string{"price", "rate", "criterion", "firm", FieldKind, FieldPackage, FieldGSTBasis, FieldStatedTotal} {
		th.Questions[f] = []Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &a}}
	}
	return th
}

func TestReadSettlesByRuleThenOneFanOut(t *testing.T) {
	ev := Harvest("Fee Proposal Town Planning (Varga).pdf", proposalText())
	d := NewDraft(ev, context1(), nil)
	for field, want := range map[string]string{FieldKind: KindFeeProposal, FieldPackage: pkg, FieldFirmABN: "88071762537", FieldFirmName: "Varga Traffic Planning Pty Ltd", FieldGSTBasis: "ex", FieldStatedTotal: "1200000"} {
		if got, _ := d.Decided(field); got.Value != want || got.DecidedBy != ByRule {
			t.Fatalf("%s = %+v, want rule %q", field, got, want)
		}
	}
	ids := map[string]string{}
	for _, a := range ev.Amounts {
		ids[a.Display] = a.ID
	}
	fake := &fakeJev{conf: 0.99, answers: map[string]string{
		"price:stage:advice": ids["$3,000"], "price:stage:report": ids["$9,000"], "price:reimbursables": choiceNotPrice,
		"price:option:panel": choiceNotPrice, "price:rate:director": ids["$480"],
	}}
	out, err := Read(context.Background(), fake, greenAt(0.9), "Fee Proposal Town Planning (Varga).pdf", proposalText(), context1(), nil)
	if err != nil || fake.calls != 1 || out.Grey {
		t.Fatalf("calls %d grey %v err %v", fake.calls, out.Grey, err)
	}
	prices := map[string]Price{}
	var other []Price
	for _, p := range out.Prices {
		if p.Key == "" {
			other = append(other, p)
			continue
		}
		prices[p.Key] = p
	}
	if p := prices["stage:advice"]; p.State != StatePriced || *p.Cents != 300000 || p.Band != BandGreen || p.Passage == "" {
		t.Fatalf("advice %+v", p)
	}
	if p := prices["rate:director"]; *p.Cents != 48000 {
		t.Fatalf("rate %+v", p)
	}
	if prices["reimbursables"].State != StateNotPriced {
		t.Fatalf("reimbursables %+v", prices["reimbursables"])
	}
	if len(other) != 1 || *other[0].Cents != 120000 || !strings.Contains(other[0].Label, "Heritage") {
		t.Fatalf("other items %+v", other)
	}
	if hasFlag(out.Flags, FlagTotalGap) {
		t.Fatalf("12,000 = 3,000 + 9,000 is no gap: %v", out.Flags)
	}
	var departures Response
	for _, r := range out.Responses {
		if r.Key == "criterion:r6" {
			departures = r
		}
	}
	if len(departures.Passages) == 0 || departures.DecidedBy != ByRule {
		t.Fatalf("departures %+v", departures)
	}
}

func hasFlag(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestReadTimeoutLeavesRulesAndGoesGrey(t *testing.T) {
	out, err := Read(context.Background(), &fakeJev{err: errors.New("deadline")}, greenAt(0.5), "Fee Proposal (Varga).pdf", proposalText(), context1(), nil)
	if err != nil || !out.Grey {
		t.Fatalf("grey %v err %v", out.Grey, err)
	}
	if out.PackageID != pkg || out.Kind != KindFeeProposal {
		t.Fatalf("rule values lost: %+v", out.Reading)
	}
	for _, p := range out.Prices {
		if p.Key == "stage:advice" && (p.Band != BandGrey || p.State != "") {
			t.Fatalf("unanswered price not grey: %+v", p)
		}
	}
}

func TestUserDecisionsAreNeverReplaced(t *testing.T) {
	users := map[string]Decision{"price:stage:advice": {Field: "price:stage:advice", Value: "cents:250000", Band: BandGreen, DecidedBy: ByUser}}
	fake := &fakeJev{conf: 0.99, answers: map[string]string{"price:stage:advice": "a1"}}
	out, _ := Read(context.Background(), fake, greenAt(0.5), "Fee Proposal (Varga).pdf", proposalText(), context1(), users)
	for _, p := range out.Prices {
		if p.Key == "stage:advice" && (*p.Cents != 250000 || p.DecidedBy != ByUser) {
			t.Fatalf("user's price replaced: %+v", p)
		}
	}
}

func TestGSTInclusiveAndPercentageFees(t *testing.T) {
	inc := text(page(1,
		[]string{"Fee proposal"},
		[]string{"Preliminary advice", "$3,300 incl. GST"},
		[]string{"Planning report", "6% of the construction cost"},
	))
	budget := int64(100_000_00)
	pc := context1()
	pc.ConstructionBudget = &budget
	ev := Harvest("Fee proposal.pdf", inc)
	ids := map[string]string{}
	for _, a := range ev.Amounts {
		ids[a.Display] = a.ID
	}
	fake := &fakeJev{conf: 0.99, answers: map[string]string{"price:stage:advice": ids["$3,300"], "price:stage:report": ids["6% of the construction cost"]}}
	out, _ := Read(context.Background(), fake, greenAt(0.5), "Fee proposal.pdf", inc, pc, nil)
	for _, p := range out.Prices {
		switch p.Key {
		case "stage:advice":
			if *p.Cents != 300000 || !strings.Contains(p.Basis, "1.1") {
				t.Fatalf("inc GST not normalised: %+v", p)
			}
		case "stage:report":
			if p.Cents == nil || *p.Cents != 600000 || !strings.Contains(p.Basis, "6%") {
				t.Fatalf("percentage fee: %+v", p)
			}
		}
	}
	pc.ConstructionBudget = nil
	out, _ = Read(context.Background(), fake, greenAt(0.5), "Fee proposal.pdf", inc, pc, nil)
	for _, p := range out.Prices {
		if p.Key == "stage:report" && (p.Cents != nil || p.State != StatePriced || !strings.Contains(p.Basis, "not set")) {
			t.Fatalf("percentage without a budget must stay unknown: %+v", p)
		}
	}
}

func TestRatesOnlyAndNoText(t *testing.T) {
	rates := text(page(1, []string{"Fee proposal"}, []string{"Director", "$300/hr"}, []string{"Engineer", "$200/hr"}))
	out, _ := Read(context.Background(), &fakeJev{}, greenAt(0.5), "Fee proposal.pdf", rates, context1(), nil)
	if !hasFlag(out.Flags, FlagRatesOnly) {
		t.Fatalf("flags %v", out.Flags)
	}
	for _, p := range out.Prices {
		if p.Key == "stage:advice" && (p.State != StateNotPriced || p.DecidedBy != ByRule) {
			t.Fatalf("rates-only fee row %+v", p)
		}
	}
	scan, _ := Read(context.Background(), &fakeJev{}, greenAt(0.5), "scan.pdf", identity.Text{Format: "pdf"}, context1(), nil)
	if scan.NotRead != NotReadNoText || !hasFlag(scan.Flags, FlagNoText) {
		t.Fatalf("scan %+v", scan)
	}
}

func TestDirectoryFirmAsksNoFirmQuestion(t *testing.T) {
	pc := context1()
	pc.Lookup = func(abns, _ []string) *Firm {
		for _, a := range abns {
			if a == "88071762537" {
				return &Firm{ID: "f1", Name: "Varga Traffic Planning", ABN: a, Email: "office@varga.com.au"}
			}
		}
		return nil
	}
	d := NewDraft(Harvest("x.pdf", proposalText()), pc, nil)
	for _, q := range d.Questions() {
		if strings.HasPrefix(q, "firm_") {
			t.Fatalf("asked %s for a directory firm", q)
		}
	}
	if got, _ := d.Decided(FieldFirmEmail); got.Value != "office@varga.com.au" {
		t.Fatalf("directory details not filled: %+v", got)
	}
}

func TestKindNeedsAStrongCue(t *testing.T) {
	report := Harvest("Civil Structural Recommendation.pdf", text(page(1, []string{"Tender evaluation"}, []string{"Total fee", "$36,000"})))
	if k := kindRule(report, "consultant.structural"); k != "" {
		t.Fatalf("an evaluation report settled as %q", k)
	}
	quote := Harvest("Works quotation.pdf", text(page(1, []string{"WORKS QUOTATION FORM"}, []string{"Arborist report", "$1,859.00"})))
	if k := kindRule(quote, "consultant.arborist"); k != KindFeeProposal {
		t.Fatalf("consultant quote = %q", k)
	}
	if k := kindRule(quote, "trade.civil_works"); k != KindTender {
		t.Fatalf("trade quote = %q", k)
	}
}

func TestResponseFormReadsByKey(t *testing.T) {
	var runs []identity.Run
	cell := func(row, col int, v string) {
		runs = append(runs, identity.Run{Text: v, Source: identity.Source{Sheet: "R1 Fees", Row: row, Col: col}})
	}
	cell(5, 1, "Firm")
	cell(5, 2, "Varga Traffic Planning")
	cell(6, 1, "ABN")
	cell(6, 2, "88 071 762 537")
	cell(12, 1, "stage:advice")
	cell(12, 2, "Preliminary advice")
	cell(12, 3, "3000")
	cell(13, 1, "stage:report")
	cell(13, 2, "Planning report")
	cell(13, 4, "6")
	cell(14, 1, "reimbursables")
	cell(14, 3, "Included in stage:advice")
	cell(15, 1, "base_total")
	cell(15, 3, "3000")
	form := identity.Text{TextLayer: true, Format: "xlsx", Runs: runs}
	out, err := Read(context.Background(), &fakeJev{}, Thresholds{}, "response.xlsx", form, context1(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Price{}
	for _, p := range out.Prices {
		got[p.Key] = p
	}
	if p := got["stage:advice"]; p.DecidedBy != ByRule || *p.Cents != 300000 {
		t.Fatalf("form fee %+v", p)
	}
	if p := got["stage:report"]; p.State != StateNotPriced {
		t.Fatalf("an empty fee cell (weeks only) is not priced: %+v", p)
	}
	if got["reimbursables"].State != StateIncluded || out.FirmFields[FieldFirmName] != "Varga Traffic Planning" || out.Kind != KindFeeProposal {
		t.Fatalf("form %+v", out)
	}
}

func TestNameKey(t *testing.T) {
	if NameKey("Mecone Pty Ltd") != NameKey("MECONE") || NameKey("BG&E Pty Limited") != "bge" {
		t.Fatalf("%q %q", NameKey("Mecone Pty Ltd"), NameKey("BG&E Pty Limited"))
	}
}

func TestExGST(t *testing.T) {
	for inc, ex := range map[int64]int64{1100: 1000, 330000: 300000, 1: 1, -1100: -1000} {
		if got := exGST(inc); got != ex {
			t.Fatalf("exGST(%d) = %d, want %d", inc, got, ex)
		}
	}
}

func TestOneLumpSumAgainstStagesStaysAmber(t *testing.T) {
	name := "Fee Proposal Town Planning (Varga).pdf"
	body := text(page(1,
		[]string{"Varga Traffic Planning Pty Ltd", "ABN 88 071 762 537"},
		[]string{"Fee proposal"},
		[]string{"Full town planning service", "$9,000", "excl. GST"},
	))
	ev := Harvest(name, body)
	id := ""
	for _, a := range ev.Amounts {
		if a.Display == "$9,000" {
			id = a.ID
		}
	}
	fake := &fakeJev{conf: 0.99, answers: map[string]string{"price:stage:report": id, "price:stage:advice": choiceNotPrice}}
	out, err := Read(context.Background(), fake, greenAt(0.9), name, body, context1(), nil)
	if err != nil || id == "" {
		t.Fatalf("id %q err %v", id, err)
	}
	amber := 0
	for _, p := range out.Prices {
		if strings.HasPrefix(p.Key, "stage:") && p.Band == BandGreen {
			t.Fatalf("%s green from one lump sum: %+v", p.Key, p)
		}
		if p.Key == "stage:report" && p.Band == BandAmber {
			amber++
		}
	}
	if amber != 1 {
		t.Fatalf("report not read amber: %+v", out.Prices)
	}
}
