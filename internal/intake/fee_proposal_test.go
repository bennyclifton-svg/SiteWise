package intake

import (
	"testing"

	"sitewise/internal/identity"
)

func TestFeeProposalFilesAsCommercialConsultant(t *testing.T) {
	md := func(lines ...string) identity.Text {
		t := identity.Text{Format: "md"}
		for _, l := range lines {
			t.Runs = append(t.Runs, identity.Run{Text: l})
		}
		return t
	}
	for _, tc := range []struct {
		name     string
		filename string
		text     identity.Text
		want     bool
	}{
		{"own heading", "01-quoin.md", md("# Fee proposal — architectural services", "body"), true},
		{"filename when no heading", "Fee Proposal - Varga Traffic.pdf", identity.Text{Format: "pdf"}, true},
		{"fee submission", "x.md", md("# Fee submission: structural engineering"), true},
		{"evaluation report about proposals", "x.md", md("# Fee proposal evaluation report"), false},
		{"rfp", "Request for fee proposal - architect.pdf", identity.Text{Format: "pdf"}, false},
		{"heading wins over filename", "fee-proposal-quoin.md", md("# Master programme"), false},
		{"tender", "Tender - joinery.pdf", identity.Text{Format: "pdf"}, false},
	} {
		got := feeProposalVocabulary(tc.filename, tc.text)
		if (len(got) == 2) != tc.want {
			t.Errorf("%s: %+v", tc.name, got)
			continue
		}
		if tc.want && (got[0].Normalized != "commercial" || got[1].Field != FieldLifecycle || got[1].Normalized != "consultant") {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}
}
