package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"sitewise/internal/identity"
	"sitewise/internal/store"
)

// FullSource preserves every extracted page before making bounded reading
// units. This is separate from the deliberately limited identity extractor.
func FullSource(st *store.Store, blobs BlobPaths) func(context.Context, string, string) (store.DocumentSource, error) {
	return func(ctx context.Context, orgID, docID string) (store.DocumentSource, error) {
		doc, err := st.GetDocument(ctx, orgID, docID)
		if err != nil {
			return store.DocumentSource{}, err
		}
		file, err := st.GetFile(ctx, orgID, doc.FileID)
		if err != nil {
			return store.DocumentSource{}, err
		}
		path, err := blobs.Path(file.SHA256)
		if err != nil {
			return store.DocumentSource{}, err
		}
		f, err := os.Open(path)
		if err != nil {
			return store.DocumentSource{}, err
		}
		defer f.Close()
		limits := fullLimits
		limits.RequireComplete = true
		// Bounds on bytes/runs still protect the host. Hitting them is an error;
		// page, row and column counts must never silently truncate full extraction.
		limits.MaxPages = int(^uint(0) >> 1)
		limits.MaxSheets = int(^uint(0) >> 1)
		limits.MaxRows = int(^uint(0) >> 1)
		limits.MaxCols = int(^uint(0) >> 1)
		text, err := identity.Extract(ctx, strings.TrimPrefix(strings.ToLower(filepath.Ext(doc.Filename)), "."), f, file.ByteSize, limits)
		if err != nil {
			return store.DocumentSource{}, err
		}
		if len(text.Runs) >= limits.MaxRuns {
			return store.DocumentSource{}, fmt.Errorf("full text reached its safety limit; document needs attention")
		}
		return SourceFromText(text), nil
	}
}

// SourceFromText keeps page/row provenance. A source page with no extractable
// text remains visible: it may be a scan or a genuinely blank page.
func SourceFromText(text identity.Text) store.DocumentSource {
	out := store.DocumentSource{Pages: text.PageCount, EmptyPages: []int{}}
	index := map[string]int{}
	if text.PageCount > 0 {
		for p := 1; p <= text.PageCount; p++ {
			index[fmt.Sprint(p)] = len(out.Source)
			out.Source = append(out.Source, store.SourcePage{Page: p, Location: fmt.Sprintf("Page %d", p)})
		}
	}
	active := ""
	lastIndex := -1
	tableHeaders := map[string]string{}
	runs := attachBullets(text.Runs)
	heads := listHeads(runs)
	for _, r := range runs {
		key := fmt.Sprint(r.Source.Page)
		loc := "Document"
		tableKey := ""
		if r.Source.Sheet != "" {
			tableKey = "sheet:" + r.Source.Sheet
			key = fmt.Sprintf("%s:%d", tableKey, r.Source.Row)
			loc = fmt.Sprintf("%s, row %d", r.Source.Sheet, r.Source.Row)
		}
		if text.PageCount == 0 && r.Source.Table > 0 {
			tableKey = fmt.Sprintf("table:%d", r.Source.Table)
			key = fmt.Sprintf("%s:%d", tableKey, r.Source.Row)
			loc = fmt.Sprintf("Table %d, row %d", r.Source.Table, r.Source.Row)
		}
		i, ok := index[key]
		if text.PageCount == 0 {
			ok = key == active && lastIndex >= 0
			i = lastIndex
		}
		if !ok {
			i = len(out.Source)
			index[key] = i
			out.Source = append(out.Source, store.SourcePage{Page: r.Source.Page, Location: loc})
			if tableKey != "" && r.Source.Row > 1 {
				out.Source[i].Context = "First table row (possible column headings): " + tableHeaders[tableKey]
			}
		}
		active = key
		lastIndex = i
		if r.Source.Page > 0 && r.Source.Width > 0 {
			out.Source[i].Regions = append(out.Source[i].Regions, store.SourceRegion{Text: r.Text, X: r.Source.X, Y: r.Source.Y, Width: r.Source.Width, Height: r.Source.Height})
		}
		part := r.Text
		sep := "\n"
		if tableKey != "" {
			sep = "\t"
		} else if r.Source.Heading || looksLikeHeading(strings.TrimSpace(part)) || text.Format == "docx" {
			sep = "\n\n"
		}
		if out.Source[i].Text != "" {
			out.Source[i].Text += sep
		}
		out.Source[i].Text += part
		if tableKey != "" && r.Source.Row == 1 {
			tableHeaders[tableKey] = out.Source[i].Text
		}
	}

	section, lead := "", ""
	repeated := map[string]int{}
	for _, p := range out.Source {
		seen := map[string]bool{}
		lines := strings.Split(p.Text, "\n")
		for i, l := range lines {
			l = strings.TrimSpace(l)
			if (i < furnitureEdge || i >= len(lines)-furnitureEdge) && l != "" && !seen[l] {
				repeated[l]++
				seen[l] = true
			}
		}
	}
	furniture := map[string]bool{}
	for line, n := range repeated {
		if n >= 3 && n*2 >= len(out.Source) {
			furniture[line] = true
		}
	}
	read := map[string]bool{}
	for _, p := range out.Source {
		if strings.TrimSpace(p.Text) == "" {
			if p.Page > 0 {
				out.EmptyPages = append(out.EmptyPages, p.Page)
			}
			continue
		}
		units, next, nextLead := sourceUnits(p, section, lead, pageLayout{furniture: furniture, read: read, heads: heads[p.Page]})
		section, lead = next, nextLead
		out.Units = append(out.Units, units...)
	}
	return out
}

var clauseStart = regexp.MustCompile(`^(?:\([a-z0-9]+\)|[a-z][.)]|[•●]|\d+(?:\.\d+)+\s)\s*`)
var footerLine = regexp.MustCompile(`^\d+(?:\.\d+)*$`)
var splitHeadingNumber = regexp.MustCompile(`^\d+(?:\.\d+)+\.?$`)

// furnitureEdge is how many lines at the top and bottom of a page are
// compared across pages: a firm's footer runs to name, address, phone, web,
// email, ABN, print date and page number.
const furnitureEdge = 10

var (
	loneBullet  = regexp.MustCompile(`^\s*[•●▪◦]\s*$`)
	bulletStart = regexp.MustCompile(`^\s*[•●▪◦]`)
	pageNumber  = regexp.MustCompile(`(?i)^page\s+\d+(?:\s+of\s+\d+)?$`)
)

// attachBullets puts a bullet glyph back on its own line. An estimating
// tool's PDF emits the glyph of a wrapped item after the item's text; the
// page geometry still says which line it sits beside. Without it the glyph
// opens the next item, and the last one in a list is a passage of its own.
func attachBullets(runs []identity.Run) []identity.Run {
	out := make([]identity.Run, 0, len(runs))
	for _, r := range runs {
		if r.Source.Width > 0 && loneBullet.MatchString(r.Text) {
			if j := bulletLine(out, r.Source); j >= 0 {
				out[j].Text = strings.TrimSpace(r.Text) + " " + out[j].Text
				out[j].Source.Width += out[j].Source.X - r.Source.X
				out[j].Source.X = r.Source.X
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// bulletLine is the earlier line on the same page that a lone bullet sits
// beside: the same baseline, to the bullet's right, with no bullet yet.
func bulletLine(prior []identity.Run, b identity.Source) int {
	for j := len(prior) - 1; j >= 0 && j >= len(prior)-40; j-- {
		s := prior[j].Source
		if s.Page != b.Page || s.Sheet != "" {
			return -1
		}
		if dy := s.Y - b.Y; s.Width > 0 && s.X > b.X && dy < b.Height/4 && dy > -b.Height/4 && !bulletStart.MatchString(prior[j].Text) {
			return j
		}
	}
	return -1
}

// listHeads are, by page, the headings only the page geometry shows: a short
// plain line set left of the bulleted list directly below it ("Roofing &
// Cladding"), or one set in clearly larger type than the page's body
// ("Quoted Categories").
func listHeads(runs []identity.Run) map[int]map[string]bool {
	heights := map[int][]float64{}
	for _, r := range runs {
		if r.Source.Width > 0 && r.Source.Sheet == "" {
			heights[r.Source.Page] = append(heights[r.Source.Page], r.Source.Height)
		}
	}
	body := map[int]float64{}
	for page, hs := range heights {
		sort.Float64s(hs)
		body[page] = hs[len(hs)/2]
	}
	out := map[int]map[string]bool{}
	for i, r := range runs {
		t := strings.TrimSpace(r.Text)
		if r.Source.Width <= 0 || t == "" || len(t) >= 100 || strings.Contains(t, "\n") || bulletStart.MatchString(t) ||
			strings.ContainsAny(t[len(t)-1:], ".;:,") || strings.Contains(t, "$") || t[0] < 'A' || t[0] > 'Z' {
			continue
		}
		large := len(heights[r.Source.Page]) >= 5 && r.Source.Height >= 1.2*body[r.Source.Page]
		aboveList := i+1 < len(runs) && runs[i+1].Source.Page == r.Source.Page && bulletStart.MatchString(runs[i+1].Text) && runs[i+1].Source.X >= r.Source.X+5
		if !large && !aboveList {
			continue
		}
		if out[r.Source.Page] == nil {
			out[r.Source.Page] = map[string]bool{}
		}
		out[r.Source.Page][t] = true
	}
	return out
}

// pageLayout is what the whole document says about one page's lines.
type pageLayout struct {
	furniture map[string]bool // lines repeated at the edges of most pages
	read      map[string]bool // furniture already in an earlier unit
	heads     map[string]bool // list headings found by geometry
}

// repeatedFurniture reports that every line of a furniture unit was in an
// earlier one, and records this unit's lines as read. A bare page number
// says nothing the first time either.
func repeatedFurniture(body string, read map[string]bool) bool {
	repeat := true
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || pageNumber.MatchString(l) {
			continue
		}
		if !read[l] {
			repeat = false
		}
		read[l] = true
	}
	return repeat
}

var quantityLine = regexp.MustCompile(`^\d+\.\d+(?:\s+[A-Za-z][A-Za-z0-9²³]{0,4})?$`)
var moneyLine = regexp.MustCompile(`^\$\s?\d[\d,]*(?:\.\d{2})?$`)

// pricedRow reports that one of the next three lines is only an amount.
func pricedRow(lines []string, i int) bool {
	for k := i + 1; k <= i+3 && k < len(lines); k++ {
		if moneyLine.MatchString(strings.TrimSpace(lines[k])) {
			return true
		}
	}
	return false
}

// bulletItem reports a unit that is one bulleted item, under its list heading
// or not. It ends at the next bullet, not at a full stop: the second
// sentence of "ASBESTOS - ... Includes Clearance Certificate." has no
// subject of its own.
func bulletItem(body string, heads map[string]bool) bool {
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || heads[l] {
			continue
		}
		return bulletStart.MatchString(l)
	}
	return false
}

// edgeFurniture marks the run of repeated lines and page numbers that touches
// the top or the bottom of a page. A page that is nothing but repeated lines
// is content (a form filled the same way on every page), not furniture.
func edgeFurniture(lines []string, furniture map[string]bool) []bool {
	edge := make([]bool, len(lines))
	is := func(i int) bool {
		l := strings.TrimSpace(lines[i])
		return l == "" || furniture[l] || pageNumber.MatchString(l)
	}
	top := 0
	for top < len(lines) && is(top) {
		edge[top] = true
		top++
	}
	if top == len(lines) {
		return make([]bool, len(lines))
	}
	for i := len(lines) - 1; i > top && is(i); i-- {
		edge[i] = true
	}
	return edge
}

// Units partition the source text (byte offsets), never delete it. Headings
// and list introductions are context, not substituted for the source clause.
func sourceUnits(p store.SourcePage, section, lead string, lay pageLayout) ([]store.SourceUnit, string, string) {
	furniture := lay.furniture
	var out []store.SourceUnit
	start := 0
	if p.Context != "" {
		lead = p.Context
	}
	inFurniture := false
	emit := func(end int) {
		if end <= start {
			return
		}
		body := p.Text[start:end]
		if strings.TrimSpace(body) != "" {
			u := store.SourceUnit{Body: body, Page: p.Page, Location: p.Location, Section: section, Context: lead, Start: start, End: end}
			if inFurniture {
				u.Section, u.Context = "", ""
				if repeatedFurniture(body, lay.read) {
					u.Category = store.RepeatedFurniture
				}
			}
			out = append(out, u)
		}
		start = end
	}
	offset := 0
	lines := strings.SplitAfter(p.Text, "\n")
	edge := edgeFurniture(lines, furniture)
	headingOnly := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		clean := strings.TrimSpace(line)
		// A page header or footer is a unit of its own, so the firm's address
		// does not ride on the last item of every page.
		if clean != "" && edge[i] != inFurniture {
			emit(offset)
			inFurniture = edge[i]
		}
		// PDFium may emit the number and title on separate lines. Combine only
		// for recognition; the original bytes and offsets remain untouched.
		if splitHeadingNumber.MatchString(clean) && i+1 < len(lines) {
			title := strings.TrimSpace(lines[i+1])
			if title != "" && title[0] >= 'A' && title[0] <= 'Z' && len(title) < 80 && !strings.HasSuffix(title, ".") {
				clean += " " + title
				line += lines[i+1]
				i++
			}
		}
		// "4.12 Rock for gravel driveway" above its quantity and price is a
		// row of a priced table, not a numbered heading.
		heading := (lay.heads[clean] || looksLikeHeading(clean) && !pricedRow(lines, i)) && !footerLine.MatchString(clean) && !furniture[clean] && len(clean) < 100
		// A To / Project / Site block after a letterhead is its own unit, so
		// the firm's office address is not read as the project's.
		addressee := i > 0 && addresseeLine.MatchString(clean) && !addresseeLine.MatchString(strings.TrimSpace(lines[i-1]))
		// "6.000 Ea" is a row's quantity, not clause 6.000.
		clause := clauseStart.MatchString(clean) && !quantityLine.MatchString(clean)
		if heading || addressee || (!headingOnly && (clause || clean == "")) {
			emit(offset)
		}
		if heading {
			section = clean
			lead = ""
			headingOnly = true
		} else if clean != "" {
			headingOnly = false
		}
		if utf8.RuneCountInString(p.Text[start:offset+len(line)]) > 1200 && offset > start {
			emit(offset)
		}
		offset += len(line)
		if !heading && strings.HasSuffix(clean, ":") {
			intro := strings.TrimSpace(p.Text[start:offset])
			if len(intro) > 1200 {
				intro = clean
			}
			emit(offset)
			lead = intro
		} else if (strings.HasSuffix(clean, ".") || strings.HasSuffix(clean, ";")) && len(strings.TrimSpace(p.Text[start:offset])) >= 80 && !bulletItem(p.Text[start:offset], lay.heads) {
			emit(offset)
		} else if moneyLine.MatchString(clean) {
			// A line that is only an amount closes a row of a priced table,
			// so each row is read with its own description.
			emit(offset)
		}
	}
	emit(len(p.Text))
	// Unbroken paragraphs still have bounded state, without dropping the tail.
	var bounded []store.SourceUnit
	for _, u := range out {
		for utf8.RuneCountInString(u.Body) > 2000 {
			cut := cutRunes(u.Body, 1800)
			part := u
			part.Body = u.Body[:cut]
			part.End = u.Start + cut
			bounded = append(bounded, part)
			u.Body = u.Body[cut:]
			u.Start += cut
		}
		bounded = append(bounded, u)
	}
	return bounded, section, lead
}
