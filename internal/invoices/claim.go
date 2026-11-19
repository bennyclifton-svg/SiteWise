package invoices

import (
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/identity"
)

// Progress claims (docs/plans/2026-11-17-variations-progress-claims.md,
// Tasks 5 to 7). A claim is an invoice of kind progress_claim: it arrives,
// is stored and is read the same way, and adds its table's rows. Go reads
// the table, maps rows by rule, checks the arithmetic and works out every
// figure; Jev is asked only which contract line an unplaced row charges
// for, in the claim's one fan-out. Jev is not a calculator
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13).

// KindProgressClaim is the ledger kind of a progress claim.
const KindProgressClaim = "progress_claim"

// Claim-level fields, stored as invoice decisions beside number, date and
// total. Dates are YYYY-MM-DD; retention is cents.
const (
	FieldReferenceDate    = "reference_date"
	FieldServedOn         = "served_on"
	FieldRetention        = "retention"
	FieldScheduleServedOn = "schedule_served_on"
)

// Claim row kinds.
const (
	RowContract  = "contract"
	RowVariation = "variation"
	RowRetention = "retention"
)

// MaxClaimRows bounds a claim's table; Jev is asked about at most this many
// rows (plan risk 4: the fall-back is a lower cap, never a second call).
const MaxClaimRows = 80

// ClaimRow is one row of a claim's table: the builder's three amounts as
// printed, what it charges for, and the PM's assessment to date.
type ClaimRow struct {
	ID          string
	Ordinal     int
	Kind        string
	Label       string
	Page        int
	LineID      string // a contract row's cost line
	VariationID string // a variation row's variation
	MappedBy    string // rule, jev or user
	Band        string
	ToDate      *int64
	Previous    *int64
	ThisClaim   *int64
	Assessed    *int64 // the PM's assessed to date; nil is not assessed
	Reason      string
	Version     int64
}

// Claimed is what the row claims this time: as printed, else to date less
// previous.
func (r ClaimRow) Claimed() *int64 {
	if r.ThisClaim != nil {
		return r.ThisClaim
	}
	if r.ToDate != nil && r.Previous != nil {
		n := *r.ToDate - *r.Previous
		return &n
	}
	return nil
}

// ClaimContract is a contract a claim may be on: its lines in contract
// order and its variations, the options a row maps to.
type ClaimContract struct {
	PackageID  string
	Lines      []Line
	Variations []ClaimVariation
}

// ClaimVariation is a variation a claim row may charge for.
type ClaimVariation struct {
	ID, Number, Title, Status string
}

// claimTable is a claim's table as printed.
type claimTable struct {
	rows  []ClaimRow
	gross *int64 // the printed total of this claim's rows, when found
	found bool
}

var (
	toDateWords   = []string{"to date", "claimed to date", "total to date", "cumulative", "completed to date", "value to date"}
	previousWords = []string{"previous", "previously", "less previous", "prior"}
	thisWords     = []string{"this claim", "this period", "this month", "this payment", "current claim"}
	totalLabel    = regexp.MustCompile(`(?i)^\s*(sub-?\s*)?total|gross|amount claimed`)
	retentionWord = regexp.MustCompile(`(?i)\bretention\b`)
	variationNo   = regexp.MustCompile(`(?i)\b(?:variation|var\.?|vo)?\s*(V\s*-?\s*\d{1,4})\b`)
)

// gridCell is one cell of the claim's table with the column it sits in.
type gridCell struct {
	col  int
	x    float64
	text string
}

type gridRow struct {
	page  int
	cells []gridCell
}

// claimGrid lays the text out as rows of cells. A spreadsheet or Word table
// says where each cell is; a Markdown table splits on its bars; a PDF row
// is its baseline, with columns placed later by the header's positions.
func claimGrid(text identity.Text) []gridRow {
	type key struct {
		sheet string
		table int
		row   int
	}
	placed := map[key]*gridRow{}
	var order []key
	var out []gridRow
	structured := false
	for _, r := range text.Runs {
		if r.Source.Row > 0 && r.Source.Col > 0 && (r.Source.Sheet != "" || r.Source.Table > 0) {
			structured = true
			k := key{r.Source.Sheet, r.Source.Table, r.Source.Row}
			g, ok := placed[k]
			if !ok {
				g = &gridRow{page: 1}
				placed[k] = g
				order = append(order, k)
			}
			g.cells = append(g.cells, gridCell{col: r.Source.Col, text: strings.TrimSpace(r.Text)})
		}
	}
	if structured {
		for _, k := range order {
			g := *placed[k]
			sort.SliceStable(g.cells, func(i, j int) bool { return g.cells[i].col < g.cells[j].col })
			out = append(out, g)
		}
		return out
	}
	for _, r := range rowsOf(text) {
		g := gridRow{page: r.page}
		if len(r.cells) == 1 && strings.Count(r.cells[0].text, "|") >= 2 {
			parts := strings.Split(strings.Trim(r.cells[0].text, " |"), "|")
			for i, p := range parts {
				g.cells = append(g.cells, gridCell{col: i + 1, text: strings.TrimSpace(p)})
			}
		} else {
			for _, c := range r.cells {
				g.cells = append(g.cells, gridCell{x: c.x + c.w/2, text: c.text})
			}
		}
		out = append(out, g)
	}
	return out
}

func hasAny(s string, words []string) bool {
	s = strings.ToLower(s)
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// cellAmount reads one cell as an amount: digits with cents or a dollar
// sign, or plain digits with thousands separators.
func cellAmount(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "—" {
		return 0, false
	}
	if m := moneyIn(s); len(m) == 1 && strings.TrimSpace(strings.Replace(s, m[0].display, "", 1)) == "" {
		return m[0].cents, true
	}
	c, ok, err := ParseCents(s)
	if err != nil || !ok {
		return 0, false
	}
	return c, true
}

// readClaimTable finds the table whose header names to date, previous and
// this claim, and reads each row under it with two or more amounts.
func readClaimTable(text identity.Text) claimTable {
	grid := claimGrid(text)
	for h, hr := range grid {
		var line strings.Builder
		for _, c := range hr.cells {
			line.WriteString(c.text + " | ")
		}
		head := line.String()
		if !hasAny(head, toDateWords) || !hasAny(head, previousWords) || !hasAny(head, thisWords) {
			continue
		}
		// Column positions: the header cells naming each amount.
		find := func(words []string, not ...[]string) (gridCell, bool) {
			for _, c := range hr.cells {
				if hasAny(c.text, words) {
					clash := false
					for _, n := range not {
						clash = clash || hasAny(c.text, n)
					}
					if !clash {
						return c, true
					}
				}
			}
			return gridCell{}, false
		}
		prev, ok1 := find(previousWords)
		this, ok2 := find(thisWords)
		toDate, ok3 := find(toDateWords, previousWords, thisWords)
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		// Every header cell is a column, so an amount under another column
		// (the contract value, a percentage) is not taken for one of these.
		heads := map[string]gridCell{"to": toDate, "prev": prev, "this": this}
		for i, c := range hr.cells {
			if c != toDate && c != prev && c != this {
				heads["other"+itoa(i)] = c
			}
		}
		t := claimTable{found: true}
		for _, r := range grid[h+1:] {
			label, amounts := "", map[string]int64{}
			var placedAny bool
			for _, c := range r.cells {
				a, isAmount := cellAmount(c.text)
				col := columnOf(c, heads)
				if isAmount && (col == "to" || col == "prev" || col == "this") {
					amounts[col] = a
					placedAny = true
					continue
				}
				if isAmount && col != "" {
					continue
				}
				if !isAmount && label == "" && strings.TrimSpace(c.text) != "" {
					label = strings.TrimSpace(c.text)
				}
			}
			if !placedAny {
				continue
			}
			if totalLabel.MatchString(label) && !retentionWord.MatchString(label) {
				if v, ok := amounts["this"]; ok {
					t.gross = &v
				}
				break
			}
			if len(amounts) < 2 && !retentionWord.MatchString(label) {
				continue
			}
			row := ClaimRow{Ordinal: len(t.rows), Kind: RowContract, Label: clipLabel(label), Page: r.page, MappedBy: DecidedByRule, Band: BandBlank}
			if v, ok := amounts["to"]; ok {
				row.ToDate = &v
			}
			if v, ok := amounts["prev"]; ok {
				row.Previous = &v
			}
			if v, ok := amounts["this"]; ok {
				row.ThisClaim = &v
			}
			t.rows = append(t.rows, row)
			if len(t.rows) == MaxClaimRows {
				break
			}
		}
		return t
	}
	return claimTable{}
}

// columnOf places a cell under a header cell: by column number when the
// layout gives one, else by the nearest header centre within a cell width.
func columnOf(c gridCell, heads map[string]gridCell) string {
	best, dist := "", 0.0
	for name, h := range heads {
		if c.col > 0 || h.col > 0 {
			if c.col == h.col {
				return name
			}
			continue
		}
		d := c.x - h.x
		if d < 0 {
			d = -d
		}
		if best == "" || d < dist {
			best, dist = name, d
		}
	}
	if best != "" && dist > 90 {
		return ""
	}
	return best
}

func clipLabel(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

func labelKey(s string) string {
	return strings.Trim(nonWord.ReplaceAllString(strings.ToLower(s), " "), " ")
}

// mapClaimRows places each row by rule: a retention row by its word, a
// variation by its printed number, a contract line by its exact label. A
// label that matches two lines is left for Jev or the PM.
func mapClaimRows(rows []ClaimRow, c ClaimContract) {
	byLabel := map[string][]string{}
	for _, l := range c.Lines {
		byLabel[labelKey(l.Label)] = append(byLabel[labelKey(l.Label)], l.ID)
	}
	byNumber := map[string]string{}
	for _, v := range c.Variations {
		byNumber[VariationNumberKey(v.Number)] = v.ID
	}
	for i := range rows {
		r := &rows[i]
		switch {
		case retentionWord.MatchString(r.Label):
			r.Kind, r.Band = RowRetention, BandGreen
		case variationNo.MatchString(r.Label) && byNumber[VariationNumberKey(variationNo.FindStringSubmatch(r.Label)[1])] != "":
			r.Kind, r.VariationID, r.Band = RowVariation, byNumber[VariationNumberKey(variationNo.FindStringSubmatch(r.Label)[1])], BandGreen
		case len(byLabel[labelKey(r.Label)]) == 1:
			r.LineID, r.Band = byLabel[labelKey(r.Label)][0], BandGreen
		}
	}
}

// VariationNumberKey reads V3, V03 and V-003 as one number.
func VariationNumberKey(n string) string {
	n = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(n))
	head := strings.TrimRight(n, "0123456789")
	digits := strings.TrimLeft(n[len(head):], "0")
	return head + digits
}

// ContractRows are a claim's rows when nothing could be read: one per
// contract line and a retention row, amounts unknown, for the PM to type.
func ContractRows(c ClaimContract) []ClaimRow {
	var out []ClaimRow
	for _, l := range c.Lines {
		out = append(out, ClaimRow{Ordinal: len(out), Kind: RowContract, Label: l.Label, LineID: l.ID, MappedBy: DecidedByRule, Band: BandGreen})
	}
	return append(out, ClaimRow{Ordinal: len(out), Kind: RowRetention, Label: "Retention", MappedBy: DecidedByRule, Band: BandGreen})
}

// withRetention adds an empty retention row when the table printed none, so
// the PM can always assess what is withheld.
func withRetention(rows []ClaimRow) []ClaimRow {
	for _, r := range rows {
		if r.Kind == RowRetention {
			return rows
		}
	}
	return append(rows, ClaimRow{Ordinal: len(rows), Kind: RowRetention, Label: "Retention", MappedBy: DecidedByRule, Band: BandGreen})
}

// claimWords is the words-plus-ABN rule: "progress claim" or "payment
// claim" printed, and the supplier settled by rule to a package with a live
// works contract.
func claimWords(words []string) bool {
	for _, w := range words {
		if w == "progress claim" || w == "payment claim" {
			return true
		}
	}
	return false
}

// shortAmount is an amount as Jev reads it in a row's question.
func shortAmount(p *int64) string {
	if p == nil {
		return "not printed"
	}
	return DisplayCents(*p)
}
