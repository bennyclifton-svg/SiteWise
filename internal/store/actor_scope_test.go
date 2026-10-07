package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestPlanningAndWorkWritesRejectForeignActor(t *testing.T) {
	for _, operation := range []string{"user_site", "user_project", "planning_site", "planning_project", "work", "scope"} {
		for _, actor := range []string{userB, "99999999-9999-4999-8999-999999999999", ""} {
			t.Run(operation+"/"+actor, func(t *testing.T) {
				ctx := context.Background()
				s, _, part := workStore(t)
				pool := rawPool(t)
				snapshot := func() string {
					t.Helper()
					var raw string
					if err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
 'user',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM profile_user_values v WHERE org_id=$1::uuid),
 'planning',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM profile_planning_values v WHERE org_id=$1::uuid),
 'works',(SELECT jsonb_agg(to_jsonb(w) ORDER BY id) FROM work_items w WHERE org_id=$1::uuid),
 'rows',(SELECT jsonb_agg(to_jsonb(r) ORDER BY part_id,key,scope) FROM profile_rows r WHERE org_id=$1::uuid),
 'build',(SELECT jsonb_agg(to_jsonb(b) ORDER BY project_id) FROM profile_builds b WHERE org_id=$1::uuid),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_revisions r WHERE org_id=$1::uuid),
 'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM events e WHERE org_id=$1::uuid))::text`, orgA).Scan(&raw); err != nil {
						t.Fatal(err)
					}
					return raw
				}
				write := func(actor string) error {
					switch operation {
					case "user_site", "user_project":
						scope, key, value := "site", "det.ncc_class", "5"
						if operation == "user_project" {
							scope, key, value = "project", "hdr.work_type", "refurb"
						}
						_, err := s.SetUserValue(ctx, orgA, projectA, part, actor, key, store.UserWrite{Scope: scope, Value: &value})
						return err
					case "planning_site", "planning_project":
						scope := "site"
						if operation == "planning_project" {
							scope = "project"
						}
						_, err := s.SetPlanningValue(ctx, orgA, projectA, actor, store.PlanningWrite{PartID: part, Scope: scope, Key: "gross_floor_area", Kind: "number", State: "set", Value: strPtr("1000"), Origin: "assumption", Meaning: "stated"})
						return err
					case "work":
						_, err := s.CreateWorkItem(ctx, orgA, projectA, actor, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Repair gas"})
						return err
					default:
						return s.SetScope(ctx, orgA, projectA, part, actor, map[string]*string{"scope.hydraulic.gas": strPtr("in")})
					}
				}
				before := snapshot()
				if err := write(actor); !errors.Is(err, store.ErrNotFound) {
					t.Fatalf("foreign/missing actor accepted: %v", err)
				}
				if snapshot() != before {
					t.Fatal("rejected actor changed saved state")
				}
				if err := write(userA); err != nil {
					t.Fatalf("same-organisation actor rejected: %v", err)
				}
			})
		}
	}
}

func TestCalculatedPlanningValueAllowsNoHumanAuthor(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	w := store.PlanningWrite{PartID: part, Scope: "site", Key: "gross_floor_area", Kind: "number", State: "set", Value: strPtr("1000"), Origin: "calculation", Meaning: "stated"}
	if _, err := s.SetPlanningValue(ctx, orgA, projectA, "", w); err != nil {
		t.Fatal(err)
	}
	var anonymous bool
	if err := rawPool(t).QueryRow(ctx, `SELECT user_id IS NULL AND review_status='proposed' FROM profile_planning_values WHERE org_id=$1::uuid AND key='gross_floor_area'`, orgA).Scan(&anonymous); err != nil || !anonymous {
		t.Fatalf("calculation acquired human attribution: %v %v", anonymous, err)
	}
}

func TestActorReferencesPreserveAuditAndAllowOrgDeletion(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "det.ncc_class", store.UserWrite{Scope: "site", Value: strPtr("5")}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetPlanningValue(ctx, orgA, projectA, userA, store.PlanningWrite{PartID: part, Scope: "site", Key: "gross_floor_area", Kind: "number", State: "set", Value: strPtr("1000"), Origin: "assumption", Meaning: "stated"}); err != nil {
		t.Fatal(err)
	}
	item, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Repair gas"})
	if err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE work_items SET verified_by=$3::uuid,verified_at=now(),verification_basis='Test verification',review_status='verified',retired_by=$3::uuid,retired_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, item.ID, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE profile_user_values SET verified_by=$2::uuid,verified_at=now(),verification_basis='Test verification',review_status='verified' WHERE org_id=$1::uuid`, orgA, userA); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, userA)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("referenced actor was removed: %v", err)
	}
	if err := s.DeleteOrg(ctx, orgA); err != nil {
		t.Fatalf("actor constraints prevent whole-organisation deletion: %v", err)
	}
	if _, err := s.GetProject(ctx, orgB, projectB); err != nil {
		t.Fatalf("deletion affected another organisation: %v", err)
	}
}
