package invoices

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/identity"
)

// Candidate is one value code found on the invoice. Value is normalized
// (cents, ISO date, upper-case number, ABN digits); Display is as printed.
// Labels are the lower-case words printed with it: its own prefix, the cell
// to its left and the cell above. They are provenance, not a decision.
type Candidate struct {
	Value   string
	Display string
	Labels  []string
	Labeled bool // bound to a label naming the field
	Count   int
	Page    int
}

// Evidence is everything harvested from one invoice.
type Evidence struct {
	Filename  string
	TextLayer bool
	Amounts   []Candidate
	Dates     []Candidate
	Numbers   []Candidate
	ABNs      []Candidate
	// BadABNs are eleven-digit values printed next to "ABN" that fail the checksum.
	BadABNs []string
	// Currency is a foreign currency code seen on the invoice, or "".
	Currency string
	// KindWords are kind phrases printed in a short heading row or the filename.
	KindWords []string
	// InvoiceCount is the number of distinct labelled invoice numbers.
	InvoiceCount int
	Excerpt      string
	// Readable is false when the text layer is there but not words: a PDF
	// whose font has no usable character map prints glyph codes.
	Readable bool
}

type cell struct {
	x, y, w float64
	text    string
}

type row struct {
	page  int
	y     float64
	cells []cell
}

func (r row) text() string {
	parts := make([]string, len(r.cells))
	for i, c := range r.cells {
		parts[i] = c.text
	}
	return strings.Join(parts, "  ")
}

// rowsOf groups runs printed on one baseline, top to bottom and left to
// right. Runs without geometry (DOCX, XLSX) are one row each, in order.
func rowsOf(text identity.Text) []row {
	var out []row
	byPage := map[int][]identity.Run{}
	var pages []int
	for _, r := range text.Runs {
		t := strings.TrimSpace(strings.ReplaceAll(r.Text, "\n", " "))
		if t == "" {
			continue
		}
		r.Text = t
		if r.Source.Width == 0 && r.Source.X == 0 && r.Source.Y == 0 {
			out = append(out, row{page: r.Source.Page, cells: []cell{{text: t}}})
			continue
		}
		if _, ok := byPage[r.Source.Page]; !ok {
			pages = append(pages, r.Source.Page)
		}
		byPage[r.Source.Page] = append(byPage[r.Source.Page], r)
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
		var cur *row
		for _, r := range runs {
			c := cell{x: r.Source.X, y: r.Source.Y, w: r.Source.Width, text: r.Text}
			if cur != nil && math.Abs(cur.y-c.y) <= 3 {
				cur.cells = append(cur.cells, c)
				continue
			}
			out = append(out, row{page: p, y: c.y, cells: []cell{c}})
			cur = &out[len(out)-1]
		}
		for i := range out {
			if out[i].page == p {
				sort.SliceStable(out[i].cells, func(a, b int) bool { return out[i].cells[a].x < out[i].cells[b].x })
			}
		}
	}
	return out
}

var (
	moneyToken     = regexp.MustCompile(`(?i)(\(\s*)?(\$\s?|AUD\s?)?(-\s?)?(\d{1,3}(?:,\d{3})+|\d+)(\.\d{2})?(\s*\))?(\s?CR\b)?`)
	dateDMY        = regexp.MustCompile(`\b(\d{1,2})[/.\-](\d{1,2})[/.\-](\d{4}|\d{2})\b`)
	dateISO        = regexp.MustCompile(`\b(20\d{2})-(\d{2})-(\d{2})\b`)
	dateWords      = regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)?[ \-]([A-Za-z]{3,9})\.?[ ,\-]+(\d{4}|\d{2})\b`)
	dateUS         = regexp.MustCompile(`(?i)\b([A-Za-z]{3,9})\.? (\d{1,2})(?:st|nd|rd|th)?,? (\d{4})\b`)
	numberLbl      = regexp.MustCompile(`(?i)\b(?:tax\s+)?inv(?:oice)?\.?\s*(?:no\b\.?|number\b|num\b|#|n°)\s*[:.]?|\binvoice\s*:|\binv#`)
	numberTok      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9\-\x{2013}/_.#]{0,29}$`)
	taxInvoiceHead = regexp.MustCompile(`(?i)^\s*tax\s+invoice\b`)
	invToken       = regexp.MustCompile(`(?i)\bINV[\-#]?\d[A-Za-z0-9\-]{0,20}\b`)
	currencyRe     = regexp.MustCompile(`\b(USD|NZD|EUR|GBP|SGD|HKD|CAD|JPY|CNY)\b|US\$|NZ\$|€|£`)
)

var months = map[string]time.Month{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4,
	"may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9,
	"september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12,
}

// Kind phrases, longest first so "tax invoice" wins over "invoice".
var kindPhrases = []string{"adjustment note", "credit note", "credit memo", "tax invoice", "progress claim", "payment claim", "payment schedule", "fee proposal", "quotation", "quote", "statement", "remittance", "receipt", "invoice"}

// Harvest reads candidates from one invoice's text. It decides nothing.
func Harvest(filename string, text identity.Text) Evidence {
	ev := Evidence{Filename: filename, TextLayer: text.TextLayer}
	rows := rowsOf(text)
	amounts, dates, numbers, abns := map[string]*Candidate{}, map[string]*Candidate{}, map[string]*Candidate{}, map[string]*Candidate{}
	var aOrder, dOrder, nOrder, bOrder []string
	add := func(set map[string]*Candidate, order *[]string, value, display string, labels []string, labeled bool, page int) {
		if c, ok := set[value]; ok {
			c.Count++
			c.Labeled = c.Labeled || labeled
			c.Labels = mergeLabels(c.Labels, labels)
			return
		}
		set[value] = &Candidate{Value: value, Display: display, Labels: mergeLabels(nil, labels), Labeled: labeled, Count: 1, Page: page}
		*order = append(*order, value)
	}
	kinds := map[string]bool{}
	for i, r := range rows {
		rowText := r.text()
		if len(strings.Fields(rowText)) <= 7 {
			for _, k := range phrasesIn(rowText) {
				kinds[k] = true
			}
		}
		for j, c := range r.cells {
			labels := labelsFor(rows, i, j)
			for _, m := range moneyIn(c.text) {
				add(amounts, &aOrder, strconv.FormatInt(m.cents, 10), m.display, append([]string{strings.ToLower(m.prefix)}, labels...), false, r.page)
			}
			for _, d := range datesIn(c.text) {
				own := strings.ToLower(d.prefix)
				add(dates, &dOrder, d.iso, d.display, append([]string{own}, labels...), isInvoiceDateLabel(own) || anyInvoiceDateLabel(labels), r.page)
			}
			for _, n := range numbersIn(rows, i, j) {
				add(numbers, &nOrder, n.value, n.display, labels, n.labeled, r.page)
			}
			for _, m := range abnPattern.FindAllString(abnGlued.ReplaceAllString(c.text, "ABN $1"), -1) {
				digits := NormalizeABN(m)
				if digits == "" {
					continue
				}
				near := strings.ToLower(c.text + " " + strings.Join(labels, " "))
				if !ValidABN(digits) {
					if strings.Contains(near, "abn") {
						ev.BadABNs = append(ev.BadABNs, digits)
					}
					continue
				}
				add(abns, &bOrder, digits, DisplayABN(digits), []string{strings.ToLower(c.text)}, strings.Contains(near, "abn"), r.page)
			}
			if m := currencyRe.FindString(c.text); m != "" && ev.Currency == "" {
				ev.Currency = currencyCode(m)
			}
		}
	}
	for _, n := range filenameNumbers(filename) {
		add(numbers, &nOrder, n.value, n.display, []string{"filename"}, false, 0)
	}
	for _, k := range phrasesIn(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))) {
		kinds[k] = true
	}
	ev.Amounts = collect(amounts, aOrder)
	ev.Dates = collect(dates, dOrder)
	ev.Numbers = collect(numbers, nOrder)
	ev.ABNs = collect(abns, bOrder)
	for k := range kinds {
		ev.KindWords = append(ev.KindWords, k)
	}
	sort.Strings(ev.KindWords)
	for _, n := range ev.Numbers {
		if n.Labeled {
			ev.InvoiceCount++
		}
	}
	ev.Excerpt = excerpt(rows, 3000)
	ev.Readable = readable(rows)
	return ev
}

func collect(set map[string]*Candidate, order []string) []Candidate {
	out := make([]Candidate, 0, len(order))
	for _, k := range order {
		out = append(out, *set[k])
	}
	return out
}

func mergeLabels(have, more []string) []string {
	for _, l := range more {
		l = strings.TrimSpace(l)
		if l == "" || len(have) >= 8 {
			continue
		}
		dup := false
		for _, h := range have {
			if h == l {
				dup = true
				break
			}
		}
		if !dup {
			have = append(have, clip(l, 80))
		}
	}
	return have
}

// labelsFor is the cell to the left on the same row and the nearest cell
// above that overlaps horizontally: the two places an invoice prints a label.
func labelsFor(rows []row, i, j int) []string {
	var out []string
	r, c := rows[i], rows[i].cells[j]
	if j > 0 {
		out = append(out, strings.ToLower(r.cells[j-1].text))
	}
	if c.y == 0 && c.x == 0 {
		if i > 0 {
			out = append(out, strings.ToLower(rows[i-1].text()))
		}
		return out
	}
	for k := i - 1; k >= 0 && k >= i-4; k-- {
		up := rows[k]
		if up.page != r.page || up.y-r.y > 26 {
			break
		}
		best, dist := "", math.Inf(1)
		for _, u := range up.cells {
			d := math.Abs(center(u) - center(c))
			if d < dist && (d <= 60 || overlaps(u, c)) {
				best, dist = u.text, d
			}
		}
		if best != "" {
			if headerRow(up) {
				best = "hdr:" + best
			}
			out = append(out, strings.ToLower(best))
			break
		}
	}
	return out
}

func center(c cell) float64 {
	w := c.w
	if w == 0 {
		w = 5 * float64(len(c.text))
	}
	return c.x + w/2
}

func overlaps(a, b cell) bool {
	aw, bw := a.w, b.w
	if aw == 0 {
		aw = 5 * float64(len(a.text))
	}
	if bw == 0 {
		bw = 5 * float64(len(b.text))
	}
	return a.x <= b.x+bw && b.x <= a.x+aw
}

type money struct {
	cents   int64
	display string
	prefix  string
}

// moneyIn finds amounts: a figure with a dollar sign or with cents. A
// percentage, part of a date or a longer number is not an amount.
func moneyIn(s string) []money {
	var out []money
	for _, loc := range moneyToken.FindAllStringSubmatchIndex(s, -1) {
		start, end := loc[0], loc[1]
		hasDollar := loc[4] >= 0
		hasCents := loc[10] >= 0
		if !hasDollar && !hasCents {
			continue
		}
		if start > 0 && strings.ContainsAny(s[start-1:start], "0123456789./-") && !hasDollar {
			continue
		}
		after := strings.TrimLeft(s[end:], " ")
		if strings.HasPrefix(after, "%") || (end < len(s) && strings.ContainsAny(s[end:end+1], "0123456789/")) {
			continue
		}
		if end < len(s) && s[end] == '.' && end+1 < len(s) && s[end+1] >= '0' && s[end+1] <= '9' {
			continue
		}
		digits := strings.ReplaceAll(s[loc[8]:loc[9]], ",", "")
		frac := "00"
		if hasCents {
			frac = s[loc[10]+1 : loc[11]]
		}
		w, err := strconv.ParseInt(digits, 10, 64)
		if err != nil || w > maxCents/100 {
			continue
		}
		cents := w*100 + mustAtoi(frac)
		neg := loc[6] >= 0 || (loc[2] >= 0 && loc[12] >= 0) || loc[14] >= 0
		if neg {
			cents = -cents
		}
		out = append(out, money{cents: cents, display: strings.TrimSpace(s[start:end]), prefix: strings.TrimSpace(s[:start])})
	}
	return out
}

func mustAtoi(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

type dateHit struct {
	iso, display, prefix string
}

func datesIn(s string) []dateHit {
	var out []dateHit
	seen := map[int]bool{}
	try := func(loc []int, y, m, d int) {
		if seen[loc[0]] {
			return
		}
		if y < 100 {
			y += 2000
		}
		if y < 2000 || y > 2099 || m < 1 || m > 12 || d < 1 || d > 31 {
			return
		}
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		if t.Day() != d {
			return
		}
		seen[loc[0]] = true
		out = append(out, dateHit{iso: t.Format("2006-01-02"), display: s[loc[0]:loc[1]], prefix: strings.TrimSpace(s[:loc[0]])})
	}
	for _, loc := range dateISO.FindAllStringSubmatchIndex(s, -1) {
		try(loc, atoi(s, loc, 1), atoi(s, loc, 2), atoi(s, loc, 3))
	}
	for _, loc := range dateDMY.FindAllStringSubmatchIndex(s, -1) {
		// Australian invoices are day first.
		try(loc, atoi(s, loc, 3), atoi(s, loc, 2), atoi(s, loc, 1))
	}
	for _, loc := range dateWords.FindAllStringSubmatchIndex(s, -1) {
		if m, ok := months[strings.ToLower(s[loc[4]:loc[5]])]; ok {
			try(loc, atoi(s, loc, 3), int(m), atoi(s, loc, 1))
		}
	}
	for _, loc := range dateUS.FindAllStringSubmatchIndex(s, -1) {
		if m, ok := months[strings.ToLower(s[loc[2]:loc[3]])]; ok {
			try(loc, atoi(s, loc, 3), int(m), atoi(s, loc, 2))
		}
	}
	return out
}

func atoi(s string, loc []int, g int) int {
	n, _ := strconv.Atoi(s[loc[2*g]:loc[2*g+1]])
	return n
}

func isInvoiceDateLabel(l string) bool {
	l = strings.ToLower(l)
	if strings.Contains(l, "due") || strings.Contains(l, "paid") || strings.Contains(l, "pay by") || strings.Contains(l, "period") || strings.Contains(l, "dated") {
		return false
	}
	return strings.Contains(l, "invoice date") || strings.Contains(l, "date of invoice") || strings.Contains(l, "tax date") ||
		strings.Contains(l, "issue date") || strings.Contains(l, "date issued") || strings.TrimRight(strings.TrimSpace(l), ":") == "date"
}

func anyInvoiceDateLabel(labels []string) bool {
	for _, l := range labels {
		if isInvoiceDateLabel(l) {
			return true
		}
	}
	return false
}

type numberHit struct {
	value, display string
	labeled        bool
}

// numbersIn finds invoice numbers bound to a label: printed after it in the
// same cell, under it in a column, or in the next cell. A value under a
// table header (a claim history's "Invoice No." column) is not this
// invoice's number, so it stays unlabelled.
func numbersIn(rows []row, i, j int) []numberHit {
	c := rows[i].cells[j]
	var out []numberHit
	if loc := numberLbl.FindStringIndex(c.text); loc != nil {
		rest := strings.TrimSpace(c.text[loc[1]:])
		tok := numberValue(rest)
		labeled := true
		if tok == "" && strings.Trim(rest, ":.# ") == "" {
			tok = below(rows, i, c)
			if tok != "" && headerRow(rows[i]) {
				labeled = false
			}
			if tok == "" && j+1 < len(rows[i].cells) {
				tok = numberValue(rows[i].cells[j+1].text)
			}
		}
		if tok != "" {
			out = append(out, numberHit{value: NormalizeNumber(tok), display: tok, labeled: labeled})
		}
	}
	// "TAX INVOICE 2027663", or a heading with the number in the next cell.
	if loc := taxInvoiceHead.FindStringIndex(c.text); loc != nil && numberLbl.FindStringIndex(c.text) == nil {
		tok := numberValue(c.text[loc[1]:])
		if tok == "" && strings.TrimSpace(c.text[loc[1]:]) == "" && j+1 < len(rows[i].cells) {
			tok = numberValue(rows[i].cells[j+1].text)
		}
		if tok != "" {
			out = append(out, numberHit{value: NormalizeNumber(tok), display: tok, labeled: true})
		}
	}
	for _, m := range invToken.FindAllString(c.text, -1) {
		out = append(out, numberHit{value: NormalizeNumber(m), display: m})
	}
	return out
}

// headerRow is a table heading: four or more cells and no digits.
func headerRow(r row) bool {
	if len(r.cells) < 4 {
		return false
	}
	for _, c := range r.cells {
		if strings.ContainsAny(c.text, "0123456789") {
			return false
		}
	}
	return true
}

func below(rows []row, i int, label cell) string {
	for k := i + 1; k < len(rows) && k <= i+3; k++ {
		r := rows[k]
		if r.page != rows[i].page || rows[i].y-r.y > 26 {
			return ""
		}
		for _, c := range r.cells {
			if math.Abs(center(c)-center(label)) <= 60 || overlaps(c, label) {
				return numberValue(c.text)
			}
		}
	}
	return ""
}

// numberValue is the number printed after a label. A short value of up to
// three parts is kept whole ("NSW 231837", "06862.14 – 1B"); in longer text
// only its first part with a digit counts.
func numberValue(s string) string {
	s = strings.TrimLeft(strings.TrimSpace(s), ":.#- ")
	f := strings.Fields(s)
	if len(f) == 0 || (len(f) <= 3 && len(datesIn(s)) > 0) {
		return ""
	}
	if len(f) <= 3 && len(s) <= 24 && strings.ContainsAny(s, "0123456789") {
		ok := true
		for _, p := range f {
			if !numberTok.MatchString(p) && p != "–" && p != "-" {
				ok = false
			}
		}
		if ok {
			return s
		}
	}
	tok := strings.TrimRight(f[0], ",;:")
	if !numberTok.MatchString(tok) || !strings.ContainsAny(tok, "0123456789") || len(datesIn(tok)) > 0 {
		return ""
	}
	return tok
}

// NormalizeNumber compares invoice numbers without case or spacing.
func NormalizeNumber(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}

// filenameNumbers are tokens a filename names as the invoice number: the
// token after "Invoice", "INV" or "Inv#", or an INV-prefixed token.
func filenameNumbers(filename string) []numberHit {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	fields := strings.FieldsFunc(base, func(r rune) bool { return r == ' ' || r == '_' })
	var out []numberHit
	for i, f := range fields {
		lower := strings.ToLower(f)
		switch {
		case invToken.MatchString(f):
			out = append(out, numberHit{value: NormalizeNumber(f), display: f})
		case (lower == "invoice" || lower == "inv") && i+1 < len(fields):
			next := strings.Trim(fields[i+1], "-")
			if numberTok.MatchString(next) && strings.ContainsAny(next, "0123456789") && len(datesIn(next)) == 0 {
				out = append(out, numberHit{value: NormalizeNumber(next), display: next})
			}
		case strings.HasPrefix(lower, "inv#") && len(f) > 4:
			out = append(out, numberHit{value: NormalizeNumber(f[4:]), display: f[4:]})
		}
	}
	return out
}

func phrasesIn(s string) []string {
	l := " " + strings.ToLower(s) + " "
	var out []string
	for _, p := range kindPhrases {
		if strings.Contains(l, p) {
			out = append(out, p)
			l = strings.ReplaceAll(l, p, " ")
		}
	}
	return out
}

func currencyCode(m string) string {
	switch m {
	case "US$":
		return "USD"
	case "NZ$":
		return "NZD"
	case "€":
		return "EUR"
	case "£":
		return "GBP"
	}
	return m
}

func excerpt(rows []row, max int) string {
	var b strings.Builder
	for _, r := range rows {
		line := r.text()
		if b.Len()+len(line)+1 > max {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// String is a compact summary for diagnostics.
func (c Candidate) String() string {
	return fmt.Sprintf("%s %q %v", c.Value, c.Display, c.Labels)
}

// readable reports whether the first page reads as words: mostly letters,
// digits and spaces, with at least one word an invoice or receipt prints.
func readable(rows []row) bool {
	var plain, other int
	var b strings.Builder
	for _, r := range rows {
		if r.page > 1 {
			break
		}
		t := r.text()
		b.WriteString(strings.ToLower(t))
		b.WriteByte(' ')
		for _, ch := range t {
			switch {
			case ch == ' ' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune(",.$:/-()&%'@#", ch):
				plain++
			default:
				other++
			}
		}
	}
	if plain == 0 || float64(plain)/float64(plain+other) < 0.9 {
		return false
	}
	text := b.String()
	for _, w := range []string{"invoice", "total", "gst", "abn", "amount", "receipt", "tax", "date"} {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}
