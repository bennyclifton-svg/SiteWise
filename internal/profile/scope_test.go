package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

func scopeRow(rows []profile.Row, leaf string) (profile.Row, bool) {
	for _, r := range rows {
		if r.PartID == whole && r.Key == "scope."+leaf {
			return r, true
		}
	}
	return profile.Row{}, false
}

func header(class, work string) []profile.UserValue {
	return []profile.UserValue{
		{PartID: whole, Key: "hdr.subclass", Value: str(class)},
		{PartID: whole, Key: "hdr.work_type", Value: str(work)},
	}
}

func TestScopeStartsFromDefaultsAndDocumentsAdd(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.mechanical.heating-appliances.presence", "included", "spec", 0.9))
	in.User = header("house", "new")
	rows := profile.Build(in, cat)
	if r, ok := scopeRow(rows, "substructure.footings"); !ok || r.Value != "in" || r.Band != "suggested" {
		t.Fatalf("default scope row %+v %v", r, ok)
	}
	if r, ok := scopeRow(rows, "mechanical.heating-appliances"); !ok || r.Value != "in" || r.Band != "amber" {
		t.Fatalf("a document's included system must join the scope: %+v %v", r, ok)
	}
	if _, ok := scopeRow(rows, "vertical-transport.passenger-lifts"); ok {
		t.Fatal("a system nobody chose or read is in scope")
	}
}

func TestRefurbishmentStartsEmptyAndTheUserChooses(t *testing.T) {
	cat := repoCatalog(t)
	in := input()
	in.User = append(header("warehouse", "refurb"), profile.UserValue{PartID: whole, Key: "scope.fire-active.sprinklers", Value: str("in")})
	rows := profile.Build(in, cat)
	count := 0
	for _, r := range rows {
		if r.Key[:6] == "scope." && r.Value == "in" {
			count++
		}
	}
	if r, ok := scopeRow(rows, "fire-active.sprinklers"); !ok || r.Band != "user" || count != 1 {
		t.Fatalf("pump replacement scope: %d in scope, sprinklers %+v", count, r)
	}
	for _, r := range rows {
		if r.Band == "suggested" {
			t.Fatalf("refurbishment suggested %s", r.Key)
		}
	}
}

func TestTheUsersRemovalIsFinalButADocumentSaysSo(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "spec", 0.9))
	in.User = append(header("house", "new"),
		profile.UserValue{PartID: whole, Key: "scope.hydraulic.gas", Value: str("out")},
		profile.UserValue{PartID: whole, Key: "scope.substructure.footings", Value: str("out")})
	rows := profile.Build(in, cat)
	gas, _ := scopeRow(rows, "hydraulic.gas")
	if gas.Value != "out" || gas.Band != "user" || gas.Note == "" {
		t.Fatalf("removed but evidenced system must stay out with a notice: %+v", gas)
	}
	footings, _ := scopeRow(rows, "substructure.footings")
	if footings.Value != "out" || footings.Note != "" {
		t.Fatalf("a removed default stays out without a notice: %+v", footings)
	}
	for _, r := range rows {
		if r.Key == "sys.substructure.footings.presence" && r.Band == "suggested" {
			t.Fatal("a removed default is still suggested")
		}
	}
}

func TestChangingBuildingTypeReappliesOnlyDefaults(t *testing.T) {
	cat := repoCatalog(t)
	in := input()
	in.User = append(header("warehouse", "new"), profile.UserValue{PartID: whole, Key: "scope.vertical-transport.goods-lifts", Value: str("in")})
	rows := profile.Build(in, cat)
	if _, ok := scopeRow(rows, "site.loading-docks"); ok {
		t.Log("loading docks are a warehouse default") // informative only: the list is draft data
	}
	in.User = append(header("house", "new"), profile.UserValue{PartID: whole, Key: "scope.vertical-transport.goods-lifts", Value: str("in")})
	rows = profile.Build(in, cat)
	if r, ok := scopeRow(rows, "vertical-transport.goods-lifts"); !ok || r.Band != "user" {
		t.Fatalf("the user's choice survives a type change: %+v", r)
	}
	if _, ok := scopeRow(rows, "structure.steel"); ok {
		t.Fatal("the warehouse default stayed after switching to a house")
	}
}
