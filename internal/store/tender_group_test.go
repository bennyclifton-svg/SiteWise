package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// A combined engagement on the tender tables: the pass is offered the
// group's rows, a reading projects cells onto each member's rows, and the
// award puts "Not broken down" under the engagement itself. Every firm and
// figure here is invented.
func TestTenderOnACombinedEngagement(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	group, members := combinedGroup(t, s, "consultant.structural", "consultant.civil")
	table, err := s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	st, cv := memberKey(t, table, members[0].ID), memberKey(t, table, members[1].ID)
	stKeys, cvKeys := groupKeys(t, table, st), groupKeys(t, table, cv)
	if len(stKeys) == 0 || len(cvKeys) == 0 {
		t.Fatalf("group rows %v %v", stKeys, cvKeys)
	}
	sub := tenderSubmission(t, s, group.ID, 71)

	// The rows offered to the pass are the group's: each starts with its member's key.
	in, err := s.TenderInput(ctx, orgA, sub)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range in.Rows {
		if !strings.HasPrefix(r.Key, st+":") && !strings.HasPrefix(r.Key, cv+":") && !strings.HasPrefix(r.Key, "combined:") {
			t.Fatalf("row %s belongs to no member", r.Key)
		}
	}

	commit(t, s, sub, tenderReadingOf(t, s, sub,
		rl("Structural concept", "$5,000.00", "base", "sum", stKeys[0]),
		rl("Civil concept", "$3,000.00", "base", "sum", cvKeys[0]),
		rl("Travel", "$500.00", "base", "sum", ""),
		rl("Total", "$8,500.00", "total", "inside", "")))
	sum, by := pricedCells(t, s, group.ID, sub)
	if sum != 850000 || by[stKeys[0]] != 500000 || by[cvKeys[0]] != 300000 || by[tender.SystemRow] != 50000 {
		t.Fatalf("cells %d %v", sum, by)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if table.Group == nil || len(table.Tender.Columns) != 1 || !table.Tender.Rows[len(table.Tender.Rows)-1].System {
		t.Fatalf("group %v, tender columns %d", table.Group != nil, len(table.Tender.Columns))
	}

	if err := s.EditSubmission(ctx, orgA, projectA, sub, store.SubmissionEdit{Firm: &store.FirmFields{Name: "Invented Engineers Pty Ltd"}}); err != nil {
		t.Fatal(err)
	}
	lines := []store.AwardLine{
		{Label: "Structural concept", Keys: stKeys[:1]},
		{Label: "Civil concept", Keys: cvKeys[:1]},
		{Label: "Not broken down", Keys: []string{tender.SystemRow}},
	}
	award := store.AwardInput{SubmissionID: sub, Lines: lines, Total: "8500.00"}
	if _, err := s.PreviewAward(ctx, orgA, projectA, group.ID, award); !errors.Is(err, store.ErrInvalidInput) || !strings.Contains(err.Error(), "Check this firm's column") {
		t.Fatalf("unchecked group column: %v", err)
	}
	check := top("check_column")
	check.Submission, check.Checked = sub, tbp(true)
	mustOps(t, s, group.ID, check)
	// Leaving "Not broken down" out would drop part of the tender sum.
	if _, err := s.PreviewAward(ctx, orgA, projectA, group.ID, store.AwardInput{SubmissionID: sub, Lines: lines[:2]}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("partial award %v", err)
	}
	// Each member's line sits under its discipline; the rest under the engagement.
	preview, err := s.PreviewAward(ctx, orgA, projectA, group.ID, award)
	if err != nil {
		t.Fatal(err)
	}
	under := map[string]string{}
	for _, l := range preview.Lines {
		under[l.Label] = l.Under
	}
	if under["Structural concept"] == "" || under["Civil concept"] == "" || under["Structural concept"] == under["Civil concept"] ||
		under["Not broken down"] == under["Structural concept"] || under["Not broken down"] == under["Civil concept"] {
		t.Fatalf("lines sit under %v", under)
	}
	v, err := s.Award(ctx, orgA, projectA, group.ID, userA, award)
	if err != nil {
		t.Fatal(err)
	}
	if v.Total != 850000 || len(v.Lines) != 3 {
		t.Fatalf("award %+v", v)
	}
}
