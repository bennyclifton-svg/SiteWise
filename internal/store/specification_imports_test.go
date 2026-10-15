package store_test

import (
	"context"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

type specificationAsker struct{ calls int }

func (a *specificationAsker) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	a.calls++
	return jev.Result{Answers: map[string]jev.Answer{"item": {Type: jev.TypeChoice, Choice: "yes"}, "kind": {Type: jev.TypeChoice, Choice: "finish"}, "meaning": {Type: jev.TypeChoice, Choice: "selection"}, "system": {Type: jev.TypeChoice, Choice: "interiors"}, "match": {Type: jev.TypeChoice, Choice: "starter:paint"}}}, nil
}
func TestSpecificationImportRetriesPreserveDecisionsAndEvidence(t *testing.T) {
	s, _, _ := workStore(t)
	ctx := context.Background()
	source := store.DocumentSource{Pages: 1, Source: []store.SourcePage{{Page: 1, Location: "Page 1", Text: "P1\nInterior paint\nColour: White\nFinish: Low sheen"}}}
	if err := s.ReplaceSource(ctx, orgA, docA, source); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestSpecificationImport(ctx, orgA, projectA, userA, docA, ""); err != nil {
		t.Fatal(err)
	}
	ask := &specificationAsker{}
	w := jobs.Worker{Store: s, Ask: ask, Kinds: []string{store.JobKindSpecification}}
	if err := w.Once(ctx, orgA); err != nil {
		t.Fatal(err)
	}
	before, err := s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Types) != 1 || len(before.Assertions) != 3 || ask.calls != 1 {
		t.Fatalf("import coverage/calls: types %d assertions %d calls %d", len(before.Types), len(before.Assertions), ask.calls)
	}
	if before.Types[0].SystemID != "interiors.painting" || before.Types[0].StarterKey != "paint" || before.Types[0].ReviewStatus != "proposed" {
		t.Fatal("mapping was lost or automatically accepted", before.Types[0])
	}
	selected := before.Assertions[0]
	if selected.Source.Hash == "" || selected.Source.Marker != "unreviewed" {
		t.Fatal("lost source review state", selected)
	}
	if _, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, selected.ID, store.SpecificationReview{Version: 1, Status: "accepted", SourceReviewed: true}); err != nil {
		t.Fatal(err)
	}
	if err = s.ReplaceSource(ctx, orgA, docA, source); err != nil {
		t.Fatal(err)
	}
	if err = s.RequestSpecificationImport(ctx, orgA, projectA, userA, docA, ""); err != nil {
		t.Fatal(err)
	}
	if err = w.Once(ctx, orgA); err != nil {
		t.Fatal(err)
	}
	after, err := s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Types) != 1 || len(after.Assertions) != 3 || ask.calls != 1 {
		t.Fatal("retry duplicated records or repeated the cached judgement")
	}
	found := false
	for _, a := range after.Assertions {
		if a.ID == selected.ID {
			found = a.Status == "accepted" && a.Source.Hash == selected.Source.Hash
		}
	}
	if !found {
		t.Fatal("re-extraction replaced the user's accepted assertion")
	}
}
