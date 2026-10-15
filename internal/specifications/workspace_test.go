package specifications

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestFrameworkFollowsProfilerHierarchyAndExplicitScope(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	sections := Sections(cat, Snapshot{})
	order := cat.BuildingOrderSystems()
	if len(sections) != len(order) {
		t.Fatal("different hierarchy")
	}
	count := 0
	for i, s := range sections {
		if s.ID != order[i].ID {
			t.Fatal("different order")
		}
		count += len(s.Starters)
	}
	if count < 30 {
		t.Fatal("empty framework")
	}
	for _, mode := range []string{"included", "excluded"} {
		sections = Sections(cat, Snapshot{Scope: []Scope{{SystemID: "interiors.tiling", Inclusion: mode}}})
		count = 0
		for _, s := range sections {
			for _, starter := range s.Starters {
				count++
				if starter.SystemID != "interiors.tiling" {
					t.Fatal("unscoped prompt", starter)
				}
			}
		}
		if mode == "included" && count != 2 || mode == "excluded" && count != 0 {
			t.Fatal(mode, count)
		}
	}
}

func TestItemResponsibilityOverridesSectionButDoesNotAdoptSource(t *testing.T) {
	item := Type{ID: "a", SystemID: "interiors.tiling", Authority: "sitewise"}
	snap := Snapshot{Coverage: []Coverage{{SystemID: "interiors", Mode: "external"}}}
	if !External(snap, item) {
		t.Fatal("section authority ignored")
	}
	snap.Coverage = append(snap.Coverage, Coverage{SystemID: item.SystemID, TypeID: item.ID, Mode: "managed_here"})
	if External(snap, item) {
		t.Fatal("item exception ignored")
	}
	item.Authority = "external"
	if !External(snap, item) {
		t.Fatal("coverage silently adopted a source")
	}
}

func TestExternalCoverageDoesNotIssueOldLocalDecisionsAsConsultantReadings(t *testing.T) {
	snap := Snapshot{Types: []Type{{ID: "tile", SystemID: "interiors.tiling", Status: "active", Authority: "sitewise", Origin: "user", Description: "Local description"}}, Coverage: []Coverage{{SystemID: "interiors", Mode: "external"}}, Assertions: []Assertion{{TypeID: "tile", Key: "colour", Meaning: "requirement", Status: "accepted", Origin: "user", Value: Value{State: "known", Text: "Blue"}}}}
	rows := Assemble(snap)
	if len(rows) != 1 || len(rows[0].Fields) != 0 || rows[0].Pending != 1 || rows[0].Description != "" {
		t.Fatal("old local decision issued as external requirement", rows)
	}
}
