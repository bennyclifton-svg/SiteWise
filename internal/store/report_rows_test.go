package store_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"sitewise/internal/delivery"
	"sitewise/internal/export"
	"sitewise/internal/files"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

func reportTable(t *testing.T, d store.ReportDraft, id string) *reports.Table {
	t.Helper()
	for _, s := range d.Sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID == id {
				return b.Table
			}
		}
	}
	t.Fatalf("no table %s in %s", id, blockIDs(d))
	return nil
}

func rowFacts(tb *reports.Table, key string) (reports.RowFacts, []string) {
	for i, f := range tb.Meta {
		if f.Key == key || f.ID == key {
			return f, tb.Rows[i]
		}
	}
	return reports.RowFacts{}, nil
}

func text(s string) *string { return &s }

// Slice 3 of the project report plan: rows the PM owns survive Update; a
// typed-over cell whose source moves holds issue until kept; the issued
// PDF is the owner's format; Undo takes a row change back.
func TestProjectReportRowsSurviveUpdateAndHoldIssue(t *testing.T) {
	ctx := context.Background()
	s, _, _ := monthlyFixture(t)
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if d.PeriodTo == nil || d.PeriodLabel == nil || !strings.HasPrefix(*d.PeriodLabel, "To ") {
		t.Fatalf("first period %v %v", d.PeriodTo, d.PeriodLabel)
	}
	site := reportTable(t, d, "site")
	if _, row := rowFacts(site, "site:open"); row == nil || !strings.Contains(row[1], "Latent ground conditions") {
		t.Fatalf("open issues: %v", row)
	}
	// A row of the PM's own, and their words over a generated cell.
	d, own, err := s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "approvals", Cells: map[string]*string{"0": text("Sydney Water"), "1": text("Section 73")}})
	if err != nil {
		t.Fatal(err)
	}
	d, _, err = s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "site", Key: "site:open", Cells: map[string]*string{"1": text("Ground risk is with the builder.")}})
	if err != nil {
		t.Fatal(err)
	}
	// A wrong table, column or version is refused.
	if _, _, err := s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "nope", Cells: map[string]*string{"0": text("x")}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown table: %v", err)
	}
	if _, _, err := s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "site", Key: "site:open", Cells: map[string]*string{"9": text("x")}}); !errors.Is(err, store.ErrReportRow) {
		t.Fatalf("unknown column: %v", err)
	}
	if _, _, err := s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version-1, store.RowWrite{Table: "site", Key: "site:open", Cells: map[string]*string{"1": text("x")}}); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	// Another org cannot see or write the report.
	if _, _, err := s.WriteReportRow(ctx, orgB, r.ID, userB, 0, store.RowWrite{Table: "site", Key: "site:open", Cells: map[string]*string{"1": text("x")}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign write: %v", err)
	}

	// Update keeps both, word for word.
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if f, row := rowFacts(reportTable(t, d, "approvals"), own); row == nil || row[0] != "Sydney Water" || f.Kind != reports.RowOwn {
		t.Fatalf("own row after Update: %+v %v", f, row)
	}
	f, row := rowFacts(reportTable(t, d, "site"), "site:open")
	if row[1] != "Ground risk is with the builder." || !f.Typed[1] || f.Flag != "" || !strings.Contains(f.Base[1], "Latent ground conditions") {
		t.Fatalf("typed cell after Update: %+v %v", f, row)
	}

	// The record under it moves: the row is flagged and issue waits.
	target := "2020-02-01"
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "risk", Title: "Confirm crane permit", Status: "open", TargetDate: &target, Details: json.RawMessage(`{}`)}}); err != nil {
		t.Fatal(err)
	}
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := rowFacts(reportTable(t, d, "site"), "site:open"); f.Flag != "moved" || !strings.Contains(f.Base[1], "Confirm crane permit") {
		t.Fatalf("moved: %+v", f)
	}
	if _, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-31", AcceptStale: true, StaleReason: "Fixture"}, blobs); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("issue with a moved cell: %v", err)
	}
	// Keep mine: the flag clears and the words stay.
	d, _, err = s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "site", Key: "site:open", Keep: []string{"1"}})
	if err != nil {
		t.Fatal(err)
	}
	if f, row := rowFacts(reportTable(t, d, "site"), "site:open"); f.Flag != "" || row[1] != "Ground risk is with the builder." {
		t.Fatalf("kept: %+v %v", f, row)
	}

	// Hide a placeholder; move the PM's row up; then give the hidden one back.
	ap := reportTable(t, d, "approvals")
	d, hide, err := s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "approvals", Key: ap.Meta[0].Key, Hidden: boolp(true)})
	if err != nil {
		t.Fatal(err)
	}
	if reportTable(t, d, "approvals").Hidden != 1 {
		t.Fatal("hidden count")
	}
	before := indexOf(reportTable(t, d, "approvals"), own)
	if d, _, err = s.WriteReportRow(ctx, orgA, r.ID, userA, d.Version, store.RowWrite{Row: own, Move: "up"}); err != nil {
		t.Fatal(err)
	}
	if after := indexOf(reportTable(t, d, "approvals"), own); after != before-1 {
		t.Fatalf("moved from %d to %d", before, after)
	}
	if d, err = s.DeleteReportRow(ctx, orgA, r.ID, userA, hide, d.Version); err != nil {
		t.Fatal(err)
	}
	if reportTable(t, d, "approvals").Hidden != 0 {
		t.Fatal("given back")
	}

	// Undo takes a row change back, in the reports area.
	marked := store.WithChange(ctx, store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "reports"})
	d, extra, err := s.WriteReportRow(marked, orgA, r.ID, userA, d.Version, store.RowWrite{Table: "insurances", Cells: map[string]*string{"0": text("Professional indemnity")}})
	if err != nil {
		t.Fatal(err)
	}
	change, _ := store.ChangeOf(marked)
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, row := rowFacts(reportTable(t, *view.Draft, "insurances"), extra); row != nil {
		t.Fatal("undo left the row")
	}
	d = *view.Draft

	// Issue: the owner's format, with no page limit, and no review lists.
	issued, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-31", AcceptStale: true, StaleReason: "Fixture"}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Snapshot.RendererVersion != export.Version {
		t.Fatalf("renderer %q", issued.Snapshot.RendererVersion)
	}
	for _, sec := range issued.Snapshot.Sections {
		if sec.Internal || sec.ID == "attention" {
			t.Fatalf("a review list was issued: %s", sec.ID)
		}
	}
	f2, err := blobs.Open(issued.ExportSHA256)
	if err != nil {
		t.Fatal(err)
	}
	pdf, _ := io.ReadAll(f2)
	f2.Close()
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}

	// The next draft: the PM's own rows are there; the period runs on from
	// the last; a period that starts inside the last one is refused.
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, row := rowFacts(reportTable(t, d, "approvals"), own); row == nil {
		t.Fatal("own row did not carry")
	}
	if d.PeriodFrom == nil {
		t.Fatalf("second period %v", d.PeriodFrom)
	}
	if _, err := s.SetReportPeriod(ctx, orgA, r.ID, userA, "test", d.Version, store.ReportPeriod{From: "2020-01-01", To: *d.PeriodTo}); !errors.Is(err, store.ErrReportRow) {
		t.Fatalf("overlapping period: %v", err)
	}
	d, err = s.SetReportPeriod(ctx, orgA, r.ID, userA, "test", d.Version, store.ReportPeriod{From: *d.PeriodFrom, To: *d.PeriodTo, Label: "This fortnight"})
	if err != nil {
		t.Fatal(err)
	}
	if *d.PeriodLabel != "This fortnight" {
		t.Fatalf("label %q", *d.PeriodLabel)
	}
	if _, row := rowFacts(reportTable(t, d, "cover"), "cover:period"); row[1] != "This fortnight" {
		t.Fatalf("cover period %v", row)
	}
}

func boolp(b bool) *bool { return &b }

func indexOf(tb *reports.Table, id string) int {
	for i, f := range tb.Meta {
		if f.ID == id {
			return i
		}
	}
	return -1
}
