package store_test

import (
	"context"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestProcurementCreateSuggestedRowBumpsPackagesAndReports(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "residential", "hdr.work_type": "new"} {
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: map[string]string{"hdr.building_class": "site", "hdr.work_type": "project"}[key], Value: &value}); err != nil {
			t.Fatal(err)
		}
	}
	_, suggestions, err := s.ReadPackageOverview(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	key := ""
	for _, sug := range suggestions {
		if sug.Title == "Architect" {
			key = sug.Key
		}
	}
	if key == "" {
		t.Fatalf("no Architect suggestion: %+v", suggestions)
	}
	pool := rawPool(t)
	counts := func() [4]int64 {
		var c [4]int64
		if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT packages FROM project_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid),0),
 COALESCE((SELECT reports FROM project_revisions WHERE org_id=$1::uuid AND project_id=$2::uuid),0),
 (SELECT count(*) FROM events WHERE org_id=$1::uuid AND kind='packages'),
 (SELECT count(*) FROM events WHERE org_id=$1::uuid AND kind='report')`, orgA, projectA).Scan(&c[0], &c[1], &c[2], &c[3]); err != nil {
			t.Fatal(err)
		}
		return c
	}
	before := counts()
	p, r, created, err := s.ProcurementCreate(ctx, orgA, projectA, userA, "", key, "consultant.architect")
	if err != nil || !created || p.DisciplineID != "consultant.architect" || r.Kind != "rfp" || r.PackageID != p.ID {
		t.Fatalf("%+v %+v %v %v", p, r, created, err)
	}
	after := counts()
	for i, name := range []string{"packages revision", "reports revision", "packages event", "report event"} {
		if after[i] != before[i]+1 {
			t.Fatalf("%s: %d -> %d", name, before[i], after[i])
		}
	}
}

func TestProcurementCreateHeadContractReacceptKeepsReportAndAddsLines(t *testing.T) {
	ctx := context.Background()
	s, _, whole := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: whole, SystemID: "mechanical.air-conditioning", Action: "replace", Title: "Chiller"}); err != nil {
		t.Fatal(err)
	}
	p, first, created, err := s.ProcurementCreate(ctx, orgA, projectA, userA, "", procurement.WorksProposalKey, "trade.main_works")
	if err != nil || !created || first.Kind != "rft" || p.DisciplineID != "trade.main_works" {
		t.Fatalf("%+v %+v %v %v", p, first, created, err)
	}
	scope, err := s.ReadPackageScope(ctx, orgA, projectA, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	lines := len(scope)
	// A new work item makes the head contract proposal reappear as an addition.
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: whole, SystemID: "fire-active", Action: "retain", Title: "Hydrants"}); err != nil {
		t.Fatal(err)
	}
	again, second, created, err := s.ProcurementCreate(ctx, orgA, projectA, userA, "", procurement.WorksProposalKey, "trade.main_works")
	if err != nil || created || second.ID != first.ID || again.ID != p.ID {
		t.Fatalf("re-accept: %+v %+v %v %v", again, second, created, err)
	}
	scope, err = s.ReadPackageScope(ctx, orgA, projectA, p.ID)
	if err != nil || len(scope) != lines+1 {
		t.Fatalf("added line not kept: %d -> %d %v", lines, len(scope), err)
	}
}
