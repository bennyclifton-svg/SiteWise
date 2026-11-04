package proposals

import (
	"regexp"
	"strconv"
	"strings"
)

// Insurance is the professional indemnity cover a letter states: only the
// two figures the app acts on (Walsh review, 2026-10-27). The insurer and
// policy number stay in the text.
type Insurance struct {
	// LimitCents is the cover limit; nil when no limit is printed.
	LimitCents *int64 `json:"limit_cents,omitempty"`
	// ExpiresOn is the last day of the stated period, YYYY-MM-DD, or "".
	ExpiresOn string `json:"expires_on,omitempty"`
}

var (
	abnLabel = regexp.MustCompile(`(?i)\bABN\b`)
	// signatureTitle is a professional title after "Director," not a firm.
	signatureTitle = regexp.MustCompile(`(?i)\b(?:registered|licensed|chartered|accredited|nominated)\b`)
	monthlyFee     = regexp.MustCompile(`(?i)\bmonthly\b|\bper\s+month\b|\bpcm\b|[×x]\s*(?:est\.?\s*)?\d+\s*months?\b`)
	piHeading      = regexp.MustCompile(`(?i)\bprofessional\s+indemnity\b`)
	piLimit        = regexp.MustCompile(`(?i)\b(?:limit|cover(?:ed)?|indemnity)\b[^$]{0,40}\$\s?([0-9][0-9,]*(?:\.[0-9]+)?)\s*(million|m)?\b`)
	piPeriod       = regexp.MustCompile(`(?i)\b(?:to|until|expir(?:es|y|ing)(?:\s+on)?|ending)\s+(\d{1,2}\s+[A-Za-z]+\s+\d{4}|\d{1,2}/\d{1,2}/\d{4})`)
)

// readInsurance reads the professional indemnity paragraph: the line that
// names it and the lines that follow until the next heading. Code reads the
// figures; nothing is asked of Jev.
func readInsurance(lines []Line) *Insurance {
	for i, l := range lines {
		if !piHeading.MatchString(l.Text) {
			continue
		}
		var b strings.Builder
		for j := i; j < min(len(lines), i+6); j++ {
			if j > i && (lines[j].Heading || strings.HasPrefix(lines[j].Text, "#")) {
				break
			}
			b.WriteString(stripEmphasis(lines[j].Text) + " ")
		}
		text := b.String()
		out := &Insurance{}
		if m := piLimit.FindStringSubmatch(text); m != nil {
			if n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil && n > 0 {
				if m[2] != "" {
					n *= 1_000_000
				}
				c := int64(n*100 + 0.5)
				out.LimitCents = &c
			}
		}
		if m := piPeriod.FindStringSubmatch(text); m != nil {
			if d, ok := parseDay(m[1]); ok {
				out.ExpiresOn = d
			}
		}
		if out.LimitCents != nil || out.ExpiresOn != "" {
			return out
		}
	}
	return nil
}
