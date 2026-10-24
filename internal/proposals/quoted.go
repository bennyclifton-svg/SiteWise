package proposals

import (
	"regexp"
	"strconv"
	"strings"

	"sitewise/internal/jev"
)

// QuoteReview preserves the commercial offer independently of the requested
// schedule. These are source readings, never additive award instructions.
type QuoteReview struct {
	Fees     []QuotedFee              `json:"fees"`
	Coverage map[string]QuoteCoverage `json:"coverage"`
	Issues   []string                 `json:"issues"`
}

type QuotedFee struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Printed    string  `json:"printed"`
	Amount     *string `json:"amount"` // ex GST; nil when tax basis is unknown
	Role       string  `json:"role"`
	Page       int     `json:"page"`
	Source     string  `json:"source"`
	Heading    string  `json:"heading"`
	Suggestion string  `json:"suggestion,omitempty"`
}

type QuoteCoverage struct {
	FeeID  string `json:"fee_id"`
	Status string `json:"status"`
}

var quoteSummaryWords = regexp.MustCompile(`(?i)lump\s*sum|total|sub\s*total|fee estimate|cost estimate|quotation|fixed fee`)
var quoteConditionalWords = regexp.MustCompile(`(?i)optional|if (?:needed|required)|additional cost|surcharge|cancellation|less than .*business days`)
var nonName = regexp.MustCompile(`[^a-z0-9]`)

func compactName(s string) string { return nonName.ReplaceAllString(strings.ToLower(s), "") }

var firmSentenceNoise = regexp.MustCompile(`(?i)^(?:please return to|trading name)\s+|(?:\s+reviewed by)\s*$`)
var firmProse = regexp.MustCompile(`(?i)\b(?:unless|shall|reimburse|capability|agrees|hereby|liable|engage|engaged)\b`)

func cleanFirmCandidate(s string) string {
	s = strings.TrimSpace(firmSentenceNoise.ReplaceAllString(stripEmphasis(s), ""))
	if firmProse.MatchString(s) {
		return ""
	}
	return s
}

// Quote candidates retain equal amounts on different source rows. The old
// leftover list discarded them by monetary value, losing real priced work.
func quoteAmounts(ev Evidence) []Amount {
	var out []Amount
	seen := map[string]bool{}
	for _, a := range ev.Amounts {
		if a.Unit || a.GSTPart || a.IncTwin || a.PerHour || a.Kind != "" || unitPrice(a.Context) || strings.Contains(strings.ToLower(a.Context), "rate only") {
			continue
		}
		key := strconv.Itoa(a.Page) + "|" + strconv.Itoa(a.Line) + "|" + strconv.FormatInt(a.Cents, 10) + "|" + a.Context
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, a)
	}
	return out
}

// Scope questions select a source bundle, not an allocation of its price.
// They share the existing fan-out. New question shapes are explicitly
// uncalibrated suggestions; they cannot populate prices or award amounts.
// https://docs.typesafe.ai/patterns/fan-out
// https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
func coverageAmounts(ev Evidence) []Amount {
	var fees []Amount
	for _, a := range quoteAmounts(ev) {
		if !unitWord.MatchString(a.Context) || len(moneyIn(a.Context)) < 2 || quoteSummaryWords.MatchString(a.Label) {
			fees = append(fees, a)
		}
	}
	return limit(fees, 35)
}

func (d *Draft) askCoverage(add func(question)) {
	fees := coverageAmounts(d.ev)
	for _, fee := range fees {
		add(question{Field: "offer-role:" + fee.ID, Family: "coverage-1",
			Instructions: roleQuestion{Question: "What role does quoted_fees " + fee.ID + " play in this offer? Use explicit wording, including any email saying these are revised costs.", NotFor: "Do not compare dates or calculate. If the wording does not establish the role, choose unclear."},
			Options: []choice{
				{ID: "current", Value: "Current offer", Describe: "A current quoted fee or package."},
				{ID: "alternative", Value: "Alternative — choose a scope", Describe: "An alternative replacing another quoted scope or fee."},
				{ID: "conditional", Value: "Conditional / optional", Describe: "A conditional charge or optional additional service."},
				{ID: "earlier", Value: "Earlier fee — revised elsewhere in this source", Describe: "An older fee explicitly replaced by revised costs in this source."},
				{ID: "unclear", Value: "", Describe: "The relationship is unclear."},
			}})
	}
	for _, it := range d.items {
		if !it.Price() {
			continue
		}
		opts := []choice{}
		for _, fee := range fees {
			opts = append(opts, choice{ID: fee.ID, Value: fee.ID, Describe: fee.ID + " in quoted_fees"})
		}
		opts = append(opts,
			choice{ID: "unstated", Value: "unstated", Describe: "Coverage is not stated clearly."},
			choice{ID: "excluded", Value: "excluded", Describe: "The proposal explicitly excludes this service."})
		add(question{Field: "coverage:" + it.Key, Family: "coverage-1", AmberAtMost: true,
			Instructions: roleQuestion{
				Question: "Which quoted fee or bundle covers the service " + it.Label + "? A bundled investigation, design or appointment fee may cover several services. Select the same bundle for each service it covers. This is coverage of the service, not confirmation of every detailed RFP requirement.",
				NotFor:   "Do not divide or add amounts. Do not assume an unmentioned service is excluded. Prefer a package fee over its component breakdown. Choose unstated when the source does not establish coverage.",
			}, Options: opts})
	}
}

func (d *Draft) quotedReview() *QuoteReview {
	r := &QuoteReview{Fees: []QuotedFee{}, Coverage: map[string]QuoteCoverage{}, Issues: []string{}}
	seen := map[string]bool{}
	for _, a := range quoteAmounts(d.ev) {
		role := "Package / fee — scope awaiting alignment"
		if unitWord.MatchString(a.Context) && len(moneyIn(a.Context)) > 1 {
			role = "Breakdown component — not additional to package"
		}
		if quoteSummaryWords.MatchString(a.Label + " " + a.Context) {
			role = "Package / total — do not add to its breakdown"
		}
		if quoteConditionalWords.MatchString(a.Context) {
			role = "Conditional / optional fee"
		}
		if strings.Contains(strings.ToLower(a.Context), "includes the") && strings.Contains(strings.ToLower(a.Context), "optional") {
			role = "Alternative including other work — do not add both"
		}
		label := a.Label
		if label == "" {
			label = "Quoted fee"
		}
		var amount *string
		gst := a.GST
		if gst == "" {
			gst = d.by[FieldGSTBasis].Value
			if d.flags[FlagGSTNotStated] && d.by[FieldGSTBasis].DecidedBy != ByUser {
				gst = ""
			}
		}
		if gst == "ex" || gst == "inc" {
			n := a.Cents
			if gst == "inc" {
				n = exGST(n)
			}
			v := decimalMoney(n)
			amount = &v
		}
		r.Fees = append(r.Fees, QuotedFee{ID: a.ID, Label: label, Printed: a.Display, Amount: amount, Role: role, Page: a.Page, Source: a.Context, Heading: a.Under, Suggestion: d.coverage["offer-role:"+a.ID]})
		seen[a.ID] = true
	}
	for field, answer := range d.coverage {
		if !strings.HasPrefix(field, "coverage:") {
			continue
		}
		key := strings.TrimPrefix(field, "coverage:")
		c := QuoteCoverage{Status: "Coverage unclear"}
		if seen[answer] {
			c.FeeID, c.Status = answer, "Suggested coverage — check source"
		} else if answer == "excluded" {
			c.Status = "Suggested exclusion — check source"
		}
		r.Coverage[key] = c
	}
	if d.ev.OCR {
		r.Issues = append(r.Issues, "Some source pages were read with OCR; check the figures.")
	}
	if d.ev.Incomplete {
		r.Issues = append(r.Issues, "Some pages could not be read. Missing coverage remains unknown.")
	}
	if d.ev.OptionsAB {
		r.Issues = append(r.Issues, "The source contains alternatives. Select one scope before confirming a comparison amount.")
	}
	return r
}

func (d *Draft) captureCoverage(f string, q question, a jev.Answer) {
	for _, o := range q.Options {
		if o.ID == a.Choice {
			d.coverage[f] = o.Value
			return
		}
	}
}
