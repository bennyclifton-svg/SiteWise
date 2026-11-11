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

// careOfRe starts the party a letter is sent through, which is not the
// addressee's name: "The Owners - Strata Plan No. 912345, c/- Northgate Strata
// Management".
var careOfRe = regexp.MustCompile(`(?i)[,;]?\s+(?:c/-|c/o|care\s+of)\s`)

// careOfLineRe is an address block's first line with no "To:" label: the
// party a quote is addressed to, care of their designer ("Priya Nair &
// Tom Whitlock C/O Harbour Lane Design").
var careOfLineRe = regexp.MustCompile(`(?im)^[*_ \t]*([A-Z][^\n:@]{2,80}?)[,;]?[ \t]+(?:c/-|c/o|care[ \t]+of)[ \t]+\S`)

// actingForRe starts the party an agent is addressed for: "Ines Hart,
// Marlowe Hart Architects, for Hamish and Rosa Delaney". The comma keeps
// "X Pty Ltd as trustee for" out.
var actingForRe = regexp.MustCompile(`(?i)[,;]\s+(?:for|on\s+behalf\s+of)\s+`)

// companyRe is a party that is a company or body rather than a person or a
// role: the last part of "Megan Holt, Asset Manager, Corvina Property
// Holdings Pty Ltd". "The Owners" opens a strata corporation's name.
var companyRe = regexp.MustCompile(`(?i)(?:\b(?:pty\.?\s+ltd|ltd|limited|group|holdings|council|corporation|trust)\.?$|^the\s+owners\b|\s+(?:as\s+trustee\s+for|atf)\s+)`)

// companyPart is the offset of the last comma-separated part of line when
// that part names a company, else -1. The person and role before it are a
// contact at the client, not the client (F46); a line whose last part is a
// person ("X Pty Ltd, attention Y") stays whole.
func companyPart(line string) int {
	at := strings.LastIndex(line, ",")
	if at < 0 {
		return -1
	}
	rest := line[at+1:]
	lead := len(rest) - len(strings.TrimLeft(rest, " 	"))
	if !companyRe.MatchString(rest[lead:]) {
		return -1
	}
	return at + 1 + lead
}

// addresseeCandidates offers the party a proposal or letter is addressed to
// as a client candidate: the whole line (or the company alone after a
// contact's name and role), and the entity alone when it acts as trustee.
// Jev picks the project's client or none; a consultant or contractor
// addressee is "none" by the client question's own wording.
// An email address is not a client name.
func addresseeCandidates(text string) []Candidate {
	var spans []span
	for _, m := range addresseeRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[3]
		line := text[start:end]
		if at := actingForRe.FindStringIndex(line); at != nil {
			start += at[1]
			line = line[at[1]:]
		}
		if at := careOfRe.FindStringIndex(line); at != nil {
			line = line[:at[0]]
		}
		line = strings.TrimRight(line, " \t*_.,;")
		if line == "" || strings.Contains(line, "@") {
			continue
		}
		if at := companyPart(line); at >= 0 {
			start += at
			line = line[at:]
		}
		end = start + len(line)
		spans = append(spans, span{start, end, line})
		if at := trusteeRe.FindStringIndex(line); at != nil {
			spans = append(spans, span{start, start + at[0], line[:at[0]]})
		}
	}
	for _, m := range careOfLineRe.FindAllStringSubmatchIndex(text, -1) {
		if name := strings.TrimRight(text[m[2]:m[3]], " \t*_.,;"); name != "" && !addresseeRe.MatchString(text[m[0]:m[1]]) {
			spans = append(spans, span{m[2], m[2] + len(name), name})
		}
	}
	return collect(text, spans)
}
