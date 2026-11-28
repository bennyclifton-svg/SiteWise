package store_test

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// Slice 6 of docs/plans/2026-11-26-payment-schedules.md: the claim counts,
// its tax invoice is linked and never counted, and an invoice no rule can
// place is held out of the totals until the PM places it (owner decision 2).

func invoiceIn(t *testing.T, s *store.Store, pkg, line, name string, total, gst int64) string {
	t.Helper()
	ctx := context.Background()
	created, err := s.CreateInvoice(ctx, orgA, projectA, upload(name), invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitInvoiceRead(ctx, orgA, created.ID, reading(pkg, line, name, total, gst), false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	return created.ID
}

func invoiceOf(t *testing.T, v store.LedgerView, id string) invoices.Invoice {
	t.Helper()
	for _, inv := range v.Invoices {
		if inv.ID == id {
			return inv
		}
	}
	t.Fatalf("invoice %s not in the ledger", id)
	return invoices.Invoice{}
}

func flagged(v store.LedgerView, id, code string) bool {
	for _, f := range v.Flags[id] {
		if f.Code == code {
			return true
		}
	}
	return false
}

func issueAsClaimed(t *testing.T, s *store.Store, claim string) {
	t.Helper()
	ctx := context.Background()
	if err := s.AssessClaim(ctx, orgA, projectA, userA, claim, store.ClaimAssess{AcceptClaimed: true}); err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssuePaymentSchedule(ctx, orgA, projectA, userA, claim, "2026-10-16", blobs); err != nil {
		t.Fatal(err)
	}
}

// The invoice arrives before the schedule (the golden's own order) at the
// claim's printed total: it is the claim's, waiting, and never counted. An
// invoice that matches nothing is held until the PM places it.
func TestTaxInvoiceBeforeTheScheduleAndAnUnplacedOne(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	claim := dropClaim(t, s, pkg, lines, "claim-4")
	// While the claim waits for its invoice, one that matches nothing is held.
	odd := invoiceIn(t, s, pkg, lines[0], "INV-51", 1234500, 112227)
	v := ledgerNow(t, s)
	if inv := invoiceOf(t, v, odd); inv.SettlesClaim != "" || !inv.Held || invoices.Counted(inv) || !flagged(v, odd, "claim_invoice_unplaced") {
		t.Fatalf("unplaced invoice: %+v flags %+v", inv, v.Flags[odd])
	}
	if c := claimOf(t, v, claim); !c.AwaitingInvoice {
		t.Fatalf("claim not awaiting: %+v", c)
	}
	early := invoiceIn(t, s, pkg, lines[0], "INV-50", 5060000, 460000)
	v = ledgerNow(t, s)
	if inv := invoiceOf(t, v, early); inv.SettlesClaim != claim || inv.SettlesBy != invoices.DecidedByRule || inv.LinkNote != invoices.LinkWaiting || invoices.Counted(inv) {
		t.Fatalf("early invoice: %+v", inv)
	}
	if v.Summary.Total != 1122273 {
		t.Fatalf("total %d: only the invoice that is not the claim's counts once the claim has its own", v.Summary.Total)
	}
	ledgerTotals(t, v)

	// Issued: the claim counts at its scheduled amount, the invoice stays
	// linked, and it invoiced the claimed amount, not the schedule's payable.
	issueAsClaimed(t, s, claim)
	v = ledgerNow(t, s)
	if inv := invoiceOf(t, v, early); inv.SettlesClaim != claim || inv.LinkNote != invoices.LinkClaimedAmount || invoices.Counted(inv) || !flagged(v, early, "claim_invoice_claimed_amount") {
		t.Fatalf("after issue: %+v flags %+v", inv, v.Flags[early])
	}
	if v.Summary.Total != 8600000+1122273 {
		t.Fatalf("total %d, want the schedule's 86,000.00 and the other invoice", v.Summary.Total)
	}
	ledgerTotals(t, v)
	// The payment list carries the linked invoice's own number.
	sheet, ok := store.ClaimSheet(v, claim)
	if !ok || len(sheet.Payments) != 1 || sheet.Payments[0].InvoiceNo != "INV-50" {
		t.Fatalf("payment list %+v", sheet.Payments)
	}

	// Once every claim has its invoice, nothing is held: a new invoice counts.
	if inv := invoiceOf(t, v, odd); inv.Held || !invoices.Counted(inv) {
		t.Fatalf("no claim waits for an invoice, yet held: %+v", inv)
	}
	if err := s.EditInvoice(ctx, orgA, projectA, odd, userA, store.InvoiceEdit{Action: "not_claim"}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if inv := invoiceOf(t, ledgerNow(t, s), odd); !inv.NotClaim || !invoices.Counted(inv) {
		t.Fatalf("not for a claim: %+v", inv)
	}
	// Unsettle puts it back for the rule to place.
	if err := s.EditInvoice(ctx, orgA, projectA, odd, userA, store.InvoiceEdit{Action: "unsettle"}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if inv := invoiceOf(t, ledgerNow(t, s), odd); inv.NotClaim {
		t.Fatalf("unsettle left not_claim: %+v", inv)
	}
	// A progress claim cannot be marked not a claim.
	if err := s.EditInvoice(ctx, orgA, projectA, claim, userA, store.InvoiceEdit{Action: "not_claim"}, invoiceNow); err == nil {
		t.Fatal("a claim was marked not a claim")
	}
}

// A cent out on GST links on the ex GST amount; two claims that could take
// one invoice take none; a claim that is its own tax invoice wants none, and
// prints its own number on the payment list.
func TestTaxInvoiceLinkRuleCases(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	first := dropClaim(t, s, pkg, lines, "claim-4")
	issueAsClaimed(t, s, first)
	// Payable 92,070.00; net 83,700.00. The invoice rounded GST the other way.
	cent := invoiceIn(t, s, pkg, lines[0], "INV-60", 9206900, 836900)
	v := ledgerNow(t, s)
	if inv := invoiceOf(t, v, cent); inv.SettlesClaim != first || invoices.Counted(inv) {
		t.Fatalf("a cent out: %+v", inv)
	}
	if v.Summary.Total != 8600000 {
		t.Fatalf("total %d", v.Summary.Total)
	}

	second := dropClaim(t, s, pkg, lines, "claim-5")
	third := dropClaim(t, s, pkg, lines, "claim-6")
	both := invoiceIn(t, s, pkg, lines[0], "INV-61", 5060000, 460000)
	v = ledgerNow(t, s)
	if inv := invoiceOf(t, v, both); inv.SettlesClaim != "" || !inv.Held || invoices.Counted(inv) {
		t.Fatalf("two claims could take it: %+v", inv)
	}

	// The third claim is its own tax invoice: it wants no other, so the
	// invoice now has one claim to go to.
	yes := "yes"
	if err := s.EditInvoice(ctx, orgA, projectA, third, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldTaxInvoice: &yes}}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v = ledgerNow(t, s)
	if inv := invoiceOf(t, v, both); inv.SettlesClaim != second || inv.Held {
		t.Fatalf("one claim left: %+v", inv)
	}
	if c := claimOf(t, v, third); !c.TaxInvoice || c.AwaitingInvoice {
		t.Fatalf("own tax invoice: %+v", c)
	}
	if c := claimOf(t, v, second); c.AwaitingInvoice {
		t.Fatalf("linked claim still awaiting: %+v", c)
	}
	issueAsClaimed(t, s, third)
	sheet, ok := store.ClaimSheet(ledgerNow(t, s), third)
	if !ok {
		t.Fatal("no sheet")
	}
	for _, p := range sheet.Payments {
		if p.ClaimID == third && (p.InvoiceNo != "4" || p.InvoiceDate != "2026-10-14") {
			t.Fatalf("own invoice row %+v", p)
		}
	}
	if err := s.EditInvoice(ctx, orgA, projectA, third, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldTaxInvoice: sp("maybe")}}, invoiceNow); err == nil {
		t.Fatal("tax_invoice took a value that is not yes or no")
	}
	_ = strings.TrimSpace
}
