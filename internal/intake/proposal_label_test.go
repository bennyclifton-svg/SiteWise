package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

// A proposal or quote names itself by its own reference (Kavanagh, 24
// October 2026: "Proposal reference: QUA-KAV-2601" left the number blank).
func TestProposalAndQuoteReferencesAreNumbers(t *testing.T) {
	for _, tc := range []struct{ line, number string }{
		{"Proposal reference: QUA-KAV-2601", "QUA-KAV-2601"},
		{"Proposal No. CST-0126", "CST-0126"},
		{"Fee proposal number FLH-014", "FLH-014"},
		{"Quotation Number: QUO-2093", "QUO-2093"},
		{"Quote ref: VCA-2607", "VCA-2607"},
	} {
		got := intake.Harvest("fee proposal.pdf", identity.Text{Runs: []identity.Run{{Text: tc.line}}})
		c, ok := findCandidate(got, intake.FieldNumber, tc.number)
		if !ok || !c.Provenance.Labeled {
			t.Errorf("%q: number %q not a labelled candidate: %+v", tc.line, tc.number, got)
		}
	}
}
