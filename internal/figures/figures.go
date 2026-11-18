// Package figures reads numbers, money, durations and dates the way a
// project manager types them, and keeps the book of figures a chat turn may
// use. It is pure Go and calls no model: an amount, a date or a duration in
// a chat operation must be in the book (the PM typed it, or SiteWise
// returned it this turn) or the operation is refused
// (docs/plans/2026-11-15-chat-agent.md, review change 2).
package figures

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	number  = regexp.MustCompile(`(?i)-?\$?\d[\d,]*(?:\.\d+)?(?:(?:k|m|bn)\b|\s(?:thousand|million|billion)\b)?`)
	amount  = regexp.MustCompile(`(?i)^(-?)\$?\s?(\d[\d,]*(?:\.\d+)?)\s?(k|m|bn|thousand|million|billion)?$`)
	isoDate = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	dmy     = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})/(\d{4}|\d{2})\b`)
	dMonth  = regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?(?:,?\s+(\d{4}))?`)
	monthD  = regexp.MustCompile(`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?`)
	word    = regexp.MustCompile(`[A-Za-z]+`)
	span    = regexp.MustCompile(`(?i)^\s*(a|an|\d+|[a-z]+)?\s*(d|w|m|days?|weeks?|months?|fortnights?|working days?)?\s*$`)
)

var scale = map[string]int64{"k": 1e3, "thousand": 1e3, "m": 1e6, "million": 1e6, "bn": 1e9, "billion": 1e9}

var words = map[string]int{"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8,
	"nine": 9, "ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16,
	"seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20, "thirty": 30, "forty": 40, "fifty": 50, "sixty": 60,
	"seventy": 70, "eighty": 80, "ninety": 90, "hundred": 100}

var months = map[string]int{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}

// Plain writes a decimal with no separators and no trailing zeros, so
// "45,000.00" and "45000" compare equal.
func Plain(dec string) (string, bool) {
	r, ok := new(big.Rat).SetString(strings.ReplaceAll(dec, ",", ""))
	if !ok {
		return "", false
	}
	s := r.FloatString(6)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" {
		s = "0"
	}
	return s, true
}

// Dollars writes a decimal as an amount for a sentence: "$45,000" or
// "$45,000.5" (trailing zeros dropped). Text that is not a decimal is
// returned as it is.
func Dollars(dec string) string {
	p, ok := Plain(dec)
	if !ok {
		return dec
	}
	whole, frac, _ := strings.Cut(strings.TrimPrefix(p, "-"), ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := "$" + b.String()
	if frac != "" {
		out += "." + frac
	}
	if strings.HasPrefix(p, "-") {
		out = "-" + out
	}
	return out
}

// Amount reads one amount as a PM writes it: "$45,000", "45k", "1.2m".
// It refuses anything that is not exactly one amount.
func Amount(s string) (string, bool) {
	m := amount.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}
	r, ok := new(big.Rat).SetString(strings.ReplaceAll(m[2], ",", ""))
	if !ok {
		return "", false
	}
	if by, ok := scale[strings.ToLower(m[3])]; ok {
		r.Mul(r, new(big.Rat).SetInt64(by))
	}
	return Plain(m[1] + r.FloatString(6))
}

// Numbers returns every number in the text in plain form. A suffix such as
// "45k" yields the scaled value and the bare one, since the book only ever
// answers "was this number given".
func Numbers(text string) []string {
	var out []string
	add := func(s string) {
		if p, ok := Plain(s); ok {
			out = append(out, p)
		}
	}
	for _, tok := range number.FindAllString(text, -1) {
		if a, ok := Amount(tok); ok {
			out = append(out, strings.TrimPrefix(a, "-"))
		}
		bare := strings.TrimLeft(tok, "-$")
		if i := strings.IndexFunc(bare, func(r rune) bool { return !(r >= '0' && r <= '9') && r != ',' && r != '.' }); i > 0 {
			bare = bare[:i]
		}
		add(strings.TrimRight(bare, ".,"))
	}
	for _, w := range word.FindAllString(text, -1) {
		if n, ok := words[strings.ToLower(w)]; ok {
			out = append(out, strconv.Itoa(n))
		}
	}
	return out
}

// Duration reads "2 weeks", "ten days", "a fortnight", "3 months", "8w".
// A bare number is days.
func Duration(s string) (n int, unit string, ok bool) {
	m := span.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil || (m[1] == "" && m[2] == "") {
		return 0, "", false
	}
	n = 1
	switch {
	case m[1] == "" || m[1] == "a" || m[1] == "an":
	case m[1][0] >= '0' && m[1][0] <= '9':
		n, _ = strconv.Atoi(m[1])
	default:
		if n, ok = words[m[1]]; !ok {
			return 0, "", false
		}
	}
	switch u := m[2]; {
	case u == "" || u == "d" || strings.Contains(u, "day"):
		if m[2] == "" && (m[1] == "a" || m[1] == "an") {
			return 0, "", false
		}
		unit = "day"
	case u == "w" || strings.HasPrefix(u, "week"):
		unit = "week"
	case strings.HasPrefix(u, "fortnight"):
		n, unit = n*2, "week"
	default:
		unit = "month"
	}
	return n, unit, true
}

// WorkingDays converts a typed duration to working days on a working week
// of five, six or seven days. A month is 52 weeks over 12, rounded.
func WorkingDays(s string, week int) (int, bool) {
	n, unit, ok := Duration(s)
	if !ok || week < 1 || week > 7 {
		return 0, false
	}
	switch unit {
	case "week":
		return n * week, true
	case "month":
		return (n*week*52 + 6) / 12, true
	}
	return n, true
}

// Date is a date as mentioned; Year is 0 when none was given.
type Date struct{ Year, Month, Day int }

// Dates returns the dates a text mentions, read day first (Australian use).
func Dates(text string) []Date {
	var out []Date
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	for _, m := range dmy.FindAllStringSubmatch(text, -1) {
		y := atoi(m[3])
		if y < 100 {
			y += 2000
		}
		out = append(out, Date{y, atoi(m[2]), atoi(m[1])})
	}
	for _, m := range dMonth.FindAllStringSubmatch(text, -1) {
		out = append(out, Date{atoi(m[3]), months[strings.ToLower(m[2])], atoi(m[1])})
	}
	for _, m := range monthD.FindAllStringSubmatch(text, -1) {
		out = append(out, Date{atoi(m[3]), months[strings.ToLower(m[1])], atoi(m[2])})
	}
	return out
}

// Book is the set of figures a chat turn may put in an operation: those the
// PM typed and those SiteWise's own tools returned this turn.
type Book struct {
	nums  map[string]bool
	iso   map[string]bool
	dates []Date
}

// Add puts every number and date of a text in the book.
func (b *Book) Add(text string) {
	if b.nums == nil {
		b.nums, b.iso = map[string]bool{}, map[string]bool{}
	}
	for _, n := range Numbers(text) {
		b.nums[n] = true
	}
	for _, m := range isoDate.FindAllString(text, -1) {
		b.iso[m] = true
	}
	b.dates = append(b.dates, Dates(text)...)
}

// HasNumber reports whether the decimal was given.
func (b *Book) HasNumber(dec string) bool {
	p, ok := Plain(dec)
	return ok && b.nums[strings.TrimPrefix(p, "-")]
}

// HasNumbersOf reports whether every number inside a typed value was given
// ("R3", "612 m2"). A value with no number passes.
func (b *Book) HasNumbersOf(value string) bool {
	for _, n := range Numbers(value) {
		if !b.nums[n] {
			return false
		}
	}
	return true
}

// HasDate reports whether a YYYY-MM-DD date was given: literally by a tool,
// or by the PM as a day and month. With no year typed, only the next such
// day from today counts, so the model cannot choose the year.
func (b *Book) HasDate(iso string, today time.Time) bool {
	if b.iso[iso] {
		return true
	}
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return false
	}
	for _, d := range b.dates {
		if d.Day != t.Day() || d.Month != int(t.Month()) {
			continue
		}
		if d.Year == t.Year() {
			return true
		}
		if d.Year == 0 {
			next := time.Date(today.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			if next.Before(today.Truncate(24 * time.Hour)) {
				next = next.AddDate(1, 0, 0)
			}
			if next.Year() == t.Year() {
				return true
			}
		}
	}
	return false
}
