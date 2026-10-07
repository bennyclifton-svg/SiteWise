package procurement

import (
	"testing"

	"sitewise/internal/knowledge"
)

func TestScopeBoundaries(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	base := ScopeContent{ItemKind: "responsibility", WorkItemID: "work", Role: "design", UserText: "Design the replacement system", Inclusion: "included"}
	for _, kind := range []string{"services", "works"} {
		if err := ValidateScope(base, Package{Kind: kind}, cat); err != nil {
			t.Fatal(err)
		}
	}
	if ValidateScope(base, Package{Kind: "supply"}, cat) == nil {
		t.Fatal("supply-only package accepted design responsibility")
	}
	for name, change := range map[string]func(*ScopeContent){
		"missing work":        func(s *ScopeContent) { s.WorkItemID = "" },
		"missing role":        func(s *ScopeContent) { s.Role = "" },
		"invalid role":        func(s *ScopeContent) { s.Role = "approve" },
		"invalid kind":        func(s *ScopeContent) { s.ItemKind = "other" },
		"invalid inclusion":   func(s *ScopeContent) { s.Inclusion = "maybe" },
		"blank text":          func(s *ScopeContent) { s.UserText = " " },
		"two wordings":        func(s *ScopeContent) { s.ClauseID = "cl.rfp-draft-status"; s.ClauseVersion = 1 },
		"orphan version":      func(s *ScopeContent) { s.ClauseVersion = 1 },
		"unknown interface":   func(s *ScopeContent) { s.InterfaceIDs = []string{"missing"} },
		"duplicate interface": func(s *ScopeContent) { s.InterfaceIDs = []string{cat.Interfaces[0].ID, cat.Interfaces[0].ID} },
	} {
		t.Run(name, func(t *testing.T) {
			s := base
			change(&s)
			if ValidateScope(s, Package{Kind: "works"}, cat) == nil {
				t.Fatal("accepted invalid scope")
			}
		})
	}
	obligation := ScopeContent{ItemKind: "obligation", UserText: "Attend coordination meetings", Inclusion: "included"}
	if err := ValidateScope(obligation, Package{Kind: "services"}, cat); err != nil {
		t.Fatal(err)
	}
	base.Role = "supply"
	if err := ValidateScope(base, Package{Kind: "supply"}, cat); err != nil {
		t.Fatal(err)
	}
	base.Role = "maintain_operation"
	base.ItemKind = "obligation"
	if err := ValidateScope(base, Package{Kind: "works"}, cat); err != nil {
		t.Fatal(err)
	}
}

func TestScopeDraftClauseRemainsProvisional(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	s := ScopeContent{ItemKind: "obligation", ClauseID: "cl.rfp-draft-status", ClauseVersion: 1, Inclusion: "included"}
	if err := ValidateScope(s, Package{Kind: "services"}, cat); err != nil {
		t.Fatal(err)
	}
	if !ClauseProvisional(s, cat) {
		t.Fatal("draft clause treated as approved")
	}
	s.ClauseVersion++
	if ValidateScope(s, Package{Kind: "services"}, cat) == nil || !ClauseProvisional(s, cat) {
		t.Fatal("unknown clause version accepted")
	}
}
