package export

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/jpeg" // a report image may be a JPEG
	_ "image/png"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"sitewise/internal/reports"
)

// Row heights and widths in points, scaled from the screen (15 px type
// prints at 10 pt, so 35 px rows print at about 23 pt).
const (
	rowHeight = 23.0
	tickWidth = 36.0
	qtyWidth  = 62.0
)

type basis struct {
	Layout    string `json:"layout"`
	StageRows int    `json:"stage_rows"`
	// A payment schedule's tables (layout "schedule"): the first Left columns
	// are words and the rest figures; Widths are fixed points per column
	// (zero shares what is left); Bold rows are totals; Notes rows are one
	// cell of words across the table; Tall rows leave room to sign.
	Left   int       `json:"left"`
	Widths []float64 `json:"widths"`
	Bold   []int     `json:"bold"`
	Notes  []int     `json:"notes"`
	Tall   []int     `json:"tall"`
	// Merge lets a row's label run across the empty cells beside it, up to
	// its first figure, so "Total claim inc retention" is one line.
	Merge bool `json:"merge"`
}

// FromSections lays out a draft's sections as the screen shows them.
// Internal sections (the project manager's review lists) are left out, as
// they are when an RFP is issued.
func FromSections(title, subtitle, date string, sections []reports.Section) Doc {
	return FromSectionsWithImages(title, subtitle, date, sections, nil)
}

// FromSectionsWithImages is FromSections for a report whose cells hold
// images: images are their bytes by SHA-256 hex. An image whose bytes are
// not given prints its caption and "image not available".
func FromSectionsWithImages(title, subtitle, date string, sections []reports.Section, images map[string][]byte) Doc {
	if golden(sections) {
		return fromGolden(title, date, sections, images)
	}
	d := Doc{Title: title, Subtitle: subtitle, Date: date}
	for _, s := range sections {
		if s.Internal || len(s.Blocks) == 0 {
			continue
		}
		out := Section{Title: s.Title}
		for _, b := range s.Blocks {
			out.Blocks = append(out.Blocks, block(b))
		}
		d.Sections = append(d.Sections, out)
	}
	return d
}

func block(b reports.Block) Block {
	out := Block{Label: b.Label, Text: strings.TrimSpace(b.Text)}
	if b.ID == "rfp:project:information" {
		// The transmittal reads as on screen: the table, then any note.
		out.Label, out.TextAfter = "", true
		out.Text = strings.TrimPrefix(out.Text, "Information provided: ")
	}
	if b.ID == "rfp:terms" && out.Text != "" {
		// Numbered as on screen, so a term is cited by its number.
		out.Table, out.Text = termsTable(out.Text), ""
		return out
	}
	if b.Table == nil || len(b.Table.Columns) == 0 {
		return out
	}
	var meta basis
	_ = json.Unmarshal(b.Basis, &meta)
	if b.Table.ID != "" && b.Table.Meta != nil {
		// A report table in the owner's format: its name above it, its note
		// under it.
		out.Table, out.TextAfter, out.Heading, out.Label = reportTable(*b.Table), true, b.Label, ""
		return out
	}
	switch {
	case meta.Layout == "stage_scope":
		out.Table = stageTable(*b.Table, meta.StageRows)
	case meta.Layout == "allowance_pairs" && len(b.Table.Columns) == 4:
		out.Table = allowanceTable(*b.Table)
	case b.ID == "rfp:project:grid" || meta.Layout == "label_value":
		out.Table = gridTable(*b.Table)
	case meta.Layout == "schedule":
		out.Table = scheduleTable(*b.Table, meta)
	default:
		out.Table = plainTable(*b.Table)
	}
	if b.ID == "rfp:project:information" && len(out.Table.Rows) == 0 {
		// As on screen: an empty transmittal says so.
		out.Table.Rows = []Row{{MinHeight: rowHeight, Cells: []Cell{{Runs: text("No documents in this transmittal."), ColSpan: len(out.Table.Columns)}}}}
	}
	return out
}

func text(s string) []Run         { return []Run{{Text: s}} }
func bold(s string) []Run         { return []Run{{Text: s, Bold: true}} }
func head(s string) Cell          { return Cell{Runs: bold(strings.ToUpper(s)), Head: true} }
func plain(s string) Cell         { return Cell{Runs: text(s)} }
func centred(c Cell) Cell         { c.Align = Center; return c }
func spanRows(c Cell, n int) Cell { c.RowSpan = n; return c }

// stageTable is the scope by stage as an open stage reads on screen: the
// stage name (and any note) and its one fee cell run down beside its lines,
// each line ticked In or Excluded. A stage left out is one row.
func stageTable(t reports.Table, stageRows int) *Table {
	priced := len(t.Columns) > 3
	if stageRows <= 0 || stageRows > len(t.Rows) {
		stageRows = len(t.Rows)
	}
	out := &Table{Columns: []Column{{Share: 21}, {Share: 79 - 11*b2f(priced)}, {Fixed: tickWidth, Align: Center}, {Fixed: tickWidth, Align: Center}}}
	out.Head = []Cell{head("Stage"), head("Scope"), centred(head("In")), centred(head("Excl."))}
	if priced {
		out.Columns = append(out.Columns, Column{Share: 11})
		out.Head = append(out.Head, head("Fee $"))
	}
	fee := func(row []string) string {
		if priced && len(row) > 3 {
			return row[3]
		}
		return ""
	}
	for i := 0; i < stageRows; {
		stage := t.Rows[i]
		lines := [][]string{}
		for j := i + 1; j < stageRows && t.Rows[j][0] == ""; j++ {
			lines = append(lines, t.Rows[j])
		}
		i += 1 + len(lines)
		name := Cell{Runs: bold(stage[0])}
		if len(lines) == 0 {
			// A stage left out, or one with nothing listed: one row.
			row := Row{Strong: true, MinHeight: rowHeight, Cells: []Cell{name, plain(stage[1])}}
			row.Cells = append(row.Cells, ticks(stage[2])...)
			if priced {
				row.Cells = append(row.Cells, plain(fee(stage)))
			}
			out.Rows = append(out.Rows, row)
			continue
		}
		if note := strings.TrimSpace(stage[1]); note != "" {
			name.Runs = append(name.Runs, Run{Text: "\n" + note, Muted: true})
		}
		for k, line := range lines {
			row := Row{MinHeight: rowHeight}
			if k == 0 {
				row.Strong = true
				row.Cells = append(row.Cells, spanRows(name, len(lines)))
			}
			row.Cells = append(row.Cells, plain(line[1]))
			row.Cells = append(row.Cells, ticks(line[2])...)
			if k == 0 && priced {
				row.Cells = append(row.Cells, spanRows(plain(fee(stage)), len(lines)))
			}
			out.Rows = append(out.Rows, row)
		}
	}
	// Reimbursables and the total: a label and what it covers.
	for _, r := range t.Rows[stageRows:] {
		row := Row{Strong: true, MinHeight: rowHeight, Cells: []Cell{{Runs: bold(r[0])}, plain(r[1]), {}, {}}}
		if priced {
			row.Cells = append(row.Cells, plain(fee(r)))
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// ticks are the In and Excluded boxes for a printed position.
func ticks(position string) []Cell {
	switch position {
	case "Included":
		return []Cell{{Tick: TickOn, Align: Center}, {Tick: TickOff, Align: Center}}
	case "Excluded":
		return []Cell{{Tick: TickOff, Align: Center}, {Tick: TickOn, Align: Center}}
	case "":
		return []Cell{{}, {}}
	}
	return []Cell{{Runs: text(position), Align: Center, ColSpan: 2}}
}

// termLines splits the terms wording into its numbered terms: one per line,
// any list marker removed. The screen numbers the same lines.
func termLines(text string) []string {
	out := []string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func termsTable(text string) *Table {
	out := &Table{Columns: []Column{{Fixed: tickWidth, Align: Center}, {Share: 100}}}
	out.Head = []Cell{centred(head("No.")), head("Term")}
	for i, line := range termLines(text) {
		out.Rows = append(out.Rows, Row{MinHeight: rowHeight, Cells: []Cell{centred(plain(strconv.Itoa(i + 1))), plain(line)}})
	}
	return out
}

// allowanceTable keeps the two pairs, each item's detail quieter after its
// name, as on screen.
func allowanceTable(t reports.Table) *Table {
	out := &Table{Columns: []Column{{Share: 50}, {Fixed: qtyWidth, Align: Center}, {Share: 50}, {Fixed: qtyWidth, Align: Center}}}
	out.Head = []Cell{head("Item"), centred(head("Qty")), head("Item"), centred(head("Qty"))}
	item := func(s string) Cell {
		label, detail, ok := strings.Cut(s, ": ")
		if !ok {
			return plain(s)
		}
		return Cell{Runs: []Run{{Text: label + " "}, {Text: detail, Muted: true}}}
	}
	for _, r := range t.Rows {
		row := Row{MinHeight: rowHeight}
		for c, v := range r {
			if c%2 == 0 {
				row.Cells = append(row.Cells, item(v))
			} else {
				row.Cells = append(row.Cells, centred(plain(v)))
			}
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// gridTable is page one as the Profile card lays it out: each group a blue
// head over label and value columns.
func gridTable(t reports.Table) *Table {
	out := &Table{}
	for _, c := range t.Columns {
		out.Columns = append(out.Columns, Column{Share: 15}, Column{Share: 18})
		h := head(c)
		h.ColSpan = 2
		out.Head = append(out.Head, h)
	}
	for _, r := range t.Rows {
		row := Row{MinHeight: rowHeight}
		for _, v := range r {
			label, value, ok := strings.Cut(v, ": ")
			if !ok {
				label, value = "", v
			}
			row.Cells = append(row.Cells, Cell{Runs: []Run{{Text: label, Muted: true}}}, plain(value))
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// scheduleTogetherRows: a schedule table this short (the totals, the
// approval, a few payments) is not split over a page.
const scheduleTogetherRows = 12

// scheduleTable is a payment schedule's table: words on the left, figures
// right aligned under right-aligned heads, a total in bold under a strong
// rule, and a reason as one quiet cell across the table. A line's note
// ("label\nnote") sits under its label.
func scheduleTable(t reports.Table, m basis) *Table {
	left := max(m.Left, 1)
	out := &Table{Together: len(t.Rows) <= scheduleTogetherRows}
	align := func(c int) Align {
		if c >= left {
			return Right
		}
		return Left
	}
	for c, name := range t.Columns {
		col := Column{Share: 1}
		if c < len(m.Widths) && m.Widths[c] > 0 {
			col.Fixed = m.Widths[c]
		}
		out.Columns = append(out.Columns, col)
		h := head(name)
		h.Align = align(c)
		out.Head = append(out.Head, h)
	}
	for i, r := range t.Rows {
		row := Row{MinHeight: rowHeight}
		if slices.Contains(m.Notes, i) {
			words := ""
			if len(r) > 0 {
				words = r[0]
			}
			row.Cells = []Cell{{Runs: []Run{{Text: words, Muted: true}}, ColSpan: len(t.Columns)}}
			out.Rows = append(out.Rows, row)
			continue
		}
		total := slices.Contains(m.Bold, i)
		row.Strong = total
		if slices.Contains(m.Tall, i) {
			row.MinHeight = 2.5 * rowHeight
		}
		value := func(c int) string {
			if c < len(r) {
				return r[c]
			}
			return ""
		}
		for c := 0; c < len(t.Columns); c++ {
			label, note, noted := strings.Cut(value(c), "\n")
			cell := Cell{Align: align(c), Runs: []Run{{Text: label, Bold: total}}}
			if noted {
				cell.Runs = append(cell.Runs, Run{Text: "\n" + note, Muted: true})
			}
			if c == 0 && m.Merge {
				// Never over the last column: that is where the figure is.
				for c+1 < len(t.Columns)-1 && value(c+1) == "" {
					c++
					cell.ColSpan++
				}
				if cell.ColSpan > 0 {
					cell.ColSpan++
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// plainTable is any other document table, its columns shared by how much
// each holds.
func plainTable(t reports.Table) *Table {
	out := &Table{}
	for c, name := range t.Columns {
		longest := len(name)
		for _, r := range t.Rows {
			if c < len(r) {
				longest = max(longest, min(len(r[c]), 80))
			}
		}
		out.Columns = append(out.Columns, Column{Share: float64(max(longest, 6))})
		out.Head = append(out.Head, head(name))
	}
	for _, r := range t.Rows {
		row := Row{MinHeight: rowHeight}
		for c := range t.Columns {
			v := ""
			if c < len(r) {
				v = r[c]
			}
			row.Cells = append(row.Cells, plain(v))
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// golden says the sections are the project report in the owner's format:
// its first section is the cover table.
func golden(sections []reports.Section) bool {
	for _, s := range sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID == "cover" {
				return true
			}
		}
	}
	return false
}

// fromGolden lays out the project report as the owner's own: a cover page
// from the cover table, a contents page, then the numbered sections.
func fromGolden(title, date string, sections []reports.Section, images map[string][]byte) Doc {
	d := Doc{Title: title, Date: date, Contents: true}
	for _, s := range sections {
		if s.Internal {
			continue
		}
		if s.ID == "cover" {
			c := &Cover{Title: title}
			for _, b := range s.Blocks {
				if b.Table == nil {
					continue
				}
				for _, row := range b.Table.Rows {
					if len(row) < 2 {
						continue
					}
					if row[0] == "Title" {
						c.Title = row[1]
						continue
					}
					if refs := ImageRefs(row[1]); len(refs) > 0 {
						if img, ok := decodeImage(images[refs[0][1]], refs[0][0]); ok {
							c.Image = &img
						}
						continue
					}
					c.Lines = append(c.Lines, [2]string{row[0], row[1]})
				}
			}
			d.Cover = c
			continue
		}
		if len(s.Blocks) == 0 {
			continue
		}
		out := Section{Title: s.Title}
		for _, b := range s.Blocks {
			eb := block(b)
			if eb.Table != nil && b.Table != nil && b.Table.Meta != nil {
				eb.Table = reportTableImages(*b.Table, images)
			}
			// A table named as its section is (07 Site activity) needs no
			// second heading.
			if strings.HasSuffix(strings.ToLower(s.Title), strings.ToLower(eb.Heading)) {
				eb.Heading = ""
			}
			out.Blocks = append(out.Blocks, eb)
		}
		d.Sections = append(d.Sections, out)
	}
	return d
}

// reportTable prints one of the report's tables: a placeholder grey (and
// italic in Word), a new row or changed cell blue, a done or key cell
// filled, a line "- x" as a bullet and "**x**" in bold.
func reportTable(t reports.Table) *Table { return reportTableImages(t, nil) }

func reportTableImages(t reports.Table, images map[string][]byte) *Table {
	out := plainTable(reports.Table{Columns: t.Columns, Rows: t.Rows})
	// No column so narrow that a word like "Approved" or a date breaks.
	for c := range out.Columns {
		out.Columns[c].Share = max(out.Columns[c].Share, 12)
	}
	for i := range out.Rows {
		f := reports.RowFacts{}
		if i < len(t.Meta) {
			f = t.Meta[i]
		}
		for c := range out.Rows[i].Cells {
			text := ""
			if c < len(t.Rows[i]) {
				text = t.Rows[i][c]
			}
			blue := f.New || c < len(f.Changed) && f.Changed[c]
			grey := f.Kind == reports.RowPlaceholder
			words, imgs := splitImages(text, images)
			cell := Cell{Runs: cellRuns(words, blue, grey), Images: imgs}
			if c < len(f.Tones) {
				cell.Fill = f.Tones[c]
			}
			out.Rows[i].Cells[c] = cell
		}
	}
	return out
}

// cellRuns turns a cell's few marks into runs.
func cellRuns(text string, blue, grey bool) []Run {
	var runs []Run
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			runs = append(runs, Run{Text: "\n"})
		}
		bold := false
		if strings.HasPrefix(line, "- ") {
			line = "• " + line[2:]
		}
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") && len(line) > 4 {
			line, bold = line[2:len(line)-2], true
		}
		runs = append(runs, Run{Text: line, Bold: bold, Blue: blue, Muted: grey, Italic: grey})
	}
	return runs
}

var imageLine = regexp.MustCompile(`^!\[([^\]]{0,200})\]\(image:([0-9a-f]{64})\)$`)

// ImageRefs are a cell's image lines: caption and SHA-256 hex each.
func ImageRefs(text string) [][2]string {
	var out [][2]string
	for _, line := range strings.Split(text, "\n") {
		if m := imageLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			out = append(out, [2]string{m[1], m[2]})
		}
	}
	return out
}

// splitImages takes a cell's image lines out of its words. An image with
// no bytes, or bytes that do not decode, stays as its caption.
func splitImages(text string, images map[string][]byte) (string, []Image) {
	var words []string
	var out []Image
	for _, line := range strings.Split(text, "\n") {
		m := imageLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			words = append(words, line)
			continue
		}
		if img, ok := decodeImage(images[m[2]], m[1]); ok {
			out = append(out, img)
			continue
		}
		words = append(words, strings.TrimSpace(m[1]+" (image not available)"))
	}
	return strings.Join(words, "\n"), out
}

func decodeImage(data []byte, caption string) (Image, bool) {
	if len(data) == 0 {
		return Image{}, false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return Image{}, false
	}
	return Image{Data: data, Caption: caption, W: cfg.Width, H: cfg.Height}, true
}
