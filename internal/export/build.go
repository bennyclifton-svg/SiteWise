package export

import (
	"encoding/json"
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
}

// FromSections lays out a draft's sections as the screen shows them.
// Internal sections (the project manager's review lists) are left out, as
// they are when an RFP is issued.
func FromSections(title, subtitle, date string, sections []reports.Section) Doc {
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
	switch {
	case meta.Layout == "stage_scope":
		out.Table = stageTable(*b.Table, meta.StageRows)
	case meta.Layout == "allowance_pairs" && len(b.Table.Columns) == 4:
		out.Table = allowanceTable(*b.Table)
	case b.ID == "rfp:project:grid":
		out.Table = gridTable(*b.Table)
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
