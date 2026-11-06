package profile

import (
	"regexp"
	"strings"

	"sitewise/internal/knowledge"
)

// impliedPrefix marks a code fact: a word a document uses for the works that
// implies a system (scope_defaults.yaml implied_by). It suggests scope and is
// never presence evidence.
const impliedPrefix = "implied."

// ImpliedReadings finds, in code, the systems a passage's words imply. Each
// reading carries the matched sentence as its excerpt and the term as its
// basis. No model call: this is a lookup.
func ImpliedReadings(text string, cat *knowledge.Catalog) []Reading {
	if cat == nil {
		return nil
	}
	var out []Reading
	seen := map[string]bool{}
	for _, term := range cat.ScopeTerms() {
		excerpt, ok := affirmedMatch(text, term)
		if !ok {
			continue
		}
		for _, leaf := range term.Systems {
			if seen[leaf] {
				continue
			}
			seen[leaf] = true
			out = append(out, Reading{QuestionID: impliedPrefix + leaf, Value: valIncluded, Basis: term.Label, Excerpt: excerpt})
		}
	}
	return out
}

// affirmedMatch is the excerpt of the term's first match whose sentence does
// not deny the work (F45: "the core bathrooms are not in scope" implies no
// wet-area works).
func affirmedMatch(text string, term knowledge.ScopeTerm) (string, bool) {
	for _, at := range term.FindAll(text) {
		lo, hi := sentenceBounds(text, at[0], at[1])
		if !deniesWork(text[lo:hi]) {
			return sentenceAround(text, at[0], at[1]), true
		}
	}
	return "", false
}

var (
	// notLimited is boilerplate that affirms a list ("including but not
	// limited to"), so its "not" denies nothing.
	notLimited = regexp.MustCompile(`(?i)\bnot\s+limited\s+to\b`)
	// denial covers the forms the documents use to exclude work: a "not"
	// or "no" clause, an exclusion, work outside scope or by others. A
	// sentence holding one is treated as denying every term in it; a missed
	// suggestion costs one click, a wrong one adds works to the program.
	denial = regexp.MustCompile(`(?i)\b(?:not|isn't|aren't|won't|nil)\b|^\W*no\b|\bno\b.{0,60}\b(?:required|proposed|planned|included|allowed|works?)\b|\bexclu(?:ded|des|ding|sions?)\b|\boutside\s+(?:the\s+)?scope\b|\bby\s+others\b`)
)

// deniesWork reports whether a sentence excludes the work it names.
func deniesWork(sentence string) bool {
	return denial.MatchString(notLimited.ReplaceAllString(sentence, ""))
}

// sentenceBounds is the sentence or line holding text[start:end].
func sentenceBounds(text string, start, end int) (int, int) {
	lo := strings.LastIndexAny(text[:start], ".\n;") + 1
	hi := strings.IndexAny(text[end:], ".\n;")
	if hi < 0 {
		return lo, len(text)
	}
	return lo, hi + end
}

// sentenceAround is the sentence or line holding text[start:end], at most
// 120 characters (the stored excerpt limit), centred on the match.
func sentenceAround(text string, start, end int) string {
	lo, hi := sentenceBounds(text, start, end)
	s := strings.Join(strings.Fields(strings.Trim(text[lo:hi], "*_-•# ")), " ")
	if len(s) <= 120 {
		return s
	}
	word := strings.Join(strings.Fields(text[start:end]), " ")
	i := strings.Index(s, word)
	from := max(0, i-50)
	for from > 0 && !utf8Start(s[from]) {
		from--
	}
	to := min(len(s), from+117)
	for to < len(s) && !utf8Start(s[to]) {
		to--
	}
	return s[from:to] + "…"
}

// splitImplied separates implied-scope facts from evidence, as leaf → the
// first excerpt, so they never reach reconciliation as rows.
func splitImplied(facts []Fact) ([]Fact, map[string]string) {
	implied := map[string]string{}
	out := facts[:0:0]
	for _, f := range facts {
		if leaf, ok := strings.CutPrefix(f.QuestionID, impliedPrefix); ok {
			if _, seen := implied[leaf]; !seen && f.Value == valIncluded {
				implied[leaf] = f.Excerpt
			}
			continue
		}
		out = append(out, f)
	}
	return out, implied
}
