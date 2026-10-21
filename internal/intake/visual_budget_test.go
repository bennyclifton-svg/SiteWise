package intake_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

type visualLatencyAsker struct{}

func (visualLatencyAsker) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	select {
	case <-ctx.Done():
		return jev.Result{}, ctx.Err()
	case <-time.After(600 * time.Millisecond):
		return jev.Result{}, nil
	}
}

func TestVisualIdentitySpeedGate(t *testing.T) {
	if os.Getenv("SITEWISE_VISUAL_GATE") != "1" {
		t.Skip("run tools/check-visual.ps1")
	}
	body, err := os.ReadFile(os.Getenv("SITEWISE_OCR_TEST_PDF"))
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := identity.NewOCR(os.Getenv("SITEWISE_TESSERACT"), os.Getenv("SITEWISE_TESSDATA"), os.Getenv("SITEWISE_OCR_PYTHON"))
	if err != nil {
		t.Fatal(err)
	}
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(350 * time.Millisecond)
		w.Write([]byte(`{"model":"jev-1.13.0","answers":{}}`))
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, time.Second)
	ctx := context.Background()
	org, project := testID(220), testID(221)
	seedProject(t, st, org, project)
	blobs := openBlobs(t, t.TempDir(), 32<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	reader := &intake.VisualReader{Store: st, Blobs: blobs, Ask: visualLatencyAsker{}, Render: renderer.Visual}
	worker := jobs.Worker{Store: st, Visual: reader.Run, Kinds: []string{store.JobKindVisualIdentity}}
	var filing, visual []time.Duration
	identity.Warm(ctx)
	for i := 0; i < 20; i++ {
		p := testID(230 + i)
		if err := st.CreateProject(ctx, org, p, "Visual gate"); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		doc, err := up.Upload(ctx, org, p, intake.Upload{Filename: "drawing.pdf", Body: bytes.NewReader(body)})
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.Run(ctx, org, doc.DocumentID); err != nil {
			t.Fatal(err)
		}
		filing = append(filing, time.Since(start))
		start = time.Now()
		if err := worker.Once(ctx, org); err != nil {
			t.Fatal(err)
		}
		visual = append(visual, time.Since(start))
		review, err := st.VisualIdentity(ctx, org, doc.DocumentID)
		if err != nil || review.Status != "ready" {
			t.Fatalf("visual failed: %s %v", review.Status, err)
		}
	}
	for name, samples := range map[string][]time.Duration{"filing": filing, "visual": visual} {
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p50, p90 := samples[9], samples[17]
		limit50, limit90 := time.Second, 2*time.Second
		if name == "visual" {
			limit50 = 10 * time.Second
			limit90 = 20 * time.Second
		}
		t.Logf("%s 20 samples p50=%s p90=%s; budgets %s/%s; provider replay 350ms filing / 600ms visual, queue excluded", name, p50, p90, limit50, limit90)
		if p50 > limit50 || p90 > limit90 {
			t.Fatalf("%s speed budget exceeded", name)
		}
	}
}
