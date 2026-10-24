package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutPlacesInputsInBuildOrder(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	order := cat.BuildOrder()
	if len(order) != len(cat.TopSystems()) || order[0] != "site" || order[1] != "substructure" {
		t.Fatalf("build order %v", order)
	}
	for key, want := range map[string]string{
		"proj.flag.basement":      "substructure", // listed
		"proj.flag.panel":         HomePlanning,
		"det.climate_zone":        "envelope",
		"det.ncc_class":           HomeProject,
		"det.flood_hazard":        HomeProject,
		"det.bushfire_prone_land": HomeProject,
		"det.bal":                 "envelope",
		"det.heritage_status":     HomeProject,
		"det.site_class":          "substructure", // first of its systems
		"det.water_supply_source": "hydraulic",    // a leaf system's top level
		"proj.client":             "",             // not placed
	} {
		if got := cat.Home(key); got != want {
			t.Errorf("Home(%s) = %q, want %q", key, got, want)
		}
	}
}

func TestLayoutRefusesABadPlacement(t *testing.T) {
	for name, edit := range map[string]func(string) string{
		"unknown home":        func(s string) string { return strings.Replace(s, "  planning:\n", "  roof:\n", 1) },
		"unknown flag":        func(s string) string { return strings.Replace(s, "proj.flag.panel", "proj.flag.no_such_flag", 1) },
		"unknown determinant": func(s string) string { return strings.Replace(s, "det.bal", "det.no_such", 1) },
		"missing system":      func(s string) string { return strings.Replace(s, "  - access-egress\n", "", 1) },
		"two homes": func(s string) string {
			return strings.Replace(s, "    - proj.flag.panel\n", "    - proj.flag.panel\n    - proj.flag.basement\n", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := copyKnowledge(t)
			p := filepath.Join(root, "profile", "layout.yaml")
			body, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			changed := edit(string(body))
			if changed == string(body) {
				t.Fatal("edit did not apply")
			}
			if err := os.WriteFile(p, []byte(changed), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "layout.yaml") {
				t.Fatalf("Load error = %v, want a layout.yaml error", err)
			}
		})
	}
}

// copyKnowledge copies the real knowledge tree so one file can be broken.
func copyKnowledge(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := "../../knowledge"
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}
