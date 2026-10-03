package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	for _, r := range text.Runs {
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
			if (i < 6 || i >= len(lines)-6) && l != "" && !seen[l] {
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
	for _, p := range out.Source {
		if strings.TrimSpace(p.Text) == "" {
			if p.Page > 0 {
				out.EmptyPages = append(out.EmptyPages, p.Page)
			}
			continue
		}
		units, next, nextLead := sourceUnits(p, section, lead, furniture)
		section, lead = next, nextLead
		out.Units = append(out.Units, units...)
	}
	return out
}

var clauseStart = regexp.MustCompile(`^(?:\([a-z0-9]+\)|[a-z][.)]|[•●]|\d+(?:\.\d+)+\s)\s*`)
var footerLine = regexp.MustCompile(`^\d+(?:\.\d+)*$`)
var splitHeadingNumber = regexp.MustCompile(`^\d+(?:\.\d+)+\.?$`)

// Units partition the source text (byte offsets), never delete it. Headings
// and list introductions are context, not substituted for the source clause.
func sourceUnits(p store.SourcePage, section, lead string, furniture map[string]bool) ([]store.SourceUnit, string, string) {
	var out []store.SourceUnit
	start := 0
	if p.Context != "" {
		lead = p.Context
	}
	emit := func(end int) {
		if end <= start {
			return
		}
		body := p.Text[start:end]
		if strings.TrimSpace(body) != "" {
			out = append(out, store.SourceUnit{Body: body, Page: p.Page, Location: p.Location, Section: section, Context: lead, Start: start, End: end})
		}
		start = end
	}
	offset := 0
	lines := strings.SplitAfter(p.Text, "\n")
	headingOnly := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		clean := strings.TrimSpace(line)
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
		heading := looksLikeHeading(clean) && !footerLine.MatchString(clean) && !furniture[clean] && len(clean) < 100
		if heading || (!headingOnly && (clauseStart.MatchString(clean) || clean == "")) {
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
		} else if (strings.HasSuffix(clean, ".") || strings.HasSuffix(clean, ";")) && len(strings.TrimSpace(p.Text[start:offset])) >= 80 {
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
