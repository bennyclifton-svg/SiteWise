package rfp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	intakevocab "sitewise/data/intake"
	"sitewise/internal/reports"
)

// moneyRE reads the amount in a budget a document states, as Jev picked it:
// "$1.8 million", "$1,800,000", "$450k", "AUD 2.1m".
var moneyRE = regexp.MustCompile(`(?i)(?:\$|aud\s*)\s*(\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?)\s*(million|mil|m|thousand|k)?\b`)

// parseMoney turns a stated amount into whole dollars. Code does the
// arithmetic; Jev only chose which span is the budget.
func parseMoney(text string) (string, bool) {
	m := moneyRE.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
	if err != nil || n <= 0 {
		return "", false
	}
	switch strings.ToLower(m[2]) {
	case "million", "mil", "m":
		n *= 1_000_000
	case "thousand", "k":
		n *= 1_000
	}
	return strconv.FormatInt(int64(n+0.5), 10), true
}

var gstRE = regexp.MustCompile(`(?i)\b(?:excl(?:uding|\.|usive)?|ex\.?)\s*(?:of\s+)?gst\b|\b(?:incl(?:uding|\.|usive)?|inc\.?)\s*(?:of\s+)?gst\b`)

// statedBasis says where a stated budget came from and its GST basis when
// the passage gives one.
func statedBasis(text string) string {
	basis := "as stated in the project documents; not yet in the cost plan"
	if m := gstRE.FindString(text); m != "" {
		if strings.HasPrefix(strings.ToLower(m), "in") {
			return "incl. GST, " + basis
		}
		return "excl. GST, " + basis
	}
	return basis
}

var vocab struct {
	sync.Once
	kinds, disciplines map[string]string
}

func vocabLabels() (map[string]string, map[string]string) {
	vocab.Do(func() {
		var k struct {
			Kinds []struct{ ID, Label string } `json:"kinds"`
		}
		var d struct {
			Disciplines []struct{ ID, Label string } `json:"disciplines"`
		}
		vocab.kinds, vocab.disciplines = map[string]string{}, map[string]string{}
		if json.Unmarshal(intakevocab.Kinds, &k) == nil {
			for _, x := range k.Kinds {
				vocab.kinds[x.ID] = x.Label
			}
		}
		if json.Unmarshal(intakevocab.Disciplines, &d) == nil {
			for _, x := range d.Disciplines {
				vocab.disciplines[x.ID] = x.Label
			}
		}
	})
	return vocab.kinds, vocab.disciplines
}

// registerSummary lists what the project register holds, grouped by the
// discipline and kind filing gave each current document, so "Information
// provided" follows the documents without anyone typing it.
func registerSummary(docs []reports.BriefValue) string {
	kinds, disciplines := vocabLabels()
	counts := map[string]int{}
	for _, doc := range docs {
		var basis struct {
			Sources []struct {
				Kind       string `json:"kind"`
				Discipline string `json:"discipline"`
			} `json:"sources"`
		}
		if json.Unmarshal(doc.Basis, &basis) != nil || len(basis.Sources) == 0 {
			continue
		}
		src := basis.Sources[0]
		kind := kinds[src.Kind]
		if kind == "" || src.Kind == "unknown" {
			kind = "Other document"
		}
		label := kind
		if d := disciplines[src.Discipline]; d != "" {
			label = d + " " + strings.ToLower(kind)
		}
		counts[label]++
	}
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	parts := []string{}
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("%s (%d)", label, counts[label]))
	}
	return strings.Join(parts, "; ")
}
