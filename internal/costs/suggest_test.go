package costs

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestCostSuggestionsScopeAndSuppression(t *testing.T) {
	cat, e := knowledge.Load("../../knowledge")
	if e != nil {
		t.Fatal(e)
	}
	in := SuggestInput{Catalog: cat, Packages: []SuggestPackage{{ID: "b", Title: "Structural", Kind: "services"}, {ID: "a", Title: "Architect", Kind: "services"}, {ID: "main", Title: "Main Works", Kind: "works", WorksScope: "head_contract"}}, Works: []ScopeWork{{ID: "1", SystemID: "electrical.lighting", Title: "Lighting"}, {ID: "2", SystemID: "access-egress.barriers", Title: "Balustrades", PartID: "a"}, {ID: "3", SystemID: "access-egress.barriers", Title: "Balustrades", PartID: "b"}}, Values: map[string]string{}}
	index := func(ss []Suggestion) map[string]Suggestion {
		m := map[string]Suggestion{}
		for _, s := range ss {
			if _, ok := m[s.Key]; ok {
				t.Fatal("duplicate key", s.Key)
			}
			if e := ValidateContent(s.Content); e != nil {
				t.Fatal(s.Key, e)
			}
			m[s.Key] = s
		}
		return m
	}
	all := index(Suggest(in))
	if all["fee:a"].Code != "F01" || all["fee:b"].Code != "F02" {
		t.Fatal(all)
	}
	child := all["works:access-egress.barriers"]
	if child.ParentKey != "works:access-egress" || len(child.ScopeWorkItems) != 2 || all[child.ParentKey].Posting {
		t.Fatal("multipart grouping", all)
	}
	if all["works:electrical"].ResolvedCategory != "Main Works" || all["pw:da-fees"].Key == "" {
		t.Fatal(all)
	}
	in.Dismissed = map[string]bool{"works:access-egress": true, "fee:a": true}
	remaining := index(Suggest(in))
	if remaining[child.Key].Key != "" || remaining["fee:b"].Code != "F02" {
		t.Fatal("dismissal or unstable code", remaining)
	}
	in.Dismissed = nil
	in.Existing = []Item{{ID: "kept", SourceKey: child.ParentKey, Content: all[child.ParentKey].Content}}
	remaining = index(Suggest(in))
	if remaining[child.ParentKey].Key != "" || remaining[child.Key].Key == "" {
		t.Fatal("kept group lost children")
	}
	in.Existing = []Item{{ID: "manual", Content: Content{LineKind: "fee", PackageID: "a"}}, {ID: "manual-works", Content: Content{LineKind: "works", SystemID: "electrical.lighting", Posting: true}}}
	remaining = index(Suggest(in))
	if remaining["fee:a"].Key != "" || remaining["works:electrical"].Key != "" {
		t.Fatal("manual overlap")
	}
	in.Values["hdr.cond.planning"] = "other"
	if index(Suggest(in))["pw:da-fees"].Key != "" {
		t.Fatal("false predicate")
	}
}
func TestOverrideSplitNeverAddsToExistingPostingAllowance(t *testing.T) {
	cat, e := knowledge.Load("../../knowledge")
	if e != nil {
		t.Fatal(e)
	}
	in := SuggestInput{Catalog: cat, Packages: []SuggestPackage{{ID: "join", Title: "Joinery", Kind: "works", DisciplineID: "trade.joinery_kitchens"}}, Works: []ScopeWork{{ID: "j", SystemID: "interiors.joinery"}, {ID: "f", SystemID: "interiors.floor-finishes"}}}
	suggestions := Suggest(in)
	var parent Suggestion
	for _, s := range suggestions {
		if s.Key == "works:interiors" {
			parent = s
		}
	}
	if parent.Key == "" || parent.Posting {
		t.Fatal("override needs group", suggestions)
	}
	var remaining Suggestion
	for _, s := range suggestions {
		if s.Key == "works:interiors:remaining" {
			remaining = s
		}
	}
	in.Existing = []Item{{ID: "group", SourceKey: parent.Key, Content: parent.Content}, {ID: "remaining", SourceKey: remaining.Key, Content: remaining.Content}}
	in.Existing[1].ParentItemID = "group"
	found := false
	for _, s := range Suggest(in) {
		found = found || s.Key == "works:interiors.joinery"
	}
	if !found {
		t.Fatal("keeping remaining scope hid the separate trade")
	}
	parent.Posting = true
	in.Existing = []Item{{ID: "priced", SourceKey: parent.Key, Content: parent.Content, Values: map[string]Value{"budget": val("500.00")}}}
	for _, s := range Suggest(in) {
		if s.LineKind == "works" {
			t.Fatal("split would double count", s)
		}
	}
}
