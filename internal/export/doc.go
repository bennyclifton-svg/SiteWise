// Package export writes a report draft as a PDF or a Word document that
// looks like the report on screen (owner direction, 8 November 2026): the
// same fonts, blue section bands, table heads, merged stage and fee cells,
// two-pair allowances and tick columns. Both writers read one Doc built from
// the draft's sections, so the PDF and the Word file always agree. Word lays
// out its own pages, so its page breaks can differ from the PDF's.
//
// Issued RFPs keep their frozen PDF from internal/reports/render; this
// package only exports the current draft.
package export

// Doc is a report laid out for print.
type Doc struct {
	Title    string
	Subtitle string
	Date     string // footer, YYYY-MM-DD
	Sections []Section
}

// Section starts with a blue band carrying its title.
type Section struct {
	Title  string
	Blocks []Block
}

// Block is a paragraph group, a table, or both. Label leads the text in
// bold; TextAfter prints the text under the table (the transmittal's note).
type Block struct {
	Label     string
	Text      string // paragraphs separated by "\n"
	Table     *Table
	TextAfter bool
}

// Table is laid out as on screen: a blue head row, then rows whose cells
// may span rows or columns, HTML style (a spanned position has no cell).
type Table struct {
	Columns []Column
	Head    []Cell // one per column unless a head cell spans columns
	Rows    []Row
}

// Column width is a share of the table width when Fixed is zero, else a
// fixed width in points.
type Column struct {
	Share float64
	Fixed float64
	Align Align
}

type Align int

const (
	Left Align = iota
	Center
)

// Row is one table row. Strong draws a heavier rule above it (a stage's
// first row); MinHeight keeps one-line rows uniform, in points.
type Row struct {
	Cells     []Cell
	Strong    bool
	MinHeight float64
}

// Cell holds styled runs of text, or a tick box.
type Cell struct {
	Runs    []Run
	Tick    Tick
	Align   Align
	RowSpan int
	ColSpan int
	// Fill tints the cell as the screen's selected or head cells are.
	Head bool
}

type Tick int

const (
	NoTick Tick = iota
	TickOn
	TickOff
)

// Run is text in one style. "\n" in Text starts a new line.
type Run struct {
	Text  string
	Bold  bool
	Muted bool
}

func span(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// Colours from the screen (web/src/table-heads.css, clerk-colours.css).
type rgb struct{ r, g, b uint8 }

var (
	colourText   = rgb{0x2C, 0x24, 0x1E} // --sw-text-primary
	colourMuted  = rgb{0x50, 0x47, 0x40} // --sw-text-secondary
	colourBand   = rgb{0xEA, 0xEE, 0xF2} // head tone 9% on the surface
	colourHead   = rgb{0x2D, 0x51, 0x82} // head tone 78% on the text colour
	colourLine   = rgb{0xE2, 0xDD, 0xD4} // --sw-border-default
	colourStrong = rgb{0x88, 0x7E, 0x74} // --sw-border-strong
	colourTick   = rgb{0x1F, 0x5F, 0x73} // a ticked box
)

func (c rgb) hex() string {
	const digits = "0123456789ABCDEF"
	b := []byte{digits[c.r>>4], digits[c.r&15], digits[c.g>>4], digits[c.g&15], digits[c.b>>4], digits[c.b&15]}
	return string(b)
}
