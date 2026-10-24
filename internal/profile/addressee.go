package profile

import (
	"regexp"
	"strings"
)

// addresseeRe is a letter's "To:" line, with or without Markdown bold.
var addresseeRe = regexp.MustCompile(`(?im)^[*_ \t]*to[*_ \t]*:[*_ \t]*([^\n]{3,160})$`)

// trusteeRe ends the entity's own name in "X Pty Ltd as trustee for the Y
// Trust" or "X Pty Ltd ATF Y Trust".
var trusteeRe = regexp.MustCompile(`(?i)\s+(?:as\s+trustee\s+for|atf)\s+`)

// addresseeCandidates offers the party a proposal or letter is addressed to
// as a client candidate: the whole line, and the entity alone when it acts
// as trustee. Jev picks the project's client or none; a consultant or
// contractor addressee is "none" by the client question's own wording.
// An email address is not a client name.
func addresseeCandidates(text string) []Candidate {
	var spans []span
	for _, m := range addresseeRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[3]
		line := strings.TrimRight(text[start:end], " \t*_.,;")
		if line == "" || strings.Contains(line, "@") {
			continue
		}
		end = start + len(line)
		spans = append(spans, span{start, end, line})
		if at := trusteeRe.FindStringIndex(line); at != nil {
			spans = append(spans, span{start, start + at[0], line[:at[0]]})
		}
	}
	return collect(text, spans)
}
