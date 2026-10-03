// Package profile drafts a project profile from filed documents. Code
// harvests candidates and reconciles readings; Jev only picks among what code
// offers (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
package profile

import (
	"fmt"
	"regexp"
	"strings"

	"sitewise/internal/knowledge"
)

// maxCandidates bounds one key's options in one passage; the first are kept.
const maxCandidates = 254

// Candidate is one verbatim value code found in a passage. Norm is the
// machine form (number without separators, ISO date); Value is never edited.
type Candidate struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Context string `json:"context"`
	Norm    string `json:"-"`
	Unit    string `json:"-"`
	Basis   string `json:"-"`
}

// Harvested is what code found in one passage: the question ids whose
// trigger matched (and, for pre-parsed keys, that have candidates), and the
// candidates per key.
type Harvested struct {
	Triggered  []string
	Candidates map[string][]Candidate
}

// Has reports whether key was triggered.
func (h Harvested) Has(key string) bool {
	for _, k := range h.Triggered {
		if k == key {
			return true
		}
	}
	return false
}

var (
	numberRe   = `(\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?)`
	areaRe     = regexp.MustCompile(`(?i)` + numberRe + `\s*(m2|m²|sqm|sq\.?\s?m|square met(?:re|er)s)\b`)
	metreRe    = regexp.MustCompile(`(?i)` + numberRe + `\s*m\b`)
	monthsRe   = regexp.MustCompile(`(?i)(\d+)\s*\(?\w*\)?\s*months?`)
	yearsRe    = regexp.MustCompile(`(?i)(\d+)\s*years?`)
	storeyRe   = regexp.MustCompile(`(?i)\b(?:[a-z]+\s*\()?(\d+|single|double|two|three|four|five|six|seven|eight|nine|ten)\)?[ -]*stor(?:e)?y`)
	consentRe  = regexp.MustCompile(`(?i)\b(?:DA|CDC|SSD)[- /]?\d[\w./-]*\d`)
	dateRe     = regexp.MustCompile(`\b(\d{1,2})[./](\d{1,2})[./](\d{4})\b`)
	longDateRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})\b`)
	integerRe  = regexp.MustCompile(`\b(\d{1,4})\b`)
	classRe    = regexp.MustCompile(`(?i)\bclass(?:ification)?\s*(1a|1b|10a|10b|10c|7a|7b|9a|9b|9c|[2-8])\b`)
	basisRes   = []struct {
		re    *regexp.Regexp
		basis string
	}{
		{regexp.MustCompile(`(?i)\bGLA\b|gross lettable`), "GLA"},
		{regexp.MustCompile(`(?i)\bNLA\b|net lettable`), "NLA"},
		{regexp.MustCompile(`(?i)\bGFA\b|gross floor area`), "GFA"},
		{regexp.MustCompile(`(?i)site area`), "site area"},
	}
	storeyWords = map[string]string{"single": "1", "double": "2", "two": "2", "three": "3", "four": "4", "five": "5", "six": "6", "seven": "7", "eight": "8", "nine": "9", "ten": "10"}
	monthNums   = map[string]int{"january": 1, "february": 2, "march": 3, "april": 4, "may": 5, "june": 6, "july": 7,
		"august": 8, "september": 9, "october": 10, "november": 11, "december": 12}
)

// Harvest finds triggered profile questions and candidates in one passage.
// It over-finds; Jev picks or answers none.
func Harvest(text string, cat *knowledge.Catalog) Harvested {
	h := Harvested{Candidates: map[string][]Candidate{}}
	if cat == nil || strings.TrimSpace(text) == "" {
		return h
	}
	t := knowledge.NewText(text)
	for _, d := range cat.ProfileDeterminants() {
		if d.Derived || !d.TriggeredIn(t) {
			continue
		}
		h.add("det."+d.ID, d.Extraction, candidatesFor(d.ID, d.Value, d.Extraction, text))
	}
	for _, f := range cat.ProjectFacts() {
		if !f.TriggeredIn(t) && !(f.ID == "consent_date" && strings.Contains(strings.ToLower(text), "date of determination")) {
			continue
		}
		h.add("fact."+f.ID, f.Extraction, candidatesFor(f.ID, f.Value, f.Extraction, text))
	}
	for _, f := range cat.ScaleFields() {
		if !f.TriggeredIn(t) && !(f.Key == "units" && unitCountRe.MatchString(text)) {
			continue
		}
		h.add("hdr.scale."+f.Key, "pre_parsed", scaleCandidates(f, text))
	}
	return h
}

// add records a trigger. A pre-parsed key with no candidates is not asked:
// Jev cannot pick a value code did not find.
func (h *Harvested) add(key, extraction string, cands []Candidate) {
	if extraction == "pre_parsed" && len(cands) == 0 {
		return
	}
	h.Triggered = append(h.Triggered, key)
	if len(cands) > 0 {
		h.Candidates[key] = cands
	}
}

func candidatesFor(id, value, extraction, text string) []Candidate {
	if id == "ncc_class" {
		return classCandidates(text)
	}
	if extraction != "pre_parsed" {
		return nil
	}
	switch {
	case id == "consent_number":
		return verbatim(consentRe, text, func(m []string) string { return strings.TrimRight(m[0], ".") })
	case strings.HasSuffix(id, "_date"):
		return dateCandidates(text)
	case id == "defects_liability_period":
		return verbatim(monthsRe, text, func(m []string) string { return m[1] })
	case id == "design_life":
		return verbatim(yearsRe, text, func(m []string) string { return m[1] })
	case id == "storage_height":
		return unitCandidates(metreRe, text, "m", "")
	case value == "number" || value == "integer":
		if c := unitCandidates(areaRe, text, "m2", basisOf(text)); len(c) > 0 {
			return c
		}
		return nil
	}
	return nil
}

var unitCountRe = regexp.MustCompile(`(?i)\b(\d+)\s+(?:sole\s+occupancy\s+)?(?:units|apartments|dwellings)\b`)
var bedroomCountRe = regexp.MustCompile(`(?i)\b(\d+)\s+bedrooms?\b`)
var parkingCountRe = regexp.MustCompile(`(?i)\b(\d+)\s+(?:car\s*(?:parking|park)?\s*spaces|parking\s+(?:spaces|bays)|car\s*parks)\b`)
var dockCountRe = regexp.MustCompile(`(?i)(?:^|[^\d.])(\d+)\s+(?:no\.?\s+)?(?:(?:loading\s+)?dock|roller\s+shutter)\s+doors?\b`)
var tenancyCountRe = regexp.MustCompile(`(?i)(?:^|[^\d.])(\d+)\s+(?:separate\s+)?tenancies\b`)

func scaleCandidates(f knowledge.ScaleField, text string) []Candidate {
	switch {
	case f.Key == "dock_doors":
		return verbatim(dockCountRe, text, func(m []string) string { return m[1] })
	case f.Key == "tenancies":
		return verbatim(tenancyCountRe, text, func(m []string) string { return m[1] })
	case f.Key == "bedrooms":
		return verbatim(bedroomCountRe, text, func(m []string) string { return m[1] })
	case f.Key == "units" || f.Key == "residential_units":
		return verbatim(unitCountRe, text, func(m []string) string { return m[1] })
	case f.Key == "car_parks":
		return verbatim(parkingCountRe, text, func(m []string) string { return m[1] })
	case f.Key == "storeys" || f.Key == "total_storeys":
		return verbatim(storeyRe, text, func(m []string) string {
			if n, ok := storeyWords[strings.ToLower(m[1])]; ok {
				return n
			}
			return m[1]
		})
	case f.Unit == "m2":
		return unitCandidates(areaRe, text, "m2", basisOf(text))
	case f.Unit == "m":
		return unitCandidates(metreRe, text, "m", "")
	case f.Type == "integer":
		return verbatim(integerRe, text, func(m []string) string { return m[1] })
	}
	return nil
}

func classCandidates(text string) []Candidate {
	seen := map[string]bool{}
	var out []Candidate
	for _, m := range classRe.FindAllStringSubmatchIndex(text, -1) {
		norm := strings.ToLower(text[m[2]:m[3]])
		if seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, candidate(len(out), text, m[0], m[1], norm, "", ""))
	}
	return capped(out)
}

func dateCandidates(text string) []Candidate {
	var out []Candidate
	for _, m := range dateRe.FindAllStringSubmatchIndex(text, -1) {
		d, mo, y := atoi(text[m[2]:m[3]]), atoi(text[m[4]:m[5]]), text[m[6]:m[7]]
		out = append(out, candidate(len(out), text, m[0], m[1], fmt.Sprintf("%s-%02d-%02d", y, mo, d), "", ""))
	}
	for _, m := range longDateRe.FindAllStringSubmatchIndex(text, -1) {
		d, mo, y := atoi(text[m[2]:m[3]]), monthNums[strings.ToLower(text[m[4]:m[5]])], text[m[6]:m[7]]
		out = append(out, candidate(len(out), text, m[0], m[1], fmt.Sprintf("%s-%02d-%02d", y, mo, d), "", ""))
	}
	return capped(out)
}

func unitCandidates(re *regexp.Regexp, text, unit, basis string) []Candidate {
	var out []Candidate
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		norm := strings.ReplaceAll(text[m[2]:m[3]], ",", "")
		out = append(out, candidate(len(out), text, m[0], m[1], norm, unit, basis))
	}
	return capped(out)
}

func verbatim(re *regexp.Regexp, text string, norm func([]string) string) []Candidate {
	var out []Candidate
	for _, idx := range re.FindAllStringSubmatchIndex(text, -1) {
		groups := make([]string, len(idx)/2)
		for g := range groups {
			if idx[2*g] >= 0 {
				groups[g] = text[idx[2*g]:idx[2*g+1]]
			}
		}
		end := idx[0] + len(strings.TrimRight(groups[0], "."))
		out = append(out, candidate(len(out), text, idx[0], end, norm(groups), "", ""))
	}
	return capped(out)
}

func candidate(i int, text string, start, end int, norm, unit, basis string) Candidate {
	return Candidate{ID: fmt.Sprintf("c%d", i+1), Value: text[start:end], Context: context(text, start, end),
		Norm: norm, Unit: unit, Basis: basis}
}

// context is up to 80 characters around a match on one line, so Jev can see
// what a candidate refers to without reading the whole passage again.
func context(text string, start, end int) string {
	lo, hi := start-40, end+40
	if lo < 0 {
		lo = 0
	}
	if hi > len(text) {
		hi = len(text)
	}
	for lo > 0 && !utf8Start(text[lo]) {
		lo--
	}
	for hi < len(text) && !utf8Start(text[hi]) {
		hi++
	}
	return strings.Join(strings.Fields(text[lo:hi]), " ")
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

func basisOf(text string) string {
	for _, b := range basisRes {
		if b.re.MatchString(text) {
			return b.basis
		}
	}
	return ""
}

func capped(c []Candidate) []Candidate {
	if len(c) > maxCandidates {
		return c[:maxCandidates]
	}
	return c
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}
