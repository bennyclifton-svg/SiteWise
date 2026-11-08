package export

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

// The screen's typeface (web/src/main.tsx loads Instrument Sans), unpacked
// from its WOFF files with tools/woff2ttf. OFL-1.1: fonts/OFL.txt.
//
//go:embed fonts/InstrumentSans-Regular.ttf
var fontRegular []byte

//go:embed fonts/InstrumentSans-SemiBold.ttf
var fontSemiBold []byte

// FontName is the family the Word file names and embeds.
const FontName = "Instrument Sans"

const (
	pageW, pageH = 595.28, 841.89 // A4
	margin       = 42.52          // 15 mm
	bodySize     = 10.0
	smallSize    = 8.0
	leading      = 1.4
	padX, padY   = 6.0, 3.0
	bandHeight   = 22.0
	headSpacing  = 0.6 // letter spacing of band and head titles
	footerGap    = 24.0
)

// ErrContent is a draft the exporter cannot lay out.
var ErrContent = errors.New("report content cannot be exported")

// Faces: the screen's for Latin text; Go's for a character it lacks.
const (
	faceBody     = "body"
	faceBold     = "bold"
	faceFallback = "fallback"
	faceFallBold = "fallbold"
)

type pdfWriter struct {
	gp      *gopdf.GoPdf
	primary *sfnt.Font
	width   float64 // content width
	pages   [][]func() error
	y       float64
}

// PDF writes the document as an A4 PDF in the screen's style.
func PDF(d Doc) ([]byte, error) {
	w, err := newPDFWriter()
	if err != nil {
		return nil, err
	}
	if err := w.layout(d); err != nil {
		return nil, err
	}
	for i, ops := range w.pages {
		w.gp.AddPage()
		for _, op := range ops {
			if err := op(); err != nil {
				return nil, err
			}
		}
		w.gp.SetTextColor(colourMuted.r, colourMuted.g, colourMuted.b)
		if err := w.gp.SetFont(faceBody, "", smallSize); err != nil {
			return nil, err
		}
		w.gp.SetXY(margin, pageH-margin+6)
		footer := fmt.Sprintf("SiteWise | %s | Page %d of %d", d.Date, i+1, len(w.pages))
		if err := w.gp.Cell(nil, footer); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	if _, err := w.gp.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func newPDFWriter() (*pdfWriter, error) {
	gp := &gopdf.GoPdf{}
	gp.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	for face, data := range map[string][]byte{faceBody: fontRegular, faceBold: fontSemiBold, faceFallback: goregular.TTF, faceFallBold: gobold.TTF} {
		if err := gp.AddTTFFontData(face, data); err != nil {
			return nil, fmt.Errorf("font %s: %w", face, err)
		}
	}
	primary, err := sfnt.Parse(fontRegular)
	if err != nil {
		return nil, err
	}
	return &pdfWriter{gp: gp, primary: primary, width: pageW - 2*margin}, nil
}

// --- pages ---------------------------------------------------------------

func (w *pdfWriter) bottom() float64 { return pageH - margin - footerGap }

func (w *pdfWriter) newPage() {
	w.pages = append(w.pages, nil)
	w.y = margin
}

func (w *pdfWriter) draw(op func() error) {
	w.pages[len(w.pages)-1] = append(w.pages[len(w.pages)-1], op)
}

// room starts a new page unless h points fit below the cursor.
func (w *pdfWriter) room(h float64) {
	if w.y+h > w.bottom() && w.y > margin {
		w.newPage()
	}
}

func (w *pdfWriter) layout(d Doc) error {
	w.newPage()
	if err := w.paragraph([]Run{{Text: d.Title, Bold: true}}, 16, 2); err != nil {
		return err
	}
	if d.Subtitle != "" {
		if err := w.paragraph([]Run{{Text: d.Subtitle, Muted: true}}, 9, 0); err != nil {
			return err
		}
	}
	w.y += 10
	for _, s := range d.Sections {
		if s.Title != "" {
			w.band(s.Title)
		}
		for _, b := range s.Blocks {
			if err := w.block(b); err != nil {
				return err
			}
		}
		w.y += 8
	}
	return nil
}

// band is a section's blue head: uppercase title on the tint, a strong rule
// beneath. A band never ends a page.
func (w *pdfWriter) band(title string) {
	w.room(bandHeight + 3*bodySize*leading)
	y := w.y
	w.draw(func() error {
		w.fill(colourBand, margin, y, w.width, bandHeight)
		w.rule(colourStrong, 0.75, margin, y+bandHeight, margin+w.width)
		_, err := w.text(strings.ToUpper(title), margin+padX+2, y+(bandHeight-smallSize*1.2)/2, smallSize, true, colourHead, headSpacing)
		return err
	})
	w.y += bandHeight + 6
}

func (w *pdfWriter) block(b Block) error {
	runs := []Run{}
	if b.Label != "" {
		runs = append(runs, Run{Text: b.Label + ": ", Bold: true})
	}
	if b.Text != "" {
		runs = append(runs, Run{Text: b.Text})
	}
	if !b.TextAfter && len(runs) > 0 {
		if err := w.paragraph(runs, bodySize, 6); err != nil {
			return err
		}
	}
	if b.Table != nil {
		if err := w.table(*b.Table); err != nil {
			return err
		}
	}
	if b.TextAfter && len(runs) > 0 {
		w.y += 4
		if err := w.paragraph(runs, bodySize, 6); err != nil {
			return err
		}
	}
	return nil
}

func (w *pdfWriter) paragraph(runs []Run, size, gap float64) error {
	lines, err := w.wrap(runs, size, w.width)
	if err != nil {
		return err
	}
	lh := size * leading
	for _, line := range lines {
		w.room(lh)
		y := w.y
		w.draw(func() error { return w.line(line, margin, y, size, Left, w.width) })
		w.y += lh
	}
	w.y += gap
	return nil
}

// --- tables --------------------------------------------------------------

type placed struct {
	cell   Cell
	col    int // first column
	cols   int
	rows   int
	lines  [][]Run
	height float64 // content height with padding
}

func (w *pdfWriter) table(t Table) error {
	widths := columnWidths(t.Columns, w.width)
	n := len(widths)
	place := func(cells []Cell, occupied []int) ([]placed, error) {
		out := []placed{}
		col := 0
		for _, c := range cells {
			for col < n && occupied[col] > 0 {
				col++
			}
			cols := span(c.ColSpan)
			if col+cols > n {
				return nil, fmt.Errorf("%w: a table row is wider than its columns", ErrContent)
			}
			width := 0.0
			for _, cw := range widths[col : col+cols] {
				width += cw
			}
			p := placed{cell: c, col: col, cols: cols, rows: span(c.RowSpan)}
			size := bodySize
			if c.Head {
				size = smallSize
			}
			var err error
			if p.lines, err = w.wrap(c.Runs, size, width-2*padX); err != nil {
				return nil, err
			}
			p.height = float64(max(len(p.lines), 1))*size*leading + 2*padY
			out = append(out, p)
			col += cols
		}
		return out, nil
	}
	head, err := place(t.Head, make([]int, n))
	if err != nil {
		return err
	}
	headH := rowHeight
	for _, p := range head {
		headH = max(headH, p.height)
	}
	// Rows: place cells around those spanning down from above, then size
	// each row so every cell fits; a spanning cell's extra height goes on
	// its last row.
	occupied := make([]int, n)
	rows := make([][]placed, len(t.Rows))
	heights := make([]float64, len(t.Rows))
	for i, r := range t.Rows {
		if rows[i], err = place(r.Cells, occupied); err != nil {
			return err
		}
		for c := range occupied {
			if occupied[c] > 0 {
				occupied[c]--
			}
		}
		heights[i] = max(r.MinHeight, bodySize*leading+2*padY)
		for _, p := range rows[i] {
			if p.rows > 1 {
				for c := p.col; c < p.col+p.cols; c++ {
					occupied[c] = p.rows - 1
				}
				continue
			}
			heights[i] = max(heights[i], p.height)
		}
	}
	for i := range rows {
		for _, p := range rows[i] {
			if p.rows > 1 && i+p.rows <= len(rows) {
				total := 0.0
				for _, h := range heights[i : i+p.rows] {
					total += h
				}
				if p.height > total {
					heights[i+p.rows-1] += p.height - total
				}
			}
		}
	}
	xs := make([]float64, n+1)
	xs[0] = margin
	for i, cw := range widths {
		xs[i+1] = xs[i] + cw
	}
	drawHead := func() {
		y := w.y
		w.draw(func() error {
			w.fill(colourBand, margin, y, w.width, headH)
			for _, p := range head {
				x, cw := xs[p.col], xs[p.col+p.cols]-xs[p.col]
				w.rect(colourLine, x, y, cw, headH)
				if err := w.cellText(p, x, y, cw, headH, false); err != nil {
					return err
				}
			}
			w.rule(colourStrong, 0.75, margin, y+headH, margin+w.width)
			return nil
		})
		w.y += headH
	}
	// Keep a head with at least its first row.
	first := 0.0
	if len(heights) > 0 {
		first = heights[0]
	}
	w.room(headH + first)
	drawHead()
	// open spans: cells drawn on a row whose span continues down the page.
	type openSpan struct {
		p      placed
		top    float64
		last   int // index of the span's last row
		drawn  bool
		strong bool
	}
	var spans []*openSpan
	flush := func(bottom float64) {
		for _, s := range spans {
			s := s
			top := s.top
			p := s.p
			strong := s.strong
			again := s.drawn
			w.draw(func() error {
				x, cw := xs[p.col], xs[p.col+p.cols]-xs[p.col]
				w.rect(colourLine, x, top, cw, bottom-top)
				if strong {
					w.rule(colourStrong, 0.75, x, top, x+cw)
				}
				if again {
					// A stage carried over a page shows its name again.
					name := p.lines[:min(1, len(p.lines))]
					if len(p.cell.Runs) > 0 {
						if lines, err := w.wrap(p.cell.Runs[:1], bodySize, cw-2*padX); err == nil {
							name = lines
						}
					}
					return w.cellText(placed{cell: Cell{Align: p.cell.Align}, lines: name}, x, top, cw, bottom-top, true)
				}
				return w.cellText(p, x, top, cw, bottom-top, true)
			})
			s.drawn = true
		}
	}
	for i := range rows {
		h := heights[i]
		// A row starting a span (a stage) needs room for the span's own
		// text too, so a stage name never spills past a page break.
		need := h
		for _, p := range rows[i] {
			if p.rows > 1 {
				need = max(need, p.height)
			}
		}
		if w.y+need > w.bottom() {
			flush(w.y)
			w.newPage()
			drawHead()
			for _, s := range spans {
				s.top = w.y
				s.strong = false
				// The stage's name repeats here: its remaining rows grow, if
				// need be, so the name fits beside them.
				if len(s.p.cell.Runs) > 0 {
					cw := xs[s.p.col+s.p.cols] - xs[s.p.col]
					if name, err := w.wrap(s.p.cell.Runs[:1], bodySize, cw-2*padX); err == nil {
						want := float64(len(name))*bodySize*leading + 2*padY
						have := 0.0
						for _, rh := range heights[i : s.last+1] {
							have += rh
						}
						if want > have {
							heights[s.last] += want - have
						}
					}
				}
			}
			h = heights[i]
		}
		y := w.y
		strong := t.Rows[i].Strong
		for _, p := range rows[i] {
			if p.rows > 1 {
				spans = append(spans, &openSpan{p: p, top: y, last: i + p.rows - 1, strong: strong})
				continue
			}
			p := p
			w.draw(func() error {
				x, cw := xs[p.col], xs[p.col+p.cols]-xs[p.col]
				w.rect(colourLine, x, y, cw, h)
				return w.cellText(p, x, y, cw, h, false)
			})
		}
		if strong {
			// The strong rule runs across the cells this row starts.
			for _, p := range rows[i] {
				p := p
				w.draw(func() error {
					w.rule(colourStrong, 0.75, xs[p.col], y, xs[p.col+p.cols])
					return nil
				})
			}
		}
		w.y += h
		// Close spans that end on this row.
		kept := spans[:0]
		ending := []*openSpan{}
		for _, s := range spans {
			if s.last == i {
				ending = append(ending, s)
			} else {
				kept = append(kept, s)
			}
		}
		spans = kept
		if len(ending) > 0 {
			saved := spans
			spans = ending
			flush(w.y)
			spans = saved
		}
	}
	flush(w.y)
	w.y += 8
	return nil
}

// columnWidths gives fixed columns their points and shares the rest.
func columnWidths(cols []Column, width float64) []float64 {
	fixed, shares := 0.0, 0.0
	for _, c := range cols {
		fixed += c.Fixed
		if c.Fixed == 0 {
			shares += max(c.Share, 1)
		}
	}
	rest := max(width-fixed, 0)
	out := make([]float64, len(cols))
	for i, c := range cols {
		if c.Fixed > 0 {
			out[i] = c.Fixed
		} else {
			out[i] = rest * max(c.Share, 1) / shares
		}
	}
	return out
}

// cellText draws a cell's runs or tick: spanning cells (a stage, its fee)
// read from the top, others centred in the row, as on screen.
func (w *pdfWriter) cellText(p placed, x, y, cw, h float64, top bool) error {
	if p.cell.Tick != NoTick {
		w.tick(x+cw/2, y+h/2, p.cell.Tick == TickOn)
		return nil
	}
	if p.cell.Head {
		lh := smallSize * leading
		ty := y + (h-float64(len(p.lines))*lh)/2
		for i, line := range p.lines {
			s := runText(line)
			x0 := x + padX
			if p.cell.Align == Center {
				wd, err := w.measure(s, true, smallSize, headSpacing)
				if err != nil {
					return err
				}
				x0 = x + (cw-wd)/2
			}
			if _, err := w.text(s, x0, ty+float64(i)*lh, smallSize, true, colourHead, headSpacing); err != nil {
				return err
			}
		}
		return nil
	}
	lh := bodySize * leading
	textH := float64(len(p.lines)) * lh
	ty := y + (h-textH)/2
	if top {
		ty = y + padY
	}
	for i, line := range p.lines {
		if err := w.line(line, x+padX, ty+float64(i)*lh, bodySize, p.cell.Align, cw-2*padX); err != nil {
			return err
		}
	}
	return nil
}

// tick draws a checkbox as the browser shows it: a filled box with a white
// tick, or an empty outlined box.
func (w *pdfWriter) tick(cx, cy float64, on bool) {
	const s = 9.0
	x0, y0 := cx-s/2, cy-s/2
	if on {
		w.gp.SetFillColor(colourTick.r, colourTick.g, colourTick.b)
		w.gp.SetStrokeColor(colourTick.r, colourTick.g, colourTick.b)
		w.gp.SetLineWidth(0.8)
		_ = w.gp.Rectangle(x0, y0, x0+s, y0+s, "FD", 1.5, 6)
		w.gp.SetStrokeColor(255, 255, 255)
		w.gp.SetLineWidth(1.3)
		w.gp.Line(x0+2, cy+0.2, x0+3.8, cy+2.1)
		w.gp.Line(x0+3.8, cy+2.1, x0+7.1, cy-2.2)
		return
	}
	w.gp.SetStrokeColor(colourStrong.r, colourStrong.g, colourStrong.b)
	w.gp.SetLineWidth(0.8)
	_ = w.gp.Rectangle(x0, y0, x0+s, y0+s, "D", 1.5, 6)
}

// --- primitives ----------------------------------------------------------

func (w *pdfWriter) fill(c rgb, x, y, wd, h float64) {
	w.gp.SetFillColor(c.r, c.g, c.b)
	w.gp.RectFromUpperLeftWithStyle(x, y, wd, h, "F")
}

func (w *pdfWriter) rect(c rgb, x, y, wd, h float64) {
	w.gp.SetStrokeColor(c.r, c.g, c.b)
	w.gp.SetLineWidth(0.5)
	w.gp.RectFromUpperLeftWithStyle(x, y, wd, h, "D")
}

func (w *pdfWriter) rule(c rgb, width, x1, y, x2 float64) {
	w.gp.SetStrokeColor(c.r, c.g, c.b)
	w.gp.SetLineWidth(width)
	w.gp.Line(x1, y, x2, y)
}

// text draws one string in one style, with letter spacing for band and
// head titles. A character the screen's face lacks is drawn in Go's face,
// and only that character. It returns the width drawn.
func (w *pdfWriter) text(s string, x, y, size float64, bold bool, c rgb, spacing float64) (float64, error) {
	w.gp.SetTextColor(c.r, c.g, c.b)
	if err := w.gp.SetCharSpacing(spacing); err != nil {
		return 0, err
	}
	defer w.gp.SetCharSpacing(0)
	start := x
	for _, seg := range w.segments(s, bold) {
		if err := w.gp.SetFont(seg.face, "", size); err != nil {
			return 0, err
		}
		w.gp.SetXY(x, y)
		if err := w.gp.Cell(nil, seg.text); err != nil {
			return 0, err
		}
		wd, err := w.gp.MeasureTextWidth(seg.text)
		if err != nil {
			return 0, err
		}
		x += wd + spacing*float64(len([]rune(seg.text)))
	}
	return x - start, nil
}

// measure is the width text() would draw, without drawing.
func (w *pdfWriter) measure(s string, bold bool, size, spacing float64) (float64, error) {
	total := 0.0
	for _, seg := range w.segments(s, bold) {
		if err := w.gp.SetFont(seg.face, "", size); err != nil {
			return 0, err
		}
		wd, err := w.gp.MeasureTextWidth(seg.text)
		if err != nil {
			return 0, err
		}
		total += wd + spacing*float64(len([]rune(seg.text)))
	}
	return total, nil
}

// line draws one wrapped line of runs.
func (w *pdfWriter) line(runs []Run, x, y, size float64, align Align, width float64) error {
	if align == Center {
		total, err := w.measureRuns(runs, size)
		if err != nil {
			return err
		}
		x += (width - total) / 2
	}
	for _, r := range runs {
		if r.Text == "" {
			continue
		}
		c := colourText
		if r.Muted {
			c = colourMuted
		}
		wd, err := w.text(r.Text, x, y, size, r.Bold, c, 0)
		if err != nil {
			return err
		}
		x += wd
	}
	return nil
}

func (w *pdfWriter) measureRuns(runs []Run, size float64) (float64, error) {
	total := 0.0
	for _, r := range runs {
		if r.Text == "" {
			continue
		}
		wd, err := w.measure(r.Text, r.Bold, size, 0)
		if err != nil {
			return 0, err
		}
		total += wd
	}
	return total, nil
}

type segment struct{ face, text string }

// segments splits text into runs of one face: the screen's where it has
// the character, Go's where it does not (the latin subset lacks "²").
func (w *pdfWriter) segments(s string, bold bool) []segment {
	main, fallback := faceBody, faceFallback
	if bold {
		main, fallback = faceBold, faceFallBold
	}
	var buf sfnt.Buffer
	var out []segment
	for _, r := range s {
		face := main
		if r != ' ' {
			if g, err := w.primary.GlyphIndex(&buf, r); err != nil || g == 0 {
				face = fallback
			}
		}
		if n := len(out); n > 0 && out[n-1].face == face {
			out[n-1].text += string(r)
		} else {
			out = append(out, segment{face, string(r)})
		}
	}
	return out
}

// wrap breaks styled runs into lines no wider than width. "\n" breaks a
// line; a word longer than the line is split by characters.
func (w *pdfWriter) wrap(runs []Run, size, width float64) ([][]Run, error) {
	var lines [][]Run
	var current []Run
	used := 0.0
	push := func(r Run) {
		if n := len(current); n > 0 && current[n-1].Bold == r.Bold && current[n-1].Muted == r.Muted {
			current[n-1].Text += r.Text
		} else {
			current = append(current, r)
		}
	}
	end := func() {
		if n := len(current); n > 0 {
			current[n-1].Text = strings.TrimRight(current[n-1].Text, " ")
		}
		lines = append(lines, current)
		current, used = nil, 0
	}
	measure := func(r Run) (float64, error) { return w.measure(r.Text, r.Bold, size, 0) }
	for _, r := range runs {
		for _, c := range r.Text {
			if c != '\n' && c != '\t' && unicode.IsControl(c) {
				return nil, fmt.Errorf("%w: remove control character U+%04X", ErrContent, c)
			}
		}
		paragraphs := strings.Split(strings.ReplaceAll(r.Text, "\t", " "), "\n")
		for pi, para := range paragraphs {
			if pi > 0 {
				end()
			}
			fields := strings.SplitAfter(para, " ")
			for _, f := range fields {
				if f == "" {
					continue
				}
				piece := Run{Text: f, Bold: r.Bold, Muted: r.Muted}
				wd, err := measure(piece)
				if err != nil {
					return nil, err
				}
				trimmed := Run{Text: strings.TrimRight(f, " "), Bold: r.Bold, Muted: r.Muted}
				tw, err := measure(trimmed)
				if err != nil {
					return nil, err
				}
				if used > 0 && used+tw > width {
					end()
				}
				if tw > width {
					// One word wider than the line: split it.
					for _, ch := range trimmed.Text {
						part := Run{Text: string(ch), Bold: r.Bold, Muted: r.Muted}
						cw, err := measure(part)
						if err != nil {
							return nil, err
						}
						if used > 0 && used+cw > width {
							end()
						}
						push(part)
						used += cw
					}
					if strings.HasSuffix(f, " ") {
						push(Run{Text: " ", Bold: r.Bold, Muted: r.Muted})
					}
					continue
				}
				push(piece)
				used += wd
			}
		}
	}
	if len(current) > 0 || len(lines) == 0 {
		end()
	}
	return lines, nil
}
