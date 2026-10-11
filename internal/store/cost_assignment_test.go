package store_test

import (
	"context"
	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestCostAutomaticPackageSnapshotAndReassignment(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	head, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Title: "Head contractor", Kind: "works", WorksScope: "head_contract", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "electrical", Action: "new", Title: "Electrical works"}); e != nil {
		t.Fatal(e)
	}
	ready, e := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if e != nil {
		t.Fatal(e)
	}
	p := ready.Plan
	var line costs.Item
	for _, i := range p.Items {
		if i.SystemID == "electrical" {
			line = i
		}
	}
	if line.ResolvedPackageID != head.ID || line.PackageID != "" {
		t.Fatal("automatic assignment missing or made explicit", line)
	}
	p, e = s.PutCostValue(ctx, orgA, projectA, line.ID, userA, "commitment", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: cv("900.01")})
	if e != nil {
		t.Fatal(e)
	}
	totals, e := s.CostTotals(ctx, orgA, projectA, "", "package")
	if e != nil || *totals.Groups[head.ID]["commitment"].Amount != "900.01" {
		t.Fatal(totals, e)
	}
	old := p.ID
	p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Title: "Second contractor", Kind: "works", WorksScope: "head_contract", LifecycleStatus: "planned"}); e != nil {
		t.Fatal(e)
	}
	current, e := s.ReadCostPlan(ctx, orgA, projectA, "")
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range current.Items {
		if i.ID == line.ID {
			if i.ResolvedPackageID != "" {
				t.Fatal("ambiguous automatic package", i)
			}
			line = i
		}
	}
	frozen, e := s.CostTotals(ctx, orgA, projectA, old, "package")
	if e != nil || *frozen.Groups[head.ID]["commitment"].Amount != "900.01" {
		t.Fatal("baseline remapped", frozen, e)
	}
	content := line.Content
	content.PackageID = head.ID
	p, e = s.PatchCostItem(ctx, orgA, projectA, line.ID, userA, store.CostItemInput{CostWrite: cw(current), Content: content})
	if e != nil {
		t.Fatal(e)
	}
	for _, i := range p.Items {
		if i.ID == line.ID && (i.ResolvedPackageID != head.ID || i.SystemID != "electrical") {
			t.Fatal("assignment changed physical scope", i)
		}
	}
	child := content
	child.Label = "Switchboard"
	child.PackageID = ""
	p, e = s.SubdivideCostItem(ctx, orgA, projectA, line.ID, userA, store.CostSubdivision{CostWrite: cw(p), Residual: "explicit", Children: []store.CostItemInput{{Content: child, Values: map[string]costs.Value{"commitment": cv("400.00")}}}})
	if e != nil {
		t.Fatal(e)
	}
	children := 0
	for _, i := range p.Items {
		if i.ParentItemID == line.ID {
			children++
			if i.PackageID != "" || i.ResolvedPackageID != head.ID {
				t.Fatal("child failed to inherit package", i)
			}
		}
	}
	if children != 2 {
		t.Fatal("missing child or residual")
	}
	totals, e = s.CostTotals(ctx, orgA, projectA, "", "package")
	if e != nil || *totals.Groups[head.ID]["commitment"].Amount != "900.01" {
		t.Fatal("subdivision changed package total", totals, e)
	}
}
