// Package render produces offline, embedded-font PDFs without external services.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

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
var ErrOverflow = errors.New("report exceeds two pages; shorten the report or identify an attachment")
var ErrContent = errors.New("report content cannot be exported")

type Result struct {
	PDF         []byte
	Pages       int
	MinimumFont float64
}
type line struct {
	text   string
	size   float64
	bold   bool
	gap    float64
	cells  [][]string
	height float64
}

// PDF uses one typography specification for measurement and drawing. It never
// shrinks text or drops blocks to make the page-count gate pass.
func PDF(s reports.IssueSnapshot) (Result, error) {
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
			wrapped, err := gp.SplitTextWithWordWrap(paragraph, width)
			if err != nil {
				return err
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
	for _, section := range s.Sections {
		if err := appendText(section.Title, 11, true, 4); err != nil {
			return out, err
		}
		for _, b := range section.Blocks {
			label := b.Label
			if b.Provisional {
				label += " [PROVISIONAL]"
			}
			if b.Origin == "assumption" {
				label += " [MATERIAL ASSUMPTION]"
			}
			text := label + ": " + b.Text
			if len(b.CitationIDs) > 0 {
				text += " [" + strings.Join(b.CitationIDs, ", ") + "]"
			}
			if err := appendText(text, BodySize, false, 4); err != nil {
				return out, err
			}
			if b.Table != nil {
				heading := "Saved source comparison"
				if b.Edited {
					heading += " — independent of edited wording"
				}
				if err := appendText(heading, BodySize, true, 3); err != nil {
					return out, err
				}
				table := b.Table
				if len(table.Columns) < 1 || len(table.Columns) > 6 || len(table.Rows) > 500 {
					return out, fmt.Errorf("%w: invalid table dimensions", ErrContent)
				}
				cellWidth := width / float64(len(table.Columns))
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
							wrapped, err := gp.SplitTextWithWordWrap(paragraph, cellWidth-10)
							if err != nil {
								return out, err
							}
							cells[col] = append(cells[col], wrapped...)
						}
						if len(cells[col]) > maxLines {
							maxLines = len(cells[col])
						}
					}
					lines = append(lines, line{size: BodySize, bold: rowIndex == 0, cells: cells, height: float64(maxLines)*BodySize*1.3 + 10})
				}
				lines[len(lines)-1].height += 7
				lines[len(lines)-1].gap = 7
			}
		}
	}
	if len(s.References) > 0 {
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
			h = l.size*1.3 + l.gap
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
	if pages > 2 {
		return Result{Pages: pages, MinimumFont: MinimumSize}, ErrOverflow
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
			h = l.size*1.3 + l.gap
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
			cellWidth := width / float64(len(l.cells))
			gp.SetStrokeColor(180, 188, 198)
			for col, cell := range l.cells {
				x := margin + float64(col)*cellWidth
				if l.bold {
					gp.SetFillColor(237, 241, 245)
					gp.RectFromUpperLeftWithStyle(x, y, cellWidth, h-l.gap, "F")
				}
				gp.RectFromUpperLeft(x, y, cellWidth, h-l.gap)
				for row, text := range cell {
					if err := drawCitations(gp, x+5, y+5+float64(row)*l.size*1.3, text); err != nil {
						return out, err
					}
				}
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
	return Result{PDF: buf.Bytes(), Pages: pages, MinimumFont: MinimumSize}, nil
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
