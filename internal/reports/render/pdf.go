// Package render produces offline, embedded-font PDFs without external services.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"sitewise/internal/reports"
)

const Version = "sitewise-pdf-2/gopdf-0.38.1/go-font-0.38.0"
const BodySize = 10.0
const MinimumSize = 8.5
const margin = 42.52 // 15 mm in PDF points.
var ErrOverflow = errors.New("report exceeds its page limit; shorten the report or identify an attachment")

// PageLimit is the issued length allowed for a report's template. Consultant
// RFPs retain complete scope and element pricing in up to four pages after
// the October 2026 template consolidation. Three remains the target; other reports keep two.
func PageLimit(s reports.IssueSnapshot) (int, float64) {
	if s.SourceRevisions.TemplateID == "tpl.specifications" {
		return 100, 20
	}
	if s.SourceRevisions.TemplateID == "tpl.rfp-consultant" {
		return 4, 3
	}
	return 2, 1.5
}

var ErrContent = errors.New("report content cannot be exported")

type Result struct {
	PDF         []byte
	Pages       int
	MinimumFont float64
	// Length is the page count as a fraction: the last page's share used.
	Length float64
}
type line struct {
	text   string
	size   float64
	bold   bool
	gap    float64
	cells  [][]string
	height float64
	// widths and pad lay out a document table row; empty means equal widths
	// and the saved-comparison padding.
	widths []float64
	pad    float64
}

// PDF uses one typography specification for measurement and drawing. It never
// shrinks text or drops blocks to make the page-count gate pass.
func PDF(s reports.IssueSnapshot) (Result, error) { return renderPDF(s, true) }

// PDFUnlimited renders without the page gate, for inspecting a draft that is
// too long. It is never used to issue.
func PDFUnlimited(s reports.IssueSnapshot) (Result, error) { return renderPDF(s, false) }

func renderPDF(s reports.IssueSnapshot, gate bool) (Result, error) {
	var out Result
	if s.RendererVersion != Version {
		return out, fmt.Errorf("renderer version unavailable")
	}
	if len(s.Sections) > 100 || len(s.References) > 5000 {
		return out, fmt.Errorf("%w: report exceeds export limits", ErrContent)
	}
	gp := &gopdf.GoPdf{}
	gp.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := gp.AddTTFFontData("body", goregular.TTF); err != nil {
		return out, err
	}
	if err := gp.AddTTFFontData("heading", gobold.TTF); err != nil {
		return out, err
	}
	date, err := time.Parse("2006-01-02", s.ReportingDate)
	if err != nil {
		return out, fmt.Errorf("invalid reporting date")
	}
	gp.SetInfo(gopdf.PdfInfo{Title: s.Title, Author: "SiteWise", Creator: Version, CreationDate: date})
	font, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		return out, err
	}
	width := gopdf.PageSizeA4.W - 2*margin
	lines := []line{}
	appendText := func(text string, size float64, bold bool, gap float64) error {
		if len(text) > 200000 {
			return fmt.Errorf("%w: text exceeds limit", ErrContent)
		}
		text = strings.ReplaceAll(text, "\t", "    ")
		for _, r := range text {
			if r == '\n' || r == '\r' {
				continue
			}
			if unicode.IsControl(r) {
				return fmt.Errorf("%w: remove control character U+%04X", ErrContent, r)
			}
			glyph, e := font.GlyphIndex(nil, r)
			if e != nil || glyph == 0 {
				return fmt.Errorf("%w: font cannot represent character U+%04X", ErrContent, r)
			}
		}
		family := "body"
		if bold {
			family = "heading"
		}
		if err := gp.SetFont(family, "", size); err != nil {
			return err
		}
		for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
			// gopdf refuses an empty string; an empty paragraph is a blank line.
			wrapped := []string{""}
			if paragraph != "" {
				var err error
				if wrapped, err = gp.SplitTextWithWordWrap(paragraph, width); err != nil {
					return err
				}
			}
			for _, v := range wrapped {
				lines = append(lines, line{text: v, size: size, bold: bold})
			}
		}
		if len(lines) > 0 {
			lines[len(lines)-1].gap += gap
		}
		return nil
	}
	if err := appendText(s.Title, 17, true, 5); err != nil {
		return out, err
	}
	if err := appendText(strings.ToUpper(s.Kind)+" | As of "+s.ReportingDate, BodySize, false, 10); err != nil {
		return out, err
	}
	if s.StaleReason != "" {
		if err := appendText("Issued using a prior saved state: "+s.StaleReason, BodySize, false, 7); err != nil {
			return out, err
		}
	}
	// A consultant RFP prints only what the consultant prices from. Its
	// citations stay in the issue record; every block is calculated from the
	// stored rules, so printing them would only repeat that.
	quiet := s.SourceRevisions.TemplateID == "tpl.rfp-consultant"
	// RFP schedules are set a little tighter; the type size is unchanged.
	leading := 1.3
	if quiet {
		leading = 1.2
	}
	for _, section := range s.Sections {
		if section.Title != "" {
			if err := appendText(section.Title, 11, true, 4); err != nil {
				return out, err
			}
		}
		for _, b := range section.Blocks {
			document := b.Table != nil && b.Table.Display == "document"
			text := b.Text
			// A block with no label is the report's own wording, not a labelled
			// saved value.
			if b.Label != "" {
				label := b.Label
				if b.Provisional {
					label += " [PROVISIONAL]"
				}
				if b.Origin == "assumption" {
					label += " [MATERIAL ASSUMPTION]"
				}
				text = label + ": " + b.Text
			}
			if len(b.CitationIDs) > 0 && !quiet {
				text = strings.TrimSpace(text + " [" + strings.Join(b.CitationIDs, ", ") + "]")
			}
			if strings.TrimSpace(text) != "" {
				if err := appendText(text, BodySize, false, 4); err != nil {
					return out, err
				}
			}
			if b.Table != nil {
				maxColumns := 6
				if document {
					maxColumns = 8
				} else {
					heading := "Saved source comparison"
					if b.Edited {
						heading += " — independent of edited wording"
					}
					if err := appendText(heading, BodySize, true, 3); err != nil {
						return out, err
					}
				}
				table := b.Table
				if len(table.Columns) < 1 || len(table.Columns) > maxColumns || len(table.Rows) > 500 {
					return out, fmt.Errorf("%w: invalid table dimensions", ErrContent)
				}
				widths := make([]float64, len(table.Columns))
				for i := range widths {
					widths[i] = width / float64(len(table.Columns))
				}
				pad := 10.0
				if document {
					widths, pad = columnWidths(*table, width), 4
					if table.Columns[0] == "Scope of services" || table.Columns[0] == "Stage / position" {
						pad = 2
					}
				}
				tableRows := append([][]string{table.Columns}, table.Rows...)
				for rowIndex, row := range tableRows {
					if len(row) != len(table.Columns) {
						return out, fmt.Errorf("%w: invalid table row", ErrContent)
					}
					cells := make([][]string, len(row))
					maxLines := 1
					for col, text := range row {
						// Reuse the same glyph and control validation as prose.
						start := len(lines)
						if err := appendText(text, BodySize, rowIndex == 0, 0); err != nil {
							return out, err
						}
						lines = lines[:start]
						for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
							// A blank cell (a price the consultant fills in) is one empty line.
							wrapped := []string{""}
							if paragraph != "" {
								var err error
								if wrapped, err = gp.SplitTextWithWordWrap(paragraph, widths[col]-10); err != nil {
									return out, err
								}
							}
							cells[col] = append(cells[col], wrapped...)
						}
						if len(cells[col]) > maxLines {
							maxLines = len(cells[col])
						}
					}
					lines = append(lines, line{size: BodySize, bold: rowIndex == 0, cells: cells, height: float64(maxLines)*BodySize*leading + pad, widths: widths, pad: pad})
				}
				lines[len(lines)-1].height += 7
				lines[len(lines)-1].gap = 7
			}
		}
	}
	if len(s.References) > 0 && !quiet {
		if err := appendText("References — E evidence / U user / C calculation / A assumption", 10, true, 5); err != nil {
			return out, err
		}
		for _, ref := range s.References {
			if err := appendText(ref.ID+": "+ref.Basis.Text, MinimumSize, false, 3); err != nil {
				return out, err
			}
		}
	}
	height := gopdf.PageSizeA4.H - margin - 24
	pages, y := 1, margin
	for _, l := range lines {
		h := l.height
		if h == 0 {
			h = l.size*leading + l.gap
		}
		if h > height-margin {
			return Result{Pages: 3, MinimumFont: MinimumSize}, ErrOverflow
		}
		if y+h > height {
			pages++
			y = margin
		}
		y += h
	}
	length := float64(pages-1) + (y-margin)/(height-margin)
	if limit, _ := PageLimit(s); gate && pages > limit {
		return Result{Pages: pages, MinimumFont: MinimumSize, Length: length}, ErrOverflow
	}
	page, y := 0, margin
	newPage := func() error {
		gp.AddPage()
		page++
		y = margin
		gp.SetTextColor(25, 31, 40)
		gp.SetXY(margin, gopdf.PageSizeA4.H-margin)
		if err := gp.SetFont("body", "", MinimumSize); err != nil {
			return err
		}
		return gp.Cell(nil, fmt.Sprintf("SiteWise | %s | Page %d of %d", s.ReportingDate, page, pages))
	}
	if err := newPage(); err != nil {
		return out, err
	}
	for _, l := range lines {
		h := l.height
		if h == 0 {
			h = l.size*leading + l.gap
		}
		if y+h > height {
			if err := newPage(); err != nil {
				return out, err
			}
		}
		family := "body"
		if l.bold {
			family = "heading"
		}
		if err := gp.SetFont(family, "", l.size); err != nil {
			return out, err
		}
		if len(l.cells) > 0 {
			gp.SetStrokeColor(180, 188, 198)
			x := margin
			for col, cell := range l.cells {
				cellWidth := width / float64(len(l.cells))
				if len(l.widths) == len(l.cells) {
					cellWidth = l.widths[col]
				}
				if l.bold {
					gp.SetFillColor(237, 241, 245)
					gp.RectFromUpperLeftWithStyle(x, y, cellWidth, h-l.gap, "F")
				}
				gp.RectFromUpperLeft(x, y, cellWidth, h-l.gap)
				top := 5.0
				if l.pad > 0 {
					top = l.pad / 2
				}
				for row, text := range cell {
					if err := drawCitations(gp, x+5, y+top+float64(row)*l.size*leading, text); err != nil {
						return out, err
					}
				}
				x += cellWidth
			}
		} else if err := drawCitations(gp, margin, y, l.text); err != nil {
			return out, err
		}
		y += h
	}
	var buf bytes.Buffer
	if _, err := gp.WriteTo(&buf); err != nil {
		return out, err
	}
	return Result{PDF: buf.Bytes(), Pages: pages, MinimumFont: MinimumSize, Length: length}, nil
}

// columnWidths shares a document table's width by how much each column
// holds, so a long deliverable is not squeezed into an equal share beside a
// one-letter position. Every column keeps room for a short word.
func columnWidths(t reports.Table, width float64) []float64 {
	if len(t.Columns) == 2 && t.Columns[0] == "Item" && t.Columns[1] == "Allowance" {
		return []float64{width * .25, width * .75}
	}
	if len(t.Columns) == 5 && t.Columns[0] == "Scope of services" {
		return []float64{width * .84, width * .04, width * .04, width * .04, width * .04}
	}
	// The consultant stage/fee schedule needs writable fee columns. Text
	// length alone would squeeze an empty pricing column to a few letters.
	if len(t.Columns) == 5 && t.Columns[0] == "Stage / position" && t.Columns[1] == "Deliverable at sign-off" {
		return []float64{width * .18, width * .51, width * .13, width * .09, width * .09}
	}
	weights := make([]float64, len(t.Columns))
	total := 0.0
	for col := range t.Columns {
		longest := utf8.RuneCountInString(t.Columns[col])
		sum, count := 0, 0
		for _, row := range t.Rows {
			if col < len(row) {
				n := utf8.RuneCountInString(row[col])
				sum += n
				count++
				if n > longest {
					longest = n
				}
			}
		}
		average := float64(sum) / math.Max(1, float64(count))
		// Blend the longest and the typical cell, capped so one long cell
		// cannot take the table.
		// A header is one unbroken word or two, so its column keeps room for it.
		header := float64(utf8.RuneCountInString(t.Columns[col]))
		weights[col] = math.Max(math.Max(7, header*1.4+2), math.Min(90, (float64(longest)+2*average)/3))
		total += weights[col]
	}
	out := make([]float64, len(weights))
	for i, w := range weights {
		out[i] = width * w / total
	}
	return out
}

// Citation letters remain explicit in monochrome printing. Colour reinforces
// provenance without becoming the only way to distinguish the four bases.
var citationToken = regexp.MustCompile(`\b[EUCA][0-9]+\b`)
var citationColours = map[byte][3]uint8{'E': {25, 78, 156}, 'U': {0, 103, 103}, 'C': {110, 55, 154}, 'A': {143, 82, 0}}

func drawCitations(gp *gopdf.GoPdf, x, y float64, text string) error {
	offset := 0
	draw := func(value string, colour [3]uint8) error {
		gp.SetTextColor(colour[0], colour[1], colour[2])
		gp.SetXY(x, y)
		if err := gp.Cell(nil, value); err != nil {
			return err
		}
		width, err := gp.MeasureTextWidth(value)
		x += width
		return err
	}
	for _, match := range citationToken.FindAllStringIndex(text, -1) {
		if err := draw(text[offset:match[0]], [3]uint8{25, 31, 40}); err != nil {
			return err
		}
		if err := draw(text[match[0]:match[1]], citationColours[text[match[0]]]); err != nil {
			return err
		}
		offset = match[1]
	}
	return draw(text[offset:], [3]uint8{25, 31, 40})
}
