package identity

import (
	"context"
	"io"
	"math"
	"regexp"
	"strings"
	"sync"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

// PDFium binding is the embedded WebAssembly build of
// github.com/klippa-app/go-pdfium v1.18.0 (MIT). PDFium and Wazero are Apache-2.0.
// The module is compiled into this process, so intake stays one Go binary
// and does not need CGO or a sidecar.
//
// Measured 2026-09-29 on Windows 10 amd64, AMD Ryzen 5 3600, Go 1.27.1.
// Warm, which compiles the module and opens one worker, took 1.54 s and
// 104 MiB of system memory (24 MiB on the Go heap). Warm extraction of
// testdata/identity/identity-page.pdf over 20 samples: p50 8.0 ms, p90 9.1 ms.
// Five benchmark runs sat near 8.3 ms/op. The gate is 80/250 ms, so the WASM
// packaging stays.
// Call Warm at process start; that cold compile is not part of an intake.
//
// An empty FSConfig is passed so the default init cannot mount the host drive
// into the sandbox. Documents are opened from bytes.

var (
	pdfOnce sync.Once
	pdfPool pdfium.Pool
	pdfInit error
)

// Warm compiles the embedded PDFium module and opens one worker.
// Later Extract calls reuse that worker.
func Warm(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := pdfiumPool()
	return err
}

func pdfiumPool() (pdfium.Pool, error) {
	pdfOnce.Do(func() {
		// Reuse one worker. Compiling the module is the cold cost; a warm
		// extract must not instantiate PDFium again. One worker matches the
		// single-file identity path and caps WASM memory growth.
		pdfPool, pdfInit = webassembly.Init(webassembly.Config{
			MinIdle:      1,
			MaxIdle:      1,
			MaxTotal:     1,
			ReuseWorkers: true,
			FSConfig:     wazero.NewFSConfig(),
			Stdout:       io.Discard,
			Stderr:       io.Discard,
		})
	})
	if pdfInit != nil {
		return nil, pdfInit
	}
	return pdfPool, nil
}

func extractPDF(ctx context.Context, r io.ReaderAt, size int64, limits Limits) (out Text, err error) {
	request := &requests.OpenDocument{}
	if size > limits.MaxBytes {
		// Large packs often contain image streams on later pages. PDFium can seek
		// past them; the read budget still bounds all bytes requested for identity.
		if size > math.MaxUint32 {
			return Text{}, ErrTooLarge
		}
		header := make([]byte, 5)
		if _, err := r.ReadAt(header, 0); err != nil || string(header) != "%PDF-" {
			return Text{}, ErrMalformed
		}
		source := &pdfBudgetReader{ctx: ctx, reader: io.NewSectionReader(r, 0, size), remaining: limits.MaxBytes - 5}
		request.FileReader = source
		request.FileReaderSize = size
		defer func() {
			if source.err != nil {
				out = Text{}
				err = source.err
			}
		}()
	} else {
		body, err := readAtMost(r, size, limits.MaxBytes)
		if err != nil {
			return Text{}, err
		}
		if len(body) < 5 || string(body[:5]) != "%PDF-" {
			return Text{}, ErrMalformed
		}
		request.File = &body
	}
	if err := ctx.Err(); err != nil {
		return Text{}, err
	}
	pool, err := pdfiumPool()
	if err != nil {
		return Text{}, err
	}
	inst, err := pool.GetInstanceWithContext(ctx)
	if err != nil {
		return Text{}, err
	}
	defer inst.Close()

	doc, err := inst.OpenDocument(request)
	if err != nil {
		return Text{}, malformed(err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	count, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return Text{}, malformed(err)
	}
	pages := count.PageCount
	if pages > limits.MaxPages {
		pages = limits.MaxPages
	}
	var runs []Run
	for page := 0; page < pages; page++ {
		if err := ctx.Err(); err != nil {
			return Text{}, err
		}
		pageRuns, err := pdfPageRuns(ctx, inst, doc.Document, page, limits.MaxRuns-len(runs))
		if err != nil {
			return Text{}, err
		}
		runs = append(runs, pageRuns...)
		if len(runs) >= limits.MaxRuns {
			runs = runs[:limits.MaxRuns]
			break
		}
	}
	if limits.ReadControlPage && pages == 1 && count.PageCount > 1 && len(runs) < limits.MaxRuns && probeControlPage(runs) {
		control, readErr := pdfPageRuns(ctx, inst, doc.Document, 1, limits.MaxRuns-len(runs))
		if readErr != nil {
			return Text{}, readErr
		}
		metadata := reportMetadata(control)
		repeat := repeatedReportCover(runs, control)
		toc := nextPageControlReference(control)
		if len(metadata) > 0 && !controlPage(control) {
			runs = append(runs, metadata...)
		}
		if !controlPage(control) && count.PageCount > 2 && (repeat || len(metadata) > 0 || toc) {
			// Some reports repeat the cover with publisher details before control.
			// Probe only one further page, and retain only explicit control text.
			control, readErr = pdfPageRuns(ctx, inst, doc.Document, 2, limits.MaxRuns-len(runs))
			if readErr != nil {
				return Text{}, readErr
			}
			if toc {
				control = boundedControlSection(control)
			}
		}
		if controlPage(control) {
			runs = append(runs, control...)
		}
	}
	return Text{Runs: runs, PageCount: count.PageCount}, nil
}

func pdfPageTextRuns(ctx context.Context, inst pdfium.Pdfium, doc references.FPDF_DOCUMENT, pageIndex, maxRuns int) ([]Run, error) {
	if maxRuns <= 0 {
		return nil, nil
	}
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: doc, Index: pageIndex}}
	loaded, err := inst.FPDFText_LoadPage(&requests.FPDFText_LoadPage{Page: page})
	if err != nil {
		return nil, malformed(err)
	}
	defer inst.FPDFText_ClosePage(&requests.FPDFText_ClosePage{TextPage: loaded.TextPage})

	counted, err := inst.FPDFText_CountChars(&requests.FPDFText_CountChars{TextPage: loaded.TextPage})
	if err != nil {
		return nil, malformed(err)
	}
	n := counted.Count
	if n <= 0 {
		return nil, nil
	}
	// One call returns the page text in PDFium's reading order, with line
	// breaks between its lines. Rects are not lines: ArchiCAD writes every
	// word as its own text object, so a rect can be "CC" or "-". Per-rect text
	// and position lookups each scan the whole page, so they also grew with
	// the square of a dense page.
	got, err := inst.FPDFText_GetText(&requests.FPDFText_GetText{TextPage: loaded.TextPage, StartIndex: 0, Count: n})
	if err != nil {
		return nil, malformed(err)
	}
	chars := []rune(got.Text)
	// Unmapped glyphs can be omitted by GetText, so text offsets no longer
	// correspond to PDFium character indices. Recover that mapping instead of
	// discarding geometry for every otherwise readable title-block character.
	if len(chars) != n {
		chars = make([]rune, n)
		for i := range chars {
			if i%256 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			ch, err := inst.FPDFText_GetUnicode(&requests.FPDFText_GetUnicode{TextPage: loaded.TextPage, Index: i})
			if err != nil {
				return nil, malformed(err)
			}
			chars[i] = rune(ch.Unicode)
		}
	}
	// A box per character is a constant-time call. It places the gaps that
	// split one PDFium line into title-block cells. A surrogate pair breaks
	// the rune-to-index mapping; the fallback above reads by character index.
	var boxes []charBox
	if len(chars) == n {
		boxes = make([]charBox, n)
		for i := range chars {
			if i%256 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			// Line separators never participate in glyph geometry. Avoid a
			// WASM round trip for each PDFium-inserted CR/LF on dense sheets.
			if chars[i] == '\r' || chars[i] == '\n' || chars[i] == 0 {
				continue
			}
			box, err := inst.FPDFText_GetLooseCharBox(&requests.FPDFText_GetLooseCharBox{TextPage: loaded.TextPage, Index: i})
			if err != nil {
				return nil, malformed(err)
			}
			boxes[i] = charBox{
				left:   float64(box.Rect.Left),
				bottom: float64(box.Rect.Bottom),
				right:  float64(box.Rect.Right),
				top:    float64(box.Rect.Top),
			}
		}
	}
	lines := pdfLines(chars, boxes, func(index int) int {
		return charRotation(inst, loaded.TextPage, index, boxes != nil)
	})
	// Drawing publishers often put the title block last in the content stream.
	// Keep both ends of the identity page under the same budget; retaining only
	// the prefix loses identity behind hundreds of dimensions and annotations.
	if len(lines) > maxRuns {
		head := maxRuns / 2
		lines = append(lines[:head:head], lines[len(lines)-(maxRuns-head):]...)
	}
	runs := make([]Run, 0, min(len(lines), maxRuns))
	for _, line := range lines {
		if len(runs) == maxRuns {
			break
		}
		text := trimKept(line.text)
		if text == "" {
			continue
		}
		runs = append(runs, Run{
			Text: text,
			Source: Source{
				X: line.box.left, Y: line.box.bottom, Width: line.box.right - line.box.left, Height: line.box.top - line.box.bottom,
				Page:     pageIndex + 1,
				Rotation: charRotation(inst, loaded.TextPage, line.first, boxes != nil),
			},
		})
	}
	return runs, nil
}

// charRotation is the angle of a line's first glyph. Without boxes the char
// index is not trusted, and the line stays upright.
func charRotation(inst pdfium.Pdfium, textPage references.FPDF_TEXTPAGE, index int, indexed bool) int {
	if !indexed || index < 0 {
		return 0
	}
	ang, err := inst.FPDFText_GetCharAngle(&requests.FPDFText_GetCharAngle{
		TextPage: textPage,
		Index:    index,
	})
	if err != nil {
		// A missing angle still leaves the text. Rotation stays upright.
		return 0
	}
	return degreesCCW(float64(ang.CharAngle))
}

// charBox is a glyph's loose box in page units, origin bottom left.
type charBox struct {
	left, bottom, right, top float64
}

func (b charBox) visible() bool {
	return b.right > b.left && b.top > b.bottom
}

// size is the glyph's larger side, which is close to the font height for
// upright and turned text alike.
func (b charBox) size() float64 {
	return math.Max(b.right-b.left, b.top-b.bottom)
}

// gap is the clear distance between two boxes along the axis they are
// furthest apart on. Boxes that touch or overlap have no gap.
func (b charBox) gap(o charBox) float64 {
	return math.Max(
		math.Max(o.left-b.right, b.left-o.right),
		math.Max(o.bottom-b.top, b.bottom-o.top),
	)
}

// cellGap is the clear space, in font heights, that splits one PDFium line
// into two runs. Words sit about a third of a height apart; title-block cells
// on a shared baseline sit several heights apart.
const cellGap = 1.5

type pdfLine struct {
	text  string
	first int // char index of the first visible glyph, or -1
	box   charBox
}

// pdfLines splits PDFium page text into lines at its line breaks, then splits
// a line again wherever two visible glyphs are a cell gap apart. boxes may be
// nil, which keeps PDFium's lines whole.
func pdfLines(chars []rune, boxes []charBox, rotation func(int) int) []pdfLine {
	var out []pdfLine
	var b strings.Builder
	first, last := -1, -1
	angle := -1
	flush := func() {
		if b.Len() > 0 {
			line := pdfLine{text: b.String(), first: first}
			if boxes != nil && first >= 0 && last >= 0 {
				a, z := boxes[first], boxes[last]
				line.box = charBox{left: math.Min(a.left, z.left), bottom: math.Min(a.bottom, z.bottom), right: math.Max(a.right, z.right), top: math.Max(a.top, z.top)}
			}
			out = append(out, line)
		}
		b.Reset()
		first, last = -1, -1
		angle = -1
	}
	for i := 0; i < len(chars); i++ {
		r := chars[i]
		if r == '\r' || r == '\n' || r == 0 {
			// PDFium can insert a line break between adjacent text objects,
			// even inside a sheet number. Join only adjacent glyphs
			// on the same baseline in reading coordinates, including rotated
			// CAD titles; never join separate rows/cells.
			if boxes != nil && last >= 0 {
				next := i + 1
				for next < len(chars) && (chars[next] == '\r' || chars[next] == '\n' || chars[next] == 0) {
					next++
				}
				if next < len(chars) {
					if angle < 0 {
						angle = 0
						if rotation != nil {
							angle = rotation(first)
						}
					}
					a, z := readingBox(boxes[last], angle), readingBox(boxes[next], angle)
					if adjacentGlyphs(a, z, .5) && (rotation == nil || rotation(next) == angle) {
						// A word-sized printed gap survives as a space even when
						// PDFium represented it with a line break.
						if !touchingGlyphs(a, z) && i > 0 && chars[i-1] != ' ' {
							b.WriteByte(' ')
						}
						i = next - 1
						continue
					}
				}
			}
			flush()
			continue
		}
		if boxes != nil && r != ' ' && boxes[i].visible() {
			if last >= 0 {
				scale := math.Max(boxes[last].size(), boxes[i].size())
				// A large cover badge can overlap a neighbouring small heading.
				// Split at a word boundary when glyph sizes change drastically;
				// ordinary within-word case changes still stay together.
				minSize := math.Min(boxes[last].size(), boxes[i].size())
				fontBreak := i > 0 && chars[i-1] == ' ' && minSize > 0 && scale > 2*minSize
				if boxes[last].gap(boxes[i]) > cellGap*scale || fontBreak {
					flush()
				}
			}
			if first < 0 {
				first = i
			}
			last = i
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

// Compare glyph adjacency along the text's baseline, not the page's x axis.
// Keep the original boxes for source provenance and downstream title binding.
func readingBox(b charBox, rotation int) charBox {
	switch rotation {
	case 0:
		return b
	case 90:
		return charBox{left: -b.top, right: -b.bottom, bottom: b.left, top: b.right}
	case 180:
		return charBox{left: -b.right, right: -b.left, bottom: -b.top, top: -b.bottom}
	case 270:
		return charBox{left: b.bottom, right: b.top, bottom: -b.right, top: -b.left}
	default:
		return charBox{}
	}
}

func touchingGlyphs(a, b charBox) bool {
	return adjacentGlyphs(a, b, .15)
}

func adjacentGlyphs(a, b charBox, maxGap float64) bool {
	if !a.visible() || !b.visible() {
		return false
	}
	h := math.Max(a.top-a.bottom, b.top-b.bottom)
	gap := b.left - a.right
	return h > 0 && math.Abs(a.bottom-b.bottom) < h*.2 && math.Abs(a.top-b.top) < h*.2 && b.left > a.left && gap >= -h*.15 && gap <= h*maxGap
}

func degreesCCW(rad float64) int {
	if math.IsNaN(rad) || math.IsInf(rad, 0) {
		return 0
	}
	n := int(math.Round(rad * 180 / math.Pi))
	n %= 360
	if n < 0 {
		n += 360
	}
	for _, axis := range []int{0, 90, 180, 270} {
		d := n - axis
		if d < 0 {
			d = -d
		}
		if d <= 2 {
			return axis
		}
	}
	if n >= 358 {
		return 0
	}
	return n
}

// PDFium seeks through this reader synchronously. Preserve a read-limit or
// cancellation error, since its callback otherwise reports only a parse failure.
type pdfBudgetReader struct {
	ctx       context.Context
	reader    *io.SectionReader
	remaining int64
	err       error
}

func (r *pdfBudgetReader) Seek(offset int64, whence int) (int64, error) {
	return r.reader.Seek(offset, whence)
}
func (r *pdfBudgetReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if err := r.ctx.Err(); err != nil {
		r.err = err
		return 0, err
	}
	if int64(len(p)) > r.remaining {
		r.err = ErrTooLarge
		return 0, r.err
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	return n, err
}

// This controls extraction scope, not document classification. A second page
// is retained only when it explicitly identifies itself as document control.
func probeControlPage(runs []Run) bool {
	if len(runs) < 3 || len(runs) > 20 {
		return false
	}
	cue := false
	application, prepared := false, false
	for _, r := range runs {
		s := strings.ToLower(strings.TrimSpace(r.Text))
		application = application || s == "development application"
		cue = cue || strings.HasSuffix(s, "management plan")
		prepared = prepared || strings.HasPrefix(s, "prepared for ")
		if strings.Contains(s, "drawing number") || strings.Contains(s, "drawing no") || strings.Contains(s, "dwg no") || strings.Contains(s, "drg no") {
			return false
		}
		for _, word := range strings.Fields(s) {
			switch strings.Trim(word, ":.,") {
			case "report", "specification", "statement", "brief", "advice", "assessment", "requirements":
				cue = true
			}
		}
	}
	return cue || (application && prepared)
}

// Keep explicit identity rows on a publisher/disclaimer page, not its prose.
func reportMetadata(runs []Run) []Run {
	labels := map[string]bool{}
	var out []Run
	for i, r := range runs {
		v := strings.ToLower(strings.TrimSpace(r.Text))
		for _, label := range []string{"report", "prepared for", "prepared by", "project no", "date", "version"} {
			prefix := label + ":"
			if !strings.HasPrefix(v, prefix) {
				continue
			}
			if len(strings.TrimSpace(v[len(prefix):])) > 0 {
				out = append(out, r)
				labels[label] = true
				break
			}
			if i+1 >= len(runs) {
				break
			}
			next := runs[i+1]
			a, b := r.Source, next.Source
			if a.Height > 0 && a.Page == b.Page && a.Rotation == b.Rotation && math.Abs(a.Y-b.Y) < a.Height*.5 && b.X >= a.X+a.Width && b.X-a.X < a.Height*40 {
				out = append(out, r, next)
				labels[label] = true
			}
			break
		}
	}
	if !labels["report"] || !labels["prepared for"] || !labels["prepared by"] || !labels["version"] {
		return nil
	}
	return out
}

// Follow only an explicit contents entry for the immediately following page.
// Contents text itself is not identity evidence.
func nextPageControlReference(runs []Run) bool {
	title, entry := false, false
	for _, r := range runs {
		v := strings.ToLower(strings.TrimSpace(r.Text))
		title = title || v == "table of contents"
		entry = entry || nextControlEntry.MatchString(v)
	}
	return title && entry
}

var nextControlEntry = regexp.MustCompile(`^document control\s*\.{2,}\s*3$`)

// A numbered control section can share a page with findings and cited reports.
// Keep its geometric region only when the next sibling section bounds it.
func boundedControlSection(runs []Run) []Run {
	for i, r := range runs {
		if i == 0 || !strings.EqualFold(strings.TrimSpace(r.Text), "Document Control") {
			continue
		}
		number, h := runs[i-1], r.Source
		if strings.TrimSpace(number.Text) != "1" || h.Height <= 0 || h.Rotation != 0 || number.Source.Page != h.Page || math.Abs(number.Source.Y-h.Y) > h.Height*.5 {
			continue
		}
		for j := i + 1; j+1 < len(runs); j++ {
			n, next := runs[j], runs[j+1]
			if strings.TrimSpace(n.Text) != "2" || n.Source.Page != h.Page || next.Source.Page != h.Page || next.Source.Rotation != 0 ||
				math.Abs(n.Source.X-number.Source.X) > h.Height*.5 || math.Abs(n.Source.Y-next.Source.Y) > h.Height*.5 ||
				math.Abs(next.Source.X-h.X) > h.Height*.5 || math.Abs(next.Source.Height-h.Height) > h.Height*.2 || next.Source.Y >= h.Y {
				continue
			}
			var out []Run
			for _, item := range runs {
				s := item.Source
				if s.Page == h.Page && s.Rotation == 0 && s.Y <= h.Y+h.Height && s.Y > next.Source.Y+next.Source.Height {
					out = append(out, item)
				}
			}
			return out
		}
	}
	return nil
}

func repeatedReportCover(first, second []Run) bool {
	if !probeControlPage(first) || !probeControlPage(second) {
		return false
	}
	shared := 0
	seen := map[string]bool{}
	for _, r := range first {
		seen[strings.ToLower(strings.TrimSpace(r.Text))] = true
	}
	for _, r := range second {
		v := strings.ToLower(strings.TrimSpace(r.Text))
		if len(v) >= 4 && seen[v] {
			shared++
			delete(seen, v)
		}
	}
	return shared >= 3
}

func controlPage(runs []Run) bool {
	for _, r := range runs {
		switch strings.ToLower(strings.Trim(strings.TrimSpace(r.Text), ":")) {
		case "document information", "document control", "document verification history":
			return true
		}
	}
	var history strings.Builder
	for _, r := range runs {
		history.WriteString(r.Text)
		history.WriteByte('\n')
	}
	// Repeated explicit revision, issue and authorisation labels identify a
	// control history even without a Document Control heading. Retaining it
	// exposes conflicts; it does not select the newest date or revision.
	value := history.String()
	// A publisher can use approval-table columns instead of a Document
	// Control heading. Require its own reference as well as all five labels.
	flat := strings.ToLower(strings.Join(strings.Fields(value), " "))
	if strings.Contains(flat, "amendment schedule") && strings.Contains(flat, "revision no") && strings.Contains(flat, "date") && strings.Contains(flat, "description") {
		return true
	}
	if strings.Contains(flat, "document number") && strings.Contains(flat, "quality information") && strings.Contains(flat, "distribution") && strings.Contains(flat, "issue revision issued to date prepared reviewed") {
		return true
	}
	if strings.Contains(flat, "document reference") && strings.Contains(flat, "status issue") && strings.Contains(flat, "prepared by") && strings.Contains(flat, "checked by") && strings.Contains(flat, "approved by") && strings.Contains(flat, "date") {
		return true
	}
	return len(controlRevision.FindAllString(value, -1)) >= 3 && len(controlIssued.FindAllString(value, -1)) >= 3 && len(controlAuthorised.FindAllString(value, -1)) >= 3
}

var (
	controlRevision   = regexp.MustCompile(`(?i)\brev(?:ision)?\s+[a-z0-9]{1,4}\b`)
	controlIssued     = regexp.MustCompile(`(?i)\bissue(?:d)?\s+\d{1,2}\s+[a-z]+\s+\d{4}\b`)
	controlAuthorised = regexp.MustCompile(`(?i)\bauthori[sz]ed\s+by\b`)
)
