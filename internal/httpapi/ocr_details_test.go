package httpapi_test

import (
	"context"
	"net/http"
	"sitewise/internal/store"
	"sort"
	"testing"
	"time"
)

func TestReprocessDetailsEndpointAndLatencyBudget(t *testing.T) {
	a := newApp(t)
	m := a.member(t, "OCR recovery")
	project := m.createProject(t, "Recovery")
	var durations []time.Duration
	for i := 0; i < 25; i++ {
		id, file := newUUID(t), newUUID(t)
		hash := make([]byte, 32)
		hash[0] = byte(i + 1)
		if err := a.store.CreateFile(context.Background(), m.orgID, store.File{ID: file, ProjectID: project, SHA256: hash, ByteSize: 1, MediaType: "application/pdf"}); err != nil {
			t.Fatal(err)
		}
		if err := a.store.CreateDocument(context.Background(), m.orgID, store.Document{ID: id, ProjectID: project, FileID: file, Filename: "scan.pdf", Status: "pending"}); err != nil {
			t.Fatal(err)
		}
		if err := a.store.EnqueueJob(context.Background(), m.orgID, newUUID(t), id, store.JobKindIntake); err != nil {
			t.Fatal(err)
		}
		if _, err := a.store.CommitFiling(context.Background(), m.orgID, id, store.CommitFiling{OCR: true}); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if got := m.status(t, http.MethodPost, "/api/documents/"+id+"/details/reprocess", nil); got != http.StatusAccepted {
			t.Fatalf("queue=%d", got)
		}
		durations = append(durations, time.Since(start))
		var doc document
		m.getJSON(t, "/api/documents/"+id, &doc)
		if doc.Status != "filed" || doc.Reason != "ocr_details_queued" {
			t.Fatalf("queue response: %+v", doc)
		}
		if got := m.status(t, http.MethodPost, "/api/documents/"+id+"/details/reprocess", nil); got != http.StatusAccepted {
			t.Fatalf("repeated click=%d", got)
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	t.Logf("reprocess queue p50=%s p90=%s; budget 50ms/150ms", durations[12], durations[22])
	if durations[12] > 50*time.Millisecond || durations[22] > 150*time.Millisecond {
		t.Fatal("reprocess queue budget exceeded")
	}
	if got := m.status(t, http.MethodPost, "/api/documents/not-a-uuid/details/reprocess", nil); got != http.StatusNotFound {
		t.Fatalf("malformed id=%d", got)
	}
}
