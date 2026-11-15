package store_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// Every firm, label and figure in these tests is invented.

func headContract(t *testing.T, s *store.Store) string {
	t.Helper()
	row, ok := procurement.DisciplinePackage("trade.main_works", "Main Works")
	if !ok {
		t.Fatal("missing builder discipline")
	}
	pkg, err := s.CreateDisciplinePackage(context.Background(), orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.ID
}

func liveRows(t *testing.T, s *store.Store, pkg string) map[string]string {
	t.Helper()
	table, err := s.ReadPriceTable(context.Background(), orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, it := range table.Items {
		out[it.Key] = it.Label
	}
	return out
}

func TestHeadContractWithTheSwitchOffIsUnchanged(t *testing.T) {
	s, _, _ := workStore(t)
	rows := liveRows(t, s, headContract(t, s))
	if _, ok := rows["stage:lump_sum"]; !ok {
		t.Fatalf("no quoted-price row: %v", rows)
	}
	for _, label := range rows {
		if label == "Preliminaries and site establishment" {
			t.Fatal("Main Works rows appeared with the switch off")
		}
	}
}

func TestHeadContractHasMainWorksRowsWhenTheSwitchIsOn(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	pkg := headContract(t, s)
	rows := liveRows(t, s, pkg)
	if _, ok := rows["stage:lump_sum"]; ok {
		t.Fatal("a new head contract kept the quoted-price row with the switch on")
	}
	have := map[string]bool{}
	for _, label := range rows {
		have[label] = true
	}
	if !have["Preliminaries and site establishment"] || !have["Substructure"] {
		t.Fatalf("Main Works rows are missing: %v", rows)
	}
	// The pass is offered those rows.
	sub := tenderSubmission(t, s, pkg, 61)
	in, err := s.TenderInput(ctx, orgA, sub)
	if err != nil {
		t.Fatal(err)
	}
	offered := map[string]bool{}
	for _, r := range in.Rows {
		offered[r.Label] = true
		if r.Key == "stage:lump_sum" || strings.HasPrefix(r.Key, "tender:") {
			t.Fatalf("row %s was offered to the model", r.Key)
		}
	}
	if !offered["Preliminaries and site establishment"] {
		t.Fatalf("offered %v", offered)
	}
}

func TestHeadContractKeepsTheLumpSumRowWhileItIsUsed(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := headContract(t, s)
	liveRows(t, s, pkg) // the table has been opened once, so its row exists
	old := tenderSubmission(t, s, pkg, 62)
	if err := s.SetSubmissionCell(ctx, orgA, projectA, old, "stage:lump_sum", store.CellInput{State: "priced", Amount: "500000.00"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	rows := liveRows(t, s, pkg)
	if _, ok := rows["stage:lump_sum"]; !ok {
		t.Fatalf("the confirmed quoted-price row was retired: %v", rows)
	}
	// The old column is still judged on that row alone.
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if c := table.Columns[0]; c.Totals.BaseTotal == nil || *c.Totals.BaseTotal != 50000000 || c.Totals.Unread != 0 {
		t.Fatalf("old builder totals changed: %+v", c.Totals)
	}
	// Once that submission is read by the pass, nothing rests on the row and it goes.
	in, err := s.TenderInput(ctx, orgA, old)
	if err != nil {
		t.Fatal(err)
	}
	prelims := ""
	for _, r := range in.Rows {
		if r.Label == "Preliminaries and site establishment" {
			prelims = r.Key
		}
	}
	commit(t, s, old, tenderReadingOf(t, s, old, rl("Preliminaries", "$500,000.00", "base", "sum", prelims), rl("Total", "$500,000.00", "total", "inside", "")))
	if _, ok := liveRows(t, s, pkg)["stage:lump_sum"]; ok {
		t.Fatal("the quoted-price row outlived its last old-style submission")
	}
}

func TestAwardATenderReadHeadContract(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	pkg := headContract(t, s)
	sub := tenderSubmission(t, s, pkg, 63)
	in, err := s.TenderInput(ctx, orgA, sub)
	if err != nil || len(in.Rows) < 3 {
		t.Fatalf("%d rows %v", len(in.Rows), err)
	}
	commit(t, s, sub, tenderReadingOf(t, s, sub,
		rl("Site set-up", "$96,000.00", "base", "sum", in.Rows[1].Key),
		rl("Frame", "$450,000.00", "base", "sum", in.Rows[2].Key),
		rl("Sundries", "$4,000.00", "base", "sum", ""),
		rl("Tender sum", "$550,000.00", "total", "inside", "")))
	if err := s.EditSubmission(ctx, orgA, projectA, sub, store.SubmissionEdit{Firm: &store.FirmFields{Name: "Invented Builders Pty Ltd"}}); err != nil {
		t.Fatal(err)
	}
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if !table.Builder || len(table.Tender.Columns) != 1 {
		t.Fatalf("builder %v, tender columns %d", table.Builder, len(table.Tender.Columns))
	}
	var lines []store.AwardLine
	for _, it := range table.Items {
		if cell, ok := table.Columns[0].Cells[it.Key]; ok && it.Base && cell.Cents != nil {
			lines = append(lines, store.AwardLine{Label: it.Label, Keys: []string{it.Key}})
		}
	}
	award := store.AwardInput{SubmissionID: sub, Lines: lines}
	// No separate confirmation of a quoted total, and no column check, is
	// needed: pressing Award is the manager's acceptance (owner, 15 Nov 2026).
	view, err := s.PreviewAward(ctx, orgA, projectA, pkg, award)
	if err != nil || view.Total != 55000000 || len(view.Lines) != 3 {
		t.Fatalf("total %d lines %d: %v", view.Total, len(view.Lines), err)
	}
	hasSystem := false
	for _, l := range lines {
		hasSystem = hasSystem || l.Keys[0] == tender.SystemRow
	}
	if !hasSystem {
		t.Fatal("Not broken down is not among the award lines")
	}
	// The award itself: a works package takes contract lines in the cost
	// plan (works lines under the package, naming no system), the contract
	// counts under construction, and the award can be undone.
	awarded, err := s.Award(ctx, orgA, projectA, pkg, userA, award)
	if err != nil || awarded.Total != 55000000 {
		t.Fatalf("award: total %d: %v", awarded.Total, err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	contract := 0
	for _, it := range plan.Items {
		if it.PackageID == pkg && it.Posting {
			if it.LineKind != "works" || it.SystemID != "" || it.WorkItemID != "" {
				t.Fatalf("award line %+v", it.Content)
			}
			contract++
		}
	}
	if contract != 3 {
		t.Fatalf("%d contract lines", contract)
	}
	live, err := s.ReadAward(ctx, orgA, projectA, pkg)
	if err != nil || live == nil || live.Total != 55000000 || len(live.Lines) != 3 {
		t.Fatalf("read award %+v: %v", live, err)
	}
	if err := s.UndoAward(ctx, orgA, projectA, pkg, userA); err != nil {
		t.Fatalf("undo award: %v", err)
	}
	if live, err := s.ReadAward(ctx, orgA, projectA, pkg); err != nil || live != nil {
		t.Fatalf("after undo %+v: %v", live, err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	for _, it := range plan.Items {
		if it.PackageID == pkg && it.LineKind == "works" && it.SystemID == "" && it.WorkItemID == "" {
			t.Fatalf("a contract line outlived the undo: %+v", it.Content)
		}
	}
}
