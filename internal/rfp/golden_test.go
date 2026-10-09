package rfp

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the RFP Markdown snapshots")

// TestGoldenRFPs is a source-fidelity snapshot: the transcribed rules for a
// fixture project render to reviewed Markdown. Selection is tested
// separately by the answer keys, so matching prose cannot preserve a wrong
// selection unnoticed.
func TestGoldenRFPs(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	tpl := consultantTemplate(t)
	for _, s := range loadScenarios(t) {
		if s.Name != "arthur-st" && s.Name != "yarramundi" {
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
				got := Markdown(sections)
				path := filepath.Join("testdata", s.Name, id+".md")
				if *update {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil || string(want) != got {
					t.Fatalf("snapshot mismatch for %s; run with -update and compare it with docs/rfp-golden/%s/rfp.md", path, id)
				}
			})
		}
	}
}
