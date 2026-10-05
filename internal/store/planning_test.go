package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/store"
)

// WP-13: a planning value is typed, versioned and superseded rather than
// overwritten; a site value is seen by the project; unknown has no value.
func TestPlanningValuesAreTypedVersionedAndKept(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	low, high := 900.0, 1100.0
	gfa := store.PlanningWrite{PartID: whole.ID, Key: "gross_floor_area", Scope: "site", Kind: "number", State: "set",
		Value: strPtr("1000"), RangeLow: &low, RangeHigh: &high, Unit: "m2", Origin: "assumption", Meaning: "stated",
		Rationale: "From the brief's area schedule", Limitations: "Excludes the basement"}
	v1, err := st.SetPlanningValue(ctx, orgA, projectA, userA, gfa)
	if err != nil || v1 != 1 {
		t.Fatalf("first value: %d %v", v1, err)
	}
	if cur, err := st.SetPlanningValue(ctx, orgA, projectA, userA, gfa); !errors.Is(err, store.ErrVersionConflict) || cur != 1 {
		t.Fatalf("stale version: %d %v", cur, err)
	}
	gfa.Version, gfa.Value = 1, strPtr("1050.5")
	if v2, err := st.SetPlanningValue(ctx, orgA, projectA, userA, gfa); err != nil || v2 != 2 {
		t.Fatalf("edit: %d %v", v2, err)
	}
	snap, err := st.ProfileInput(ctx, orgA, projectA)
	if err != nil || len(snap.Planning) != 1 {
		t.Fatalf("live values %+v %v", snap.Planning, err)
	}
	p := snap.Planning[0]
	if *p.Value != "1050.5" || p.Version != 2 || p.Origin != "assumption" || p.ReviewStatus != "accepted_for_planning" ||
		*p.RangeLow != 900 || p.Limitations != "Excludes the basement" {
		t.Fatalf("live value %+v", p)
	}
	all, err := st.PlanningValues(ctx, orgA, projectA, true)
	if err != nil || len(all) != 2 || all[0].ReviewStatus != "superseded" || *all[0].Value != "1000" {
		t.Fatalf("history kept: %+v %v", all, err)
	}

	// Withdrawing supersedes with nothing; history stays.
	if cur, err := st.WithdrawPlanningValue(ctx, orgA, projectA, whole.ID, "gross_floor_area", "site", 1); !errors.Is(err, store.ErrVersionConflict) || cur != 2 {
		t.Fatalf("stale withdraw: %d %v", cur, err)
	}
	if _, err := st.WithdrawPlanningValue(ctx, orgA, projectA, whole.ID, "gross_floor_area", "site", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := st.WithdrawPlanningValue(ctx, orgA, projectA, whole.ID, "gross_floor_area", "site", 2); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("withdraw with no live value: %v", err)
	}
	if all, _ := st.PlanningValues(ctx, orgA, projectA, true); len(all) != 2 {
		t.Fatalf("withdraw deleted history: %+v", all)
	}
	// A new value after a withdrawal continues the version history.
	gfa.Version = 0
	if v3, err := st.SetPlanningValue(ctx, orgA, projectA, userA, gfa); err != nil || v3 != 3 {
		t.Fatalf("after withdraw: %d %v", v3, err)
	}

	// Unknown is stored with no value, never zero or false.
	if _, err := st.SetPlanningValue(ctx, orgA, projectA, userA, store.PlanningWrite{PartID: whole.ID,
		Key: "existing_structure_adequate", Scope: "site", Kind: "boolean", State: "unknown", Origin: "assumption", Meaning: "stated"}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := rawPool(t).QueryRow(ctx, `SELECT num_nonnulls(value_text, value_numeric, value_bool) FROM profile_planning_values
WHERE org_id = $1::uuid AND key = 'existing_structure_adequate'`, orgA).Scan(&n); err != nil || n != 0 {
		t.Fatalf("unknown stored a value: %d %v", n, err)
	}
	// The database refuses two typed values, a set value with none, and a money key.
	for _, sql := range []string{
		`UPDATE profile_planning_values SET value_text = 'x' WHERE org_id = $1::uuid AND key = 'gross_floor_area' AND review_status <> 'superseded'`,
		`UPDATE profile_planning_values SET value_state = 'set' WHERE org_id = $1::uuid AND key = 'existing_structure_adequate'`,
		`UPDATE profile_planning_values SET key = 'cost_total' WHERE org_id = $1::uuid AND key = 'existing_structure_adequate'`,
		`UPDATE profile_planning_values SET review_status = 'verified' WHERE org_id = $1::uuid AND key = 'existing_structure_adequate'`,
	} {
		if _, err := rawPool(t).Exec(ctx, sql, orgA); err == nil {
			t.Fatalf("database accepted: %s", sql)
		}
	}
}

// AT-22: a planning value cannot name a part on another site, or a project
// of another org, whatever the service does.
func TestPlanningOwnershipIsEnforced(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	wholeA, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	other := "00000000-0000-4000-8000-00000000c013"
	if err := st.CreateProject(ctx, orgA, other, "Other building"); err != nil {
		t.Fatal(err)
	}
	w := store.PlanningWrite{PartID: wholeA.ID, Key: "construction_duration", Scope: "project", Kind: "number", State: "set",
		Value: strPtr("40"), Unit: "weeks", Origin: "assumption", Meaning: "forecast"}
	if _, err := st.SetPlanningValue(ctx, orgA, other, userA, w); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("part of another project's site: %v", err)
	}
	if _, err := st.SetPlanningValue(ctx, orgB, projectA, userA, w); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org: %v", err)
	}
	site, err := st.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `
INSERT INTO profile_planning_values (org_id, site_id, project_id, scope, part_id, key, value_state, value_numeric,
  origin, review_status, meaning, version)
VALUES ($1::uuid, $2::uuid, $3::uuid, 'project', $4::uuid, 'construction_duration', 'set', 40, 'assumption',
  'accepted_for_planning', 'forecast', 1)`, orgA, site.ID, other, wholeA.ID); err == nil {
		t.Fatal("database accepted a planning value on another project's site")
	}
}
