package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/store"
)

func TestPlanningSuccessorKeepsValueOwnership(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	w := store.PlanningWrite{PartID: part, Key: "gross_floor_area", Scope: "project", Kind: "number", State: "set", Value: strPtr("1000"), Origin: "assumption", Meaning: "stated"}
	if _, err := s.SetPlanningValue(ctx, orgA, projectA, userA, w); err != nil {
		t.Fatal(err)
	}
	w.Version, w.Value = 1, strPtr("1050")
	if _, err := s.SetPlanningValue(ctx, orgA, projectA, userA, w); err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	var old, successor string
	if err := pool.QueryRow(ctx, `SELECT id::text,superseded_by::text FROM profile_planning_values WHERE org_id=$1::uuid AND key=$2 AND version=1`, orgA, w.Key).Scan(&old, &successor); err != nil {
		t.Fatal(err)
	}
	otherPart, err := s.CreatePart(ctx, orgA, projectA, "Other building", "building", "")
	if err != nil {
		t.Fatal(err)
	}
	const otherProject = "33333333-3333-4333-8333-111111111111"
	if err := s.CreateProject(ctx, orgA, otherProject, "Other project"); err != nil {
		t.Fatal(err)
	}
	otherSitePart, err := s.EnsureWholePart(ctx, orgA, otherProject)
	if err != nil {
		t.Fatal(err)
	}
	foreignPart, err := s.EnsureWholePart(ctx, orgB, projectB)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, org, project, part, key, scope string }{
		{"key", orgA, projectA, part, "building_age", "project"},
		{"part", orgA, projectA, otherPart.ID, w.Key, "project"},
		{"site_project", orgA, otherProject, otherSitePart.ID, w.Key, "project"},
		{"scope", orgA, "", part, w.Key, "site"},
		{"organisation", orgB, projectB, foreignPart.ID, w.Key, "project"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, `UPDATE profile_planning_values SET superseded_by=$3::uuid WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, old, successor); err != nil {
				t.Fatalf("valid successor rejected: %v", err)
			}
			var target string
			// Historical rows avoid altering the live value. The target itself is
			// valid, so only the proposed relationship should be rejected.
			if err := pool.QueryRow(ctx, `INSERT INTO profile_planning_values(org_id,site_id,project_id,part_id,key,scope,value_state,origin,review_status,meaning,version,superseded_at)
 SELECT org_id,site_id,NULLIF($2,'')::uuid,id,$4,$5,'unknown','calculation','superseded','stated',7,now()
 FROM project_parts WHERE org_id=$1::uuid AND id=$3::uuid RETURNING id::text`, tc.org, tc.project, tc.part, tc.key, tc.scope).Scan(&target); err != nil {
				t.Fatal(err)
			}
			_, err := pool.Exec(ctx, `UPDATE profile_planning_values SET superseded_by=$3::uuid WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, old, target)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
				t.Fatalf("unrelated %s successor accepted: %v", tc.name, err)
			}
			var retained string
			if err := pool.QueryRow(ctx, `SELECT superseded_by::text FROM profile_planning_values WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, old).Scan(&retained); err != nil || retained != successor {
				t.Fatalf("rejected link changed history: %s %v", retained, err)
			}
		})
	}
	// Withdrawal remains a legitimate end of history with no successor.
	if _, err := s.WithdrawPlanningValue(ctx, orgA, projectA, part, w.Key, w.Scope, 2); err != nil {
		t.Fatal(err)
	}
}
