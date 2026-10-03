package intake_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

// Run through tools/check-ocr.ps1. Real rasterisation/OCR, database queue and
// filing; only provider latency is replayed (350 ms), as in intake-bench.
func TestOCRFilingSpeedGate(t *testing.T) {
	if os.Getenv("SITEWISE_OCR_GATE") != "1" {
		t.Skip("run tools/check-ocr.ps1 for installed OCR gate")
	}
	path := os.Getenv("SITEWISE_OCR_TEST_PDF")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := identity.NewOCR(os.Getenv("SITEWISE_TESSERACT"), os.Getenv("SITEWISE_TESSDATA"), os.Getenv("SITEWISE_OCR_PYTHON"))
	if err != nil {
		t.Fatal(err)
	}
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(350 * time.Millisecond)
		_, _ = w.Write([]byte(`{"answers":{}}`))
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, time.Second)
	ctx := context.Background()
	org, project := testID(140), testID(141)
	seedProject(t, st, org, project)
	blobs := openBlobs(t, t.TempDir(), 32<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	runner.OCR = reader.Extract
	worker := &jobs.Worker{Store: st, OCR: runner.RunOCR, Kinds: []string{store.JobKindOCR}}
	if err := identity.Warm(ctx); err != nil {
		t.Fatal(err)
	}
	var timings []float64
	var recoveryTimings []float64
	for i := 0; i < 20; i++ {
		p := testID(150 + i)
		if err := st.CreateProject(ctx, org, p, "OCR benchmark"); err != nil {
			t.Fatal(err)
		}
		doc, err := up.Upload(ctx, org, p, intake.Upload{Filename: "scan.pdf", Body: bytes.NewReader(body)})
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if err := runner.Run(ctx, org, doc.DocumentID); err != nil {
			t.Fatal(err)
		}
		queued, _ := st.DocumentView(ctx, org, doc.DocumentID)
		if queued.Reason != "ocr_queued" {
			t.Fatalf("fixture did not need OCR: %+v", queued)
		}
		if err := worker.Once(ctx, org); err != nil {
			t.Fatal(err)
		}
		view, _ := st.DocumentView(ctx, org, doc.DocumentID)
		if view.Status != store.StatusFiled || view.Reason != "ocr_review" {
			t.Fatalf("OCR failed: %+v", view)
		}
		// Include the worker's worst idle-poll delay in the bounded single-file
		// model. Burst queue waiting is observable separately, not hidden here.
		timings = append(timings, float64(time.Since(start).Microseconds())/1000+2000)
		start = time.Now()
		if err := st.ReprocessOCRDetails(ctx, org, doc.DocumentID); err != nil {
			t.Fatal(err)
		}
		if err := worker.Once(ctx, org); err != nil {
			t.Fatal(err)
		}
		view, _ = st.DocumentView(ctx, org, doc.DocumentID)
		if view.Status != store.StatusFiled || view.Reason != "ocr_details_unchanged" {
			t.Fatalf("recovery outcome: %+v", view)
		}
		recoveryTimings = append(recoveryTimings, float64(time.Since(start).Microseconds())/1000+2000)
	}
	sort.Float64s(timings)
	p50, p90 := timings[9], timings[17]
	t.Logf("20 OCR filings, including 2000ms polling allowance and replayed 350ms provider: p50=%.0fms p90=%.0fms; budget 10000/20000ms", p50, p90)
	sort.Float64s(recoveryTimings)
	t.Logf("20 detail recoveries, including 2000ms polling allowance and replayed 350ms provider: p50=%.0fms p90=%.0fms; budget 10000/20000ms", recoveryTimings[9], recoveryTimings[17])
	if recoveryTimings[9] > 10000 || recoveryTimings[17] > 20000 {
		t.Fatal("detail recovery speed budget exceeded")
	}
	if hits.Load() != 40 {
		t.Fatalf("expected one fan-out per OCR filing, got %d", hits.Load())
	}
	if output := os.Getenv("SITEWISE_OCR_TIMING_OUTPUT"); output != "" {
		b, _ := json.MarshalIndent(map[string]any{"samples": 20, "p50_ms": p50, "p90_ms": p90, "timings_ms": timings, "recovery_p50_ms": recoveryTimings[9], "recovery_p90_ms": recoveryTimings[17], "poll_allowance_ms": 2000, "provider_replay_ms": 350}, "", "  ")
		if err := os.WriteFile(output, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if p50 > 10000 || p90 > 20000 {
		t.Fatal("OCR filing speed budget exceeded")
	}
}
