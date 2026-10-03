package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"
)

func TestDeleteDocuments(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Deletion")
	keep := m.upload(t, project, "identity-page.pdf", http.StatusCreated)
	gone := m.upload(t, project, "docx-table.docx", http.StatusCreated)
	path := "/api/projects/" + project + "/documents/delete"
	body := func(ids ...string) []byte {
		raw, _ := json.Marshal(map[string]any{"document_ids": ids})
		return raw
	}

	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodPost, path, body(gone.ID)); code != http.StatusNotFound {
		t.Fatalf("cross-org delete = %d", code)
	}
	theirs := other.createProject(t, "Theirs")
	if code := m.status(t, http.MethodPost, "/api/projects/"+theirs+"/documents/delete", body(gone.ID)); code != http.StatusNotFound {
		t.Fatalf("delete through another org's project = %d", code)
	}
	m.call(t, http.MethodPost, path, body(), http.StatusBadRequest, nil)
	m.call(t, http.MethodPost, path, body("not-a-uuid"), http.StatusNotFound, nil)
	m.call(t, http.MethodPost, path, body(gone.ID, newUUID(t)), http.StatusNotFound, nil)
	if got := m.list(t, project); len(got.Documents) != 2 {
		t.Fatalf("a rejected delete removed documents: %d left", len(got.Documents))
	}

	var out struct {
		Deleted []string `json:"deleted"`
	}
	m.call(t, http.MethodPost, path, body(gone.ID), http.StatusOK, &out)
	if len(out.Deleted) != 1 || out.Deleted[0] != gone.ID {
		t.Fatalf("deleted %+v", out)
	}
	got := m.list(t, project)
	if len(got.Documents) != 1 || got.Documents[0].ID != keep.ID {
		t.Fatalf("after delete %+v", got.Documents)
	}
	if code := m.status(t, http.MethodGet, "/api/documents/"+gone.ID, nil); code != http.StatusNotFound {
		t.Fatalf("deleted document still readable: %d", code)
	}
	if code := m.status(t, http.MethodGet, "/api/documents/"+gone.ID+"/file", nil); code != http.StatusNotFound {
		t.Fatalf("deleted file still downloadable: %d", code)
	}
	m.call(t, http.MethodPost, path, body(gone.ID), http.StatusNotFound, nil)
}

func TestDeleteLatencyBudget(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Deletion timing")
	var ids []string
	for i := 0; i < 20; i++ {
		doc := m.uploadBytes(t, project, fmt.Sprintf("note-%d.pdf", i), []byte(fmt.Sprintf("not a pdf %d", i)), http.StatusCreated)
		ids = append(ids, doc.ID)
	}
	var durations []time.Duration
	for _, id := range ids {
		raw, _ := json.Marshal(map[string]any{"document_ids": []string{id}})
		start := time.Now()
		m.call(t, http.MethodPost, "/api/projects/"+project+"/documents/delete", raw, http.StatusOK, nil)
		durations = append(durations, time.Since(start))
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50, p90 := durations[9], durations[17]
	t.Logf("document_delete p50=%s p90=%s (budget 500ms/1000ms)", p50, p90)
	if p50 > 500*time.Millisecond || p90 > time.Second {
		t.Fatal("document_delete exceeded budget")
	}
}
