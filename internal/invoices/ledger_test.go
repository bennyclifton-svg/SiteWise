package invoices

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func i64(n int64) *int64 { return &n }

func readInvoice(id string, ex, gst int64) Invoice {
	return Invoice{ID: id, Status: StatusFiled, Kind: "invoice", Currency: "AUD", Number: "INV-" + id, Date: "2026-10-01", Period: "2026-10-01",
		PackageID: "pkg", Ex: i64(ex), GST: i64(gst), Total: i64(ex + gst), Allocations: []Allocation{{ID: "a" + id, LineID: "L1"}}}
}

func flagged(flags map[string][]Flag, id, code string) bool {
	for _, f := range flags[id] {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestChecksPassAndFail(t *testing.T) {
	today := "2026-10-16"
	cases := []struct {
		name  string
		code  string
		pass  []Invoice
		fail  []Invoice
		abns  map[string]string
		check string // invoice id the flag lands on
	}{
		{name: "duplicate number", code: "duplicate",
			pass:  []Invoice{readInvoice("1", 1000, 100), func() Invoice { i := readInvoice("2", 2000, 200); return i }()},
			fail:  []Invoice{readInvoice("1", 1000, 100), func() Invoice { i := readInvoice("2", 2000, 200); i.Number = "inv-1"; return i }()},
			check: "2"},
		{name: "look-alike", code: "look_alike",
			pass:  []Invoice{readInvoice("1", 1000, 100), func() Invoice { i := readInvoice("2", 1000, 100); i.Date = "2026-09-01"; return i }()},
			fail:  []Invoice{readInvoice("1", 1000, 100), readInvoice("2", 1000, 100)},
			check: "2"},
		{name: "arithmetic", code: "arithmetic",
			pass:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Total = i64(1101); return i }()},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Total = i64(1110); return i }()},
			check: "1"},
		{name: "gst rate", code: "gst_rate",
			pass:  []Invoice{readInvoice("1", 265650, 26565)},
			fail:  []Invoice{readInvoice("1", 739151, 72000)},
			check: "1"},
		{name: "abn invalid", code: "abn_invalid",
			pass:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.ABN = "54165354218"; return i }()},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.ABN = "54165354219"; return i }()},
			check: "1"},
		{name: "abn differs from supplier", code: "abn_differs", abns: map[string]string{"pkg": "54165354218"},
			pass:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.ABN = "54165354218"; return i }()},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.ABN = "81139719529"; return i }()},
			check: "1"},
		{name: "split total", code: "split_total",
			pass: []Invoice{func() Invoice {
				i := readInvoice("1", 1000, 100)
				i.Allocations = []Allocation{{LineID: "L1", Amount: i64(600)}, {LineID: "L2", Amount: i64(400)}}
				return i
			}()},
			fail: []Invoice{func() Invoice {
				i := readInvoice("1", 1000, 100)
				i.Allocations = []Allocation{{LineID: "L1", Amount: i64(600)}, {LineID: "L2", Amount: i64(300)}}
				return i
			}()},
			check: "1"},
		{name: "period mismatch", code: "period",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Date = "2026-09-28"; return i }()},
			check: "1"},
		{name: "odd date", code: "odd_date",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Date = "2026-12-01"; return i }()},
			check: "1"},
		{name: "unmapped", code: "unmapped",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Allocations[0].LineID = ""; return i }()},
			check: "1"},
		{name: "foreign currency", code: "currency",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Currency = "USD"; return i }()},
			check: "1"},
		{name: "not an invoice", code: "not_invoice",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.Kind = "not_invoice"; return i }()},
			check: "1"},
		{name: "no matching package", code: "no_supplier",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.PackageID = ""; i.ABN = "54165354218"; return i }()},
			check: "1"},
		{name: "several invoices in one file", code: "many",
			pass:  []Invoice{readInvoice("1", 1000, 100)},
			fail:  []Invoice{func() Invoice { i := readInvoice("1", 1000, 100); i.InvoiceCount = 3; return i }()},
			check: "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Check(c.pass, c.abns, today); flagged(got, c.check, c.code) {
				t.Fatalf("passing ledger flagged %s: %v", c.code, got[c.check])
			}
			if got := Check(c.fail, c.abns, today); !flagged(got, c.check, c.code) {
				t.Fatalf("failing ledger not flagged %s: %v", c.code, got[c.check])
			}
		})
	}
}

func TestDuplicateKeepBothAndVoidClearTheFlag(t *testing.T) {
	a, b := readInvoice("1", 1000, 100), readInvoice("2", 2000, 200)
	b.Number = a.Number
	if !flagged(Check([]Invoice{a, b}, nil, ""), "2", "duplicate") {
		t.Fatal("duplicate not flagged")
	}
	b.DuplicateOK = true
	if flagged(Check([]Invoice{a, b}, nil, ""), "2", "duplicate") {
		t.Fatal("keep both still flagged")
	}
	b.DuplicateOK = false
	a.Status = StatusVoid
	if flagged(Check([]Invoice{a, b}, nil, ""), "2", "duplicate") {
		t.Fatal("a void invoice still counts as a duplicate")
	}
}

func TestStatusNeedsReviewForWeakFieldsAndFlags(t *testing.T) {
	inv := readInvoice("1", 1000, 100)
	inv.Decisions = map[string]Decision{}
	for _, f := range required {
		inv.Decisions[f] = Decision{Field: f, Value: "x", Band: BandGreen, DecidedBy: DecidedByRule}
	}
	if s := Status(inv, nil); s != StatusFiled {
		t.Fatalf("all green = %s", s)
	}
	inv.Decisions[FieldSupplier] = Decision{Field: FieldSupplier, Value: "pkg", Band: BandAmber, DecidedBy: DecidedByJev}
	if s := Status(inv, nil); s != StatusNeedsReview {
		t.Fatalf("amber supplier = %s", s)
	}
	inv.Decisions[FieldSupplier] = Decision{Field: FieldSupplier, Value: "pkg", Band: BandAmber, DecidedBy: DecidedByUser}
	if s := Status(inv, []Flag{{Code: "gst_free", Info: true}}); s != StatusFiled {
		t.Fatalf("user value with an info flag = %s", s)
	}
	if s := Status(inv, []Flag{{Code: "duplicate"}}); s != StatusNeedsReview {
		t.Fatalf("duplicate flag = %s", s)
	}
}

func TestSummarizeLinesBudgetsAndBasis(t *testing.T) {
	lines := []BudgetLine{{ID: "L1", Label: "Architect", Budget: i64(1000000)}, {ID: "L2", Label: "Structure", Budget: i64(500000)}, {ID: "L3", Label: "Removed", Archived: true}}
	ledger := []Invoice{readInvoice("1", 400000, 40000), readInvoice("2", 700000, 70000)}
	ledger[1].Allocations = []Allocation{{LineID: "L2", Amount: i64(600000)}, {LineID: "L3", Amount: i64(100000)}}
	credit := readInvoice("3", -50000, -5000)
	credit.Kind = "credit_note"
	unmapped := readInvoice("4", 10000, 1000)
	unmapped.Allocations[0].LineID = ""
	excluded := readInvoice("5", 99999, 0)
	excluded.Status = StatusVoid
	ledger = append(ledger, credit, unmapped, excluded)
	s := Summarize(lines, ledger, "ex_tax")
	if s.Total != 400000+700000-50000+10000 {
		t.Fatalf("total %d", s.Total)
	}
	if s.Lines[0].Invoiced != 350000 || s.Lines[1].Invoiced != 600000 || s.Lines[2].Invoiced != 100000 || s.Unallocated.Invoiced != 10000 {
		t.Fatalf("lines %+v unallocated %d", s.Lines, s.Unallocated.Invoiced)
	}
	if !s.Lines[1].OverBudget || s.Lines[0].OverBudget || s.Lines[1].Percent != "120.0" || s.Lines[0].Percent != "35.0" {
		t.Fatalf("budget figures %+v", s.Lines[:2])
	}
	if s.Remaining == nil || *s.Remaining != 1500000-s.Total || s.Excluded != 1 {
		t.Fatalf("overall %+v", s)
	}
	inc := Summarize(lines, ledger, "inc_tax")
	if inc.Remaining != nil || inc.Lines[0].Remaining != nil || inc.Lines[0].Percent != "" {
		t.Fatal("an including-tax plan must not mix bases")
	}
}

// The ledger total always equals the lines plus Unallocated, after any run
// of adds, re-maps, splits, voids and credit notes.
func TestLedgerInvariantUnderRandomEdits(t *testing.T) {
	rng := rand.New(rand.NewSource(20261016))
	lines := []BudgetLine{{ID: "A", Budget: i64(5000000)}, {ID: "B"}, {ID: "C", Archived: true}}
	ids := []string{"A", "B", "C", "", "gone"}
	for run := 0; run < 200; run++ {
		var ledger []Invoice
		for step := 0; step < 60; step++ {
			switch op := rng.Intn(6); {
			case op == 0 || len(ledger) == 0:
				ex := int64(rng.Intn(2000000))
				inv := readInvoice(strconv.Itoa(len(ledger)), ex, GSTOf(ex))
				inv.Allocations[0].LineID = ids[rng.Intn(len(ids))]
				ledger = append(ledger, inv)
			case op == 1:
				i := rng.Intn(len(ledger))
				ledger[i].Allocations = []Allocation{{LineID: ids[rng.Intn(len(ids))]}}
			case op == 2:
				i := rng.Intn(len(ledger))
				if ledger[i].Ex != nil {
					part := *ledger[i].Ex / 3
					ledger[i].Allocations = []Allocation{{LineID: ids[rng.Intn(len(ids))], Amount: i64(part)}, {LineID: ids[rng.Intn(len(ids))], Amount: i64(*ledger[i].Ex - part - int64(rng.Intn(3)))}}
				}
			case op == 3:
				ledger[rng.Intn(len(ledger))].Status = StatusVoid
			case op == 4:
				ex := -int64(rng.Intn(500000))
				inv := readInvoice(strconv.Itoa(len(ledger)), ex, GSTOf(ex))
				inv.Kind = "credit_note"
				ledger = append(ledger, inv)
			case op == 5:
				ledger[rng.Intn(len(ledger))].Ex = nil
			}
			s := Summarize(lines, ledger, "ex_tax")
			var want, held int64
			for _, inv := range ledger {
				if Counted(inv) {
					want += *inv.Ex
				}
			}
			for _, l := range s.Lines {
				held += l.Invoiced
			}
			if s.Total != want || held+s.Unallocated.Invoiced != want {
				t.Fatalf("run %d step %d: total %d want %d, lines %d + unallocated %d", run, step, s.Total, want, held, s.Unallocated.Invoiced)
			}
		}
	}
}

func TestPeriodIsTheSydneyMonth(t *testing.T) {
	// 31 Oct 14:30 UTC is 1 Nov 01:30 in Sydney (daylight saving).
	if got := PeriodOf(time.Date(2026, 10, 31, 14, 30, 0, 0, time.UTC)); got != "2026-11-01" {
		t.Fatal(got)
	}
	if got := PeriodOf(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)); got != "2026-10-01" {
		t.Fatal(got)
	}
}
