package costs

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestCostPackageAmbiguityAndInheritance(t *testing.T) {
	packages := []SuggestPackage{{ID: "a", Title: "Head contractor", Kind: "works", WorksScope: "head_contract"}}
	line := Item{ID: "child", Content: Content{LineKind: "works", SystemID: "electrical", Posting: true}}
	id, label := ResolveCostPackage(nil, line, nil, packages)
	if id != "a" || label != "Head contractor" {
		t.Fatal(id, label)
	}
	packages = append(packages, SuggestPackage{ID: "b", Title: "D&C contractor", Kind: "works", WorksScope: "head_contract"})
	if id, label = ResolveCostPackage(nil, line, nil, packages); id != "" || label != "Unassigned" {
		t.Fatal("ambiguous match selected or hidden", id, label)
	}
	parent := Item{ID: "parent", Content: Content{LineKind: "works", PackageID: "b"}}
	line.ParentItemID = parent.ID
	if id, _ = ResolveCostPackage(nil, line, []Item{parent, line}, packages); id != "b" {
		t.Fatal("inheritance lost", id)
	}
	line.PackageID = "a"
	packages[0].Retired = true
	if id, label = ResolveCostPackage(nil, line, []Item{parent, line}, packages); id != "a" || label != "Head contractor (retired)" {
		t.Fatal("explicit history lost", id, label)
	}
	amount := Money("100.01")
	line.ResolvedPackageID = id
	line.Values = map[string]Value{"commitment": {Amount: &amount, ValueState: "known"}}
	totals, e := Summarize(Plan{Items: []Item{parent, line}}, "package")
	if e != nil || *totals.Groups["a"]["forecast"].Amount != amount {
		t.Fatal(totals, e)
	}
}

func TestTradeOverridesHeadAndAmbiguityRemainsUnassigned(t *testing.T) {
	cat, e := knowledge.Load("../../knowledge")
	if e != nil {
		t.Fatal(e)
	}
	row := Item{Content: Content{LineKind: "works", SystemID: "electrical"}}
	packages := []SuggestPackage{{ID: "head", Kind: "works", WorksScope: "head_contract", Title: "D&C"}, {ID: "trade", Kind: "works", DisciplineID: "trade.electrical", Title: "Electrician"}}
	if id, _ := ResolveCostPackage(cat, row, nil, packages); id != "trade" {
		t.Fatal(id)
	}
	packages = append(packages, SuggestPackage{ID: "other", Kind: "works", DisciplineID: "trade.electrical", Title: "Another electrician"})
	if id, label := ResolveCostPackage(cat, row, nil, packages); id != "" || label != "Unassigned" {
		t.Fatal("ambiguous trades picked a package or were hidden", id, label)
	}
}

func TestCostDefaultHeadContractorDoesNotInventAssignment(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	row := Item{Content: Content{LineKind: "works", SystemID: "electrical", Posting: true}}
	for _, packages := range [][]SuggestPackage{
		nil,
		{{ID: "retired", Kind: "works", WorksScope: "head_contract", Retired: true}},
		{{ID: "consultant", Kind: "services", Title: "Architect"}},
	} {
		id, label := ResolveCostPackage(cat, row, nil, packages)
		if id != "" || label != "Head contractor" || mappedPackage(cat, row.SystemID, packages) != label {
			t.Fatal("missing planning default or invented assignment", id, label)
		}
	}
	row.ResolvedPackageID, row.ResolvedCategory = ResolveCostPackage(cat, row, nil, nil)
	amount := Money("100.01")
	row.Values = map[string]Value{"commitment": {Amount: &amount, ValueState: "known"}}
	totals, err := Summarize(Plan{Items: []Item{row}}, "package")
	if err != nil || len(totals.Groups) != 1 || totals.Groups["unallocated"]["commitment"].Amount == nil || *totals.Groups["unallocated"]["commitment"].Amount != amount {
		t.Fatal("planning role changed package totals", totals, err)
	}
	row.PackageID = "missing"
	if id, label := ResolveCostPackage(cat, row, nil, nil); id != "" || label != "Unassigned" {
		t.Fatal("missing explicit assignment hidden", id, label)
	}
	row.PackageID, row.LineKind = "", "fee"
	if _, label := ResolveCostPackage(cat, row, nil, nil); label != "Unassigned" {
		t.Fatal("construction default applied to consultants", label)
	}
}
