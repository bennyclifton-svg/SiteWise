package proposals

import (
	"regexp"
	"strings"
)

var markdownLabel = regexp.MustCompile(`^\*\*(Scope|Exclusions?|Assumptions?|Fee|Programme|Services|Limitations?|Qualifications?|Appointment|Conditions)(?::)?\*\*`)
var markdownSeparator = regexp.MustCompile(`^\|?[\s:|\-]+\|?$`)
var feeHeading = regexp.MustCompile(`(?i)^(?:fixed (?:professional )?fee|fee|fee basis|professional fee)\s*:?$`)
var repeatedQuote = regexp.MustCompile(`(?i)\b(?:quote|quotation|fee)\s+(?:attached|above|below|previously quoted)\b`)

// Preserve Markdown's explicit structure instead of treating pipes and bold
// markers as part of a company name or one long table cell. No rendering or
// external-resource access is needed; the original document remains untouched.
func markdownLines(lines []Line) []Line {
	out := make([]Line, 0, len(lines))
	for _, l := range lines {
		raw := l.Text
		if markdownSeparator.MatchString(raw) {
			continue
		}
		l.Heading = l.Heading || markdownLabel.MatchString(raw)
		if m := markdownLabel.FindStringSubmatch(raw); m != nil {
			l.HeadingText = m[1]
		}
		if l.Heading && strings.HasPrefix(raw, "#") {
			raw = strings.TrimSpace(strings.TrimLeft(raw, "#"))
		}
		l.Text = strings.ReplaceAll(raw, "**", "")
		l.Cells = []string{l.Text}
		if strings.HasPrefix(l.Text, "|") && strings.HasSuffix(l.Text, "|") {
			l.Cells = strings.Split(strings.Trim(l.Text, "|"), "|")
			for i := range l.Cells {
				l.Cells[i] = strings.TrimSpace(l.Cells[i])
			}
		}
		out = append(out, l)
	}
	return out
}

// Only explicit back-references collapse. Equal prices for separate tasks,
// or a revised/alternative offer at the same price, remain separate fees.
func markRepeatedReferences(amounts []Amount) {
	for i := range amounts {
		a := &amounts[i]
		if !repeatedQuote.MatchString(a.Context) {
			continue
		}
		match := ""
		for j := 0; j < i; j++ {
			b := amounts[j]
			if b.Cents == a.Cents && b.GST == a.GST && b.RepeatOf == "" && !b.IncTwin && !b.GSTPart && !b.PerHour {
				if match != "" {
					match = ""
					break
				}
				match = b.ID
			}
		}
		a.RepeatOf = match
	}
}
