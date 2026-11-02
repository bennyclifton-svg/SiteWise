package proposals

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

func harvestMarkdown(t *testing.T, filename, body string) Evidence {
	t.Helper()
	txt, err := identity.Extract(context.Background(), "md", strings.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return Harvest(filename, txt)
}

// A tax invoice dropped on a package row is a claim against a proposal, not a
// new one. If it reads as a proposal it supersedes the firm's real proposal
// (ST01 CDC, 2026-11-02: the $80,000 four-stage fee became one $18,000 line).
func TestInvoiceTitleIsNotAProposal(t *testing.T) {
	invoices := map[string]string{
		"Tax invoice INV-2307 Harbourline.md": "Title: Tax invoice\nInvoice number: INV-2307\nDate: 29 May 2026\nABN: 95 471 173 493\n\n# Tax invoice\n\nHarbourline Architects Pty Ltd\nOur ref: HLA-FP-2611\n\n## Services this claim\n\n| Stage | Fee | Complete | Claimed (ex GST) |\n|---|---|---|---|\n| Stage 1 Concept design | $18,000.00 | 100% | $18,000.00 |\n\nSubtotal ex GST: $18,000.00\n",
		"TCE invoice 5521.md":                 "| Invoice | 5521 |\n|---|---|\n| Date | 05/06/2026 |\n| ABN | 25 999 583 278 |\n\nTallow Consulting Engineers Pty Ltd\n\n| Item | Fee |\n|---|---|\n| Concept structural advice | $1,500.00 |\n",
	}
	for name, body := range invoices {
		if k := kindRule(harvestMarkdown(t, name, body), "consultant.architect"); k != KindNone {
			t.Errorf("%s settled as %q, want %q", name, k, KindNone)
		}
	}
	// A proposal that says how it will invoice is still a proposal.
	proposal := "Title: Fee proposal: architectural services\nProposal no: HLA-FP-2611\n\n# Fee proposal: architectural services\n\nHarbourline Architects Pty Ltd\n\n## 3. Fees\n\n| Stage | Fee (ex GST) |\n|---|---|\n| Stage 1 Concept design | $18,000 |\n| Stage 2 CDC documentation and lodgement | $22,000 |\n\n## 5. Payment\n\nFees are invoiced monthly. We issue a tax invoice at the end of each month.\n"
	if k := kindRule(harvestMarkdown(t, "Fee proposal - Architectural services.md", proposal), "consultant.architect"); k != KindFeeProposal {
		t.Errorf("fee proposal settled as %q", k)
	}
}
