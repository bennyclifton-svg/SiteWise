package intake

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"sitewise/internal/identity"
)

const (
	FieldNumber   = "number"
	FieldRevision = "revision"
	FieldTitle    = "title"
	FieldDate     = "date"

	OriginFilename = "filename"
	OriginText     = "text"
)

// Provenance locates a candidate in the filename or in one identity run.
// Start and End are byte offsets into that filename or run text.
type Provenance struct {
	Origin   string
	Page     int
	Rotation int
	Sheet    string
	Cell     string
	Merge    string
	Table    int
	Row      int
	Col      int
	Start    int
	End      int
	Labeled  bool
	Heading  bool
	Run      int
}

// Candidate is one possible field value. Display is the literal source text.
// Normalized is only for comparison.
type Candidate struct {
	Field      string
	Display    string
	Normalized string
	Provenance Provenance
}

// Harvest over-finds number, revision, title, and date candidates.
// It does not choose among them.
func Harvest(filename string, text identity.Text) []Candidate {
	out := harvestFilename(filename)
	return append(out, harvestRuns(text)...)
}

// titleWindow is how many runs either side of a standalone sheet-number cell
// are read as its title-block neighbours. Title blocks often list values
// apart from their captions, so a title has no label to follow.
const titleWindow = 2

const (
	// Dates stay ahead of numbers so 12/03/2024 is not a document number.
	dateRE = `\d{4}-\d{2}-\d{2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}|\d{1,2}\s+(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\s+\d{2,4}`
	// A bare P-series token is a preliminary revision, not a document number.
	prelimRE = `P\d{1,3}`
	// Hyphenated sheet numbers, lettered job codes, 5–6 digit project numbers,
	// and compact sheets with at least three digits.
	numberRE = `[A-Z]{1,4}(?:-[A-Z]{1,4}){0,2}-\d{2,4}|[A-Z]{3,}\d{2,}|\d{5,6}|[A-Z]{1,3}\d{3,4}(?:-\d{2,4})?`
)

var (
	datePattern   = regexp.MustCompile(`(?i)\b(?:` + dateRE + `)\b`)
	numberPattern = regexp.MustCompile(`(?i)\b(?:` + numberRE + `)\b`)
	prelimPattern = regexp.MustCompile(`(?i)\b(?:` + prelimRE + `)\b`)
	revBracket    = regexp.MustCompile(`(?i)\[([A-Z0-9]{1,4})\]`)
	revWord       = regexp.MustCompile(`(?i)\brev(?:ision)?\s+([A-Z0-9]{1,4})\b`)
)

type span struct{ start, end int }

func harvestFilename(name string) []Candidate {
	stem, base := fileStem(name)
	if stem == "" {
		return nil
	}
	var covered []span
	var out []Candidate
	add := func(field, display string, start, end int, cover span) {
		c, ok := newCandidate(field, display, filenameProv(base+start, base+end))
		if !ok {
			return
		}
		out = append(out, c)
		covered = append(covered, cover)
	}
	for _, loc := range datePattern.FindAllStringIndex(stem, -1) {
		add(FieldDate, stem[loc[0]:loc[1]], loc[0], loc[1], span{loc[0], loc[1]})
	}
	for _, loc := range revBracket.FindAllStringSubmatchIndex(stem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		add(FieldRevision, stem[loc[2]:loc[3]], loc[2], loc[3], span{loc[0], loc[1]})
	}
	for _, loc := range revWord.FindAllStringSubmatchIndex(stem, -1) {
		if overlaps(covered, loc[2], loc[3]) {
			continue
		}
		add(FieldRevision, stem[loc[2]:loc[3]], loc[2], loc[3], span{loc[0], loc[1]})
	}
	for _, loc := range prelimPattern.FindAllStringIndex(stem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		add(FieldRevision, stem[loc[0]:loc[1]], loc[0], loc[1], span{loc[0], loc[1]})
	}
	for _, loc := range numberPattern.FindAllStringIndex(stem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		display := stem[loc[0]:loc[1]]
		if fullMatch(datePattern, display) || fullMatch(prelimPattern, display) {
			continue
		}
		// S204-03 may be sheet S204 at revision 03. A stem that is only that
		// is the sheet-and-revision naming convention, so code splits it.
		// Inside a longer name both readings are offered and code does not
		// choose; Jev reads the fused literal too literally to be asked.
		m := suffixedSheet.FindStringSubmatchIndex(display)
		if m == nil || loc[0] != 0 || loc[1] != len(stem) {
			add(FieldNumber, display, loc[0], loc[1], span{loc[0], loc[1]})
		}
		if m != nil {
			add(FieldNumber, display[m[2]:m[3]], loc[0]+m[2], loc[0]+m[3], span{loc[0], loc[1]})
			add(FieldRevision, display[m[4]:m[5]], loc[0]+m[4], loc[0]+m[5], span{loc[0], loc[1]})
		}
	}
	if end := leadToken(stem); end > 0 && isShortSheet(stem[:end]) && !overlaps(covered, 0, end) {
		add(FieldNumber, stem[:end], 0, end, span{0, end})
	}
	if shortName(stem) {
		return out
	}
	if display, start, end, ok := titleFromGaps(stem, covered); ok {
		c, made := newCandidate(FieldTitle, display, filenameProv(base+start, base+end))
		if made {
			out = append(out, c)
		}
	}
	return out
}

func harvestRuns(text identity.Text) []Candidate {
	labelOnly := make([]string, len(text.Runs))
	out := make([]Candidate, 0, len(text.Runs)*4)
	for i, run := range text.Runs {
		var only string
		out, only = appendRun(out, run, i)
		labelOnly[i] = only
	}
	for i := 0; i < len(text.Runs)-1; i++ {
		if labelOnly[i] == "" {
			continue
		}
		c, ok := candidateFromRun(text.Runs[i+1], i+1, labelOnly[i])
		if !ok {
			continue
		}
		out = mergeCandidate(out, c)
	}
	return appendNeighbourTitles(out, text)
}

// appendNeighbourTitles offers title-like cells near a cell that holds only
// a sheet number. They are unlabeled candidates; Jev chooses among them.
func appendNeighbourTitles(out []Candidate, text identity.Text) []Candidate {
	seen := map[int]bool{}
	for i, run := range text.Runs {
		v := strings.TrimSpace(run.Text)
		if v == "" || strings.ContainsAny(v, " \t") || !(isNumber(v) || isShortSheet(v)) {
			continue
		}
		for j := i - titleWindow; j <= i+titleWindow; j++ {
			if j < 0 || j >= len(text.Runs) || j == i || seen[j] {
				continue
			}
			display := strings.TrimSpace(text.Runs[j].Text)
			if !cellTitle(display) {
				continue
			}
			start := strings.Index(text.Runs[j].Text, display)
			if c, ok := newCandidate(FieldTitle, display, textProv(text.Runs[j].Source, j, start, start+len(display), false)); ok {
				out = append(out, c)
				seen[j] = true
			}
		}
	}
	return out
}

// cellTitle is a short multi-word cell that reads as a title, not a note.
func cellTitle(s string) bool {
	words := strings.Fields(s)
	return len(words) >= 2 && len(words) <= 12 && len(s) <= 120 && looksLikeTitle(s)
}

type word struct{ start, end int }

func appendRun(out []Candidate, run identity.Run, index int) ([]Candidate, string) {
	text := run.Text
	var storage [24]word
	tokens := words(text, storage[:0])
	labelOnly := ""
	emittedTitle := false
	for i := 0; i < len(tokens); {
		if field, n, ok := labelAt(text, tokens, i); ok {
			next := i + n
			if field == FieldTitle {
				if display, start, end, took, titled := takeTitle(text, tokens, next); titled {
					if c, made := newCandidate(FieldTitle, display, textProv(run.Source, index, start, end, true)); made {
						out = append(out, c)
						emittedTitle = true
					}
					i = next + took
					continue
				}
			} else if next < len(tokens) && accepts(field, text[tokens[next].start:tokens[next].end]) {
				tok := tokens[next]
				if c, made := newCandidate(field, text[tok.start:tok.end], textProv(run.Source, index, tok.start, tok.end, true)); made {
					out = append(out, c)
				}
				i = next + 1
				continue
			}
			if i == 0 && next == len(tokens) {
				labelOnly = field
			}
			i = next
			continue
		}
		tok := tokens[i]
		display := text[tok.start:tok.end]
		standalone := len(tokens) == 1
		if standalone && isShortSheet(display) {
			if c, ok := newCandidate(FieldNumber, display, textProv(run.Source, index, tok.start, tok.end, false)); ok {
				out = append(out, c)
			}
			i++
			continue
		}
		// A P-number inside a note (a pit, a pump) is not a revision unless
		// the note is about issue or revision.
		if isPrelim(display) && !standalone && !revisionContext(text) {
			i++
			continue
		}
		if kind := kindOf(display); kind != "" {
			if c, ok := newCandidate(kind, display, textProv(run.Source, index, tok.start, tok.end, false)); ok {
				out = append(out, c)
			}
			i++
			continue
		}
		if isDay(display) && i+2 < len(tokens) && isMonth(text[tokens[i+1].start:tokens[i+1].end]) && isYear(text[tokens[i+2].start:tokens[i+2].end]) {
			start, end := tok.start, tokens[i+2].end
			if c, ok := newCandidate(FieldDate, text[start:end], textProv(run.Source, index, start, end, false)); ok {
				out = append(out, c)
			}
			i += 3
			continue
		}
		i++
	}
	if run.Source.Heading && !emittedTitle {
		display := strings.TrimSpace(text)
		if looksLikeTitle(display) {
			start := strings.Index(text, display)
			if start >= 0 {
				if c, ok := newCandidate(FieldTitle, display, textProv(run.Source, index, start, start+len(display), false)); ok {
					out = append(out, c)
				}
			}
		}
	}
	return out, labelOnly
}

func words(s string, buf []word) []word {
	buf = buf[:0]
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		j := i
		for j < len(s) && s[j] != ' ' && s[j] != '\t' {
			j++
		}
		buf = append(buf, word{i, j})
		i = j
	}
	return buf
}

func candidateFromRun(run identity.Run, index int, field string) (Candidate, bool) {
	display := strings.TrimSpace(run.Text)
	start := strings.Index(run.Text, display)
	if display == "" || start < 0 {
		return Candidate{}, false
	}
	end := start + len(display)
	switch field {
	case FieldNumber:
		if !isNumber(display) {
			return Candidate{}, false
		}
	case FieldRevision:
		if _, ok := parseRevision(display); !ok {
			return Candidate{}, false
		}
	case FieldDate:
		if !isDate(display) {
			return Candidate{}, false
		}
	case FieldTitle:
		if !looksLikeTitle(display) {
			return Candidate{}, false
		}
	default:
		return Candidate{}, false
	}
	return newCandidate(field, display, textProv(run.Source, index, start, end, true))
}

func mergeCandidate(out []Candidate, c Candidate) []Candidate {
	for i := range out {
		if out[i].Provenance.Origin != OriginText || out[i].Provenance.Run != c.Provenance.Run {
			continue
		}
		if out[i].Provenance.Start != c.Provenance.Start || out[i].Provenance.End != c.Provenance.End {
			continue
		}
		out[i].Field = c.Field
		out[i].Display = c.Display
		out[i].Normalized = c.Normalized
		out[i].Provenance.Labeled = true
		return out
	}
	return append(out, c)
}

func newCandidate(field, display string, prov Provenance) (Candidate, bool) {
	normalized, ok := normalize(field, display)
	if !ok {
		return Candidate{}, false
	}
	prov.Labeled = prov.Labeled || prov.Origin == OriginFilename
	return Candidate{Field: field, Display: display, Normalized: normalized, Provenance: prov}, true
}

// Normalize is the comparison form intake uses for a harvested value. ok is
// false when the value is not a valid number, revision, title or date.
func Normalize(field, display string) (string, bool) {
	return normalize(field, display)
}

func normalize(field, display string) (string, bool) {
	switch field {
	case FieldNumber:
		n := normalizeNumber(display)
		return n, n != ""
	case FieldRevision:
		return revisionNormalized(display)
	case FieldDate:
		n := strings.TrimSpace(display)
		return n, n != ""
	case FieldTitle:
		n := normalizeTitle(display)
		return n, n != ""
	default:
		return "", false
	}
}

func normalizeNumber(s string) string {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c == ' ' || c == '\t' || c == '_' || c >= 0x80 {
			return normalizeNumberSlow(s)
		}
	}
	return s
}

func normalizeNumberSlow(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t':
			continue
		case '\u2013', '\u2014', '\u2212', '_':
			b.WriteByte('-')
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

func normalizeTitle(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '_', '-', '\u2013', '\u2014':
			b.WriteByte(' ')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func looksLikeTitle(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if isCaption(s) {
		return false
	}
	if isDate(s) || isNumber(s) {
		return false
	}
	if _, ok := parseRevision(s); ok && !strings.ContainsAny(s, " \t") {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func labelAt(text string, tokens []word, i int) (field string, n int, ok bool) {
	a := text[tokens[i].start:tokens[i].end]
	if !wordLabel(a) {
		return "", 0, false
	}
	switch {
	case eq(a, "date"):
		return FieldDate, 1, true
	case eq(a, "title"):
		return FieldTitle, 1, true
	case eq(a, "rev") || eq(a, "revision"):
		return FieldRevision, 1, true
	}
	if i+1 >= len(tokens) {
		return "", 0, false
	}
	b := text[tokens[i+1].start:tokens[i+1].end]
	switch {
	case eq(a, "drawing") && eq(b, "title"):
		return FieldTitle, 2, true
	case eq(a, "drawing") && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case (eq(a, "drg") || eq(a, "dwg")) && eq(b, "no"):
		return FieldNumber, 2, true
	case eq(a, "document") && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case eq(a, "doc") && eq(b, "ref"):
		return FieldNumber, 2, true
	case eq(a, "sheet") && eq(b, "title"):
		return FieldTitle, 2, true
	case eq(a, "sheet") && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case eq(a, "version") && eq(b, "no"):
		return FieldRevision, 2, true
	case eq(a, "issue") && eq(b, "date"):
		return FieldDate, 2, true
	default:
		return "", 0, false
	}
}

func takeTitle(text string, tokens []word, from int) (display string, start, end, took int, ok bool) {
	if from >= len(tokens) {
		return "", 0, 0, 0, false
	}
	last := from
	for last < len(tokens) {
		if _, _, isLabel := labelAt(text, tokens, last); isLabel {
			break
		}
		last++
	}
	if last == from {
		return "", 0, 0, 0, false
	}
	rawStart := tokens[from].start
	rawEnd := tokens[last-1].end
	raw := text[rawStart:rawEnd]
	display = strings.Trim(raw, " \t-_:.")
	if !looksLikeTitle(display) {
		return "", 0, 0, 0, false
	}
	rel := strings.Index(raw, display)
	return display, rawStart + rel, rawStart + rel + len(display), last - from, true
}

func accepts(field, display string) bool {
	switch field {
	case FieldNumber:
		return isNumber(display)
	case FieldDate:
		return isDate(display)
	case FieldRevision:
		_, ok := parseRevision(display)
		return ok
	default:
		return false
	}
}

func kindOf(s string) string {
	switch {
	case isDate(s):
		return FieldDate
	case isPrelim(s):
		return FieldRevision
	case isNumber(s):
		return FieldNumber
	default:
		return ""
	}
}

func isDate(s string) bool {
	if len(s) == 10 && s[4] == '-' && s[7] == '-' && digitsOnly(s[:4]) && digitsOnly(s[5:7]) && digitsOnly(s[8:]) {
		return calendar(atoiOr(s[8:]), atoiOr(s[5:7]))
	}
	sep := byte(0)
	parts := 0
	run := 0
	first := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '/' || c == '.' {
			if sep == 0 {
				sep = c
			} else if c != sep {
				return false
			}
			if parts == 0 {
				first = run
			}
			if run < 1 || run > 4 {
				return false
			}
			parts++
			run = 0
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		run++
	}
	if parts != 2 || first < 1 || first > 2 || run < 2 || run > 4 {
		return false
	}
	// Day first, as Australian documents write it.
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == rune(sep) })
	return calendar(atoiOr(fields[0]), atoiOr(fields[1]))
}

func calendar(day, month int) bool {
	return day >= 1 && day <= 31 && month >= 1 && month <= 12
}

func atoiOr(s string) int {
	n, ok := atoi(s)
	if !ok {
		return -1
	}
	return n
}

// isCaption is title-block caption text: a label, not a value.
func isCaption(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ":") {
		return true
	}
	return captions[strings.Join(strings.Fields(strings.ToLower(strings.Trim(s, ".:"))), " ")]
}

var captions = map[string]bool{
	"rev": true, "revision": true, "date": true, "drawing": true, "no": true, "number": true,
	"sheet": true, "title": true, "project": true, "drawn": true, "drawn by": true,
	"checked": true, "checked by": true, "designed": true, "designed by": true,
	"approved": true, "approved by": true, "architect": true, "client": true,
	"scale": true, "job": true, "job no": true, "project no": true, "project title": true,
	"drawing no": true, "drawing number": true, "drawing title": true, "sheet title": true,
	"sheet no": true, "status": true, "issue": true, "amendment": true, "amendments": true,
	"north": true, "notes": true, "consultant": true, "engineer": true, "description": true,
}

// revisionContext reports whether a run talks about issue or revision.
func revisionContext(s string) bool {
	l := strings.ToLower(s)
	for _, w := range []string{"issue", "rev", "amend", "status", "prelim"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// isShortSheet is a one- or two-letter sheet id with two digits, such as
// E01. It is only a number candidate where it stands alone.
func isShortSheet(s string) bool {
	if len(s) < 3 || len(s) > 4 || isPrelim(s) {
		return false
	}
	letters, digits, ok := trailingDigits(s)
	return ok && len(letters) <= 2 && len(digits) == 2 && strings.ToUpper(letters) == letters
}

// leadToken is the end of the first filename token, before a space, hyphen
// or underscore.
func leadToken(stem string) int {
	for i := 0; i < len(stem); i++ {
		switch stem[i] {
		case ' ', '-', '_':
			return i
		}
	}
	return len(stem)
}

// shortName is a Windows 8.3 alias such as E01-EL~1: truncated, so its words
// are not a title.
func shortName(stem string) bool {
	i := strings.IndexByte(stem, '~')
	return i >= 0 && i+1 < len(stem) && stem[i+1] >= '0' && stem[i+1] <= '9'
}

var suffixedSheet = regexp.MustCompile(`^([A-Za-z]{1,3}\d{3,4})-(\d{2})$`)

func isNumber(s string) bool {
	if s == "" || isDate(s) || isPrelim(s) {
		return false
	}
	if digitsOnly(s) {
		return len(s) == 5 || len(s) == 6
	}
	if !strings.Contains(s, "-") {
		letters, digits, ok := trailingDigits(s)
		if !ok {
			return false
		}
		if len(letters) >= 3 && len(digits) >= 2 {
			return true
		}
		return len(letters) >= 1 && len(letters) <= 3 && len(digits) >= 3 && len(digits) <= 4
	}
	return hyphenatedNumber(s)
}

func isPrelim(s string) bool {
	if len(s) < 2 || len(s) > 4 || (s[0] != 'P' && s[0] != 'p') {
		return false
	}
	return digitsOnly(s[1:])
}

func trailingDigits(s string) (letters, digits string, ok bool) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	if i == 0 || i == len(s) || !lettersASCII(s[:i]) {
		return "", "", false
	}
	return s[:i], s[i:], true
}

func hyphenatedNumber(s string) bool {
	hyphens := strings.Count(s, "-")
	if hyphens < 1 || hyphens > 3 {
		return false
	}
	last := strings.LastIndexByte(s, '-')
	tail := s[last+1:]
	if len(tail) < 2 || len(tail) > 4 || !digitsOnly(tail) {
		return false
	}
	head := s[:last]
	if hyphens == 1 && compactSheet(head) {
		return true
	}
	seg := 0
	for i := 0; i <= len(head); i++ {
		if i == len(head) || head[i] == '-' {
			if seg < 1 || seg > 4 || !lettersASCII(head[i-seg:i]) {
				return false
			}
			seg = 0
			continue
		}
		if !isLetter(head[i]) {
			return false
		}
		seg++
	}
	return true
}

func compactSheet(s string) bool {
	i := 0
	for i < len(s) && isLetter(s[i]) {
		i++
	}
	digits := len(s) - i
	return i >= 1 && i <= 3 && digits >= 3 && digits <= 4 && digitsOnly(s[i:])
}

func lettersASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return false
		}
	}
	return true
}

func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDay(s string) bool {
	return (len(s) == 1 || len(s) == 2) && digitsOnly(s)
}

func isYear(s string) bool {
	return (len(s) == 2 || len(s) == 4) && digitsOnly(s)
}

func isMonth(s string) bool {
	for len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	switch len(s) {
	case 3:
		return eq(s, "jan") || eq(s, "feb") || eq(s, "mar") || eq(s, "apr") || eq(s, "may") || eq(s, "jun") || eq(s, "jul") || eq(s, "aug") || eq(s, "sep") || eq(s, "oct") || eq(s, "nov") || eq(s, "dec")
	case 4:
		return eq(s, "june") || eq(s, "july") || eq(s, "sept")
	case 5:
		return eq(s, "march") || eq(s, "april")
	case 6:
		return eq(s, "august")
	case 7:
		return eq(s, "january") || eq(s, "october")
	case 8:
		return eq(s, "february") || eq(s, "november") || eq(s, "december")
	case 9:
		return eq(s, "september")
	default:
		return false
	}
}

func wordLabel(s string) bool {
	if s == "" || !isLetter(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' || c == '-' || c == '/' {
			return false
		}
	}
	return true
}

func eq(s, lower string) bool {
	for len(s) > 0 && (s[len(s)-1] == '.' || s[len(s)-1] == ':') {
		s = s[:len(s)-1]
	}
	if len(s) != len(lower) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != lower[i] {
			return false
		}
	}
	return true
}

func filenameProv(start, end int) Provenance {
	return Provenance{Origin: OriginFilename, Start: start, End: end, Labeled: true, Run: -1}
}

func textProv(src identity.Source, index, start, end int, labeled bool) Provenance {
	return Provenance{
		Origin:   OriginText,
		Page:     src.Page,
		Rotation: src.Rotation,
		Sheet:    src.Sheet,
		Cell:     src.Cell,
		Merge:    src.Merge,
		Table:    src.Table,
		Row:      src.Row,
		Col:      src.Col,
		Start:    start,
		End:      end,
		Labeled:  labeled,
		Heading:  src.Heading,
		Run:      index,
	}
}

func fileStem(name string) (stem string, offset int) {
	base := filepath.Base(name)
	offset = strings.LastIndex(name, base)
	if offset < 0 {
		return "", 0
	}
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".pdf", ".docx", ".xlsx", ".xlsm":
		base = base[:len(base)-len(ext)]
	}
	return base, offset
}

func titleFromGaps(stem string, covered []span) (string, int, int, bool) {
	merged := mergeSpans(covered)
	type piece struct {
		text       string
		start, end int
	}
	var pieces []piece
	cursor := 0
	for _, s := range merged {
		if s.start > cursor {
			pieces = append(pieces, piece{stem[cursor:s.start], cursor, s.start})
		}
		if s.end > cursor {
			cursor = s.end
		}
	}
	if cursor < len(stem) {
		pieces = append(pieces, piece{stem[cursor:], cursor, len(stem)})
	}
	bestText := ""
	bestStart, bestEnd := 0, 0
	for _, p := range pieces {
		trimmed, rel := trimSep(p.text)
		if len(trimmed) < len(bestText) || !looksLikeTitle(trimmed) {
			continue
		}
		bestText = trimmed
		bestStart = p.start + rel
		bestEnd = bestStart + len(trimmed)
	}
	if bestText == "" {
		return "", 0, 0, false
	}
	return bestText, bestStart, bestEnd, true
}

func mergeSpans(covered []span) []span {
	if len(covered) == 0 {
		return nil
	}
	sorted := append([]span(nil), covered...)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1].start > sorted[j].start {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
			j--
		}
	}
	merged := []span{sorted[0]}
	for _, s := range sorted[1:] {
		last := &merged[len(merged)-1]
		if s.start > last.end {
			merged = append(merged, s)
			continue
		}
		if s.end > last.end {
			last.end = s.end
		}
	}
	return merged
}

func trimSep(s string) (string, int) {
	start := 0
	end := len(s)
	for start < end && isSep(s[start]) {
		start++
	}
	for end > start && isSep(s[end-1]) {
		end--
	}
	return s[start:end], start
}

func isSep(b byte) bool {
	switch b {
	case ' ', '\t', '_', '-', ':', '.':
		return true
	default:
		return false
	}
}

func fullMatch(re *regexp.Regexp, s string) bool {
	loc := re.FindStringIndex(s)
	return loc != nil && loc[0] == 0 && loc[1] == len(s)
}

func overlaps(spans []span, start, end int) bool {
	for _, s := range spans {
		if start < s.end && end > s.start {
			return true
		}
	}
	return false
}
