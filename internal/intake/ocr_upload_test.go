package intake_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
	"testing"
	"time"
)

func TestUploadOCRIsConditionalDurableAndReviewOnly(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", 503) }), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
	ctx := context.Background()
	org, project := runOrg, testID(92)
	seedProject(t, st, org, project)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	calls := 0
	runner.OCR = func(context.Context, string) (identity.Text, error) {
		calls++
		return ruledText("A-101", "B", "Ground floor"), nil
	}
	readable := uploadFixture(t, up, project, "identity-page.pdf")
	if err := runner.Run(ctx, org, readable.DocumentID); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("readable PDF invoked OCR")
	}
	scanned := uploadFixture(t, up, project, "scanned-empty.pdf")
	if err := runner.Run(ctx, org, scanned.DocumentID); err != nil {
		t.Fatal(err)
	}
	view, _ := st.DocumentView(ctx, org, scanned.DocumentID)
	if calls != 0 || view.Status != store.StatusPending || view.Reason != "ocr_queued" {
		t.Fatalf("not queued: %+v calls=%d", view, calls)
	}
	var delivered []string
	runner.Wake = func(orgID string) {
		if orgID != org {
			t.Fatalf("notified wrong org: %s", orgID)
		}
		committed, err := st.DocumentView(ctx, orgID, scanned.DocumentID)
		if err != nil {
			t.Fatal(err)
		}
		delivered = append(delivered, committed.Reason)
	}
	if err := runner.RunOCR(ctx, testID(93), scanned.DocumentID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	if calls != 0 {
		t.Fatal("wrong org read blob")
	}
	if err := svc.Correct(ctx, org, scanned.DocumentID, "title", "Owner's title"); err != nil {
		t.Fatal(err)
	}
	lease, err := st.ClaimJob(ctx, org, time.Second, []string{store.JobKindOCR})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ExpireLease(ctx, org, lease.ID); err != nil {
		t.Fatal(err)
	}
	worker := &jobs.Worker{Store: st, OCR: runner.RunOCR, Kinds: []string{store.JobKindOCR}}
	if err := worker.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	if err := runner.RunOCR(ctx, org, scanned.DocumentID); err != nil {
		t.Fatal(err)
	}
	view, _ = st.DocumentView(ctx, org, scanned.DocumentID)
	if calls != 1 || view.Status != store.StatusFiled {
		t.Fatalf("outcome %+v calls=%d", view, calls)
	}
	if !reflect.DeepEqual(delivered, []string{"ocr_reading", "ocr_classifying", "ocr_review"}) {
		t.Fatalf("live OCR stages must be delivered after commit: %v", delivered)
	}
	for _, f := range view.Fields {
		if f.DecidedBy != "user" && f.Band == "green" {
			t.Fatalf("OCR marked confirmed: %+v", f)
		}
		if f.Field == "title" && f.Value != "Owner's title" {
			t.Fatal("overwrote correction")
		}
	}
}

func TestOCRFailureAndExplicitRetry(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		err          error
	}{
		{"empty", "ocr_no_text", nil}, {"unavailable", "ocr_unavailable", identity.ErrOCRUnavailable},
		{"limit", "ocr_limit", identity.ErrTooLarge}, {"missing", "ocr_failed", errors.New("missing file")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unexpected Jev", 500) }), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
			ctx := context.Background()
			project := testID(94)
			seedProject(t, st, runOrg, project)
			blobs := openBlobs(t, t.TempDir(), 1<<20)
			up := intake.NewUploader(blobs, st)
			runner := intake.NewRunner(blobs, st, svc)
			calls := 0
			runner.OCR = func(context.Context, string) (identity.Text, error) { calls++; return identity.Text{}, tc.err }
			doc := uploadFixture(t, up, project, "scanned-empty.pdf")
			if err := runner.Run(ctx, runOrg, doc.DocumentID); err != nil {
				t.Fatal(err)
			}
			var delivered []string
			runner.Wake = func(orgID string) {
				if orgID != runOrg {
					t.Fatalf("notified wrong org: %s", orgID)
				}
				view, err := st.DocumentView(ctx, orgID, doc.DocumentID)
				if err != nil {
					t.Fatal(err)
				}
				delivered = append(delivered, view.Reason)
			}
			worker := &jobs.Worker{Store: st, OCR: runner.RunOCR, Kinds: []string{store.JobKindOCR}}
			if err := worker.Once(ctx, runOrg); err != nil {
				t.Fatal(err)
			}
			view, _ := st.DocumentView(ctx, runOrg, doc.DocumentID)
			if view.Status != store.StatusNotFiled || view.Reason != tc.reason || calls != 1 || hits.Load() != 0 {
				t.Fatalf("outcome %+v calls=%d", view, calls)
			}
			if !reflect.DeepEqual(delivered, []string{"ocr_reading", tc.reason}) {
				t.Fatalf("live failure must be delivered after commit: %v", delivered)
			}
			if err := worker.Once(ctx, runOrg); !errors.Is(err, store.ErrIdle) {
				t.Fatalf("automatic repeat: %v", err)
			}
			if err := st.RetryOCR(ctx, testID(93), doc.DocumentID); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("cross-org retry: %v", err)
			}
			if err := st.RetryOCR(ctx, runOrg, doc.DocumentID); err != nil {
				t.Fatal(err)
			}
			if err := worker.Once(ctx, runOrg); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("explicit retry calls=%d", calls)
			}
		})
	}
}
