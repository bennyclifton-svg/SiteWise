package profile

import (
	"regexp"
	"strings"
)

// Candidate finders for the project details Jev reads (knowledge/profile/
// project_details.yaml). Each over-finds: Jev picks the candidate the passage
// states for this project, or none, and code copies the text, so the value
// stored is always a verbatim span (https://docs.typesafe.ai/cookbooks/
// pre_parsed_value_extraction_cookbook).

const streetTypes = `(?:Street|St|Road|Rd|Avenue|Ave|Parade|Pde|Drive|Dr|Place|Pl|Lane|Ln|Highway|Hwy|Boulevard|Blvd|Crescent|Cres|Court|Ct|Close|Terrace|Tce|Way|Circuit|Cct|Esplanade|Grove)`

var (
	// number, street name (one to three capitalised words), street type, then
	// at most one suburb word. A second suburb word is taken only after the
	// few words that start two-word suburbs (see suburbSecond).
	addressRe = regexp.MustCompile(`\b(\d{1,4}[A-Za-z]?(?:\s*[-–/]\s*\d{1,4}[A-Za-z]?)?)\s+((?:[A-Z][A-Za-z'’]+\s+){1,3}?` + streetTypes + `)\b\.?(?:,?\s+([A-Z][a-z]+)(?:\s+([A-Z][a-z]+))?)?(?:,?\s+(NSW|VIC|QLD|SA|WA|TAS|NT|ACT)\b)?(?:\s+(\d{4})\b)?`)
	// suburbFirst are first words of two-word suburbs (Mount Druitt, North Sydney).
	suburbFirst = map[string]bool{"Mount": true, "Mt": true, "North": true, "South": true, "East": true, "West": true, "St": true,
		"Port": true, "Lake": true, "Glen": true, "Castle": true, "New": true, "Old": true, "Upper": true, "Lower": true,
		"Surry": true, "Rose": true, "Bella": true, "Kings": true, "Point": true, "Bondi": true, "Darling": true}

	lotRe = regexp.MustCompile(`(?i)\bLots?\s+\d+[A-Z]?(?:\s*(?:,|and|&)\s*\d+[A-Z]?)*(?:\s+(?:in|on|of))?(?:,?\s+Section\s+\d+[A-Z]?)?,?\s*(?:DP|D\.P\.|SP|RP|PS|CP|Deposited Plan|Strata Plan)\s*\d+`)
	lotIn = regexp.MustCompile(`(?i)\s+(?:in|on|of)\s+`)

	clientCueRe = regexp.MustCompile(`(?i)\b(?:Principal|Client|Owner|Proponent|Applicant|Developer|Landowner)\b(?:\s*\([^)]{0,40}\))?\s*[:–]\s*|\b(?:prepared for|commissioned by)\s+`)
	wordRe      = regexp.MustCompile(`\S+`)
	// clientStop ends a name: the next label or number on a cover block.
	clientStop = map[string]bool{"ABN": true, "ACN": true, "DRAFT": true, "Revision": true, "Rev": true, "Date": true,
		"Project": true, "Address": true, "Phone": true, "Email": true, "Contact": true, "Architect": true,
		"Prepared": true, "Issue": true, "Version": true, "Job": true, "Page": true, "Principal": true}
	clientSuffix = map[string]bool{"pty": true, "ltd": true, "inc": true, "co": true, "corp": true}
	// clientSuffixEnd close a company name; a full stop after them ends it.
	clientSuffixEnd = map[string]bool{"ltd": true, "inc": true, "co": true, "corp": true}
	clientJoin      = map[string]bool{"and": true, "of": true, "the": true, "&": true}

	lgaAreaRe   = regexp.MustCompile(`((?:[A-Z][A-Za-z'’-]*)(?:\s+[A-Z][A-Za-z'’-]*){0,2})\s+(?i:local\s+government\s+area|LGA)\b`)
	lgaCouncil  = regexp.MustCompile(`((?:[A-Z][A-Za-z'’-]*)(?:\s+[A-Z][A-Za-z'’-]*){0,2})\s+Council\b`)
	lgaCityOf   = regexp.MustCompile(`\b((?:City|Shire|Municipality|Town|Borough|Rural City|Regional Council) of\s+[A-Z][A-Za-z'’-]*(?:\s+[A-Z][A-Za-z'’-]*){0,2})`)
	lgaGeneric  = map[string]bool{"The": true, "Local": true, "Relevant": true, "Each": true, "Any": true, "This": true, "That": true, "Council": true, "Planning": true}
	lgaTrailing = map[string]bool{"City": true, "Municipal": true, "Shire": true, "Regional": true, "Rural": true, "Town": true, "Council": true}

	zoneCode = `(?:R[1-5]|B[1-8]|E[1-5]|MU1|IN[1-4]|SP[1-3]|RE[12]|RU[1-6]|SU[1-2]|W[1-4]|C[1-4])`
	zoneRe   = regexp.MustCompile(`\b(` + zoneCode + `)\b((?:\s+(?:[A-Z][a-z]+|and|&)){0,3})`)

	// An amount with its unit and any GST basis, so the stored span carries
	// both: "$1.8 million excluding GST".
	moneyRe = regexp.MustCompile(`(?i)(?:\$|\bAUD\s*)\s*\d[\d,]*(?:\.\d+)?(?:\s*(?:million|m|k)\b)?(?:\s*\(?(?:excl(?:uding|\.)?|ex\.?|incl(?:uding|\.)?|inc\.?)\s*(?:of\s+)?GST\)?)?`)

	sustainRe = regexp.MustCompile(`(?i)\b(?:\d(?:\.\d)?\s*[- ]?star\s+)?(?:Green\s+Star|NABERS|NatHERS|Passive\s*Haus|Passive\s*House|LEED|Net[- ]Zero)\b[^.;\n]{0,60}`)
)

// streetTypeFull writes abbreviated street types out, so "Parramatta Rd" and
// "Parramatta Road" in different passages are one address, not a conflict.
var streetTypeFull = map[string]string{"St": "Street", "Rd": "Road", "Ave": "Avenue", "Pde": "Parade", "Dr": "Drive", "Pl": "Place",
	"Ln": "Lane", "Hwy": "Highway", "Blvd": "Boulevard", "Cres": "Crescent", "Ct": "Court", "Tce": "Terrace", "Cct": "Circuit"}

// notSuburb are capitalised words that follow a street in running text.
var notSuburb = map[string]bool{"Additional": true, "Proposed": true, "Refer": true, "Principal": true, "Residential": true,
	"Commercial": true, "Existing": true, "Note": true, "See": true, "Building": true, "Level": true, "Unit": true,
	"Project": true, "Contractor": true, "Council": true, "Street": true, "Road": true, "Works": true, "Site": true}

// detailCandidates returns the verbatim candidates for one detail key
// ("site.address", "proj.client") in a passage.
func detailCandidates(key, text string) []Candidate {
	switch key {
	case "site.address":
		return addressCandidates(text)
	case "site.lot":
		return lotCandidates(text)
	case "proj.client":
		return append(clientCandidates(text), addresseeCandidates(text)...)
	case "proj.lga":
		return lgaCandidates(text)
	case "proj.zone":
		return zoneCandidates(text)
	case "proj.sustainability":
		return sustainabilityCandidates(text)
	case "proj.budget_stated", "proj.contingency_stated":
		return budgetCandidates(text)
	case "proj.owner_supply_stated":
		return clauseCandidates(text, ownerSupplyRe)
	case "proj.exclusions_stated":
		return clauseCandidates(text, exclusionRe)
	}
	return nil
}

var (
	// ownerSupplyRe finds the clause after an owner-supply cue, up to the end
	// of its sentence: "Owners will supply: pendant lights (allowance $8,000),
	// appliances package (allowance $22,000)".
	ownerSupplyRe = regexp.MustCompile(`(?i)\b(?:(?:owners?|client|principal)s?\b[^.\n:]{0,30}\bsuppl(?:y|ies|ied)|(?:owner|client|principal)[- ]supplied(?:\s+items)?|supplied\s+by\s+(?:the\s+)?(?:owners?|client|principal))\s*:?\s*([^\n]{3,300}?)(?:\.\s|\.$|\n|$)`)
	// exclusionRe finds what a budget excludes, up to a closing bracket or
	// the end of its sentence: "(excluding landscaping, loose furniture, and
	// authority fees)".
	exclusionRe = regexp.MustCompile(`(?i)\b(?:excluding|excludes|exclusions?\s*:)\s*([^)\n.]{3,200})`)
)

// clauseCandidates offers each clause the pattern captures, verbatim.
func clauseCandidates(text string, re *regexp.Regexp) []Candidate {
	var spans []span
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		clause := strings.Trim(strings.TrimSpace(text[m[2]:m[3]]), "*,;: ")
		if clause != "" {
			spans = append(spans, span{m[2], m[3], clause})
		}
	}
	return collect(text, spans)
}

// span is a verbatim stretch of the passage with its stored form.
type span struct {
	start, end int
	norm       string
}

// collect turns spans into candidates, one per distinct stored value.
func collect(text string, spans []span) []Candidate {
	var out []Candidate
	seen := map[string]bool{}
	for _, s := range spans {
		norm := strings.Join(strings.Fields(s.norm), " ")
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, candidate(len(out), text, s.start, s.end, norm, "", ""))
	}
	return capped(out)
}

// capsWord is an ASCII word in capitals; titleCaseCaps rewrites it in place.
var capsWord = regexp.MustCompile(`\b[A-Z]{2,}\b`)

// stateCodes stay in capitals: the address pattern reads them as states.
var stateCodes = map[string]bool{"NSW": true, "VIC": true, "QLD": true, "SA": true, "WA": true, "TAS": true, "NT": true, "ACT": true}

// titleCaseCaps writes capitalised words in title case ("FORRESTER RD" to
// "Forrester Rd") so a title block address reads like running text. ASCII
// only, so every byte offset still points into the original text.
func titleCaseCaps(text string) string {
	return capsWord.ReplaceAllStringFunc(text, func(w string) string {
		if stateCodes[w] {
			return w
		}
		return w[:1] + strings.ToLower(w[1:])
	})
}

func addressCandidates(text string) []Candidate {
	var spans []span
	// Offsets come from the title-cased copy; the candidate keeps the
	// verbatim text and the written-out form is its norm.
	titled := titleCaseCaps(text)
	for _, m := range addressRe.FindAllStringSubmatchIndex(titled, -1) {
		group := func(g int) string {
			if m[2*g] < 0 {
				return ""
			}
			return titled[m[2*g]:m[2*g+1]]
		}
		// Groups: 1 number, 2 street, 3 and 4 suburb words, 5 state, 6 postcode.
		end := m[5]
		if m[6] >= 0 && !notSuburb[group(3)] {
			end = m[7]
			// A second word is a suburb only after a word that starts two-word
			// suburbs; otherwise it is the next sentence ("Petersham Principal").
			if group(4) != "" {
				if !suburbFirst[group(3)] {
					spans = append(spans, span{m[0], end, writeOutStreetType(titled[m[0]:end], titled[m[4]:m[5]])})
					continue
				}
				end = m[9]
			}
			for _, g := range []int{5, 6} {
				if m[2*g] >= 0 {
					end = m[2*g+1]
				}
			}
		}
		spans = append(spans, span{m[0], end, writeOutStreetType(titled[m[0]:end], titled[m[4]:m[5]])})
	}
	return collect(text, spans)
}

// writeOutStreetType replaces the abbreviated street type at the end of the
// street part of an address ("Parramatta Rd") with its full word.
func writeOutStreetType(address, street string) string {
	words := strings.Fields(street)
	last := strings.TrimRight(words[len(words)-1], ".")
	full, ok := streetTypeFull[last]
	if !ok {
		return address
	}
	i := strings.Index(address, street)
	if i < 0 {
		return address
	}
	return address[:i] + strings.TrimSuffix(street, words[len(words)-1]) + full + address[i+len(street):]
}

func lotCandidates(text string) []Candidate {
	var spans []span
	for _, m := range lotRe.FindAllStringIndex(text, -1) {
		spans = append(spans, span{m[0], m[1], lotIn.ReplaceAllString(text[m[0]:m[1]], " ")})
	}
	return collect(text, spans)
}

// clientCandidates reads the capitalised name that follows a label such as
// "Principal:" or "Prepared for". The name ends at a label, a lower-case word,
// a line break or punctuation; "Pty." and "Ltd." keep their full stop.
func clientCandidates(text string) []Candidate {
	var spans []span
	for _, cue := range clientCueRe.FindAllStringIndex(text, -1) {
		rest := text[cue[1]:]
		if len(rest) > 160 {
			rest = rest[:160]
		}
		var words [][2]int
		prevEnd := -1
		for _, w := range wordRe.FindAllStringIndex(rest, -1) {
			if prevEnd >= 0 && strings.ContainsRune(rest[prevEnd:w[0]], '\n') {
				break
			}
			word := rest[w[0]:w[1]]
			bare := strings.TrimRight(word, ".,;:")
			if bare == "" || clientStop[bare] {
				break
			}
			upper := bare[0] >= 'A' && bare[0] <= 'Z' || bare[0] >= '0' && bare[0] <= '9'
			if !upper && !(len(words) > 0 && clientJoin[strings.ToLower(bare)]) {
				break
			}
			end, trail := w[0]+len(bare), word[len(bare):]
			if trail == "." && clientSuffix[strings.ToLower(bare)] {
				end, trail = w[1], ""
				if clientSuffixEnd[strings.ToLower(bare)] {
					trail = "."
				}
			}
			words = append(words, [2]int{w[0], end})
			prevEnd = w[1]
			if trail != "" || len(words) >= 8 {
				break
			}
		}
		for len(words) > 0 && clientJoin[strings.ToLower(rest[words[len(words)-1][0]:words[len(words)-1][1]])] {
			words = words[:len(words)-1]
		}
		if len(words) == 0 {
			continue
		}
		start, end := words[0][0], words[len(words)-1][1]
		spans = append(spans, span{cue[1] + start, cue[1] + end, strings.TrimRight(rest[start:end], ".")})
	}
	return collect(text, spans)
}

func lgaCandidates(text string) []Candidate {
	var spans []span
	add := func(start, end int, name string) {
		name = strings.TrimSpace(name)
		for {
			words := strings.Fields(name)
			if len(words) > 0 && (lgaGeneric[words[0]] && len(words) > 1) {
				name = strings.Join(words[1:], " ")
				continue
			}
			if len(words) > 1 && lgaTrailing[words[len(words)-1]] && !strings.Contains(name, " of ") {
				name = strings.Join(words[:len(words)-1], " ")
				continue
			}
			break
		}
		if words := strings.Fields(name); len(words) == 0 || lgaGeneric[words[0]] {
			return
		}
		spans = append(spans, span{start, end, name})
	}
	for _, re := range []*regexp.Regexp{lgaAreaRe, lgaCouncil} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			add(m[0], m[1], text[m[2]:m[3]])
		}
	}
	for _, m := range lgaCityOf.FindAllStringSubmatchIndex(text, -1) {
		name := strings.TrimSuffix(text[m[2]:m[3]], " Council")
		add(m[0], m[1], name)
	}
	return collect(text, spans)
}

// zoneCandidates takes a zone code with its name, only where the word "zone"
// precedes it closely, so a stray grade or room code is not offered.
func zoneCandidates(text string) []Candidate {
	var spans []span
	for _, m := range zoneRe.FindAllStringSubmatchIndex(text, -1) {
		lo := m[0] - 60
		if lo < 0 {
			lo = 0
		}
		if !strings.Contains(strings.ToLower(text[lo:m[0]]), "zon") {
			continue
		}
		code := text[m[2]:m[3]]
		spans = append(spans, span{m[0], m[3], code})
		raw := text[m[4]:m[5]]
		name := strings.TrimSpace(raw)
		for _, join := range []string{" and", " &"} {
			name = strings.TrimSuffix(name, join)
		}
		if name != "" {
			start := m[4] + len(raw) - len(strings.TrimLeft(raw, " \t\r\n"))
			spans = append(spans, span{m[0], start + len(name), code + " " + name})
		}
	}
	return collect(text, spans)
}

func sustainabilityCandidates(text string) []Candidate {
	var spans []span
	for _, m := range sustainRe.FindAllStringIndex(text, -1) {
		end := m[1]
		// The 60-character tail may stop mid-word; end at the last word.
		if end < len(text) && text[end] != ' ' && text[end] != '.' && text[end] != ',' {
			if i := strings.LastIndex(text[m[0]:end], " "); i > 0 {
				end = m[0] + i
			}
		}
		name := strings.TrimRight(text[m[0]:end], " ,:;")
		spans = append(spans, span{m[0], m[0] + len(name), name})
	}
	return collect(text, spans)
}

// budgetCandidates lists every dollar amount; Jev picks the one the passage
// states as the construction budget, or none. Code parses the amount later.
func budgetCandidates(text string) []Candidate {
	var spans []span
	for _, m := range moneyRe.FindAllStringIndex(text, -1) {
		spans = append(spans, span{m[0], m[1], strings.TrimSpace(text[m[0]:m[1]])})
	}
	return collect(text, spans)
}
