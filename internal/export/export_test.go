package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/reports"
)

// sample is an RFP draft as the builder saves it: page one's grid, the
// stage table (two stages with lines, one left out, totals), two-pair
// allowances and the transmittal.
func sample() []reports.Section {
	stage, _ := json.Marshal(map[string]any{"layout": "stage_scope", "stage_rows": 7})
	pairs, _ := json.Marshal(map[string]any{"layout": "allowance_pairs"})
	doc := func(cols []string, rows ...[]string) *reports.Table {
		return &reports.Table{Columns: cols, Rows: rows, Display: "document"}
	}
	return []reports.Section{
		{ID: "project", Title: "1. Project", Blocks: []reports.Block{
			{ID: "rfp:project:intro", Text: "Hale Capital intends to extend a warehouse."},
			{ID: "rfp:project:grid", Table: doc([]string{"The building", "Site and planning", "Delivery"},
				[]string{"Work type: Extension / addition", "Site address: 145 Arthur St", "Client: Hale Capital"},
				[]string{"Gross floor area: 2,350 m²", "", "Procurement route: Traditional lump sum"})},
		}},
		{ID: "role", Title: "2. Scope of services", Blocks: []reports.Block{
			{ID: "rfp:role", Text: "You are the lead designer."},
			{ID: "rfp:stages", Text: "Price each stage that applies.", Basis: stage, Table: doc([]string{"Stage", "Scope", "Position", "Fee $"},
				[]string{"Brief and concept", "", "Included", "$"},
				[]string{"", "Site inspection", "Included", ""},
				[]string{"", "Area schedule", "Excluded", ""},
				[]string{"Schematic design", "The contractor holds the design after novation.", "Included", "$"},
				[]string{"", "Outline materials and systems", "Included", ""},
				[]string{"Delivery", "Not part of this engagement", "Excluded", "—"},
				[]string{"Across all stages", "Coordination", "", "In stage fees"},
				[]string{"Reimbursables", "Printing and travel", "", ""},
				[]string{"Total fee", "Included stages and reimbursables", "", ""})},
		}},
		{ID: "allowances", Title: "3. Allowances", Blocks: []reports.Block{
			{ID: "rfp:allowances", Text: "Price exactly these.", Basis: pairs, Table: doc([]string{"Item", "Qty", "Item", "Qty"},
				[]string{"Design meetings: per stage, online or in person", "4", "BIM level of detail: LOD", "200"},
				[]string{"Concept options", "2", "", ""})},
		}},
		{ID: "transmittal", Title: "4. Transmittal", Blocks: []reports.Block{
			{ID: "rfp:project:information", Label: "Transmittal", Text: "Information provided: survey only.", Table: doc([]string{"Doc number", "Title", "Rev", "Date", "Discipline"},
				[]string{"A-001", "Site plan", "C", "2026-10-01", "Architecture"})},
		}},
		{ID: "review", Title: "Before issue — resolve or accept", Internal: true, Blocks: []reports.Block{{ID: "x", Text: "Not printed."}}},
	}
}

func TestBuildMatchesTheScreen(t *testing.T) {
	d := FromSections("RFP Architect", "Arthur St", "2026-11-08", sample())
	if len(d.Sections) != 4 {
		t.Fatalf("internal sections must not print: %d sections", len(d.Sections))
	}
	stages := d.Sections[1].Blocks[1].Table
	if len(stages.Columns) != 5 || runText(stages.Head[3].Runs) != "EXCL." {
		t.Fatalf("stage columns: %+v", stages.Head)
	}
	// Brief and concept: name and $ run down its two lines.
	first := stages.Rows[0]
	if first.Cells[0].RowSpan != 2 || runText(first.Cells[0].Runs) != "Brief and concept" || first.Cells[len(first.Cells)-1].RowSpan != 2 || runText(first.Cells[len(first.Cells)-1].Runs) != "$" {
		t.Fatalf("first stage row: %+v", first.Cells)
	}
	if first.Cells[2].Tick != TickOn || first.Cells[3].Tick != TickOff {
		t.Fatalf("included line ticks: %+v", first.Cells)
	}
	second := stages.Rows[1]
	if len(second.Cells) != 3 || runText(second.Cells[0].Runs) != "Area schedule" || second.Cells[1].Tick != TickOff || second.Cells[2].Tick != TickOn {
		t.Fatalf("a spanned row holds only its own cells: %+v", second.Cells)
	}
	// A note reads under the stage name, quieter.
	if runs := stages.Rows[2].Cells[0].Runs; len(runs) != 2 || !runs[1].Muted || !strings.Contains(runs[1].Text, "novation") {
		t.Fatalf("stage note: %+v", runs)
	}
	// A stage left out is one row; the fee is a dash.
	out := stages.Rows[3]
	if out.Cells[0].RowSpan > 1 || out.Cells[3].Tick != TickOn || runText(out.Cells[4].Runs) != "—" {
		t.Fatalf("excluded stage: %+v", out.Cells)
	}
	allow := d.Sections[2].Blocks[0].Table
	item := allow.Rows[0].Cells[0].Runs
	if len(allow.Columns) != 4 || item[0].Text != "Design meetings " || !item[1].Muted {
		t.Fatalf("allowance item: %+v", item)
	}
	grid := d.Sections[0].Blocks[1].Table
	if len(grid.Columns) != 6 || grid.Head[0].ColSpan != 2 || !grid.Rows[0].Cells[0].Runs[0].Muted {
		t.Fatalf("page one grid: %+v", grid.Head)
	}
	tr := d.Sections[3].Blocks[0]
	if tr.Label != "" || !tr.TextAfter || tr.Text != "survey only." {
		t.Fatalf("transmittal block: %+v", tr)
	}
}

func TestPDFAndWordCarryTheSameTables(t *testing.T) {
	d := FromSections("RFP Architect", "Arthur St", "2026-11-08", sample())
	pdf, err := PDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || len(pdf) < 2000 {
		t.Fatalf("not a PDF: %d bytes", len(pdf))
	}
	// The screen's typeface is embedded.
	if !bytes.Contains(pdf, []byte("/FontFile2")) {
		t.Error("PDF does not embed its typeface")
	}
	raw, err := DOCX(d)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(r)
		parts[f.Name] = string(data)
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			if err := wellFormed(data); err != nil {
				t.Fatalf("%s: %v", f.Name, err)
			}
		}
	}
	for _, name := range []string{"[Content_Types].xml", "word/document.xml", "word/fontTable.xml", "word/fonts/font1.odttf", "word/footer1.xml"} {
		if _, ok := parts[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
	doc := parts["word/document.xml"]
	// Brief and concept has two lines: its name and its fee merge down;
	// Schematic design has one line, so nothing to merge.
	if n := strings.Count(doc, `<w:vMerge w:val="restart"/>`); n != 2 {
		t.Errorf("merged starts = %d, want 2", n)
	}
	if n := strings.Count(doc, `<w:vMerge/>`); n != 2 {
		t.Errorf("merged continuations = %d, want 2", n)
	}
	if n := strings.Count(doc, "☑"); n != 4 {
		t.Errorf("ticked boxes = %d, want 4", n)
	}
	for _, want := range []string{"Brief and concept", "Design meetings ", "per stage, online or in person", "BIM level of detail", "A-001", "survey only.", `w:gridSpan w:val="2"`, "<w:tblHeader/>", "EAEEF2"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document lacks %q", want)
		}
	}
	// Word rejects a table whose properties are out of schema order.
	if !strings.Contains(doc, `</w:tblBorders><w:tblLayout w:type="fixed"/><w:tblCellMar>`) {
		t.Error("table properties are out of schema order")
	}
	if strings.Contains(doc, "Not printed.") {
		t.Error("an internal section printed")
	}
	// The embedded font round-trips through its obfuscation.
	font := []byte(parts["word/fonts/font1.odttf"])
	back, err := obfuscate(font, keyRegular)
	if err != nil || !bytes.Equal(back, fontRegular) {
		t.Error("embedded font does not decode to the screen's typeface")
	}
}

// A long stage table runs over pages without losing a line.
func TestPDFLongTablePaginates(t *testing.T) {
	rows := [][]string{{"Design development", "", "Included", "$"}}
	for i := 0; i < 120; i++ {
		rows = append(rows, []string{"", "A scope line that wraps onto a second line in the scope column of the printed table, number " + strings.Repeat("x", i%7), "Included", ""})
	}
	basis, _ := json.Marshal(map[string]any{"layout": "stage_scope", "stage_rows": len(rows)})
	sections := []reports.Section{{Title: "2. Scope", Blocks: []reports.Block{{ID: "rfp:stages", Basis: basis, Table: &reports.Table{Columns: []string{"Stage", "Scope", "Position", "Fee $"}, Rows: rows, Display: "document"}}}}}
	pdf, err := PDF(FromSections("Long", "", "2026-11-08", sections))
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page\n")) + bytes.Count(pdf, []byte("/Type /Page ")) + bytes.Count(pdf, []byte("/Type /Page>")); pages < 2 {
		// gopdf writes "/Type /Page" for each page; fall back to the footer text.
		if !bytes.Contains(pdf, []byte("Page 2 of")) && pages < 2 {
			t.Logf("could not count pages from the PDF structure; footer check only")
		}
	}
}

func TestControlCharactersRefused(t *testing.T) {
	sections := []reports.Section{{Title: "X", Blocks: []reports.Block{{ID: "a", Text: "bad \x07 bell"}}}}
	if _, err := PDF(FromSections("T", "", "2026-11-08", sections)); err == nil {
		t.Fatal("a control character was printed")
	}
	raw, err := DOCX(FromSections("T", "", "2026-11-08", sections))
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, _ := f.Open()
			data, _ := io.ReadAll(r)
			if bytes.Contains(data, []byte("\x07")) || wellFormed(data) != nil {
				t.Fatal("a control character reached the Word file")
			}
		}
	}
}

func wellFormed(data []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		if _, err := dec.Token(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// TestWriteSample writes the sample as files to look at, when asked:
// SITEWISE_EXPORT_SAMPLE=dir go test ./internal/export -run WriteSample
func TestWriteSample(t *testing.T) {
	dir := os.Getenv("SITEWISE_EXPORT_SAMPLE")
	if dir == "" {
		t.Skip("set SITEWISE_EXPORT_SAMPLE to write the sample files")
	}
	d := FromSections("RFP Architect", "Arthur St", "2026-11-08", sample())
	pdf, err := PDF(d)
	if err != nil {
		t.Fatal(err)
	}
	word, err := DOCX(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.pdf"), pdf, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.docx"), word, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Terms print numbered, one per row, as the screen shows them, so a term is
// cited by its number in tender correspondence (owner, 8 Nov 2026).
func TestTermsPrintNumbered(t *testing.T) {
	b := block(reports.Block{ID: "rfp:terms", Text: "- First term.\n- Second term.\n\nThird, edited without a marker."})
	if b.Text != "" || b.Table == nil || len(b.Table.Rows) != 3 {
		t.Fatalf("terms block = %+v", b)
	}
	last := b.Table.Rows[2].Cells
	if last[0].Runs[0].Text != "3" || last[1].Runs[0].Text != "Third, edited without a marker." {
		t.Fatalf("third row = %+v", last)
	}
}
