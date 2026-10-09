package render

import (
	"strings"
	"testing"

	"sitewise/internal/reports"
)

func TestFitCountsDraftPagesWithoutIssuing(t *testing.T) {
	s := fixture()
	fit := Fit(s)
	if fit.Pages != 1 || !fit.Fits || fit.Limit != 2 || fit.Target != 1.5 || fit.Error != "" {
		t.Fatalf("short draft %+v", fit)
	}
	long := reports.Block{ID: "long", Label: "Scope", Text: strings.Repeat("Coordinate the works with the occupied building. ", 400)}
	s.Sections = append(s.Sections, reports.Section{ID: "more", Title: "More", Blocks: []reports.Block{long}})
	fit = Fit(s)
	if fit.Pages <= 2 || fit.Fits || fit.Error != "" {
		t.Fatalf("long draft %+v", fit)
	}
}
