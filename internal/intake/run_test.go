package intake_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/intake"
	"sitewise/internal/store"
)

const runOrg = "44444444-4444-4444-8444-444444444441"

func TestRunFilesStoredUploads(t *testing.T) {
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	ctx := context.Background()
	project := testID(40)
	seedProject(t, st, runOrg, project)

	cases := []struct {
		fixture string
		status  string
		reason  string
	}{
		{"identity-page.pdf", store.StatusFiled, ""},
		{"docx-table.docx", store.StatusFiled, ""},
		{"merged-cells.xlsx", store.StatusFiled, ""},
		{"scanned-empty.pdf", store.StatusNotFiled, intake.ReasonNoText},
		{"malformed.docx", store.StatusNotFiled, intake.ReasonUnreadable},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			filing := uploadFixture(t, up, project, tc.fixture)
			if err := runner.Run(ctx, runOrg, filing.DocumentID); err != nil {
				t.Fatal(err)
			}
			view, err := st.DocumentView(ctx, runOrg, filing.DocumentID)
			if err != nil {
				t.Fatal(err)
			}
			if view.Status != tc.status || view.Reason != tc.reason {
				t.Fatalf("status %s reason %q", view.Status, view.Reason)
			}
			if tc.status == store.StatusFiled && len(view.Fields) == 0 {
				t.Fatal("filed without fields")
			}
			// A second run is safe: the filing is already committed.
			if err := runner.Run(ctx, runOrg, filing.DocumentID); err != nil {
				t.Fatal(err)
			}
		})
	}
	if hits.Load() > 3 {
		t.Fatalf("jev calls %d for three readable files", hits.Load())
	}
	pending, err := st.PendingIntake(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pending {
		if p.OrgID == runOrg {
			t.Fatalf("left pending %+v", p)
		}
	}
}

func TestUnsupportedFormatIsStoredNotFiled(t *testing.T) {
	st := storeFrom(t)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	project := testID(41)
	seedProject(t, st, runOrg, project)
	filing, err := up.Upload(context.Background(), runOrg, project, intake.Upload{
		Filename: "notes.txt",
		Body:     bytes.NewReader([]byte("plain text")),
		Reason:   intake.ReasonFor("notes.txt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filing.Status != store.StatusNotFiled || filing.Reason != intake.ReasonUnsupported {
		t.Fatalf("%+v", filing)
	}
	if intake.ReasonFor("A-100 Plan.PDF") != "" || intake.ReasonFor("sheet.xlsx") != "" || intake.ReasonFor("spec.docx") != "" {
		t.Fatal("supported format given a reason")
	}
}

func uploadFixture(t *testing.T, up *intake.Uploader, project, name string) intake.Filing {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "identity", name))
	if err != nil {
		t.Fatal(err)
	}
	filing, err := up.Upload(context.Background(), runOrg, project, intake.Upload{
		Filename: name,
		Body:     bytes.NewReader(body),
		Reason:   intake.ReasonFor(name),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filing.Status != store.StatusPending {
		t.Fatalf("upload status %s", filing.Status)
	}
	return filing
}
