package store_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// "Engaged by" is retired (combined engagements slice 5, owner decision 3 of
// 7 November 2026): a subconsultant is a group with its lead. No new value is
// written; a value stored earlier stays, keeps its RFP wording, and is cleared
// when the PM combines.
func TestPackageEngagedByRetired(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Structural", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if p.EngagedBy != "" {
		t.Fatalf("a new package is not decided: %q", p.EngagedBy)
	}
	set := func(version int64, v string) (procurement.Package, error) {
		return s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Version: version, EngagedBy: &v})
	}
	for _, v := range []string{procurement.EngagedByLead, procurement.EngagedByClient, "architect"} {
		_, err := set(p.Version, v)
		if !errors.Is(err, store.ErrInvalidPackage) || !strings.Contains(err.Error(), "combine") {
			t.Fatalf("setting %q is refused, pointing to Combine: %v", v, err)
		}
	}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Access", LifecycleStatus: "planned", EngagedBy: procurement.EngagedByLead}); !errors.Is(err, store.ErrInvalidPackage) || !strings.Contains(err.Error(), "combine") {
		t.Fatalf("creating with engaged by is refused: %v", err)
	}
	// Clearing is allowed: it is the retirement's direction.
	p, err = set(p.Version, "")
	if err != nil || p.EngagedBy != "" {
		t.Fatalf("clearing returns to not decided: %+v %v", p, err)
	}
	v := procurement.EngagedByClient
	if _, err := s.PatchPackage(ctx, orgB, projectA, p.ID, userB, procurement.PackagePatch{Version: p.Version, EngagedBy: &v}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org cannot answer: %v", err)
	}
}

func seedEngagedBy(t *testing.T, id string) {
	t.Helper()
	if _, err := rawPool(t).Exec(context.Background(), `UPDATE packages SET engaged_by='lead_consultant' WHERE org_id=$1 AND project_id=$2 AND id=$3`, orgA, projectA, id); err != nil {
		t.Fatal(err)
	}
}

func draftHas(d store.ReportDraft, text string) bool {
	for _, section := range d.Sections {
		for _, b := range section.Blocks {
			if strings.Contains(b.Text, text) {
				return true
			}
		}
	}
	return false
}

func TestStoredEngagedByKeptUntilCombined(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	r, _ := rfpProject(t, s, b, part)
	items, err := s.ReadPackages(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var architect, civil procurement.Package
	for _, p := range items {
		switch p.Title {
		case "Architect":
			architect = p
		case "Civil":
			civil = p
		}
	}
	seedEngagedBy(t, civil.ID)
	// An unrelated edit keeps the old value. The seed bumped no version.
	title := "Civil engineer"
	civil, err = s.PatchPackage(ctx, orgA, projectA, civil.ID, userA, procurement.PackagePatch{Version: civil.Version, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if civil.EngagedBy != procurement.EngagedByLead {
		t.Fatalf("stored value kept by an unrelated edit: %q", civil.EngagedBy)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	if !draftHas(d, "You engage these consultants as your subconsultants") || !strings.Contains(blockTable(d, "rfp:role:team"), "Civil engineer|You, as your subconsultant") {
		t.Fatal("the lead's RFP keeps the stored subconsultant wording")
	}
	// The prompt offers the lead and its subconsultant, ticked.
	o, err := s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Prompts) != 1 || o.Prompts[0].Key != procurement.PromptEngagedBy || !reflect.DeepEqual(o.Prompts[0].PackageIDs, []string{architect.ID, civil.ID}) {
		t.Fatalf("engaged-by prompt: %+v", o.Prompts)
	}
	if _, err := s.CombinePackages(ctx, orgA, projectA, userA, o.Prompts[0].PackageIDs, ""); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadPackages(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range items {
		if p.EngagedBy != "" {
			t.Fatalf("combine clears engaged by on %s: %q", p.Title, p.EngagedBy)
		}
	}
	if o, err := s.ReadPackageProposals(ctx, orgA, projectA); err != nil || len(o.Prompts) != 0 {
		t.Fatalf("no prompt once combined: %+v %v", o.Prompts, err)
	}
}
