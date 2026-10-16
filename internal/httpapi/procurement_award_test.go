package httpapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/proposals"
	psample "sitewise/internal/proposals/sample"
)

type priceTableBody struct {
	PackageID string `json:"package_id"`
	Items     []struct {
		Key   string `json:"key"`
		Kind  string `json:"kind"`
		Label string `json:"label"`
		Base  bool   `json:"base"`
	} `json:"items"`
	Columns []struct {
		Submission struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"submission"`
		Firm *struct {
			Name string `json:"name"`
		} `json:"firm"`
		Cells map[string]struct {
			State  string  `json:"state"`
			Amount *string `json:"amount"`
		} `json:"cells"`
		Totals struct {
			BaseTotal *string `json:"base_total"`
		} `json:"totals"`
	} `json:"columns"`
	Award *struct {
		Total string `json:"total"`
	} `json:"award"`
}

func greenProposals(o *httpapi.Options) {
	o.ProposalJev = psample.Jev{}
	lo, hi, g := 2, 255, 0.5
	o.ProposalThresholds = proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	for _, f := range []string{"price", "rate", "criterion", "firm", "kind", "package", "gst_basis", "stated_total"} {
		o.ProposalThresholds.Questions[f] = []proposals.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &g}}
	}
}

func TestProposalDroppedOnPackageReadsComparesAwardsAndStaysPrivate(t *testing.T) {
	a := newApp(t, withProfile(t), greenProposals)
	m, other := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Award project")
	var pkg struct {
		ID string `json:"id"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated, &pkg)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	var table priceTableBody
	m.getJSON(t, base+"/prices", &table)
	var lines []psample.Line
	for _, it := range table.Items {
		if it.Base {
			lines = append(lines, psample.Line{Label: it.Label, Cents: 100000})
		}
	}
	if len(lines) == 0 {
		t.Fatal("no schedule")
	}
	pdf := psample.PDF(psample.Proposal{Firm: "Example Planning Pty Ltd", ABN: "88071762537", Email: "office@example-planning.test", Lines: lines, Rates: []psample.Line{{Label: "Director", Cents: 45000}}})
	var up struct {
		SubmissionID string `json:"submission_id"`
	}
	m.call(t, http.MethodPost, base+"/submissions?name="+url.QueryEscape("Fee proposal (Example Planning).pdf"), pdf, http.StatusCreated, &up)
	deadline := time.Now().Add(15 * time.Second)
	for {
		m.getJSON(t, base+"/prices", &table)
		if len(table.Columns) == 1 && table.Columns[0].Submission.Status != "reading" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("proposal not read: %+v", table.Columns)
		}
		time.Sleep(50 * time.Millisecond)
	}
	col := table.Columns[0]
	want := fmt.Sprintf("%d.00", len(lines)*1000)
	if col.Firm == nil || col.Totals.BaseTotal == nil || *col.Totals.BaseTotal != want {
		t.Fatalf("column %+v, want base total %s", col, want)
	}
	sub := col.Submission.ID
	first := ""
	for _, it := range table.Items {
		if it.Base {
			first = it.Key
			break
		}
	}

	// Every route refuses another org, with the same 404 as a missing record.
	cell := "/api/projects/" + project + "/submissions/" + sub
	for _, r := range []struct{ method, path, body string }{
		{"GET", base + "/prices", ""},
		{"GET", base + "/prices/export", ""},
		{"GET", "/api/projects/" + project + "/procurement/board", ""},
		{"PUT", cell + "/cells/" + url.PathEscape(first), `{"state":"not_priced"}`},
		{"PATCH", cell, `{"status":"withdrawn"}`},
		{"GET", cell + "/file", ""},
		{"POST", base + "/award/preview", `{"submission_id":"` + sub + `","lines":[{"label":"All","item_keys":["` + first + `"]}]}`},
		{"POST", base + "/award", `{"submission_id":"` + sub + `","lines":[{"label":"All","item_keys":["` + first + `"]}]}`},
		{"DELETE", base + "/award", ""},
		{"GET", base + "/report", ""},
		{"PUT", base + "/report", `{"text":"x"}`},
		{"PUT", base + "/submissions/" + sub + "/ratings/criterion:r4", `{"rating":"strong"}`},
		{"POST", base + "/firms", `{"firm":{"name":"Intruder"}}`},
	} {
		var body []byte
		if r.body != "" {
			body = []byte(r.body)
		}
		if got := other.status(t, r.method, r.path, body); got != http.StatusNotFound {
			t.Errorf("%s %s from another org = %d, want 404", r.method, r.path, got)
		}
	}
	var firms struct {
		Firms []json.RawMessage `json:"firms"`
	}
	other.getJSON(t, "/api/firms?q=", &firms)
	if len(firms.Firms) != 0 {
		t.Fatalf("another org sees %d firms", len(firms.Firms))
	}
	m.getJSON(t, "/api/firms?q=example", &firms)
	if len(firms.Firms) != 1 {
		t.Fatalf("own directory %d firms", len(firms.Firms))
	}

	// A user's cell is final and the totals follow.
	m.call(t, http.MethodPut, cell+"/cells/"+url.PathEscape(first), []byte(`{"state":"priced","amount":"2,000"}`), http.StatusOK, &table)
	want = fmt.Sprintf("%d.00", len(lines)*1000+1000)
	if got := table.Columns[0].Totals.BaseTotal; got == nil || *got != want {
		t.Fatalf("after correction %v, want %s", got, want)
	}
	if got := m.status(t, http.MethodPut, cell+"/cells/"+url.PathEscape(first), []byte(`{"state":"priced","amount":"lots"}`)); got != http.StatusUnprocessableEntity {
		t.Fatalf("bad amount = %d", got)
	}

	// Export is a workbook in the screen's layout.
	resp := m.do(t, http.MethodGet, base+"/prices/export", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "spreadsheetml") || len(raw) < 200 {
		t.Fatalf("export %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	// Award, then undo.
	award := `{"submission_id":"` + sub + `","lines":[{"label":"First stage","item_keys":["` + first + `"]}],"total":"2000.00"}`
	var preview struct {
		Total string `json:"total"`
	}
	m.call(t, http.MethodPost, base+"/award/preview", []byte(award), http.StatusOK, &preview)
	if preview.Total != "2000.00" {
		t.Fatalf("preview total %s", preview.Total)
	}
	if got := m.status(t, http.MethodPost, base+"/award", []byte(strings.Replace(award, "2000.00", "1999.99", 1))); got != http.StatusUnprocessableEntity {
		t.Fatalf("award with a wrong total = %d", got)
	}
	m.call(t, http.MethodPost, base+"/award", []byte(award), http.StatusCreated, &table)
	if table.Award == nil || table.Award.Total != "2000.00" || table.Columns[0].Submission.Status != "awarded" {
		t.Fatalf("award %+v", table.Award)
	}
	var report struct {
		Report struct {
			Draft string `json:"draft"`
		} `json:"report"`
	}
	m.getJSON(t, base+"/report", &report)
	if !strings.Contains(report.Report.Draft, "Example Planning") {
		t.Fatalf("report draft %q", report.Report.Draft)
	}
	m.call(t, http.MethodDelete, base+"/award", nil, http.StatusOK, &table)
	if table.Award != nil || table.Columns[0].Submission.Status != "received" {
		t.Fatalf("after undo %+v", table.Columns[0].Submission)
	}

	// The RFP's response form downloads for its org only.
	var created struct {
		Report struct {
			ID string `json:"id"`
		} `json:"report"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/create", []byte(`{"package_id":"`+pkg.ID+`"}`), http.StatusCreated, &created)
	m.call(t, http.MethodPost, "/api/reports/"+created.Report.ID+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, nil)
	resp = m.do(t, http.MethodGet, "/api/reports/"+created.Report.ID+"/response-form", nil)
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(raw) < 500 {
		t.Fatalf("response form %d: %s", resp.StatusCode, raw)
	}
	if got := other.status(t, http.MethodGet, "/api/reports/"+created.Report.ID+"/response-form", nil); got != http.StatusNotFound {
		t.Fatalf("response form from another org = %d", got)
	}
}
