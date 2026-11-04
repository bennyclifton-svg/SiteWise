package costs

import "testing"

func TestGroupRollupsKeepContractSeparateFromForecast(t *testing.T) {
	items := []Item{
		{ID: "architect", LineKind: "fee", Values: map[string]Value{"budget": val("100000.00")}},
		{ID: "concept", ParentItemID: "architect", LineKind: "fee", Posting: true, Values: map[string]Value{"commitment": val("14500.00")}},
		{ID: "stages", ParentItemID: "architect", LineKind: "fee"},
		{ID: "planning", ParentItemID: "stages", LineKind: "fee", Posting: true, Values: map[string]Value{"commitment": val("24000.00")}},
		{ID: "excluded", ParentItemID: "architect", LineKind: "fee", Posting: true, Excluded: true, Values: map[string]Value{"commitment": val("999.00")}},
	}
	r := GroupRollups(items)
	if *r["architect"].Values["commitment"].Amount != "38500.00" || *r["architect"].Values["budget"].Amount != "100000.00" || *r["architect"].BudgetVariance != "61500.00" || *r["stages"].Values["commitment"].Amount != "24000.00" {
		t.Fatal(r)
	}
	totals, err := Summarize(Plan{Items: items}, "overall")
	if err != nil || *totals.Overall["commitment"].Amount != "38500.00" || *totals.Overall["budget"].Amount != "100000.00" {
		t.Fatal(totals, err)
	}
	// A budget-only activity contributes to forecast, never approved contract.
	items[1].Values = map[string]Value{"budget": val("15000.00")}
	r = GroupRollups(items)
	if r["architect"].Values["commitment"].Amount != nil || r["architect"].Values["commitment"].Unknown != 1 || *r["architect"].Values["forecast"].Amount != "39000.00" {
		t.Fatal(r)
	}
	// Ranges and unknowns survive disclosure; neither becomes a fabricated zero.
	items[1].Values = map[string]Value{"commitment": {ValueState: "known", Low: money("10000.00"), High: money("20000.00")}}
	r = GroupRollups(items)
	if r["architect"].Values["commitment"].Amount != nil || *r["architect"].Values["commitment"].Low != "34000.00" || *r["architect"].Values["commitment"].High != "44000.00" || r["architect"].BudgetVariance != nil {
		t.Fatal(r)
	}
}

func TestSubdivisionRollupAndBrokenAncestry(t *testing.T) {
	items := []Item{{ID: "g"}, {ID: "a", ParentItemID: "g", Posting: true, Values: map[string]Value{"budget": val("300.00")}}, {ID: "b", ParentItemID: "g", Posting: true, Values: map[string]Value{"budget": val("700.00")}}}
	r := GroupRollups(items)["g"]
	if *r.Values["budget"].Amount != "1000.00" || r.Values["commitment"].Amount != nil || r.Values["commitment"].Unknown != 2 {
		t.Fatal(r)
	}
	items = append(items, Item{ID: "orphan", ParentItemID: "missing"}, Item{ID: "cycle", ParentItemID: "cycle"})
	r = GroupRollups(items)["cycle"]
	if r.Values["forecast"].Amount != nil {
		t.Fatal(r)
	}
}
