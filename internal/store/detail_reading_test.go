package store_test

import (
	"context"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

// A client, address and flag read from a document are stored, become
// profile rows with their source and owner, and give way to the user's word.
func TestDetailReadingsBecomeProfileRowsAndYieldToTheUser(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	whole, err := st.EnsureWholePart(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	compute := func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User,
			Thresholds: profile.Thresholds{Amber: map[string]float64{"detail": .6}}}, cat)
	}
	facts := []store.StoredFact{
		{PassageID: passageA, QuestionID: "proj.client", Value: "Joins Win Pty Ltd", Excerpt: "Principal: Joins Win Pty Ltd ABN", Confidence: conf(.9), DecidedBy: "jev"},
		{PassageID: passageA, QuestionID: "site.address", Value: "568-572 Parramatta Road Petersham", Excerpt: "568-572 Parramatta Road Petersham", Confidence: conf(.9), DecidedBy: "jev"},
		{PassageID: passageA, QuestionID: "proj.flag.basement", Value: "yes", Confidence: conf(.9), DecidedBy: "jev"},
	}
	if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"proj.", "site."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	rebuild := func() map[string]profile.Row {
		t.Helper()
		if err := st.RebuildProfile(ctx, orgA, projectA, "detail", compute); err != nil {
			t.Fatal(err)
		}
		view, err := st.ReadProfile(ctx, orgA, projectA, nil)
		if err != nil {
			t.Fatal(err)
		}
		rows := map[string]profile.Row{}
		for _, r := range view.Rows {
			rows[r.Key] = r
		}
		return rows
	}
	rows := rebuild()
	for key, want := range map[string]struct{ value, scope string }{
		"proj.client": {"Joins Win Pty Ltd", "project"}, "site.address": {"568-572 Parramatta Road Petersham", "site"}, "proj.flag.basement": {"yes", "project"},
	} {
		r, ok := rows[key]
		if !ok || r.Value != want.value || r.Scope != want.scope || r.Band != "amber" || r.PartID != whole.ID {
			t.Errorf("%s = %+v", key, r)
		}
	}
	if s := rows["proj.client"].Sources; len(s) != 1 || s[0].DocumentID != docA || s[0].Excerpt != "Principal: Joins Win Pty Ltd ABN" {
		t.Errorf("client source lost: %+v", s)
	}
	// The user's word wins and a rerun of the document keeps it.
	if _, err := st.SetUserValue(ctx, orgA, projectA, whole.ID, userA, "proj.client", store.UserWrite{Scope: "project", Value: strPtr("Joins Win Group")}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"proj.", "site."}, profile.QuestionVersion, facts); err != nil {
		t.Fatal(err)
	}
	if r := rebuild()["proj.client"]; r.Value != "Joins Win Group" || r.Band != "user" {
		t.Errorf("user word lost: %+v", r)
	}
	// Re-reading a document that no longer states them removes the readings.
	if err := st.ReplaceDocumentFacts(ctx, orgA, docA, []string{"proj.", "site."}, profile.QuestionVersion, nil); err != nil {
		t.Fatal(err)
	}
	rows = rebuild()
	if _, ok := rows["site.address"]; ok {
		t.Errorf("stale reading kept: %+v", rows["site.address"])
	}
}
