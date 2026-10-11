package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCostStructureRejectsInvalidCatalogue(t *testing.T) {
	for name, edit := range map[string]func(string) string{
		"unknown trade":  func(s string) string { return strings.Replace(s, "trade.main_works", "trade.no_such_trade", 1) },
		"wrong parent":   func(s string) string { return strings.Replace(s, "sub: structure.steel", "sub: interiors.joinery", 1) },
		"duplicate code": func(s string) string { return strings.Replace(s, `code: "02"`, `code: "01"`, 1) },
		"missing system": func(s string) string { return strings.Replace(s, "system: site,", "system: missing,", 1) },
	} {
		t.Run(name, func(t *testing.T) {
			root := copyKnowledge(t)
			path := filepath.Join(root, "costs", "elements.yaml")
			raw, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			edited := edit(string(raw))
			if edited == string(raw) {
				t.Fatal("fixture unchanged")
			}
			if e = os.WriteFile(path, []byte(edited), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = Load(root); e == nil {
				t.Fatal("invalid catalogue accepted")
			}
		})
	}
}
func TestCostStarterPredicateBoundary(t *testing.T) {
	cat, e := Load("../../knowledge")
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range []any{nil, map[string]any{"field": "hdr.cond.no_such", "in": []any{"da"}}, map[string]any{"field": "hdr.cond.planning", "in": []any{}}, map[string]any{"field": "hdr.cond.planning", "in": []any{true}}} {
		if e = cat.validateCostStarter(p); e == nil {
			t.Fatal("accepted invalid predicate", p)
		}
	}
	rule := map[string]any{"field": "hdr.cond.planning", "in": []any{"da"}}
	for value, want := range map[string]Truth{"": Unknown, "da": True, "other": False} {
		if got := cat.CostStarterHolds(rule, map[string]string{"hdr.cond.planning": value}); got != want {
			t.Fatal(value, got, want)
		}
	}
}
