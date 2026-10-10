package knowledge_test

import (
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func TestProjectDetailsLoadWithTheirScope(t *testing.T) {
	root := profileFixture(t, `'x'`)
	mustWrite(t, filepath.Join(root, "profile", "project_details.yaml"), `version: 1
status: draft
details:
  - {key: client, label: Client, value: text, scope: project}
  - {key: zone, label: Zone, value: text, scope: site}
  - {key: regulated_design, label: Regulated design applies, value: yes_no, scope: project}
`)
	cat, err := knowledge.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := cat.ProjectDetail("zone"); !ok || d.Label != "Zone" || d.Value != "text" {
		t.Fatalf("zone = %+v, %v", d, ok)
	}
	if len(cat.ProjectDetails()) != 3 {
		t.Fatalf("details = %d", len(cat.ProjectDetails()))
	}
	if cat.KeyScope("proj.zone") != "site" || cat.KeyScope("proj.client") != "project" || cat.KeyScope("proj.other") != "" {
		t.Fatal("detail key scope")
	}
	if d, _ := cat.ProjectDetail("regulated_design"); d.Check("maybe") == "" || d.Check("yes") != "" || d.Check("no") != "" {
		t.Fatal("yes_no check")
	}
}

func TestProjectDetailsRefuseBadEntries(t *testing.T) {
	for _, details := range []string{
		`[{key: Client, label: Client, value: text, scope: project}]`,
		`[{key: client, label: "", value: text, scope: project}]`,
		`[{key: client, label: Client, value: money, scope: project}]`,
		`[{key: client, label: Client, value: text, scope: lot}]`,
		`[{key: client, label: Client, value: text, scope: project}, {key: client, label: Again, value: text, scope: project}]`,
	} {
		root := profileFixture(t, `'x'`)
		mustWrite(t, filepath.Join(root, "profile", "project_details.yaml"), "version: 1\nstatus: draft\ndetails: "+details+"\n")
		if _, err := knowledge.Load(root); err == nil || !strings.Contains(err.Error(), "project detail") {
			t.Fatalf("%s: err = %v", details, err)
		}
	}
}
