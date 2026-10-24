package proposals

import (
	"context"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/rfp"
	"strings"
	"testing"
)

func markdownEvidence(t *testing.T, body string) Evidence {
	t.Helper()
	txt, err := identity.Extract(context.Background(), "md", strings.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return Harvest("fee-proposal.md", txt)
}

func TestForwardedQuoteIsOneOfferAndPartialScopeDoesNotConfirmPrice(t *testing.T) {
	ev := markdownEvidence(t, "From: Kate Example <kate@exampleflow.com.au>\nTo: Architect <a@architect.test>\n**Scope:**\nConcept detention sizing and location options.\n**Fee:** $4,850 + GST\nKate Example\nPrincipal, ExampleFlow Hydraulic Consultants\nPlease approve the quote attached ($4,850 + GST).\n")
	d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": {{Key: "option:osd", Kind: rfp.ItemOption, Label: "OSD design"}}}}, nil)
	if len(coverageAmounts(ev)) != 1 {
		t.Fatalf("repeated quote offered twice: %+v", coverageAmounts(ev))
	}
	d.Apply(jev.Result{Answers: map[string]jev.Answer{"coverage:option:osd": {Type: jev.TypeChoice, Choice: "partial_a1"}}}, nil, Thresholds{})
	r := d.Finish()
	if r.Quote.Coverage["option:osd"].Status != "Partial scope only — check source" {
		t.Fatal(r.Quote.Coverage)
	}
	if r.FirmFields[FieldFirmName] != "ExampleFlow Hydraulic Consultants" || r.FirmFields[FieldFirmEmail] != "kate@exampleflow.com.au" {
		t.Fatal(r.FirmFields)
	}
	for _, p := range r.Prices {
		if p.Key != "" && p.Cents != nil {
			t.Fatal("partial coverage confirmed money")
		}
	}
	if len(r.Quote.Fees) != 2 || !strings.HasPrefix(r.Quote.Fees[1].Role, "Repeated") {
		t.Fatal(r.Quote.Fees)
	}
	if len(r.Quote.Terms) != 1 || !strings.Contains(r.Quote.Terms[0].Text, "Concept") {
		t.Fatal(r.Quote.Terms)
	}
}

func TestMarkdownFeeReferenceIsNotUnitRate(t *testing.T) {
	ev := markdownEvidence(t, "# FEE PROPOSAL\n## Fee\n**$148,500 ex GST** staged per engagement letter.\n**Amelia Example**\nDirector, Example Clarke Studio\n")
	q := NewDraft(ev, Context{}, nil).Finish().Quote
	if len(q.Fees) != 1 || q.Fees[0].Amount == nil || *q.Fees[0].Amount != "148500.00" {
		t.Fatalf("lost fixed fee: %+v", q.Fees)
	}
	if !strings.Contains(strings.Join(ev.Names, "|"), "Example Clarke Studio") || strings.Contains(strings.Join(ev.Names, "|"), "Director,") {
		t.Fatalf("firm: %v", ev.Names)
	}
	if len(ev.Contacts) != 1 || ev.Contacts[0] != "Amelia Example" {
		t.Fatalf("contact: %v", ev.Contacts)
	}
	if unitPrice("staged per engagement letter") || !unitPrice("$250 per visit") {
		t.Fatal("unit rate boundary")
	}
}

func TestMarkdownStageTaxAndExclusions(t *testing.T) {
	ev := markdownEvidence(t, "# LETTER OF ENGAGEMENT\n## Scope of services\nConcept and schematic design.\n**Exclusions** unless separately agreed: landscape design; legal advice.\n## Fee basis\nFixed professional fee: **$30,000 excluding GST** ($33,000 incl. GST), staged as follows:\n| Stage | Trigger | Fee (ex GST) |\n| --- | --- | --- |\n| Concept | Signed | $10,000 |\n| Documentation | Issue | $20,000 |\n## Insurance\nProfessional indemnity $5,000,000\n")
	d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": {{Key: "criterion:r6", Kind: rfp.ItemCriterion, Label: "Exclusions"}}}}, nil)
	r := d.Finish()
	if len(r.Quote.Fees) != 3 {
		t.Fatalf("fees: %+v", r.Quote.Fees)
	}
	for _, f := range r.Quote.Fees {
		if f.Amount == nil {
			t.Errorf("tax lost: %+v", f)
		}
	}
	if len(r.Responses) != 1 || len(r.Responses[0].Passages) == 0 {
		t.Fatal("exclusions lost")
	}
	if r.StatedTotal == nil || *r.StatedTotal != 3000000 {
		t.Fatalf("total: %v", r.StatedTotal)
	}
}
