package invoices

import (
	"sitewise/internal/costs"
	"testing"
)

func TestGroupSpendCountsSplitInvoiceOnceAndRetainsCredits(t *testing.T) {
	b := costs.Money("1000.00")
	items := []costs.Item{{ID: "g", Values: map[string]costs.Value{"budget": {ValueState: "known", Amount: &b}}}, {ID: "nested", ParentItemID: "g"}, {ID: "a", ParentItemID: "g", Posting: true}, {ID: "b", ParentItemID: "nested", Posting: true}}
	inv := readInvoice("split", 30000, 3000)
	inv.Allocations = []Allocation{{LineID: "a", Amount: i64(10000)}, {LineID: "b", Amount: i64(20000)}}
	credit := readInvoice("credit", -5000, -500)
	credit.Allocations = []Allocation{{LineID: "b"}}
	void := inv
	void.Status = StatusVoid
	ledger := []Invoice{inv, credit, void}
	g := GroupSummaries(items, ledger, "ex_tax")
	if g["g"].Invoiced != 25000 || g["g"].Invoices != 2 || *g["g"].Remaining != 75000 || g["g"].Percent != "25.0" || g["nested"].Invoiced != 15000 {
		t.Fatal(g)
	}
	if GroupSummaries(items, ledger, "inc_tax")["g"].Remaining != nil {
		t.Fatal("mixed tax bases")
	}
	f := Filter{Line: "g", LineIDs: g["g"].LineIDs}
	if !f.Match(inv, Row{LineID: "b"}, StatusFiled) || f.Match(inv, Row{LineID: "elsewhere"}, StatusFiled) {
		t.Fatal("group filter lost scope")
	}
}
