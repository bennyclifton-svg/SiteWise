package store_test

import (
	"context"
	"testing"
)

func TestProfileReadCountsPreservePoliciesMissingKindsAndSkippedTie(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	addDocument(t, st, orgA, projectA, sheetA, 0x52, "report")
	if _, err := pool.Exec(ctx, `DELETE FROM decisions WHERE org_id=$1 AND document_id=$2 AND field='kind'`, orgA, sheetA); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name          string
		kinds         []string
		read, skipped int64
		kind          string
	}{
		{"automatic", readKinds, 1, 2, "drawing"},
		{"unfiltered", nil, 3, 0, ""},
		{"no matches tie", []string{"other"}, 0, 3, "drawing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := st.ReadProfile(ctx, orgA, projectA, tc.kinds)
			if err != nil || int64(got.ReadDocuments) != tc.read || int64(got.SkippedDocuments) != tc.skipped || got.SkippedKind != tc.kind {
				t.Fatalf("got %+v err %v", got, err)
			}
		})
	}
	if _, err := pool.Exec(ctx, `UPDATE documents SET profile_read=CASE WHEN id=$2 THEN 'read' ELSE 'skip' END WHERE org_id=$1 AND project_id=$3`, orgA, sheetA, projectA); err != nil {
		t.Fatal(err)
	}
	got, err := st.ReadProfile(ctx, orgA, projectA, nil)
	if err != nil || got.ReadDocuments != 1 || got.SkippedDocuments != 2 || got.SkippedKind != "drawing" {
		t.Fatalf("explicit override %+v %v", got, err)
	}
}

func TestProfileCompletedCountsRequireFinishedReading(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	textReady(t, st, orgA, docA)
	check := func(want int) {
		t.Helper()
		v, err := st.ReadProfile(ctx, orgA, projectA, nil)
		if err != nil || v.CompletedDocuments != want {
			t.Fatalf("completed=%d want=%d err=%v", v.CompletedDocuments, want, err)
		}
	}
	check(0) // Selected and text-ready is not read.
	for _, kind := range []string{"label", "evidence"} {
		if err := st.EnqueueStage(ctx, orgA, docA, kind); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	check(0) // Evidence still queued.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2 AND kind='evidence'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	check(1)
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='failed' WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	check(0) // A failed reread must not appear complete.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2 AND kind='label'`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	check(1)
	if _, err := st.SetProfileReading(ctx, orgA, projectA, []string{docA}, "skip", nil); err != nil {
		t.Fatal(err)
	}
	check(0)
	v, err := st.ReadProfile(ctx, orgB, projectA, nil)
	if err == nil || v.CompletedDocuments != 0 {
		t.Fatal("cross-org profile exposed")
	}
}
