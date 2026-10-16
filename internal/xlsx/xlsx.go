// Package xlsx writes small Office Open XML workbooks with the standard
// library: the invoice ledger export, the RFP response form and the
// procurement comparison.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Cell is one spreadsheet value. Money is exact cents written as a number;
// dates are Excel serials with a date format, so the workbook can be
// reworked rather than retyped.
type Cell struct {
	Text    string
	Cents   *int64
	Number  *float64
	Date    string // YYYY-MM-DD
	Month   string // YYYY-MM-DD, shown as the month
	Formula string // without "="; Cents holds the cached value
	Bold    bool
}

// Sheet is one worksheet. TitleRow (1-based) repeats on every printed page.
type Sheet struct {
	Name     string
	Widths   []float64
	Rows     [][]Cell
	TitleRow int
}

// Style indices in styles.xml below.
const (
	styleText = iota
	styleBold
	styleMoney
	styleMoneyBold
	styleDate
	styleMonth
	styleNumber
)

// WriteXLSX writes a minimal Office Open XML workbook. It is the standard
// library's zip and XML escaping: no dependency for two plain sheets.
// https://learn.microsoft.com/en-us/openspecs/office_standards/ms-xlsx
func WriteXLSX(w io.Writer, sheets []Sheet) error {
	z := zip.NewWriter(w)
	file := func(name, body string) error {
		f, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}
	var types, rels, entries, names strings.Builder
	for i, s := range sheets {
		n := i + 1
		fmt.Fprintf(&types, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, n)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, n, n)
		fmt.Fprintf(&entries, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(sheetName(s.Name)), n, n)
		if s.TitleRow > 0 {
			fmt.Fprintf(&names, `<definedName name="_xlnm.Print_Titles" localSheetId="%d">'%s'!$%d:$%d</definedName>`, i, esc(strings.ReplaceAll(sheetName(s.Name), "'", "''")), s.TitleRow, s.TitleRow)
		}
	}
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` + types.String() + `</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` + entries.String() + `</sheets>` + definedNames(names.String()) + `<calcPr calcId="0" fullCalcOnLoad="1"/></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rels.String() + fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, len(sheets)+1) + `</Relationships>`},
		{"xl/styles.xml", stylesXML},
	}
	for _, p := range parts {
		if err := file(p.name, p.body); err != nil {
			return err
		}
	}
	for i, s := range sheets {
		if err := file(fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), sheetXML(s)); err != nil {
			return err
		}
	}
	return z.Close()
}

func definedNames(s string) string {
	if s == "" {
		return ""
	}
	return "<definedNames>" + s + "</definedNames>"
}

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
	`<numFmts count="2"><numFmt numFmtId="164" formatCode="d mmm yyyy"/><numFmt numFmtId="165" formatCode="mmm yyyy"/></numFmts>` +
	`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
	`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
	`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="7"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`<xf numFmtId="4" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="4" fontId="1" fillId="0" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1"/>` +
	`<xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>` +
	`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>` +
	`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`

func sheetXML(s Sheet) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	if s.TitleRow > 0 {
		fmt.Fprintf(&b, `<sheetViews><sheetView workbookViewId="0"><pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`, s.TitleRow, s.TitleRow+1)
	}
	if len(s.Widths) > 0 {
		b.WriteString("<cols>")
		for i, w := range s.Widths {
			fmt.Fprintf(&b, `<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, w)
		}
		b.WriteString("</cols>")
	}
	b.WriteString("<sheetData>")
	for r, row := range s.Rows {
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c, cell := range row {
			ref := ColumnName(c) + strconv.Itoa(r+1)
			writeCell(&b, ref, cell)
		}
		b.WriteString("</row>")
	}
	b.WriteString(`</sheetData><pageMargins left="0.5" right="0.5" top="0.6" bottom="0.6" header="0.3" footer="0.3"/><pageSetup paperSize="9" orientation="landscape" fitToWidth="1" fitToHeight="0"/></worksheet>`)
	return b.String()
}

func writeCell(b *strings.Builder, ref string, c Cell) {
	switch {
	case c.Formula != "":
		style := styleMoney
		if c.Bold {
			style = styleMoneyBold
		}
		v := ""
		if c.Cents != nil {
			v = "<v>" + formatCents(*c.Cents) + "</v>"
		}
		fmt.Fprintf(b, `<c r="%s" s="%d"><f>%s</f>%s</c>`, ref, style, esc(c.Formula), v)
	case c.Cents != nil:
		style := styleMoney
		if c.Bold {
			style = styleMoneyBold
		}
		fmt.Fprintf(b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, style, formatCents(*c.Cents))
	case c.Number != nil:
		fmt.Fprintf(b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, styleNumber, strconv.FormatFloat(*c.Number, 'f', -1, 64))
	case c.Date != "" || c.Month != "":
		iso, style := c.Date, styleDate
		if iso == "" {
			iso, style = c.Month, styleMonth
		}
		if serial, ok := excelSerial(iso); ok {
			fmt.Fprintf(b, `<c r="%s" s="%d"><v>%d</v></c>`, ref, style, serial)
			return
		}
		fmt.Fprintf(b, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, esc(iso))
	case c.Text != "":
		style := styleText
		if c.Bold {
			style = styleBold
		}
		fmt.Fprintf(b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, style, esc(c.Text))
	}
}

// excelSerial is days since 30 December 1899, Excel's date origin.
func excelSerial(iso string) (int, bool) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return 0, false
	}
	return int(t.Sub(time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)).Hours() / 24), true
}

// ColumnName is the spreadsheet column for a zero-based index: 0 → A, 26 → AA.
func ColumnName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

func sheetName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return ' '
		}
		return r
	}, s)
	if len(s) > 31 {
		s = s[:31]
	}
	return s
}

func esc(s string) string {
	var b strings.Builder
	// Control characters other than tab and newline are not allowed in XML.
	s = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' {
			return -1
		}
		return r
	}, s)
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// formatCents is the exact decimal for a money cell, e.g. 1100000 → "11000.00".
func formatCents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	frac := strconv.FormatInt(c%100, 10)
	if len(frac) < 2 {
		frac = "0" + frac
	}
	return sign + strconv.FormatInt(c/100, 10) + "." + frac
}
