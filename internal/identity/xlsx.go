package identity

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"io"
	"path"
	"strconv"
	"strings"
)

func extractXLSX(ctx context.Context, r io.ReaderAt, size int64, limits Limits) (Text, error) {
	zr, err := openPackage(r, size, limits.MaxBytes)
	if err != nil {
		return Text{}, err
	}
	sheets, err := workbookSheets(ctx, zr, limits)
	if err != nil {
		return Text{}, err
	}
	var cells []pendingCell
	needed := map[int]struct{}{}
	for _, sheet := range sheets {
		if err := ctx.Err(); err != nil {
			return Text{}, err
		}
		member := zipMember(zr, sheet.path)
		if member == nil {
			return Text{}, ErrMalformed
		}
		rc, br, err := openMember(ctx, member, limits.MaxBytes)
		if err != nil {
			return Text{}, err
		}
		sheetCells, err := readSheetCells(ctx, br, sheet.name, limits)
		rc.Close()
		if err != nil {
			return Text{}, err
		}
		for _, cell := range sheetCells {
			if len(cells) >= limits.MaxRuns {
				break
			}
			if cell.shared >= 0 {
				needed[cell.shared] = struct{}{}
			}
			cells = append(cells, cell)
		}
	}
	var shared map[int]string
	if len(needed) > 0 {
		shared, err = sharedStrings(ctx, zr, needed, limits.MaxBytes)
		if err != nil {
			return Text{}, err
		}
	}
	runs := make([]Run, 0, len(cells))
	for _, cell := range cells {
		if cell.shared >= 0 {
			text, ok := shared[cell.shared]
			if !ok {
				continue
			}
			cell.text = text
		}
		cell.text = trimKept(cell.text)
		if cell.text == "" {
			continue
		}
		runs = append(runs, cell.run())
	}
	return Text{Runs: runs}, nil
}

type sheetRef struct {
	name string
	path string
}

type pendingCell struct {
	text   string
	shared int // -1 when text is already resolved
	source Source
}

func (c pendingCell) run() Run {
	return Run{Text: c.text, Source: c.source}
}

func workbookSheets(ctx context.Context, zr *zip.Reader, limits Limits) ([]sheetRef, error) {
	book := zipMember(zr, "xl/workbook.xml")
	rels := zipMember(zr, "xl/_rels/workbook.xml.rels")
	if book == nil || rels == nil {
		return nil, ErrMalformed
	}
	rc, br, err := openMember(ctx, book, limits.MaxBytes)
	if err != nil {
		return nil, err
	}
	names, ids, err := parseWorkbook(br)
	rc.Close()
	if err != nil {
		return nil, err
	}
	rc, br, err = openMember(ctx, rels, limits.MaxBytes)
	if err != nil {
		return nil, err
	}
	targets, err := parseSheetRels(br)
	rc.Close()
	if err != nil {
		return nil, err
	}
	if len(names) > limits.MaxSheets {
		names = names[:limits.MaxSheets]
		ids = ids[:limits.MaxSheets]
	}
	out := make([]sheetRef, 0, len(names))
	for i, name := range names {
		target, ok := targets[ids[i]]
		if !ok {
			return nil, ErrMalformed
		}
		resolved, err := resolveSheetPath(target)
		if err != nil {
			return nil, err
		}
		out = append(out, sheetRef{name: name, path: resolved})
	}
	if len(out) == 0 {
		return nil, ErrMalformed
	}
	return out, nil
}

func parseWorkbook(r io.Reader) (names, ids []string, err error) {
	dec := newXML(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return names, ids, nil
		}
		if err != nil {
			return nil, nil, malformed(err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != "sheet" {
			continue
		}
		var name, id string
		for _, attr := range el.Attr {
			switch attr.Name.Local {
			case "name":
				name = attr.Value
			case "id":
				id = attr.Value
			}
		}
		if name == "" || id == "" {
			return nil, nil, ErrMalformed
		}
		names = append(names, name)
		ids = append(ids, id)
	}
}

func parseSheetRels(r io.Reader) (map[string]string, error) {
	dec := newXML(r)
	targets := map[string]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return targets, nil
		}
		if err != nil {
			return nil, malformed(err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != "Relationship" {
			continue
		}
		var id, target, typ string
		for _, attr := range el.Attr {
			switch attr.Name.Local {
			case "Id":
				id = attr.Value
			case "Target":
				target = attr.Value
			case "Type":
				typ = attr.Value
			}
		}
		if id == "" || target == "" || !strings.Contains(typ, "worksheet") {
			continue
		}
		targets[id] = target
	}
}

func resolveSheetPath(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" || strings.Contains(target, "\\") {
		return "", ErrMalformed
	}
	target = strings.TrimPrefix(target, "/")
	joined := target
	if !strings.HasPrefix(target, "xl/") {
		joined = path.Join("xl", target)
	}
	joined = path.Clean(joined)
	if err := safeZipName(joined); err != nil {
		return "", err
	}
	return joined, nil
}

func readSheetCells(ctx context.Context, r io.Reader, sheet string, limits Limits) ([]pendingCell, error) {
	dec := newXML(r)
	merges := map[string]string{}
	var cells []pendingCell
	var (
		row          int
		inCell       bool
		inValue      bool
		inFormula    bool
		inInline     bool
		inInlineText bool
		cellRef      string
		cellType     string
		formula      bool
		value        strings.Builder
		inline       strings.Builder
		col          int
		tokens       int
	)
	finish := func() {
		defer func() {
			inCell = false
			inValue = false
			inFormula = false
			inInline = false
			inInlineText = false
			formula = false
			cellRef = ""
			cellType = ""
			value.Reset()
			inline.Reset()
		}()
		if !inCell || col < 1 || col > limits.MaxCols || row < 1 || row > limits.MaxRows {
			return
		}
		if len(cells) >= limits.MaxRuns {
			return
		}
		src := Source{
			Sheet:  sheet,
			Cell:   cellRef,
			Merge:  merges[cellRef],
			Row:    row,
			Col:    col,
			Cached: formula,
		}
		if formula {
			// The cached <v> is evidence. The formula text is never the value.
			text := trimKept(value.String())
			if text == "" {
				return
			}
			cells = append(cells, pendingCell{text: text, shared: -1, source: src})
			return
		}
		switch cellType {
		case "s":
			n, err := strconv.Atoi(trimKept(value.String()))
			if err != nil || n < 0 {
				return
			}
			cells = append(cells, pendingCell{shared: n, source: src})
		case "inlineStr":
			text := trimKept(inline.String())
			if text == "" {
				return
			}
			cells = append(cells, pendingCell{text: text, shared: -1, source: src})
		default:
			text := trimKept(value.String())
			if text == "" {
				return
			}
			cells = append(cells, pendingCell{text: text, shared: -1, source: src})
		}
	}
	for {
		tokens++
		if tokens%32 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, malformed(err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "row":
				row = attrInt(el, "r")
			case "c":
				if inCell {
					finish()
				}
				inCell = true
				formula = false
				cellRef = attr(el, "r")
				cellType = attr(el, "t")
				parsedCol, parsedRow, ok := parseCellRef(cellRef)
				if ok {
					col = parsedCol
					if parsedRow > 0 {
						row = parsedRow
					}
				} else {
					col = 0
				}
				value.Reset()
				inline.Reset()
			case "v":
				if inCell {
					inValue = true
				}
			case "f":
				if inCell {
					inFormula = true
					formula = true
				}
			case "is":
				if inCell {
					inInline = true
				}
			case "t":
				if inInline {
					inInlineText = true
				}
			case "mergeCell":
				ref := attr(el, "ref")
				anchor, _, ok := strings.Cut(ref, ":")
				if ok && anchor != "" {
					merges[anchor] = ref
				}
			}
		case xml.EndElement:
			switch el.Name.Local {
			case "v":
				inValue = false
			case "f":
				inFormula = false
			case "t":
				inInlineText = false
			case "is":
				inInline = false
				inInlineText = false
			case "c":
				finish()
			}
		case xml.CharData:
			if inFormula {
				break
			}
			if inInlineText {
				inline.Write(el)
				break
			}
			if inValue {
				value.Write(el)
			}
		}
	}
	// mergeCells are usually after sheetData, so cells were finished before
	// the merge map was filled. Apply them now.
	for i := range cells {
		if merge, ok := merges[cells[i].source.Cell]; ok {
			cells[i].source.Merge = merge
		}
	}
	return cells, nil
}

func sharedStrings(ctx context.Context, zr *zip.Reader, needed map[int]struct{}, budget int64) (map[int]string, error) {
	member := zipMember(zr, "xl/sharedStrings.xml")
	if member == nil {
		return nil, ErrMalformed
	}
	maxNeeded := 0
	for n := range needed {
		if n > maxNeeded {
			maxNeeded = n
		}
	}
	rc, br, err := openMember(ctx, member, budget)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	dec := newXML(br)
	found := map[int]string{}
	index := -1
	var (
		inSI     bool
		inText   bool
		phonetic int
		buf      strings.Builder
		tokens   int
	)
	store := func() {
		if !inSI || index < 0 {
			return
		}
		if _, ok := needed[index]; ok {
			found[index] = buf.String()
		}
	}
	for {
		tokens++
		if tokens%32 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		tok, err := dec.Token()
		if err == io.EOF {
			store()
			break
		}
		if err != nil {
			return nil, malformed(err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "si":
				store()
				index++
				buf.Reset()
				inSI = true
			case "rPh":
				phonetic++
			case "t":
				inText = inSI && phonetic == 0
			}
		case xml.EndElement:
			switch el.Name.Local {
			case "t":
				inText = false
			case "rPh":
				if phonetic > 0 {
					phonetic--
				}
			case "si":
				store()
				inSI = false
				if index >= maxNeeded && len(found) == len(needed) {
					return found, nil
				}
			}
		case xml.CharData:
			if inText {
				buf.Write(el)
			}
		}
	}
	return found, nil
}

func attr(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func attrInt(el xml.StartElement, local string) int {
	n, err := strconv.Atoi(attr(el, local))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseCellRef(ref string) (col, row int, ok bool) {
	if ref == "" {
		return 0, 0, false
	}
	i := 0
	for i < len(ref) {
		c := ref[i]
		if c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		if c < 'A' || c > 'Z' {
			break
		}
		col = col*26 + int(c-'A'+1)
		if col > 16384 {
			return 0, 0, false
		}
		i++
	}
	if i == 0 || i == len(ref) {
		return 0, 0, false
	}
	for _, c := range ref[i:] {
		if c < '0' || c > '9' {
			return 0, 0, false
		}
		row = row*10 + int(c-'0')
		if row > 1_048_576 {
			return 0, 0, false
		}
	}
	return col, row, row > 0
}
