package jobs

import "sort"

// LabeledAnswer is one Question A answer from a scored run against the
// owner's key: Jev's probability that the document answers the RFI, and
// whether the key says it does.
type LabeledAnswer struct {
	Probability float64
	Answers     bool
}

// ProposeRegisterCutoff proposes the lowest amber cut-off at which at least
// precision of the picks shown are right, so the owner can approve it in
// data/register/thresholds.json. The cut-off is on the probability of the
// action taken, here "this document answers this RFI"
// (https://docs.typesafe.ai/confidence). ok is false when no cut-off
// reaches the precision, or the run has no positive answer: then nothing
// should be shown.
func ProposeRegisterCutoff(run []LabeledAnswer, precision float64) (cut, reached, recall float64, ok bool) {
	sorted := append([]LabeledAnswer(nil), run...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Probability > sorted[j].Probability })
	positives := 0
	for _, a := range sorted {
		if a.Answers {
			positives++
		}
	}
	if positives == 0 {
		return 0, 0, 0, false
	}
	right := 0
	for i, a := range sorted {
		if a.Answers {
			right++
		}
		// A cut-off sits between distinct probabilities only.
		if i+1 < len(sorted) && sorted[i+1].Probability == a.Probability {
			continue
		}
		if p := float64(right) / float64(i+1); p >= precision {
			cut, reached, recall, ok = a.Probability, p, float64(right)/float64(positives), true
		}
	}
	return cut, reached, recall, ok
}
