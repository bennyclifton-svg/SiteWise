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
