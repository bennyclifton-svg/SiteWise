package httpapi_test

import (
	"encoding/json"
	"net/http"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
)

func TestRFPTransmittalIsolationAndSnapshot(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Transmittal")
	doc := m.upload(t, project, "identity-page.pdf", http.StatusCreated)
	m.correct(t, doc.ID, "title", "Survey issued for pricing", http.StatusOK)
	foreign := a.member(t, newUUID(t))
	foreignProject := foreign.createProject(t, "Foreign")
	foreignDoc := foreign.upload(t, foreignProject, "identity-page.pdf", http.StatusCreated)
	otherProject := m.createProject(t, "Other project")
	otherDoc := m.upload(t, otherProject, "identity-page.pdf", http.StatusCreated)
	var p procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect","discipline_id":"consultant.architect"}`), http.StatusCreated, &p)
	var r store.Report
	raw, _ := json.Marshal(map[string]string{"kind": "rfp", "package_id": p.ID})
	m.call(t, http.MethodPost, "/api/projects/"+project+"/reports", raw, http.StatusCreated, &r)
	base := "/api/reports/" + r.ID
	var draft store.ReportDraft
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &draft)
	var view store.ReportView
	put := func(ids []string, want int) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"value": ids, "version": draft.Version})
		m.call(t, http.MethodPut, base+"/settings/transmittal", body, want, &view)
		if want == http.StatusOK {
			draft = *view.Draft
		}
	}
	put([]string{foreignDoc.ID}, http.StatusBadRequest)
	put([]string{otherDoc.ID}, http.StatusBadRequest)
	put([]string{newUUID(t)}, http.StatusBadRequest)
	put([]string{doc.ID, doc.ID}, http.StatusBadRequest)
	put([]string{doc.ID}, http.StatusOK)
	check := func(want int, title string) {
		t.Helper()
		for _, section := range draft.Sections {
			for _, block := range section.Blocks {
				if block.ID == "rfp:project:information" {
					if block.Table == nil || len(block.Table.Rows) != want {
						t.Fatalf("schedule: %+v", block.Table)
					}
					if want > 0 && block.Table.Rows[0][1] != title {
						t.Fatalf("title: %v", block.Table.Rows)
					}
					return
				}
			}
		}
		t.Fatal("missing transmittal")
	}
	check(1, "Survey issued for pricing")
	m.correct(t, doc.ID, "title", "Later title", http.StatusOK)
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &draft)
	check(1, "Survey issued for pricing")
	put([]string{doc.ID}, http.StatusOK)
	check(1, "Later title")
	put([]string{}, http.StatusOK)
	check(0, "")
	m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &draft)
	check(0, "")
	raw, _ = json.Marshal(map[string]any{"value": []string{}, "version": draft.Version})
	foreign.call(t, http.MethodPut, base+"/settings/transmittal", raw, http.StatusNotFound, nil)
}
