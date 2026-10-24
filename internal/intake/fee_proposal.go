package intake

import (
	"regexp"
	"strings"

	"sitewise/internal/identity"
)

var (
	feeProposalName = regexp.MustCompile(`(?i)\bfee[\s_-]+(?:proposal|submission)\b`)
	// A document about fee proposals is not one: an evaluation, comparison,
	// register or request for them.
	aboutFeeProposals = regexp.MustCompile(`(?i)\b(?:evaluation|comparison|assessment|report|recommendation|register|request|rfp|summary|schedule of)\b`)
)

// feeProposalVocabulary files a consultant's fee proposal by what it calls
// itself: its own first heading, or the filename when the text has no
// heading. A fee proposal prices services, so its kind is Commercial, and
// it belongs to the consultant's engagement, so its area is Consultant
// (folder 02-consultant). A contractor's tender or quote is not covered.
func feeProposalVocabulary(filename string, text identity.Text) []Result {
	name := firstHeading(text)
	if name == "" {
		name = strings.TrimSuffix(filename, ".")
	}
	if !feeProposalName.MatchString(name) || aboutFeeProposals.MatchString(name) {
		return nil
	}
	return []Result{
		{Field: FieldKind, Settled: true, Rule: "fee_proposal_heading", Display: "commercial", Normalized: "commercial"},
		{Field: FieldLifecycle, Settled: true, Rule: "fee_proposal_heading", Display: "consultant", Normalized: "consultant"},
	}
}

// firstHeading is a Markdown document's first "# " heading, or "".
func firstHeading(text identity.Text) string {
	if text.Format != "md" && text.Format != "markdown" {
		return ""
	}
	for _, run := range text.Runs {
		line := strings.TrimSpace(run.Text)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
		return ""
	}
	return ""
}
