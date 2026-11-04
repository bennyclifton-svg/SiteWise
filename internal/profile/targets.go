package profile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Owner targets and a builder's preliminary estimate (Walsh review,
// 2026-10-27). Code offers verbatim candidates; Jev picks one or none; the
// functions below read the picked words in Go. Nothing is guessed: a
// phrase this code cannot read stays text.

var (
	targetDateRe = regexp.MustCompile(`(?i)\b(?:(?:early|mid|late)[- ]?\d{4}|q[1-4]\s+\d{4}|\d{1,2}\s+(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{4}|(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{4})\b`)
	monthsRe2    = regexp.MustCompile(`(?i)\b(\d{1,2})(?:\s*(?:[–-]|to)\s*(\d{1,2}))?\s*months?\b`)
	moneyRangeRe = regexp.MustCompile(`(?i)\$\s?\d[\d,]*(?:\.\d+)?\s*(?:k|m|million)?\b(?:\s*(?:[–-]|to)\s*\$?\s?\d[\d,]*(?:\.\d+)?\s*(?:k|m|million)?\b)?`)
)

func targetDateCandidates(text string) []Candidate {
	var spans []span
	for _, m := range targetDateRe.FindAllStringIndex(text, -1) {
		spans = append(spans, span{m[0], m[1], text[m[0]:m[1]]})
	}
	return collect(text, spans)
}

func monthCountCandidates(text string) []Candidate {
	var spans []span
	for _, m := range monthsRe2.FindAllStringIndex(text, -1) {
		spans = append(spans, span{m[0], m[1], text[m[0]:m[1]]})
	}
	return collect(text, spans)
}

func moneyRangeCandidates(text string) []Candidate {
	var spans []span
	for _, m := range moneyRangeRe.FindAllStringIndex(text, -1) {
		v := strings.TrimSpace(text[m[0]:m[1]])
		spans = append(spans, span{m[0], m[0] + len(v), v})
	}
	return collect(text, spans)
}

// LatestDay reads a target phrase as the last day it allows: "July 2026"
// is 2026-07-31, "mid-2026" 2026-08-31, "early 2027" 2027-04-30, "late
// 2026" 2026-12-31, "Q3 2026" 2026-09-30, "3 March 2026" that day. ok is
// false for a phrase without one of these forms.
func LatestDay(phrase string) (string, bool) {
	m := targetDateRe.FindString(phrase)
	if m == "" {
		return "", false
	}
	s := strings.ToLower(strings.Join(strings.Fields(m), " "))
	year := func(t string) (int, bool) {
		y, err := strconv.Atoi(t[len(t)-4:])
		return y, err == nil
	}
	end := func(y, month int) string {
		return time.Date(y, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	switch {
	case strings.HasPrefix(s, "early"):
		y, ok := year(s)
		return end(y, 4), ok
	case strings.HasPrefix(s, "mid"):
		y, ok := year(s)
		return end(y, 8), ok
	case strings.HasPrefix(s, "late"):
		y, ok := year(s)
		return end(y, 12), ok
	case strings.HasPrefix(s, "q"):
		y, ok := year(s)
		return end(y, int(s[1]-'0')*3), ok
	}
	// time.Parse matches month names without regard to case.
	if t, err := time.Parse("2 January 2006", s); err == nil {
		return t.Format("2006-01-02"), true
	}
	if t, err := time.Parse("January 2006", s); err == nil {
		return end(t.Year(), int(t.Month())), true
	}
	return "", false
}

// MonthRange reads "18 months" as 18 to 18 and "14–16 months" as 14 to 16.
func MonthRange(phrase string) (low, high int, ok bool) {
	m := monthsRe2.FindStringSubmatch(phrase)
	if m == nil {
		return 0, 0, false
	}
	low, _ = strconv.Atoi(m[1])
	high = low
	if m[2] != "" {
		high, _ = strconv.Atoi(m[2])
	}
	if low <= 0 || high < low {
		return 0, 0, false
	}
	return low, high, true
}

// MonthsText writes a month range back as words for a note.
func MonthsText(low, high int) string {
	if low == high {
		return fmt.Sprintf("%d months", low)
	}
	return fmt.Sprintf("%d–%d months", low, high)
}
