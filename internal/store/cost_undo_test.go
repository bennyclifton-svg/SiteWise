package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestCostHistoryBlocksProposalUndo(t *testing.T) {
	for _, kind := range []string{"discipline", "obligation"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s, b, part := workStore(t)
			for i := range b.Catalog.InterfaceConsequences() {
				r := &b.Catalog.InterfaceConsequences()[i]
				if r.ID == "ic.loads-investigate-supported" {
					r.Propose.Kind = kind
				}
			}
			if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
				t.Fatal(err)
			}
			evaluator, err := works.NewEvaluator(b.Catalog)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
				t.Fatal(err)
			}
			proposals, err := s.ReadProposals(ctx, orgA, projectA)
			if err != nil {
				t.Fatal(err)
			}
			var proposal store.ProposalView
			for _, p := range proposals {
				if p.RecordID == "ic.loads-investigate-supported" && p.InterfaceID == "if.plant-loads-structure" {
					proposal = p
					break
				}
			}
			if proposal.Key == "" {
				t.Fatal("missing proposal")
			}
			var pkg procurement.Package
			var scopeID string
			if kind == "discipline" {
				pkg, err = s.AcceptPackageProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint)
			} else {
				pkg, err = s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Engineering", LifecycleStatus: "planned"})
				if err != nil {
					t.Fatal(err)
				}
				scope, e := s.AcceptScopeProposal(ctx, orgA, projectA, proposal.Key, userA, proposal.InputsFingerprint, store.ScopeAcceptance{PackageID: pkg.ID, StageID: pkg.Stages[0].ID})
				err = e
				scopeID = scope.ID
			}
			if err != nil {
				t.Fatal(err)
			}
			in := ci("Engineering fee")
			in.LineKind = "fee"
			in.Category = ""
			in.PackageID = pkg.ID
			in.PackageStageID = pkg.Stages[0].ID
			plan, err := s.CreateCostItem(ctx, orgA, projectA, userA, in)
			if err != nil {
				t.Fatal(err)
			}
			lineID := plan.Items[0].ID
			if scopeID != "" {
				plan, err = s.LinkScopeCost(ctx, orgA, projectA, userA, store.CostLinkInput{CostWrite: cw(plan), CostItemID: lineID, ScopeItemID: scopeID})
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = s.UndoProposalDecision(ctx, orgA, projectA, proposal.Key, userA, 1); !errors.Is(err, store.ErrProposalUndoBlocked) {
				t.Fatalf("live cost undo: %v", err)
			}
			plan, err = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(plan))
			if err != nil {
				t.Fatal(err)
			}
			plan, err = s.RemoveCostItem(ctx, orgA, projectA, lineID, userA, cw(plan))
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Items) != 0 {
				t.Fatal("draft should be empty")
			}
			if err = s.UndoProposalDecision(ctx, orgA, projectA, proposal.Key, userA, 1); !errors.Is(err, store.ErrProposalUndoBlocked) {
				t.Fatalf("historical cost undo: %v", err)
			}
		})
	}
}
