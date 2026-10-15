package invoices

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
)

// page lays out lines as runs with geometry, top to bottom, label and value
// in separate cells when split with "|".
func page(lines ...string) identity.Text {
	t := identity.Text{TextLayer: true, Format: "pdf", PageCount: 1}
	y := 800.0
	for _, l := range lines {
		x := 50.0
		for _, cell := range splitCells(l) {
			t.Runs = append(t.Runs, identity.Run{Text: cell, Source: identity.Source{Page: 1, X: x, Y: y, Width: 6 * float64(len(cell)), Height: 9}})
			x += 220
		}
		y -= 14
	}
	return t
}

func splitCells(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '|' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

func TestParseCentsAndFormat(t *testing.T) {
	for in, want := range map[string]int64{"11,000.00": 1100000, "$5,500": 550000, "-12.5": -1250, "(1,000.00)": -100000, "0.41": 41} {
		got, ok, err := ParseCents(in)
		if err != nil || !ok || got != want {
			t.Fatalf("%q = %d %v %v", in, got, ok, err)
		}
	}
	if _, ok, err := ParseCents(""); ok || err != nil {
		t.Fatal("blank is unknown, not an error")
	}
	for _, bad := range []string{"12.345", "abc", "1e5", "99999999999999999"} {
		if _, _, err := ParseCents(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if FormatCents(-1250) != "-12.50" || DisplayCents(123456789) != "1,234,567.89" {
		t.Fatal(FormatCents(-1250), DisplayCents(123456789))
	}
}

func TestABNChecksum(t *testing.T) {
	for _, ok := range []string{"54165354218", "81139719529", "74099962231"} {
		if !ValidABN(ok) {
			t.Fatalf("%s rejected", ok)
		}
	}
	for _, bad := range []string{"54165354219", "00000000000", "5416535421"} {
		if ValidABN(bad) {
			t.Fatalf("%s accepted", bad)
		}
	}
}

func TestRulesSettleAPlainTaxInvoice(t *testing.T) {
	text := page("TAX INVOICE", "Invoice Date|20 Jun 2025", "Invoice Number|A25183", "ABN 54 165 354 218",
		"Subtotal|5,000.00", "Total GST 10%|500.00", "Invoice Total AUD|5,500.00")
	d := NewDraft(Harvest("ERVM INV A25183.pdf", text), Context{}, nil)
	r := d.Finish(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC))
	if r.Kind != "invoice" || r.Number != "A25183" || r.Date != "2025-06-20" || *r.Total != 550000 || *r.GST != 50000 || *r.Ex != 500000 {
		t.Fatalf("%+v", r)
	}
	if r.Period != "2026-10-01" || r.Currency != "AUD" {
		t.Fatalf("period %s currency %s", r.Period, r.Currency)
	}
	// The issuer ABN is Jev's: a lone ABN can be the customer's.
	call, ok := d.Call()
	if !ok || len(call.Questions) != 1 || call.Questions[FieldABN].Type != jev.TypeChoice {
		t.Fatalf("questions %v", call.Questions)
	}
}

// A certification page with the contract's totals beside this invoice's
// must not be settled by rule.
func TestAmountRuleRefusesTwoConsistentSets(t *testing.T) {
	text := page("TAX INVOICE: NSW 231837", "Total (excl. GST)|$10,400", "GST|$1,040", "Total (incl. GST)|$11,440",
		"Sub-Total|$5,400.00", "GST|$540.00", "Due: 26 February 2024|$5,940.00")
	d := NewDraft(Harvest("x.pdf", text), Context{}, nil)
	r := d.Finish(time.Now())
	if r.Total != nil || r.GST != nil {
		t.Fatalf("rule settled an ambiguous page: %v %v", *r.Total, *r.GST)
	}
	call, _ := d.Call()
	if _, ok := call.Questions[FieldTotal]; !ok {
		t.Fatal("total not asked")
	}
}

func TestNumbersUnderAClaimHistoryHeaderAreNotThisInvoice(t *testing.T) {
	text := page("TAX INVOICE NO. 22510 - PROGRESS CLAIM NO. 2", "Date|Invoice No.|Claimed|Paid|Outstanding", "29/10/2025|22424|$11,000.00|$0.00|$0.00")
	ev := Harvest("x.pdf", text)
	if uniqueLabeled(ev.Numbers) != "22510" || ev.InvoiceCount != 1 {
		t.Fatalf("numbers %v", ev.Numbers)
	}
}

type fakeAsk struct {
	res jev.Result
	err error
	got jev.Call
}

func (f *fakeAsk) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	f.got = c
	return f.res, f.err
}

func conf(v float64) *float64 { return &v }

func calibrated(fields ...string) Thresholds {
	th := Thresholds{QuestionVersion: QuestionVersion, Questions: map[string][]Threshold{}}
	lo, hi := 2, 255
	for _, f := range fields {
		th.Questions[f] = []Threshold{{Options: &lo, MaxOptions: &hi, Green: conf(0.9), Amber: conf(0.5)}}
	}
	return th
}

func TestOneFanOutThenCodeDerivesTheLine(t *testing.T) {
	text := page("Invoice", "Invoice no: 9127", "Issue date|21/11/2023", "ABN: 58359914843", "Subtotal|$1,690.00", "Tax|$169.00", "Balance due|$1,859.00")
	pc := Context{Packages: []Package{{ID: "arb", Title: "Arborist", Kind: "services"}, {ID: "arch", Title: "Architect", Kind: "services"}},
		Lines: []Line{{ID: "L-arb", Label: "Arborist fee", PackageID: "arb"}, {ID: "L-arch", Label: "Architect fee", PackageID: "arch"}}}
	ask := &fakeAsk{}
	ask.res = jev.Result{Answers: map[string]jev.Answer{
		FieldKind:     {Type: jev.TypeChoice, Choice: "invoice", Confidence: conf(0.95)},
		FieldABN:      {Type: jev.TypeChoice, Choice: "c1", Confidence: conf(0.95)},
		FieldSupplier: {Type: jev.TypeChoice, Choice: "p1", Confidence: conf(0.95)},
		FieldLine:     {Type: jev.TypeChoice, Choice: "l2", Confidence: conf(0.95)},
	}}
	out, err := Read(context.Background(), ask, calibrated(FieldKind, FieldABN, FieldSupplier, FieldLine), "Invoice 9127.pdf", text, pc, nil, time.Now(), "")
	if err != nil {
		t.Fatal(err)
	}
	if ask.got.Priority != jev.PriorityInteractive || ask.got.QuestionVersion != QuestionVersion {
		t.Fatalf("call %+v", ask.got)
	}
	// The supplier's only line wins over Jev's line pick: code, not a second call.
	if out.PackageID != "arb" || out.LineID != "L-arb" || out.ABN != "58359914843" || *out.Ex != 169000 {
		t.Fatalf("%+v", out.Reading)
	}
}

func TestJevTimeoutKeepsRuleValuesAndGoesGrey(t *testing.T) {
	text := page("TAX INVOICE", "ABN 54 165 354 218", "Invoice Number|INV-1", "Subtotal|100.00", "GST|10.00", "Total AUD|110.00")
	out, err := Read(context.Background(), &fakeAsk{err: errors.New("deadline")}, calibrated(), "a.pdf", text, Context{}, nil, time.Now(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Grey || out.Number != "INV-1" || *out.Total != 11000 {
		t.Fatalf("%+v", out)
	}
	for _, d := range out.Decisions {
		if d.Field == FieldABN && d.Band != BandGrey {
			t.Fatalf("unanswered ABN band %s", d.Band)
		}
	}
}

func TestUserDecisionsAreNeverReplaced(t *testing.T) {
	text := page("TAX INVOICE", "Invoice Number|INV-1", "Subtotal|100.00", "GST|10.00", "Total AUD|110.00")
	users := map[string]Decision{FieldNumber: {Field: FieldNumber, Value: "INV-1A", Band: BandGreen, DecidedBy: DecidedByUser}}
	out, _ := Read(context.Background(), &fakeAsk{}, calibrated(), "a.pdf", text, Context{}, users, time.Now(), "")
	if out.Number != "INV-1A" {
		t.Fatalf("user number replaced: %s", out.Number)
	}
}

func TestCreditNoteIsNegative(t *testing.T) {
	by := map[string]Decision{
		FieldKind:  {Value: "credit_note", Band: BandGreen, DecidedBy: DecidedByUser},
		FieldTotal: {Value: "11000", Band: BandGreen, DecidedBy: DecidedByRule},
		FieldGST:   {Value: "1000", Band: BandGreen, DecidedBy: DecidedByRule},
	}
	r := Resolve(by)
	if *r.Ex != -10000 || *r.Total != -11000 || r.Kind != "credit_note" {
		t.Fatalf("%v %v", *r.Ex, *r.Total)
	}
}

func TestNotReadPaths(t *testing.T) {
	out, _ := Read(context.Background(), &fakeAsk{}, Thresholds{}, "scan.pdf", identity.Text{TextLayer: false}, Context{}, nil, time.Now(), "")
	if out.NotRead != NotReadNoText {
		t.Fatal(out.NotRead)
	}
	garbled := page("+,-./012/34", "/+5.,6789:;<:.+=.><?.@,A0.BC", "V)'W%")
	out, _ = Read(context.Background(), &fakeAsk{}, Thresholds{}, "x.pdf", garbled, Context{}, nil, time.Now(), "")
	if out.NotRead != NotReadUnreadable {
		t.Fatal(out.NotRead)
	}
}

func TestForeignCurrencyIsAmber(t *testing.T) {
	text := page("TAX INVOICE", "Invoice Number|INV-9", "Subtotal USD|100.00", "GST|10.00", "Total|110.00")
	r := NewDraft(Harvest("a.pdf", text), Context{}, nil).Finish(time.Now())
	if r.Currency != "USD" {
		t.Fatalf("currency %q", r.Currency)
	}
	inv := Invoice{Status: StatusFiled, Kind: "invoice", Currency: r.Currency, Ex: r.Ex}
	if Counted(inv) {
		t.Fatal("a USD invoice counts in AUD totals")
	}
	_ = strconv.Itoa
}
