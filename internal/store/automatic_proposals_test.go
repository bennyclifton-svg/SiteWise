package store_test

import (
	"context"
	"errors"
	"reflect"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestRebuildAddsMissingWholePartBeforeExistingParts(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	part, err := s.CreatePart(ctx, orgA, projectA, "Building B", "building", "")
	if err != nil {
		t.Fatal(err)
	}
	var first []profile.Part
	compute := func(snap store.ProfileSnapshot) []profile.Row {
		first = append([]profile.Part{}, snap.Parts...)
		return nil
	}
	if err = s.RebuildProfile(ctx, orgA, projectA, "test", compute); err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Kind != "whole" || first[1].ID != part.ID {
		t.Fatalf("whole initialization lost order/part: %+v", first)
	}
	want := append([]profile.Part{}, first...)
	if err = s.RebuildProfile(ctx, orgA, projectA, "test", compute); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatal("rebuild changed existing part identity")
	}
	if err = s.RebuildProfile(ctx, orgB, projectA, "test", compute); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org rebuild: %v", err)
	}
}

func TestOrdinaryWorkEditRefreshesProposals(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	item, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("ordinary work write did not generate proposals")
	}
	action := "retain"
	if _, err = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: item.Version, Action: &action}); err != nil {
		t.Fatal(err)
	}
	after, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) >= len(before) {
		t.Fatalf("changed action retained obsolete proposals: before %d after %d", len(before), len(after))
	}
}

func TestUndoRestoresEvidenceAddressedState(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	proposals, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil || len(proposals) == 0 {
		t.Fatalf("proposals: %v", err)
	}
	p := proposals[0]
	pool := rawPool(t)
	// Exercise the saved-state restoration independently of automatic evaluation,
	// as package/delivery undo paths also return without rebuilding the profile.
	if _, err = pool.Exec(ctx, `UPDATE proposals SET reason=jsonb_set(reason,'{signals}','[{"id":"sig.test","state":"true"}]'::jsonb),state='addressed_by_evidence' WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, p.Key); err != nil {
		t.Fatal(err)
	}
	d, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "")
	if err != nil {
		t.Fatal(err)
	}
	plain := s.WithProfile(store.ProfileBuild{Compute: b.Compute})
	if err = plain.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, d.Version); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = pool.QueryRow(ctx, `SELECT state FROM proposals WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, p.Key).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "addressed_by_evidence" {
		t.Fatalf("undo state %s", state)
	}
}

func TestUnchangedProposalInputsReuseProjection(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	// A statement trigger detects even an upsert that would change no rows.
	// The unchanged-input path must avoid projection writes altogether.
	if _, err := pool.Exec(ctx, `CREATE FUNCTION test_reject_proposal_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'unexpected proposal rewrite'; END $$; CREATE TRIGGER test_reject_proposal_write BEFORE INSERT OR UPDATE OR DELETE ON proposals FOR EACH STATEMENT EXECUTE FUNCTION test_reject_proposal_write()`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DROP TRIGGER IF EXISTS test_reject_proposal_write ON proposals; DROP FUNCTION IF EXISTS test_reject_proposal_write()`)
	})
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal("unchanged rebuild wrote proposals:", err)
	}
	evaluator, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal("unchanged explicit rebuild wrote projection", err)
	}
	if _, err = pool.Exec(ctx, `DROP TRIGGER test_reject_proposal_write ON proposals; DROP FUNCTION test_reject_proposal_write()`); err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	var stamp string
	if err = pool.QueryRow(ctx, `SELECT COALESCE(inputs->>'proposal_fingerprint','') FROM profile_builds WHERE org_id=$1 AND project_id=$2`, orgA, projectA).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	if stamp != "" {
		t.Fatal("explicit rebuild did not invalidate automatic projection stamp")
	}
	if err = s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT COALESCE(inputs->>'proposal_fingerprint','') FROM profile_builds WHERE org_id=$1 AND project_id=$2`, orgA, projectA).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	if stamp == "" {
		t.Fatal("automatic rebuild did not restore stamp")
	}
}

func TestUnchangedProposalRefreshKeepsTriggerMembership(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	before, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil || len(before) == 0 {
		t.Fatalf("proposals: %v", err)
	}
	pool := rawPool(t)
	if _, err = pool.Exec(ctx, `CREATE FUNCTION test_reject_trigger_rewrite() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'unexpected trigger membership write'; END $$; CREATE TRIGGER test_reject_trigger_rewrite BEFORE INSERT OR UPDATE OR DELETE ON proposal_triggers FOR EACH STATEMENT EXECUTE FUNCTION test_reject_trigger_rewrite()`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DROP TRIGGER IF EXISTS test_reject_trigger_rewrite ON proposal_triggers; DROP FUNCTION IF EXISTS test_reject_trigger_rewrite()`)
	})
	evaluator, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	after, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unchanged refresh changed complete projection")
	}
}

func TestLiveProposalRankMatchesGoAndStaysProjectScoped(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pool := rawPool(t)
	site, err := s.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	otherSite, err := s.ProjectSite(ctx, orgB, projectB)
	if err != nil {
		t.Fatal(err)
	}
	want := []works.Proposal{}
	for i, entry := range []struct {
		id, severity string
		specificity  int
	}{{"É", "", 2}, {"Z", "", 2}, {"a", "cost", 2}, {"b", "life-safety", 0}, {"c", "compliance", 1}, {"d", "durability", 3}, {"e", "programme", 1}, {"f", "cost", 3}} {
		record := "cq." + entry.id
		p := works.Proposal{Key: record + "||||0", Severity: entry.severity, Specificity: entry.specificity}
		want = append(want, p)
		if _, err = pool.Exec(ctx, `INSERT INTO proposals(org_id,project_id,site_id,key,record_kind,record_id,proposal_index,kind,label,reason,severity,specificity,draft,unaccepted_triggers,inputs_fingerprint,knowledge_version,state) VALUES($1,$2,$3,$4,'cq',$5,0,'investigation','Rank fixture','{}',$6,$7,true,false,repeat('a',64),'fixture','open')`, orgA, projectA, site.ID, p.Key, record, p.Severity, p.Specificity); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err = pool.Exec(ctx, `INSERT INTO proposals(org_id,project_id,site_id,key,record_kind,record_id,proposal_index,kind,label,reason,severity,specificity,draft,unaccepted_triggers,inputs_fingerprint,knowledge_version,state) VALUES($1,$2,$3,'cq.foreign||||0','cq','cq.foreign',0,'investigation','Other project','{}','life-safety',3,true,false,repeat('b',64),'fixture','open')`, orgB, projectB, otherSite.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	check := func() {
		t.Helper()
		expected := append([]works.Proposal{}, want...)
		works.RankProposals(expected)
		got, err := s.ReadProposals(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(expected) {
			t.Fatalf("rank count %d != %d", len(got), len(expected))
		}
		for i := range got {
			if got[i].Key != expected[i].Key || got[i].Rank != expected[i].Rank {
				t.Fatalf("rank %d got%s/%d want%s/%d", i, got[i].Key, got[i].Rank, expected[i].Key, expected[i].Rank)
			}
		}
	}
	check()
	if _, err = pool.Exec(ctx, `UPDATE proposals SET severity='life-safety',specificity=3,state='accepted' WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, want[0].Key); err != nil {
		t.Fatal(err)
	}
	want[0].Severity = "life-safety"
	want[0].Specificity = 3
	check()
	if _, err = pool.Exec(ctx, `DELETE FROM proposals WHERE org_id=$1 AND project_id=$2 AND key=$3`, orgA, projectA, want[1].Key); err != nil {
		t.Fatal(err)
	}
	want = append(want[:1], want[2:]...)
	check()
}
