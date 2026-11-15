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

// A header table names the client in a labelled row ("| Client | … Pty Ltd |").
// That row is the addressee, never the firm, even though it carries the only
// registered company name in the header; nor is the project's address the
// firm's (Seven Hills corpus, 15 Nov 2026: 23 proposals read the client as
// the firm). Every name and address here is invented.
func TestHeaderTableClientRowIsNotTheFirm(t *testing.T) {
	body := "# Fee Proposal — Structural Engineering\n\n" +
		"> Invented document for a test.\n\n" +
		"| Field | Detail |\n|---|---|\n" +
		"| Project | 14–18 Example Avenue, Exampleton NSW 2147 |\n" +
		"| Client | Example Developments Pty Ltd |\n" +
		"| RFP | RFP-STR-01 |\n" +
		"| Proponent | Gridline Engineers |\n" +
		"| Proposal date | 26 June 2025 |\n\n" +
		"## Offer\n\nGridline Engineers offers its services for a lump-sum fee of **$112,000.00 excluding GST**.\n"
	ev := markdownEvidence(t, body)
	if ev.HeaderFirm != "" {
		t.Fatalf("header firm %q: the client row was read as the firm", ev.HeaderFirm)
	}
	d := NewDraft(ev, Context{PackageID: "p", SiteAddress: "14-18 Example Avenue, Exampleton NSW 2147"}, nil)
	for _, field := range []string{FieldFirmName, FieldFirmAddress} {
		if got, ok := d.Decided(field); ok && strings.Contains(got.Value, "Example") {
			t.Fatalf("%s decided as %q: that is the client's or the project's", field, got.Value)
		}
	}
	// The row the document labels as the proponent names the firm.
	if got, ok := d.Decided(FieldFirmName); !ok || got.Value != "Gridline Engineers" {
		t.Fatalf("firm %+v", got)
	}
}

// A header table row that names the firm under another label ("| Builder |
// … Pty Ltd |") gives the name alone, without its label. Invented names.
func TestHeaderTableFirmRowGivesTheNameAlone(t *testing.T) {
	body := "# Letter of Acceptance\n\n" +
		"| Field | Detail |\n|---|---|\n" +
		"| Client | Example Developments Pty Ltd |\n" +
		"| Builder | Example Building Group Pty Ltd |\n" +
		"| Date | 4 May 2026 |\n\n## Acceptance\n\nThe tender is accepted.\n"
	if got := markdownEvidence(t, body).HeaderFirm; got != "Example Building Group Pty Ltd" {
		t.Fatalf("header firm %q", got)
	}
}

// Candidate names come from a table row's cells, never the row with its
// label: Jev and the directory see "Example Studio", not "Consultant |
// Example Studio". A labelled client or project row's address is not the
// firm's, even when the site's address is not known. Invented names.
func TestHeaderTableNamesAreCellsAndProjectRowIsNoAddress(t *testing.T) {
	body := "# Letter of Appointment\n\n" +
		"| Field | Detail |\n|---|---|\n" +
		"| Project | 14–18 Example Avenue, Exampleton NSW 2147 |\n" +
		"| Client | Example Developments Pty Ltd |\n" +
		"| Consultant | Example Studio |\n\n" +
		"The Client appoints **Example Studio** to provide architectural services.\n"
	ev := markdownEvidence(t, body)
	if !containsFold(ev.Names, "Example Studio") {
		t.Fatalf("names %q", ev.Names)
	}
	for _, n := range ev.Names {
		if strings.Contains(n, "|") || strings.HasPrefix(n, "Consultant") || strings.HasPrefix(n, "Client") {
			t.Fatalf("a whole table row is a candidate name: %q", n)
		}
	}
	if len(ev.Addresses) != 0 {
		t.Fatalf("addresses %q: the project row is the site's", ev.Addresses)
	}
}

// A role-labelled row offers its value to Jev as a candidate, even when the
// name has no company word ("Northline Structures"), so an appointment
// letter's firm is there to choose. It is offered, not decided. Invented names.
func TestRoleLabelledRowOffersItsFirm(t *testing.T) {
	body := "# Letter of Appointment\n\n" +
		"| Field | Detail |\n|---|---|\n" +
		"| Client | Example Developments Pty Ltd |\n" +
		"| Consultant | Northline Example |\n\n" +
		"The Client appoints Northline Example to provide structural services.\n"
	ev := markdownEvidence(t, body)
	if !containsFold(ev.Names, "Northline Example") {
		t.Fatalf("names %q", ev.Names)
	}
	if got, ok := NewDraft(ev, Context{PackageID: "p"}, nil).Decided(FieldFirmName); ok {
		t.Fatalf("a Consultant row decided the firm by rule: %+v", got)
	}
}
