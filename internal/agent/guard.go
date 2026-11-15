package agent

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/figures"
)

// Answer is the model's final text after Go has checked it
// (docs/design/2026-11-13-pencil.md 5.3; plan review change 3 and Task 8).
type Answer struct {
	Text     string `json:"text"`
	Marks    []Mark `json:"marks,omitempty"`
	Question bool   `json:"question,omitempty"`
	// Coverage is Go's line saying what a document search covered. It is
	// shown under every document answer.
	Coverage string `json:"coverage,omitempty"`
	// Unchecked counts figures, dates, clauses and quotations that are the
	// model's own: in no tag, in nothing read this turn, typed by nobody.
	Unchecked int `json:"unchecked"`
	// Verdicts counts conclusions on risk level or compliance, which chat
	// does not draw.
	Verdicts int      `json:"verdicts,omitempty"`
	Cited    []string `json:"cited,omitempty"` // titles of documents whose quotations checked out
}

// Mark is one span of the answer's text and what Go knows about it.
//
//	fact     a figure Go returned, printed by Go from its tag
//	quote    a quotation that is a literal part of a stored passage; links to its page
//	source   a passage named without a quotation
//	seen     a figure the PM typed or a tool returned this turn, not tied to a row
//	check    the model's own figure, date, clause or quotation: "check it"
//	verdict  a conclusion on risk or compliance
type Mark struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Kind  string `json:"kind"`
	Note  string `json:"note,omitempty"`
	DocID string `json:"document_id,omitempty"`
	Page  int    `json:"page,omitempty"`
}

const monthNames = `(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?`

var (
	tagged = regexp.MustCompile(`"([^"\n]{2,}?)"\s*\[(p\d+)\]|“([^”\n]{2,}?)”\s*\[(p\d+)\]|\{(f\d+)\}|\[(p\d+)\]`)
	dates  = regexp.MustCompile(`(?i)\b\d{1,2}(?:st|nd|rd|th)?\s+(?:of\s+)?` + monthNames + `(?:,?\s+\d{4})?|\b` + monthNames + `\s+\d{1,2}(?:st|nd|rd|th)?(?:,?\s+\d{4})?\b|\b\d{4}-\d{2}-\d{2}\b|\b\d{1,2}/\d{1,2}/\d{2,4}\b`)
	codes  = regexp.MustCompile(`\b[A-Z]{1,4}[ -]?\d+(?:[.\-/]\d+)*[A-Za-z]?\b`)
	nums   = regexp.MustCompile(`(?i)-?\$?\d(?:[\d,]*\d)?(?:\.\d+)*(?:\s?%|\s(?:per cent|percent|million|thousand|billion)\b|(?:k|m|bn)\b)?`)
	counts = regexp.MustCompile(`(?i)\b(?:two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety|hundred|thousand|million|billion)\b`)
	spaces = regexp.MustCompile(`\s+`)
	isMark = regexp.MustCompile(`(?m)^\s*$`)
)

// verdicts are conclusions chat does not draw (review change 1; cases 2,
// 4 and 21). They are marked, never silently removed.
var verdicts = []string{"low risk", "low-risk", "no risk", "not risky", "is compliant", "it complies", "complies with", "does comply",
	"does not comply", "non-compliant", "no asbestos", "asbestos-free", "there are no trees", "is safe"}

func squash(s string) string { return strings.TrimSpace(spaces.ReplaceAllString(s, " ")) }

// Check rewrites the model's final text: figure tags become Go's values,
// quotations are tested against the passages read this turn, and every
// other figure, date and clause is marked.
func Check(text string, st *State) Answer {
	a := Answer{Coverage: st.Coverage}
	var b strings.Builder
	cited := map[string]bool{}
	mark := func(s, kind, note string, p Passage) {
		a.Marks = append(a.Marks, Mark{Start: b.Len(), End: b.Len() + len(s), Kind: kind, Note: note, DocID: p.DocID, Page: p.Page})
		b.WriteString(s)
	}
	where := func(p Passage) string {
		note := p.Document
		if p.Page > 0 {
			note += ", page " + strconv.Itoa(p.Page)
		}
		if p.Note != "" {
			note += "; " + p.Note
		}
		return note
	}
	last := 0
	for _, m := range tagged.FindAllStringSubmatchIndex(text, -1) {
		b.WriteString(text[last:m[0]])
		last = m[1]
		group := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return text[m[2*i]:m[2*i+1]]
		}
		quote, pid := group(1), group(2)
		if quote == "" {
			quote, pid = group(3), group(4)
		}
		switch {
		case quote != "":
			p, ok := findQuote(st, pid, quote)
			shown := `"` + quote + `"`
			if ok {
				mark(shown, "quote", where(p), p)
				if !cited[p.Document] {
					cited[p.Document] = true
					a.Cited = append(a.Cited, p.Document)
				}
			} else {
				mark(shown, "check", "These words are not in what was read. Check them.", Passage{})
				a.Unchecked++
			}
		case group(5) != "":
			if f, ok := st.Facts[group(5)]; ok {
				mark(f.Value, "fact", f.Label+" ("+f.Source+")", Passage{})
			} else {
				mark(text[m[0]:m[1]], "check", "This figure is not tied to a record. Check it.", Passage{})
				a.Unchecked++
			}
		default:
			if p, ok := st.Passages[group(6)]; ok {
				mark("("+where(p)+")", "source", "", p)
			}
		}
	}
	b.WriteString(text[last:])
	a.Text = b.String()
	// Scan what is left for figures of the model's own.
	taken := append([]Mark{}, a.Marks...)
	free := func(s, e int) bool {
		for _, m := range taken {
			if s < m.End && e > m.Start {
				return false
			}
		}
		return true
	}
	scan := func(re *regexp.Regexp, date bool) {
		for _, loc := range re.FindAllStringIndex(a.Text, -1) {
			s, e := loc[0], loc[1]
			tok := a.Text[s:e]
			if !free(s, e) || listNumber(a.Text, s, e) {
				continue
			}
			kind := "seen"
			known := st.Seen.HasNumbersOf(tok)
			if date {
				known = known && len(figures.Numbers(tok)) > 0
			}
			if !known {
				kind = "check"
				a.Unchecked++
			}
			m := Mark{Start: s, End: e, Kind: kind}
			if kind == "check" {
				m.Note = "Typed by the assistant. Check it."
			}
			taken = append(taken, m)
			a.Marks = append(a.Marks, m)
		}
	}
	scan(dates, true)
	scan(codes, false)
	scan(nums, false)
	scan(counts, false)
	lower := strings.ToLower(a.Text)
	for _, v := range verdicts {
		for from := 0; ; {
			i := strings.Index(lower[from:], v)
			if i < 0 {
				break
			}
			s := from + i
			from = s + len(v)
			if Declined(lower, s) {
				continue
			}
			a.Marks = append(a.Marks, Mark{Start: s, End: s + len(v), Kind: "verdict", Note: "SiteWise does not judge risk level or compliance. This is the assistant's reading."})
			a.Verdicts++
		}
	}
	sort.SliceStable(a.Marks, func(i, j int) bool { return a.Marks[i].Start < a.Marks[j].Start })
	return a
}

var declining = []string{"can't", "cannot", "can not", "don't", "doesn't", "does not", "do not", "not ", "never", "isn't", "won't", "whether", "no statement", "rather than"}

// Declined reports whether the words just before position at decline to
// say what follows ("I can't call this project low risk"). lower is the
// text in lower case.
func Declined(lower string, at int) bool {
	from := max(at-48, 0)
	before := lower[from:at]
	if i := strings.LastIndexAny(before, ".!?\n"); i >= 0 {
		before = before[i+1:]
	}
	for _, d := range declining {
		if strings.Contains(before, d) {
			return true
		}
	}
	return false
}

// findQuote returns the passage a quotation is a literal part of: the one
// the model named, or, when the words are not there, another passage read
// this turn that does hold them. The link then goes to where the words
// really are. A quotation in no passage is not found.
func findQuote(st *State, pid, quote string) (Passage, bool) {
	want := squash(quote)
	if p, ok := st.Passages[pid]; ok && strings.Contains(squash(p.Text), want) {
		return p, true
	}
	ids := make([]string, 0, len(st.Passages))
	for id := range st.Passages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if p := st.Passages[id]; strings.Contains(squash(p.Text), want) {
			return p, true
		}
	}
	return Passage{}, false
}

// listNumber reports whether a number is a list marker ("1. ", "2) ") at
// the start of a line.
func listNumber(text string, s, e int) bool {
	if e >= len(text) || (text[e] != '.' && text[e] != ')') {
		return false
	}
	if e+1 < len(text) && text[e+1] != ' ' {
		return false
	}
	line := text[:s]
	if i := strings.LastIndexByte(line, '\n'); i >= 0 {
		line = line[i+1:]
	}
	return isMark.MatchString(line)
}
