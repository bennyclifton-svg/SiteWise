package proposals

import (
	"regexp"
	"strings"

	"sitewise/internal/invoices"
)

// An Offer is a firm's quoted fees read as the lines of its appointment:
// the fee lines that make up the offer, and the figures that are not fee
// lines at all. It is computed from a stored QuoteReview when the
// comparison is read, so no Jev call and no re-reading is involved.
//
// Lines add to a printed total to the cent when Reconciled. Without a
// printed total, Lines are the current (not optional) fees as listed,
// and the total is their sum. Nothing is allocated, split or guessed.
type Offer struct {
	Lines []OfferLine
	// TotalID is the printed total the lines add to, or the one printed
	// total when Single. Empty when the letter prints no total.
	TotalID    string
	Reconciled bool
	// Single is one printed total with no breakdown that adds to it.
	Single bool
	// Hidden holds figures that are not fee lines: insurance cover, clause
	// amounts, rate ranges, invoicing splits, repeats and restated totals.
	Hidden map[string]string
}

type OfferLine struct {
	Fee   QuotedFee
	Label string
	// Amount is ex GST. When the line's own basis is unknown it is taken
	// from an ex-GST total it adds to exactly; otherwise it stays nil.
	Amount *string
}

const (
	hiddenInsurance = "insurance cover"
	hiddenClause    = "contract clause amount"
	hiddenRange     = "rate range"
	hiddenInvoicing = "invoicing split of the fee"
	hiddenRepeat    = "repeated in the letter"
	hiddenTotal     = "restates the total"
	hiddenTotalInc  = "total including GST"
)

var (
	insuranceWords = regexp.MustCompile(`(?i)\b(?:insurances?|insurer|indemnity|public liability|policy (?:no|number)|workers'? comp|certificates? of currency|icare)\b`)
	clauseWords    = regexp.MustCompile(`(?i)\b(?:amount claimed|court proceedings|expert determination|dispute resolution|liquidated damages|limit(?:ation)? of liability)\b`)
	rateRange      = regexp.MustCompile(`\$\s?\d[\d,]*(?:\.\d+)?\s*[-–—]\s*\$\s?\d`)
	invoicingSplit = regexp.MustCompile(`\(\s*\d{1,3}(?:\.\d+)?\s*%\s*\)`)
	invoicingHead  = regexp.MustCompile(`(?i)invoic|payment schedule|schedule of payments|progress (?:claim|payment)`)
	printedFigure  = regexp.MustCompile(`\d[\d,]*(?:\.\d{1,2})?`)
	stageID        = regexp.MustCompile(`(?i)^(?:phase|stage|task|part|step|item)\s*[0-9][0-9a-z.]*\s*[-–:]?$`)
	bulletLead     = regexp.MustCompile(`^[^\p{L}\p{N}(]+`)
	cellBreak      = regexp.MustCompile(`\s{2,}`)
	labelKey       = regexp.MustCompile(`[^a-z0-9]+`)
)

// ReadOffer reads the offer from quoted fees in document order. stated is
// the firm's stated ex-GST total when the reader settled one.
func ReadOffer(fees []QuotedFee, stated *int64) Offer {
	o := Offer{Hidden: map[string]string{}}
	cents := make([]int64, len(fees))
	known := make([]bool, len(fees))
	for i, f := range fees {
		cents[i], known[i] = printedCents(f)
	}
	for i, f := range fees {
		if reason := notAFee(f); reason != "" {
			o.Hidden[f.ID] = reason
		} else if strings.HasPrefix(f.Role, "Repeated") || strings.HasPrefix(f.Suggestion, "Earlier") {
			o.Hidden[f.ID] = hiddenRepeat
		} else if !known[i] {
			o.Hidden[f.ID] = "no amount"
		}
	}
	// A figure equal to an insurance cover is the same cover printed again
	// (a certificate of currency), not a fee.
	for i, f := range fees {
		if o.Hidden[f.ID] != "" {
			continue
		}
		for j, g := range fees {
			if o.Hidden[g.ID] == hiddenInsurance && known[j] && cents[j] == cents[i] {
				o.Hidden[f.ID] = hiddenInsurance
				break
			}
		}
	}
	live := func(i int) bool { return o.Hidden[fees[i].ID] == "" }

	type group struct {
		total int
		comps []int
		score int
	}
	var groups []group
	for t := range fees {
		if !live(t) || optional(fees[t]) {
			continue
		}
		explicit := isTotal(fees[t])
		var sum int64
		var comps []int
		for j := t - 1; j >= 0; j-- {
			if !live(j) || optional(fees[j]) {
				continue
			}
			if isTotal(fees[j]) || fees[j].Page < fees[t].Page-1 {
				break
			}
			sum += cents[j]
			comps = append([]int{j}, comps...)
			if sum == cents[t] && (explicit || len(comps) >= 2) {
				g := group{total: t, comps: comps}
				if explicit {
					g.score += 4
				}
				if stated != nil && *stated == cents[t] {
					g.score += 2
				}
				good := true
				for _, c := range comps {
					good = good && goodLabel(fees[c])
				}
				if good {
					g.score++
				}
				groups = append(groups, g)
				break
			}
			if sum > cents[t] {
				break
			}
		}
	}
	best := -1
	for i, g := range groups {
		if best < 0 || g.score > groups[best].score || g.score == groups[best].score && cents[g.total] > cents[groups[best].total] {
			best = i
		}
	}

	inLine := map[int]bool{}
	var total int64
	switch {
	case best >= 0:
		g := groups[best]
		t := fees[g.total]
		o.TotalID, o.Reconciled, total = t.ID, true, cents[g.total]
		// Components inherit an ex-GST basis only from a total printed ex GST.
		totalEx := false
		if t.Amount != nil {
			c, ok, err := invoices.ParseCents(*t.Amount)
			totalEx = ok && err == nil && c == cents[g.total]
		}
		for _, c := range g.comps {
			inLine[c] = true
			line := OfferLine{Fee: fees[c], Label: CleanFeeLabel(fees[c]), Amount: fees[c].Amount}
			if line.Amount == nil && totalEx {
				v := invoices.FormatCents(cents[c])
				line.Amount = &v
			}
			o.Lines = append(o.Lines, line)
		}
		// Another breakdown with the same amounts is the same offer printed
		// again (a schedule repeated in an appendix or agreement).
		for i, other := range groups {
			if i == best || !sameAmounts(cents, other.comps, g.comps) {
				continue
			}
			for _, c := range other.comps {
				o.Hidden[fees[c].ID] = hiddenRepeat
			}
		}
	default:
		var totals []int
		for i := range fees {
			if live(i) && !optional(fees[i]) && isTotal(fees[i]) {
				totals = append(totals, i)
			}
		}
		if len(totals) > 0 && distinctAmounts(cents, totals) == 1 {
			o.Single, o.TotalID, total = true, fees[totals[0]].ID, cents[totals[0]]
			break
		}
		for i := range fees {
			if live(i) && !optional(fees[i]) && !isTotal(fees[i]) && !repeats(fees, cents, i, inLine) {
				inLine[i] = true
				o.Lines = append(o.Lines, OfferLine{Fee: fees[i], Label: CleanFeeLabel(fees[i]), Amount: fees[i].Amount})
			}
		}
	}
	for i, f := range fees {
		if !live(i) || inLine[i] || f.ID == o.TotalID {
			continue
		}
		switch {
		case total > 0 && cents[i] == total && !optional(f):
			o.Hidden[f.ID] = hiddenTotal
		case total > 0 && abs64(cents[i]*10-total*11) <= 10:
			o.Hidden[f.ID] = hiddenTotalInc
		case repeats(fees, cents, i, inLine):
			o.Hidden[f.ID] = hiddenRepeat
		}
	}
	return o
}

// notAFee is why a figure printed in a fee letter is not a fee.
func notAFee(f QuotedFee) string {
	text := f.Source + " " + f.Heading + " " + f.Label
	switch {
	case insuranceWords.MatchString(text):
		return hiddenInsurance
	case clauseWords.MatchString(f.Source):
		return hiddenClause
	case rateRange.MatchString(f.Source):
		return hiddenRange
	case invoicingSplit.MatchString(f.Source) || invoicingHead.MatchString(f.Heading):
		return hiddenInvoicing
	}
	return ""
}

func optional(f QuotedFee) bool {
	return strings.HasPrefix(f.Role, "Conditional") || strings.Contains(f.Role, "Alternative") ||
		strings.HasPrefix(f.Suggestion, "Conditional") || strings.HasPrefix(f.Suggestion, "Alternative")
}

func isTotal(f QuotedFee) bool {
	return f.Role == RoleTotal || quoteSummaryWords.MatchString(f.Label)
}

// repeats is a fee whose label and amount match a fee already listed.
func repeats(fees []QuotedFee, cents []int64, i int, listed map[int]bool) bool {
	key := func(f QuotedFee) string { return labelKey.ReplaceAllString(strings.ToLower(f.Label), "") }
	for j := range listed {
		if j != i && cents[j] == cents[i] && key(fees[j]) == key(fees[i]) {
			return true
		}
	}
	return false
}

func sameAmounts(cents []int64, a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	count := map[int64]int{}
	for _, i := range a {
		count[cents[i]]++
	}
	for _, i := range b {
		count[cents[i]]--
	}
	for _, n := range count {
		if n != 0 {
			return false
		}
	}
	return true
}

func distinctAmounts(cents []int64, idx []int) int {
	seen := map[int64]bool{}
	for _, i := range idx {
		seen[cents[i]] = true
	}
	return len(seen)
}

// printedCents is the figure as printed, before any GST conversion, so
// lines and totals are compared on the letter's own numbers.
func printedCents(f QuotedFee) (int64, bool) {
	if m := printedFigure.FindString(f.Printed); m != "" {
		if c, ok, err := invoices.ParseCents(m); ok && err == nil && c > 0 {
			return c, true
		}
	}
	if f.Amount != nil {
		if c, ok, err := invoices.ParseCents(*f.Amount); ok && err == nil && c > 0 {
			return c, true
		}
	}
	return 0, false
}

// goodLabel is a label that names the work: not the section heading, a
// fragment of a sentence or a number.
func goodLabel(f QuotedFee) bool {
	l := strings.TrimSpace(f.Label)
	if strings.Contains(l, "▪") || len(l) < 4 || len(strings.Fields(l)) > 12 || fragment(l) {
		return false
	}
	return !strings.EqualFold(l, strings.TrimSpace(f.Heading)) && !strings.EqualFold(l, "Quoted fee") && strings.Trim(l, "0123456789-., ") != ""
}

// CleanFeeLabel names a fee line from its printed row. A bulleted scope
// sentence keeps the row's phase or task number in front; a fragment
// ("(SEE) for client review.") gives way to the row's first cell.
func CleanFeeLabel(f QuotedFee) string {
	label := strings.TrimSpace(bulletLead.ReplaceAllString(f.Label, ""))
	cells := cellBreak.Split(strings.TrimSpace(f.Source), -1)
	first := ""
	if len(cells) > 1 {
		first = strings.TrimSpace(cells[0])
	}
	if first != "" && !strings.ContainsAny(first, "$") {
		trimmed := strings.TrimSpace(strings.TrimRight(first, "-–: "))
		switch {
		case stageID.MatchString(first) && label != "" && !strings.EqualFold(trimmed, label):
			return trimmed + " – " + label
		case fragment(label) && len(strings.Fields(first)) <= 8 && !fragment(first):
			return trimmed
		}
	}
	// A sentence fragment with no row cells reads better as its sentence:
	// "suggest a" becomes "suggest a $2,500 allowance for the same."
	if fragment(label) && len(cells) == 1 && cells[0] != "" && len(cells[0]) <= 120 {
		return cells[0]
	}
	if label == "" {
		return "Quoted fee"
	}
	return label
}

func fragment(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	r := []rune(s)[0]
	return r == '(' || r >= 'a' && r <= 'z' || strings.HasPrefix(s, "▪")
}
