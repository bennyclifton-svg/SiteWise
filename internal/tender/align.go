package tender

import (
	"sort"
	"strings"
	"unicode"
)

// Slots places each firm's lines of a row into slots shared across the
// firms, so that like lines sit side by side when the row is opened (owner,
// 15 Nov 2026). cols holds each firm's lines in column order. It returns a
// slot for every line a row shows; lines it does not show have none.
//
// A row the manager has arranged (any of its lines has a saved Slot) keeps
// those places, and lines without one go below them. Otherwise the first
// firm keeps its own order and each later firm's line joins the slot whose
// wording shares the most words with it, or takes a new slot below. It is
// plain word matching in Go: no model, no figures, nothing stored.
func Slots(rows []Row, cols [][]Line) map[string]int {
	known := map[string]bool{}
	for _, r := range rows {
		if r.Key != SystemRow {
			known[r.Key] = true
		}
	}
	// byRow[row][firm] is that firm's shown lines of the row, counted first.
	byRow := map[string][][]Line{}
	for f, lines := range cols {
		for _, l := range lines {
			row, ok := shownRow(l, known)
			if !ok {
				continue
			}
			if byRow[row] == nil {
				byRow[row] = make([][]Line, len(cols))
			}
			byRow[row][f] = append(byRow[row][f], l)
		}
	}
	out := map[string]int{}
	for _, firms := range byRow {
		for f := range firms {
			sort.SliceStable(firms[f], func(i, j int) bool { return firms[f][i].Counted && !firms[f][j].Counted })
		}
		if arranged(firms) {
			placeSaved(firms, out)
		} else {
			placeByWords(firms, out)
		}
	}
	return out
}

// shownRow is the row a line is shown under, as the screen lists them:
// subtotals, GST and rates are never shown; a line on no known row is shown
// under "Not broken down" when it is part of the price.
func shownRow(l Line, known map[string]bool) (string, bool) {
	if l.Kind == "subtotal" || l.Kind == "gst" || l.Kind == "rate" {
		return "", false
	}
	if l.Row != "" && known[l.Row] {
		return l.Row, l.Kind != "total"
	}
	if l.Counted || l.Kind == "base" || l.Kind == "ps" || l.Kind == "pc" || (l.Kind == "total" && l.DecidedBy == "user") {
		return SystemRow, true
	}
	return "", false
}

func arranged(firms [][]Line) bool {
	for _, lines := range firms {
		for _, l := range lines {
			if l.Slot != nil {
				return true
			}
		}
	}
	return false
}

// placeSaved keeps the manager's places. A line without one, or one whose
// place its firm already uses, goes below everything, one slot each.
func placeSaved(firms [][]Line, out map[string]int) {
	next := 0
	for _, lines := range firms {
		for _, l := range lines {
			if l.Slot != nil && *l.Slot >= next {
				next = *l.Slot + 1
			}
		}
	}
	for _, lines := range firms {
		used := map[int]bool{}
		var rest []Line
		for _, l := range lines {
			if l.Slot != nil && !used[*l.Slot] {
				used[*l.Slot] = true
				out[l.ID] = *l.Slot
				continue
			}
			rest = append(rest, l)
		}
		for _, l := range rest {
			out[l.ID] = next
			next++
		}
	}
}

type slot struct {
	words map[string]bool
	firms map[int]bool
}

// placeByWords lines up the counted lines by shared words, then puts each
// firm's lines out of the count below them, side by side in order.
func placeByWords(firms [][]Line, out map[string]int) {
	var slots []*slot
	for f, lines := range firms {
		var counted []Line
		for _, l := range lines {
			if l.Counted {
				counted = append(counted, l)
			}
		}
		type pair struct {
			line, slot int
			score      float64
		}
		var pairs []pair
		for i, l := range counted {
			w := words(l.Label)
			for s, sl := range slots {
				if sl.firms[f] {
					continue
				}
				if score := overlap(w, sl.words); score >= 0.2 {
					pairs = append(pairs, pair{i, s, score})
				}
			}
		}
		// Best pairs first; a line and a slot are each taken once.
		sort.SliceStable(pairs, func(a, b int) bool { return pairs[a].score > pairs[b].score })
		lineAt := map[int]int{}
		taken := map[int]bool{}
		for _, p := range pairs {
			if _, ok := lineAt[p.line]; ok || taken[p.slot] {
				continue
			}
			lineAt[p.line], taken[p.slot] = p.slot, true
		}
		for i, l := range counted {
			s, ok := lineAt[i]
			if !ok {
				slots = append(slots, &slot{words: map[string]bool{}, firms: map[int]bool{}})
				s = len(slots) - 1
			}
			slots[s].firms[f] = true
			for w := range words(l.Label) {
				slots[s].words[w] = true
			}
			out[l.ID] = s
		}
	}
	base := len(slots)
	for _, lines := range firms {
		k := 0
		for _, l := range lines {
			if !l.Counted {
				out[l.ID] = base + k
				k++
			}
		}
	}
}

// overlap is the share of words two wordings have in common (Jaccard).
func overlap(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	both := 0
	for w := range a {
		if b[w] {
			both++
		}
	}
	return float64(both) / float64(len(a)+len(b)-both)
}

// slotStopWords say nothing about what a line is for.
var slotStopWords = map[string]bool{"and": true, "the": true, "for": true, "with": true, "from": true, "all": true, "per": true,
	"allow": true, "allowance": true, "provide": true, "provision": true, "supply": true, "install": true, "including": true, "incl": true,
	"works": true, "work": true, "item": true, "items": true, "sum": true, "provisional": true, "nominal": true, "only": true, "as": true}

// words is a wording's telling words: lower case, three letters or more,
// a plural "s" dropped, stop words left out.
func words(label string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(label), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len(w) > 4 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = w[:len(w)-1]
		}
		if len(w) < 3 || slotStopWords[w] {
			continue
		}
		out[w] = true
	}
	return out
}
