package render

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sitewise/internal/reports"
	"sort"
	"strings"
	"testing"
	"time"
)

func fixture() reports.IssueSnapshot {
	return reports.IssueSnapshot{SchemaVersion: 1, ReportID: "report", VersionID: "issue", Kind: "rft", Title: "Petersham — Services works", ReportingDate: "2026-10-07", RendererVersion: Version, Sections: []reports.Section{
		{ID: "brief", Title: "Project brief", Blocks: []reports.Block{{ID: "brief", Label: "Works", Text: "Upgrade the plant and coordinate connections with the existing occupied building.", CitationIDs: []string{"U1"}}}},
		{ID: "scope", Title: "Responsibilities and retained systems", Blocks: []reports.Block{{ID: "scope", Label: "Scope", Text: "Provide installation, testing and commissioning. Keep the retained fire-water supply in operation during the works.", CitationIDs: []string{"U2"}}}},
		{ID: "dates", Title: "Dates and assumptions", Blocks: []reports.Block{{ID: "dates", Label: "Shutdown window", Text: "Confirm the shutdown with the building manager before mobilisation.", Origin: "assumption", Provisional: true, CitationIDs: []string{"A1"}}}},
	}, References: []reports.Reference{{ID: "U1", Label: "U", Basis: reports.ReferenceBasis{Text: "User input: project brief; accepted for planning, 7 October 2026."}}, {ID: "U2", Label: "U", Basis: reports.ReferenceBasis{Text: "User input: installation and retained-system responsibilities; accepted for planning."}}, {ID: "A1", Label: "A", Basis: reports.ReferenceBasis{Text: "Assumption: shutdown access remains subject to building-manager confirmation.", MaterialAssumption: true}}}}
}

func TestPDFComparisonTableAndCitationColours(t *testing.T) {
	s := fixture()
	s.Sections[0].Blocks[0].CitationIDs = []string{"E1", "U1", "C1", "A1"}
	s.Sections[0].Blocks[0].Edited = true
	s.Sections[0].Blocks[0].Table = &reports.Table{Columns: []string{"Target", "Current (Forecast)", "Variance"}, Rows: [][]string{{"2026-10-08", "2026-10-10", "2 days later"}, {"Unknown", "Unknown", "Unknown"}}}
	r, err := PDF(s)
	if err != nil {
		t.Fatal(err)
	}
	var streams []byte
	for _, match := range regexp.MustCompile(`(?s)stream\r?\n(.*?)endstream`).FindAllSubmatch(r.PDF, -1) {
		reader, err := zlib.NewReader(bytes.NewReader(match[1]))
		if err != nil {
			continue
		}
		body, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		streams = append(streams, body...)
	}
	for label, rgb := range citationColours {
		operator := fmt.Sprintf("%.3f %.3f %.3f rg", float64(rgb[0])/255, float64(rgb[1])/255, float64(rgb[2])/255)
		if !bytes.Contains(streams, []byte(operator)) {
			t.Fatalf("missing %c citation colour %s", label, operator)
		}
	}
	if path := os.Getenv("SITEWISE_RENDER_TABLE_SAMPLE"); path != "" {
		if err := os.WriteFile(path, r.PDF, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s.Sections[0].Blocks[0].Table.Rows = [][]string{{strings.Repeat("Saved value ", 10000), "a", "b"}}
	if _, err := PDF(s); !errors.Is(err, ErrOverflow) {
		t.Fatal("oversized table row accepted", err)
	}
	s.Sections[0].Blocks[0].Table.Rows = [][]string{{"missing cells"}}
	if _, err := PDF(s); !errors.Is(err, ErrContent) {
		t.Fatal("malformed table accepted", err)
	}
}

func TestPDFReadableBoundedAndRepeatable(t *testing.T) {
	s := fixture()
	r, err := PDF(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.Pages > 2 || r.MinimumFont < 8.5 || !bytes.HasPrefix(r.PDF, []byte("%PDF-")) {
		t.Fatal("invalid PDF", r.Pages, r.MinimumFont)
	}
	other, err := PDF(s)
	if err != nil || !bytes.Equal(r.PDF, other.PDF) {
		t.Fatal("same snapshot changed export", err)
	}
	if path := os.Getenv("SITEWISE_RENDER_SAMPLE"); path != "" {
		if err := os.WriteFile(path, r.PDF, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s.Sections[0].Blocks[0].Text = strings.Repeat("Essential scope must remain visible. ", 2000)
	if result, err := PDF(s); !errors.Is(err, ErrOverflow) || len(result.PDF) != 0 {
		t.Fatal("overflow silently exported", err)
	}
	s = fixture()
	s.Sections[0].Blocks[0].Text = "unsupported \x00 input"
	if _, err := PDF(s); !errors.Is(err, ErrContent) {
		t.Fatal("unsupported text accepted")
	}
}

func TestPDFExportSpeedBudget(t *testing.T) {
	var times []time.Duration
	for i := 0; i < 40; i++ {
		start := time.Now()
		if _, err := PDF(fixture()); err != nil {
			t.Fatal(err)
		}
		times = append(times, time.Since(start))
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	t.Logf("report_export p50=%s p90=%s; budget 100/250ms", times[19], times[35])
	if times[19] > 100*time.Millisecond || times[35] > 250*time.Millisecond {
		t.Fatal("PDF export exceeded speed budget")
	}
}
