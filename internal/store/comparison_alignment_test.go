package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

func TestComparisonAlignmentPreservesMoneyAndUndo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Test firm"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, it := range table.Items {
		if it.Base {
			keys = append(keys, it.Key)
		}
	}
	if len(keys) < 2 {
		t.Fatal("need two base rows")
	}
	for _, k := range keys {
		if err = s.SetSubmissionCell(ctx, orgA, projectA, id, k, store.CellInput{State: "priced", Amount: "100.00"}); err != nil {
			t.Fatal(err)
		}
	}
	read := func() store.PriceTable {
		t.Helper()
		v, e := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	table = read()
	c := table.Columns[0]
	before := c.Totals.KnownBase
	in := store.AlignmentInput{Target: keys[0], PriceIDs: []string{c.Cells[keys[0]].ID, c.Cells[keys[1]].ID}, Version: c.Submission.Version}
	if e := s.AlignSubmission(ctx, orgB, projectA, id, in); !errors.Is(e, store.ErrNotFound) {
		t.Fatalf("org: %v", e)
	}
	if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Target: keys[0], PriceIDs: []string{c.Cells[keys[1]].ID}, Version: c.Submission.Version}); !errors.Is(e, store.ErrConflict) {
		t.Fatalf("overwrite: %v", e)
	}
	if err = s.AlignSubmission(ctx, orgA, projectA, id, in); err != nil {
		t.Fatal(err)
	}
	table = read()
	c = table.Columns[0]
	if c.Totals.KnownBase != before || *c.Cells[keys[0]].Cents != 20000 || c.Cells[keys[1]].State != "included" || !c.UndoAlignment {
		t.Fatalf("combined: %+v", c)
	}
	if e := s.AlignSubmission(ctx, orgA, projectA, id, in); !errors.Is(e, store.ErrConflict) {
		t.Fatalf("stale: %v", e)
	}
	if err = s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Undo: true, Version: c.Submission.Version}); err != nil {
		t.Fatal(err)
	}
	table = read()
	c = table.Columns[0]
	if *c.Cells[keys[0]].Cents != 10000 || *c.Cells[keys[1]].Cents != 10000 || c.UndoAlignment {
		t.Fatal("undo did not restore original fees")
	}
	if err = s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Key: keys[0], Label: "Assessment and approval", Section: "option"}); err != nil {
		t.Fatal(err)
	}
	table = read()
	if table.Columns[0].Totals.KnownBase != before-10000 {
		t.Fatal("option entered base total")
	}
	if err = s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Label: "Additional certification", Section: "base"}); err != nil {
		t.Fatal(err)
	}
	table = read()
	if table.Columns[0].Totals.BaseTotal != nil {
		t.Fatal("unpriced new base row must remain unknown")
	}
	if err = s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Label: "Additional inspection", Section: "rate", Unit: "inspection"}); err != nil {
		t.Fatal(err)
	}
	table = read()
	var rate string
	for _, it := range table.Items {
		if it.Label == "Additional inspection" {
			rate = it.Key
			if it.Unit != "inspection" || it.Base {
				t.Fatal("rate unit lost")
			}
		}
	}
	if rate == "" {
		t.Fatal("missing rate")
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, rate, store.CellInput{State: "priced", Amount: "1250.00", Note: "Per additional inspection; no quantity assumed"}); err != nil {
		t.Fatal(err)
	}
	table = read()
	c = table.Columns[0]
	if c.Totals.KnownBase != before-10000 {
		t.Fatal("rate counted in base")
	}
	if _, e := s.PreviewAward(ctx, orgA, projectA, pkg.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Rate", Keys: []string{rate}}}}); !errors.Is(e, store.ErrInvalidInput) {
		t.Fatalf("unit rate awarded as lump sum: %v", e)
	}
	if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Target: keys[0], PriceIDs: []string{c.Cells[rate].ID}, Version: c.Submission.Version}); e == nil {
		t.Fatal("rate moved into fee")
	}
	if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Exclude: true, PriceIDs: []string{c.Cells[rate].ID}, Version: c.Submission.Version}); e != nil {
		t.Fatal(e)
	}
	c = read().Columns[0]
	if c.Cells[rate].State != "excluded" {
		t.Fatal("set aside failed")
	}
	if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Undo: true, Version: c.Submission.Version}); e != nil {
		t.Fatal(e)
	}
	c = read().Columns[0]
	if *c.Cells[rate].Cents != 125000 || c.Cells[rate].Basis == "" {
		t.Fatal("undo lost rate or note")
	}
	// A gate for the actual database-backed reads and writes (20 samples).
	var reads, writes, alignments []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		read()
		reads = append(reads, time.Since(start))
		start = time.Now()
		if err = s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Key: keys[0], Label: "Assessment and approval", Section: "option"}); err != nil {
			t.Fatal(err)
		}
		writes = append(writes, time.Since(start))
		current := read().Columns[0]
		start = time.Now()
		if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Target: keys[0], PriceIDs: []string{current.Cells[keys[0]].ID, current.Cells[keys[1]].ID}, Version: current.Submission.Version}); e != nil {
			t.Fatal(e)
		}
		alignments = append(alignments, time.Since(start))
		current = read().Columns[0]
		if e := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Undo: true, Version: current.Submission.Version}); e != nil {
			t.Fatal(e)
		}
	}
	for name, samples := range map[string][]time.Duration{"read": reads, "write": writes, "alignment": alignments} {
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p50, p90 := samples[9], samples[17]
		limit50, limit90 := 150*time.Millisecond, 300*time.Millisecond
		if name != "read" {
			limit50, limit90 = 100*time.Millisecond, 250*time.Millisecond
		}
		t.Logf("%s p50=%v p90=%v", name, p50, p90)
		if p50 > limit50 || p90 > limit90 {
			t.Fatalf("%s exceeded budget", name)
		}
	}
	if _, err = s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Fee", Keys: keys[:1]}}, Total: "100.00"}); err != nil {
		t.Fatal(err)
	}
	if e := s.SetComparisonRow(ctx, orgA, projectA, pkg.ID, store.ComparisonRowInput{Key: keys[0], Label: "Changed", Section: "base"}); !errors.Is(e, store.ErrConflict) {
		t.Fatalf("award row lock: %v", e)
	}
	if e := s.SetSubmissionCell(ctx, orgA, projectA, id, keys[0], store.CellInput{State: "priced", Amount: "1"}); !errors.Is(e, store.ErrConflict) {
		t.Fatalf("award cell lock: %v", e)
	}
}
