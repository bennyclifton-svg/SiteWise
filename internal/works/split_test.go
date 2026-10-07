package works_test

import (
	"sitewise/internal/knowledge"
	"sitewise/internal/works"
	"testing"
)

func TestSplitKeepsScopeWithoutDuplicatingQuantity(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	q, u := "12", "m"
	parent := works.Item{ID: "parent", PartID: "part", SystemID: "hydraulic.gas", Action: "repair", Inclusion: "included", Quantity: &q, Unit: &u, ReviewStatus: "verified", Target: works.Target{Text: "Leak free"}}
	children, err := works.SplitChildren(parent, works.SplitRequest{Version: 1, Children: []works.SplitChild{{Title: "One"}, {Title: "Two", Action: "replace"}}}, "actor", cat)
	if err != nil {
		t.Fatal(err)
	}
	if children[0].Quantity != nil || children[0].Unit != nil || children[0].ReviewStatus == "verified" || children[0].Target.Text != "Leak free" || children[1].Action != "replace" {
		t.Fatalf("unsafe defaults %+v", children)
	}
	if _, err = works.SplitChildren(parent, works.SplitRequest{Version: 1, Children: []works.SplitChild{{Title: "One"}}}, "actor", cat); err == nil {
		t.Fatal("single child")
	}
}
