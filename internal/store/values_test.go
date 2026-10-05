package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/store"
)

// WP-12: a site value belongs to the site (no project), a project value to
// the project; a stale version is refused; reset and unknown behave as the
// plan says (§4.3).
func TestUserValuesHaveScopeVersionAndState(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	v1, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "det.ncc_class", store.UserWrite{Value: strPtr("5"), Scope: "site"})
	if err != nil || v1 != 1 {
		t.Fatalf("site value: %d %v", v1, err)
	}
	var project *string
	if err := rawPool(t).QueryRow(ctx, `SELECT project_id::text FROM profile_user_values WHERE org_id = $1::uuid AND key = 'det.ncc_class'`,
		orgA).Scan(&project); err != nil || project != nil {
		t.Fatalf("site value stored with project %v (%v)", project, err)
	}
	snap, err := st.ProfileInput(ctx, orgA, projectA)
	if err != nil || len(snap.User) != 1 || snap.User[0].Version != 1 {
		t.Fatalf("project sees its site's value: %+v %v", snap.User, err)
	}

	stale := int64(0)
	if cur, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "det.ncc_class",
		store.UserWrite{Value: strPtr("6"), Scope: "site", Version: &stale}); !errors.Is(err, store.ErrVersionConflict) || cur != 1 {
		t.Fatalf("stale version: %d %v", cur, err)
	}
	if v2, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "det.ncc_class",
		store.UserWrite{Value: strPtr("6"), Scope: "site", Version: &v1}); err != nil || v2 != 2 {
		t.Fatalf("current version: %d %v", v2, err)
	}

	if _, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "hdr.work_type",
		store.UserWrite{State: "unknown", Origin: "assumption", Scope: "project"}); err != nil {
		t.Fatal(err)
	}
	snap, _ = st.ProfileInput(ctx, orgA, projectA)
	found := false
	for _, u := range snap.User {
		if u.Key == "hdr.work_type" {
			found = u.State == "unknown" && u.Value == nil && u.Origin == "assumption"
		}
	}
	if !found {
		t.Fatalf("unknown assumption not stored: %+v", snap.User)
	}
	// The database refuses a value whose state and value disagree.
	if _, err := rawPool(t).Exec(ctx, `UPDATE profile_user_values SET value = NULL, value_state = 'set'
WHERE org_id = $1::uuid AND key = 'det.ncc_class'`, orgA); err == nil {
		t.Fatal("database accepted a set value with no value")
	}

	if err := st.DeleteUserValue(ctx, orgA, projectA, whole.ID, "det.ncc_class", "site", nil); err != nil {
		t.Fatal(err)
	}
	snap, _ = st.ProfileInput(ctx, orgA, projectA)
	for _, u := range snap.User {
		if u.Key == "det.ncc_class" {
			t.Fatal("reset left the site value")
		}
	}
}

// A user value cannot point at a part on another site, or at another org's
// project, whatever the service does.
func TestUserValueOwnershipIsEnforcedByTheDatabase(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	wholeA, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	other := "00000000-0000-4000-8000-00000000c012"
	if err := st.CreateProject(ctx, orgA, other, "Other building"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetUserValue(ctx, orgA, other, wholeA.ID, userA, "hdr.work_type",
		store.UserWrite{Value: strPtr("new"), Scope: "project"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("part of another project's site: %v", err)
	}
	site, err := st.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO profile_user_values (org_id, id, project_id, site_id, part_id, scope, key, value, user_id)
VALUES ($1::uuid, gen_random_uuid(), $2::uuid, $3::uuid, $4::uuid, 'project', 'hdr.work_type', 'new', $5::uuid)`,
		orgA, other, site.ID, wholeA.ID, userA); err == nil {
		t.Fatal("database accepted a project value on another project's site")
	}
}
