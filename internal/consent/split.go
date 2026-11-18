// Package consent reads a development consent's conditions
// (docs/plans/2026-11-17-consent-conditions.md). Go spots a notice of
// determination, splits it into its numbered conditions by their printed
// numbers and stage headings, maps a heading to a gate, and reads the
// consent reference and the determination date. Jev is asked only what the
// words leave open (read.go). A condition's number and words are the
// instrument's: nothing here renumbers, rewords or summarises one.
package consent

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"sitewise/internal/delivery"
)

// Page is one page of a notice's text, as filing extracted it (or OCR).
type Page struct {
	Page int
	Text string
}

// Condition is one numbered condition as printed.
type Condition struct {
	Number       string // as printed: "46", "A1"
	Heading      string // the printed heading, or ""
	Text         string // the council's words, verbatim (cut at 12,000)
	Page         int    // the page it starts on
	EndPage      int    // the page it ends on
	StageHeading string // the printed stage heading above it
	Stage        string // the gate the heading settles, or "" for Jev
	Deferred     bool   // under a deferred commencement heading
	// Marker is "deleted" or "changed" when a modification restates it so,
	// and MarkerBy the modification the notice names for it ("MOD/2023/0177").
	Marker, MarkerBy string
}

// Notice is what Split read.
type Notice struct {
	ConsentRef string // the development consent the notice is for, "" when not found
	// ModRef is the modification's own reference (MOD/2023/0177) when the
	// notice modifies a consent.
	ModRef       string
	Modification bool
	Determined   string // YYYY-MM-DD by rule, "" when not found; orders notices only
	Conditions   []Condition
	// Gaps are numbers missing from a sequence, as ranges ("14", "20–22").
	Gaps []string
}

// MaxConditions is the most a notice may hold; more is refused, as the
// specification import refuses past 1,000 candidates.
const MaxConditions = 600

var noticeMarkers = []string{"notice of determination", "development consent", "determination no"}

// Detect reports a notice of determination: a marker in its text and at
// least three numbered conditions.
func Detect(pages []Page) bool {
	var b strings.Builder
	for _, p := range pages {
		b.WriteString(strings.ToLower(p.Text))
		b.WriteByte('\n')
	}
	text := b.String()
	marked := false
	for _, m := range noticeMarkers {
		marked = marked || strings.Contains(text, m)
	}
	return marked && len(Split(pages).Conditions) >= 3
}

type line struct {
	text string
	page int
}

// Split reads the conditions in printed order.
func Split(pages []Page) Notice {
	lines := cleanLines(pages)
	n := Notice{}
	all := joinLines(lines)
	n.ConsentRef, n.ModRef, n.Modification = references(all)
	n.Determined = determined(all, n.Modification)
	if rows, ok := tableConditions(lines); ok {
		n.Conditions = rows
	} else {
		n.Conditions = lineConditions(lines)
	}
	if !n.Modification {
		n.Gaps = gaps(n.Conditions)
	}
	return n
}

func joinLines(lines []line) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.text)
		b.WriteByte('\n')
	}
	return b.String()
}

var (
	pageOfRe    = regexp.MustCompile(`(?i)^\s*page\s+\d+\s+(of\s+\d+)?\s*$`)
	digitsRe    = regexp.MustCompile(`\d+`)
	footerWords = []string{"document set id", "version: ", "printed:"}
)

// cleanLines splits pages into lines and drops repeated footers: "Page 5
// of 25", "Document Set ID: …", and any line among a page's first or last
// two that repeats (numbers aside) at the edge of at least three pages
// and half of them. Lines inside a page are never dropped, however often
// the council repeats its words.
func cleanLines(pages []Page) []line {
	split := make([][]string, len(pages))
	edge := make([]map[int]bool, len(pages))
	seen := map[string]map[int]bool{}
	for i, p := range pages {
		split[i] = strings.Split(strings.ReplaceAll(p.Text, "\r", ""), "\n")
		edge[i] = map[int]bool{}
		var filled []int
		for j, raw := range split[i] {
			if strings.TrimSpace(raw) != "" {
				filled = append(filled, j)
			}
		}
		for k, j := range filled {
			if (k < 2 || k >= len(filled)-2) && footerLike(split[i][j]) {
				edge[i][j] = true
				if key := footerKey(split[i][j]); key != "" {
					if seen[key] == nil {
						seen[key] = map[int]bool{}
					}
					seen[key][p.Page] = true
				}
			}
		}
	}
	repeated := func(key string) bool {
		return key != "" && len(pages) >= 3 && len(seen[key]) >= 3 && len(seen[key])*2 >= len(pages)
	}
	var out []line
	for i, p := range pages {
		for j, raw := range split[i] {
			t := strings.TrimRightFunc(raw, unicode.IsSpace)
			lower := strings.ToLower(strings.TrimSpace(t))
			if pageOfRe.MatchString(lower) || edge[i][j] && repeated(footerKey(raw)) {
				continue
			}
			skip := false
			for _, w := range footerWords {
				skip = skip || strings.HasPrefix(lower, w)
			}
			if !skip {
				out = append(out, line{text: t, page: p.Page})
			}
		}
	}
	return out
}

func footerKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 4 {
		return ""
	}
	return digitsRe.ReplaceAllString(s, "#")
}

var (
	bareDARe = regexp.MustCompile(`(?i)\bDA\d{6,12}\b`)
	refRe    = regexp.MustCompile(`(?i)\b(DA|MOD|CDC)\s*[/\-]?\s*(\d{1,4})\s*[/\-.]\s*(\d{1,5})(?:\s*[/\-.]\s*(\d{1,5}))?(?:\s*[/\-.]\s*([A-Z]))?\b`)
	modWords = []string{"notice of modification", "modification of development consent", "section 4.55", "s4.55", "s 4.55", "modification application", "modified determination"}
)

// references finds the consent's reference and, for a modification, the
// modification's own, as printed ("DA/2025/0733", "DA0167/21",
// "DA201500704"). The first DA reference is the consent; a MOD reference
// is the modification's.
func references(text string) (consent, mod string, modification bool) {
	lower := strings.ToLower(text)
	for _, w := range modWords {
		modification = modification || strings.Contains(lower, w)
	}
	type hit struct {
		at  int
		ref string
	}
	var hits []hit
	for _, m := range refRe.FindAllStringIndex(text, 20) {
		hits = append(hits, hit{m[0], strings.Join(strings.Fields(text[m[0]:m[1]]), "")})
	}
	for _, m := range bareDARe.FindAllStringIndex(text, 20) {
		hits = append(hits, hit{m[0], text[m[0]:m[1]]})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
	for _, h := range hits {
		switch {
		case strings.HasPrefix(strings.ToUpper(h.ref), "MOD"):
			if mod == "" {
				mod = h.ref
			}
			modification = true
		case consent == "":
			consent = h.ref
		}
	}
	return consent, mod, modification
}

var (
	labelledDate = regexp.MustCompile(`(?i)(?:date of determination|determination date|determined on|date of decision|consent date)\s*[:\-]?\s*([^\n]{6,40})`)
	modDate      = regexp.MustCompile(`(?i)date of modification`)
	anyDate      = regexp.MustCompile(`(?i)\b(\d{1,2}(?:st|nd|rd|th)?\s+[A-Za-z]{3,9},?\s+\d{4}|\d{1,2}/\d{1,2}/\d{4}|\d{4}-\d{2}-\d{2})\b`)
)

// determined reads the determination date by rule: a modification's own
// date first, then a labelled date, else the first date on the notice. It
// orders notices; nothing else.
func determined(text string, modification bool) string {
	if at := modDate.FindStringIndex(text); at != nil {
		// The label and its date can be several lines apart in a form.
		if d := parseDate(text[at[1]:min(len(text), at[1]+160)]); d != "" {
			return d
		}
		// A modification restates the consent's own date ("determined on 18
		// October 2016"); its own is the latest date on its first page.
	} else if modification {
		latest := ""
		for _, m := range anyDate.FindAllString(text[:min(len(text), 4000)], 40) {
			if d := parseDate(m); d > latest {
				latest = d
			}
		}
		if latest != "" {
			return latest
		}
	}
	if m := labelledDate.FindStringSubmatch(text); m != nil {
		if d := parseDate(m[1]); d != "" {
			return d
		}
	}
	for _, m := range anyDate.FindAllString(text, 5) {
		if d := parseDate(m); d != "" {
			return d
		}
	}
	return ""
}

func parseDate(s string) string {
	m := anyDate.FindString(s)
	if m == "" {
		return ""
	}
	m = ordinalRe.ReplaceAllString(m, "$1")
	m = strings.ReplaceAll(m, ",", "")
	for _, layout := range []string{"2 January 2006", "2 Jan 2006", "2/1/2006", "2006-01-02"} {
		if t, err := time.Parse(layout, m); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// ---- Markdown tables

var tableNoRe = regexp.MustCompile(`(?i)^(no\.?|number|condition( no\.?)?|#)$`)

// tableConditions reads a Markdown table with a No. column as one condition
// a row. A column named When or Stage is the row's stage heading.
func tableConditions(lines []line) ([]Condition, bool) {
	var out []Condition
	found := false
	for i := 0; i < len(lines); i++ {
		head := cells(lines[i].text)
		if head == nil || i+1 >= len(lines) || !separator(lines[i+1].text) {
			continue
		}
		no, when := -1, -1
		for j, h := range head {
			switch {
			case tableNoRe.MatchString(h):
				no = j
			case whenRe.MatchString(h):
				when = j
			}
		}
		if no < 0 {
			continue
		}
		found = true
		for i += 2; i < len(lines); i++ {
			row := cells(lines[i].text)
			if row == nil {
				break
			}
			if no >= len(row) || !conditionNumber(row[no]) {
				continue
			}
			c := Condition{Number: strings.TrimSuffix(strings.TrimSpace(row[no]), "."), Page: lines[i].page}
			var text []string
			for j, v := range row {
				switch j {
				case no:
				case when:
					c.StageHeading = v
				default:
					if strings.TrimSpace(v) != "" {
						text = append(text, v)
					}
				}
			}
			c.Text = strings.Join(text, " ")
			c.Stage, c.Deferred = StageOf(c.StageHeading)
			c.Marker, c.MarkerBy = marker(c.Text)
			out = append(out, c)
		}
	}
	return out, found && len(out) > 0
}

func cells(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "|") || !strings.HasSuffix(s, "|") || len(s) < 3 {
		return nil
	}
	parts := strings.Split(s[1:len(s)-1], "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func separator(s string) bool {
	c := cells(s)
	if c == nil {
		return false
	}
	for _, v := range c {
		if strings.Trim(v, ":- ") != "" {
			return false
		}
	}
	return true
}

var (
	numberOnly = regexp.MustCompile(`^([A-Z]{0,2})(\d{1,3})\.?$`)
	whenRe     = regexp.MustCompile(`(?i)^(when|stage|timing)$`)
	monthRe    = regexp.MustCompile(`(?i)^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?(\s+\d{2,4})?\b`)
	ordinalRe  = regexp.MustCompile(`(?i)(\d)(st|nd|rd|th)`)
	punctRe    = regexp.MustCompile(`^\s*(?:\*\*)?(?:Condition\s+|Cond\.\s*)?\(?[A-Z]{0,2}\s?\d{1,3}(\.|\)|:|\s-\s|\s–\s)`)
)

func conditionNumber(s string) bool { return numberOnly.MatchString(strings.TrimSpace(s)) }

// ---- Printed lines

// startRe is a condition's opening line: an optional "Condition", a number
// with an optional letter prefix ("46", "A3", "B1") and its punctuation,
// then the rest of the line.
var startRe = regexp.MustCompile(`^\s*(?:\*\*)?(Condition\s+|Cond\.\s*)?\(?([A-Z]{0,2})\s?(\d{1,3})(?:\.|\)|:|\s-\s|\s–\s)?(?:\*\*)?(?:\s+(.*))?$`)

type seq struct {
	prefix string
	n      int
}

// lineConditions walks the lines. A number counts only as the next in its
// sequence (a jump of up to four, so a scan's missing condition still
// leaves the next one found), which keeps a cross-reference or a wrapped
// figure out. "(a)" and "i)" are not numbers here, so they stay inside.
// "Advisory notes" or "Notes" ends the conditions.
func lineConditions(lines []line) []Condition {
	var out []Condition
	last := map[string]int{}
	stageHeading := ""
	var cur *Condition
	var body []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.Text = strings.TrimSpace(strings.Join(body, "\n"))
		if cur.Heading == "" && cur.Text != "" {
			// "46. Landscape plan" with its words on the lines below. A first
			// line that runs on into the next ("… issued in" / "relation to")
			// is the condition's words, not a heading.
			first, rest, _ := strings.Cut(cur.Text, "\n")
			rest = strings.TrimSpace(rest)
			if headingLike(first) && len(strings.Fields(first)) <= 10 && rest != "" && unicode.IsUpper([]rune(rest)[0]) {
				cur.Heading, cur.Text = strings.TrimSpace(first), strings.TrimSpace(rest)
			}
		}
		cur.Marker, cur.MarkerBy = marker(cur.Heading + "\n" + cur.Text)
		out = append(out, *cur)
		cur, body = nil, nil
	}
	for i, l := range lines {
		t := strings.TrimSpace(l.text)
		if t == "" {
			if cur != nil {
				body = append(body, "")
			}
			continue
		}
		if endOfConditions(t) && len(out)+boolInt(cur != nil) > 0 {
			break
		}
		if s, rest, explicit, ok := conditionStart(t); ok && next(last, s, explicit) {
			flush()
			last[s.prefix] = s.n
			cur = &Condition{Number: s.prefix + strconv.Itoa(s.n), Page: l.page, EndPage: l.page, StageHeading: stageHeading}
			cur.Stage, cur.Deferred = StageOf(stageHeading)
			if rest != "" {
				body = append(body, rest)
			}
			continue
		}
		if stageLine(t) && followedByCondition(lines, i, last, capitals(t)) {
			flush()
			stageHeading = strings.Trim(strings.TrimLeft(t, "#* "), "*: ")
			continue
		}
		if cur != nil {
			body = append(body, t)
			cur.EndPage = l.page
		}
	}
	flush()
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func conditionStart(t string) (seq, string, bool, bool) {
	m := startRe.FindStringSubmatch(t)
	if m == nil {
		return seq{}, "", false, false
	}
	n, err := strconv.Atoi(m[3])
	if err != nil || n < 1 {
		return seq{}, "", false, false
	}
	rest := strings.TrimSpace(strings.Trim(m[4], "* "))
	explicit := m[1] != ""
	// "12 metres from the boundary" is a wrapped line, not condition 12: a
	// bare number needs punctuation or a capital after it.
	punct := punctRe.MatchString(t)
	if !explicit && !punct && (rest == "" || !unicode.IsUpper([]rune(rest)[0]) || monthRe.MatchString(rest) || strings.HasPrefix(m[3], "0")) {
		// "04 August 2023" is a date in a schedule, not condition 4.
		return seq{}, "", false, false
	}
	return seq{prefix: m[2], n: n}, rest, explicit, true
}

// next reports whether s continues its sequence: the first number of a
// prefix is 1 to 5 (or any, written "Condition 13"), and later ones go up
// by at most four.
func next(last map[string]int, s seq, explicit bool) bool {
	prev, ok := last[s.prefix]
	if !ok {
		return s.n <= 5 || explicit
	}
	if explicit {
		return s.n > prev
	}
	return s.n > prev && s.n <= prev+4
}

var stageWords = []string{"general", "condition", "prior to", "before", "during", "occupation", "construction certificate", "subdivision",
	"commencement", "operational", "ongoing", "site works", "deferred", "post", "use of the"}

// stageLine is a printed stage heading: short, no closing full stop, and
// either in capitals, a Markdown heading, or naming a stage.
func stageLine(t string) bool {
	if strings.HasSuffix(t, ".") || strings.HasSuffix(t, ";") || len([]rune(t)) > 110 || len(strings.Fields(t)) > 14 {
		return false
	}
	if strings.HasPrefix(t, "#") {
		return true
	}
	letters, upper := 0, 0
	for _, r := range t {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters >= 6 && upper == letters {
		return true
	}
	lower := strings.ToLower(t)
	for _, w := range stageWords {
		if strings.Contains(lower, w) && unicode.IsUpper([]rune(t)[0]) {
			return true
		}
	}
	return false
}

// followedByCondition: a heading sits right above a condition. A heading in
// capitals may have a short paragraph between it and its first condition
// ("For the purpose of interpreting this consent …"); any other heading
// must be followed by the condition itself.
func followedByCondition(lines []line, i int, last map[string]int, capitals bool) bool {
	reach := 3
	if capitals {
		reach = 10
	}
	for j := i + 1; j < len(lines) && j <= i+reach; j++ {
		t := strings.TrimSpace(lines[j].text)
		if t == "" {
			continue
		}
		s, _, explicit, ok := conditionStart(t)
		if ok && next(last, s, explicit) {
			return true
		}
		if !capitals {
			return false
		}
	}
	return false
}

// capitals is a heading set in capitals: at least two words, no digits,
// not a bracketed aside.
func capitals(t string) bool {
	if strings.HasPrefix(t, "(") || strings.ContainsAny(t, "0123456789") || len(strings.Fields(t)) < 2 {
		return false
	}
	letters := 0
	for _, r := range t {
		if unicode.IsLetter(r) {
			letters++
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return letters >= 6
}

var endRe = regexp.MustCompile(`(?i)^(#+\s*)?(\*\*)?(advisory notes?|advisings?|notes|advice to applicant|general advice)(\*\*)?\s*:?\s*$`)

func endOfConditions(t string) bool { return endRe.MatchString(t) }

// headingLike is a printed condition heading: short, starting with a
// capital, not closing like a sentence.
func headingLike(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 120 || len(strings.Fields(s)) > 16 {
		return false
	}
	last := s[len(s)-1]
	if last == '.' || last == ',' || last == ';' || last == ':' {
		return false
	}
	return unicode.IsUpper([]rune(s)[0]) || unicode.IsDigit([]rune(s)[0])
}

var (
	deletedRe = regexp.MustCompile(`(?i)^\s*[\[(]?\s*(deleted|condition deleted|this condition (has been|is) deleted)\b`)
	changedRe = regexp.MustCompile(`(?i)\b(amended|modified|inserted|added)\b[^\n]{0,40}\b(MOD|modification|s\s?4\.55|section 4\.55)`)
)

var (
	bracketRe = regexp.MustCompile(`(?i)\(\s*(amended|deleted|inserted|added|modified)\b[^)]{0,80}\)`)
	modRefRe  = regexp.MustCompile(`(?i)\bMOD\s*/\s*\d{4}\s*/\s*\d{1,5}\b`)
)

// marker reads a modification's restated condition: deleted, or changed,
// and the modification that did it when the notice names one. A condition
// restated with several bracketed markers ("(Amended – 13 October 2022 –
// MOD/2022/0191)", "(Deleted – …)") takes its last.
func marker(text string) (kind, by string) {
	t := strings.TrimSpace(text)
	if all := bracketRe.FindAllString(t, -1); len(all) > 0 {
		last := all[len(all)-1]
		kind = "changed"
		if strings.Contains(strings.ToLower(last), "deleted") {
			kind = "deleted"
		}
		return kind, strings.Join(strings.Fields(modRefRe.FindString(last)), "")
	}
	switch {
	case deletedRe.MatchString(t) || deletedRe.MatchString(strings.TrimSpace(after(t, "\n"))):
		return "deleted", ""
	case changedRe.MatchString(t):
		return "changed", strings.Join(strings.Fields(modRefRe.FindString(t)), "")
	}
	return "", ""
}

func after(s, sep string) string {
	_, rest, _ := strings.Cut(s, sep)
	return rest
}

// StageOf maps a printed stage heading to a gate, in the plan's order:
// subdivision certificate; occupation; construction certificate with prior
// or before; commencement, prior to construction, prior to any works;
// during, site works; operational, ongoing. "Deferred commencement" is
// before work starts, flagged. Anything else ("General", none) is "": the
// gate is then Jev's question.
func StageOf(heading string) (stage string, deferred bool) {
	h := strings.ToLower(heading)
	switch {
	case h == "":
		return "", false
	case strings.Contains(h, "deferred commencement"):
		return delivery.GateWorkStarts, true
	case strings.Contains(h, "subdivision certificate"):
		return delivery.GateSubdivision, false
	case strings.Contains(h, "occupation"):
		return delivery.GateOccupation, false
	case strings.Contains(h, "construction certificate") && (strings.Contains(h, "prior") || strings.Contains(h, "before")):
		return delivery.GateConstructionCertificate, false
	case strings.Contains(h, "commenc") || strings.Contains(h, "prior to construction") || strings.Contains(h, "prior to any works") ||
		strings.Contains(h, "before work") || strings.Contains(h, "works start") || strings.Contains(h, "work starts"):
		return delivery.GateWorkStarts, false
	case strings.Contains(h, "during") || strings.Contains(h, "site works"):
		return delivery.GateDuringWorks, false
	case strings.Contains(h, "operational") || strings.Contains(h, "ongoing"):
		return delivery.GateOngoing, false
	}
	return "", false
}

// gaps lists the numbers missing from each prefix's run from 1 (or the
// prefix's first number) to its last, as ranges.
func gaps(cs []Condition) []string {
	byPrefix := map[string][]int{}
	var order []string
	for _, c := range cs {
		m := numberOnly.FindStringSubmatch(c.Number)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		if _, ok := byPrefix[m[1]]; !ok {
			order = append(order, m[1])
		}
		byPrefix[m[1]] = append(byPrefix[m[1]], n)
	}
	var out []string
	for _, p := range order {
		ns := byPrefix[p]
		sort.Ints(ns)
		have := map[int]bool{}
		for _, n := range ns {
			have[n] = true
		}
		start := -1
		for n := 1; n <= ns[len(ns)-1]+1; n++ {
			switch {
			case !have[n] && n <= ns[len(ns)-1] && start < 0:
				start = n
			case (have[n] || n > ns[len(ns)-1]) && start >= 0:
				if start == n-1 {
					out = append(out, fmt.Sprintf("%s%d", p, start))
				} else {
					out = append(out, fmt.Sprintf("%s%d–%s%d", p, start, p, n-1))
				}
				start = -1
			}
		}
	}
	return out
}

// Kept is a condition's text as a row keeps it: at most 12,000 characters,
// with "continues on page N" when cut.
func Kept(c Condition) string {
	r := []rune(c.Text)
	if len(r) <= delivery.MaxConditionText {
		return c.Text
	}
	tail := fmt.Sprintf(" … continues on page %d", max(c.Page, c.EndPage))
	return string(r[:delivery.MaxConditionText-len([]rune(tail))]) + tail
}

// footerLike is a line that can be a running header or footer: not a
// condition's opening line, and not closing like a sentence.
func footerLike(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" || strings.HasSuffix(t, ".") || strings.HasSuffix(t, ":") || strings.HasSuffix(t, ";") {
		return false
	}
	_, _, _, start := conditionStart(t)
	return !start
}
