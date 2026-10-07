package works

import (
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/knowledge"
)

func proposalCatalogue(t testing.TB) *knowledge.Catalog {
	t.Helper()
	cat, err := knowledge.Load(filepath.Join("..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestEveryInterfaceConsequenceDirection(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, record := range cat.InterfaceConsequences() {
		t.Run(record.ID, func(t *testing.T) {
			cat.Interfaces = []knowledge.Interface{{ID: "if.fixture", Type: record.Type, From: []string{"mechanical.air-conditioning"}, To: []string{"structure"}, Status: "draft"}}
			directions := []string{record.Touches}
			if record.Touches == "either" {
				directions = []string{"from", "to"}
			}
			for _, direction := range directions {
				system, target := "mechanical.air-conditioning", "structure"
				if direction == "to" {
					system, target = target, system
				}
				action := "repair"
				if !record.AnyAct {
					action = record.Actions[0]
				}
				items := []Item{{ID: "work", PartID: "part", SystemID: system, Action: action, Inclusion: "included"}}
				proposals, err := InterfaceProposals(cat, items, func(string, string) knowledge.Truth { return knowledge.True })
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, p := range proposals {
					if p.RecordID != record.ID {
						continue
					}
					found = true
					if p.TargetSystemID != target || p.TargetPartID != "part" || p.Kind != record.Propose.Kind || !p.Draft || len(p.Reason.Triggers) != 1 || p.InputsFingerprint == "" {
						t.Fatal(p)
					}
				}
				if !found {
					t.Fatalf("no %s proposal for %s", record.ID, direction)
				}
			}
		})
	}
}

func TestRooftopProposalAndReplacementScope(t *testing.T) {
	cat := proposalCatalogue(t)
	item := Item{ID: "plant", PartID: "roof", SystemID: "mechanical.air-conditioning", Action: "new", Inclusion: "included"}
	for _, test := range []string{"existing", "unknown", "absent", "replaced", "different-part-replaced", "group", "excluded", "retired"} {
		t.Run(test, func(t *testing.T) {
			items := []Item{item}
			state := knowledge.True
			switch test {
			case "unknown":
				state = knowledge.Unknown
			case "absent":
				state = knowledge.False
			case "replaced", "different-part-replaced":
				part := "roof"
				if test == "different-part-replaced" {
					part = "annex"
				}
				items = append(items, Item{ID: "structure", PartID: part, SystemID: "structure", Action: "replace", Inclusion: "included"})
			case "group":
				items[0].IsGroup = true
			case "excluded":
				items[0].Inclusion = "excluded"
			case "retired":
				now := time.Now()
				items[0].RetiredAt = &now
			}
			ps, err := InterfaceProposals(cat, items, func(string, string) knowledge.Truth { return state })
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range ps {
				if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" && p.TargetPartID == "roof" {
					found = true
				}
			}
			want := test == "existing" || test == "unknown" || test == "different-part-replaced"
			if found != want {
				t.Fatalf("found %v want %v", found, want)
			}
		})
	}
}

func TestInterfaceSplitPreservesKeyAndFingerprint(t *testing.T) {
	cat := proposalCatalogue(t)
	item := Item{ID: "original", PartID: "roof", SystemID: "mechanical.air-conditioning", Action: "new", Inclusion: "included"}
	existing := func(string, string) knowledge.Truth { return knowledge.True }
	before, err := InterfaceProposals(cat, []Item{item}, existing)
	if err != nil {
		t.Fatal(err)
	}
	a, b := item, item
	a.ID = "child-a"
	b.ID = "child-b"
	after, err := InterfaceProposals(cat, []Item{b, a}, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 || len(before) != len(after) {
		t.Fatalf("%d %d", len(before), len(after))
	}
	for i, p := range before {
		if p.Key != after[i].Key || p.InputsFingerprint != after[i].InputsFingerprint {
			t.Fatalf("split changed proposal %s", p.Key)
		}
	}
}

func BenchmarkInterfaceProposals(b *testing.B) {
	cat := proposalCatalogue(b)
	items := []Item{{ID: "hydrants", PartID: "C", SystemID: "fire-active.hydrants", Action: "upgrade", Inclusion: "included"}, {ID: "sprinklers-a", PartID: "A", SystemID: "fire-active.sprinklers", Action: "replace", Inclusion: "included"}, {ID: "sprinklers-b", PartID: "B", SystemID: "fire-active.sprinklers", Action: "replace", Inclusion: "included"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := InterfaceProposals(cat, items, func(string, string) knowledge.Truth { return knowledge.Unknown }); err != nil {
			b.Fatal(err)
		}
	}
}
