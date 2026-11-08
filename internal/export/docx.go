package export

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

// Word measures in twentieths of a point (twips) and half-points.
const (
	twip      = 20
	pageTwW   = 11906 // A4
	pageTwH   = 16838
	marginTw  = 850 // 15 mm
	contentTw = pageTwW - 2*marginTw
)

// Fixed keys obfuscate the embedded fonts (ECMA-376 Part 1, 17.8.1); fixed
// so the same draft always writes the same file.
const (
	keyRegular  = "{4E5B1C2D-7A61-4F0B-9C3E-2D8F6A1B0C51}"
	keySemiBold = "{9A2C3E4F-1B7D-4C6E-8F20-5D3A9B7C1E62}"
)

// DOCX writes the document as a Word file in the screen's style, with the
// screen's typeface embedded so Word shows it where it is not installed.
func DOCX(d Doc) ([]byte, error) {
	body, err := docxBody(d)
	if err != nil {
		return nil, err
	}
	regular, err := obfuscate(fontRegular, keyRegular)
	if err != nil {
		return nil, err
	}
	semibold, err := obfuscate(fontSemiBold, keySemiBold)
	if err != nil {
		return nil, err
	}
	files := []struct {
		name string
		data []byte
	}{
		{"[Content_Types].xml", []byte(contentTypes)},
		{"_rels/.rels", []byte(packageRels)},
		{"docProps/core.xml", []byte(fmt.Sprintf(coreProps, esc(d.Title)))},
		{"word/document.xml", body},
		{"word/styles.xml", []byte(fmt.Sprintf(styles, FontName, colourText.hex()))},
		{"word/settings.xml", []byte(settings)},
		{"word/fontTable.xml", []byte(fmt.Sprintf(fontTable, FontName, keyRegular, keySemiBold))},
		{"word/footer1.xml", []byte(fmt.Sprintf(footer, colourMuted.hex(), esc(d.Date)))},
		{"word/_rels/document.xml.rels", []byte(documentRels)},
		{"word/_rels/fontTable.xml.rels", []byte(fontRels)},
		{"word/fonts/font1.odttf", regular},
		{"word/fonts/font2.odttf", semibold},
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := z.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.data); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// obfuscate XORs the font's first 32 bytes with the key's bytes, read from
// the end of its hex digits, as Word expects of an embedded font.
func obfuscate(font []byte, guid string) ([]byte, error) {
	digits := strings.NewReplacer("{", "", "}", "", "-", "").Replace(guid)
	raw, err := hex.DecodeString(digits)
	if err != nil || len(raw) != 16 || len(font) < 32 {
		return nil, fmt.Errorf("font key %s", guid)
	}
	out := append([]byte(nil), font...)
	for i := 0; i < 32; i++ {
		out[i] ^= raw[15-i%16]
	}
	return out, nil
}

func docxBody(d Doc) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>`)
	para(&b, []Run{{Text: d.Title, Bold: true}}, 32, `w:before="0" w:after="40"`)
	if d.Subtitle != "" {
		para(&b, []Run{{Text: d.Subtitle, Muted: true}}, 18, `w:before="0" w:after="200"`)
	}
	for _, s := range d.Sections {
		if s.Title != "" {
			band(&b, s.Title)
		}
		for _, blk := range s.Blocks {
			runs := []Run{}
			if blk.Label != "" {
				runs = append(runs, Run{Text: blk.Label + ": ", Bold: true})
			}
			if blk.Text != "" {
				runs = append(runs, Run{Text: blk.Text})
			}
			if !blk.TextAfter && len(runs) > 0 {
				para(&b, runs, 20, `w:before="0" w:after="120" w:line="336" w:lineRule="auto"`)
			}
			if blk.Table != nil {
				if err := table(&b, *blk.Table); err != nil {
					return nil, err
				}
				// Word joins adjacent tables; a short spacer keeps them apart.
				para(&b, nil, 8, `w:before="0" w:after="0"`)
			}
			if blk.TextAfter && len(runs) > 0 {
				para(&b, runs, 20, `w:before="80" w:after="120" w:line="336" w:lineRule="auto"`)
			}
		}
	}
	fmt.Fprintf(&b, `<w:sectPr><w:footerReference w:type="default" r:id="rIdFooter"/><w:pgSz w:w="%d" w:h="%d"/>`+
		`<w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="425" w:footer="425" w:gutter="0"/></w:sectPr></w:body></w:document>`,
		pageTwW, pageTwH, marginTw, marginTw, marginTw, marginTw)
	return []byte(b.String()), nil
}

// para writes a paragraph; spacing is its w:spacing attributes.
func para(b *strings.Builder, runs []Run, size int, spacing string) {
	fmt.Fprintf(b, `<w:p><w:pPr><w:spacing %s/></w:pPr>`, spacing)
	writeRuns(b, runs, size, "")
	b.WriteString(`</w:p>`)
}

// band is a section's blue head: uppercase title on the tint, a strong rule
// beneath, kept with what follows.
func band(b *strings.Builder, title string) {
	fmt.Fprintf(b, `<w:p><w:pPr><w:keepNext/><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="4" w:color="%s"/></w:pBdr>`+
		`<w:shd w:val="clear" w:color="auto" w:fill="%s"/><w:spacing w:before="240" w:after="120" w:line="440" w:lineRule="exact"/><w:ind w:left="120" w:right="120"/></w:pPr>`,
		colourStrong.hex(), colourBand.hex())
	fmt.Fprintf(b, `<w:r><w:rPr><w:b/><w:caps/><w:color w:val="%s"/><w:spacing w:val="12"/><w:sz w:val="16"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>`,
		colourHead.hex(), esc(title))
}

func writeRuns(b *strings.Builder, runs []Run, size int, extra string) {
	for _, r := range runs {
		colour := colourText
		if r.Muted {
			colour = colourMuted
		}
		for i, part := range strings.Split(r.Text, "\n") {
			b.WriteString(`<w:r><w:rPr>`)
			if r.Bold {
				b.WriteString(`<w:b/>`)
			}
			fmt.Fprintf(b, `%s<w:color w:val="%s"/><w:sz w:val="%d"/></w:rPr>`, extra, colour.hex(), size)
			if i > 0 {
				b.WriteString(`<w:br/>`)
			}
			if part != "" {
				fmt.Fprintf(b, `<w:t xml:space="preserve">%s</w:t>`, esc(part))
			}
			b.WriteString(`</w:r>`)
		}
	}
}

func table(b *strings.Builder, t Table) error {
	widths := columnWidths(t.Columns, float64(contentTw)/twip)
	tw := make([]int, len(widths))
	total := 0
	for i, w := range widths {
		tw[i] = int(w*twip + 0.5)
		total += tw[i]
	}
	line := colourLine.hex()
	fmt.Fprintf(b, `<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/>`+
		`<w:tblBorders><w:top w:val="single" w:sz="4" w:color="%[2]s"/><w:left w:val="single" w:sz="4" w:color="%[2]s"/><w:bottom w:val="single" w:sz="4" w:color="%[2]s"/><w:right w:val="single" w:sz="4" w:color="%[2]s"/><w:insideH w:val="single" w:sz="4" w:color="%[2]s"/><w:insideV w:val="single" w:sz="4" w:color="%[2]s"/></w:tblBorders><w:tblLayout w:type="fixed"/>`+
		`<w:tblCellMar><w:top w:w="60" w:type="dxa"/><w:left w:w="120" w:type="dxa"/><w:bottom w:w="60" w:type="dxa"/><w:right w:w="120" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid>`, total, line)
	for _, w := range tw {
		fmt.Fprintf(b, `<w:gridCol w:w="%d"/>`, w)
	}
	b.WriteString(`</w:tblGrid>`)
	// The head repeats on each page and has a strong rule beneath.
	fmt.Fprintf(b, `<w:tr><w:trPr><w:tblHeader/><w:cantSplit/><w:trHeight w:val="%d" w:hRule="atLeast"/></w:trPr>`, int(rowHeight*twip))
	col := 0
	for _, c := range t.Head {
		n := span(c.ColSpan)
		if col+n > len(tw) {
			return fmt.Errorf("%w: a table head is wider than its columns", ErrContent)
		}
		cell(b, c, sum(tw[col:col+n]), n, "", true, false)
		col += n
	}
	b.WriteString(`</w:tr>`)
	// Cells spanning down from above leave a merged continuation cell in
	// each row below, as Word requires.
	type down struct{ left, cols int }
	occupied := make([]down, len(tw))
	for _, r := range t.Rows {
		fmt.Fprintf(b, `<w:tr><w:trPr><w:cantSplit/><w:trHeight w:val="%d" w:hRule="atLeast"/></w:trPr>`, int(r.MinHeight*twip))
		col := 0
		next := 0
		for col < len(tw) {
			if occupied[col].left > 0 {
				n := occupied[col].cols
				fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s<w:vMerge/></w:tcPr><w:p/></w:tc>`, sum(tw[col:col+n]), gridSpan(n))
				occupied[col].left--
				col += n
				continue
			}
			if next >= len(r.Cells) {
				// A short row: fill the rest so Word keeps the grid.
				fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr><w:p/></w:tc>`, tw[col])
				col++
				continue
			}
			c := r.Cells[next]
			next++
			n := span(c.ColSpan)
			if col+n > len(tw) {
				return fmt.Errorf("%w: a table row is wider than its columns", ErrContent)
			}
			merge := ""
			if c.RowSpan > 1 {
				merge = `<w:vMerge w:val="restart"/>`
				occupied[col] = down{left: c.RowSpan - 1, cols: n}
			}
			cell(b, c, sum(tw[col:col+n]), n, merge, false, r.Strong)
			col += n
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return nil
}

func cell(b *strings.Builder, c Cell, width, cols int, merge string, head, strong bool) {
	fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s%s`, width, gridSpan(cols), merge)
	if strong {
		fmt.Fprintf(b, `<w:tcBorders><w:top w:val="single" w:sz="6" w:color="%s"/></w:tcBorders>`, colourStrong.hex())
	}
	if head {
		fmt.Fprintf(b, `<w:tcBorders><w:bottom w:val="single" w:sz="6" w:color="%s"/></w:tcBorders><w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, colourStrong.hex(), colourBand.hex())
	}
	valign := "center"
	if c.RowSpan > 1 {
		valign = "top"
	}
	fmt.Fprintf(b, `<w:vAlign w:val="%s"/></w:tcPr><w:p><w:pPr><w:spacing w:before="0" w:after="0" w:line="300" w:lineRule="auto"/>`, valign)
	if c.Align == Center {
		b.WriteString(`<w:jc w:val="center"/>`)
	}
	b.WriteString(`</w:pPr>`)
	switch {
	case c.Tick != NoTick:
		box, colour := "☐", colourStrong
		if c.Tick == TickOn {
			box, colour = "☑", colourTick
		}
		fmt.Fprintf(b, `<w:r><w:rPr><w:rFonts w:ascii="Segoe UI Symbol" w:hAnsi="Segoe UI Symbol" w:cs="Segoe UI Symbol"/><w:color w:val="%s"/><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r>`, colour.hex(), box)
	case head:
		fmt.Fprintf(b, `<w:r><w:rPr><w:b/><w:caps/><w:color w:val="%s"/><w:spacing w:val="12"/><w:sz w:val="16"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r>`, colourHead.hex(), esc(runText(c.Runs)))
	default:
		writeRuns(b, c.Runs, 20, "")
	}
	b.WriteString(`</w:p></w:tc>`)
}

func runText(runs []Run) string {
	var s strings.Builder
	for _, r := range runs {
		s.WriteString(r.Text)
	}
	return s.String()
}

func gridSpan(n int) string {
	if n > 1 {
		return fmt.Sprintf(`<w:gridSpan w:val="%d"/>`, n)
	}
	return ""
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

// esc escapes text for XML and drops characters XML 1.0 cannot carry.
func esc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\t' || r == '\n' || r == '\r' || r >= 0x20 && r != 0xFFFE && r != 0xFFFF:
			b.WriteRune(r)
		}
	}
	return b.String()
}

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Default Extension="odttf" ContentType="application/vnd.openxmlformats-officedocument.obfuscatedFont"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
	`<Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>` +
	`<Override PartName="/word/fontTable.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.fontTable+xml"/>` +
	`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
	`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>` +
	`</Types>`

const packageRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
	`</Relationships>`

const coreProps = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">` +
	`<dc:title>%s</dc:title><dc:creator>SiteWise</dc:creator></cp:coreProperties>`

const documentRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
	`<Relationship Id="rIdSettings" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>` +
	`<Relationship Id="rIdFonts" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/fontTable" Target="fontTable.xml"/>` +
	`<Relationship Id="rIdFooter" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>` +
	`</Relationships>`

const fontRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rIdFont1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/font" Target="fonts/font1.odttf"/>` +
	`<Relationship Id="rIdFont2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/font" Target="fonts/font2.odttf"/>` +
	`</Relationships>`

const fontTable = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:fonts xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<w:font w:name="%s"><w:charset w:val="00"/><w:family w:val="swiss"/><w:pitch w:val="variable"/>` +
	`<w:embedRegular r:id="rIdFont1" w:fontKey="%s"/><w:embedBold r:id="rIdFont2" w:fontKey="%s"/></w:font>` +
	`<w:font w:name="Segoe UI Symbol"><w:charset w:val="00"/><w:family w:val="swiss"/><w:pitch w:val="variable"/></w:font>` +
	`</w:fonts>`

const settings = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:embedTrueTypeFonts/><w:defaultTabStop w:val="720"/><w:compat><w:compatSetting w:name="compatibilityMode" w:uri="http://schemas.microsoft.com/office/word" w:val="15"/></w:compat></w:settings>`

const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults>` +
	`<w:rPrDefault><w:rPr><w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:eastAsia="%[1]s" w:cs="%[1]s"/><w:color w:val="%[2]s"/><w:sz w:val="20"/><w:szCs w:val="20"/><w:lang w:val="en-AU"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	`<w:style w:type="table" w:default="1" w:styleId="TableNormal"><w:name w:val="Normal Table"/><w:tblPr><w:tblInd w:w="0" w:type="dxa"/><w:tblCellMar><w:top w:w="0" w:type="dxa"/><w:left w:w="108" w:type="dxa"/><w:bottom w:w="0" w:type="dxa"/><w:right w:w="108" w:type="dxa"/></w:tblCellMar></w:tblPr></w:style>` +
	`</w:styles>`

const footer = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:spacing w:before="0" w:after="0"/></w:pPr>` +
	`<w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:t xml:space="preserve">SiteWise | %[2]s | Page </w:t></w:r>` +
	`<w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:t>1</w:t></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="end"/></w:r>` +
	`<w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:t xml:space="preserve"> of </w:t></w:r>` +
	`<w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:instrText xml:space="preserve"> NUMPAGES </w:instrText></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:t>1</w:t></w:r><w:r><w:rPr><w:color w:val="%[1]s"/><w:sz w:val="16"/></w:rPr><w:fldChar w:fldCharType="end"/></w:r>` +
	`</w:p></w:ftr>`
