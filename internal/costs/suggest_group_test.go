package costs

import (
	"sitewise/internal/knowledge"
	"testing"
)

// A combined engagement's fee line is a parent whose children are its
// members' fee lines: prepare suggests the parent first, then each member
// beneath it, and never a second line for a member that already has one.
func TestFeeSuggestionsNestMembersUnderTheirGroup(t *testing.T) {
	cat, e := knowledge.Load("../../knowledge")
	if e != nil {
		t.Fatal(e)
	}
	pkgs := []SuggestPackage{
		{ID: "arch", Title: "Architect", Kind: "services"},
		{ID: "civ", Title: "Civil engineer", Kind: "services", GroupID: "g"},
		{ID: "str", Title: "Structural engineer", Kind: "services", GroupID: "g"},
		{ID: "g", Title: "Structural and civil", Kind: "services", Role: "group"},
	}
	fees := func(ss []Suggestion) []Suggestion {
		out := []Suggestion{}
		for _, s := range ss {
			if s.LineKind == "fee" {
				if e := ValidateContent(s.Content); e != nil {
					t.Fatal(s.Key, e)
				}
				out = append(out, s)
			}
		}
		return out
	}
	got := fees(Suggest(SuggestInput{Catalog: cat, Packages: pkgs, Values: map[string]string{}}))
	keys := []string{}
	for _, s := range got {
		keys = append(keys, s.Key+"<"+s.ParentKey)
	}
	want := []string{"fee:arch<", "fee:g<", "fee:civ<fee:g", "fee:str<fee:g"}
	if len(keys) != len(want) {
		t.Fatalf("fee suggestions %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("fee suggestions %v, want %v", keys, want)
		}
	}
	if got[1].Posting || !got[2].Posting {
		t.Fatalf("the group line holds its members and posts nothing itself: %+v", got[1])
	}
	// The group line and one member's line exist: only the other member is
	// suggested, under the existing group line.
	existing := []Item{
		{ID: "gl", Content: Content{LineKind: "fee", PackageID: "g", Label: "Structural and civil"}, SourceKey: "fee:g"},
		{ID: "sl", Content: Content{LineKind: "fee", PackageID: "str", Label: "Structural engineer", ParentItemID: "gl", Posting: true}, SourceKey: "fee:str"},
	}
	got = fees(Suggest(SuggestInput{Catalog: cat, Packages: pkgs, Existing: existing, Values: map[string]string{}}))
	if len(got) != 2 || got[0].Key != "fee:arch" || got[1].Key != "fee:civ" || got[1].ParentKey != "fee:g" {
		t.Fatalf("with existing lines %+v", got)
	}
}
