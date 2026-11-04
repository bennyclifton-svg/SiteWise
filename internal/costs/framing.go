package costs

import "regexp"

// BudgetFraming is what the words around a stated budget or contingency say
// about it, read in code from the source excerpt (Walsh brief, 2026-10-27:
// "$920,000 (including contingency for latent conditions…)"; "Owners hold
// $85,000 additional contingency"). It never sets an amount: a contingency
// included in a budget stays unquantified when the document does not say.
type BudgetFraming struct {
	// IncludesContingency: the budget's own words say it includes a contingency.
	IncludesContingency bool `json:"includes_contingency,omitempty"`
	// Additional: a contingency's words say it is held beyond the budget.
	Additional bool `json:"additional,omitempty"`
	// GST is "ex", "inc" or "not_stated", from the excerpt.
	GST string `json:"gst"`
}

var (
	includesContingency = regexp.MustCompile(`(?i)\b(?:includ(?:es|ing|ed)|incl\.?|inclusive of|allows? for)\b[^.;)]{0,40}\bcontingenc`)
	additionalHeld      = regexp.MustCompile(`(?i)\b(?:additional|in addition|on top of|over and above|separate(?:ly)?|outside (?:the|of the) (?:construction )?budget)\b`)
)

// PreliminaryEstimate is a builder's or estimator's preliminary cost advice
// as a document states it (Walsh, 2026-10-27: "$880,000 – $980,000 ex
// GST"). It is shown beside the budget; it is never a budget or a price.
type PreliminaryEstimate struct {
	Text string `json:"text"`
	// Low and High are the range in dollars as printed; equal for one figure.
	Low  Money  `json:"low"`
	High Money  `json:"high"`
	GST  string `json:"gst"` // "ex", "inc" or "not_stated", as printed
	// Source is the excerpt the estimate was read from.
	Source string `json:"source,omitempty"`
}

// ReadEstimate reads one or two dollar amounts from the stated words; ok is
// false when there are none or more than two.
func ReadEstimate(text, excerpt string) (PreliminaryEstimate, bool) {
	e := PreliminaryEstimate{Text: text, Source: excerpt, GST: ReadBudgetFraming(text + " " + excerpt).GST}
	found := statedAmount.FindAllStringSubmatch(text, -1)
	if len(found) == 0 || len(found) > 2 {
		return e, false
	}
	var amounts []Money
	for _, f := range found {
		one, ok := ParseStatedBudget(f[0])
		if !ok {
			return e, false
		}
		amounts = append(amounts, one.Amount)
	}
	e.Low, e.High = amounts[0], amounts[len(amounts)-1]
	return e, true
}

// ReadBudgetFraming reads an excerpt's framing words.
func ReadBudgetFraming(excerpt string) BudgetFraming {
	f := BudgetFraming{GST: "not_stated"}
	f.IncludesContingency = includesContingency.MatchString(excerpt)
	f.Additional = additionalHeld.MatchString(excerpt)
	switch {
	case gstInc.MatchString(excerpt):
		f.GST = "inc"
	case gstEx.MatchString(excerpt):
		f.GST = "ex"
	}
	return f
}
