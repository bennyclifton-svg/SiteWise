package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

func workStore(t *testing.T) (*store.Store, store.ProfileBuild, string) {
	t.Helper()
	s := profileStore(t)
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	part, err := s.EnsureWholePart(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	b := store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion, Compute: func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: profile.Thresholds{Amber: map[string]float64{"presence": .6, "header": .6, "action": .8}}}, cat)
	}}
	return s.WithProfile(b), b, part.ID
}

func TestWorkScopeOwnershipRebuildAndConcurrency(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "hdr.work_type", store.UserWrite{Scope: "project", Value: strPtr("extend")}); err != nil {
		t.Fatal(err)
	}
	facts := []store.StoredFact{{PassageID: passageA, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Confidence: conf(.9), DecidedBy: "jev", Excerpt: "Gas included"}}
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	rebuild := func() {
		t.Helper()
		if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
			t.Fatal(err)
		}
	}
	rebuild()
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 1 {
		t.Fatalf("initial %+v %v", items, err)
	}
	initial := items[0]
	if initial.Action != "new" || initial.ReviewStatus != "proposed" || initial.UserTouched || initial.ID != works.CoarseID(projectA, part, "hydraulic.gas") {
		t.Fatalf("proposal %+v", initial)
	}
	rebuild()
	same, _ := s.ReadWorks(ctx, orgA, projectA)
	if same[0].Version != initial.Version {
		t.Fatal("unchanged rebuild rewrote item")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": strPtr("out")})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].ID != initial.ID || items[0].Inclusion != "excluded" || !items[0].UserTouched {
		t.Fatalf("lost exclusion %+v", items)
	}
	rebuild()
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if items[0].Inclusion != "excluded" {
		t.Fatal("rebuild overwrote user")
	}
	view, err := s.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range view.Rows {
		if r.Key == "scope.hydraulic.gas" {
			found = r.Value == "out" && r.Band == "user" && r.Note != ""
		}
	}
	if !found {
		t.Fatal("compatibility projection lost exclusion or evidence conflict")
	}
	snap, err := s.ProfileInput(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range snap.User {
		if u.Key == "scope.hydraulic.gas" {
			t.Fatal("second scope authority remains")
		}
	}
	if err := s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": nil}); err != nil {
		t.Fatal(err)
	}
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].ID != initial.ID || items[0].UserTouched || items[0].Inclusion != "included" {
		t.Fatalf("reset %+v", items)
	}
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, nil); err != nil {
		t.Fatal(err)
	}
	rebuild()
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 0 {
		t.Fatalf("unsupported proposal stayed live %+v", items)
	}
	var retired bool
	if err := rawPool(t).QueryRow(ctx, `SELECT retired_at IS NOT NULL FROM work_items WHERE org_id=$1 AND id=$2`, orgA, initial.ID).Scan(&retired); err != nil || !retired {
		t.Fatalf("proposal deleted instead of retired: %v %v", retired, err)
	}
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	rebuild()
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].ID != initial.ID {
		t.Fatal("support did not revive stable ID")
	}
}

func TestWorkItemsRejectWrongOwnerAndDuplicate(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	item := works.Item{PartID: part, SystemID: "structure", Action: "investigate"}
	got, err := s.CreateWorkItem(ctx, orgA, projectA, userA, item)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, item); !errors.Is(err, store.ErrWorkConflict) {
		t.Fatalf("duplicate %v", err)
	}
	if _, err := s.CreateWorkItem(ctx, orgB, projectA, userB, item); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign project %v", err)
	}
	if _, err := s.CreateWorkItem(ctx, orgB, projectB, userB, item); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign part %v", err)
	}
	if _, err := s.ReadWorks(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, _ := s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].ID != got.ID || !items[0].UserTouched {
		t.Fatal("user item retired by rebuild")
	}
	broken := b
	broken.Compute = func(store.ProfileSnapshot) []profile.Row {
		return []profile.Row{{PartID: part, Key: "bad", Band: "invalid"}}
	}
	item.SystemID = "hydraulic"
	if _, err := s.WithProfile(broken).CreateWorkItem(ctx, orgA, projectA, userA, item); err == nil {
		t.Fatal("failed rebuild committed")
	}
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 {
		t.Fatal("partial item commit")
	}
}

func TestExistingPresenceStaysOnSiteUntilScopeChosen(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "hdr.work_type", store.UserWrite{Scope: "project", Value: strPtr("refurb")}); err != nil {
		t.Fatal(err)
	}
	facts := []store.StoredFact{{PassageID: passageA, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Confidence: conf(.9), DecidedBy: "jev", Excerpt: "Gas is present"}}
	if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 0 {
		t.Fatalf("existing system became work %+v %v", items, err)
	}
	view, err := s.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range view.Rows {
		if r.Key == "sys.hydraulic.gas.existing" {
			found = r.Scope == "site" && r.Value == "present" && len(r.Sources) == 1
		}
	}
	if !found {
		t.Fatal("site presence or provenance missing")
	}
	if err := s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": strPtr("in")}); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 1 || items[0].Action != "alter" || !items[0].UserTouched {
		t.Fatalf("explicit works choice %+v %v", items, err)
	}
	if err := s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": nil}); err != nil {
		t.Fatal(err)
	}
	items, err = s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 0 {
		t.Fatalf("reset revived presence as work %+v %v", items, err)
	}
}

func TestDocumentActionChangesOnlyUntouchedWork(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	apply := func(action string) {
		t.Helper()
		facts := []store.StoredFact{
			{PassageID: passageA, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Confidence: conf(.9), DecidedBy: "jev"},
			{PassageID: passageA, QuestionID: "sys.hydraulic.gas.action", Value: action, Confidence: conf(.9), DecidedBy: "jev"},
		}
		if err := s.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
			t.Fatal(err)
		}
		if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
			t.Fatal(err)
		}
	}
	apply("replace")
	items, err := s.ReadWorks(ctx, orgA, projectA)
	if err != nil || len(items) != 1 || items[0].Action != "replace" {
		t.Fatalf("action not applied: %+v %v", items, err)
	}
	id := items[0].ID
	apply("repair")
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].ID != id || items[0].Action != "repair" {
		t.Fatalf("proposal did not update: %+v", items)
	}
	if err := s.SetScope(ctx, orgA, projectA, part, userA, map[string]*string{"scope.hydraulic.gas": strPtr("in")}); err != nil {
		t.Fatal(err)
	}
	apply("replace")
	items, _ = s.ReadWorks(ctx, orgA, projectA)
	if len(items) != 1 || items[0].Action != "repair" || !items[0].UserTouched {
		t.Fatalf("accepted item overwritten: %+v", items)
	}
	view, err := s.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil {
		t.Fatal(err)
	}
	conflict := false
	for _, r := range view.Rows {
		if r.Key == "scope.hydraulic.gas" {
			conflict = r.Note != ""
		}
	}
	if !conflict {
		t.Fatal("accepted-action conflict hidden")
	}
}
