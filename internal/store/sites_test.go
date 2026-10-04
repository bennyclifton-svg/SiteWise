package store_test

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"testing"

	"sitewise/internal/store"
)

// Migration 011: every project has its own site, parts belong to the site,
// and the database itself refuses a part on another org's site.
func TestSitesOwnPartsAndRejectCrossTenantRows(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)

	siteA, err := st.ProjectSite(ctx, orgA, projectA)
	if err != nil || siteA.ID == "" || siteA.Label != "Project" || siteA.Version != 1 {
		t.Fatalf("site of a new project: %+v %v", siteA, err)
	}
	if _, err := st.ProjectSite(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org reads the site: %v", err)
	}

	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	roof, err := st.CreatePart(ctx, orgA, projectA, "Roof", "roof", "")
	if err != nil {
		t.Fatalf("roof kind: %v", err)
	}
	if _, err := st.CreatePart(ctx, orgA, projectA, "Plant deck", "plant_area", ""); err != nil {
		t.Fatalf("plant_area kind: %v", err)
	}
	if _, err := st.CreatePart(ctx, orgA, projectA, "Roof", "building", ""); err == nil {
		t.Fatal("duplicate label on one site accepted")
	}
	if _, err := st.CreatePart(ctx, orgB, projectA, "Stolen", "building", ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("part on another org's project: %v", err)
	}
	label := "Roof level"
	if _, err := st.UpdatePart(ctx, orgB, projectB, roof.ID, &label, nil, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org renames the part: %v", err)
	}
	if _, err := st.UpdatePart(ctx, orgA, projectA, roof.ID, &label, nil, nil); err != nil {
		t.Fatal(err)
	}

	var partSite string
	if err := rawPool(t).QueryRow(ctx, `SELECT site_id::text FROM project_parts WHERE org_id = $1::uuid AND id = $2::uuid`,
		orgA, whole.ID).Scan(&partSite); err != nil || partSite != siteA.ID {
		t.Fatalf("whole part site %q, want %q (%v)", partSite, siteA.ID, err)
	}

	// The constraint, not the service, refuses a part whose site is another org's.
	siteB, err := st.ProjectSite(ctx, orgB, projectB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO project_parts (org_id, id, site_id, label, kind) VALUES ($1::uuid, gen_random_uuid(), $2::uuid, 'Forged', 'building')`,
		orgA, siteB.ID); err == nil {
		t.Fatal("database accepted a part on another org's site")
	}
	// One whole part per site.
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO project_parts (org_id, id, site_id, label, kind) VALUES ($1::uuid, gen_random_uuid(), $2::uuid, 'Second whole', 'whole')`,
		orgA, siteA.ID); err == nil {
		t.Fatal("database accepted a second whole part on one site")
	}
	// A part's creating project must be on the part's own site.
	second := "00000000-0000-4000-8000-00000000c011"
	if err := st.CreateProject(ctx, orgA, second, "Second building"); err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO project_parts (org_id, id, site_id, created_by_project_id, label, kind)
VALUES ($1::uuid, gen_random_uuid(), $2::uuid, $3::uuid, 'Crossed', 'building')`,
		orgA, siteA.ID, second); err == nil {
		t.Fatal("database accepted a part whose creating project is on another site")
	}
	// Site ids derive from the project id, so a rerun on a restored database agrees.
	sum := md5.Sum([]byte("site:" + projectA))
	want := fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
	if siteA.ID != want {
		t.Fatalf("site id %s, want %s", siteA.ID, want)
	}
	// Version 1: one project per site.
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO projects (org_id, id, name, site_id) VALUES ($1::uuid, gen_random_uuid(), 'Second', $2::uuid)`,
		orgA, siteA.ID); err == nil {
		t.Fatal("database accepted a second project on one site")
	}
}

func TestUpdateSiteChecksVersionAndOrg(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	site, err := st.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	label, address := "12 Smith St", "12 Smith Street, Newtown NSW"
	updated, err := st.UpdateSite(ctx, orgA, site.ID, site.Version, &label, &address, nil)
	if err != nil || updated.Label != label || updated.Address != address || updated.Version != site.Version+1 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	stale, err := st.UpdateSite(ctx, orgA, site.ID, site.Version, &label, nil, nil)
	if !errors.Is(err, store.ErrVersionConflict) || stale.Version != updated.Version {
		t.Fatalf("stale version: %+v %v", stale, err)
	}
	if _, err := st.UpdateSite(ctx, orgB, site.ID, updated.Version, &label, nil, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org edits the site: %v", err)
	}
}
