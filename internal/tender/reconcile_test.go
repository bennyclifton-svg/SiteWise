package tender

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// Every figure in these tests is invented.

var testRows = []Row{{Key: "stage:prelims", Label: "Preliminaries"}, {Key: "stage:structure", Label: "Structure"}, {Key: "stage:services", Label: "Services"}}

func ln(label, amt, kind, where, group, row string) RawLine {
	st := "priced"
	if amt == "" {
		st = "included"
	}
	return RawLine{P: 1, Label: label, Amt: amt, G: "unknown", St: st, Kind: kind, Where: where, Group: group, Row: row, Sure: true}
}

// read reconciles lines against one page that prints every amount.
func read(t *testing.T, gst string, lines ...RawLine) Reading {
	t.Helper()
	var text []string
	for _, l := range lines {
		text = append(text, l.Label+" "+l.Amt)
	}
	return Reconcile(RawReading{Basis: "fixed", GST: gst, Lines: lines}, []Page{{Number: 1, Text: strings.Join(text, "\n")}}, testRows, nil)
}

func sumIs(t *testing.T, r Reading, want int64) {
	t.Helper()
	if r.TenderSum == nil || *r.TenderSum != want {
		t.Fatalf("tender sum = %v (%s), want %d", r.TenderSum, r.SumHow, want)
	}
}

func counted(r Reading) (n int, cents int64) {
	for _, l := range r.Lines {
		if l.Counted && l.Cents != nil {
			n++
			cents += *l.Cents
		}
	}
	return
}

func hasFlag(flags []Flag, code string) bool {
	for _, f := range flags {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestLinesThatAddUpToThePrintedTotal(t *testing.T) {
	r := read(t, "ex",
		ln("Site set-up", "$10,000.00", "base", "sum", "", "stage:prelims"),
		ln("Frame", "$60,000.00", "base", "sum", "", "stage:structure"),
		ln("Electrical", "$30,000.00", "base", "sum", "", "stage:services"),
		ln("Total", "$100,000.00", "total", "inside", "", ""))
	sumIs(t, r, 10000000)
	if n, c := counted(r); n != 3 || c != 10000000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
	if r.Gap != 0 || r.Remainder != 0 || r.Why != "" || len(r.Flags) != 0 {
		t.Fatalf("gap %d remainder %d why %q flags %v", r.Gap, r.Remainder, r.Why, r.Flags)
	}
}

func TestMarginPrintedAtTheFoot(t *testing.T) {
	margin := ln("Plus 10% builder margin", "$110,000.00", "total", "inside", "", "")
	margin.G = "ex"
	r := read(t, "ex",
		ln("Stage 1", "$60,000.00", "base", "sum", "Stage 1", "stage:structure"),
		ln("Stage 2", "$40,000.00", "base", "sum", "Stage 2", "stage:services"),
		ln("Price exc GST", "$100,000.00", "total", "inside", "", ""),
		margin)
	sumIs(t, r, 11000000)
	if r.Why != "margin:10" || r.Gap != 1000000 || r.Remainder != 1000000 {
		t.Fatalf("why %q gap %d remainder %d", r.Why, r.Gap, r.Remainder)
	}
	if hasFlag(r.Flags, FlagLinesShort) {
		t.Fatal("a margin at the foot is not a shortfall")
	}
}

func TestTenPerCentAboveTheLinesWithNoBasisIsNotCalledMargin(t *testing.T) {
	r := read(t, "unknown",
		ln("Fee", "$100,000.00", "base", "sum", "", "stage:structure"),
		ln("Total", "$100,000.00", "total", "inside", "", ""),
		ln("Total payable", "$110,000.00", "total", "inside", "", ""))
	sumIs(t, r, 10000000)
	if !hasFlag(r.Flags, FlagCheckTenderSum) {
		t.Fatal("want a flag asking whether the higher total is GST or margin")
	}
}

func TestAPricePrintedInPartsWithNoOverallTotal(t *testing.T) {
	r := read(t, "ex",
		ln("Block walls", "$30,000.00", "base", "sum", "Amenities", "stage:structure"),
		ln("Block services", "$20,000.00", "base", "sum", "Amenities", "stage:services"),
		ln("Contract sum", "$50,000.00", "total", "inside", "Amenities", ""),
		ln("Pool shell", "$10,000.00", "base", "sum", "Pool", "stage:structure"),
		ln("Pool plant", "$5,000.00", "base", "sum", "Pool", "stage:services"),
		ln("Total", "$15,000.00", "total", "inside", "Pool", ""))
	sumIs(t, r, 6500000)
	if r.Gap != 0 {
		t.Fatalf("gap %d", r.Gap)
	}
}

func TestPartsPlusABlockThatPrintsNoTotal(t *testing.T) {
	r := read(t, "ex",
		ln("Design fees", "$8,000.00", "base", "sum", "Fees", "stage:prelims"),
		ln("Pack 1 works", "$30,000.00", "base", "sum", "Pack 1", "stage:structure"),
		ln("Pack 1 works two", "$21,000.00", "base", "sum", "Pack 1", "stage:structure"),
		ln("Total cost", "$51,000.00", "total", "inside", "Pack 1", ""),
		ln("Pack 2 works", "$10,000.00", "base", "sum", "Pack 2", "stage:services"),
		ln("Pack 2 works two", "$6,000.00", "base", "sum", "Pack 2", "stage:services"),
		ln("Total cost", "$16,000.00", "total", "inside", "Pack 2", ""))
	sumIs(t, r, 7500000)
	if r.Gap != 0 || !hasFlag(r.Flags, FlagNoTotalPrinted) {
		t.Fatalf("gap %d flags %v", r.Gap, r.Flags)
	}
}

func TestTwoDisciplineTotalsUnderOneHeading(t *testing.T) {
	r := read(t, "ex",
		ln("Civil: schematic", "$19,500.00", "base", "sum", "Fees", "stage:prelims"),
		ln("Civil: detailed", "$25,500.00", "base", "sum", "Fees", "stage:prelims"),
		ln("Civil total", "$45,000.00", "total", "inside", "Fees", ""),
		ln("Structural: schematic", "$5,500.00", "base", "sum", "Fees", "stage:structure"),
		ln("Structural: detailed", "$38,000.00", "base", "sum", "Fees", "stage:structure"),
		ln("Structural total", "$43,500.00", "total", "inside", "Fees", ""))
	sumIs(t, r, 8850000)
	if n, c := counted(r); n != 4 || c != 8850000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
}

func TestSummaryAddsUpAndDetailDoesNot(t *testing.T) {
	r := read(t, "ex",
		ln("Supervision", "$10,000.00", "base", "sum", "1. Alpha", "stage:prelims"),
		ln("Insurance", "$20,000.00", "base", "sum", "1. Alpha", "stage:prelims"),
		// a third Alpha item of $5,000 was not listed
		ln("Frame", "$40,000.00", "base", "sum", "2. Beta", "stage:structure"),
		ln("1.0 Alpha", "$35,000.00", "subtotal", "inside", "Summary", ""),
		ln("2.0 Beta", "$40,000.00", "subtotal", "inside", "Summary", ""),
		ln("Sub Total", "$86,250.00", "subtotal", "inside", "Summary", ""))
	sumIs(t, r, 8625000)
	if n, c := counted(r); n != 2 || c != 7500000 {
		t.Fatalf("counted %d lines, %d cents; want the two summary lines", n, c)
	}
	if r.Why != "margin:15" || !hasFlag(r.Flags, FlagSummaryCounted) {
		t.Fatalf("why %q flags %v", r.Why, r.Flags)
	}
	for _, l := range r.Lines {
		if l.Label == "1.0 Alpha" && l.Row != "stage:prelims" {
			t.Fatalf("a counted subtotal takes its detail's row, got %q", l.Row)
		}
	}
}

func TestOneLineMarkedOutsideMakesItAddUp(t *testing.T) {
	r := read(t, "ex",
		ln("Works", "$100,000.00", "base", "sum", "", "stage:structure"),
		ln("Irrigation", "$20,000.00", "base", "outside", "", "stage:services"),
		ln("Total", "$120,000.00", "total", "inside", "", ""))
	sumIs(t, r, 12000000)
	if n, c := counted(r); n != 2 || c != 12000000 || r.Gap != 0 {
		t.Fatalf("counted %d lines, %d cents, gap %d", n, c, r.Gap)
	}
	if !hasFlag(r.Lines[1].Flags, FlagCountedToMatch) {
		t.Fatal("the line brought in must say so")
	}
}

func TestAnOutsideLineThatIsNotARoundPercentage(t *testing.T) {
	r := read(t, "ex",
		ln("Works", "$100,000.00", "base", "sum", "", "stage:structure"),
		ln("Irrigation", "$3,220.40", "ps", "outside", "", "stage:services"),
		ln("Bore", "$1,000.00", "ps", "outside", "", "stage:services"),
		ln("Total", "$104,220.40", "total", "inside", "", ""))
	sumIs(t, r, 10422040)
	if n, c := counted(r); n != 3 || c != 10422040 || r.Gap != 0 {
		t.Fatalf("counted %d lines, %d cents, gap %d; both allowances belong in the price", n, c, r.Gap)
	}
}

func TestOneLineCountedTwice(t *testing.T) {
	r := read(t, "ex",
		ln("Demolition price in the notes", "$30,000.00", "pc", "sum", "", "stage:prelims"),
		ln("Demolition and site", "$50,000.00", "base", "sum", "", "stage:prelims"),
		ln("Structure", "$30,000.00", "base", "sum", "", "stage:structure"),
		ln("Services", "$20,000.00", "base", "sum", "", "stage:services"),
		ln("Total", "$100,000.00", "total", "inside", "", ""))
	sumIs(t, r, 10000000)
	if n, c := counted(r); n != 3 || c != 10000000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
	if !hasFlag(r.Lines[0].Flags, FlagBreakdown) {
		t.Fatal("the allowance is the line to leave out")
	}
}

func TestTheSamePricesListedTwice(t *testing.T) {
	r := read(t, "ex",
		ln("Electrical: schematic", "$10,000.00", "base", "sum", "Fee table", "stage:services"),
		ln("Mechanical: schematic", "$20,000.00", "base", "sum", "Fee table", "stage:structure"),
		ln("Stage 1", "$10,000.00", "base", "sum", "Electrical form", "stage:services"),
		ln("Stage 1", "$20,000.00", "base", "sum", "Mechanical form", "stage:structure"),
		ln("Total", "$30,000.00", "total", "inside", "Fee table", ""))
	sumIs(t, r, 3000000)
	if n, c := counted(r); n != 2 || c != 3000000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
	if !hasFlag(r.Lines[2].Flags, FlagRepeat) || !hasFlag(r.Lines[3].Flags, FlagRepeat) {
		t.Fatal("the later copies are the repeats")
	}
}

func TestAnAmountNotOnThePageIsThrownAway(t *testing.T) {
	raw := RawReading{GST: "ex", Lines: []RawLine{ln("Works", "$5,000.00", "base", "sum", "", "stage:structure")}}
	r := Reconcile(raw, []Page{{Number: 1, Text: "Works as described"}}, testRows, nil)
	l := r.Lines[0]
	if l.Cents != nil || l.Printed != "" || l.Counted || !hasFlag(l.Flags, FlagAmountNotFound) {
		t.Fatalf("line kept an amount the page does not print: %+v", l)
	}
	if r.TenderSum != nil {
		t.Fatal("no tender sum can come from a thrown-away amount")
	}
}

func TestAnAmountOnTheNextPageCorrectsThePage(t *testing.T) {
	raw := RawReading{GST: "ex", Lines: []RawLine{ln("Works", "$5,000.00", "base", "sum", "", "stage:structure")}}
	r := Reconcile(raw, []Page{{Number: 1, Text: "Letter"}, {Number: 2, Text: "Works $5,000.00", OCR: true}}, testRows, nil)
	l := r.Lines[0]
	if l.Page != 2 || l.Cents == nil || *l.Cents != 500000 || !hasFlag(l.Flags, FlagFromScan) {
		t.Fatalf("%+v", l)
	}
}

func TestALumpSumWithAllowancesOnly(t *testing.T) {
	total := ln("Pricing proposal inclusive of GST", "$1,100,000.00", "total", "inside", "", "")
	total.G = "inc"
	r := read(t, "inc",
		total,
		ln("Windows", "$110,000.00", "ps", "sum", "Provisional sums", "stage:structure"),
		ln("Joinery", "$220,000.00", "ps", "sum", "Provisional sums", "stage:services"))
	sumIs(t, r, 100000000)
	if n, c := counted(r); n != 2 || c != 30000000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
	if r.Why != "unbroken" || r.Remainder != 70000000 || hasFlag(r.Flags, FlagLinesShort) {
		t.Fatalf("why %q remainder %d flags %v", r.Why, r.Remainder, r.Flags)
	}
}

func TestDetailThatIsCloseIsKeptAndFlagged(t *testing.T) {
	r := read(t, "ex",
		ln("Trade one", "$100,000.00", "base", "sum", "Pack 3", "stage:structure"),
		ln("Trade two", "$200,000.00", "base", "sum", "Pack 3", "stage:services"),
		// a third trade of $5,000 was not listed
		ln("Sub total", "$305,000.00", "subtotal", "inside", "Pack 3", ""),
		ln("Tender sum", "$305,000.00", "total", "inside", "", ""))
	sumIs(t, r, 30500000)
	if n, c := counted(r); n != 2 || c != 30000000 {
		t.Fatalf("counted %d lines, %d cents; the detail is kept", n, c)
	}
	if r.Remainder != 500000 || !hasFlag(r.Flags, FlagGroupDiffers) || !hasFlag(r.Flags, FlagLinesShort) {
		t.Fatalf("remainder %d flags %v", r.Remainder, r.Flags)
	}
}

func TestPartsPricedOnlyAsTotals(t *testing.T) {
	r := read(t, "ex",
		ln("Pack 1 total", "$100,000.00", "subtotal", "inside", "Pack 1", "stage:structure"),
		ln("Pack 2 total", "$50,000.00", "subtotal", "inside", "Pack 2", "stage:services"),
		ln("Tender sum", "$150,000.00", "total", "inside", "", ""))
	sumIs(t, r, 15000000)
	if n, c := counted(r); n != 2 || c != 15000000 || r.Gap != 0 {
		t.Fatalf("counted %d lines, %d cents, gap %d", n, c, r.Gap)
	}
}

func TestASubtotalTheModelCalledScope(t *testing.T) {
	r := read(t, "ex",
		ln("Item one", "$10,000.00", "base", "sum", "Section", "stage:structure"),
		ln("Item two", "$20,000.00", "base", "sum", "Section", "stage:structure"),
		ln("Item three", "$30,000.00", "base", "sum", "Section", "stage:structure"),
		ln("Section total", "$60,000.00", "base", "sum", "Section", "stage:structure"))
	if n, c := counted(r); n != 3 || c != 6000000 {
		t.Fatalf("counted %d lines, %d cents; the section total must not be counted", n, c)
	}
	if r.Lines[3].Kind != "subtotal" {
		t.Fatalf("kind %q", r.Lines[3].Kind)
	}
}

func TestAThirdLineEqualToTheTwoBeforeItIsStillScope(t *testing.T) {
	r := read(t, "ex",
		ln("Item one", "$10,000.00", "base", "sum", "Section", "stage:structure"),
		ln("Item two", "$20,000.00", "base", "sum", "Section", "stage:structure"),
		ln("Item three", "$30,000.00", "base", "sum", "Section", "stage:structure"))
	if n, c := counted(r); n != 3 || c != 6000000 {
		t.Fatalf("counted %d lines, %d cents", n, c)
	}
}

func TestRowsAreCheckedAgainstTheSchedule(t *testing.T) {
	unsure := ln("Fittings", "$1,000.00", "base", "sum", "", "stage:services")
	unsure.Sure = false
	r := read(t, "ex", ln("Works", "$2,000.00", "base", "sum", "", "stage:nowhere"), unsure)
	if r.Lines[0].Row != "" || !hasFlag(r.Lines[0].Flags, FlagRowUnknown) {
		t.Fatalf("%+v", r.Lines[0])
	}
	if !hasFlag(r.Lines[1].Flags, FlagRowUnsure) {
		t.Fatalf("%+v", r.Lines[1])
	}
}

func TestPricesThatIncludeGSTAreConverted(t *testing.T) {
	total := ln("Total ex GST", "$200.00", "total", "inside", "", "")
	total.G = "ex"
	r := read(t, "inc",
		ln("One", "$110.00", "base", "sum", "", "stage:structure"),
		ln("Two", "$110.00", "base", "sum", "", "stage:services"),
		total)
	sumIs(t, r, 20000)
	if _, c := counted(r); c != 20000 || r.Gap != 0 {
		t.Fatalf("counted %d cents, gap %d", c, r.Gap)
	}
}

func TestOptionsSavingsAndRatesStayOut(t *testing.T) {
	r := read(t, "ex",
		ln("Works", "$100,000.00", "base", "sum", "", "stage:structure"),
		ln("Option: better grout", "$18,000.00", "option", "outside", "", ""),
		ln("Saving: laminate tops", "$6,500.00", "saving", "outside", "", ""),
		ln("Director", "$350.00 per hour", "rate", "outside", "", ""),
		ln("Asbestos removal", "$1,100.00 per tonne plus GST", "ps", "outside", "", ""),
		ln("Total", "$100,000.00", "total", "inside", "", ""))
	sumIs(t, r, 10000000)
	if n, _ := counted(r); n != 1 {
		t.Fatalf("counted %d lines", n)
	}
	if *r.Lines[2].Cents != -650000 {
		t.Fatalf("a saving is negative, got %d", *r.Lines[2].Cents)
	}
	if r.Lines[4].Kind != "rate" {
		t.Fatalf("an allowance quoted per tonne is a rate, got %q", r.Lines[4].Kind)
	}
	if !hasFlag(r.Flags, FlagOptionsOffered) {
		t.Fatal("want the options flag")
	}
}

func TestNoTotalPrintedUsesTheTotalRulesFound(t *testing.T) {
	stated := int64(9000000)
	raw := RawReading{GST: "ex", Lines: []RawLine{ln("Works", "$50,000.00", "base", "sum", "", "stage:structure")}}
	r := Reconcile(raw, []Page{{Number: 1, Text: "Works $50,000.00"}}, testRows, &stated)
	sumIs(t, r, 9000000)
	if r.Remainder != 4000000 {
		t.Fatalf("remainder %d", r.Remainder)
	}
}

func TestCellsAddUpToTheTenderSum(t *testing.T) {
	r := read(t, "ex",
		ln("Site set-up", "$10,000.00", "base", "sum", "", "stage:prelims"),
		ln("Frame", "$60,000.00", "base", "sum", "", "stage:structure"),
		ln("Sundries", "$5,000.00", "base", "sum", "", ""),
		ln("Electrical", "", "base", "sum", "", "stage:services"),
		ln("Total", "$80,000.00", "total", "inside", "", ""))
	cells := Cells(r, testRows)
	var sum int64
	for _, c := range cells {
		if c.Cents != nil {
			sum += *c.Cents
		}
	}
	if sum != 8000000 {
		t.Fatalf("cells add to %d, want the tender sum", sum)
	}
	if c := cells[SystemRow]; c.Cents == nil || *c.Cents != 1000000 || c.Lines != 1 {
		t.Fatalf("system row %+v", c)
	}
	if cells["stage:services"].State != StateIncluded {
		t.Fatalf("services %+v", cells["stage:services"])
	}
}

func TestRecountAfterAnEdit(t *testing.T) {
	r := read(t, "ex",
		ln("Works", "$100,000.00", "base", "sum", "", "stage:structure"),
		ln("Total", "$100,000.00", "total", "inside", "", ""))
	r.Lines[0].Counted = false
	Recount(&r)
	if r.Gap != 10000000 || r.Why != "unbroken" {
		t.Fatalf("gap %d why %q", r.Gap, r.Why)
	}
	r.Lines[0].Counted = true
	Recount(&r)
	if r.Gap != 0 || len(r.Flags) != 0 {
		t.Fatalf("gap %d flags %v", r.Gap, r.Flags)
	}
}

func TestParsePrinted(t *testing.T) {
	for in, want := range map[string]int64{"$1,279,781": 127978100, "$ 11, 950": 1195000, "$45 500.00": 4550000, "-$   6,500.00": -650000,
		"$9,580-00": 958000, "3,1250.00": 3125000, "$5,000.00 Provisional Sum": 500000, "(1,200.50)": -120050} {
		if got, ok, _ := parsePrinted(in); !ok || got != want {
			t.Errorf("parsePrinted(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "Included", "N/A", "$", "TBA"} {
		if _, ok, _ := parsePrinted(in); ok {
			t.Errorf("parsePrinted(%q) should find no amount", in)
		}
	}
	if _, _, rate := parsePrinted("$350.00 / hr"); !rate {
		t.Error("an hourly figure is a rate")
	}
}

func TestPageFiguresSeeSpacedThousands(t *testing.T) {
	f := pageFigures("TOTAL FEES $ 45 500.00 and 2,400 x 300 windows")
	if !f[4550000] || !f[240000] || !f[30000] {
		t.Fatalf("%v", f)
	}
}

func TestNormGroup(t *testing.T) {
	if a, b := normGroup("2. PROJECT MANAGEMENT"), normGroup("2.0 Project Management"); a != b || a == "" {
		t.Fatalf("%q vs %q", a, b)
	}
	if normGroup("SUB TOTAL (ex GST)") != "" {
		t.Fatal("a pure total has no name")
	}
	if normGroup("Stage 1") == normGroup("Stage 2") {
		t.Fatal("numbers tell groups apart")
	}
}

// bigReading is a 330-line tender: six packs of 54 lines and a subtotal each.
func bigReading() (RawReading, []Page) {
	var lines []RawLine
	var text []string
	for pack := 0; pack < 6; pack++ {
		group := "Pack " + string(rune('1'+pack))
		var sub int64
		for i := 0; i < 54; i++ {
			c := int64(10000+i*137+pack*11) * 100
			sub += c
			amt := dollars(c)
			lines = append(lines, ln("Trade line", amt, "base", "sum", group, testRows[i%3].Key))
			text = append(text, amt)
		}
		lines = append(lines, ln("Sub total", dollars(sub), "subtotal", "inside", group, ""))
		text = append(text, dollars(sub))
	}
	return RawReading{GST: "ex", Lines: lines}, []Page{{Number: 1, Text: strings.Join(text, "\n")}}
}

func BenchmarkReconcile330Lines(b *testing.B) {
	raw, pages := bigReading()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Reconcile(raw, pages, testRows, nil)
	}
}

// TestReconcileBudget holds the tender_reconcile row of bench/budgets.json:
// checks, reconciling and cells for 330 lines, with no model.
func TestReconcileBudget(t *testing.T) {
	raw, pages := bigReading()
	times := make([]time.Duration, 20)
	for i := range times {
		began := time.Now()
		Cells(Reconcile(raw, pages, testRows, nil), testRows)
		times[i] = time.Since(began)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	p50, p90 := times[9], times[17]
	if p50 > 100*time.Millisecond || p90 > 250*time.Millisecond {
		t.Fatalf("tender_reconcile p50 %s (budget 100ms), p90 %s (budget 250ms)", p50, p90)
	}
	t.Logf("tender_reconcile p50 %s, p90 %s", p50, p90)
}
