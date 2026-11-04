package profile

import (
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
		at := term.Find(text)
		if at == nil {
			continue
		}
		excerpt := sentenceAround(text, at[0], at[1])
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

// sentenceAround is the sentence or line holding text[start:end], at most
// 120 characters (the stored excerpt limit), centred on the match.
func sentenceAround(text string, start, end int) string {
	lo := strings.LastIndexAny(text[:start], ".\n;") + 1
	hi := strings.IndexAny(text[end:], ".\n;")
	if hi < 0 {
		hi = len(text)
	} else {
		hi += end
	}
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
