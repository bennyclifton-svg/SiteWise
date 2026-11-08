package profile

import (
	"regexp"
	"strings"
)

// Unit mix: one count per apartment type (docs/plans/2026-11-08-unit-mix.md).
// Code offers only a number written against a type phrase, before it
// ("14 one-bedroom", "6 x 2B", "six two-bedroom") or after it ("2 Bed: 4",
// "two-bedroom apartments (38)"). Jev picks one or none
// (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).

func bedType(digits, words string) *regexp.Regexp {
	return regexp.MustCompile(`(?:\b(?i:(?:` + digits + `|` + words + `)[ -]?(?:bed(?:room)?s?|br))\b|\b(?:` + digits + `)B\b)`)
}

var unitMixTypes = map[string]*regexp.Regexp{
	"units_studio": regexp.MustCompile(`\b(?i:studios?)\b`),
	"units_1_bed":  bedType(`1`, `one`),
	"units_2_bed":  bedType(`2`, `two`),
	"units_3_bed":  bedType(`3`, `three`),
	"units_4_bed":  bedType(`[45]\+?`, `four|five`),
}

var countWords = map[string]string{"one": "1", "two": "2", "three": "3", "four": "4", "five": "5", "six": "6", "seven": "7",
	"eight": "8", "nine": "9", "ten": "10", "eleven": "11", "twelve": "12", "fifteen": "15", "twenty": "20"}

var (
	// A count just before the type, on the same line: "14 ", "6 x ", "six ".
	countBeforeRe = regexp.MustCompile(`(?i)(?:^|[^\w.,%])(\d{1,4}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|fifteen|twenty)[ \t]*(?:x|×|no\.?)?[ \t]+$`)
	// A count just after it: "2 Bed: 4", "2 Bed | 20", "two-bedroom apartments (38)".
	countAfterRe = regexp.MustCompile(`(?i)^(?:[ \t]+(?:apartments?|units?|dwellings?|apts?|homes?))?[ \t]*(?:[:=|–—-]|\()?[ \t]*(\d{1,4})\b`)
	// A number that measures something else: an area, a share, a decimal.
	measureAfterRe = regexp.MustCompile(`(?i)^(?:\.\d|[ \t]*(?:%|m2\b|m²|sqm\b|sq\.?\s?m\b|m\b|mm\b|bath))`)
)

func unitMixCandidates(key, text string) []Candidate {
	re, ok := unitMixTypes[key]
	if !ok {
		return nil
	}
	var out []Candidate
	seen := map[int]bool{}
	add := func(start, end int) {
		if seen[start] {
			return
		}
		seen[start] = true
		v := text[start:end]
		norm := v
		if n, ok := countWords[strings.ToLower(v)]; ok {
			norm = n
		}
		out = append(out, candidate(len(out), text, start, end, norm, "", ""))
	}
	for _, m := range re.FindAllStringIndex(text, -1) {
		before := text[:m[0]]
		if b := countBeforeRe.FindStringSubmatchIndex(before); b != nil {
			add(b[2], b[3])
		}
		after := text[m[1]:]
		if a := countAfterRe.FindStringSubmatchIndex(after); a != nil && !measureAfterRe.MatchString(after[a[3]:]) {
			add(m[1]+a[2], m[1]+a[3])
		}
	}
	return capped(out)
}

// IsUnitMix reports whether a scale field key is one of the mix counts.
func IsUnitMix(key string) bool { _, ok := unitMixTypes[key]; return ok }
