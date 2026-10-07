package store_test

import (
	"context"
	"encoding/json"
	"sitewise/internal/files"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/store"
	"strings"
	"testing"
)

func TestDisclosedBudgetReportHistoryIgnoresPresentationOnlyChanges(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineering", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	in := ci("Engineering fee")
	in.LineKind = "fee"
	in.Category = ""
	in.PackageID = pkg.ID
	in.PackageStageID = pkg.Stages[0].ID
	plan, err := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if err != nil {
		t.Fatal(err)
	}
	lineID := plan.Items[0].ID
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	// A real generated pricing block is enough to exercise disclosure/history;
	// exclude unrelated draft catalogue clauses without promoting them to reviewed.
	var pricing reports.Block
	for _, section := range draft.Sections {
		for _, block := range section.Blocks {
			if block.ID == "cost:line:"+lineID {
				pricing = block
			}
		}
	}
	if pricing.ID == "" {
		t.Fatal("missing generated price line")
	}
	raw, err := json.Marshal([]reports.Section{{ID: "fee_return", Title: "Fee return", Blocks: []reports.Block{pricing}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = rawPool(t).Exec(ctx, `UPDATE report_versions SET sections=$3::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, draft.ID, raw); err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: draft.Version, ReportingDate: "2026-10-07", BudgetDisclosed: true, AcceptStale: true, StaleReason: "History fixture"}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	if !issued.Snapshot.BudgetDisclosed {
		t.Fatal("disclosure missing")
	}
	if _, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Issues) != 1 || !view.Issues[0].BudgetDisclosed {
		t.Fatal("history lost disclosure", view.Issues)
	}
	for _, change := range view.Changes {
		if change.TargetID == pricing.ID {
			t.Fatalf("unchanged pricing falsely changed: %+v", change)
		}
	}
	value := cv("125.00")
	value.Version = plan.Items[0].Values["budget"].Version
	if _, err = s.PutCostValue(ctx, orgA, projectA, lineID, userA, "budget", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: value}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true); err != nil {
		t.Fatal(err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, change := range view.Changes {
		if change.TargetID == pricing.ID {
			found = true
			if change.Kind != "changed" || !strings.Contains(change.Before, "100.00") || !strings.Contains(change.After, "125.00") {
				t.Fatal("actual budget change lost", change)
			}
		}
	}
	if !found {
		t.Fatal("actual disclosed budget change was not reported")
	}
}
