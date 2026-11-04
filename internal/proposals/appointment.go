package proposals

import (
	"path/filepath"
	"regexp"
	"strings"
)

// An appointment that states its fee prices a consultant as surely as a fee
// proposal does: the owner's email "we have appointed X Pty Ltd ... Fee:
// $6,800 + GST" is the accepted offer. Both cues must be present, so an
// appointment with no fee, or a fee mentioned in passing, is not read.
// "Appointed" alone is common in proposals' terms ("a mediator appointed
// by"), credentials and recommendations, so the document must call itself
// an appointment in its name or opening lines, or name the company it
// appointed.
var (
	appointRe   = regexp.MustCompile(`(?i)\b(?:letter of appointment|appointment|appointed|letter of engagement|engagement letter)\b`)
	feeLineRe   = regexp.MustCompile(`(?i)\bfees?\b[^$\n]{0,60}\$\s?\d`)
	appointeeRe = regexp.MustCompile(`(?i:\bappointed|\bappointment of)[\s:*_]*(?i:the\s+)?([A-Z][A-Za-z0-9&'.\- ]{0,50}?\s(?i:pty\.?\s*ltd|pty\.?\s*limited|limited|ltd)\b\.?)`)
)

// subjectRe is an email's subject line, where it names what it is about.
var subjectRe = regexp.MustCompile(`(?i)^[#*\s]*(?:subject|re)\s*:`)

// appointmentFee reports an appointment that states a fee.
func appointmentFee(filename string, lines []Line) bool {
	fee := false
	for _, l := range lines {
		fee = fee || feeLineRe.MatchString(l.Text)
	}
	if !fee {
		return false
	}
	name := strings.NewReplacer("-", " ", "_", " ").Replace(filepath.Base(filename))
	if appointRe.MatchString(name) || len(appointees(lines)) > 0 {
		return true
	}
	return appointRe.MatchString(title(lines))
}

// title is what a document calls itself: an email's subject, else its
// first line (a letter's or a Markdown file's heading).
func title(lines []Line) string {
	first := ""
	for _, l := range lines[:min(len(lines), 12)] {
		t := strings.TrimSpace(l.Text)
		if subjectRe.MatchString(t) {
			return t
		}
		if first == "" {
			first = t
		}
	}
	return first
}

// appointees are the companies a document says were appointed, once each.
func appointees(lines []Line) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lines {
		for _, m := range appointeeRe.FindAllStringSubmatch(l.Text, -1) {
			n := cleanFirmCandidate(strings.TrimSpace(m[1]))
			if k := NameKey(n); k != "" && !seen[k] {
				seen[k] = true
				out = append(out, n)
			}
		}
	}
	return out
}
