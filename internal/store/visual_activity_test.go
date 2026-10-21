package store_test

import (
	"context"
	"testing"
)

func TestVisualActivityTracksLiveLease(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE documents SET filename='activity.pdf',status='filed' WHERE org_id=$1 AND id=$2;`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if err := st.QueueVisualIdentity(ctx, orgA, docA); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ status, lease, want string }{
		{"queued", "1 minute", "queued"},
		{"leased", "1 minute", "reading"},
		{"leased", "-1 second", "queued"},
		{"failed", "-1 second", "unavailable"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE jobs SET status=$3,locked_until=now()+$4::interval WHERE org_id=$1 AND document_id=$2 AND kind='visual_identity'`, orgA, docA, tc.status, tc.lease); err != nil {
			t.Fatal(err)
		}
		review, err := st.VisualIdentity(ctx, orgA, docA)
		if err != nil || review.Status != tc.want {
			t.Fatalf("review %s: %+v %v", tc.status, review, err)
		}
		doc, err := st.DocumentView(ctx, orgA, docA)
		if err != nil || doc.VisualStatus != tc.want {
			t.Fatalf("register %s: %+v %v", tc.status, doc, err)
		}
	}
}
