package store_test

import (
	"context"
	"errors"
	"sitewise/internal/latency"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
	"time"
)

func TestBuilderComparisonPreservesCorrectionsAndPreventsDoubleCounting(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, ok := procurement.DisciplinePackage("trade.main_works", "Main Works")
	if !ok {
		t.Fatal("missing builder discipline")
	}
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, id, pageText([]string{"Example Builder Pty Ltd"}, []string{"QUOTATION"}, []string{"Kitchen allowance $45,000.00"}, []string{"Total price including GST $228,000.00"}), answers{})
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !table.Builder || len(table.Columns) != 1 {
		t.Fatalf("missing builder review: %+v", table)
	}
	c := table.Columns[0]
	if c.Builder == nil || c.Builder.ExGST == nil || *c.Builder.ExGST != "207272.73" {
		t.Fatalf("lost total %+v", c.Builder)
	}
	for _, it := range table.Items {
		if it.Key == "reimbursables" || it.Kind == "rate" {
			t.Fatalf("consultant default remains %+v", it)
		}
	}
	if len(c.Others) == 0 {
		t.Fatal("missing allowance fixture")
	}
	if err := s.AlignSubmission(ctx, orgA, projectA, id, store.AlignmentInput{Target: "stage:lump_sum", Version: c.Submission.Version, PriceIDs: []string{c.Others[0].ID}}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("raw breakdown accepted: %v", err)
	}
	if _, err = s.ReadPriceTable(ctx, orgB, projectA, pkg.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross org: %v", err)
	}
	firm := store.FirmFields{Name: "Confirmed Builder"}
	if err = s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{Firm: &firm}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, "stage:lump_sum", store.CellInput{State: "priced", Amount: "205000.00", Note: "User correction"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
		if err != nil {
			t.Fatal(err)
		}
		c = table.Columns[0]
		if *c.Cells["stage:lump_sum"].Cents != 20500000 || *c.Builder.ExGST != "207272.73" {
			t.Fatal("source total overwrote user correction")
		}
	}
	award := store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Builder", Keys: []string{"stage:lump_sum"}}}}
	view, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, award)
	if err != nil || view.Total != 20500000 {
		t.Fatalf("preview %+v %v", view, err)
	}
	award.Lines[0].PriceIDs = []string{c.Others[0].ID}
	if _, err = s.PreviewAward(ctx, orgA, projectA, pkg.ID, award); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("double count allowed: %v", err)
	}
	budgets, err := latency.LoadBudgets("../../bench/budgets.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range budgets.Paths {
		if b.Name != "price_table" && b.Name != "submission_write" {
			continue
		}
		var samples []int64
		for i := 0; i < budgets.MinSamples; i++ {
			start := time.Now()
			if b.Name == "submission_write" {
				err = s.SetSubmissionCell(ctx, orgA, projectA, id, "stage:lump_sum", store.CellInput{State: "priced", Amount: "205000.00"})
			}
			if err == nil {
				_, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			samples = append(samples, time.Since(start).Microseconds())
		}
		code, report := latency.Gate(latency.Budgets{MinSamples: budgets.MinSamples, Paths: []latency.PathBudget{b}}, map[string][]int64{b.Name: samples}, true)
		p50, _ := latency.Percentile(samples, 0.5)
		p90, _ := latency.Percentile(samples, 0.9)
		t.Logf("%s p50=%.2fms p90=%.2fms (budget %.0f/%.0fms) %s", b.Name, float64(p50)/1000, float64(p90)/1000, float64(b.P50US)/1000, float64(b.P90US)/1000, report)
		if code != 0 {
			t.Fatal("builder budget exceeded")
		}
	}
}
