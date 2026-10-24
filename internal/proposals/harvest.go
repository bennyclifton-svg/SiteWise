// Package proposals reads fee proposals and tender submissions into a
// package's returnable schedule. Code harvests every amount with the words
// printed around it, the firm's details and the proposal's sections, settles
// what rules can, and computes every figure; Jev only picks among the
// harvested candidates, in one fan-out per proposal.
//
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
//	https://docs.typesafe.ai/concepts/how-to-build-with-system-one
package proposals

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/identity"
	"sitewise/internal/invoices"
	"sitewise/internal/rfp"
)

// ExtractLimits reads a proposal far enough for its fee schedule: fee tables
// sit after the methodology, often on pages 4 to 10, and a response form
// holds up to twelve sheets.
func ExtractLimits() identity.Limits {
	l := identity.DefaultLimits()
	l.MaxPages = 60
	l.ReadControlPage = false
	l.MaxRuns = 20000
	l.MaxSheets = 12
	l.MaxRows = 400
	l.MaxCols = 8
	return l
}

// Line is one printed line: the runs on one baseline of a page, left to
// right, or one cell row of a workbook or Word table.
type Line struct {
	Heading     bool
	HeadingText string
	Page        int
	Text        string
	Cells       []string
	// Cols are the 1-based columns of Cells for a workbook or table row.
	Cols []int
	// X and W are each cell's left edge and width in PDF points; nil for a
	// workbook or Word table.
	X, W  []float64
	Sheet string
	Row   int
}

// Lines groups runs printed on one baseline, top to bottom. Runs without
// geometry (DOCX, XLSX) group by table row or sheet row.
func Lines(text identity.Text) []Line {
	var out []Line
	byPage := map[int][]identity.Run{}
	var pages []int
	type rowKey struct {
		sheet string
		table int
		row   int
	}
	var keyed []rowKey
	byKey := map[rowKey][]string{}
	colsOf := map[rowKey][]int{}
	for _, r := range text.Runs {
		t := strings.TrimSpace(strings.Join(strings.Fields(r.Text), " "))
		if t == "" {
			continue
		}
		src := r.Source
		if src.Width == 0 && src.X == 0 && src.Y == 0 {
			if src.Row > 0 {
				k := rowKey{src.Sheet, src.Table, src.Row}
				if _, ok := byKey[k]; !ok {
					keyed = append(keyed, k)
				}
				byKey[k] = append(byKey[k], t)
				colsOf[k] = append(colsOf[k], src.Col)
				continue
			}
			out = append(out, Line{Page: src.Page, Text: t, Cells: []string{t}, Heading: src.Heading})
			continue
		}
		if _, ok := byPage[src.Page]; !ok {
			pages = append(pages, src.Page)
		}
		r.Text = t
		byPage[src.Page] = append(byPage[src.Page], r)
	}
	for _, k := range keyed {
		cells := byKey[k]
		out = append(out, Line{Text: strings.Join(cells, "  "), Cells: cells, Cols: colsOf[k], Sheet: k.sheet, Row: k.row})
	}
	sort.Ints(pages)
	for _, p := range pages {
		runs := byPage[p]
		sort.SliceStable(runs, func(i, j int) bool {
			if math.Abs(runs[i].Source.Y-runs[j].Source.Y) > 3 {
				return runs[i].Source.Y > runs[j].Source.Y
			}
			return runs[i].Source.X < runs[j].Source.X
		})
		var cur *Line
		var y float64
		for _, r := range runs {
			if cur != nil && math.Abs(y-r.Source.Y) <= 3 {
				cur.Cells = append(cur.Cells, r.Text)
				cur.X = append(cur.X, r.Source.X)
				cur.W = append(cur.W, r.Source.Width)
				cur.Text += "  " + r.Text
				continue
			}
			out = append(out, Line{Page: p, Text: r.Text, Cells: []string{r.Text}, X: []float64{r.Source.X}, W: []float64{r.Source.Width}})
			cur, y = &out[len(out)-1], r.Source.Y
		}
	}
	return out
}

// Amount is one money figure and the words around it. ID is stable within
// one proposal ("a1", "a2", ...). Context is its own line; Under is the
// nearest heading above it, which in a fee proposal usually names the task.
type Amount struct {
	ID      string `json:"id"`
	Cents   int64  `json:"-"`
	Display string `json:"amount"`
	Context string `json:"printed_with"`
	Under   string `json:"under,omitempty"`
	Before  string `json:"line_above,omitempty"`
	Page    int    `json:"page,omitempty"`
	// PerHour is a rate: "$250/hour", "per hour" or a line in a rates list.
	PerHour bool `json:"per_hour,omitempty"`
	// Total is printed with "total" or "lump sum".
	Total bool `json:"total,omitempty"`
	// GST is "inc" or "ex" when the amount's own line says so.
	GST string `json:"gst,omitempty"`
	// Label is the row's own words: the nearest text to the left of the
	// figure. Column is the table heading above it, matched by position.
	Label  string `json:"row_label,omitempty"`
	Column string `json:"column,omitempty"`
	// Unit is a unit rate whose extended amount is on the same line; GSTPart
	// is a GST amount; IncTwin is the GST-inclusive copy of another figure.
	// None of them is a fee.
	Unit    bool `json:"-"`
	GSTPart bool `json:"-"`
	IncTwin bool `json:"-"`
	// Percent is a fee stated as a percentage of a cost, in basis points.
	PercentBP int      `json:"-"`
	Section   int      `json:"section"`
	Line      int      `json:"-"`
	Kind      string   `json:"-"` // "", "percent"
	Breakdown []string `json:"printed_breakdown_headings,omitempty"`
	RepeatOf  string   `json:"repeat_of,omitempty"`
}

// Section is the proposal split at its headings.
type Section struct {
	Ordinal int
	Heading string
	Page    int
	Body    string
}

// Evidence is everything harvested from one proposal. It decides nothing.
type Evidence struct {
	OCR        bool
	Incomplete bool
	Filename   string
	TextLayer  bool
	Readable   bool
	Amounts    []Amount
	// GSTWords counts "excl. GST" and "incl. GST" wording across the text.
	GSTEx, GSTInc int
	ABNs          []string
	Emails        []string
	Phones        []string
	Names         []string
	// Printed is how many of Names were printed in the text; the rest come
	// from the filename and email domains, which are Hints.
	Printed   int
	Hints     []string
	Addresses []string
	// Letterhead is the firm name and office address printed beside the
	// proposal's one ABN, outside any addressee block; nil when unclear.
	Letterhead *Letterhead
	// Dated is the proposal's own header date, YYYY-MM-DD, or "".
	Dated    string
	Contacts []string
	Sections []Section
	// Form holds a SiteWise response form's cells by schedule key, read
	// verbatim; FormFirm its labelled firm fields.
	Form     map[string]string
	FormFirm map[string]string
	// KindWords are the proposal cues printed in the filename or a heading.
	KindWords []string
	// OptionsAB is true when the proposal prices "Option A" and "Option B".
	OptionsAB bool
	Excerpt   string
}

var (
	// A figure is money when it has a dollar sign, or thousands groups with cents.
	moneyRe   = regexp.MustCompile(`(?i)(\$\s?|AUD\s?)(\d{1,3}(?:(?:, ?| )\d{3})+|\d+)(\.\d{2})?(\s?(?:k|m|million)\b)?|\b(\d{1,3}(?:,\d{3})+)\.(\d{2})\b`)
	percentRe = regexp.MustCompile(`(?i)\b(\d{1,2}(?:\.\d{1,2})?)\s?%\s+(?:of\s+(?:the\s+)?)?(?:total\s+)?(?:construction|build|building|contract|project|works)\s+(?:cost|value|sum|budget)`)
	perHourRe = regexp.MustCompile(`(?i)^\s*(?:\+\s?GST\s*)?(?:/|per\s+|p/?)\s?(?:h\b|hr\b|hrs\b|hour|hourly)|^\s*(?:ph|p\.h\.)\b`)
	exGSTRe   = regexp.MustCompile(`(?i)(?:\b(?:excl?\.?(?:uding|usive of)?|ex\.?|exclusive of|plus(?: 10%)?)\s?|\+\s?(?:10%\s)?)GST\b|\bGST\s+(?:excl?\.?|exclusive)`)
	incGSTRe  = regexp.MustCompile(`(?i)\b(?:incl?\.?(?:uding|usive of)?|inclusive of)\s?GST\b|\bGST\s+incl`)
	emailRe   = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	phoneRe   = regexp.MustCompile(`(?:\+61\s?\(?0?\)?\s?[2-478]|\(0[2-478]\)|\b0[2-478])[\s\-]?\d{4}[\s\-]?\d{4}\b|\b04\d{2}[\s\-]?\d{3}[\s\-]?\d{3}\b|\b1[38]00[\s\-]?\d{3}[\s\-]?\d{3}\b`)
	addressRe = regexp.MustCompile(`(?i)\b(?:(?:level|suite|unit|shop)\s+\d+[a-z]?,?\s+)?\d{1,5}[a-z]?(?:-\d{1,5})?\s+[a-z][a-z' ]{1,40}\s(?:street|st|road|rd|avenue|ave|parade|pde|drive|dr|lane|ln|place|pl|highway|hwy|terrace|tce|crescent|cres|boulevard|blvd|way|circuit|cct|close|court|ct)\b[.,]?[^|]{0,60}?\b(?:NSW|VIC|QLD|SA|WA|TAS|ACT|NT)\b,?\s*\d{4}\b`)
	companyRe = regexp.MustCompile(`(?i)\b(?:pty\.?\s*ltd|pty\.?\s*limited|limited|ltd|group|consulting|consultants|partners|associates|architects?|architecture|engineers?|engineering|planning|planners|surveyors?|surveying|studio|design|geotechnic\w*|arbor\w*|landscape\w*)\b`)
	signoffRe = regexp.MustCompile(`(?i)^(?:yours\s+(?:sincerely|faithfully)|kind\s+regards|regards|best\s+regards|warm\s+regards|sincerely)\b`)
	headingRe = regexp.MustCompile(`(?i)^(?:\d{1,2}(?:\.\d{1,2}){0,2}\.?\s+\S|(?:task|stage|phase|part|section|item|schedule|appendix|option)\s+[0-9a-z]{1,3}\b)`)
	abnGlued  = regexp.MustCompile(`(?i)ABN[:#]?\s?(\d)`)
	abnRe     = regexp.MustCompile(`\b\d{2}\s?\d{3}\s?\d{3}\s?\d{3}\b`)
	optionAB  = regexp.MustCompile(`(?i)\boption\s+(?:a|1)\b[\s\S]*\boption\s+(?:b|2)\b`)
)

// excludedContext marks a figure that is not a fee: insurance cover, the
// project's own value, a turnover or a staff count.
var excludedContext = regexp.MustCompile(`(?i)services concerned|failure to exercise reasonable care|policy value|imprisonment|if no invoice has been issued|insurance|insurer|indemnity|public liability|liability\b|\bworkers\b|any one (?:claim|occurrence)|aggregate|valued (?:at|over|up to)|projects? (?:valued|worth|value)|value of the (?:project|works)|estimated value|contract value|turnover|construction (?:cost|value|budget) of|capital (?:value|investment)|\$\s?\d+(?:\.\d+)?\s?(?:m|million|billion|bn)\b`)

var totalWords = regexp.MustCompile(`(?i)\b(?:cost|fee) estimate\b|\btotal\b|\bsub-?total\b|\blump\s?sum\b|\bfixed\s+(?:professional\s+)?fee\b|\bfee\s+(?:total|summary)\b`)

// Harvest collects candidates from a proposal's text.
func Harvest(filename string, text identity.Text) Evidence {
	ev := Evidence{Filename: filename, TextLayer: text.TextLayer, OCR: text.OCR, Incomplete: text.Incomplete}
	lines := Lines(text)
	markdown := strings.EqualFold(filepath.Ext(filename), ".md") || strings.EqualFold(filepath.Ext(filename), ".markdown")
	if markdown {
		lines = markdownLines(lines)
	}
	ev.Readable = readable(lines)
	ev.Form, ev.FormFirm = formCells(lines)
	running := runningLines(lines)
	heading, before := "", ""
	section := -1
	newSection := func(h string, page int) {
		section++
		ev.Sections = append(ev.Sections, Section{Ordinal: section, Heading: clip(h, 200), Page: page})
	}
	newSection("", firstPage(lines))
	seen := map[string]bool{}
	addUnique := func(list *[]string, v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[strings.ToLower(v)] {
			return
		}
		seen[strings.ToLower(v)] = true
		*list = append(*list, v)
	}
	ratesBlock := 0
	incAt := map[int]int{}
	inAddressee := 0
	for i, l := range lines {
		t := l.Text
		lower := strings.ToLower(t)
		if (l.Heading || !markdown && isHeading(t)) && !running[runningKey(t)] {
			heading = t
			if l.HeadingText != "" {
				heading = l.HeadingText
			}
			newSection(heading, l.Page)
		}
		s := &ev.Sections[section]
		if len(s.Body) < 20000 {
			s.Body += t + "\n"
		}
		if strings.Contains(lower, "hourly rate") || strings.Contains(lower, "schedule of rates") || strings.Contains(lower, "charge out rate") || strings.Contains(lower, "charge-out rate") {
			ratesBlock = 12
		} else if ratesBlock > 0 {
			ratesBlock--
		}
		ev.GSTEx += len(exGSTRe.FindAllString(t, -1))
		if n := len(incGSTRe.FindAllString(t, -1)); n > 0 {
			ev.GSTInc += n
			incAt[i] = n
		}
		for _, m := range emailRe.FindAllString(t, -1) {
			addUnique(&ev.Emails, strings.ToLower(m))
		}
		for _, m := range phoneRe.FindAllString(t, -1) {
			addUnique(&ev.Phones, strings.Join(strings.Fields(m), " "))
		}
		// An address in a To / Client / Site block is the addressee's or the
		// project's, never the submitting firm's office.
		if addressee(t) {
			inAddressee = 3
		}
		if inAddressee == 0 {
			for _, m := range addressRe.FindAllString(t, -1) {
				addUnique(&ev.Addresses, strings.Trim(m, " ,."))
			}
		} else {
			inAddressee--
		}
		for _, m := range abnRe.FindAllString(abnGlued.ReplaceAllString(t, "ABN $1"), -1) {
			if d := invoices.NormalizeABN(m); d != "" && invoices.ValidABN(d) {
				addUnique(&ev.ABNs, d)
			}
		}
		if companyRe.MatchString(t) && len(strings.Fields(t)) <= 8 && !moneyRe.MatchString(t) && !emailRe.MatchString(t) {
			addUnique(&ev.Names, cleanFirmCandidate(strings.Trim(t, " ,.-|")))
			if signatureRole.MatchString(t) && i > 0 && personName(lines[i-1].Text) {
				addUnique(&ev.Contacts, lines[i-1].Text)
			}
		}
		if signoffRe.MatchString(strings.TrimSpace(t)) {
			for _, next := range lines[i+1 : min(len(lines), i+5)] {
				if n := strings.TrimSpace(next.Text); personName(n) {
					addUnique(&ev.Contacts, n)
					break
				}
			}
		}
		for j, cell := range l.Cells {
			for _, m := range moneyIn(cell) {
				ctx := t
				// A drawing's area schedule can OCR a dimension as "$700".
				// Its GLA/GFA total is physical area, not a quoted fee.
				if drawingArea.MatchString(ctx) && !feeContext.MatchString(ctx) {
					continue
				}
				label := rowLabel(l.Cells, j, m.before)
				if markdown && len(l.Cells) > 1 && j > 0 {
					label = l.Cells[0]
				}
				above, localHeading := amountColumnContext(lines, i, j, m.before)
				if excludedContext.MatchString(localHeading) || excludedContext.MatchString(label+" "+cell) || excludedContext.MatchString(above) && !strings.Contains(strings.ToLower(label+" "+cell), "fee") {
					continue
				}
				a := Amount{Cents: m.cents, Display: m.display, Context: clip(ctx, 240), Under: clip(heading, 160), Before: clip(before, 160), Page: l.Page, Section: section, Line: i}
				a.Label = label
				if localHeading != "" {
					a.Under = localHeading
				}
				a.Column = columnHeading(lines, i, l, j)
				after := strings.ToLower(m.after)
				a.PerHour = perHourRe.MatchString(m.after) || strings.Contains(after, "per hour") || ((ratesBlock > 0 || strings.Contains(strings.ToLower(localHeading), "rate")) && !totalWords.MatchString(ctx) && m.cents < 100000)
				// A total is printed as one: a short label or line, not a
				// sentence that mentions "the fee summary table".
				a.Total = (totalWords.MatchString(a.Label) && len(strings.Fields(a.Label)) <= 10) ||
					(totalWords.MatchString(ctx) && len(strings.Fields(ctx)) <= 8) ||
					(totalWords.MatchString(before) && len(strings.Fields(before)) <= 4 && len(strings.Fields(ctx)) <= 4)
				if (feeHeading.MatchString(heading) || feeHeading.MatchString(label)) && len(l.Cells) == 1 && len(moneyIn(t)) == 1 && len(strings.Fields(t)) <= 12 {
					a.Total = true
				}
				if incGSTRe.MatchString(ctx) && !exGSTRe.MatchString(ctx) {
					a.GST = "inc"
				} else if exGSTRe.MatchString(ctx) && !incGSTRe.MatchString(ctx) {
					a.GST = "ex"
				}
				if a.GST == "" {
					if exGSTRe.MatchString(a.Column) && !incGSTRe.MatchString(a.Column) {
						a.GST = "ex"
					} else if incGSTRe.MatchString(a.Column) && !exGSTRe.MatchString(a.Column) {
						a.GST = "inc"
					}
				}
				ev.Amounts = append(ev.Amounts, a)
			}
		}
		for _, m := range percentRe.FindAllStringSubmatch(t, -1) {
			bp := percentBasisPoints(m[1])
			if bp <= 0 {
				continue
			}
			ev.Amounts = append(ev.Amounts, Amount{Display: strings.TrimSpace(m[0]), Context: clip(t, 240), Under: clip(heading, 160), Before: clip(before, 160), Page: l.Page, Section: section, Line: i, PercentBP: bp, Kind: "percent"})
		}
		before = t
	}
	dropGSTPairs(ev.Amounts, lines)
	// "Total fee including GST" printed as the inclusive copy of an ex-GST
	// total is arithmetic, not the basis the fees are quoted on.
	for _, a := range ev.Amounts {
		if a.IncTwin && incAt[a.Line] > 0 {
			ev.GSTInc -= incAt[a.Line]
			incAt[a.Line] = 0
		}
	}
	markUnitRates(ev.Amounts, lines)
	ev.Letterhead = letterhead(lines, ev.ABNs)
	ev.Dated = headerDate(lines)
	// Keep table structure with a subtotal: the headings above its component
	// rows explain the scope a single bundled figure covers.
	for i := range ev.Amounts {
		a := &ev.Amounts[i]
		if !a.Total || a.IncTwin || a.GSTPart {
			continue
		}
		seenHeading := map[string]bool{}
		for j := i - 1; j >= 0 && ev.Amounts[j].Page == a.Page; j-- {
			b := ev.Amounts[j]
			if b.Total {
				break
			}
			if b.Under != "" && !seenHeading[b.Under] {
				a.Breakdown = append(a.Breakdown, b.Under)
				seenHeading[b.Under] = true
				if len(a.Breakdown) == 10 {
					break
				}
			}
		}
	}
	for i := range ev.Amounts {
		ev.Amounts[i].ID = "a" + strconv.Itoa(i+1)
	}
	markRepeatedReferences(ev.Amounts)
	ev.Printed = len(ev.Names)
	for _, n := range filenameNames(filename) {
		ev.Hints = append(ev.Hints, n)
		addUnique(&ev.Names, n)
	}
	for _, e := range ev.Emails {
		if n := domainName(e); n != "" {
			ev.Hints = append(ev.Hints, n)
			addUnique(&ev.Names, n)
		}
	}
	ev.KindWords = cueWords(filename, lines)
	ev.OptionsAB = optionAB.MatchString(sectionHeadings(ev.Sections))
	ev.Excerpt = feeExcerpt(lines, ev.Amounts, 9000)
	return ev
}

// A facing column may hold insurance or a different stage. Keep the heading
// above this row's label, instead of borrowing prose from the facing column.
func amountColumnContext(lines []Line, i, j int, prefix string) (string, string) {
	l := lines[i]
	if len(l.X) != len(l.Cells) || j >= len(l.X) {
		if i > 0 && lines[i-1].Page == l.Page {
			return lines[i-1].Text, ""
		}
		return "", ""
	}
	label := j
	if strings.TrimSpace(prefix) == "" {
		for k := j - 1; k >= 0; k-- {
			if !moneyRe.MatchString(l.Cells[k]) {
				label = k
				break
			}
		}
	}
	x := l.X[label]
	var nearby []string
	for k := i - 1; k >= 0 && k >= i-80 && lines[k].Page == l.Page; k-- {
		h := lines[k]
		for c, cell := range h.Cells {
			if c >= len(h.X) || math.Abs(h.X[c]-x) > 16 {
				continue
			}
			if len(nearby) < 3 {
				nearby = append(nearby, cell)
			}
			if strings.Contains(strings.ToLower(h.Text), "role/title") && strings.Contains(strings.ToLower(h.Text), "rate") {
				return strings.Join(nearby, " "), "Role/Title Rate"
			}
			if isHeading(cell) {
				return strings.Join(nearby, " "), clip(cell, 160)
			}
		}
	}
	return strings.Join(nearby, " "), ""
}

type money struct {
	cents   int64
	display string
	before  string
	after   string
}

// moneyIn finds fee figures in one line. "$15.5k" and "$1.2 million" scale;
// a figure inside a date, a phone number or a longer number is not money.
func moneyIn(s string) []money {
	var out []money
	for _, loc := range moneyRe.FindAllStringSubmatchIndex(s, -1) {
		start, end := loc[0], loc[1]
		var digits, frac, scale string
		if loc[2] >= 0 {
			digits = s[loc[4]:loc[5]]
			if loc[6] >= 0 {
				frac = s[loc[6]+1 : loc[7]]
			}
			if loc[8] >= 0 {
				scale = strings.ToLower(strings.TrimSpace(s[loc[8]:loc[9]]))
			}
		} else {
			digits, frac = s[loc[10]:loc[11]], s[loc[12]:loc[13]]
			if start > 0 && strings.ContainsAny(s[start-1:start], "0123456789/.-") {
				continue
			}
		}
		if end < len(s) && strings.ContainsAny(s[end:end+1], "0123456789%") && scale == "" {
			continue
		}
		w, err := strconv.ParseInt(strings.NewReplacer(",", "", " ", "").Replace(digits), 10, 64)
		if err != nil || w > 1_000_000_000 {
			continue
		}
		f, _ := strconv.ParseInt(frac+strings.Repeat("0", 2-len(frac)), 10, 64)
		cents := w*100 + f
		switch scale {
		case "k":
			cents *= 1000
		case "m", "million":
			cents *= 1_000_000
		}
		out = append(out, money{cents: cents, display: strings.TrimSpace(s[start:end]), before: s[:start], after: s[end:]})
	}
	return out
}

// dropGSTPairs marks the GST and the GST-inclusive figures where a proposal
// prints an amount, its GST and their sum, on one line or on lines close
// together ("Subtotal $1,100 / GST $110 / Total $1,210"), and a figure
// printed with its inclusive twin ("$10,000 + GST = $11,000").
func dropGSTPairs(amounts []Amount, lines []Line) {
	near := func(a, b Amount) bool { return a.Page == b.Page && a.Line-b.Line <= 4 && b.Line-a.Line <= 4 }
	for i := range amounts {
		x := &amounts[i]
		if x.Kind != "" || x.Cents <= 0 {
			continue
		}
		for j := range amounts {
			t := &amounts[j]
			if i == j || t.Kind != "" || !near(*x, *t) {
				continue
			}
			if d := t.Cents - (x.Cents + invoices.GSTOf(x.Cents)); d < -1 || d > 1 {
				continue
			}
			// Across lines only a printed total can be the inclusive sum; in
			// a rate table the next row's rate often equals this one plus GST.
			if t.Line != x.Line && !t.Total {
				continue
			}
			// Arithmetic alone cannot distinguish GST from a genuinely
			// different offer. In particular, never replace explicit ex-GST
			// wording on a separate alternative.
			taxEvidence := t.GST == "inc"
			if t.Line == x.Line {
				taxEvidence = taxEvidence || strings.Contains(strings.ToLower(x.Context), "gst")
			} else if t.GST == "ex" {
				continue
			}
			for k := range amounts {
				g := &amounts[k]
				if k != i && k != j && near(*x, *g) && abs64(g.Cents-invoices.GSTOf(x.Cents)) <= 1 && strings.Contains(strings.ToLower(lines[g.Line].Text+" "+g.Column), "gst") {
					taxEvidence = true
				}
			}
			if !taxEvidence {
				continue
			}
			// t is x plus GST. Mark t inclusive, and the GST figure itself.
			if x.GST == "" {
				x.GST = "ex"
			}
			t.GST, t.IncTwin = "inc", true
			for k := range amounts {
				g := &amounts[k]
				if k != i && k != j && near(*x, *g) && g.Kind == "" && abs64(g.Cents-invoices.GSTOf(x.Cents)) <= 1 && strings.Contains(strings.ToLower(lines[g.Line].Text+" "+g.Column), "gst") {
					g.GSTPart = true
				}
			}
		}
	}
}

var unitWord = regexp.MustCompile(`(?i)\b(?:hours?|hrs?|days?|weeks?|sum|each|ea|item|tests?|samples?|suites?|visits?|box(?:es)?|cores?|holes?|carts?|wells?|mob|no\.?|m2|lm|sheets?|copies)\b`)
var drawingArea = regexp.MustCompile(`(?i)\btotal\s+(?:gla|gfa)\b`)
var feeContext = regexp.MustCompile(`(?i)\b(?:fee|price|rate|cost|gst)\b`)

// markUnitRates marks a rate printed just before its extended amount on a
// quantity line ("30  hour  $220  $6,600"): the extended amount is the fee.
func markUnitRates(amounts []Amount, lines []Line) {
	for i := 0; i+1 < len(amounts); i++ {
		u, e := &amounts[i], amounts[i+1]
		if u.Cents <= 0 || u.Kind != "" || e.Kind != "" || e.Line != u.Line || e.Cents <= u.Cents || e.Cents%u.Cents != 0 {
			continue
		}
		text := lines[u.Line].Text
		if !unitWord.MatchString(text) {
			continue
		}
		words := map[string]bool{}
		for _, w := range strings.Fields(text) {
			words[strings.Trim(w, "x×(),")] = true
		}
		if n := e.Cents / u.Cents; n >= 2 && n <= 10000 && words[strconv.FormatInt(n, 10)] {
			u.Unit = true
		}
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// rowLabel is the row's own words: what the figure's cell prints before it,
// or else the nearest cell to its left that holds no figure.
func rowLabel(cells []string, j int, prefix string) string {
	if p := strings.Trim(prefix, " :-–=|$"); len(p) >= 3 {
		return clip(p, 100)
	}
	for k := j - 1; k >= 0; k-- {
		c := strings.Trim(cells[k], " :-–=|")
		if moneyRe.MatchString(c) {
			continue
		}
		if len(c) >= 3 && !strings.EqualFold(unitWord.FindString(c), c) {
			return clip(c, 100)
		}
	}
	return ""
}

// columnHeading is the table heading printed above a figure's cell: the
// nearest line above, on the same page and before the next heading, whose
// cells hold no figures and one of which sits over this cell.
func columnHeading(lines []Line, i int, l Line, j int) string {
	if len(l.X) == 0 && len(l.Cells) > 1 {
		for k := i - 1; k >= 0 && k >= i-30; k-- {
			h := lines[k]
			if h.Page != l.Page || len(h.Cells) != len(l.Cells) {
				break
			}
			if !moneyRe.MatchString(h.Text) && j < len(h.Cells) && (exGSTRe.MatchString(h.Cells[j]) || incGSTRe.MatchString(h.Cells[j])) {
				return h.Cells[j]
			}
		}
	}
	if j >= len(l.X) || len(l.Cells) < 2 {
		return ""
	}
	x0, x1 := l.X[j], l.X[j]+l.W[j]
	for k := i - 1; k >= 0 && k >= i-15; k-- {
		h := lines[k]
		if h.Page != l.Page {
			return ""
		}
		if moneyRe.MatchString(h.Text) {
			continue
		}
		if len(h.X) != len(h.Cells) {
			continue
		}
		for c := range h.Cells {
			if len(h.Cells[c]) <= 40 && h.X[c] <= x1+4 && x0 <= h.X[c]+h.W[c]+4 {
				return clip(h.Cells[c], 60)
			}
		}
	}
	return ""
}

func percentBasisPoints(s string) int {
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.Atoi(whole)
	if err != nil {
		return 0
	}
	f := 0
	if frac != "" {
		f, _ = strconv.Atoi((frac + "0")[:2])
	}
	return w*100 + f
}

// runningLines are lines printed on three or more pages: page headers,
// footers and firm addresses. They are never a section heading.
func runningLines(lines []Line) map[string]bool {
	pages := map[string]map[int]bool{}
	for _, l := range lines {
		k := runningKey(l.Text)
		if pages[k] == nil {
			pages[k] = map[int]bool{}
		}
		pages[k][l.Page] = true
	}
	out := map[string]bool{}
	for k, ps := range pages {
		if len(ps) >= 3 {
			out[k] = true
		}
	}
	return out
}

var digitsRe = regexp.MustCompile(`\d+`)

func runningKey(t string) string { return strings.ToLower(digitsRe.ReplaceAllString(t, "#")) }

// isHeading is a short numbered or labelled line ("Task 2 - Pre-DA meeting",
// "3.1 Scope", "STAGE 1"), or a short line in capitals.
func isHeading(t string) bool {
	words := strings.Fields(t)
	if len(words) == 0 || len(words) > 12 || len(t) > 100 || moneyRe.MatchString(t) {
		return false
	}
	if headingRe.MatchString(t) {
		return true
	}
	letters, upper := 0, 0
	for _, r := range t {
		if r >= 'A' && r <= 'Z' {
			upper++
			letters++
		} else if r >= 'a' && r <= 'z' {
			letters++
		}
	}
	return letters >= 6 && upper == letters && len(words) <= 8
}

func personName(s string) bool {
	words := strings.Fields(s)
	if len(words) < 2 || len(words) > 4 || len(s) > 50 {
		return false
	}
	for _, w := range words {
		if w[0] < 'A' || w[0] > 'Z' {
			return false
		}
	}
	return !companyRe.MatchString(s)
}

// filenameNames are firm names a filename carries in brackets, the way a PM
// saves them: "Fee Traffic (Varga).pdf".
func filenameNames(filename string) []string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	var out []string
	for _, m := range regexp.MustCompile(`\(([^()]{2,60})\)`).FindAllStringSubmatch(base, -1) {
		n := strings.TrimSpace(m[1])
		if !regexp.MustCompile(`(?i)^(rev|r)\s?[a-z0-9]{1,3}$|^\d+$`).MatchString(n) {
			out = append(out, n)
		}
	}
	return out
}

// domainName is the firm name an email domain implies: info@mecone.com.au →
// "Mecone". Public mail domains imply nothing.
func domainName(email string) string {
	_, domain, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	label, _, _ := strings.Cut(domain, ".")
	switch label {
	case "gmail", "hotmail", "outlook", "bigpond", "yahoo", "icloud", "live", "optusnet", "iinet", "tpg":
		return ""
	}
	if len(label) < 3 {
		return ""
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

var cuePhrases = []string{"fee proposal", "fee submission", "fee estimate", "fee quote", "fee schedule", "proposal for", "tender submission", "tender", "quotation", "quote", "proposal", "response form", "fee"}

// cueWords are proposal phrases in the filename and the leading lines.
func cueWords(filename string, lines []Line) []string {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	text := base
	for _, l := range lines[:min(len(lines), 40)] {
		text += "\n" + strings.ToLower(l.Text)
	}
	var out []string
	for _, p := range cuePhrases {
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(p) + `\b`).MatchString(text) {
			out = append(out, p)
		}
	}
	return out
}

func sectionHeadings(sections []Section) string {
	var b strings.Builder
	for _, s := range sections {
		b.WriteString(s.Heading)
		b.WriteByte('\n')
	}
	return b.String()
}

// formCells reads a returned SiteWise response form: column A holds a
// schedule key and column C its fee or rate; the firm block holds labelled
// cells. Anything else is ignored.
func formCells(lines []Line) (map[string]string, map[string]string) {
	keys := map[string]bool{"base_total": true, "options_total": true}
	form, firm := map[string]string{}, map[string]string{}
	isForm := false
	for _, l := range lines {
		if l.Sheet == "R1 Fees" {
			isForm = true
		}
	}
	if !isForm {
		return nil, nil
	}
	for _, l := range lines {
		if len(l.Cells) == 0 || l.Sheet == "" {
			continue
		}
		first := l.Cells[0]
		if l.Sheet == "R1 Fees" {
			for _, f := range rfp.FormFirmFields {
				if first == f && len(l.Cells) > 1 {
					firm[f] = l.Cells[1]
				}
			}
		}
		if !(keys[first] || strings.Contains(first, ":") || first == rfp.ReimbursablesKey) {
			continue
		}
		// The fee is column C on the fee sheet; a rate is column D.
		want := 3
		if strings.HasPrefix(first, "rate:") {
			want = 4
		}
		// A row the firm left blank is on the form, and is not priced.
		form[first] = ""
		for i, c := range l.Cols {
			if c == want {
				form[first] = l.Cells[i]
			}
		}
	}
	return form, firm
}

func firstPage(lines []Line) int {
	if len(lines) == 0 {
		return 0
	}
	return lines[0].Page
}

// feeExcerpt is the proposal's opening lines, then the lines around each
// fee, so the state carries the fee schedule rather than the firm profile.
func feeExcerpt(lines []Line, amounts []Amount, max int) string {
	keep := map[int]bool{}
	for i := 0; i < min(len(lines), 25); i++ {
		keep[i] = true
	}
	for _, a := range amounts {
		before := 8
		if a.PerHour {
			before = 2
		}
		for i := max0(a.Line - before); i <= min(len(lines)-1, a.Line+2); i++ {
			keep[i] = true
		}
	}
	var b strings.Builder
	last := -1
	for i, l := range lines {
		if !keep[i] {
			continue
		}
		if last >= 0 && i != last+1 {
			b.WriteString("…\n")
		}
		if b.Len()+len(l.Text)+1 > max {
			break
		}
		b.WriteString(l.Text)
		b.WriteByte('\n')
		last = i
	}
	return b.String()
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func excerpt(lines []Line, max int) string {
	var b strings.Builder
	for _, l := range lines {
		if b.Len()+len(l.Text)+1 > max {
			break
		}
		b.WriteString(l.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

// readable is false when the text layer is glyph codes, not words.
func readable(lines []Line) bool {
	letters, total := 0, 0
	for _, l := range lines {
		for _, r := range l.Text {
			total++
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
				letters++
			}
		}
	}
	return total > 0 && letters*3 >= total
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut]
}

// Cue reports whether a filed document should be read as a proposal: it
// calls itself one in its filename or opening lines, or it was filed as
// commercial and prices fees. A wrong cue costs one fan-out whose kind
// question answers "not a proposal"; a missed one leaves the PM to drop the
// file on the package row.
func Cue(filename string, text identity.Text, kind string) bool {
	words := map[string]bool{}
	for _, w := range cueWords(filename, Lines(text)) {
		words[w] = true
	}
	for _, strong := range []string{"fee proposal", "fee submission", "fee estimate", "fee quote", "fee schedule", "tender submission", "quotation", "response form", "proposal for"} {
		if words[strong] {
			return true
		}
	}
	return kind == "commercial" && (words["fee"] || words["proposal"] || words["quote"] || words["tender"])
}
