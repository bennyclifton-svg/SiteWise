package costs

import (
	"math/big"
	"regexp"
	"sort"
	"strings"
)

// StatedBudget is the construction budget as the profile holds it (a
// document's words or the user's), with the single amount Go reads from it.
// Amount is ex GST: an inclusive figure is converted at 10 %, and a figure
// with no stated basis is taken as written, since the plan is ex GST.
type StatedBudget struct {
	Text      string `json:"text"`
	Amount    Money  `json:"amount"`
	GST       string `json:"gst"` // "ex", "inc" or "not_stated"
	Converted bool   `json:"converted"`
}

var (
	statedAmount = regexp.MustCompile(`(?i)\$\s*([0-9][0-9,]*(?:\.[0-9]+)?)\s*(million|mil|m|thousand|k)?\b`)
	statedRange  = regexp.MustCompile(`(?i)\$\s*[0-9][0-9,.]*\s*(?:million|m|k)?\s*(?:-|–|—|to)\s*\$?\s*[0-9]`)
	gstInc       = regexp.MustCompile(`(?i)\b(?:incl(?:uding|usive|\.)?|inc\.?)\s*(?:of\s*)?(?:gst|tax)\b`)
	gstEx        = regexp.MustCompile(`(?i)\b(?:excl(?:uding|usive|\.)?|ex\.?|plus|\+)\s*(?:of\s*)?(?:gst|tax)\b`)
)

// ParseStatedBudget reads one amount from a stated budget. A range, or text
// with no amount or more than one, gives no amount: unknown stays unknown.
func ParseStatedBudget(text string) (StatedBudget, bool) {
	out := StatedBudget{Text: strings.TrimSpace(text), GST: "not_stated"}
	if statedRange.MatchString(text) {
		return out, false
	}
	found := statedAmount.FindAllStringSubmatch(text, -1)
	if len(found) != 1 {
		return out, false
	}
	n, ok := new(big.Rat).SetString(strings.ReplaceAll(found[0][1], ",", ""))
	if !ok || n.Sign() <= 0 {
		return out, false
	}
	switch strings.ToLower(found[0][2]) {
	case "million", "mil", "m":
		n.Mul(n, big.NewRat(1_000_000, 1))
	case "thousand", "k":
		n.Mul(n, big.NewRat(1_000, 1))
	}
	switch {
	case gstInc.MatchString(text):
		out.GST, out.Converted = "inc", true
		n.Quo(n, big.NewRat(11, 10))
	case gstEx.MatchString(text):
		out.GST = "ex"
	}
	out.Amount = Round(n)
	return out, true
}

// SpreadLine is one line the budget is spread over, with its weight.
type SpreadLine struct {
	ID     string
	Weight *big.Rat
}

// Spread divides total over the lines in proportion to their weights,
// rounded to the cent. The rounding residual goes to the heaviest line
// (first in order on a tie), so the parts always add to the total.
func Spread(total Money, lines []SpreadLine) map[string]Money {
	t, err := Parse(string(total))
	if err != nil {
		return nil
	}
	sum := new(big.Rat)
	for _, l := range lines {
		sum.Add(sum, l.Weight)
	}
	if sum.Sign() <= 0 {
		return nil
	}
	out := map[string]Money{}
	allocated := Money("0.00")
	heaviest := -1
	for n, l := range lines {
		share := new(big.Rat).Quo(new(big.Rat).Mul(t, l.Weight), sum)
		out[l.ID] = Round(share)
		allocated = Add(allocated, out[l.ID])
		if heaviest < 0 || l.Weight.Cmp(lines[heaviest].Weight) > 0 {
			heaviest = n
		}
	}
	if residual := Sub(total, allocated); Compare(residual, "0.00") != 0 {
		id := lines[heaviest].ID
		out[id] = Add(out[id], residual)
	}
	return out
}

// SortedSystems lists a weight map's keys, so a spread is deterministic.
func SortedSystems(w map[string]*big.Rat) []string {
	keys := make([]string, 0, len(w))
	for k := range w {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ConstructionBudget is the profile's construction budget and the split the
// cost plan spreads it by.
type ConstructionBudget struct {
	Stated StatedBudget `json:"stated"`
	// Known is false when no single amount could be read from the words.
	Known bool `json:"known"`
	// Origin is "document" (a reading) or "user" (typed).
	Origin string `json:"origin"`
	// Split names the split chosen from the profile, or why there is none.
	Split           string `json:"split"`
	SplitConfidence string `json:"split_confidence"`
	SplitBasis      string `json:"split_basis"`
	NoSplit         string `json:"no_split,omitempty"`
}

// BudgetCheck is the budget against what the construction lines hold.
// Allocated is the known subtotal; Difference is budget less allocated, so
// a positive figure is still to allocate.
type BudgetCheck struct {
	Amount     *Money `json:"amount"`
	Allocated  Money  `json:"allocated"`
	Unpriced   int    `json:"unpriced"`
	Difference *Money `json:"difference"`
}

func checkBudget(b *ConstructionBudget, lines Total) *BudgetCheck {
	if b == nil {
		return nil
	}
	allocated := lines.KnownSubtotal
	if allocated == "" {
		allocated = "0.00"
	}
	out := &BudgetCheck{Allocated: allocated, Unpriced: lines.Unknown}
	if b.Known {
		a := b.Stated.Amount
		d := Sub(a, allocated)
		out.Amount, out.Difference = &a, &d
	}
	return out
}
