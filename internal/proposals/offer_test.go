package proposals

import (
	"strings"
	"testing"
)

// Fixtures are synthetic: they copy the shapes of real fee letters (task
// tables, unlabelled totals, repeated agreement pages, insurance schedules)
// with invented firms and amounts.

func fee(id, label, printed, amount, role string) QuotedFee {
	f := QuotedFee{ID: id, Label: label, Printed: printed, Role: role, Page: 2, Source: label + "  " + printed}
	if amount != "" {
		f.Amount = &amount
	}
	return f
}

const current = "Package / fee — scope awaiting alignment"

func lineIDs(o Offer) string {
	var ids []string
	for _, l := range o.Lines {
		ids = append(ids, l.Fee.ID)
	}
	return strings.Join(ids, ",")
}

func TestOfferTaskTableAddsToUnlabelledTotal(t *testing.T) {
	// Tasks then a bare "$7,000 + CFT" that is their sum; an insurance
	// schedule later in the letter.
	fees := []QuotedFee{
		fee("a1", "Task 1 - Planning advice", "$1,000", "1000.00", current),
		fee("a2", "Task 2 - Pre-DA meeting", "$500", "500.00", current),
		fee("a3", "Task 3 - Statement of Environmental Effects", "$5,000", "5000.00", current),
		fee("a4", "Task 4 - Lodgement", "$500", "500.00", current),
		fee("a5", "Task 6 - Meetings  Cost for Time", "$7,000", "7000.00", current),
		fee("a9", "123456ABC", "$20,000,000", "20000000.00", current),
	}
	fees[4].Source = "$7,000 + CFT"
	fees[5].Source, fees[5].Heading = "Public Liability Insurer  123456ABC  $20,000,000  01/03/2024", "INSURANCE DETAILS"
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a1,a2,a3,a4" || o.TotalID != "a5" || !o.Reconciled {
		t.Fatalf("lines %s total %q reconciled %v", lineIDs(o), o.TotalID, o.Reconciled)
	}
	if o.Hidden["a9"] == "" {
		t.Fatal("insurance cover shown as a fee")
	}
}

func TestOfferInfersExGSTFromItsTotal(t *testing.T) {
	// Stage amounts have no basis of their own; the total says excl. GST
	// and they add to it exactly.
	fees := []QuotedFee{
		fee("a1", "Stage 1: Lodgement of DA", "$6,000", "", current),
		fee("a2", "Stage 3: Post Lodgement", "$1,500", "", current),
		fee("a3", "Total (excl. GST)", "$7,500", "7500.00", RoleTotal),
		fee("a4", "suggest a", "$1,500", "", current),
		fee("a10", "2019-04 R3", "$20,000,000.00", "", current),
		fee("a11", "116558", "$1,234,567.89", "", current),
		fee("a50", "Agreed Fee (ex. GST)", "$ 7,500", "7500.00", current),
		fee("a51", "782300 Consulting Services", "$1,234,567.89", "", current),
	}
	fees[3].Suggestion = "Conditional / optional"
	fees[4].Source = "Public Liability  Insurer  2019-04 R3  $20,000,000.00"
	fees[5].Source = "Workers  icare  116558  $1,234,567.89"
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a1,a2" || o.TotalID != "a3" {
		t.Fatalf("lines %s total %q", lineIDs(o), o.TotalID)
	}
	if o.Lines[0].Amount == nil || *o.Lines[0].Amount != "6000.00" || *o.Lines[1].Amount != "1500.00" {
		t.Fatalf("ex GST not inferred %+v", o.Lines)
	}
	for _, id := range []string{"a10", "a11", "a50", "a51"} {
		if o.Hidden[id] == "" {
			t.Fatalf("%s not hidden: %v", id, o.Hidden)
		}
	}
	if o.Hidden["a4"] != "" {
		t.Fatal("an optional allowance must stay visible below the line")
	}
	// Inclusive totals are not used to infer ex-GST components.
	fees[2].Amount = ptr("6818.18")
	if o := ReadOffer(fees, nil); len(o.Lines) > 0 && o.Lines[0].Amount != nil {
		t.Fatal("inclusive total inferred an ex-GST stage")
	}
}

func ptr(s string) *string { return &s }

func TestOfferPrefersTheBetterLabelledRepeat(t *testing.T) {
	fees := []QuotedFee{
		fee("a1", "PROPOSAL", "$2,000.00", "2000.00", current),
		fee("a2", "PROPOSAL", "$3,000.00", "3000.00", current),
		fee("a3", "Total", "$5,000.00", "5000.00", RoleTotal),
		fee("a7", "Stage 1: Schematic Design", "2,000.00", "2000.00", current),
		fee("a8", "Stage 2: Detailed Design", "3,000.00", "3000.00", current),
		fee("a9", "TOTAL (excl. GST)", "5,000.00", "5000.00", RoleTotal),
		fee("a10", "Initial performance solution", "$2,000.00", "2000.00", "Conditional / optional fee"),
	}
	fees[0].Heading, fees[1].Heading = "PROPOSAL", "PROPOSAL"
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a7,a8" || o.TotalID != "a9" {
		t.Fatalf("lines %s total %q", lineIDs(o), o.TotalID)
	}
	for _, id := range []string{"a1", "a2", "a3"} {
		if o.Hidden[id] == "" {
			t.Fatalf("repeat %s shown", id)
		}
	}
	if o.Hidden["a10"] != "" {
		t.Fatal("an optional fee of equal amount is not a repeat")
	}
}

func TestOfferSingleTotalHidesRestatements(t *testing.T) {
	fees := []QuotedFee{
		fee("a1", "We advise that tasks 1 – 12 will incur a fee of", "$6,000", "6000.00", RoleTotal),
		fee("a4", "Director", "$1,200", "1200.00", current),
		fee("a9", "We accept the fee proposal contained herein", "$6,000", "6000.00", current),
		fee("a10", "to", "$6,600", "6600.00", current),
		fee("a11", "Commencement", "$1,980", "1980.00", current),
		fee("a12", "Upon release of letter of advice", "$4,620", "4620.00", current),
	}
	fees[1].Suggestion = "Conditional / optional"
	fees[4].Source = "Commencement…………… $1,980 (30%)"
	fees[5].Source = "Upon release of letter of advice……… $4,620 (70%)"
	o := ReadOffer(fees, nil)
	if !o.Single || o.TotalID != "a1" || len(o.Lines) != 0 {
		t.Fatalf("single total %+v", o)
	}
	for _, id := range []string{"a9", "a10", "a11", "a12"} {
		if o.Hidden[id] == "" {
			t.Fatalf("%s not hidden %v", id, o.Hidden)
		}
	}
	if o.Hidden["a4"] != "" {
		t.Fatal("optional fee hidden")
	}
}

func TestOfferWithoutTotalListsCurrentFees(t *testing.T) {
	fees := []QuotedFee{
		fee("a1", "▪ Undertake a high-level review of the draft development", "$4,000", "4000.00", current),
		fee("a2", "▪ Attend x1 site visit as required.", "$1,000", "1000.00", current),
		fee("a3", "(SEE) for client review.", "$8,000", "8000.00", current),
		fee("a6", "An experienced consultant will be responsible for", "$175", "175.00", current),
		fee("a7", "175", "$290", "290.00", current),
		fee("a8", "(ii) where the amount claimed by a party", "$50,000", "50000.00", current),
	}
	fees[0].Source = "Phase 1A -  ▪ Undertake a high-level review of the draft development  $4,000"
	fees[1].Suggestion = "Conditional / optional"
	fees[2].Source = "Prepare SEE  (SEE) for client review.  of $8,000"
	fees[3].Source = "Consultant/Senior  An experienced consultant will be responsible for  $175-$290"
	fees[4].Source = fees[3].Source
	fees[5].Source = "(ii) where the amount claimed by a party or determined by the expert is more than $50,000 provided that a party commences court proceedings"
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a1,a3" || o.TotalID != "" || o.Reconciled {
		t.Fatalf("lines %s total %q", lineIDs(o), o.TotalID)
	}
	if o.Lines[0].Label != "Phase 1A – Undertake a high-level review of the draft development" || o.Lines[1].Label != "Prepare SEE" {
		t.Fatalf("labels %q %q", o.Lines[0].Label, o.Lines[1].Label)
	}
	for _, id := range []string{"a6", "a7", "a8"} {
		if o.Hidden[id] == "" {
			t.Fatalf("%s not hidden", id)
		}
	}
	// A repeated agreement page lists the same fees again.
	again := append(fees[:0:0], fees...)
	again = append(again, fee("a20", "▪ Undertake a high-level review of the draft development", "$ 4,000.00", "4000.00", current))
	if o := ReadOffer(again, nil); lineIDs(o) != "a1,a3" || o.Hidden["a20"] == "" {
		t.Fatalf("repeat listed %s", lineIDs(o))
	}
}

func TestOfferSingleLineAndTotal(t *testing.T) {
	fees := []QuotedFee{
		fee("a1", "Traffic Engineering Services", "$12,000", "12000.00", current),
		fee("a2", "Total", "$12,000", "12000.00", RoleTotal),
	}
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a1" || o.TotalID != "a2" || !o.Reconciled {
		t.Fatalf("lines %s total %q", lineIDs(o), o.TotalID)
	}
}

func TestOfferUnknownBasisStaysUnknown(t *testing.T) {
	fees := []QuotedFee{
		fee("a1", "Prepare Traffic Report", "$5,000", "", current),
		fee("a2", "Traffic Counts", "$3,000", "", current),
	}
	o := ReadOffer(fees, nil)
	if lineIDs(o) != "a1,a2" || o.Lines[0].Amount != nil {
		t.Fatalf("unknown basis %+v", o.Lines)
	}
}
