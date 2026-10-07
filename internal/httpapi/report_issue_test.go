package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sitewise/internal/store"
	"testing"
)

func TestReportIssueDownloadIsolationAndHistory(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	foreign := a.member(t, newUUID(t))
	project := m.createProject(t, "Issue fixture")
	var r store.Report
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", []byte(`{"kind":"pmp"}`), http.StatusCreated, &r)
	base := "/api/reports/" + r.ID
	var d store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	body, _ := json.Marshal(store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Synthetic fixture with missing profile explicitly acknowledged"})
	foreign.call(t, http.MethodPost, base+"/issue", body, http.StatusNotFound, nil)
	var issued store.IssuedReport
	m.call(t, http.MethodPost, base+"/issue", body, http.StatusCreated, &issued)
	path := base + "/versions/" + issued.ID + "/file"
	foreign.call(t, http.MethodGet, path, nil, http.StatusNotFound, nil)
	response := m.do(t, http.MethodGet, path, nil)
	first, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || !bytes.HasPrefix(first, []byte("%PDF-")) || response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatal("invalid issued PDF", response.StatusCode, err)
	}
	m.call(t, http.MethodPost, base+"/issue", body, http.StatusConflict, nil)
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
	if d.ID == issued.ID {
		t.Fatal("issued version reused as draft")
	}
	var view store.ReportView
	m.getJSON(t, base, &view)
	if len(view.Issues) != 1 || view.Issues[0].ID != issued.ID {
		t.Fatal("issue absent from history")
	}
	response = m.do(t, http.MethodGet, path, nil)
	second, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !bytes.Equal(first, second) {
		t.Fatal("refresh changed issued export")
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("issue called Jev")
	}
}
