package rfp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
)

// TestScenarioRFPsFitThePageLimit renders every scenario RFP through the
// issue renderer, at full body size and with nothing dropped, and holds it
// to the consultant RFP page limit (owner decision O1: three pages).
func TestScenarioRFPsFitThePageLimit(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	tpl := consultantTemplate(t)
	for _, s := range loadScenarios(t) {
		ids := make([]string, 0, len(s.Expect))
		for id := range s.Expect {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			t.Run(s.Name+"/"+id, func(t *testing.T) {
				d, _ := cat.RFP(id)
				drivers := s.drivers(t, sk, d)
				sections, err := Sections(Resolve(d, sk, drivers, s.settings(t, id, drivers)), tpl)
				if err != nil {
					t.Fatal(err)
				}
				pages, length, err := renderPages(s.Name+"/"+id, sections)
				t.Logf("%s/%s: %d pages (%.2f)", s.Name, id, pages, length)
				if err != nil {
					t.Fatalf("%d pages: %v", pages, err)
				}
			})
		}
	}
}

func renderPages(name string, sections []reports.Section) (int, float64, error) {
	sections, _, err := reports.Cite(sections)
	if err != nil {
		return 0, 0, err
	}
	issued, refs, err := reports.IssueContent(sections, "rfp", false)
	if err != nil {
		return 0, 0, err
	}
	snap := reports.IssueSnapshot{SchemaVersion: 1, ReportID: "r", VersionID: "v", Kind: "rfp", Title: "RFP", ReportingDate: "2026-10-12", SourceRevisions: reports.SourceState{TemplateID: "tpl.rfp-consultant"}, Sections: issued, References: refs, RendererVersion: render.Version}
	r, err := render.PDF(snap)
	if dir := os.Getenv("SITEWISE_RFP_PDF_DIR"); dir != "" {
		// Render without the page gate so an overlong draft can be inspected.
		if full, e := render.PDFUnlimited(snap); e == nil {
			_ = os.WriteFile(filepath.Join(dir, strings.ReplaceAll(name, "/", "-")+".pdf"), full.PDF, 0o644)
		}
	}
	return r.Pages, r.Length, err
}
