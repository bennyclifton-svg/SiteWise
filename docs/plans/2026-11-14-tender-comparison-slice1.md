# Tender Comparison Slice 1: The Reconciler and the Gate

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Go turns the model's listed lines into a checked reading whose column always equals the firm's tender sum, and an answer key proves how right the whole pass is before any screen is built.

**Architecture:** `tender.Reconcile` is a pure function: lines in, reading out, no model and no database. `cmd/tender-eval` replays recorded answers through it and scores them against a private answer key. The slice ends with a go or no-go for the owner.

**Tech Stack:** Go only.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md`. **Design:** `docs/design/2026-11-14-tender-comparison.md` section 7.1 (rules R1 to R9) and section 15 (pass marks).

**Status:** written 14 November 2026. The code in tasks 1 to 3 was written and tested before this plan: 24 tests pass, and replayed over the 22 real trial answers it gets the tender sum right for 20 of 21. Tasks 4 to 8 have not been compiled or run.

---

### Task 0: Check slice 0 is in

Run `go test ./internal/tender/... ./internal/llm/... -count=1`. Expected: PASS. `internal/tender/types.go` must hold contract C2 of the master plan, including `Reading.Gap`, the flag constants and `SystemRow`. If it does not, bring it in line with C2 first.

### Task 1: The tests, failing

**Files:**
- Create: `internal/tender/reconcile_test.go`

The reconciler was built test-first before this plan was written. Bring its tests in unchanged, then its code. Do not rewrite either. Read both until you can say what each rule does and which test pins it.

**Step 1:** Create `internal/tender/reconcile_test.go` with exactly this:

```go
package tender

import (
	"strings"
	"testing"
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

func BenchmarkReconcile330Lines(b *testing.B) {
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
	pages := []Page{{Number: 1, Text: strings.Join(text, "\n")}}
	raw := RawReading{GST: "ex", Lines: lines}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Reconcile(raw, pages, testRows, nil)
	}
}
```

**Step 2:** Run `go test ./internal/tender -count=1`. Expected: FAIL to build, `Reconcile`, `Cells`, `Recount`, `parsePrinted`, `pageFigures`, `normGroup` and `dollars` undefined.

**Step 3:** Do not commit yet.

### Task 2: Amounts

**Files:**
- Create: `internal/tender/amount.go`

**Step 1:** Create it with exactly this:

```go
package tender

import (
	"strconv"
	"strings"
)

// A dollar: the tolerance when two sums are compared.
const tol = 100

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// scanAmount reads one amount starting at s[i], which must be a digit. It
// accepts thousands commas ("1,279,781", "11, 950"), and, when loose, a
// space before a group of three digits ("45 500.00"). Decimals follow a dot
// or a dash ("9,580-00"). It returns the cents and the index after it.
func scanAmount(s string, i int, loose bool) (cents int64, end int, ok bool) {
	var digits []byte
	j := i
scan:
	for j < len(s) {
		c := s[j]
		switch {
		case isDigit(c):
			digits = append(digits, c)
			j++
		case c == ',' && j+1 < len(s) && isDigit(s[j+1]):
			j++
		case c == ',' && j+2 < len(s) && s[j+1] == ' ' && isDigit(s[j+2]):
			j += 2
		case loose && c == ' ' && threeDigitsThenStop(s, j+1):
			j++
		default:
			break scan
		}
	}
	if len(digits) == 0 || len(digits) > 13 {
		return 0, j, false
	}
	n, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil {
		return 0, j, false
	}
	cents = n * 100
	if j < len(s) && (s[j] == '.' || s[j] == '-') {
		k := j + 1
		for k < len(s) && isDigit(s[k]) {
			k++
		}
		if d := k - (j + 1); d == 1 || d == 2 {
			frac, _ := strconv.ParseInt(s[j+1:k], 10, 64)
			if d == 1 {
				frac *= 10
			}
			cents += frac
			j = k
		}
	}
	return cents, j, true
}

func threeDigitsThenStop(s string, i int) bool {
	if i+3 > len(s) {
		return false
	}
	for k := i; k < i+3; k++ {
		if !isDigit(s[k]) {
			return false
		}
	}
	return i+3 == len(s) || !isDigit(s[i+3])
}

// parsePrinted reads an amount as the model copied it. rate is true when
// words after the figure make it a rate ("per tonne", "/hr", "each").
func parsePrinted(s string) (cents int64, ok, rate bool) {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return 0, false, false
	}
	c, end, ok := scanAmount(s, i, true)
	if !ok {
		return 0, false, false
	}
	if strings.ContainsAny(s[:i], "-(") {
		c = -c
	}
	rest := strings.ToLower(strings.TrimLeft(s[end:], " )"))
	for _, w := range []string{"/", "per ", "p/", "each", "ea.", "ea ", "an hour", "hourly"} {
		if strings.HasPrefix(rest, w) {
			rate = true
		}
	}
	return c, true, rate
}

// pageFigures is every amount that appears in a page's text, in cents,
// without sign. It is what R3 checks a returned amount against.
func pageFigures(text string) map[int64]bool {
	out := map[int64]bool{}
	for i := 0; i < len(text); i++ {
		if !isDigit(text[i]) || (i > 0 && isDigit(text[i-1])) {
			continue
		}
		if c, _, ok := scanAmount(text, i, false); ok {
			out[c] = true
		}
		if c, _, ok := scanAmount(text, i, true); ok {
			out[c] = true
		}
	}
	return out
}

// exGST takes one eleventh off an amount that includes GST, to the cent.
func exGST(inc int64) int64 {
	if inc < 0 {
		return -exGST(-inc)
	}
	return (inc*10 + 5) / 11
}

// dollars writes cents for a flag, e.g. "$83,697.00" or "-$120.50".
func dollars(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	whole := strconv.FormatInt(c/100, 10)
	var b strings.Builder
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	frac := strconv.FormatInt(c%100, 10)
	if len(frac) < 2 {
		frac = "0" + frac
	}
	return sign + "$" + b.String() + "." + frac
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func near(a, b int64) bool { return abs64(a-b) <= tol }
```

What to notice:

- `scanAmount` reads the figures firms really print: `$1,279,781`, `$ 11, 950`, `$45 500.00`, `$9,580-00`.
- `pageFigures` is the set rule R3 checks against. It is deliberately generous (it reads each number both with and without a space as a thousands separator), because its job is to refuse an amount that is nowhere on the page, not to parse the page.
- `exGST` is the same arithmetic as `exGST` in `internal/proposals/read.go`. Do not import that one: it is unexported, and this package must not depend on the old reader.

**Step 2:** The package still does not build. Go on.

### Task 3: The reconciler and the cells

**Files:**
- Create: `internal/tender/reconcile.go`
- Create: `internal/tender/cells.go`

**Step 1:** Create `internal/tender/reconcile.go` with exactly this:

```go
package tender

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Margins a builder adds at the foot of a price, in per cent.
var marginRates = []float64{5, 7.5, 10, 12, 12.5, 15, 17.5, 20, 25}

// closeEnough: detail within this share of the tender sum is kept as the
// counted level even when it does not add up exactly (design section 7.1).
const closeEnough = 0.05

var (
	dotZero   = regexp.MustCompile(`(\d+)\.0\b`)
	wordToken = regexp.MustCompile(`[a-z0-9]+`)
	stopWords = map[string]bool{"the": true, "and": true, "of": true, "to": true, "for": true, "ex": true, "exc": true, "excl": true,
		"excluding": true, "exclusive": true, "inc": true, "incl": true, "including": true, "inclusive": true, "gst": true, "sub": true,
		"subtotal": true, "total": true, "cost": true, "costs": true, "price": true, "amount": true, "sum": true}
)

// normGroup reduces a heading or a label to the words that name it, so
// "2. PROJECT MANAGEMENT" and "2.0 Project Management" are the same group.
func normGroup(s string) string {
	s = dotZero.ReplaceAllString(strings.ToLower(s), "$1")
	var out []string
	for _, w := range wordToken.FindAllString(s, -1) {
		if !stopWords[w] {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func allowance(kind string) bool { return kind == "ps" || kind == "pc" }
func scope(kind string) bool     { return kind == "base" || allowance(kind) }

func (l *Line) flag(code, text string) { l.Flags = append(l.Flags, Flag{code, text}) }

// checkLines applies R1 to R3 and the row checks to what the model returned.
func checkLines(raw RawReading, pages []Page, rows []Row) []Line {
	figures := map[int]map[int64]bool{}
	ocr := map[int]bool{}
	for _, p := range pages {
		figures[p.Number] = pageFigures(p.Text)
		ocr[p.Number] = p.OCR
	}
	known := map[string]bool{}
	for _, r := range rows {
		known[r.Key] = true
	}
	lines := make([]Line, 0, len(raw.Lines))
	for i, r := range raw.Lines {
		l := Line{Ordinal: i, Page: r.P, Label: r.Label, GST: r.G, Kind: r.Kind, Placed: r.Where, Heading: r.Group, Row: r.Row, Note: r.Note, DecidedBy: "model"}
		switch r.St {
		case "priced":
			l.State = StatePriced
		case "included":
			l.State = StateIncluded
		case "excluded":
			l.State = StateExcluded
		default:
			l.State = StateNotPriced
			if l.Note == "" && r.St == "na" {
				l.Note = "N/A as printed"
			}
			if l.Note == "" && r.St == "tbc" {
				l.Note = "To be confirmed"
			}
		}
		if l.GST == "unknown" || l.GST == "" {
			l.GST = raw.GST
		}
		if l.GST == "" {
			l.GST = "unknown"
		}
		if l.State == StatePriced {
			c, ok, rate := parsePrinted(r.Amt)
			page, found := 0, false
			if ok {
				page, found = findFigure(figures, r.P, abs64(c))
			}
			switch {
			case !ok:
				l.flag(FlagAmountNotFound, "No amount could be read for this line. Type it from the document.")
			case !found:
				// R3: the model supplies no amount.
				l.flag(FlagAmountNotFound, "This amount is not in the text of the page. Type it from the document.")
			default:
				l.Page, l.Printed = page, r.Amt
				if l.GST == "inc" {
					c = exGST(c)
				}
				if l.Kind == "saving" && c > 0 {
					c = -c
				}
				l.Cents = &c
				if ocr[page] {
					l.flag(FlagFromScan, "Read from a scanned page. Check the figure.")
				}
				if rate && scope(l.Kind) {
					l.Kind, l.Placed = "rate", "outside"
				}
			}
		}
		if l.Row != "" && !known[l.Row] {
			l.Row = ""
			l.flag(FlagRowUnknown, "The row chosen for this line is not in the schedule.")
		} else if !r.Sure && scope(l.Kind) {
			l.flag(FlagRowUnsure, "Check the row this line is in.")
		}
		lines = append(lines, l)
	}
	return lines
}

// findFigure looks for an amount on the cited page, then on the pages just
// before and after it, nearest first.
func findFigure(figures map[int]map[int64]bool, page int, cents int64) (int, bool) {
	for _, p := range []int{page, page - 1, page + 1, page - 2, page + 2} {
		if figures[p][cents] {
			return p, true
		}
	}
	return 0, false
}

type unit struct {
	cents int64
	group string
}

// agg is a printed subtotal or total and the group it adds up.
type agg struct {
	i       int    // index into the lines
	target  string // normalised group it adds; "" for the whole document
	doc     bool   // a total of the whole document (a tender sum candidate)
	matched bool   // the lines it adds were found and add up to it
}

// Reconcile turns the model's answer into a checked reading (rules R1 to
// R8). stated is the total the existing rules found, ex GST, or nil.
func Reconcile(raw RawReading, pages []Page, rows []Row, stated *int64) Reading {
	r := Reading{Firm: raw.Firm, Basis: raw.Basis, GST: raw.GST, Exclusions: raw.Exclusions, Notes: raw.Notes}
	r.Lines = checkLines(raw, pages, rows)
	L := r.Lines
	leaf := func(i int) bool {
		l := &L[i]
		return l.Cents != nil && l.State == StatePriced && l.Placed == "sum" && scope(l.Kind)
	}
	isAgg := func(i int) bool {
		return L[i].Cents != nil && (L[i].Kind == "subtotal" || L[i].Kind == "total")
	}
	group := func(i int) string { return normGroup(L[i].Heading) }

	// Which groups have priced detail.
	leaves := map[string]int{}
	for i := range L {
		if leaf(i) {
			leaves[group(i)]++
		}
	}
	// R4: walk the lines in order, checking each printed subtotal against the
	// detail before it, and catching a subtotal the model called scope.
	var open []unit
	var aggs []*agg
	collapse := func(idx []int, cents int64, g string) {
		keep := open[:0:0]
		drop := map[int]bool{}
		for _, k := range idx {
			drop[k] = true
		}
		for k, u := range open {
			if !drop[k] {
				keep = append(keep, u)
			}
		}
		open = append(keep, unit{cents, g})
	}
	// trailing finds the shortest run of at least min units at the end of
	// open, all in group g (any group when g is ""), that adds to cents.
	trailing := func(g string, cents int64, min int) []int {
		var sum int64
		var idx []int
		for k := len(open) - 1; k >= 0; k-- {
			if g != "" && open[k].group != g {
				break
			}
			sum += open[k].cents
			idx = append(idx, k)
			if len(idx) >= min && near(sum, cents) {
				return idx
			}
		}
		return nil
	}
	for i := range L {
		c := L[i].Cents
		switch {
		case leaf(i):
			g := group(i)
			looksLikeTotal := strings.Contains(strings.ToLower(L[i].Label), "total")
			if idx := trailing(g, *c, 2); *c != 0 && idx != nil && (len(idx) >= 3 || looksLikeTotal) {
				L[i].Kind, L[i].Placed = "subtotal", "inside"
				leaves[g]--
				collapse(idx, *c, g)
				aggs = append(aggs, &agg{i: i, target: g, matched: true})
				continue
			}
			open = append(open, unit{*c, g})
		case isAgg(i):
			a := &agg{i: i}
			h, lb := group(i), normGroup(L[i].Label)
			switch {
			case h != "" && leaves[h] > 0:
				a.target = h
			case lb != "" && leaves[lb] > 0:
				a.target = lb // a summary line for a group of detail elsewhere
			case lb == "" || (L[i].Kind == "total" && h == ""):
				a.doc = true
			case h != "":
				a.target = h // a part priced only as a total
			default:
				a.target = "label:" + lb
			}
			if a.doc {
				if idx := trailing("", *c, 1); idx != nil {
					a.matched = true
					collapse(idx, *c, "")
				}
			} else if idx := trailing(a.target, *c, 1); idx != nil {
				a.matched = true
				collapse(idx, *c, a.target)
			} else {
				// The detail may sit elsewhere in the document.
				var idx []int
				var sum int64
				for k, u := range open {
					if u.group == a.target {
						idx, sum = append(idx, k), sum+u.cents
					}
				}
				if len(idx) > 0 && near(sum, *c) {
					a.matched = true
					collapse(idx, *c, a.target)
				}
			}
			aggs = append(aggs, a)
		}
	}

	// The top printed figure of each group.
	top := map[string]*agg{}
	for _, a := range aggs {
		if a.doc {
			continue
		}
		// The largest printed figure of the group; on a tie the total wins.
		if t := top[a.target]; t == nil || *L[a.i].Cents > *L[t.i].Cents || (*L[a.i].Cents == *L[t.i].Cents && L[a.i].Kind == "total") {
			top[a.target] = a
		}
	}
	sum := func(set []int) int64 {
		var s int64
		for _, i := range set {
			s += *L[i].Cents
		}
		return s
	}
	// Priced scope lines of any placement, by group: a group that has some is
	// not "priced only as a subtotal", even when they are all breakdowns.
	scopeLines := map[string]int{}
	for i := range L {
		if L[i].Cents != nil && L[i].State == StatePriced && scope(L[i].Kind) {
			scopeLines[group(i)]++
		}
	}
	var detail, level1, level3 []int
	for i := range L {
		if leaf(i) {
			detail = append(detail, i)
			level1 = append(level1, i)
			if top[group(i)] == nil {
				level3 = append(level3, i)
			}
		}
	}
	for g, a := range top {
		// A part priced only as a subtotal. A named subtotal with no heading
		// (a column total of a grid, say) is one only when nothing else is priced.
		part := scopeLines[g] == 0 && (!strings.HasPrefix(g, "label:") || len(detail) == 0)
		switch {
		case leaves[g] > 0:
			level3 = append(level3, a.i)
		case part:
			level1 = append(level1, a.i)
			level3 = append(level3, a.i)
		}
	}
	sort.Ints(level1)
	sort.Ints(level3)

	// R7: the tender sum.
	d := sum(level1)
	tender, how, flagged := tenderSum(L, aggs, d, sum(level3), stated)
	r.TenderSum, r.SumHow = tender, how
	if flagged != "" {
		r.Flags = append(r.Flags, Flag{FlagCheckTenderSum, flagged})
	}
	if tender == nil {
		for _, i := range level1 {
			L[i].Counted = true
		}
		finish(&r)
		return r
	}

	// R6: which level is counted.
	chosen, level := level1, 1
	fits := func(s int64) bool { return near(s, *tender) || marginOf(s, *tender) != 0 }
	// R8's single fixes come before a margin is assumed: a line that equals
	// the gap exactly is stronger evidence than a round percentage.
	repaired := false
	if !near(d, *tender) {
		if repaired = repair(L, level1, *tender); repaired {
			d = extraCounted(L, level1)
			for _, i := range level1 {
				if !uncounted(L, i) {
					d += *L[i].Cents
				}
			}
		}
	}
	switch {
	case fits(d) || repaired:
	case float64(abs64(*tender-d)) <= closeEnough*float64(abs64(*tender)):
	case len(top) > 0 && fits(sum(level3)):
		chosen, level = level3, 3
	case len(detail) > 0 && len(detail) != len(level1) && fits(sum(detail)):
		chosen, level = detail, 2
	}
	for _, i := range chosen {
		if !uncounted(L, i) {
			L[i].Counted = true
		}
	}
	if level == 3 {
		r.Flags = append(r.Flags, Flag{FlagSummaryCounted, "The printed subtotals add up and the detail does not, so the subtotals are counted and the detail is shown beneath them."})
		// A counted subtotal with no row takes the row most of its detail is in.
		for g, a := range top {
			if L[a.i].Row == "" {
				L[a.i].Row = majorityRow(L, g, group, leaf)
			}
		}
	}
	if level == 1 {
		var differs []string
		for g, a := range top {
			if a.matched || leaves[g] == 0 {
				continue
			}
			var s int64
			for _, i := range detail {
				if group(i) == g {
					s += *L[i].Cents
				}
			}
			if diff := *L[a.i].Cents - s; !near(diff, 0) {
				words := "are " + dollars(diff) + " short of"
				if diff < 0 {
					words = "add to " + dollars(-diff) + " more than"
				}
				differs = append(differs, fmt.Sprintf("%s: its lines %s its printed subtotal", L[a.i].Heading, words))
			}
		}
		sort.Strings(differs)
		for _, text := range differs {
			r.Flags = append(r.Flags, Flag{FlagGroupDiffers, text})
		}
	}
	finish(&r)
	return r
}

// uncounted reports whether a repair took this line out of the count.
func uncounted(L []Line, i int) bool {
	for _, f := range L[i].Flags {
		if f.Code == FlagBreakdown || f.Code == FlagRepeat {
			return true
		}
	}
	return false
}

// extraCounted is the sum of lines a repair brought into the count that
// were not in the level.
func extraCounted(L []Line, level []int) int64 {
	in := map[int]bool{}
	for _, i := range level {
		in[i] = true
	}
	var s int64
	for i := range L {
		if L[i].Counted && !in[i] && L[i].Cents != nil {
			s += *L[i].Cents
		}
	}
	return s
}

// marginOf returns the round margin, in per cent, that takes s to total, or 0.
func marginOf(s, total int64) float64 {
	if s <= 0 || total <= s {
		return 0
	}
	for _, rate := range marginRates {
		if near(int64(float64(s)*(1+rate/100)+0.5), total) {
			return rate
		}
	}
	return 0
}

// tenderSum applies R7. It returns the sum, how it was chosen, and words
// for a flag when it needs a look.
func tenderSum(L []Line, aggs []*agg, d, parts int64, stated *int64) (*int64, string, string) {
	type cand struct {
		cents int64
		ownEx bool // the line itself says ex GST
	}
	var totals []cand
	add := func(c int64, ownEx bool) {
		for k := range totals {
			if near(totals[k].cents, c) {
				totals[k].ownEx = totals[k].ownEx || ownEx
				return
			}
		}
		totals = append(totals, cand{c, ownEx})
	}
	// Lines printed ex GST first, so an inc GST copy of the same total folds into it.
	anyDoc := false
	for pass := 0; pass < 2; pass++ {
		for _, a := range aggs {
			l := &L[a.i]
			if ex := l.GST == "ex"; (a.doc || l.Kind == "total") && ex == (pass == 0) {
				add(*l.Cents, ex)
				anyDoc = anyDoc || a.doc
			}
		}
	}
	pick := func(c int64, how string) (*int64, string, string) { return &c, how, "" }
	var all int64
	for _, t := range totals {
		all += t.cents
	}
	cands := append([]cand{}, totals...)
	if len(totals) > 1 && !anyDoc {
		cands = append(cands, cand{all, true})
	}
	ambiguous := ""
	if d > 0 {
		for _, t := range cands {
			if rate := marginOf(d, t.cents); rate != 0 {
				if rate == 10 && !t.ownEx {
					ambiguous = "A printed total is 10% above the lines. Check whether that is GST or margin."
					continue
				}
				return pick(t.cents, "the lines plus "+trimRate(rate)+"% margin printed at the foot")
			}
		}
		for k, t := range cands {
			if near(t.cents, d) {
				how := "the printed total, which the lines add up to"
				if k >= len(totals) {
					how = "no overall total is printed; the parts' totals add up to the lines"
				}
				c := t.cents
				return &c, how, ambiguous
			}
		}
	}
	switch {
	case len(totals) == 1:
		return pick(totals[0].cents, "one printed total")
	case len(totals) > 1:
		for _, t := range totals {
			if near(all-t.cents, t.cents) {
				return pick(t.cents, "the printed total of the parts")
			}
		}
		// Every printed total belongs to a part and none is the whole: add
		// each part's printed figure, and the lines of any part without one.
		if !anyDoc {
			return pick(parts, "no overall total is printed; this is the sum of the parts")
		}
		largest := totals[0].cents
		for _, t := range totals {
			if t.cents > largest {
				largest = t.cents
			}
		}
		c := largest
		return &c, "the largest of several printed totals", "Several totals are printed and none matches the lines. Check the tender sum."
	}
	if stated != nil {
		c, how, _ := pick(*stated, "the total found on the document")
		return c, how, ""
	}
	if d > 0 {
		return &d, "no total is printed; this is the sum of the lines", ""
	}
	return nil, "", ""
}

func trimRate(r float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(strconv.FormatFloat(r, 'f', 1, 64), "0"), ".")
}

// repair applies R8's single fixes to the level and reports whether the
// level then adds up to the tender sum.
func repair(L []Line, level []int, tender int64) bool {
	var s int64
	in := map[int]bool{}
	for _, i := range level {
		s += *L[i].Cents
		in[i] = true
	}
	gap := tender - s
	if gap > 0 {
		// One line the model put outside the price makes it add up.
		var outside []int
		for i := range L {
			if !in[i] && L[i].Cents != nil && L[i].State == StatePriced && L[i].Placed == "outside" && scope(L[i].Kind) {
				outside = append(outside, i)
			}
		}
		for _, i := range outside {
			if near(*L[i].Cents, gap) {
				L[i].Counted = true
				L[i].flag(FlagCountedToMatch, "Counted so the lines add up to the tender sum. Check it belongs in the price.")
				return true
			}
		}
		var allow int64
		var allowances []int
		for _, i := range outside {
			if allowance(L[i].Kind) {
				allow += *L[i].Cents
				allowances = append(allowances, i)
			}
		}
		if len(allowances) > 1 && near(allow, gap) {
			for _, i := range allowances {
				L[i].Counted = true
				L[i].flag(FlagCountedToMatch, "Counted so the lines add up to the tender sum. Check it belongs in the price.")
			}
			return true
		}
		return false
	}
	// One counted line is a breakdown of another.
	pick := -1
	for _, i := range level {
		if near(*L[i].Cents, -gap) && (pick < 0 || allowance(L[i].Kind) || !allowance(L[pick].Kind)) {
			pick = i
		}
	}
	if pick >= 0 {
		L[pick].flag(FlagBreakdown, "Left out of the count: with it the lines add to more than the tender sum.")
		return true
	}
	// The same prices listed twice.
	seen := map[string]bool{}
	var dups []int
	var dupSum int64
	for _, i := range level {
		key := L[i].Row
		if key == "" {
			key = normGroup(L[i].Label)
		}
		key = strconv.FormatInt(*L[i].Cents, 10) + "|" + key
		if seen[key] {
			dups = append(dups, i)
			dupSum += *L[i].Cents
		}
		seen[key] = true
	}
	if len(dups) > 0 && near(dupSum, -gap) {
		for _, i := range dups {
			L[i].flag(FlagRepeat, "Left out of the count: the same price is listed earlier.")
		}
		return true
	}
	return false
}

func majorityRow(L []Line, g string, group func(int) string, leaf func(int) bool) string {
	byRow := map[string]int64{}
	var all int64
	for i := range L {
		if leaf(i) && group(i) == g {
			byRow[L[i].Row] += *L[i].Cents
			all += *L[i].Cents
		}
	}
	for row, c := range byRow {
		if row != "" && c*2 > all {
			return row
		}
	}
	return ""
}

// finish sets the gap, the remainder and the column flags from the counted lines.
func finish(r *Reading) {
	r.Gap, r.Remainder, r.Why = 0, 0, ""
	var counted, onRows int64
	baseCounted, offered, gstUnknown := false, false, false
	for i := range r.Lines {
		l := &r.Lines[i]
		if l.Kind == "option" || l.Kind == "saving" {
			offered = true
		}
		if !l.Counted || l.Cents == nil {
			continue
		}
		counted += *l.Cents
		if l.Row != "" {
			onRows += *l.Cents
		}
		baseCounted = baseCounted || l.Kind == "base" || l.Kind == "subtotal" || l.Kind == "total"
		gstUnknown = gstUnknown || l.GST == "unknown"
	}
	if r.Basis == "estimate" || r.Basis == "cost_plus" || r.Basis == "rates_only" {
		r.Flags = append(r.Flags, Flag{FlagNotFixedPrice, "This is not a fixed price: " + map[string]string{"estimate": "the firm calls it an estimate", "cost_plus": "it is a cost-plus offer", "rates_only": "only rates are given"}[r.Basis] + "."})
	}
	if gstUnknown {
		r.Flags = append(r.Flags, Flag{FlagGSTUnknown, "The document does not say whether its prices include GST. They are treated as excluding GST."})
	}
	if offered {
		r.Flags = append(r.Flags, Flag{FlagOptionsOffered, "The firm offers options, alternatives or savings. They are not in the tender sum."})
	}
	if r.TenderSum == nil {
		return
	}
	if strings.HasPrefix(r.SumHow, "no total is printed") || strings.HasPrefix(r.SumHow, "no overall total") {
		r.Flags = append(r.Flags, Flag{FlagNoTotalPrinted, "The firm prints no single total. The tender sum is added up from what it prints."})
	}
	r.Gap = *r.TenderSum - counted
	r.Remainder = *r.TenderSum - onRows
	if near(r.Gap, 0) && strings.HasPrefix(r.SumHow, "the lines plus") {
		r.SumHow = "the printed total, which the lines add up to"
	}
	switch rate := marginOf(counted, *r.TenderSum); {
	case r.Gap == 0:
	case near(r.Gap, 0):
		r.Why = "rounding"
	case rate != 0:
		r.Why = "margin:" + trimRate(rate)
	case !baseCounted || counted*2 < *r.TenderSum:
		r.Why = "unbroken" // a lump sum the firm did not break down, or broke down only in part
	case r.Gap > 0:
		r.Why = "differs"
		r.Flags = append(r.Flags, Flag{FlagLinesShort, "The lines are " + dollars(r.Gap) + " short of the tender sum."})
	default:
		r.Why = "differs"
		r.Flags = append(r.Flags, Flag{FlagLinesOver, "The lines add to " + dollars(-r.Gap) + " more than the tender sum. Some may be counted twice."})
	}
}
```

**Step 2:** Create `internal/tender/cells.go` with exactly this:

```go
package tender

// Cell is one firm's figure on one row of the first table.
type Cell struct {
	State   string // priced | included | excluded | not_priced
	Cents   *int64 // nil when priced with no known amount, or not priced
	Lines   int    // counted lines behind it
	ToCheck bool   // a line in it has a flag
	Note    string // for included, excluded or not priced: the firm's words
}

// Cells is R9: one reading as a column of the first table. The system row
// holds the remainder, so the priced cells always add up to the tender sum.
func Cells(r Reading, rows []Row) map[string]Cell {
	known := map[string]bool{}
	for _, row := range rows {
		known[row.Key] = true
	}
	out := map[string]Cell{}
	sums := map[string]int64{}
	sysLines, sysCheck := 0, false
	for _, l := range r.Lines {
		if !l.Counted || l.Cents == nil {
			continue
		}
		if l.Row == "" || !known[l.Row] {
			sysLines++
			sysCheck = sysCheck || len(l.Flags) > 0
			continue
		}
		c := out[l.Row]
		sums[l.Row] += *l.Cents
		c.State, c.Lines, c.ToCheck = StatePriced, c.Lines+1, c.ToCheck || len(l.Flags) > 0
		out[l.Row] = c
	}
	for key, s := range sums {
		c := out[key]
		v := s
		c.Cents = &v
		out[key] = c
	}
	// A row with no counted line shows what the firm said about it. A priced
	// line whose amount was thrown away outranks the rest: it needs typing.
	rank := map[string]int{StatePriced: 4, StateIncluded: 3, StateExcluded: 2, StateNotPriced: 1}
	for _, l := range r.Lines {
		if l.Counted || l.Row == "" || !known[l.Row] || !scope(l.Kind) || (l.State == StatePriced && l.Cents != nil) {
			continue
		}
		if c, ok := out[l.Row]; !ok || (c.Lines == 0 && rank[l.State] > rank[c.State]) {
			out[l.Row] = Cell{State: l.State, Note: l.Note, ToCheck: l.State == StatePriced}
		}
	}
	if r.TenderSum != nil && (r.Remainder != 0 || sysLines > 0) {
		rem := r.Remainder
		out[SystemRow] = Cell{State: StatePriced, Cents: &rem, Lines: sysLines, ToCheck: sysCheck || r.Why == "differs"}
	}
	return out
}

// Recount refreshes the gap, the remainder and the column flags after the
// manager has edited lines or the tender sum. It moves and uncounts nothing.
func Recount(r *Reading) {
	keep := r.Flags[:0]
	for _, f := range r.Flags {
		switch f.Code {
		case FlagLinesShort, FlagLinesOver, FlagNotFixedPrice, FlagGSTUnknown, FlagOptionsOffered, FlagNoTotalPrinted:
			// finish writes these again from the lines as they now stand.
		default:
			keep = append(keep, f)
		}
	}
	r.Flags = keep
	finish(r)
}
```

**Step 3:** Run `gofmt -l internal/tender`, `go vet ./internal/tender` and `go test ./internal/tender -count=1`. Expected: nothing listed, clean, PASS.

**Step 4: Read it against the design.** Where each rule lives:

| Rule | Code | Pinned by |
|---|---|---|
| R1 amounts, R2 GST | `checkLines`, `parsePrinted`, `exGST` | `TestParsePrinted`, `TestPricesThatIncludeGSTAreConverted` |
| R3 the model supplies no amount | `checkLines`, `findFigure`, `pageFigures` | `TestAnAmountNotOnThePageIsThrownAway`, `TestAnAmountOnTheNextPageCorrectsThePage` |
| Row checks | `checkLines` | `TestRowsAreCheckedAgainstTheSchedule` |
| R4 subtotals | the walk in `Reconcile` (`open`, `trailing`, `collapse`) | `TestASubtotalTheModelCalledScope`, `TestAThirdLineEqualToTheTwoBeforeItIsStillScope`, `TestTwoDisciplineTotalsUnderOneHeading` |
| R5 repeats | `repair`, the last block | `TestTheSamePricesListedTwice` |
| R6 level | `level1`, `level3`, `detail` and the `switch` after `repair` | `TestSummaryAddsUpAndDetailDoesNot`, `TestDetailThatIsCloseIsKeptAndFlagged`, `TestPartsPricedOnlyAsTotals` |
| R7 tender sum | `tenderSum` | `TestMarginPrintedAtTheFoot`, `TestTenPerCentAboveTheLinesWithNoBasisIsNotCalledMargin`, `TestAPricePrintedInPartsWithNoOverallTotal`, `TestPartsPlusABlockThatPrintsNoTotal`, `TestNoTotalPrintedUsesTheTotalRulesFound` |
| R8 the remainder | `repair`, `finish` | `TestOneLineMarkedOutsideMakesItAddUp`, `TestAnOutsideLineThatIsNotARoundPercentage`, `TestOneLineCountedTwice`, `TestALumpSumWithAllowancesOnly` |
| R9 cells | `Cells` | `TestCellsAddUpToTheTenderSum` |
| After an edit | `Recount` | `TestRecountAfterAnEdit` |

Three things in the code refine the design's wording. They came from replaying the real answers, and the design's intent is unchanged:

1. **R6 keeps detail that is close.** When the detail is within 5% of the tender sum it stays counted and the difference is flagged, pack by pack. Only when it is further out, and the printed subtotals do add up, are the subtotals counted instead. Collapsing a nearly-right schedule into pack totals would throw away what the manager came for.
2. **R8's single fixes run before a margin is assumed.** A line that equals the gap exactly is stronger evidence than a round percentage.
3. **"Not broken down" is not a fault.** When fewer than half the dollars are in counted lines, the reason is `unbroken` and no shortfall is flagged.

**Step 5:** Commit: `Tender: the reconciler, rules R1 to R9, with its tests`.

### Task 4: The reconcile budget

**Files:**
- Modify: `internal/tender/reconcile_test.go`

**Step 1: Write the test.** Add `TestReconcileBudget`: build the 330-line reading that `BenchmarkReconcile330Lines` builds (move the building into a helper both use), run `Reconcile` 20 times, sort the durations, and fail when the median is over 100 ms or the 18th of 20 is over 250 ms. The message must give both figures.

**Step 2:** Run `go test ./internal/tender -run TestReconcileBudget -count=1 -v`. Expected: PASS with a median near 2 ms. It was 1.4 ms when this plan was written.

**Step 3:** Add the row to `bench/budgets.json`, beside `price_table`:

```json
	{"name":"tender_reconcile","p50_us":100000,"p90_us":250000,"note":"Go work on one tender pass of 330 lines: checks, reconciling and cells; held by TestReconcileBudget (no model)"},
```

*Verify* that `bench/budgets.go` does not demand a sample for every row; if it does, mark the row the way other test-held rows are marked.

**Step 4:** Commit: `Tender: hold the reconcile budget in a test`.

### Task 5: The answer key and how a case is scored

**Files:**
- Create: `cmd/tender-eval/key.go`
- Create: `cmd/tender-eval/score.go`
- Test: `cmd/tender-eval/score_test.go`

The key lives at `data/eval/tender/private/answer-key.json` and is never committed.

**Step 1: The key's shape.** `key.go`:

```go
// answerKey is the private answer key. It names real firms and prices.
type answerKey struct {
	Version    int       `json:"version"`
	Reviewed   bool      `json:"reviewed"`    // true once the owner has checked it
	CorpusRoot string    `json:"corpus_root"` // "testing/rft"
	Cases      []keyCase `json:"cases"`
}

type keyCase struct {
	ID        string    `json:"id"`         // e.g. "camp-b1"; never printed without -show
	Path      string    `json:"path"`       // under CorpusRoot
	SHA256    string    `json:"sha256"`     // of the file; a changed file fails the run
	Schedules []string  `json:"schedules"`  // RFP ids for the rows, e.g. ["main-works"]
	Split     string    `json:"split"`      // "tune" or "hold"
	Scan      bool      `json:"scan"`       // the file has no text layer
	TenderSum string    `json:"tender_sum"` // decimal, ex GST; "" when the firm prints no single price
	Lines     []keyLine `json:"lines"`
}

type keyLine struct {
	Label   string `json:"label"`
	Amount  string `json:"amount"`  // decimal, ex GST, signed
	Row     string `json:"row"`     // schedule key; "" for not broken down
	Counted bool   `json:"counted"` // true: part of the tender sum
	Kind    string `json:"kind"`    // base | ps | pc | option | saving
}
```

`loadKey(path, corpusRoot)` reads it, checks every `sha256` against the file on disk, and returns an error naming the case id (not the path) when one differs.

A key line is one priced line of scope, an option or a saving. Subtotals, totals, GST lines and rates are not keyed.

**Step 2: Write the failing tests** for `scoreCase(k keyCase, r tender.Reading) caseScore` in `score_test.go`, with invented figures:

- `TestScoreRightSum`: key sum 100,000.00 and reading sum 10000000 cents gives `Sum == "right"`.
- `TestScoreNoSumExpected`: key sum `""` and a nil reading sum gives `"right"`.
- `TestScoreWrongSumWithAFlagIsFlagged`: sums differ and the reading carries `check_tender_sum`: `"flagged"`.
- `TestScoreWrongSumWithNoFlagIsSilent`: sums differ and no flag: `"silent"`.
- `TestScoreDollarsInTheRightRow`: three key lines of 50,000, 30,000 and 20,000 counted on rows A, B and C; the reading counts all three with the 20,000 on row B. `RowDollars == 80000_00`, `KeyDollars == 100000_00`.
- `TestScoreAMissingLineCountsAgainstRows`: the reading lacks the 30,000 line: `RowDollars == 70000_00` at most.
- `TestScoreOutsideMoneyCounted`: a key option of 18,000 (`counted: false`) that the reading counts with no `counted_to_match` flag gives `OutsideCounted == 1`. The same line counted with that flag gives 0.
- `TestScoreLinesAreMatchedOnce`: two key lines of the same amount match two different reading lines.
- `TestScoreAmountsKept`: a reading of ten priced lines of which one carries `amount_not_found` gives `AmountsKept == 9`, `AmountsAsked == 10`.

**Step 3:** Run. Expected: FAIL.

**Step 4: Implement.**

```go
type caseScore struct {
	Sum            string // "right", "flagged" or "silent"
	KeyDollars     int64  // sum of |amount| of the key's counted lines
	RowDollars     int64  // of those, matched to a counted reading line on the key's row
	FoundDollars   int64  // of those, matched to any reading line with the same amount
	OutsideCounted int    // key lines that are outside the price but counted, unflagged
	AmountsAsked   int    // reading lines the model gave an amount for
	AmountsKept    int    // of those, kept by R3
	Lines          int
}
```

Rules, in this order:

1. **Sum.** Parse `TenderSum` with `invoices.ParseCents`. Right when both are absent, or both present and within 100 cents. Otherwise flagged when the reading's tender sum is nil or its flags include any of `check_tender_sum`, `no_total_printed`, `lines_short`, `lines_over`, `reading_incomplete`. Otherwise silent.
2. **Matching.** Take the key lines largest first. For each, among reading lines not yet matched whose `Cents` is within 100 cents of the key amount, prefer one on the key's row, then one whose label shares the most words with the key label (`strings.Fields` of the lower-cased text), then the earliest. Each reading line matches once.
3. **Rows.** For each counted key line: add `|amount|` to `KeyDollars`; when matched, to `FoundDollars`; when matched to a counted line whose `Row` equals the key row, to `RowDollars`.
4. **Outside money.** A key line with `counted: false` matched to a reading line that is counted and has no `counted_to_match` flag adds one to `OutsideCounted`.
5. **Amounts.** `AmountsAsked` is the reading lines in state priced. `AmountsKept` is those without the `amount_not_found` flag.

**Step 5:** Run the tests. Expected: PASS.

**Step 6:** Commit: `Tender: the answer key and how one case is scored`.

### Task 6: Recording, replay and the report

**Files:**
- Create: `cmd/tender-eval/recording.go`
- Modify: `cmd/tender-eval/main.go`
- Test: `cmd/tender-eval/recording_test.go`

Model answers are recorded at the `llm.Asker` seam, not at HTTP: the SDK streams, and a streamed exchange is awkward to replay.

**Step 1: Write the failing tests.**

- `TestRecorderStoresAndReplays`: a recorder wrapping a fake asker writes one JSON line per call; a replayer built from that file returns the same response for the same request and `errReplayMiss` for a different one.
- `TestRecordingKeyIgnoresNothingThatMatters`: two requests that differ only in one page's text have different keys; two identical requests have the same key.

**Step 2: Implement.** One line per exchange in `data/eval/tender/private/recording.jsonl`:

```go
type record struct {
	Case      string    `json:"case"`
	Prompt    string    `json:"prompt"`   // tender.PromptVersion
	Model     string    `json:"model"`
	Request   string    `json:"request"`  // sha256 of the request: model, system, schema, every part
	Text      string    `json:"text"`
	Stop      string    `json:"stop"`
	InTokens  int64     `json:"in_tokens"`
	OutTokens int64     `json:"out_tokens"`
	LatencyMS int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	At        time.Time `json:"at"`
}
```

`recorder` implements `llm.Asker`, forwards to the real client and appends a record. `replayer` implements `llm.Asker` from the file, keyed by `Request`, and restores `Latency` from `LatencyMS`.

**Step 3: The modes of the command.**

```
go run ./cmd/tender-eval                 # replay every case, print the marks, write the aggregate
go run ./cmd/tender-eval -live           # call the model for cases with no recording under this prompt version
go run ./cmd/tender-eval -live -fresh    # call it for every case
go run ./cmd/tender-eval -split hold     # only the hold-out cases (default: tune)
go run ./cmd/tender-eval -show           # also print, to the terminal, each case's misses
go run ./cmd/tender-eval -dump <case>    # print a draft key for one case from its reading
go run ./cmd/tender-eval -one <file> -schedule <id>   # slice 0's mode, unchanged
```

For each case: extract, OCR the pages with no text when OCR is set up, build pages, run `tender.Pass` through the recorder or the replayer, `tender.Reconcile(raw, pages, rows, nil)`, `scoreCase`.

`-dump` prints the reading as key JSON: the tender sum, and one key line per line that has an amount and whose kind is base, ps, pc, option or saving, with `counted`, `row` and `kind` as the reconciler left them. It is a labelling aid and nothing else (task 7).

**Step 4: The report.** Printed to the terminal, and written to `data/eval/tender/results/latest.json` with no case ids, labels, firm names or amounts:

| Mark | How it is worked out | Pass |
|---|---|---|
| Sum shown wrong with no flag | cases scored `silent` | 0 |
| Sum right | cases scored `right` over cases | at least 90% |
| Dollars in the right row | mean over cases of `RowDollars / KeyDollars`, leaving out cases with no counted key line | at least 80% |
| The same, large tenders only | cases with 100 or more key lines | reported |
| Outside money counted | sum of `OutsideCounted` | 0 |
| Amounts kept by R3 | sum of `AmountsKept` over sum of `AmountsAsked` | reported; under 97% is a finding |
| Amounts kept, scans | the same, cases with `scan` | reported |
| Seconds, up to 100 lines | p50 and p90 of `LatencyMS` | 15 and 30 |
| Seconds, over 100 lines | p50 and p90 | 60 and 90 |
| Cost | tokens at the model's rates | reported |

The command exits 1 on a changed corpus file, a replay miss, or any `silent` case. A mark under its pass line is reported but does not change the exit code: the gate in task 9 decides.

**Step 5:** Run the tests. Expected: PASS.

**Step 6:** Commit: `Tender: record, replay and score the corpus`.

### Task 7: Write the answer key

This is careful clerical work on private documents. Nothing from it is committed.

**Step 1: Choose at least 22 cases** from `testing/rft/` and give each an id that does not name the firm:

- every builder submission (15);
- at least seven consultant proposals, covering: a grid of disciplines by stages; the same fees printed in the firm's table and in a returned form; an option or alternative; a total printed inc GST only; a proposal offering alternatives with no single price; a scan with no text layer; a proposal priced in parts.

Mark six as `"split": "hold"`: two builder schedules of over 100 lines, two smaller builder quotes and two consultant proposals. Do not look at how the pass does on the hold-out cases until task 9.

**Step 2: Seed each case.** `go run ./cmd/tender-eval -live`, then `-dump <case>` for each. This sends the pages to the provider, which the owner has approved for this corpus.

**Step 3: Check each seeded case against the document, page by page.** The seed came from the pass, so it carries the pass's mistakes. For every case:

1. The tender sum: find it printed in the document. If the firm prints its price in parts, add the parts yourself and write down the working in a private note.
2. Every printed subtotal: check that the key lines under it add up to it. A group that adds up to its printed subtotal is almost certainly listed right. A group that does not has a missing or misread line: find it in the document and fix the key.
3. Every key line worth 1% or more of the tender sum: check its amount, whether it is in the price, and its row.
4. Rows: decide each from the schedule row's label and detail, as a project manager would, not from what the pass chose. Where a line truly spans rows, key the row most of the money belongs to.
5. Options, savings and allowances the firm says are extra: `counted: false`.

To read a page, render it with `ocr.Pages` or open the PDF. Do not trust extracted text for a table: the trial showed it scrambles them.

**Step 4: Owner review.** Give the owner the key and the notes. Only the owner sets `"reviewed": true`. Until then every report says the key is unreviewed.

### Task 8: Tune the prompt

Use the hill-climbing method in the claude-api skill (`shared/evals/eval-hillclimb.md`): one change at a time, scored on the tune split, the hold-out untouched.

**Step 1: Baseline.** Run the tune split with prompt `t1`. Record the marks in a private log, `data/eval/tender/private/rounds.md`.

**Step 2: Read the misses** with `-show`. Sort them into: a line missed; a line listed twice; a wrong kind or placement; a wrong row; a wrong group; a page not sent. A page not sent is a page-selection fix (`SelectPages`), not a prompt fix.

**Step 3: One change per round.** Change `prompt.txt` or `Schema`, bump `PromptVersion`, run `-live`, compare. Keep a change only when the marks do not fall. The trial found the wording moves the large schedules a great deal, so always check them.

**Step 4: Try the shorter answer.** The answer is about 108 tokens a line and time is the length of the answer. In one round, try: rows sent to the model as short codes (`r1`, `r2`, ...) and mapped back in Go; and leaving `note` out of lines that need none by telling the model to return an empty string. Keep it if the marks hold and the tokens per line fall below 75.

**Step 5: Stop** when a round changes no mark by more than a point, or after eight rounds, whichever comes first.

**Step 6:** Commit the prompt and schema: `Tender: prompt t<N> after tuning`. The log stays private.

### Task 9: The gate

**Step 1:** Run the hold-out split once: `go run ./cmd/tender-eval -live -split hold`, then the whole corpus in replay.

**Step 2:** Write `docs/evidence/2026-11-14-tender-comparison-gate.md`: the marks table for tune, hold-out and all cases; the same for large tenders alone; seconds and cost; the number of cases and whether the key is reviewed; what the tuning changed. No case ids, firm names or amounts.

**Step 3: Decide.**

- Every pass line met on the hold-out: go. Start slice 2.
- Met except on large tenders: stop. Take to the owner the choice in design section 15: measure Sonnet 5.5 on the large tenders only (set `SITEWISE_TENDER_MODEL`, re-run those cases, report marks, seconds and cost side by side), or ship with the flags as they are.
- A `silent` wrong sum anywhere, or outside money counted: stop. That is a reconciler bug or a missing flag. Fix it with a failing test first, then re-run.
- Amounts kept by R3 under 97% with the product's extractor: stop and report. The design's rule 1 depends on it.

Do not start slice 2 without a go.

### Finish

Write `docs/evidence/2026-11-14-tender-comparison-slice1.md` as the master plan's "Reporting" asks, pointing at the gate note for the figures.
