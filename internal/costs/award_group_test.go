package costs

import "testing"

func known(s string) Value {
	return Value{ValueState: "known", Amount: groupMoney(s), Origin: "user", Meaning: "stated"}
}

// An awarded package's line is a group that keeps the PM's budget while its
// children hold the contract: the budget counts once, the forecast is the
// children's, and the group's variance is budget less contract.
func TestAwardedGroupKeepsItsBudget(t *testing.T) {
	plan := Plan{Items: []Item{
		{ID: "g", Content: Content{LineKind: "fee", PackageID: "p", Label: "Town planning: Mecone"}, Values: map[string]Value{"budget": known("16000.00")}},
		{ID: "c1", Content: Content{ParentItemID: "g", LineKind: "fee", PackageID: "p", Posting: true, Label: "Stage 1"}, Values: map[string]Value{"commitment": known("3000.00")}},
		{ID: "c2", Content: Content{ParentItemID: "g", LineKind: "fee", PackageID: "p", Posting: true, Label: "Stages 2 to 4"}, Values: map[string]Value{"commitment": known("12500.00")}},
		{ID: "other", Content: Content{LineKind: "fee", PackageID: "q", Posting: true, Label: "Survey"}, Values: map[string]Value{"budget": known("5000.00")}},
	}}
	tot, err := Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if b := tot.Overall["budget"]; b.Amount == nil || *b.Amount != "21000.00" || !b.Complete {
		t.Fatalf("budget %+v", b)
	}
	if f := tot.Overall["forecast"]; f.Amount == nil || *f.Amount != "20500.00" {
		t.Fatalf("forecast %+v", f)
	}
	g := tot.Items["g"]
	if g.Forecast.Amount == nil || *g.Forecast.Amount != "15500.00" || g.BudgetVariance == nil || *g.BudgetVariance != "500.00" {
		t.Fatalf("group figures %+v", g)
	}
	if tot.Items["c1"].BudgetVariance != nil {
		t.Fatalf("a child has no budget, so no variance: %+v", tot.Items["c1"])
	}
	// No budget set on the parent: the contract shows, the variance does not.
	delete(plan.Items[0].Values, "budget")
	plan.Items[0].Values["budget"] = Value{ValueState: "unknown", Origin: "user", Meaning: "allowance"}
	tot, _ = Summarize(plan, "overall")
	if g := tot.Items["g"]; g.Forecast.Amount == nil || g.BudgetVariance != nil {
		t.Fatalf("unknown budget: %+v", g)
	}
	if tot.Overall["budget"].Complete {
		t.Fatal("an unknown group budget leaves the total incomplete")
	}
}
func groupMoney(s string) *Money { m := Money(s); return &m }
