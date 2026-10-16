package proposals

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/invoices"
)

// FirmFacts are one firm's computed figures and the PM's ratings: all a
// drafted recommendation may state.
type FirmFacts struct {
	Name      string
	BaseTotal *int64
	KnownBase int64
	NotPriced int
	Unread    int
	Lowest    bool
	Gap       *int64
	Ratings   map[string]string // criterion label → strong | acceptable | concern
}

// DraftRecommendation is the code's factual recommendation from the price
// table and the PM's ratings. It states figures and ratings only; it does
// not choose a firm. The PM edits it and signs it off.
func DraftRecommendation(pkg string, firms []FirmFacts) string {
	if len(firms) == 0 {
		return ""
	}
	var lines []string
	var lowest *FirmFacts
	for i := range firms {
		if firms[i].Lowest {
			lowest = &firms[i]
		}
	}
	if lowest != nil {
		lines = append(lines, "Lowest complete base fee: "+lowest.Name+" at $"+dollars(*lowest.BaseTotal)+" ex GST.")
	} else {
		lines = append(lines, "No firm has priced every base item, so no price is the lowest complete one yet.")
	}
	sorted := append([]FirmFacts(nil), firms...)
	sort.SliceStable(sorted, func(i, j int) bool { return figure(sorted[i]) < figure(sorted[j]) })
	for _, f := range sorted {
		if lowest != nil && f.Name == lowest.Name {
			continue
		}
		s := f.Name + ": "
		if f.BaseTotal != nil {
			s += "$" + dollars(*f.BaseTotal) + " ex GST"
		} else {
			s += "$" + dollars(f.KnownBase) + " ex GST known so far"
		}
		var notes []string
		if f.NotPriced > 0 {
			notes = append(notes, plural(f.NotPriced, "item")+" not priced")
		}
		if f.Unread > 0 {
			notes = append(notes, plural(f.Unread, "item")+" still to read")
		}
		if f.Gap != nil {
			notes = append(notes, "its stated total differs by $"+dollars(abs(*f.Gap)))
		}
		if len(notes) > 0 {
			s += " (" + strings.Join(notes, "; ") + ")"
		}
		lines = append(lines, s+".")
	}
	var concerns []string
	for _, f := range firms {
		var crit []string
		for c, r := range f.Ratings {
			if r == "concern" {
				crit = append(crit, c)
			}
		}
		sort.Strings(crit)
		if len(crit) > 0 {
			concerns = append(concerns, f.Name+" ("+strings.Join(crit, ", ")+")")
		}
	}
	if len(concerns) > 0 {
		lines = append(lines, "Rated a concern: "+strings.Join(concerns, "; ")+".")
	}
	lines = append(lines, "Recommendation for "+pkg+": [state the firm to appoint and why].")
	return strings.Join(lines, "\n")
}

func figure(f FirmFacts) int64 {
	if f.BaseTotal != nil {
		return *f.BaseTotal
	}
	return 1 << 62
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// dollars prints cents as whole dollars when there are no cents.
func dollars(c int64) string {
	return strings.TrimSuffix(invoices.DisplayCents(c), ".00")
}

// GeneratedDraft is what a generative model would return for a report:
// short summaries and departures that cite passage ids, and a
// recommendation. No model is wired (AGENTS.md non-negotiable #1); this is
// the contract a draft must meet before any of it is shown.
type GeneratedDraft struct {
	Summaries      []GeneratedText `json:"summaries"`
	Departures     []GeneratedText `json:"departures"`
	Recommendation string          `json:"recommendation"`
}

// GeneratedText is one firm's text on one criterion, with its citations.
type GeneratedText struct {
	Firm      string   `json:"firm"`
	Criterion string   `json:"criterion"`
	Text      string   `json:"text"`
	Cites     []string `json:"cites"`
}

// DraftProblem is one part of a draft refused, and why.
type DraftProblem struct {
	Firm, Criterion, Reason string
}

var numberToken = regexp.MustCompile(`\$?\d[\d,]*(?:\.\d+)?%?`)

// CheckDraft keeps only the parts of a generated draft that meet the
// contract: every cited id exists, every summary cites something, a summary
// is at most 25 words and the recommendation at most 120, and every number
// in the text is a computed figure or appears in a cited passage. Anything
// else is dropped and reported, never shown.
func CheckDraft(d GeneratedDraft, passages map[string]string, figures []int64) (GeneratedDraft, []DraftProblem) {
	known := map[string]bool{}
	for _, f := range figures {
		for _, s := range []string{strconv.FormatInt(f/100, 10), strings.ReplaceAll(invoices.DisplayCents(f), ",", ""), strings.ReplaceAll(dollars(f), ",", "")} {
			known[s] = true
		}
	}
	var problems []DraftProblem
	checkText := func(t GeneratedText, maxWords int) (bool, string) {
		if len(t.Cites) == 0 {
			return false, "cites nothing"
		}
		var cited strings.Builder
		for _, id := range t.Cites {
			p, ok := passages[id]
			if !ok {
				return false, "cites " + id + ", which does not exist"
			}
			cited.WriteString(strings.ReplaceAll(p, ",", ""))
			cited.WriteByte('\n')
		}
		if len(strings.Fields(t.Text)) > maxWords {
			return false, "is longer than " + strconv.Itoa(maxWords) + " words"
		}
		for _, n := range numberToken.FindAllString(t.Text, -1) {
			bare := strings.Trim(strings.ReplaceAll(strings.TrimPrefix(n, "$"), ",", ""), ".")
			if !known[bare] && !strings.Contains(cited.String(), bare) {
				return false, "states " + n + ", which is neither computed nor in a cited passage"
			}
		}
		return true, ""
	}
	var out GeneratedDraft
	for _, t := range d.Summaries {
		if ok, why := checkText(t, 25); ok {
			out.Summaries = append(out.Summaries, t)
		} else {
			problems = append(problems, DraftProblem{t.Firm, t.Criterion, "summary " + why})
		}
	}
	for _, t := range d.Departures {
		if ok, why := checkText(t, 60); ok {
			out.Departures = append(out.Departures, t)
		} else {
			problems = append(problems, DraftProblem{t.Firm, "departures", why})
		}
	}
	rec := GeneratedText{Text: d.Recommendation, Cites: []string{"_figures"}}
	if len(strings.Fields(d.Recommendation)) > 120 {
		problems = append(problems, DraftProblem{"", "recommendation", "is longer than 120 words"})
	} else {
		// The recommendation may state only computed figures.
		ok := true
		for _, n := range numberToken.FindAllString(rec.Text, -1) {
			bare := strings.Trim(strings.ReplaceAll(strings.TrimPrefix(n, "$"), ",", ""), ".")
			if !known[bare] {
				ok = false
				problems = append(problems, DraftProblem{"", "recommendation", "states " + n + ", which is not a computed figure"})
				break
			}
		}
		if ok {
			out.Recommendation = d.Recommendation
		}
	}
	return out, problems
}
