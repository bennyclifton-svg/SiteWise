package procurement

import (
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/works"
)

func TestProposeHeadContract(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	item := func(id, action string) works.Item {
		return works.Item{ID: id, Title: id, Action: action, Inclusion: "included", ReviewStatus: "accepted_for_planning"}
	}
	items := []works.Item{item("chiller", "replace"), item("hydrants", "retain"), item("survey", "investigate")}
	lines := func(p *WorksProposal) map[string]bool {
		out := map[string]bool{}
		if p != nil {
			for _, l := range p.Lines {
				out[l.WorkItemID+":"+l.Role] = true
				if l.Text == "" {
					t.Fatalf("line without wording: %+v", l)
				}
			}
		}
		return out
	}
	propose := func(route string, packages []Package, scope []Assignment) *WorksProposal {
		gaps, err := CheckGaps(items, packages, scope, cat)
		if err != nil {
			t.Fatal(err)
		}
		return ProposeHeadContract(gaps, items, packages, route)
	}

	// Unanswered route defaults to design and construct.
	got := lines(propose("", nil, nil))
	for _, want := range []string{"chiller:install", "chiller:design", "hydrants:protect"} {
		if !got[want] {
			t.Fatalf("default route missing %s: %v", want, got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("investigation must stay with the user: %v", got)
	}
	if got := lines(propose("traditional", nil, nil)); got["chiller:design"] || !got["chiller:install"] {
		t.Fatalf("traditional keeps design with consultants: %v", got)
	}

	// An accepted head contract receives only what is still missing.
	head := Package{ID: "hc", Kind: "works", WorksScope: "head_contract"}
	p := propose("design_construct", []Package{head}, []Assignment{{ID: "a", PackageID: "hc", ScopeContent: ScopeContent{WorkItemID: "chiller", Role: "install", Inclusion: "included"}}})
	if p == nil || p.PackageID != "hc" || len(lines(p)) != 2 || lines(p)["chiller:install"] {
		t.Fatalf("additions to accepted head contract: %+v", p)
	}

	// A user who split the works into trades gets no head contract proposal.
	if p := propose("", []Package{{ID: "mech", Kind: "works", WorksScope: "trade"}}, nil); p != nil {
		t.Fatalf("trade split overridden: %+v", p)
	}

	// Nothing missing means nothing proposed.
	full := []Assignment{}
	for _, l := range propose("", nil, nil).Lines {
		full = append(full, Assignment{ID: l.WorkItemID + l.Role, PackageID: "hc", ScopeContent: ScopeContent{WorkItemID: l.WorkItemID, Role: l.Role, Inclusion: "included"}})
	}
	if p := propose("", []Package{head}, full); p != nil {
		t.Fatalf("complete head contract re-proposed: %+v", p)
	}
}

func TestRouteDefaults(t *testing.T) {
	for route, want := range map[string][2]bool{
		"":                    {true, true},
		"design_construct":    {true, false},
		"eci":                 {true, false},
		"traditional":         {false, false},
		"managing_contractor": {true, true},
		"alliance":            {true, true},
		"ppp":                 {true, true},
	} {
		if BuilderDesigns(route) != want[0] || RouteNeedsConfirm(route) != want[1] {
			t.Fatalf("%q: designs=%v confirm=%v", route, BuilderDesigns(route), RouteNeedsConfirm(route))
		}
	}
}
