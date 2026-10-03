package store_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/store"
)

func documentExists(t *testing.T, org, id string) bool {
	t.Helper()
	var ok bool
	if err := rawPool(t).QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM documents WHERE org_id=$1 AND id=$2)`, org, id).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestDeleteDocumentsRemovesTheDocumentAndItsUnsharedFile(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	if err := st.EnqueueStage(ctx, orgA, specA, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	res, err := st.DeleteDocuments(ctx, orgA, projectA, []string{specA}, userA)
	if err != nil {
		t.Fatal(err)
	}
	if documentExists(t, orgA, specA) || !documentExists(t, orgA, docA) {
		t.Fatal("wrong documents deleted")
	}
	if len(res.Deleted) != 1 || res.Deleted[0] != specA || len(res.Unreferenced) != 1 || !bytes.Equal(res.Unreferenced[0], bytes32(0x51)) {
		t.Fatalf("result %+v", res)
	}
	if jobStatus(t, orgA, specA, store.JobKindLabel) != "" {
		t.Fatal("job of a deleted document survived")
	}
	var events int
	if err := rawPool(t).QueryRow(ctx, `SELECT count(*) FROM events WHERE org_id=$1 AND kind='deleted' AND document_id=$2`, orgA, specA).Scan(&events); err != nil || events != 1 {
		t.Fatalf("deleted events %d %v", events, err)
	}
	// Both tenants' seed files share bytes: the blob stays referenced.
	used, err := st.BlobReferenced(ctx, bytes32(0xab))
	if err != nil || !used {
		t.Fatalf("shared blob referenced=%v %v", used, err)
	}
	if used, err := st.BlobReferenced(ctx, bytes32(0x51)); err != nil || used {
		t.Fatalf("deleted blob referenced=%v %v", used, err)
	}
}

func TestDeletingADrawingSetDeletesItsSheetsButNotTheOtherWayRound(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	const sheet2 = "11111111-1111-4111-8111-aaaaaaaaab23"
	addDocument(t, st, orgA, projectA, sheetA, 0x52, "drawing")
	addDocument(t, st, orgA, projectA, sheet2, 0x53, "drawing")
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO drawing_expansions (org_id, source_id, page_count, status) VALUES ($1, $2, 2, 'complete')`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO drawing_sheets (org_id, source_id, page_number, document_id) VALUES ($1, $2, 1, $3), ($1, $2, 2, $4)`, orgA, docA, sheetA, sheet2); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, []string{sheet2}, userA); err != nil {
		t.Fatal(err)
	}
	if documentExists(t, orgA, sheet2) || !documentExists(t, orgA, docA) || !documentExists(t, orgA, sheetA) {
		t.Fatal("deleting one sheet must keep the set and its other sheets")
	}
	res, err := st.DeleteDocuments(ctx, orgA, projectA, []string{docA}, userA)
	if err != nil || len(res.Deleted) != 2 {
		t.Fatalf("set delete %+v %v", res, err)
	}
	if documentExists(t, orgA, docA) || documentExists(t, orgA, sheetA) {
		t.Fatal("a deleted set left its sheet behind")
	}
}

func TestDeletingANewerRevisionMakesThePriorCurrent(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "drawing")
	if _, err := rawPool(t).Exec(ctx, `INSERT INTO supersessions (org_id, document_id, prior_document_id) VALUES ($1, $2, $3)`, orgA, specA, docA); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, []string{specA}, userA); err != nil {
		t.Fatal(err)
	}
	var superseded bool
	if err := rawPool(t).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM supersessions WHERE org_id=$1 AND prior_document_id=$2)`, orgA, docA).Scan(&superseded); err != nil || superseded {
		t.Fatalf("prior revision still superseded: %v %v", superseded, err)
	}
}

func TestDeleteIsOrgScopedAndAllOrNothing(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	if _, err := st.DeleteDocuments(ctx, orgB, projectA, []string{docA}, userB); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's project: %v", err)
	}
	if _, err := st.DeleteDocuments(ctx, orgB, projectB, []string{docA}, userB); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's document through own project: %v", err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, []string{docA, docB}, userA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("mixed ids: %v", err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, nil, userA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("no ids: %v", err)
	}
	if !documentExists(t, orgA, docA) || !documentExists(t, orgB, docB) {
		t.Fatal("a rejected delete removed a document")
	}
}

func TestAJobWhoseDocumentIsDeletedEndsWithoutRetry(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	if err := st.EnqueueStage(ctx, orgA, specA, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	job, err := st.ClaimJob(ctx, orgA, time.Minute, []string{store.JobKindLabel})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, []string{specA}, userA); err != nil {
		t.Fatal(err)
	}
	if err := st.FailJob(ctx, orgA, job.ID, job.Token, "writes failed", 0, store.EventWrite{}); !errors.Is(err, store.ErrLeaseLost) {
		t.Fatalf("finishing a deleted document's job: %v", err)
	}
	if jobStatus(t, orgA, specA, store.JobKindLabel) != "" {
		t.Fatal("a deleted document's job would be retried")
	}
}

func TestRestoreCheckAcceptsDeletedDocumentsButNotLostOnes(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	if _, err := st.AppendEvent(ctx, orgA, "filing", specA, "{}"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteDocuments(ctx, orgA, projectA, []string{specA}, userA); err != nil {
		t.Fatal(err)
	}
	orphans := func() int64 {
		t.Helper()
		facts, err := st.RestoreFacts(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range facts.CrossOrg {
			if c.Reference == "events.document_id" {
				return c.Rows
			}
		}
		t.Fatal("event reference check did not run")
		return 0
	}
	if n := orphans(); n != 0 {
		t.Fatalf("a deleted document's history counted as broken: %d", n)
	}
	// A document that vanished without a deletion event is still reported.
	if _, err := rawPool(t).Exec(ctx, `DELETE FROM documents WHERE org_id=$1 AND id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(ctx, orgA, "filing", docA, "{}"); err != nil {
		t.Fatal(err)
	}
	if n := orphans(); n == 0 {
		t.Fatal("a lost document was not reported")
	}
}
