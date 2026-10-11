package costs

import "testing"

func TestSectionsKeepTreeAndExactUnknownTotals(t *testing.T) {
	money := Money("9007199254740993.01")
	items := []Item{
		{ID: "authority", Content: Content{LineKind: "project_wide", Category: "authority_fees", Posting: true}, Values: map[string]Value{"budget": {Amount: &money, ValueState: "known"}}},
		{ID: "consultant", Content: Content{LineKind: "fee", Posting: true}},
		{ID: "group", Content: Content{LineKind: "works"}},
		{ID: "child", Content: Content{LineKind: "project_wide", Category: "other", ParentItemID: "group", Posting: true}, Values: map[string]Value{"budget": {Amount: &money, ValueState: "known"}}},
		{ID: "other", Content: Content{LineKind: "project_wide", Category: "other", Posting: true}},
		{ID: "excluded", Content: Content{LineKind: "project_wide", Category: "exclusions", Posting: true, Excluded: true}, Values: map[string]Value{"budget": {Amount: &money, ValueState: "known"}}},
	}
	totals, e := Summarize(Plan{Items: items}, "overall")
	if e != nil {
		t.Fatal(e)
	}
	if totals.SectionItems["child"] != "construction" || totals.SectionItems["other"] != "contingency" {
		t.Fatal(totals.SectionItems)
	}
	if *totals.Sections["authority"]["budget"].Amount != money || *totals.Sections["construction"]["budget"].Amount != money {
		t.Fatal("precision or tree total lost", totals)
	}
	if totals.Overall["budget"].KnownSubtotal != "18014398509481986.02" || totals.Overall["budget"].Unknown != 2 || totals.Overall["budget"].Amount != nil {
		t.Fatal("unknown/exclusion accounting changed", totals.Overall)
	}
	if totals.Sections["contingency"]["budget"].Lines != 1 || totals.Sections["consultants"]["budget"].Unknown != 1 {
		t.Fatal(totals.Sections)
	}
}
