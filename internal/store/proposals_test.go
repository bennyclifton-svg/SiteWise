package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"sitewise/internal/delivery"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestProposalUndoRefusesEditedOrReferencedWork(t *testing.T) {
	for _, mode := range []string{"edited", "referenced", "historical_reference", "package_scope", "child", "delivery", "report"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			s, b, part := workStore(t)
			if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
				t.Fatal(err)
			}
			e, err := works.NewEvaluator(b.Catalog)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
				t.Fatal(err)
			}
			ps, err := s.ReadProposals(ctx, orgA, projectA)
			if err != nil {
				t.Fatal(err)
			}
			var p store.ProposalView
			for _, candidate := range ps {
				if candidate.RecordID == "ic.loads-investigate-supported" && candidate.InterfaceID == "if.plant-loads-structure" {
					p = candidate
					break
				}
			}
			if p.Key == "" {
				t.Fatal("missing proposal")
			}
			item, err := s.AcceptProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint)
			if err != nil {
				t.Fatal(err)
			}
			pool := rawPool(t)
			if mode == "edited" {
				title := "Edited by user"
				_, err = s.PatchWorkItem(ctx, orgA, projectA, item.ID, userA, works.Patch{Version: item.Version, Title: &title})
			} else if mode == "child" {
				// Splitting is a later API, but a persisted child already counts as use.
				_, err = pool.Exec(ctx, `INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,parent_id,origin,review_status) SELECT org_id,gen_random_uuid(),project_id,site_id,part_id,system_id,action,inclusion,'Child investigation',id,'user','accepted_for_planning' FROM work_items WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, item.ID)
			} else if mode == "delivery" {
				_, err = s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "milestone", Title: "Investigation complete", WorkItemID: item.ID}})
			} else if mode == "report" {
				pkg, createErr := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Investigation", LifecycleStatus: "planned"})
				if createErr != nil {
					t.Fatal(createErr)
				}
				report, createErr := s.CreateReport(ctx, orgA, projectA, userA, "rfp", pkg.ID)
				if createErr != nil {
					t.Fatal(createErr)
				}
				_, err = s.RefreshReport(ctx, orgA, report.ID, userA, "test-build", true)
			} else if mode == "package_scope" {
				pkg, createErr := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Investigation", LifecycleStatus: "planned"})
				if createErr != nil {
					t.Fatal(createErr)
				}
				_, err = pool.Exec(ctx, `INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,user_text,inclusion,origin,review_status) VALUES($1::uuid,gen_random_uuid(),$2::uuid,$3::uuid,'responsibility',$4::uuid,'inspect','Inspect the structure','included','user','accepted_for_planning')`, orgA, projectA, pkg.ID, item.ID)
			} else if mode == "historical_reference" {
				// A later decision can replace its current snapshot; prior use survives
				// in the audit history and must still prevent retirement.
				_, err = pool.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,decision,inputs_fingerprint,actor,undo_history) VALUES($1::uuid,gen_random_uuid(),$2::uuid,'ic.reference||||0','ic.reference','dismissed',$4,$5::uuid,jsonb_build_array(jsonb_build_object('inputs_snapshot',jsonb_build_object('reason',jsonb_build_object('triggers',jsonb_build_array(jsonb_build_object('work_item_id',$3::text)))))))`, orgA, projectA, item.ID, p.InputsFingerprint, userA)
			} else {
				_, err = pool.Exec(ctx, `INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor) VALUES($1::uuid,gen_random_uuid(),$2::uuid,'ic.reference||||0','ic.reference',$3::uuid,'dismissed',$4,$5::uuid)`, orgA, projectA, item.ID, p.InputsFingerprint, userA)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 1); !errors.Is(err, store.ErrProposalUndoBlocked) {
				t.Fatalf("undo %s: %v", mode, err)
			}
			var live bool
			if err := pool.QueryRow(ctx, `SELECT retired_at IS NULL FROM work_items WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, item.ID).Scan(&live); err != nil || !live {
				t.Fatalf("blocked undo changed item: %v %v", live, err)
			}
			var version int64
			var history int
			if err := pool.QueryRow(ctx, `SELECT version,jsonb_array_length(undo_history) FROM proposal_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key=$3`, orgA, projectA, p.Key).Scan(&version, &history); err != nil || version != 1 || history != 0 {
				t.Fatalf("blocked undo changed decision: %d %d %v", version, history, err)
			}
		})
	}
}

func TestProposalAcceptanceConcurrentRetries(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Rooftop plant"}); err != nil {
		t.Fatal(err)
	}
	e, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
		t.Fatal(err)
	}
	ps, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var p store.ProposalView
	for _, candidate := range ps {
		if candidate.RecordID == "ic.loads-investigate-supported" && candidate.InterfaceID == "if.plant-loads-structure" {
			p = candidate
			break
		}
	}
	if p.Key == "" {
		t.Fatal("missing rooftop proposal")
	}
	if _, err := s.AcceptProposal(ctx, orgB, projectA, p.Key, userB, p.InputsFingerprint); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project: %v", err)
	}
	if _, err := s.AcceptProposal(ctx, orgA, projectA, p.Key, userB, p.InputsFingerprint); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor: %v", err)
	}
	if _, err := s.AcceptProposal(ctx, orgA, projectA, p.Key, userA, "stale"); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale accept: %v", err)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := s.AcceptProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint)
			if err != nil {
				t.Error(err)
				return
			}
			if item.Action != "investigate" || item.ReviewStatus != "accepted_for_planning" || item.SourceProposalKey != p.Key || item.CoarseKey != "" {
				t.Errorf("invalid accepted item: %+v", item)
			}
			ids <- item.ID
		}()
	}
	wg.Wait()
	close(ids)
	id, count := "", 0
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate accepted work")
		}
		id = got
		count++
	}
	if count != 4 {
		t.Fatalf("only %d successful accepts", count)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
		t.Fatal(err)
	}
	ps, err = s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range ps {
		if got.Key == p.Key && (got.State != "accepted" || got.Decision == nil || got.Decision.CreatedRecordID != id || got.Decision.Version != 1) {
			t.Fatalf("acceptance lost: %+v", got)
		}
	}
	if _, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, ""); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("dismiss accepted proposal: %v", err)
	}
	// Accepted records outlive removed knowledge, and replaying the original
	// request must still find the durable result rather than create a duplicate.
	b.Catalog.Interfaces = nil
	e, err = works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
		t.Fatal(err)
	}
	got, err := s.AcceptProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint)
	if err != nil || got.ID != id {
		t.Fatalf("retry after removal: %+v %v", got, err)
	}
}

func TestProposalPersistenceDismissalAndReopen(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Inclusion: "included", Title: "Rooftop plant"}); err != nil {
		t.Fatal(err)
	}
	e, err := works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	rebuild := func() {
		t.Helper()
		if err := s.RebuildProposals(ctx, orgA, projectA, e); err != nil {
			t.Fatal(err)
		}
	}
	read := func() []store.ProposalView {
		t.Helper()
		ps, err := s.ReadProposals(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		return ps
	}
	find := func(ps []store.ProposalView) store.ProposalView {
		t.Helper()
		for _, p := range ps {
			if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" {
				return p
			}
		}
		t.Fatal("rooftop proposal missing")
		return store.ProposalView{}
	}
	rebuild()
	p := find(read())
	if len(p.TriggerWorkItemIDs) != 1 || p.State != "open" {
		t.Fatal(p)
	}
	if _, err := s.ReadProposals(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read: %v", err)
	}
	if _, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userB, p.InputsFingerprint, ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor: %v", err)
	}
	if _, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, "stale", ""); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale dismissal: %v", err)
	}
	d, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "Review later")
	if err != nil {
		t.Fatal(err)
	}
	if d.Version != 1 || len(d.InputsSnapshot) < 3 {
		t.Fatal(d)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			same, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, "Review later")
			if err != nil || same.ID != d.ID || same.Version != 1 {
				t.Errorf("non-idempotent retry: %+v %v", same, err)
			}
		}()
	}
	wg.Wait()
	rebuild()
	if got := find(read()); got.State != "dismissed" || got.Decision == nil || got.Decision.Version != 1 {
		t.Fatal(got)
	}
	// A removed interface removes its projection, never the durable decision.
	interfaces := b.Catalog.Interfaces
	b.Catalog.Interfaces = nil
	e, err = works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	rebuild()
	for _, got := range read() {
		if got.Key == p.Key {
			t.Fatal("removed interface still projected")
		}
	}
	b.Catalog.Interfaces = interfaces
	e, err = works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	rebuild()
	if got := find(read()); got.State != "dismissed" || got.Decision == nil || got.Decision.ID != d.ID {
		t.Fatal("decision lost", got)
	}
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "sys.structure.existing", store.UserWrite{Scope: "site", Value: strPtr("present")}); err != nil {
		t.Fatal(err)
	}
	rebuild()
	changed := find(read())
	if changed.State != "reopened" || !changed.InputsChanged || changed.InputsFingerprint == p.InputsFingerprint || changed.Decision.InputsFingerprint != p.InputsFingerprint {
		t.Fatal(changed)
	}
	if _, err := s.DismissProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, ""); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("old fingerprint accepted: %v", err)
	}
	// Force a projection constraint failure after DELETE; the old projection
	// and decision must remain intact because replacement is one transaction.
	before := read()
	records := b.Catalog.InterfaceConsequences()
	for i := range records {
		if records[i].ID == p.RecordID {
			records[i].Propose.Label = ""
		}
	}
	e, err = works.NewEvaluator(b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, e); err == nil {
		t.Fatal("invalid projection accepted")
	}
	after := read()
	if len(after) != len(before) || find(after).InputsFingerprint != changed.InputsFingerprint || find(after).State != "reopened" {
		t.Fatal("failed replacement changed old projection")
	}
}
