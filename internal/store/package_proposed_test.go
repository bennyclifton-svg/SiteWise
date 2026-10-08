package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestProposedHeadContractAcceptAndRoute(t *testing.T) {
	ctx := context.Background()
	s, _, whole := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: whole, SystemID: "mechanical.air-conditioning", Action: "replace", Title: "Chiller"}); err != nil {
		t.Fatal(err)
	}
	o, err := s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if o.Route.Value != "" || !o.Route.BuilderDesigns || !o.Route.Confirm {
		t.Fatalf("unanswered route must assume D&C and ask: %+v", o.Route)
	}
	if o.ProposedWork == nil || o.ProposedWork.PackageID != "" || len(o.ProposedWork.Lines) != 2 {
		t.Fatalf("head contract proposal: %+v", o.ProposedWork)
	}
	if _, err := s.AcceptProposedPackage(ctx, orgB, projectA, userB, procurement.WorksProposalKey); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org: %v", err)
	}
	if _, err := s.AcceptProposedPackage(ctx, orgA, projectA, userB, procurement.WorksProposalKey); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign actor: %v", err)
	}
	p, err := s.AcceptProposedPackage(ctx, orgA, projectA, userA, procurement.WorksProposalKey)
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != "works" || p.WorksScope != "head_contract" || p.Origin != "calculation" {
		t.Fatalf("accepted %+v", p)
	}
	scope, err := s.ReadPackageScope(ctx, orgA, projectA, p.ID)
	if err != nil || len(scope) != 2 {
		t.Fatalf("scope %+v %v", scope, err)
	}
	o, err = s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil || o.ProposedWork != nil || len(o.Gaps) != 0 {
		t.Fatalf("after accept: %+v %+v %v", o.ProposedWork, o.Gaps, err)
	}
	if _, err := s.AcceptProposedPackage(ctx, orgA, projectA, userA, procurement.WorksProposalKey); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second accept must find nothing to add: %v", err)
	}

	// A new work item becomes an addition to the accepted head contract.
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: whole, SystemID: "fire-active", Action: "retain", Title: "Hydrants"}); err != nil {
		t.Fatal(err)
	}
	o, err = s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil || o.ProposedWork == nil || o.ProposedWork.PackageID != p.ID || len(o.ProposedWork.Lines) != 1 || o.ProposedWork.Lines[0].Role != "protect" {
		t.Fatalf("addition: %+v %v", o.ProposedWork, err)
	}

	// A traditional route keeps design with consultants and stops novation.
	route := "traditional"
	if _, err := s.SetUserValue(ctx, orgA, projectA, whole, userA, "hdr.cond.procurement_route", store.UserWrite{Scope: "site", Value: &route}); err != nil {
		t.Fatal(err)
	}
	o, err = s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil || o.Route.Value != "traditional" || o.Route.BuilderDesigns || o.Route.Confirm {
		t.Fatalf("stated route: %+v %v", o.Route, err)
	}
}
