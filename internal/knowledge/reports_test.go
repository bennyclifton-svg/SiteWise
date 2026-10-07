package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportTemplateReferencesAndDraftBoundary(t *testing.T) {
	cat, err := Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	template, ok := cat.ReportTemplate("tpl.rfp-capex", 1)
	if !ok || template.Status != "draft" || len(template.Sections) != 7 {
		t.Fatal("missing draft RFP template")
	}
	for _, section := range template.Sections {
		if !section.Essential {
			t.Fatal("essential section omitted", section.ID)
		}
		for _, ref := range section.Clauses {
			clause, ok := cat.Clause(ref.ID, ref.Version)
			if !ok || clause.Status != "draft" || cat.ApprovedClause(ref.ID, ref.Version) {
				t.Fatal("draft clause treated as approved", ref)
			}
		}
	}
	if _, ok := cat.ReportTemplate("tpl.rfp-capex", 2); ok {
		t.Fatal("wrong template version accepted")
	}
	raw, err := os.ReadFile("../../knowledge/reports/templates.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"essential missing": strings.Replace(string(raw), "essential: true", "", 1),
		"duplicate section": strings.Replace(string(raw), "id: services", "id: brief", 1),
		"wrong output":      strings.Replace(string(raw), "kind: rfp", "kind: rft", 1),
		"missing clause":    strings.Replace(string(raw), "cl.rfp-draft-status", "cl.missing", 1),
		"wrong version":     strings.Replace(string(raw), "cl.rfp-draft-status, version: 1", "cl.rfp-draft-status, version: 2", 1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "templates.yaml")
			if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
				t.Fatal(err)
			}
			c := &Catalog{loaded: map[string][]byte{}, clauses: cat.clauses}
			if err := c.loadReportTemplates(path); err == nil {
				t.Fatal("invalid template accepted")
			}
		})
	}
}
