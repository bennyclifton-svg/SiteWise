package invoices

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
)

func claimContext() Context {
	return Context{
		Packages: []Package{{ID: "mw", Title: "Main Works", Kind: "works"}},
		Contracts: map[string]ClaimContract{"mw": {PackageID: "mw",
			Lines:      []Line{{ID: "L-sub", Label: "Substructure"}, {ID: "L-sup", Label: "Superstructure"}, {ID: "L-fin", Label: "Finishes"}},
			Variations: []ClaimVariation{{ID: "V-1", Number: "V01", Title: "Rock excavation", Status: VariationApproved}}}},
	}
}

// The PM dropped it on Main Works: kind and supplier are the PM's.
func dropped() map[string]Decision {
	return map[string]Decision{
		FieldKind:     {Field: FieldKind, Value: KindProgressClaim, Band: BandGreen, DecidedBy: DecidedByUser},
		FieldSupplier: {Field: FieldSupplier, Value: "mw", Band: BandGreen, DecidedBy: DecidedByUser},
	}
}

// A PDF-shaped claim: the header names the three columns; rows under it
// carry them by position.
func pdfClaim() identity.Text {
	return page(
		"Progress Claim No. 4",
		"Claim no: PC-04",
		"Item|Contract value|Claimed to date|Previous claims|This claim",
		"Substructure|$100,000.00|$60,000.00|$40,000.00|$20,000.00",
		"Superstructure|$50,000.00|$10,000.00|$0.00|$10,000.00",
		"Roof works|$30,000.00|$5,000.00|$0.00|$5,000.00",
		"V01 Rock excavation|$11,000.00|$11,000.00|$0.00|$11,000.00",
		"Total|$191,000.00|$86,000.00|$40,000.00|$46,000.00",
	)
}

func TestClaimRowsMapByRuleAndOneFanOut(t *testing.T) {
	ask := &fakeAsk{}
	out, err := Read(context.Background(), ask, Thresholds{}, "claim-4.pdf", pdfClaim(), claimContext(), dropped(), time.Now(), "")
	if err != nil || out.Claim == nil {
		t.Fatalf("read: %v %+v", err, out)
	}
	c := out.Claim
	if !c.TableFound || c.PrintedGross == nil || *c.PrintedGross != 4600000 {
		t.Fatalf("table: %+v", c)
	}
	want := []struct{ kind, line, variation, label string }{
		{RowContract, "L-sub", "", "Substructure"}, {RowContract, "L-sup", "", "Superstructure"},
		{RowContract, "", "", "Roof works"}, {RowVariation, "", "V-1", "V01 Rock excavation"}, {RowRetention, "", "", "Retention"},
	}
	if len(c.Rows) != len(want) {
		t.Fatalf("rows: %+v", c.Rows)
	}
	for i, w := range want {
		r := c.Rows[i]
		if r.Kind != w.kind || r.LineID != w.line || r.VariationID != w.variation || r.Label != w.label {
			t.Fatalf("row %d: %+v, want %+v", i, r, w)
		}
	}
	if r := c.Rows[0]; *r.ToDate != 6000000 || *r.Previous != 4000000 || *r.ThisClaim != 2000000 {
		t.Fatalf("amounts as printed: %+v", r)
	}
	// One call, claim-1, asking only the row rules left open.
	if ask.got.QuestionVersion != ClaimQuestionVersion {
		t.Fatalf("version %q", ask.got.QuestionVersion)
	}
	var rowQs []string
	for id := range ask.got.Questions {
		if strings.HasPrefix(id, "row.") {
			rowQs = append(rowQs, id)
		}
		if id == FieldKind || id == FieldLine {
			t.Fatalf("asked %s on a claim", id)
		}
	}
	if len(rowQs) != 1 || rowQs[0] != "row.2" {
		t.Fatalf("row questions %v", rowQs)
	}
	if out.Kind != KindProgressClaim {
		t.Fatalf("kind %q", out.Kind)
	}
}

// With no claim-1 cut-off, Jev's pick is stored blank with its confidence
// and never placed; a calibrated cut-off places it.
func TestClaimPicksWithheldUntilCalibrated(t *testing.T) {
	ask := &fakeAsk{res: jev.Result{Answers: map[string]jev.Answer{"row.2": {Type: jev.TypeChoice, Choice: "l3", Confidence: conf(0.97)}}}}
	out, _ := Read(context.Background(), ask, Thresholds{}, "claim-4.pdf", pdfClaim(), claimContext(), dropped(), time.Now(), "")
	if r := out.Claim.Rows[2]; r.LineID != "" || r.MappedBy != DecidedByJev || r.Band != BandBlank {
		t.Fatalf("uncalibrated pick must stay blank: %+v", r)
	}
	lo, hi := 2, 255
	th := Thresholds{QuestionVersion: ClaimQuestionVersion, Questions: map[string][]Threshold{"row.2": {{Options: &lo, MaxOptions: &hi, Green: conf(0.9), Amber: conf(0.5)}}}}
	out, _ = Read(context.Background(), ask, Thresholds{}, "claim-4.pdf", pdfClaim(), claimContext(), dropped(), time.Now(), "", th)
	if r := out.Claim.Rows[2]; r.LineID != "L-fin" || r.Band != BandGreen {
		t.Fatalf("calibrated pick: %+v", r)
	}
	// The invoice-1 cut-offs never band a claim's answers.
	if _, ok := calibrated("row.2").BandFor(ClaimQuestionVersion, "row.2", conf(0.99), 6); ok {
		t.Fatal("invoice-1 cut-offs applied to claim-1")
	}
}

// Jev down: grey, and the rule values stand.
func TestClaimGreyKeepsRuleValues(t *testing.T) {
	ask := &fakeAsk{err: errors.New("timeout")}
	out, _ := Read(context.Background(), ask, Thresholds{}, "claim-4.pdf", pdfClaim(), claimContext(), dropped(), time.Now(), "")
	if !out.Grey || out.Claim.Rows[0].LineID != "L-sub" || out.Claim.Rows[2].Band != BandGrey {
		t.Fatalf("grey: %+v", out.Claim.Rows)
	}
}

// Rules settle every row: no call at all.
func TestClaimNoCallWhenRulesSettleAll(t *testing.T) {
	text := page("Item|Claimed to date|Previous|This claim", "Substructure|$60,000.00|$40,000.00|$20,000.00", "Retention|-$1,000.00|-$500.00|-$500.00")
	users := dropped()
	users[FieldNumber] = Decision{Field: FieldNumber, Value: "4", Band: BandGreen, DecidedBy: DecidedByUser}
	users[FieldDate] = Decision{Field: FieldDate, Value: "2026-11-01", Band: BandGreen, DecidedBy: DecidedByUser}
	users[FieldTotal] = Decision{Field: FieldTotal, Value: "2145000", Band: BandGreen, DecidedBy: DecidedByUser}
	users[FieldGST] = Decision{Field: FieldGST, Value: "195000", Band: BandGreen, DecidedBy: DecidedByUser}
	users[FieldABN] = Decision{Field: FieldABN, Value: "", Band: BandGreen, DecidedBy: DecidedByUser}
	ask := &fakeAsk{}
	out, err := Read(context.Background(), ask, Thresholds{}, "c.pdf", text, claimContext(), users, time.Now(), "")
	if err != nil || out.Asked || ask.got.Questions != nil {
		t.Fatalf("asked %v: %+v", out.Asked, ask.got.Questions)
	}
	if r := out.Claim.Rows[1]; r.Kind != RowRetention || *r.ThisClaim != -50000 {
		t.Fatalf("retention row: %+v", r)
	}
}

// A spreadsheet claim carries its cells' rows and columns.
func TestClaimFromASpreadsheet(t *testing.T) {
	text := identity.Text{TextLayer: true, Format: "xlsx"}
	cell := func(row, col int, s string) {
		text.Runs = append(text.Runs, identity.Run{Text: s, Source: identity.Source{Sheet: "Claim", Row: row, Col: col}})
	}
	for i, h := range []string{"Trade", "Value to date", "Less previous", "This period"} {
		cell(3, i+1, h)
	}
	cell(4, 1, "Superstructure")
	cell(4, 2, "10000")
	cell(4, 3, "2500")
	cell(4, 4, "7500")
	cell(5, 1, "Subtotal")
	cell(5, 4, "7500")
	out, _ := Read(context.Background(), &fakeAsk{}, Thresholds{}, "claim.xlsx", text, claimContext(), dropped(), time.Now(), "")
	if r := out.Claim.Rows[0]; out.Claim.Rows == nil || r.LineID != "L-sup" || *r.ToDate != 1000000 || *r.ThisClaim != 750000 || *out.Claim.PrintedGross != 750000 {
		t.Fatalf("spreadsheet rows: %+v", out.Claim)
	}
}

// The words-plus-ABN rule: "progress claim" printed and the supplier
// settled by rule to a live works contract makes it a claim before any call.
func TestClaimByWordsAndSupplierRule(t *testing.T) {
	pc := claimContext()
	pc.Packages[0].ABN = "58359914843"
	text := page("PROGRESS CLAIM", "ABN: 58359914843", "Item|Claimed to date|Previous|This claim", "Substructure|$60,000.00|$40,000.00|$20,000.00")
	out, _ := Read(context.Background(), &fakeAsk{}, Thresholds{}, "pc.pdf", text, pc, nil, time.Now(), "")
	if out.Claim == nil || out.Kind != KindProgressClaim {
		t.Fatalf("not read as a claim: kind %q", out.Kind)
	}
	pc.Contracts = nil
	out, _ = Read(context.Background(), &fakeAsk{}, Thresholds{}, "pc.pdf", text, pc, nil, time.Now(), "")
	if out.Claim != nil {
		t.Fatal("no live contract: it stays an invoice")
	}
}

// No table: rows from the contract's lines, amounts unknown.
func TestClaimWithoutATable(t *testing.T) {
	out, _ := Read(context.Background(), &fakeAsk{}, Thresholds{}, "c.pdf", page("Progress claim 5", "Amount $10,000.00"), claimContext(), dropped(), time.Now(), "")
	if out.Claim.TableFound || len(out.Claim.Rows) != 4 || out.Claim.Rows[0].ToDate != nil || out.Claim.Rows[3].Kind != RowRetention {
		t.Fatalf("contract rows: %+v", out.Claim.Rows)
	}
}
