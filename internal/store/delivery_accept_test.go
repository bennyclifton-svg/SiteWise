package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"sitewise/internal/delivery"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func deliveryProposal(t *testing.T, kind string) (*store.Store, store.ProposalView) {
	t.Helper()
	ctx := context.Background()
	s, build, part := workStore(t)
	for i := range build.Catalog.InterfaceConsequences() {
		r := &build.Catalog.InterfaceConsequences()[i]
		if r.ID == "ic.loads-investigate-supported" {
			r.Propose.Kind = kind
			r.Propose.Label = "Review before installation"
		}
	}
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	e, err := works.NewEvaluator(build.Catalog)
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
	for _, p := range ps {
		if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" && p.Kind == kind {
			return s, p
		}
	}
	t.Fatal("missing synthetic delivery proposal")
	return nil, store.ProposalView{}
}

func TestDraftFireReviewDoesNotGrantApprovalOrInferAssignments(t *testing.T) {
	ctx := context.Background()
	s, build, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "fire-active.hydrants", Action: "upgrade", Title: "Hydrant upgrade"}); err != nil {
		t.Fatal(err)
	}
	evaluator, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	proposals, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range proposals {
		if p.RecordID != "cq.existing-fire-measures-approval-basis-review" {
			continue
		}
		if !p.Draft {
			t.Fatal("seed draft became reviewed")
		}
		item, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, store.DeliveryAcceptance{})
		if err != nil {
			t.Fatal(err)
		}
		if item.Status != "not_submitted" || item.ReviewStatus != "accepted_for_planning" || item.PackageID != "" || item.WorkItemID != "" || item.StageID != "" {
			t.Fatalf("acceptance granted approval or guessed assignments: %+v", item)
		}
		var provenance struct {
			Proposal works.Proposal `json:"proposal"`
		}
		if err := json.Unmarshal(item.Provenance, &provenance); err != nil {
			t.Fatal(err)
		}
		if !provenance.Proposal.Draft || provenance.Proposal.RecordID != p.RecordID || len(provenance.Proposal.Reason.Triggers) != 1 {
			t.Fatal("acceptance lost draft status or triggering work")
		}
		return
	}
	t.Fatal("missing fire review proposal")
}

func TestDeliveryProposalAcceptanceAndUndo(t *testing.T) {
	for _, kind := range []string{"approval", "hold_point"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s, p := deliveryProposal(t, kind)
			choice := store.DeliveryAcceptance{}
			if _, err := s.AcceptDeliveryProposal(ctx, orgB, projectA, p.Key, userB, p.InputsFingerprint, choice); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("foreign org %v", err)
			}
			if _, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userB, p.InputsFingerprint, choice); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("foreign actor %v", err)
			}
			if _, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, "stale", choice); !errors.Is(err, store.ErrVersionConflict) {
				t.Fatalf("stale %v", err)
			}
			var wg sync.WaitGroup
			items := make(chan delivery.Item, 4)
			for range 4 {
				wg.Go(func() {
					item, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, choice)
					if err != nil {
						t.Error(err)
						return
					}
					items <- item
				})
			}
			wg.Wait()
			close(items)
			id, count := "", 0
			for item := range items {
				if id != "" && id != item.ID {
					t.Fatal("duplicate acceptance")
				}
				id = item.ID
				count++
				wantKind, wantStatus := "approval", "not_submitted"
				if kind == "hold_point" {
					wantKind, wantStatus = "milestone", "planned"
				}
				if item.Kind != wantKind || item.Status != wantStatus || item.ReviewStatus != "accepted_for_planning" || item.Origin != "calculation" || item.Meaning != "requirement" || item.Version != 1 || item.TargetDate != nil || item.ActualDate != nil || item.OwnerUserID != "" {
					t.Fatalf("invented or missing state %+v", item)
				}
				var provenance struct {
					Proposal works.Proposal `json:"proposal"`
				}
				if err := json.Unmarshal(item.Provenance, &provenance); err != nil || provenance.Proposal.Kind != kind || !provenance.Proposal.Draft {
					t.Fatalf("lost draft proposal provenance %s", item.Provenance)
				}
			}
			if count != 4 {
				t.Fatal("missing results", count)
			}
			foreign, err := s.CreateDelivery(ctx, orgB, projectB, userB, store.DeliveryInput{Content: delivery.Content{Kind: "approval", Title: "Other organisation"}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := rawPool(t).Exec(ctx, `UPDATE proposal_decisions SET created_record_id=$4::uuid WHERE org_id=$1::uuid AND project_id=$2::uuid AND proposal_key=$3`, orgA, projectA, p.Key, foreign.ID); err == nil {
				t.Fatal("delivery decision FK allowed a foreign target")
			}
			if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 2); !errors.Is(err, store.ErrVersionConflict) {
				t.Fatalf("stale undo %v", err)
			}
			if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 1); err != nil {
				t.Fatal(err)
			}
			live, err := s.ReadDelivery(ctx, orgA, projectA)
			if err != nil || len(live) != 0 {
				t.Fatalf("undo left active records %+v %v", live, err)
			}
			restored, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, choice)
			if err != nil || restored.ID != id || restored.Version != 3 {
				t.Fatalf("revive %+v %v", restored, err)
			}
			// A retry still returns its accepted record after the projection disappears.
			if _, err := rawPool(t).Exec(ctx, `DELETE FROM proposals WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=$3`, orgA, projectA, p.Key); err != nil {
				t.Fatal(err)
			}
			retry, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, choice)
			if err != nil || retry.ID != id {
				t.Fatalf("retry %+v %v", retry, err)
			}
		})
	}
}

func TestMechanicalCommissioningProposalCreatesUndatedReview(t *testing.T) {
	ctx := context.Background()
	s, build, part := workStore(t)
	work, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "alter", Title: "Alter cooling distribution"})
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	proposals, err := s.ReadProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range proposals {
		if p.RecordID != "cq.mechanical-commissioning-plan-review" {
			continue
		}
		item, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, store.DeliveryAcceptance{})
		if err != nil {
			t.Fatal(err)
		}
		if item.Kind != "milestone" || item.Status != "planned" || item.ReviewStatus != "accepted_for_planning" || item.TargetDate != nil || item.ActualDate != nil || item.BaselineDate != nil || item.ForecastDate != nil || item.PackageID != "" || item.StageID != "" || item.WorkItemID != "" || item.OwnerUserID != "" {
			t.Fatalf("review inferred completion, dates or assignments: %+v", item)
		}
		var provenance struct {
			Proposal works.Proposal `json:"proposal"`
		}
		if err := json.Unmarshal(item.Provenance, &provenance); err != nil {
			t.Fatal(err)
		}
		if !provenance.Proposal.Draft || provenance.Proposal.TargetPartID != part || len(provenance.Proposal.Reason.Triggers) != 1 || provenance.Proposal.Reason.Triggers[0].WorkItemID != work.ID {
			t.Fatal("commissioning review lost its draft scope provenance")
		}
		if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 1); err != nil {
			t.Fatal(err)
		}
		var retired bool
		if err := rawPool(t).QueryRow(ctx, `SELECT retired_at IS NOT NULL FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, orgA, projectA, item.ID).Scan(&retired); err != nil || !retired {
			t.Fatalf("undo failed to retire untouched review: %v, %v", retired, err)
		}
		return
	}
	t.Fatal("missing mechanical commissioning proposal")
}

func TestDeliveryProposalUndoRefusesEditedOrUsed(t *testing.T) {
	for _, mode := range []string{"edited", "dependency", "report"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			s, p := deliveryProposal(t, "approval")
			pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Review", LifecycleStatus: "planned"})
			if err != nil {
				t.Fatal(err)
			}
			choice := store.DeliveryAcceptance{PackageID: pkg.ID, StageID: pkg.Stages[0].ID}
			item, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, choice)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.AcceptDeliveryProposal(ctx, orgA, projectA, p.Key, userA, p.InputsFingerprint, store.DeliveryAcceptance{}); !errors.Is(err, store.ErrVersionConflict) {
				t.Fatalf("changed retry assignment %v", err)
			}
			switch mode {
			case "edited":
				title := "Human correction"
				_, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: 1, Title: &title}})
			case "dependency":
				other, e := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "milestone", Title: "Following work"}})
				if e != nil {
					t.Fatal(e)
				}
				_, err = rawPool(t).Exec(ctx, `INSERT INTO delivery_dependencies(org_id,project_id,predecessor_id,successor_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, orgA, projectA, item.ID, other.ID)
			case "report":
				r, e := s.CreateReport(ctx, orgA, projectA, userA, "rfp", pkg.ID)
				if e != nil {
					t.Fatal(e)
				}
				_, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.UndoProposalDecision(ctx, orgA, projectA, p.Key, userA, 1); !errors.Is(err, store.ErrProposalUndoBlocked) {
				t.Fatalf("%s undo %v", mode, err)
			}
			live, err := s.ReadDelivery(ctx, orgA, projectA)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, v := range live {
				found = found || v.ID == item.ID
			}
			if !found {
				t.Fatal("blocked undo changed item")
			}
		})
	}
}
