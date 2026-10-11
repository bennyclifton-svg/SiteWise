package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	intakevocab "sitewise/data/intake"
)

// Every suggested consultant must be a discipline the Procurement table can
// make an RFP for; a roster name like "Services Engineer" drifting from the
// discipline list is what this guards.
func TestPackageDefaultsNamesAreDisciplines(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	var vocab struct{ Disciplines []struct{ ID, Label string } }
	if err := json.Unmarshal(intakevocab.Disciplines, &vocab); err != nil {
		t.Fatal(err)
	}
	labels := map[string]bool{}
	for _, d := range vocab.Disciplines {
		if strings.HasPrefix(d.ID, "consultant.") {
			labels[d.Label] = true
		}
	}
	d := cat.PackageDefaults()
	var names []string
	for _, b := range d.Baselines {
		names = append(names, b.Consultants...)
	}
	for _, a := range d.ScopeAdditions {
		names = append(names, a.Consultants...)
	}
	for _, a := range d.ComplexityAdditions {
		names = append(names, a.Consultants...)
	}
	for _, n := range names {
		if !labels[n] {
			t.Errorf("consultant %q is not a discipline label", n)
		}
	}
	if len(d.ScopeAdditions) == 0 {
		t.Fatal("no scope additions loaded")
	}
}

func TestPackageDefaultsRejectsUnknownNamesAndSystems(t *testing.T) {
	for name, broken := range map[string][2]string{
		"consultant": {"  - Vertical Transport\n", "  - Lift Wizard\n"},
		"system":     {"  - vertical-transport\n", "  - lifts-magic\n"},
		"class":      {"  except_building_classes:\n  - infrastructure\n", "  except_building_classes:\n  - spaceport\n"},
	} {
		t.Run(name, func(t *testing.T) {
			root := copyKnowledge(t)
			path := filepath.Join(root, "works", "package_defaults.yaml")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.ReplaceAll(string(body), "\r\n", "\n")
			if !strings.Contains(text, broken[0]) {
				t.Fatalf("fixture text %q not found", broken[0])
			}
			text = strings.Replace(text, broken[0], broken[1], 1)
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatal("broken package defaults loaded")
			}
		})
	}
}
