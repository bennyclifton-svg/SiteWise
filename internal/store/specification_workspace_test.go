package store_test

import (
	"context"
	"errors"
	"sitewise/internal/files"
	"sitewise/internal/jobs"
	"sitewise/internal/specifications"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
)

func TestSpecificationCoverageProtectsAuthorityAndPinsRevision(t *testing.T) {
	s, _, _ := workStore(t)
	ctx := context.Background()
	item, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Tile", Kind: "finish", SystemID: "interiors.tiling"})
	if err != nil {
		t.Fatal(err)
	}
	coverage := specifications.Coverage{SystemID: "interiors", Mode: "external", Extent: "All internal finishes", Maintainer: "Architect", References: []specifications.DocumentReference{{DocumentID: docA, Location: "Page 1"}}}
	c, err := s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, coverage)
	if err != nil {
		t.Fatal(err)
	}
	if c.References[0].Hash == "" {
		t.Fatal("revision not pinned")
	}
	coverage.References = []specifications.DocumentReference{{DocumentID: docB}}
	if _, err = s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, coverage); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign reference", err)
	}
	cell := store.SpecificationCell{TypeID: item.ID, Key: "colour", Meaning: "requirement", Value: specifications.Value{State: "known", Text: "Blue"}}
	if _, err = s.SaveSpecificationCell(ctx, orgA, projectA, userA, cell); err == nil {
		t.Fatal("external cell overwritten")
	}
	component, e := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Local component", Kind: "material"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddSpecificationRelationship(ctx, orgA, projectA, userA, specifications.Relationship{FromID: component.ID, ToID: item.ID, Kind: "component_of"}); e == nil {
		t.Fatal("external assembly changed via local component")
	}
	proposed, err := s.AddSpecificationAssertion(ctx, orgA, projectA, userA, specifications.Assertion{TypeID: item.ID, Key: "colour", Meaning: "requirement", Value: cell.Value})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, proposed.ID, store.SpecificationReview{Version: 1, Status: "accepted"}); err == nil {
		t.Fatal("review bypassed authority")
	}
	if _, err = s.AdoptSpecification(ctx, orgA, projectA, userA, item.ID, 1, "Agreed transfer"); err == nil {
		t.Fatal("adopt bypassed section authority")
	}
	snap, err := s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	blobs, _ := files.Open(t.TempDir(), 10<<20)
	issue, err := s.IssueSpecification(ctx, orgA, projectA, userA, "Reference register", "preliminary", snap.Revision, blobs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = rawPool(t).Exec(ctx, `UPDATE documents SET revision='B' WHERE org_id=$1 AND id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	snap, err = s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Coverage[0].References[0].Stale {
		t.Fatal("revision change invisible")
	}
	frozen, err := s.GetSpecificationIssue(ctx, orgA, projectA, issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Snapshot.Coverage[0].References[0].Revision == "B" {
		t.Fatal("issued reference mutated")
	}
	old := c
	c.Mode = "managed_here"
	c.Reason = "Architect handed maintenance to project manager"
	c, err = s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, old); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("stale coverage", err)
	}
	accepted, err := s.SaveSpecificationCell(ctx, orgA, projectA, userA, cell)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationCell(ctx, orgA, projectA, userA, cell); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("lost concurrent edit", err)
	}
	cell.ExpectedAcceptedID = accepted.ID
	cell.Value.Text = "Grey"
	if _, err = s.SaveSpecificationCell(ctx, orgA, projectA, userA, cell); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationCell(ctx, orgB, projectA, userB, cell); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign cell", err)
	}
	itemCoverage := specifications.Coverage{SystemID: item.SystemID, TypeID: item.ID, Mode: "managed_here"}
	if _, err = s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, itemCoverage); err != nil {
		t.Fatal(err)
	}
	itemCoverage.SystemID = "interiors"
	if _, err = s.SaveSpecificationCoverage(ctx, orgA, projectA, userA, itemCoverage); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("second item authority accepted", err)
	}
}

func TestSpecificationScopeReadsProfilerWithoutDivergence(t *testing.T) {
	s, _, part := workStore(t)
	ctx := context.Background()
	work, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "interiors.tiling", Title: "Tile works", Action: "new", Inclusion: "included"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Tile", Kind: "finish", SystemID: "interiors.tiling"})
	if err != nil {
		t.Fatal(err)
	}
	app, err := s.SaveSpecificationApplication(ctx, orgA, projectA, userA, specifications.Application{TypeID: item.ID, WorkItemID: work.ID, Location: "Contradiction", Action: "repair", Inclusion: "excluded", Selection: "undecided"})
	if err != nil {
		t.Fatal(err)
	}
	if app.Action != "new" || app.Inclusion != "included" || app.Location == "Contradiction" {
		t.Fatal("saved competing scope", app)
	}
	before, _ := s.ReadSpecifications(ctx, orgA, projectA)
	action := "replace"
	if _, err = s.PatchWorkItem(ctx, orgA, projectA, work.ID, userA, works.Patch{Version: work.Version, Action: &action}); err != nil {
		t.Fatal(err)
	}
	after, err := s.ReadSpecifications(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if after.Applications[0].Action != "replace" || after.Scope[0].Action != "replace" || after.Revision <= before.Revision {
		t.Fatal("profiler edit did not flow through", after)
	}
}

func TestSpecificationUpdateMappingAndExplicitAdoption(t *testing.T) {
	s, _, _ := workStore(t)
	ctx := context.Background()
	source := store.DocumentSource{Pages: 1, Source: []store.SourcePage{{Page: 1, Location: "Page 1", Text: "P1\nInterior paint\nColour: White\nFinish: Low sheen"}}}
	if err := s.ReplaceSource(ctx, orgA, docA, source); err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE documents SET status='filed' WHERE org_id=$1 AND id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if n, err := s.RequestSpecificationUpdate(ctx, orgA, projectA, userA); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if n, err := s.RequestSpecificationUpdate(ctx, orgA, projectA, userA); err != nil || n != 0 {
		t.Fatal("duplicate queue", n, err)
	}
	worker := jobs.Worker{Store: s, Ask: &specificationAsker{}, Kinds: []string{store.JobKindSpecification}}
	if err := worker.Once(ctx, orgA); err != nil {
		t.Fatal(err)
	}
	if n, err := s.RequestSpecificationUpdate(ctx, orgA, projectA, userA); err != nil || n != 0 {
		t.Fatal("completed requeued", n, err)
	}
	snap, _ := s.ReadSpecifications(ctx, orgA, projectA)
	candidate := snap.Types[0]
	if candidate.Authority != "external" {
		t.Fatal("source adopted automatically")
	}
	if _, err := s.AdoptSpecification(ctx, orgA, projectA, userA, candidate.ID, candidate.Version, ""); err == nil {
		t.Fatal("adoption without reason")
	}
	if _, err := s.AdoptSpecification(ctx, orgA, projectA, userA, candidate.ID, candidate.Version+1, "Agreed transfer"); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatal("stale adoption", err)
	}
	candidate, err := s.AdoptSpecification(ctx, orgA, projectA, userA, candidate.ID, candidate.Version, "Agreed transfer from consultant")
	if err != nil || candidate.Authority != "sitewise" || candidate.Origin != "document" || candidate.SourceDocumentID != docA {
		t.Fatal("adoption lost provenance", candidate, err)
	}
	local, err := s.SaveSpecificationType(ctx, orgA, projectA, userA, specifications.Type{Title: "Paint", Kind: "finish"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.LinkSpecificationCandidate(ctx, orgA, projectA, userA, candidate.ID, local.ID, candidate.Version); err != nil {
		t.Fatal(err)
	}
	if err = s.RequestSpecificationImport(ctx, orgA, projectA, userA, docA, ""); err != nil {
		t.Fatal(err)
	}
	if err = worker.Once(ctx, orgA); err != nil {
		t.Fatal(err)
	}
	snap, _ = s.ReadSpecifications(ctx, orgA, projectA)
	if len(snap.Assertions) != 3 {
		t.Fatal("retry duplicated mapped readings")
	}
	for _, a := range snap.Assertions {
		if a.TypeID != local.ID || a.Status != "proposed" {
			t.Fatal("mapping changed decision", a)
		}
	}
	for _, item := range snap.Types {
		if item.ID == candidate.ID && item.Status != "retired" {
			t.Fatal("retry resurrected mapped candidate")
		}
	}
	a := snap.Assertions[0]
	if _, err = s.ReviewSpecificationAssertion(ctx, orgA, projectA, userA, a.ID, store.SpecificationReview{Version: a.Version, Status: "accepted", SourceReviewed: true}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSpecificationCell(ctx, orgA, projectA, userA, store.SpecificationCell{TypeID: local.ID, Key: a.Key, Meaning: a.Meaning, Value: a.Value, ExpectedAcceptedID: a.ID}); err == nil {
		t.Fatal("confirmed source overwritten inline")
	}
}
