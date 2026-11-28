package store_test

import (
	"context"
	"testing"
	"time"

	"sitewise/internal/store"
)

// Every firm, label and figure in this test is invented.

// An award on a tender-read column seeds the contract's breakdown in the
// award's own transaction (docs/plans/2026-11-26-payment-schedules.md,
// section 5.2). A cost line whose rows hold the firm's own base lines and
// provisional sums is split into them, because they add up to its contract
// value. A cost line that is one base line stays one line.
func TestAwardSeedsTheBreakdownFromTheTender(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	pkg := headContract(t, s)
	sub := tenderSubmission(t, s, pkg, 71)
	in, err := s.TenderInput(ctx, orgA, sub)
	if err != nil || len(in.Rows) < 3 {
		t.Fatalf("%d rows %v", len(in.Rows), err)
	}
	commit(t, s, sub, tenderReadingOf(t, s, sub,
		rl("Demolition", "$60,000.00", "base", "sum", in.Rows[1].Key),
		rl("Rock, 40 tonne at $750", "$30,000.00", "ps", "sum", in.Rows[1].Key),
		rl("Door hardware", "$10,000.00", "pc", "sum", in.Rows[1].Key),
		rl("Frame", "$450,000.00", "base", "sum", in.Rows[2].Key),
		rl("Tender sum", "$550,000.00", "total", "inside", "")))
	if err := s.EditSubmission(ctx, orgA, projectA, sub, store.SubmissionEdit{Firm: &store.FirmFields{Name: "Invented Builders Pty Ltd"}}); err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	var lines []store.AwardLine
	for _, it := range table.Items {
		if cell, ok := table.Columns[0].Cells[it.Key]; ok && it.Base && cell.Cents != nil {
			lines = append(lines, store.AwardLine{Label: it.Label, Keys: []string{it.Key}})
		}
	}
	if _, err := s.Award(ctx, orgA, projectA, pkg, userA, store.AwardInput{SubmissionID: sub, Lines: lines}); err != nil {
		t.Fatalf("award: %v", err)
	}
	contracts, _, err := s.VariationRegister(ctx, orgA, projectA)
	if err != nil || len(contracts) != 1 {
		t.Fatalf("%d contracts: %v", len(contracts), err)
	}
	c := contracts[0]
	if len(c.Gaps) != 0 {
		t.Fatalf("a seeded breakdown adds up to its cost lines: %+v", c.Gaps)
	}
	type want struct {
		section, label, source string
		amount                 int64
	}
	var got []want
	for _, b := range c.Breakdown {
		if !b.Stored || b.Amount == nil {
			t.Fatalf("a seeded line is stored with its sum: %+v", b)
		}
		got = append(got, want{b.Section, b.Label, b.Source, *b.Amount})
	}
	split := []want{
		{"works", "Demolition", "tender", 6000000},
		{"provisional", "Rock, 40 tonne at $750", "tender", 3000000},
		{"provisional", "Door hardware", "tender", 1000000},
	}
	if len(got) != 4 {
		t.Fatalf("breakdown: %+v", got)
	}
	for i, w := range split {
		if got[i] != w {
			t.Errorf("line %d is %+v, want %+v", i, got[i], w)
		}
	}
	// One base line is the cost line itself: it keeps the cost line's
	// label and stands for the whole of it.
	if last := got[3]; last.section != "works" || last.source != "award" || last.amount != 45000000 || last.label != c.Lines[1].Label {
		t.Errorf("the unsplit line is %+v (cost line %q)", last, c.Lines[1].Label)
	}
}
