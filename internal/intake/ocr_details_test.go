package intake_test

import (
	"context"
	"errors"
	"net/http"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
	"testing"
	"time"
)

func TestReprocessOCRDetailsPreservesFilingAndCorrections(t *testing.T) {
	for _, race := range []bool{false, true} {
		name := "recover"
		if race {
			name = "edit_during_processing"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			var duringCall func()
			st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if duringCall != nil {
					duringCall()
				}
				_, _ = w.Write([]byte(`{"answers":{}}`))
			}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, time.Second)
			project := testID(194)
			seedProject(t, st, runOrg, project)
			id := seedDoc(t, st, runOrg, project, "1115 CC-06 LEVEL 02 F.pdf", "", "", store.StatusPending, "recovery", true)
			_, err := st.CommitFiling(ctx, runOrg, id, store.CommitFiling{OCR: true, Decisions: []store.DecisionWrite{
				{Field: "revision", Value: "F", Band: "amber", DecidedBy: "rule"},
				{Field: "kind", Value: "drawing", Band: "amber", DecidedBy: "jev"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.Correct(ctx, runOrg, id, "date", ""); err != nil {
				t.Fatal(err)
			} // intentional blank
			if err := st.ReprocessOCRDetails(ctx, testID(193), id); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("wrong org: %v", err)
			}
			if err := st.ReprocessOCRDetails(ctx, runOrg, id); err != nil {
				t.Fatal(err)
			}
			lease, err := st.ClaimJob(ctx, runOrg, time.Minute, []string{store.JobKindOCR})
			if err != nil {
				t.Fatal(err)
			}
			if err := st.ReprocessOCRDetails(ctx, runOrg, id); err != nil {
				t.Fatal(err)
			}
			if _, err := st.ClaimJob(ctx, runOrg, time.Minute, []string{store.JobKindOCR}); !errors.Is(err, store.ErrIdle) {
				t.Fatalf("double click reset lease: %v", err)
			}
			if err := st.ExpireLease(ctx, runOrg, lease.ID); err != nil {
				t.Fatal(err)
			}
			// Exercise recovery after a worker restart, with a current user edit made
			// after the service has taken its initial decision-version snapshot.
			if race {
				duringCall = func() {
					if err := svc.Correct(ctx, runOrg, id, "title", "Owner's title"); err != nil {
						t.Error(err)
					}
				}
			}
			blobs := openBlobs(t, t.TempDir(), 1<<20)
			runner := intake.NewRunner(blobs, st, svc)
			calls := 0
			runner.OCR = func(context.Context, string) (identity.Text, error) {
				calls++
				return ocrTitleBlock("06", "2", "Z"), nil
			}
			worker := &jobs.Worker{Store: st, OCR: runner.RunOCR, Kinds: []string{store.JobKindOCR}}
			if err := worker.Once(ctx, runOrg); err != nil {
				t.Fatal(err)
			}
			view, err := st.DocumentView(ctx, runOrg, id)
			if err != nil {
				t.Fatal(err)
			}
			if view.Status != "filed" || view.Reason != "ocr_details_review" || view.Number != "CC-06" || view.Revision != "F" {
				t.Fatalf("outcome: %+v", view)
			}
			for _, f := range view.Fields {
				if f.Field == "title" {
					want := "LEVEL 2"
					if race {
						want = "Owner's title"
					}
					if f.Value != want {
						t.Errorf("title=%q want %q", f.Value, want)
					}
				}
				if f.Field == "date" && (f.Value != "" || f.DecidedBy != "user") {
					t.Error("overwrote intentional blank")
				}
				if f.Field == "kind" && (f.Value != "drawing" || f.DecidedBy != "jev") {
					t.Error("changed saved classification")
				}
				if f.Field == "number" && f.Band != "amber" {
					t.Error("OCR number not marked for review")
				}
			}
			if calls != 1 || hits.Load() != 1 {
				t.Fatalf("passes=%d Jev calls=%d", calls, hits.Load())
			}
			if _, err := st.ClaimJob(ctx, runOrg, time.Minute, []string{store.JobKindOCR}); !errors.Is(err, store.ErrIdle) {
				t.Fatal("automatic repeat")
			}
			// A later pass with no new evidence must end clearly, preserving values.
			duringCall = nil
			if err := st.ReprocessOCRDetails(ctx, runOrg, id); err != nil {
				t.Fatal(err)
			}
			if err := worker.Once(ctx, runOrg); err != nil {
				t.Fatal(err)
			}
			view, _ = st.DocumentView(ctx, runOrg, id)
			if view.Reason != "ocr_details_unchanged" {
				t.Fatalf("no-change outcome: %+v", view)
			}
			runner.OCR = func(context.Context, string) (identity.Text, error) {
				return identity.Text{}, errors.New("missing file")
			}
			if err := st.ReprocessOCRDetails(ctx, runOrg, id); err != nil {
				t.Fatal(err)
			}
			if err := worker.Once(ctx, runOrg); err != nil {
				t.Fatal(err)
			}
			view, _ = st.DocumentView(ctx, runOrg, id)
			if view.Status != "filed" || view.Reason != "ocr_details_failed" || view.Number != "CC-06" {
				t.Fatalf("failure damaged filing: %+v", view)
			}
		})
	}
}
