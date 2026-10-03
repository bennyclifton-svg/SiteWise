package store_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/store"
)

var readKinds = []string{"design_brief", "specification", "report"}

const (
	specA  = "11111111-1111-4111-8111-aaaaaaaaab21"
	sheetA = "11111111-1111-4111-8111-aaaaaaaaab22"
)

// addDocument files one more document with its own bytes and kind in a
// tenant's project.
func addDocument(t *testing.T, st *store.Store, org, project, id string, sum byte, kind string) {
	t.Helper()
	ctx := context.Background()
	file := id[:len(id)-2] + "f1"
	if err := st.CreateFile(ctx, org, store.File{ID: file, ProjectID: project, SHA256: bytes32(sum), ByteSize: 8, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDocument(ctx, org, store.Document{ID: id, ProjectID: project, FileID: file, Filename: id + ".pdf", Status: "filed"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDecision(ctx, org, store.Decision{ID: id[:len(id)-2] + "d1", DocumentID: id, Field: "kind", Value: kind, Band: "green", DecidedBy: "rule"}); err != nil {
		t.Fatal(err)
	}
}

// textReady marks a document's text as prepared, the state reading needs.
func textReady(t *testing.T, st *store.Store, org, doc string) {
	t.Helper()
	ctx := context.Background()
	if err := st.EnqueueStage(ctx, org, doc, store.JobKindFullText); err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2 AND kind='full_text'`, org, doc); err != nil {
		t.Fatal(err)
	}
}

func jobStatus(t *testing.T, org, doc, kind string) string {
	t.Helper()
	var status string
	err := rawPool(t).QueryRow(context.Background(), `SELECT COALESCE((SELECT status FROM jobs WHERE org_id=$1 AND document_id=$2 AND kind=$3), '')`, org, doc, kind).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func readSetting(t *testing.T, org, doc string) string {
	t.Helper()
	var s string
	if err := rawPool(t).QueryRow(context.Background(), `SELECT profile_read FROM documents WHERE org_id=$1 AND id=$2`, org, doc).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestProfileReadQueuesOnlyDocumentsTheProfileReads(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	textReady(t, st, orgA, docA) // docA is a drawing
	textReady(t, st, orgA, specA)

	n, err := st.RequestProfileRead(ctx, orgA, projectA, readKinds)
	if err != nil || n != 1 {
		t.Fatalf("queued %d, %v; want the specification only", n, err)
	}
	if jobStatus(t, orgA, docA, store.JobKindLabel) != "" || jobStatus(t, orgA, specA, store.JobKindLabel) != "queued" {
		t.Fatal("drawing queued or specification not queued")
	}
	view, err := st.ReadProfile(ctx, orgA, projectA, readKinds)
	if err != nil || view.ReadDocuments != 1 || view.SkippedDocuments != 1 || view.SkippedKind != "drawing" || view.UnreadDocuments != 0 {
		t.Fatalf("counts %+v %v", view, err)
	}

	if _, err := st.SetProfileReading(ctx, orgA, projectA, []string{docA}, "read", readKinds); err != nil {
		t.Fatal(err)
	}
	if view, _ := st.ReadProfile(ctx, orgA, projectA, readKinds); view.ReadDocuments != 2 || view.SkippedDocuments != 0 {
		t.Fatalf("a drawing the user chose to read is not counted as read: %+v", view)
	}
	if n, err := st.RequestProfileRead(ctx, orgA, projectA, readKinds); err != nil || n != 1 {
		t.Fatalf("queued %d, %v; want the chosen drawing", n, err)
	}
}

func TestSkippingWithdrawsReadingNotYetStartedAndKeepsFinishedReading(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, specA, 0x51, "specification")
	for _, doc := range []string{specA, docA} {
		if err := st.EnqueueStage(ctx, orgA, doc, store.JobKindLabel); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE jobs SET status='done' WHERE org_id=$1 AND document_id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	n, err := st.SetProfileReading(ctx, orgA, projectA, []string{specA, docA}, "skip", readKinds)
	if err != nil || n != 2 {
		t.Fatalf("changed %d, %v", n, err)
	}
	if jobStatus(t, orgA, specA, store.JobKindLabel) != "" {
		t.Fatal("queued reading of a skipped document was kept")
	}
	if jobStatus(t, orgA, docA, store.JobKindLabel) != "done" {
		t.Fatal("finished reading was discarded; turning the document back on would cost Jev calls")
	}
}

func TestReadingSettingIsAllOrNothingAndOrgScoped(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	if _, err := st.SetProfileReading(ctx, orgA, projectA, []string{docA}, "maybe", readKinds); !errors.Is(err, store.ErrBadReadSetting) {
		t.Fatalf("bad setting: %v", err)
	}
	if _, err := st.SetProfileReading(ctx, orgB, projectA, []string{docA}, "skip", readKinds); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's project: %v", err)
	}
	if _, err := st.SetProfileReading(ctx, orgA, projectA, []string{docA, docB}, "skip", readKinds); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("mixed ids: %v", err)
	}
	if readSetting(t, orgA, docA) != "auto" || readSetting(t, orgB, docB) != "auto" {
		t.Fatal("a rejected request changed a setting")
	}
	if _, err := st.SetProfileReading(ctx, orgA, projectA, nil, "skip", readKinds); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("no ids: %v", err)
	}
}

func TestReadingSettingOnADrawingSetAppliesToItsSheets(t *testing.T) {
	ctx := context.Background()
	st := profileStore(t)
	addDocument(t, st, orgA, projectA, sheetA, 0x52, "drawing")
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO drawing_expansions (org_id, source_id, page_count, status) VALUES ($1, $2, 2, 'complete')`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO drawing_sheets (org_id, source_id, page_number, document_id) VALUES ($1, $2, 1, $3)`, orgA, docA, sheetA); err != nil {
		t.Fatal(err)
	}
	if n, err := st.SetProfileReading(ctx, orgA, projectA, []string{docA}, "read", readKinds); err != nil || n != 2 {
		t.Fatalf("changed %d, %v; want the set and its sheet", n, err)
	}
	if readSetting(t, orgA, sheetA) != "read" {
		t.Fatal("sheet did not follow its set")
	}
	views, err := st.ProjectDocumentViews(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.ProfileRead != "read" {
			t.Fatalf("view %s reports %q", v.ID, v.ProfileRead)
		}
	}
}
