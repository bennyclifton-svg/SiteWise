package store_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/store"
	"sitewise/internal/works"
)

func TestIntegratedProposalRebuildMatchesAndRollsBack(t *testing.T) {
	ctx := context.Background()
	s, build, part := workStore(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"}); err != nil {
		t.Fatal(err)
	}
	evaluator, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	read := func() (store.ProfileView, []store.ProposalView) {
		t.Helper()
		profile, err := s.ReadProfile(ctx, orgA, projectA, nil)
		if err != nil {
			t.Fatal(err)
		}
		proposals, err := s.ReadProposals(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		return profile, proposals
	}
	before, proposals := read()
	if len(proposals) == 0 {
		t.Fatal("fixture needs proposals")
	}
	if err := s.RebuildProfileWithProposals(ctx, orgA, projectA, evaluator); err != nil {
		t.Fatal(err)
	}
	after, integrated := read()
	if !reflect.DeepEqual(proposals, integrated) || !reflect.DeepEqual(before.Rows, after.Rows) || before.InputFingerprint != after.InputFingerprint || after.Revision != before.Revision+1 {
		t.Fatal("integrated rebuild differs from separate projections")
	}
	if err := s.RebuildProfileWithProposals(ctx, orgB, projectA, evaluator); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign rebuild: %v", err)
	}
	if err := s.RebuildProfileWithProposals(ctx, orgA, projectA, nil); !errors.Is(err, store.ErrProposalUnavailable) {
		t.Fatalf("nil evaluator: %v", err)
	}
	// Fail proposal persistence after the profile rows have been replaced.
	// The profile build timestamp/revision and both projections must roll back.
	pool := rawPool(t)
	var eventsBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1`, orgA).Scan(&eventsBefore); err != nil {
		t.Fatal(err)
	}
	records := build.Catalog.InterfaceConsequences()
	for i := range records {
		records[i].Propose.Label = ""
	}
	evaluator, err = works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfileWithProposals(ctx, orgA, projectA, evaluator); err == nil {
		t.Fatal("invalid proposal persisted")
	}
	failed, failedProposals := read()
	if !reflect.DeepEqual(after, failed) || !reflect.DeepEqual(integrated, failedProposals) {
		t.Fatal("failed integrated rebuild changed saved projections")
	}
	var eventsAfter int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1`, orgA).Scan(&eventsAfter); err != nil {
		t.Fatal(err)
	}
	if eventsAfter != eventsBefore {
		t.Fatal("failed rebuild emitted an event")
	}
	// A catalogue changed after compilation forces the concurrent evaluator
	// to fail. The write goroutine must join it and roll back its profile work.
	for i := range records {
		records[i].ID += "-not-compiled"
	}
	err = s.RebuildProfileWithProposals(ctx, orgA, projectA, evaluator)
	if err == nil || !strings.Contains(err.Error(), "proposal record not in evaluator catalogue") {
		t.Fatalf("expected evaluator failure, got %v", err)
	}
	failed, failedProposals = read()
	if !reflect.DeepEqual(after, failed) || !reflect.DeepEqual(integrated, failedProposals) {
		t.Fatal("evaluator failure changed saved projections")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1`, orgA).Scan(&eventsAfter); err != nil {
		t.Fatal(err)
	}
	if eventsAfter != eventsBefore {
		t.Fatal("evaluator failure emitted an event")
	}
}
