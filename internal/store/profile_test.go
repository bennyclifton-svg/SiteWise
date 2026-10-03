package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/profile"
	"sitewise/internal/store"
)

func profileStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, orgID := range []string{orgA, orgB} {
		if err := st.DeleteOrg(ctx, orgID); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, orgID := range []string{orgA, orgB} {
			_ = st.DeleteOrg(context.Background(), orgID)
		}
	})
	if err := seedTenants(ctx, st); err != nil {
		t.Fatal(err)
	}
	return st
}

func conf(v float64) *float64 { return &v }

func TestProfileStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)

	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil || whole.Kind != "whole" {
		t.Fatalf("whole %+v %v", whole, err)
	}
	again, _ := st.EnsureWholePart(ctx, orgA, projectA)
	if again.ID != whole.ID {
		t.Fatal("whole part created twice")
	}

	facts := []store.StoredFact{
		{PassageID: passageA, QuestionID: "sys.hydraulic.gas.presence", Value: "included", Excerpt: "Two 45 kg LPG gas bottles", Confidence: conf(0.9), DecidedBy: "jev"},
		{PassageID: passageA, QuestionID: "det.bal", Value: "BAL-40", Confidence: conf(0.8), DecidedBy: "jev"},
	}
	for i := 0; i < 2; i++ { // a rerun replaces, never duplicates
		if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"sys."}, "profile-1", facts); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"det."}, "profile-1", facts); err != nil {
		t.Fatal(err)
	}
	snap, err := st.ProfileInput(ctx, orgA, projectA)
	if err != nil || len(snap.Facts) != 2 {
		t.Fatalf("facts %+v %v", snap.Facts, err)
	}

	compute := func(s store.ProfileSnapshot) []profile.Row {
		var rows []profile.Row
		for _, f := range s.Facts {
			rows = append(rows, profile.Row{PartID: s.Parts[0].ID, Key: f.QuestionID, Value: f.Value, Band: "amber",
				Sources: []profile.Source{{DocumentID: f.DocumentID, Excerpt: f.Excerpt}}})
		}
		for _, u := range s.User {
			rows = append(rows, profile.Row{PartID: u.PartID, Key: u.Key, Value: *u.Value, Band: "user"})
		}
		return rows
	}
	if err := st.RebuildProfile(ctx, orgA, projectA, "profile-1", compute); err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "hdr.subclass", strPtr("warehouse"), "set by test"); err != nil {
		t.Fatal(err)
	}
	if err := st.RebuildProfile(ctx, orgA, projectA, "profile-1", compute); err != nil {
		t.Fatal(err)
	}
	view, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || view.BuiltAt == nil || len(view.Parts) != 1 || len(view.Rows) != 3 {
		t.Fatalf("view %+v %v", view, err)
	}
	if view.Rows[0].Sources == nil {
		t.Fatal("sources lost in JSON round trip")
	}

	part, err := st.CreatePart(ctx, orgA, projectA, "Building B", "building", "2")
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := st.UpdatePart(ctx, orgA, projectA, part.ID, strPtr("Building B (mixed use)"), nil, nil)
	if err != nil || renamed.Label != "Building B (mixed use)" || renamed.NCCClass != "2" {
		t.Fatalf("rename %+v %v", renamed, err)
	}
}

func TestProfileOrgIsolation(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureWholePart(ctx, orgA, projectB); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org whole part: %v", err)
	}
	if _, err := st.ReadProfile(ctx, orgB, projectA, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org read: %v", err)
	}
	if _, err := st.CreatePart(ctx, orgB, projectA, "X", "building", ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org part: %v", err)
	}
	if _, err := st.UpdatePart(ctx, orgB, projectA, whole.ID, strPtr("X"), nil, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org rename: %v", err)
	}
	if err := st.SetUserValue(ctx, orgB, projectA, whole.ID, userB, "hdr.subclass", strPtr("x"), ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org user value: %v", err)
	}
	if err := st.ReplaceDocumentFacts(ctx, orgB, docA, []string{"sys."}, "profile-1", nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org facts: %v", err)
	}
	snap, err := st.ProfileInput(ctx, orgB, projectA)
	if err != nil || len(snap.Parts)+len(snap.Facts)+len(snap.User) != 0 {
		t.Fatalf("cross-org snapshot %+v %v", snap, err)
	}
}

func strPtr(s string) *string { return &s }
