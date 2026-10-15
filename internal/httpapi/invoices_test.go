package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"sitewise/internal/invoices/sample"
	"sitewise/internal/procurement"
)

type invoiceLedger struct {
	Summary struct {
		Lines []struct {
			ID       string  `json:"id"`
			Invoiced *string `json:"invoiced"`
		} `json:"lines"`
		Unallocated struct {
			Invoiced *string `json:"invoiced"`
		} `json:"unallocated"`
		Invoiced *string `json:"invoiced"`
	} `json:"summary"`
	Invoices []struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		Number string  `json:"number"`
		Ex     *string `json:"ex_gst"`
		Flags  []struct {
			Code string `json:"code"`
		} `json:"flags"`
		Rows []struct {
			LineID string `json:"line_id"`
		} `json:"rows"`
	} `json:"invoices"`
}

func (m *member) waitInvoices(t *testing.T, project string, n int) invoiceLedger {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var v invoiceLedger
		m.getJSON(t, "/api/projects/"+project+"/invoices", &v)
		read := 0
		for _, inv := range v.Invoices {
			if inv.Status != "reading" {
				read++
			}
		}
		if read >= n {
			return v
		}
		if time.Now().After(deadline) {
			t.Fatalf("invoices not read: %+v", v.Invoices)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestInvoiceUploadReadEditExport(t *testing.T) {
	a := newApp(t, withProfile(t))
	m, other := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Invoice project")
	var pkg procurement.Package
	m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", []byte(`{"kind":"services","title":"Architect"}`), http.StatusCreated, &pkg)
	var ready struct {
		Plan struct {
			Items []struct {
				ID        string `json:"cost_item_id"`
				PackageID string `json:"package_id"`
			} `json:"items"`
		} `json:"plan"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/cost-plan/prepare", []byte(`{}`), http.StatusOK, &ready)
	line := ""
	for _, it := range ready.Plan.Items {
		if it.PackageID == pkg.ID {
			line = it.ID
		}
	}
	inv := sample.Batch(1, 1)[0]
	body := sample.PDF(inv)
	path := "/api/projects/" + project + "/invoices?name=" + url.QueryEscape(inv.Number+".pdf")
	var created struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Created bool   `json:"created"`
	}
	m.call(t, http.MethodPost, path, body, http.StatusCreated, &created)
	if !created.Created || created.Status != "reading" {
		t.Fatalf("%+v", created)
	}
	v := m.waitInvoices(t, project, 1)
	got := v.Invoices[0]
	if got.Number != inv.Number || got.Ex == nil || *got.Ex != "1123.45" || got.Status != "needs_review" {
		t.Fatalf("read %+v", got)
	}
	var again struct {
		ID      string `json:"id"`
		Created bool   `json:"created"`
	}
	m.call(t, http.MethodPost, path, body, http.StatusOK, &again)
	if again.Created || again.ID != created.ID {
		t.Fatalf("re-drop %+v", again)
	}
	edit, _ := json.Marshal(map[string]any{"fields": map[string]string{"supplier": pkg.ID, "line": line}})
	m.call(t, http.MethodPatch, "/api/projects/"+project+"/invoices/"+created.ID, edit, http.StatusOK, &v)
	if v.Summary.Invoiced == nil || *v.Summary.Invoiced != "1123.45" || v.Invoices[0].Rows[0].LineID != line {
		t.Fatalf("after edit %+v", v.Summary)
	}
	bad, _ := json.Marshal(map[string]any{"fields": map[string]string{"total": "12.345"}})
	if got := m.status(t, http.MethodPatch, "/api/projects/"+project+"/invoices/"+created.ID, bad); got != http.StatusUnprocessableEntity {
		t.Fatalf("bad amount = %d", got)
	}

	resp := m.do(t, http.MethodGet, "/api/projects/"+project+"/invoices/export", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "spreadsheetml") || len(raw) < 500 {
		t.Fatalf("xlsx %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	resp = m.do(t, http.MethodGet, "/api/projects/"+project+"/invoices/export?format=csv&sheet=invoices&line="+line, nil)
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(raw), inv.Number) || !strings.Contains(string(raw), "1123.45") {
		t.Fatalf("csv %d %s", resp.StatusCode, raw)
	}
	resp = m.do(t, http.MethodGet, "/api/projects/"+project+"/invoices/"+created.ID+"/file?view=1", nil)
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/pdf" || !strings.Contains(resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'self'") || string(raw) != string(body) {
		t.Fatalf("file %d %v", resp.StatusCode, resp.Header)
	}

	// An unreadable format is stored and listed as not read.
	var txt struct {
		Status string `json:"status"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/invoices?name=notes.txt", []byte("hello"), http.StatusCreated, &txt)
	if txt.Status != "not_read" {
		t.Fatalf("txt %+v", txt)
	}

	// Another org sees none of it and changes none of it.
	for _, c := range []struct {
		method, path string
		body         []byte
	}{
		{http.MethodGet, "/api/projects/" + project + "/invoices", nil},
		{http.MethodGet, "/api/projects/" + project + "/invoices/export", nil},
		{http.MethodPatch, "/api/projects/" + project + "/invoices/" + created.ID, edit},
		{http.MethodPut, "/api/projects/" + project + "/invoices/" + created.ID + "/allocations", []byte(`{"shares":[{"line_id":""}]}`)},
		{http.MethodGet, "/api/projects/" + project + "/invoices/" + created.ID + "/file", nil},
		{http.MethodPost, "/api/projects/" + project + "/invoices?name=x.pdf", body},
	} {
		if got := other.status(t, c.method, c.path, c.body); got != http.StatusNotFound {
			t.Fatalf("%s %s from another org = %d", c.method, c.path, got)
		}
	}
	otherProject := other.createProject(t, "Theirs")
	if got := m.status(t, http.MethodPatch, "/api/projects/"+otherProject+"/invoices/"+created.ID, edit); got != http.StatusNotFound {
		t.Fatalf("own invoice through another org's project = %d", got)
	}
}
