package procurement

import (
	"sitewise/internal/knowledge"
	"sitewise/internal/works"
	"testing"
	"time"
)

func TestGapRules(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	pkgs := []Package{{ID: "builder", Kind: "works"}, {ID: "other", Kind: "works"}, {ID: "engineer", Kind: "services"}, {ID: "owner", Kind: "supply"}}
	a := func(pkg, role string) Assignment {
		return Assignment{ID: pkg + role, PackageID: pkg, ScopeContent: ScopeContent{WorkItemID: "work", Role: role, Inclusion: "included"}}
	}
	for _, tc := range []struct {
		name, action string
		scope        []Assignment
		want         []string
	}{
		{"unassigned new", "new", nil, []string{"design:gap", "install:gap"}},
		{"owner PM", "new", []Assignment{a("builder", "install"), a("engineer", "design")}, nil},
		{"design and construct", "replace", []Assignment{a("builder", "install"), a("builder", "design"), a("builder", "supply")}, nil},
		{"two designers", "upgrade", []Assignment{a("builder", "install"), a("builder", "design"), a("engineer", "design")}, []string{"design:overlap"}},
		{"supply not design or install", "alter", []Assignment{a("owner", "design"), a("owner", "install"), a("owner", "supply")}, []string{"design:gap", "install:gap"}},
		{"repair needs design", "repair", []Assignment{a("builder", "install")}, []string{"design:gap"}},
		{"removal carries out", "remove", []Assignment{a("builder", "install")}, nil},
		{"two installers suppliers", "remove", []Assignment{a("builder", "install"), a("other", "install"), a("builder", "supply"), a("owner", "supply")}, []string{"install:overlap", "supply:overlap"}},
		{"investigate missing", "investigate", nil, []string{"inspect_or_test:gap"}},
		{"inspect and test same party", "investigate", []Assignment{a("engineer", "inspect"), a("engineer", "test")}, nil},
		{"two investigating parties", "investigate", []Assignment{a("engineer", "inspect"), a("builder", "test")}, []string{"inspect_or_test:overlap"}},
		{"retain missing", "retain", nil, []string{"maintain_operation_or_protect:gap"}},
		{"retain multiple allowed", "retain", []Assignment{a("builder", "protect"), a("other", "maintain_operation")}, nil},
		{"services cannot maintain retained work", "retain", []Assignment{a("engineer", "protect")}, []string{"maintain_operation_or_protect:gap"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := works.Item{ID: "work", Action: tc.action, Inclusion: "included", ReviewStatus: "accepted_for_planning"}
			got, err := CheckGaps([]works.Item{item}, pkgs, tc.scope, cat)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %+v want %v", got, tc.want)
			}
			for i, g := range got {
				if g.Role+":"+g.State != tc.want[i] {
					t.Fatalf("got %+v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestGapInheritanceAndEligibility(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	group := works.Item{ID: "group", IsGroup: true, Inclusion: "included", ReviewStatus: "accepted_for_planning", Action: "new"}
	child := works.Item{ID: "child", ParentID: "group", Inclusion: "included", ReviewStatus: "accepted_for_planning", Action: "new"}
	pkgs := []Package{{ID: "builder", Kind: "works"}, {ID: "designer", Kind: "services"}, {ID: "child-designer", Kind: "services"}}
	scope := []Assignment{{ID: "install", PackageID: "builder", ScopeContent: ScopeContent{WorkItemID: "group", Role: "install", Inclusion: "included"}}, {ID: "design", PackageID: "designer", ScopeContent: ScopeContent{WorkItemID: "group", Role: "design", Inclusion: "included"}}, {ID: "child-design", PackageID: "child-designer", ScopeContent: ScopeContent{WorkItemID: "child", Role: "design", Inclusion: "included"}}}
	got, err := CheckGaps([]works.Item{group, child}, pkgs, scope, cat)
	if err != nil || len(got) != 0 {
		t.Fatalf("inheritance %+v %v", got, err)
	}
	child.ReviewStatus = "proposed"
	got, err = CheckGaps([]works.Item{group, child}, pkgs, nil, cat)
	if err != nil || len(got) != 1 || got[0].State != "not_yet_accepted" {
		t.Fatalf("proposed %+v %v", got, err)
	}
	child.Inclusion = "excluded"
	got, err = CheckGaps([]works.Item{group, child}, pkgs, nil, cat)
	if err != nil || len(got) != 0 {
		t.Fatalf("excluded %+v %v", got, err)
	}
	child.Inclusion = "included"
	child.ReviewStatus = "verified"
	now := time.Now()
	pkgs[0].RetiredAt = &now
	got, err = CheckGaps([]works.Item{group, child}, pkgs, scope, cat)
	if err != nil || len(got) != 1 || got[0].Role != "install" {
		t.Fatalf("retired package %+v %v", got, err)
	}
	child.RetiredAt = &now
	got, err = CheckGaps([]works.Item{group, child}, pkgs, nil, cat)
	if err != nil || len(got) != 0 {
		t.Fatalf("retired work %+v %v", got, err)
	}
}
