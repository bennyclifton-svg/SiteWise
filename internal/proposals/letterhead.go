package proposals

import (
	"regexp"
	"strings"
	"time"
)

// Letterhead is the submitting firm's name and office address as printed
// beside its ABN at the head of the proposal.
type Letterhead struct {
	Name    string
	Address string
}

var (
	emphasis = regexp.MustCompile(`\*\*|__`)
	// addresseeRe marks a line that names someone other than the firm: the
	// client, the project or the site.
	addresseeRe = regexp.MustCompile(`(?i)^\s*(?:to|attn\.?|attention|client|principal|owner|project|site|prepared\s+for|re|subject)\s*[:–]`)
	bareLabelRe = regexp.MustCompile(`(?i)^\s*(?:to|client|principal|owner|attention|prepared\s+for)\s*$`)
	dateLabelRe = regexp.MustCompile(`(?i)^\s*(?:date|dated|proposal\s+date)\s*:\s*(.+)$`)
)

// stripEmphasis removes Markdown bold markers so "**Quoin Pty Ltd**" reads
// as the name it prints.
func stripEmphasis(s string) string {
	return strings.TrimSpace(emphasis.ReplaceAllString(s, ""))
}

func addressee(s string) bool { return addresseeRe.MatchString(s) || bareLabelRe.MatchString(s) }

// letterhead reads the firm from the lines around the proposal's only ABN:
// the company name printed just above it (or before it on its line) and the
// one street address beside it. Any addressee label near the ABN means the
// block may be the client's, so nothing is read. A rule, not a judgement:
// it settles only the plain letter layout and leaves the rest to Jev.
func letterhead(lines []Line, abns []string) *Letterhead {
	if len(abns) != 1 {
		return nil
	}
	at := -1
	for i, l := range lines {
		if strings.Contains(digitsOf(l.Text), abns[0]) {
			at = i
			break
		}
	}
	if at < 0 || addressee(lines[at].Text) {
		return nil
	}
	for j := max(0, at-3); j < at; j++ {
		if addressee(lines[j].Text) {
			return nil
		}
	}
	name := ""
	for j := at; j >= max(0, at-3) && name == ""; j-- {
		t := lines[j].Text
		if j == at {
			cut := strings.Index(strings.ToUpper(t), "ABN")
			if cut <= 0 {
				continue
			}
			t = t[:cut]
		}
		t = strings.Trim(t, " ,.-|")
		if companyRe.MatchString(t) && len(strings.Fields(t)) <= 8 && !moneyRe.MatchString(t) && !emailRe.MatchString(t) && !addressRe.MatchString(t) {
			name = cleanFirmCandidate(t)
		}
	}
	if name == "" {
		return nil
	}
	var addresses []string
	for j := max(0, at-3); j <= min(len(lines)-1, at+3); j++ {
		t := lines[j].Text
		if j > at && (addressee(t) || isHeading(t) || strings.HasPrefix(t, "#")) {
			break
		}
		for _, m := range addressRe.FindAllString(t, -1) {
			m = strings.Trim(m, " ,.")
			if !containsFold(addresses, m) {
				addresses = append(addresses, m)
			}
		}
	}
	out := &Letterhead{Name: name}
	if len(addresses) == 1 {
		out.Address = addresses[0]
	}
	return out
}

// headerDate is the proposal's own date: a "Date:" line in the header,
// before the first section. Other dates (validity, programme) are not it.
func headerDate(lines []Line) string {
	for i, l := range lines {
		if i >= 25 || strings.HasPrefix(l.Text, "## ") {
			break
		}
		m := dateLabelRe.FindStringSubmatch(l.Text)
		if m == nil {
			continue
		}
		if d, ok := parseDay(strings.TrimSpace(m[1])); ok {
			return d
		}
	}
	return ""
}

func parseDay(s string) (string, bool) {
	for _, layout := range []string{"2 January 2006", "2 Jan 2006", "January 2, 2006", "2/01/2006", "02/01/2006", "2006-01-02", "2.01.2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), true
		}
	}
	return "", false
}

func digitsOf(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
