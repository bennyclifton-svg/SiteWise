//go:build ignore

// Command make writes the identity fixtures and manifest.
// Run from the repo root: go run testdata/identity/make.go
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	dir := filepath.Dir(os.Args[0])
	if strings.HasSuffix(filepath.ToSlash(dir), "go-build") || dir == "." || filepath.Base(dir) == "exe" {
		dir = "testdata/identity"
	}
	// go run puts the binary in a temp dir. Write next to this source file
	// when launched from the repo, which is the documented command.
	if _, err := os.Stat("testdata/identity/make.go"); err == nil {
		dir = "testdata/identity"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err)
	}
	files := []struct {
		name string
		body []byte
	}{
		{"rotated-title-block.pdf", rotatedTitleBlockPDF()},
		{"scanned-empty.pdf", emptyPagePDF()},
		{"identity-page.pdf", denseIdentityPDF()},
		{"fragmented-title-block.pdf", fragmentedTitleBlockPDF()},
		{"malformed.docx", []byte("this is not a zip document")},
		{"large-shared-strings.xlsx", largeSharedStringsXLSX()},
		{"merged-cells.xlsx", mergedCellsXLSX()},
		{"docx-table.docx", docxTable()},
		{"padded-document.docx", paddedDOCX()},
	}
	var manifest manifest
	for _, file := range files {
		path := filepath.Join(dir, file.name)
		if err := os.WriteFile(path, file.body, 0o644); err != nil {
			fatal(err)
		}
		sum := sha256.Sum256(file.body)
		manifest.Fixtures = append(manifest.Fixtures, fixture{
			File:   file.name,
			SHA256: hex.EncodeToString(sum[:]),
			Bytes:  len(file.body),
		})
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644); err != nil {
		fatal(err)
	}
}

type manifest struct {
	Fixtures []fixture `json:"fixtures"`
}

type fixture struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func writePDF(objects []string) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, body := range objects {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objects)+1)
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return []byte(b.String())
}

func pdfStream(content string) string {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
}

func pageDict(contentsID, fontID int) string {
	return fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 842 595] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R >> >> >>", contentsID, fontID)
}

const fontDict = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"

func rotatedTitleBlockPDF() []byte {
	// 0 1 -1 0 is a quarter turn. PDFium reports the glyph angle as 270 degrees counterclockwise.
	page1 := "BT\n/F1 12 Tf\n1 0 0 1 72 520 Tm\n(GROUND FLOOR) Tj\n0 1 -1 0 760 72 Tm\n(A-101) Tj\nET"
	page2 := "BT\n/F1 12 Tf\n1 0 0 1 72 520 Tm\n(PAGE TWO SECRET) Tj\nET"
	return writePDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 6 0 R] /Count 2 >>",
		pageDict(4, 5),
		pdfStream(page1),
		fontDict,
		pageDict(7, 5),
		pdfStream(page2),
	})
}

func emptyPagePDF() []byte {
	return writePDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		pageDict(4, 5),
		pdfStream(""),
		fontDict,
	})
}

func denseIdentityPDF() []byte {
	var content strings.Builder
	content.WriteString("BT\n/F1 10 Tf\n")
	for i := 1; i <= 40; i++ {
		y := 560 - (i-1)*12
		fmt.Fprintf(&content, "1 0 0 1 36 %d Tm\n(Fire note %02d hydrant booster pump room access) Tj\n", y, i)
	}
	content.WriteString("0 1 -1 0 800 48 Tm\n(A-101) Tj\nET")
	return writePDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		pageDict(4, 5),
		pdfStream(content.String()),
		fontDict,
	})
}

// fragmentedTitleBlockPDF writes text the way ArchiCAD publishes it: every
// word and punctuation mark is its own text object, placed edge to edge.
// 500 note words come first in the content stream, so a reader that keeps
// the first 400 fragments never reaches the title block.
func fragmentedTitleBlockPDF() []byte {
	var content strings.Builder
	frag := func(x, y float64, s string) {
		fmt.Fprintf(&content, "BT\n/F1 10 Tf\n1 0 0 1 %.2f %.2f Tm\n(%s) Tj\nET\n", x, y, s)
	}
	// Helvetica advance widths at 10 pt.
	width := map[rune]float64{'C': 7.22, '-': 3.33, 'A': 6.67, '0': 5.56, '1': 5.56, '2': 5.56, '3': 5.56, '6': 5.56, '.': 2.78, 'S': 6.67, 'I': 2.78, 'T': 6.11, 'E': 6.67, 'P': 6.67, 'L': 5.56, 'N': 7.22}
	run := func(x, y float64, parts ...string) {
		for _, p := range parts {
			frag(x, y, p)
			for _, r := range p {
				x += width[r]
			}
		}
	}
	for line := 0; line < 50; line++ {
		for w := 0; w < 10; w++ {
			frag(36+float64(w)*36, 560-float64(line)*10, "note")
		}
	}
	// Words are separated by a gap, not a space character.
	run(600, 80, "SITE")
	frag(600+22.23+2.78, 80, "PLAN")
	run(600, 60, "CC", "-", "A", "-", "010")
	run(600, 40, "06", ".", "11", ".", "2023")
	frag(420, 100, "DRAWING TITLE")
	frag(640, 100, "DRAWING NUMBER")
	return writePDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		pageDict(4, 5),
		pdfStream(content.String()),
		fontDict,
	})
}

func writeZip(members map[string][]byte) []byte {
	buf := &byteBuffer{}
	zw := zip.NewWriter(buf)
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			fatal(err)
		}
		if _, err := w.Write(members[name]); err != nil {
			fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		fatal(err)
	}
	return buf.Bytes()
}

type byteBuffer struct {
	buf []byte
}

func (b *byteBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *byteBuffer) Bytes() []byte { return b.buf }

func contentTypes(extra string) []byte {
	body := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  ` + extra + `
</Types>`
	return []byte(body)
}

func largeSharedStringsXLSX() []byte {
	const n = 5000
	var sst strings.Builder
	sst.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(&sst, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, n, n)
	sst.WriteString(`<si><t>Project Hale</t></si>`)
	for i := 1; i < n-1; i++ {
		fmt.Fprintf(&sst, `<si><t>PAD-%04d unused shared string padding for the identity cap</t></si>`, i)
	}
	sst.WriteString(`<si><t>SHARED-STRING-NOT-IN-IDENTITY</t></si></sst>`)
	sheet := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c>
      <c r="AA1" t="s"><v>4999</v></c>
    </row>
  </sheetData>
</worksheet>`
	return xlsxPackage("Register", []byte(sheet), []byte(sst.String()), nil)
}

func mergedCellsXLSX() []byte {
	sheet := `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="inlineStr"><is><t>Hale House</t></is></c></row>
    <row r="2"><c r="A2"><f>1+2</f><v>99</v></c></row>
    <row r="200"><c r="A200" t="inlineStr"><is><t>ROW-FAR</t></is></c></row>
  </sheetData>
  <mergeCells count="1"><mergeCell ref="A1:C1"/></mergeCells>
</worksheet>`
	return xlsxPackage("Cover", []byte(sheet), nil, []byte("macro-payload-do-not-execute"))
}

func xlsxPackage(sheetName string, sheet, sst, macro []byte) []byte {
	workbook := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="%s" sheetId="1" r:id="rId1"/></sheets>
</workbook>`, sheetName)
	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`
	root := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	members := map[string][]byte{
		"[Content_Types].xml":        contentTypes(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`),
		"_rels/.rels":                []byte(root),
		"xl/workbook.xml":            []byte(workbook),
		"xl/_rels/workbook.xml.rels": []byte(rels),
		"xl/worksheets/sheet1.xml":   sheet,
	}
	if sst != nil {
		members["xl/sharedStrings.xml"] = sst
	}
	if macro != nil {
		members["xl/vbaProject.bin"] = macro
	}
	return writeZip(members)
}

func docxTable() []byte {
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Fire Services Specification</w:t></w:r>
    </w:p>
    <w:p><w:r><w:t>Issued for tender</w:t></w:r></w:p>
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Drawing</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>A-101</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Revision</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>C</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`
	return writeZip(map[string][]byte{
		"[Content_Types].xml": contentTypes(`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`),
		"word/document.xml":   []byte(doc),
	})
}

func paddedDOCX() []byte {
	doc := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` +
		strings.Repeat("PADDING ", 2000) +
		`</w:t></w:r></w:p></w:body></w:document>`
	return writeZip(map[string][]byte{
		"[Content_Types].xml": contentTypes(""),
		"word/document.xml":   []byte(doc),
	})
}
