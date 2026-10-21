package intake_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"sitewise/internal/intake"
	"sitewise/internal/jobs"
	"sitewise/internal/latency"
	"sitewise/internal/store"
)

func TestMarkdownFilingAndFullSource(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
	ctx := context.Background()
	project := testID(43)
	seedProject(t, st, runOrg, project)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	var samples []int64
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("strategy-%d.MD", i)
		if i%2 == 1 {
			name = fmt.Sprintf("strategy-%d.markdown", i)
		}
		body := fmt.Sprintf("# Fire strategy %d\nDocument number: FS-%03d\n", i, i) + strings.Repeat("Wall requirements.\n", 420) + "END OF SOURCE"
		start := time.Now()
		filed, err := up.Upload(ctx, runOrg, project, intake.Upload{Filename: name, Body: strings.NewReader(body), Reason: intake.ReasonFor(name)})
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.Run(ctx, runOrg, filed.DocumentID); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, time.Since(start).Microseconds())
		doc, err := st.GetDocument(ctx, runOrg, filed.DocumentID)
		if err != nil || doc.Status != store.StatusFiled {
			t.Fatalf("document=%+v err=%v", doc, err)
		}
		file, err := st.GetFile(ctx, runOrg, doc.FileID)
		if err != nil || file.MediaType != "text/markdown; charset=utf-8" {
			t.Fatalf("file=%+v err=%v", file, err)
		}
		source, err := jobs.FullSource(st, blobs)(ctx, runOrg, doc.ID)
		if err != nil || len(source.Source) == 0 || !strings.Contains(source.Source[0].Text, "END OF SOURCE") {
			t.Fatalf("full source missing tail: %v", err)
		}
		duplicate, err := up.Upload(ctx, runOrg, project, intake.Upload{Filename: name, Body: strings.NewReader(body)})
		if err != nil || duplicate.DocumentID != doc.ID {
			t.Fatalf("duplicate=%+v err=%v", duplicate, err)
		}
	}
	p50, _ := latency.Percentile(samples, 0.5)
	p90, _ := latency.Percentile(samples, 0.9)
	t.Logf("Markdown filing (local unavailable-provider fixture): p50 %.2f ms, p90 %.2f ms; budget 1000/2000 ms", float64(p50)/1000, float64(p90)/1000)
	if p50 > 1000000 || p90 > 2000000 {
		t.Fatal("filing budget exceeded")
	}
}
