package rfp

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"sitewise/internal/export"
)

// Every golden RFP exports as a PDF and a Word file (owner direction,
// 8 November 2026). SITEWISE_EXPORT_SAMPLE=dir also writes them to look at.
func TestEveryGoldenRFPExports(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	tpl := consultantTemplate(t)
	dir := os.Getenv("SITEWISE_EXPORT_SAMPLE")
	for _, s := range loadScenarios(t) {
		if !s.Snapshot && s.Name != "arthur-st" && s.Name != "yarramundi" {
			continue
		}
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
				doc := export.FromSections("RFP "+id, s.Name, "2026-11-08", sections)
				pdf, err := export.PDF(doc)
				if err != nil {
					t.Fatalf("PDF: %v", err)
				}
				word, err := export.DOCX(doc)
				if err != nil {
					t.Fatalf("Word: %v", err)
				}
				if dir != "" {
					base := filepath.Join(dir, s.Name+"-"+id)
					_ = os.WriteFile(base+".pdf", pdf, 0o644)
					_ = os.WriteFile(base+".docx", word, 0o644)
				}
			})
		}
	}
}
