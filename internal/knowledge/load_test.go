package knowledge_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"sitewise/internal/knowledge"
)

func TestLoadHierarchyAndFireEdges(t *testing.T) {
	cat := loadRepo(t)
	sprinklers, ok := cat.System("fire-active.sprinklers")
	if !ok || sprinklers.Parent != "fire-active" {
		t.Fatalf("sprinklers %+v", sprinklers)
	}
	if len(cat.Children("fire-active")) == 0 {
		t.Fatal("fire-active children")
	}
	var supply, tanks, smoke, power bool
	for _, edge := range cat.Edges() {
		switch edge.InterfaceID {
		case "if.fire-water-supplies-sprinklers":
			if edge.From == "fire-active.fire-water" && edge.To == "fire-active.sprinklers" && edge.Type == "supplies" {
				supply = true
			}
		case "if.fire-tanks-load-structure":
			if edge.From == "fire-active.fire-water" && edge.To == "structure" && edge.Type == "loads" {
				tanks = true
			}
		case "if.fire-detection-controls-smoke-plant":
			if edge.Type == "controls" {
				smoke = true
			}
		case "if.essential-power-supplies-fire-water":
			if edge.From == "electrical" && edge.To == "fire-active.fire-water" {
				power = true
			}
		}
	}
	if !supply || !tanks || !smoke || !power {
		t.Fatalf("supply %v tanks %v smoke %v power %v", supply, tanks, smoke, power)
	}
}

func loadRepo(t *testing.T) *knowledge.Catalog {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	cat, err := knowledge.Load(filepath.Join(filepath.Dir(file), "..", "..", "knowledge"))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func writeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "systems.yaml"), `
version: 1
systems:
  - id: fire-passive
    label: Passive fire
    describes: Fire-resisting construction.
    excludes: Sprinklers.
    status: reviewed
  - id: structure
    label: Structure
    describes: The frame.
    excludes: Fire ratings.
    status: reviewed
`)
	mustWrite(t, filepath.Join(root, "tables", "type_of_construction.yaml"), `
version: 1
id: type_of_construction
status: reviewed
verified: true
inputs: [ncc_class, rise_in_storeys]
outputs: [type_of_construction]
rows:
  - {classes: ['5'], rise_min: 4, type_of_construction: A}
`)
	mustWrite(t, filepath.Join(root, "tables", "draft_limits.yaml"), `
version: 1
id: draft_limits
status: draft
verified: true
inputs: [ncc_class]
outputs: [max_floor_area_m2]
rows:
  - {classes: ['5'], max_floor_area_m2: 8000}
`)
	cluster := filepath.Join(root, "clusters", "fire")
	mustWrite(t, filepath.Join(cluster, "rules.yaml"), `
version: 1
rules:
  - id: rule.demo.type
    status: reviewed
    derives:
      table: type_of_construction
      inputs: [ncc_class, rise_in_storeys]
      gives: type_of_construction
  - id: rule.demo.pending
    status: reviewed
    derives:
      table: type_of_construction
      inputs: [ncc_class, rise_in_storeys]
      gives: type_of_construction
      pending: true
      reason: table not ready
  - id: rule.demo.missing
    status: reviewed
    derives:
      table: missing_table
      inputs: [ncc_class]
      gives: type_of_construction
  - id: rule.demo.draft
    status: draft
    derives:
      table: type_of_construction
      inputs: [ncc_class, rise_in_storeys]
      gives: type_of_construction
  - id: rule.demo.draft-table
    status: reviewed
    derives:
      table: draft_limits
      inputs: [ncc_class]
      gives: max_floor_area_m2
`)
	mustWrite(t, filepath.Join(cluster, "interfaces.yaml"), `
version: 1
interfaces:
  - id: if.fire-water-supplies-sprinklers
    type: supplies
    from: [fire-passive]
    to: [structure]
    status: draft
    resolved_when:
      - type: noul
        instructions: Using `+"`text`"+`, does this passage state the water supply?
        criteria:
          true: It names a supply.
          false: It does not.
        runs_on: [fire-passive]
`)
	return root
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
