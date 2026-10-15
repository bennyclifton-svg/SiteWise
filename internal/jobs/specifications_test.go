package jobs

import (
	"sitewise/internal/store"
	"testing"
)

func TestSpecificationGeometryRefreshPreservesOCRPages(t *testing.T) {
	cached := []store.SourcePage{{Page: 1, Text: "Old native extraction"}, {Page: 2, Text: "Recovered scanned requirement"}}
	fresh := []store.SourcePage{{Page: 1, Text: "Native extraction with positions", Regions: []store.SourceRegion{{Text: "CODE", X: 30}}}, {Page: 2}}
	got := sourceWithGeometry(cached, fresh)
	if got[0].Text != fresh[0].Text || len(got[0].Regions) != 1 || got[1].Text != cached[1].Text || cached[0].Text != "Old native extraction" {
		t.Fatal("lost native positions or cached OCR evidence", got)
	}
}
