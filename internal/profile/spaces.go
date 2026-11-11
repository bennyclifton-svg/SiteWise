package profile

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/knowledge"
)

// Accommodation: the spaces a building provides after the works
// (docs/plans/2026-11-11-accommodation-and-full-capture.md). Code finds each
// space's name on a line, then every count, area, capacity and level on that
// line, and gives each number to the nearest space name in its clause. Jev
// says whether the passage states the project provides the space and picks
// one candidate per value or none
// (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).

var (
	// spaceCapacityRe is people or seats a space holds: "8-person", "60-seat".
	spaceCapacityRe = regexp.MustCompile(`(?i)\b(\d{1,4})\s*[- ]?\s*(?:persons?|people|pax|seats?|seater|places|students|pupils|children)\b`)
	// spaceSeatsRe is a capacity written after its word: "seats 400",
	// "seating for 60", "capacity of 120".
	spaceSeatsRe = regexp.MustCompile(`(?i)\b(?:seats|seating\s+for|capacity(?:\s+of)?|accommodat(?:es|ing))\s+(?:up\s+to\s+)?(\d{1,4})\b`)
	// spaceLevelRe is a floor: "Level 3", "L2", "Ground floor". A range
	// ("Levels 3–4", "Levels 3 and partial Level 4") is read whole.
	spaceLevelRe = regexp.MustCompile(`(?i)\b(?:levels?\s+(\d{1,2})(?:\s*(?:–|-|to|and|&)\s*(?:partial\s+|part\s+(?:of\s+)?)?(?:level\s+)?(\d{1,2})\b)?|L(\d{1,2})\b|(ground|first|second|third|lower\s+ground|upper\s+ground|basement|mezzanine|roof(?:top)?)[- ](?:floor|level)|(ground|basement|rooftop)\b)`)
	// spaceCountRe is a whole number, or a small number in words.
	spaceCountRe = regexp.MustCompile(`(?i)\b(\d{1,4}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|fifteen|twenty)\b`)
	// spaceSkipRe marks numbers that are not counts: money, times, dates,
	// shares, decimals and units of other measures.
	spaceSkipRe = regexp.MustCompile(`(?i)\$\s*\d[\d,.]*\s*(?:k|m|million)?|\b\d{1,2}[:.]\d{2}\b|\b\d{1,2}\s*(?:am|pm)\b|\b\d{1,2}\s+(?:january|february|march|april|may|june|july|august|september|october|november|december)\b|\b(?:18|19|20)\d{2}\b|\d+(?:\.\d+)?\s*(?:%|kva|kw|l/s|°|mm\b|m\b|weeks?|months?|years?|days?|hours?|stars?|storeys?|stories)|\d+\.\d+`)
	// spaceTimesRe is one "N × item" of a list: "4 × 8-person", "2 × male".
	spaceTimesRe = regexp.MustCompile(`(?i)\b(\d{1,3})\s*[×x]\s*([^,;|]+)`)
	// clauseBreak ends a clause: a semicolon, or a full stop that does not sit
	// inside a number.
	clauseBreak = regexp.MustCompile(`;|\.(?:\s|$)`)
	// clauseOrItemBreak also ends a clause at a comma between list items.
	clauseOrItemBreak = regexp.MustCompile(`;|\.(?:\s|$)|,\s`)
	// spaceAdjacentRe places one space by another: "kitchen adjacent to the
	// dining room", "opening onto the deck".
	spaceAdjacentRe = regexp.MustCompile(`(?i)\b(?:adjacent\s+to|next\s+to|close\s+to|adjoin(?:s|ing)?|beside|connected\s+to|direct\s+access\s+to|open(?:s|ing)?\s+(?:on)?to)\b`)
	// spaceEachAfterRe and spaceEachBeforeRe mark an area written per room:
	// "14 m² each", "each approx. 14 m²".
	spaceEachAfterRe  = regexp.MustCompile(`(?i)^\s*(?:each|per\s+(?:room|office|unit|suite|space|pod|bedroom|classroom|bay)|apiece)\b`)
	spaceEachBeforeRe = regexp.MustCompile(`(?i)\beach\s+(?:(?:of\s+)?(?:approx(?:imately)?\.?|about|around|~)\s*)?$`)
)

// spaceMatch is one space name on a line.
type spaceMatch struct {
	id         string
	start, end int
}

// spaceToken is one value found on a line, before it is given to a space.
type spaceToken struct {
	attr       string
	start, end int
	norm, unit string
	list       bool // a list total, which belongs to the row's first-cell space
}

// harvestSpaces adds each space named in text and the values written beside
// it. A space name with no values still asks whether it is provided.
func (h *Harvested) harvestSpaces(text string, cat *knowledge.Catalog) {
	cands := map[string][]Candidate{}
	named := map[string]bool{}
	// Comment keys: the space's requirement phrases and the spaces it sits
	// next to, each a yes/no question about one space.
	notes := map[string]bool{}
	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		matches := spaceMatches(line, cat)
		for _, m := range matches {
			named[m.id] = true
		}
		for _, key := range spaceNotes(line, matches, cat) {
			notes[key] = true
		}
		if len(matches) > 0 {
			for _, tok := range spaceTokens(line, matches) {
				id, far := nearestSpace(line, tok.start, tok.end, matches)
				if far {
					continue // too far from any space name to describe one
				}
				if next, ok := spaceRightAfter(line, tok.end, matches); ok && (tok.attr == "count" || tok.attr == "capacity") && !tok.list {
					id = next // "1 × comms room", "60-seat breakout"
				}
				if first, ok := firstCellSpace(line, matches); ok && tok.list {
					id = first
				}
				key := knowledge.SpacePrefix + id + "." + tok.attr
				cands[key] = append(cands[key], candidate(len(cands[key]), text, offset+tok.start, offset+tok.end, tok.norm, tok.unit, ""))
			}
		}
		offset += len(line)
	}
	ids := make([]string, 0, len(named))
	for id := range named {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		h.add(knowledge.SpacePrefix+id+".provided", "choice", nil)
		for _, attr := range knowledge.SpaceAttributes[1:] {
			key := knowledge.SpacePrefix + id + "." + attr
			h.add(key, "pre_parsed", distinctCandidates(cands[key]))
		}
	}
	keys := make([]string, 0, len(notes))
	for key := range notes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		h.add(key, "choice", nil)
	}
}

// spaceNotes lists the comment keys a line asks: each requirement phrase
// given to the space named in its clause ("secure records room", "meeting
// rooms (AV-ready)"), and each space placed next to another ("kitchen
// adjacent to the dining room": the kitchen is the one placed).
func spaceNotes(line string, matches []spaceMatch, cat *knowledge.Catalog) []string {
	if len(matches) == 0 {
		return nil
	}
	var out []string
	for _, r := range cat.SpaceRequirements() {
		for _, m := range r.FindAll(line) {
			if namesSpace(m[0], m[1], matches) {
				continue // the words are part of a space's own name
			}
			id, far := nearestSpace(line, m[0], m[1], matches)
			if next, ok := spaceRightAfter(line, m[1], matches); ok {
				id, far = next, false
			}
			if !far && id != "" {
				out = append(out, knowledge.SpacePrefix+id+".need."+r.ID)
			}
		}
	}
	_, table := firstCellSpace(line, matches)
	for _, cue := range spaceAdjacentRe.FindAllStringIndex(line, -1) {
		lo, hi := clauseAround(line, cue[0], cue[1], !table)
		owner, target := "", ""
		for _, m := range matches {
			if m.start >= lo && m.end <= cue[0] {
				owner = m.id // the last space named before the cue
			}
			if target == "" && m.start >= cue[1] && m.end <= hi && nearAfter(line[cue[1]:m.start]) {
				target = m.id
			}
		}
		if owner != "" && target != "" && owner != target {
			out = append(out, knowledge.SpacePrefix+owner+".next_to."+target)
		}
	}
	return out
}

// nearAfter reports whether a gap holds at most three words and no
// punctuation: "the", "the main", not "and then, later, the".
func nearAfter(gap string) bool {
	return len(strings.Fields(gap)) <= 3 && !strings.ContainsAny(gap, ",;|()")
}

// spaceMatches finds the space names on a line. Where two names overlap the
// longer wins, so "accessible WC" is not also a toilet.
func spaceMatches(line string, cat *knowledge.Catalog) []spaceMatch {
	var all []spaceMatch
	for _, s := range cat.Spaces() {
		for _, m := range s.FindAll(line) {
			all = append(all, spaceMatch{s.ID, m[0], m[1]})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].end-all[i].start > all[j].end-all[j].start })
	var kept []spaceMatch
	for _, m := range all {
		overlaps := false
		for _, k := range kept {
			overlaps = overlaps || m.start < k.end && k.start < m.end
		}
		if !overlaps {
			kept = append(kept, m)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].start < kept[j].start })
	return kept
}

// spaceTokens lists the areas, capacities, levels and counts on a line. A
// number used by one is not offered as another; a number inside a space's
// own name ("8 offices") is its count.
func spaceTokens(line string, matches []spaceMatch) []spaceToken {
	var out []spaceToken
	_, table := firstCellSpace(line, matches)
	used := make([]bool, len(line)+1)
	take := func(t spaceToken) {
		for i := t.start; i < t.end; i++ {
			used[i] = true
		}
		out = append(out, t)
	}
	free := func(start, end int) bool {
		for i := start; i < end; i++ {
			if used[i] {
				return false
			}
		}
		return true
	}
	for _, m := range areaRe.FindAllStringSubmatchIndex(line, -1) {
		attr := "area_sqm"
		if spaceEachAfterRe.MatchString(line[m[1]:]) || spaceEachBeforeRe.MatchString(line[:m[0]]) {
			attr = "area_each_sqm" // "14 m² each": every room is that size
		}
		take(spaceToken{attr, m[0], m[1], strings.ReplaceAll(line[m[2]:m[3]], ",", ""), "m2", false})
	}
	for _, re := range []*regexp.Regexp{spaceCapacityRe, spaceSeatsRe} {
		for _, m := range re.FindAllStringSubmatchIndex(line, -1) {
			if free(m[0], m[1]) {
				take(spaceToken{"capacity", m[0], m[1], line[m[2]:m[3]], "", false})
			}
		}
	}
	var levels []spaceToken
	for _, m := range spaceLevelRe.FindAllStringSubmatchIndex(line, -1) {
		if free(m[0], m[1]) {
			t := spaceToken{"level", m[0], m[1], levelName(line, m), "", false}
			take(t)
			levels = append(levels, t)
		}
	}
	for _, m := range spaceSkipRe.FindAllStringIndex(line, -1) {
		for i := m[0]; i < m[1]; i++ {
			used[i] = true
		}
	}
	for _, m := range spaceCountRe.FindAllStringSubmatchIndex(line, -1) {
		if !free(m[0], m[1]) {
			continue
		}
		word := strings.ToLower(line[m[2]:m[3]])
		norm, isWord := countWords[word]
		if !isWord {
			norm = word
		}
		// A count is written against its space: just before the name ("8
		// partner offices", "six meeting rooms"), just after it ("Meeting
		// rooms: 6"), or in a table row that names it. "Demolition of 3
		// walls" beside a boardroom counts no boardrooms.
		if !beforeSpace(line, m[1], matches) && !(table && !isWord) && !afterSpace(line, m[0], matches) {
			continue
		}
		take(spaceToken{"count", m[0], m[1], norm, "", false})
	}
	out = append(out, listTotals(line, matches)...)
	out = append(out, joinedLevels(levels, matches, line)...)
	return out
}

// levelName writes a level the same way wherever it is found: "Level 3",
// "Ground floor".
func levelName(line string, m []int) string {
	switch {
	case m[2] >= 0 && m[4] >= 0:
		return "Level " + line[m[2]:m[3]] + ", Level " + line[m[4]:m[5]]
	case m[2] >= 0:
		return "Level " + line[m[2]:m[3]]
	case m[6] >= 0:
		return "Level " + line[m[6]:m[7]]
	case m[8] >= 0:
		return strings.ToUpper(line[m[8]:m[8]+1]) + strings.ToLower(strings.Join(strings.Fields(line[m[8]+1:m[9]]), " ")) + " floor"
	}
	return strings.ToUpper(line[m[10]:m[10]+1]) + strings.ToLower(line[m[10]+1:m[11]])
}

// beforeSpace reports whether a space name starts within two words after at.
func beforeSpace(line string, at int, matches []spaceMatch) bool {
	_, ok := spaceRightAfter(line, at, matches)
	return ok
}

// afterSpace reports whether a space name ends just before at, past a
// colon, bracket, dash or equals sign: "Meeting rooms: 6", "workstations (42)".
func afterSpace(line string, at int, matches []spaceMatch) bool {
	for _, s := range matches {
		if s.end <= at && strings.Trim(line[s.end:at], " \t:=(–—-") == "" {
			return true
		}
	}
	return false
}

// spaceRightAfter is the space named within two words after at, past a
// multiplication sign: a count or capacity written before its noun
// ("8 partner offices", "1 × comms room", "60-seat breakout").
func spaceRightAfter(line string, at int, matches []spaceMatch) (string, bool) {
	for _, s := range matches {
		if s.start < at {
			continue
		}
		gap := strings.Fields(strings.NewReplacer("×", " ", " x ", " ").Replace(line[at:s.start]))
		if len(gap) <= 2 && !strings.ContainsAny(line[at:s.start], ",;|()") {
			return s.id, true
		}
		return "", false
	}
	return "", false
}

// listTotals offers the total of a list of "N × item" written for one space
// ("4 × 8-person, 2 × 16-person" is 6), as one more count candidate. Items
// naming another space ("1 × accessible WC") are left out of the total. Code
// adds; Jev only says whether the total is the space's count.
//
// When every item is a room size ("4 × 8-person, 2 × 16-person") the list is
// also offered as the space's mix, "4x8,2x16", so each line can carry its
// own capacity.
func listTotals(line string, matches []spaceMatch) []spaceToken {
	var out []spaceToken
	items := spaceTimesRe.FindAllStringSubmatchIndex(line, -1)
	for i := 0; i < len(items); {
		j, total, end := i, 0, 0
		var sizes []string
		sized := true
		for ; j < len(items); j++ {
			it := items[j]
			if j > i && strings.Trim(line[end:it[0]], " ,") != "" {
				break // not the same list
			}
			if namesSpace(it[0], it[1], matches) {
				break
			}
			n, _ := strconv.Atoi(line[it[2]:it[3]])
			total += n
			end = it[1]
			item := line[it[4]:it[5]]
			if c := spaceCapacityRe.FindStringSubmatchIndex(item); c != nil && strings.TrimSpace(item[:c[0]]) == "" {
				sizes = append(sizes, strconv.Itoa(n)+"x"+item[c[2]:c[3]])
			} else {
				sized = false
			}
		}
		if j-i >= 2 {
			out = append(out, spaceToken{"count", items[i][0], end, strconv.Itoa(total), "", true})
			if sized {
				out = append(out, spaceToken{"mix", items[i][0], end, strings.Join(sizes, ","), "", true})
			}
		}
		i = max(j, i+1)
	}
	return out
}

func namesSpace(start, end int, matches []spaceMatch) bool {
	for _, m := range matches {
		if m.start < end && start < m.end {
			return true
		}
	}
	return false
}

// joinedLevels offers, for a space with several levels on its line, the
// levels joined in order ("Level 3, Level 4"), for one space on each.
func joinedLevels(levels []spaceToken, matches []spaceMatch, line string) []spaceToken {
	byID := map[string][]spaceToken{}
	for _, l := range levels {
		if id, far := nearestSpace(line, l.start, l.end, matches); !far {
			byID[id] = append(byID[id], l)
		}
	}
	var out []spaceToken
	for _, ls := range byID {
		var names []string
		seen := map[string]bool{}
		for _, l := range ls {
			if !seen[l.norm] {
				seen[l.norm] = true
				names = append(names, l.norm)
			}
		}
		if len(names) >= 2 {
			out = append(out, spaceToken{"level", ls[0].start, ls[len(ls)-1].end, strings.Join(names, ", "), "", false})
		}
	}
	return out
}

// spaceReach is how far, in bytes, a value outside a table row may sit from
// the space name it describes. "mezzanine insert to Level 4, revised
// occupancy load, open-plan area, new meeting suite" puts no meeting room on
// Level 4.
const spaceReach = 60

// nearestSpace is the space a value at [start, end) describes: the nearest
// name in the same clause, else the table row's first-cell space. On a tie
// the name after the number wins ("8 partner offices"). far is true when,
// outside a table row, no name is in the clause or the nearest is beyond
// spaceReach.
func nearestSpace(line string, start, end int, matches []spaceMatch) (id string, far bool) {
	first, table := firstCellSpace(line, matches)
	lo, hi := clauseAround(line, start, end, !table)
	var pool []spaceMatch
	for _, m := range matches {
		if m.start >= lo && m.end <= hi {
			pool = append(pool, m)
		}
	}
	if len(pool) == 0 {
		return first, !table
	}
	best, bestD, after := "", 1<<30, false
	for _, m := range pool {
		d, follows := 0, false
		switch {
		case m.start >= end:
			d, follows = m.start-end, true
		case m.end <= start:
			d = start - m.end
		}
		if d < bestD || d == bestD && follows && !after {
			best, bestD, after = m.id, d, follows
		}
	}
	return best, !table && bestD > spaceReach
}

// clauseAround is the clause holding [start, end): bounded by semicolons
// and sentence stops, and in running text by commas too, so one item of a
// list ("Levels 3–4 fit-out, mezzanine insert, AV-ready meeting rooms") says
// nothing about another. A table row keeps its commas and its cell borders:
// the first cell names the space ("Open-plan legal library") and the next
// gives its values ("~180 m² open work area with 42 workstations",
// "3 on Level 3, 5 on Level 4").
func clauseAround(line string, start, end int, commas bool) (int, int) {
	lo, hi := 0, len(line)
	breaks := clauseBreak
	if commas {
		breaks = clauseOrItemBreak
	}
	for _, m := range breaks.FindAllStringIndex(line, -1) {
		if m[1] <= start && m[1] > lo {
			lo = m[1]
		}
		if m[0] >= end && m[0] < hi {
			hi = m[0]
		}
	}
	return lo, hi
}

// firstCellSpace is the first space named in a table row's first cell.
func firstCellSpace(line string, matches []spaceMatch) (string, bool) {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "|") {
		return "", false
	}
	open := len(line) - len(t)
	end := strings.Index(line[open+1:], "|")
	if end < 0 {
		return "", false
	}
	for _, m := range matches {
		if m.start > open && m.end <= open+1+end {
			return m.id, true
		}
	}
	return "", false
}

// distinctCandidates keeps one candidate per stored value and renumbers them.
func distinctCandidates(in []Candidate) []Candidate {
	var out []Candidate
	seen := map[string]bool{}
	for _, c := range in {
		if seen[c.Norm] {
			continue
		}
		seen[c.Norm] = true
		c.ID = "c" + strconv.Itoa(len(out)+1)
		out = append(out, c)
	}
	return capped(out)
}
