package store_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestProposalProjectionRetainsUnchangedRows(t *testing.T) {
	ctx := context.Background()
	s, build, part := workStore(t)
	initial, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	rebuild := func() {
		t.Helper()
		if err := s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
			t.Fatal(err)
		}
	}
	read := func() []store.ProposalView {
		t.Helper()
		v, err := s.ReadProposals(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	pool := rawPool(t)
	identities := func() map[string]string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT 'p/'||key,ctid::text||'/'||xmin::text FROM proposals WHERE org_id=$1 AND project_id=$2
UNION ALL SELECT 't/'||proposal_key||'/'||work_item_id::text,ctid::text||'/'||xmin::text FROM proposal_triggers WHERE org_id=$1 AND project_id=$2`, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]string{}
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				t.Fatal(err)
			}
			out[key] = value
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	rebuild()
	want := read()
	if len(want) < 2 {
		t.Fatal("fixture needs multiple proposals")
	}
	before := identities()
	rebuild()
	if !reflect.DeepEqual(before, identities()) {
		t.Fatal("unchanged rebuild rewrote projection rows")
	}
	if !reflect.DeepEqual(want, read()) {
		t.Fatal("unchanged rebuild changed proposal content")
	}
	var target store.ProposalView
	for _, p := range want {
		for _, id := range p.TriggerWorkItemIDs {
			if id == initial.ID {
				target = p
				break
			}
		}
		if target.Key != "" {
			break
		}
	}
	if target.Key == "" {
		t.Fatal("fixture needs a trigger")
	}
	for _, mode := range []string{"label", "rank", "reason", "unknown_reason", "critical", "missing_trigger"} {
		t.Run(mode, func(t *testing.T) {
			before := identities()
			if mode == "label" {
				_, err = pool.Exec(ctx, `UPDATE proposals SET label='Stale label' WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key)
			} else if mode == "rank" {
				_, err = pool.Exec(ctx, `UPDATE proposals SET rank=rank+500 WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key)
			} else if mode == "reason" {
				_, err = pool.Exec(ctx, `UPDATE proposals SET reason=jsonb_set(reason,'{determinants}','[{"key":"stale","value":"stale","origin":"user"}]'::jsonb),rank=rank+500 WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key)
			} else if mode == "unknown_reason" {
				_, err = pool.Exec(ctx, `UPDATE proposals SET reason=reason||'{"unexpected":"stale"}'::jsonb,rank=rank+500 WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key)
			} else if mode == "critical" {
				_, err = pool.Exec(ctx, `UPDATE proposals SET critical=NOT critical,rank=rank+500 WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key)
			} else {
				_, err = pool.Exec(ctx, `DELETE FROM proposal_triggers WHERE org_id=$1 AND project_id=$2 AND proposal_key=$3 AND work_item_id=$4`, orgA, projectA, target.Key, target.TriggerWorkItemIDs[0])
			}
			if err != nil {
				t.Fatal(err)
			}
			rebuild()
			var unknown bool
			if err := pool.QueryRow(ctx, `SELECT reason ? 'unexpected' FROM proposals WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, target.Key).Scan(&unknown); err != nil || unknown {
				t.Fatalf("stale JSON field survived: %v %v", unknown, err)
			}
			if !reflect.DeepEqual(want, read()) {
				t.Fatal("rebuild did not restore complete projection")
			}
			after := identities()
			for key, value := range before {
				if key == "p/"+target.Key || (mode == "missing_trigger" && strings.HasPrefix(key, "t/"+target.Key+"/")) {
					continue
				}
				if after[key] != value {
					t.Fatalf("unrelated projection row rewritten: %s", key)
				}
			}
		})
	}
	// A later split/replacement can change actual work IDs without changing
	// the semantic fingerprint. Both the reason and relational links must move.
	var replacementID string
	if err := pool.QueryRow(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,user_touched,parent_id)
SELECT org_id,gen_random_uuid(),project_id,site_id,part_id,system_id,action,inclusion,'Child plant',origin,review_status,true,id
FROM work_items WHERE org_id=$1 AND id=$2 RETURNING id::text`, orgA, initial.ID).Scan(&replacementID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE work_items SET is_group=true,version=version+1 WHERE org_id=$1 AND id=$2`, orgA, initial.ID); err != nil {
		t.Fatal(err)
	}
	rebuild()
	var updated store.ProposalView
	for _, p := range read() {
		if p.Key == target.Key {
			updated = p
		}
	}
	if updated.Key == "" || updated.InputsFingerprint != target.InputsFingerprint {
		t.Fatal("replacement changed semantic proposal identity")
	}
	found := false
	for _, id := range updated.TriggerWorkItemIDs {
		if id == initial.ID {
			t.Fatal("retired trigger survived replacement")
		}
		found = found || id == replacementID
	}
	if !found {
		t.Fatal("replacement trigger missing")
	}
	var stale int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM proposal_triggers WHERE org_id=$1 AND project_id=$2 AND work_item_id=$3`, orgA, projectA, initial.ID).Scan(&stale); err != nil {
		t.Fatal(err)
	}
	if stale != 0 {
		t.Fatal("obsolete trigger relations survived")
	}
}
