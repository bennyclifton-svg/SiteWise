package invoices

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"sitewise/internal/identity"
)

// The invoices in testdata are copied from the synthetic test corpus
// (testing/synthetic/scenarios, findings F5, F8 and F13).
func readMarkdown(t *testing.T, name string, pc Context) Outcome {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(raw), int64(len(raw)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	out, err := Read(context.Background(), &fakeAsk{}, calibrated(), name, text, pc, nil, time.Now(), "")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ledgerFlags(out Outcome) map[string]bool {
	inv := Invoice{ID: "1", Status: StatusFiled, Kind: out.Kind, Currency: out.Currency, Number: out.Number, Date: out.Date,
		ABN: out.ABN, PackageID: out.PackageID, Ex: out.Ex, GST: out.GST, Total: out.Total}
	got := map[string]bool{}
	for _, f := range Check([]Invoice{inv}, nil, "2026-11-04")["1"] {
		got[f.Code] = true
	}
	return got
}

func decision(out Outcome, field string) Decision {
	for _, d := range out.Decisions {
		if d.Field == field {
			return d
		}
	}
	return Decision{}
}

// A header printed as a Markdown table, "| Date | 05/05/2026 |", labels the date.
func TestTableRowDateIsTheInvoiceDate(t *testing.T) {
	out := readMarkdown(t, "karrak-3307.md", Context{})
	if out.Date != "2026-05-05" || decision(out, FieldDate).DecidedBy != DecidedByRule {
		t.Fatalf("date %q by %+v", out.Date, decision(out, FieldDate))
	}
}

// F13: the only ABN printed beside "ABN" fails the checksum. It is kept as
// printed so the ledger can say so.
func TestChecksumFailingABNIsKeptAndFlagged(t *testing.T) {
	out := readMarkdown(t, "karrak-3307.md", Context{})
	if out.ABN != "77650369250" {
		t.Fatalf("abn %q", out.ABN)
	}
	if !ledgerFlags(out)["abn_invalid"] {
		t.Fatalf("no abn_invalid: %v", ledgerFlags(out))
	}
}

// A bad ABN printed for the customer is not the issuer's.
func TestCustomersBadABNIsNotTheIssuers(t *testing.T) {
	text := page("TAX INVOICE", "Invoice Number|INV-7", "Client ABN|54 165 354 219", "Subtotal|100.00", "GST|10.00", "Total|110.00")
	r := NewDraft(Harvest("a.pdf", text), Context{}, nil).Finish(time.Now())
	if r.ABN != "" {
		t.Fatalf("customer's ABN taken as the issuer's: %q", r.ABN)
	}
}

// A valid ABN on the page leaves the issuer's ABN to Jev, as before.
func TestBadABNWaitsWhenAValidOneIsPrinted(t *testing.T) {
	text := page("TAX INVOICE", "Invoice Number|INV-7", "ABN|54 165 354 219", "Customer ABN|81 139 719 529", "Subtotal|100.00", "GST|10.00", "Total|110.00")
	d := NewDraft(Harvest("a.pdf", text), Context{}, nil)
	if _, ok := d.by[FieldABN]; ok {
		t.Fatalf("abn settled by rule: %+v", d.by[FieldABN])
	}
}

// F5: ex $3,800, GST $380, total $4,280. The printed ex GST is kept, so the
// ledger says the totals don't add up, not that GST isn't 10 %.
func TestPrintedExGSTShowsAnArithmeticError(t *testing.T) {
	out := readMarkdown(t, "rg-0418.md", Context{})
	if out.Ex == nil || *out.Ex != 380000 || out.GST == nil || *out.GST != 38000 || out.Total == nil || *out.Total != 428000 {
		t.Fatalf("amounts ex=%v gst=%v total=%v", out.Ex, out.GST, out.Total)
	}
	flags := ledgerFlags(out)
	if !flags["arithmetic"] || flags["gst_rate"] {
		t.Fatalf("flags %v", flags)
	}
}

// F31: the T17 fire engineer prints "Fee $14,500.00, GST $1,450.00, Total
// due $16,850.00". A "Fee" line before the GST is the ex-GST figure, so the
// ledger says the totals don't add up, not that GST isn't 10 %.
func TestPrintedFeeLineIsTheExGST(t *testing.T) {
	out := readMarkdown(t, "kestrel-k-26-0412.md", Context{})
	if out.Ex == nil || *out.Ex != 1450000 || out.GST == nil || *out.GST != 145000 || out.Total == nil || *out.Total != 1685000 {
		t.Fatalf("amounts ex=%v gst=%v total=%v", out.Ex, out.GST, out.Total)
	}
	flags := ledgerFlags(out)
	if !flags["arithmetic"] || flags["gst_rate"] {
		t.Fatalf("flags %v", flags)
	}
}

// Of several "Amount" lines, only the one the GST is 10 % of is the ex GST.
func TestFeeLineNeedsItsGST(t *testing.T) {
	cands := []Candidate{
		{Value: "50000", Labels: []string{"amount"}},
		{Value: "100000", Labels: []string{"| amount |"}},
		{Value: "10000", Labels: []string{"| gst |"}},
		{Value: "120000", Labels: []string{"| total |"}},
	}
	e, g, tot, ok := printedAmounts(cands)
	if !ok || e != 100000 || g != 10000 || tot != 120000 {
		t.Fatalf("got %d %d %d %v", e, g, tot, ok)
	}
}

// The printed ex GST survives a later recompute from stored decisions.
func TestResolveKeepsAPrintedExGST(t *testing.T) {
	out := readMarkdown(t, "rg-0418.md", Context{})
	by := map[string]Decision{}
	for _, d := range out.Decisions {
		by[d.Field] = d
	}
	by[FieldTotal] = Decision{Field: FieldTotal, Value: "418000", Band: BandGreen, DecidedBy: DecidedByUser}
	r := Resolve(by)
	if r.Ex == nil || *r.Ex != 380000 {
		t.Fatalf("ex %v", r.Ex)
	}
}

// A consistent invoice still reads ex GST as total less GST.
func TestConsistentInvoiceUnchanged(t *testing.T) {
	out := readMarkdown(t, "tce-5521.md", Context{})
	if out.Ex == nil || *out.Ex != 150000 || decision(out, FieldExGST).QuestionVersion != "" {
		t.Fatalf("ex %v %+v", out.Ex, decision(out, FieldExGST))
	}
}

var structural = Package{ID: "structural", Title: "Structural", Kind: "services"}

// F8: the firm that proposed for a package is named in the invoice's heading.
func TestSupplierByProposingFirmName(t *testing.T) {
	s := structural
	s.Firms = []Firm{{NameKey: "karrakengineers", ABN: "77650369205"}}
	b := Package{ID: "building", Title: "Building Consultant", Kind: "services",
		Firms: []Firm{{NameKey: "parapetbuilding", ABN: "66049460444"}, {NameKey: "halyardremedial", ABN: "41434467555"}}}
	out := readMarkdown(t, "karrak-3307.md", Context{Packages: []Package{b, s}})
	if out.PackageID != "structural" || decision(out, FieldSupplier).DecidedBy != DecidedByRule {
		t.Fatalf("supplier %q %+v", out.PackageID, decision(out, FieldSupplier))
	}
}

// F8: an ABN on the invoice matches the ABN of a firm that proposed.
func TestSupplierByProposingFirmABN(t *testing.T) {
	b := Package{ID: "building", Title: "Building Consultant", Kind: "services",
		Firms: []Firm{{NameKey: "parapetbuilding", ABN: "66049460444"}, {NameKey: "halyardremedial", ABN: "41434467555"}}}
	out := readMarkdown(t, "pbc-inv-1192.md", Context{Packages: []Package{structural, b}})
	if out.PackageID != "building" || decision(out, FieldSupplier).Band != BandGreen {
		t.Fatalf("supplier %q %+v", out.PackageID, decision(out, FieldSupplier))
	}
}

// A firm named only as the customer is not the supplier.
func TestCustomerFirmIsNotTheSupplier(t *testing.T) {
	text := page("Tallow Engineers Pty Ltd", "TAX INVOICE", "Invoice Number|INV-7", "Bill to: Harbourline Architects Pty Ltd",
		"Subtotal|100.00", "GST|10.00", "Total|110.00")
	arch := Package{ID: "arch", Title: "Architect", Kind: "services", Firms: []Firm{{NameKey: "harbourlinearchitects"}}}
	r := NewDraft(Harvest("a.pdf", text), Context{Packages: []Package{arch}}, nil).Finish(time.Now())
	if r.PackageID != "" {
		t.Fatalf("customer taken as supplier: %q", r.PackageID)
	}
}

// One firm proposing for two packages names neither.
func TestFirmOnTwoPackagesIsAmbiguous(t *testing.T) {
	s := structural
	s.Firms = []Firm{{NameKey: "karrakengineers"}}
	c := Package{ID: "civil", Title: "Civil", Kind: "services", Firms: []Firm{{NameKey: "karrakengineers"}}}
	out := readMarkdown(t, "karrak-3307.md", Context{Packages: []Package{s, c}})
	if out.PackageID != "" {
		t.Fatalf("supplier %q", out.PackageID)
	}
}
