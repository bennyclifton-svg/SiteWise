package httpapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/latency"
	psample "sitewise/internal/proposals/sample"
)

func TestQuotedComparisonFiveFirmsBudget(t *testing.T) {
	a := newApp(t, withProfile(t), greenProposals)
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Quoted comparison budget")
	var pkg struct{ ID string }
	m.call(t, http.MethodPost, "/api/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.architect"}`), http.StatusCreated, &pkg)
	base := "/api/projects/" + project + "/procurement/packages/" + pkg.ID
	var sub string
	for i := 0; i < 5; i++ {
		pdf := psample.PDF(psample.Proposal{Firm: fmt.Sprintf("Design Practice %d Pty Ltd", i), Lines: []psample.Line{{Label: "Concept and documentation lump sum", Cents: 1200000}}})
		var up struct {
			SubmissionID string `json:"submission_id"`
		}
		m.call(t, http.MethodPost, base+"/submissions?name="+url.QueryEscape(fmt.Sprintf("Design fee %d.pdf", i)), pdf, http.StatusCreated, &up)
		sub = up.SubmissionID
	}
	var table priceTableBody
	deadline := time.Now().Add(20 * time.Second)
	for {
		m.getJSON(t, base+"/prices", &table)
		ready := len(table.Columns) == 5
		for _, c := range table.Columns {
			ready = ready && c.Submission.Status != "reading"
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("five readings not ready: %+v", table)
		}
		time.Sleep(20 * time.Millisecond)
	}
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	scoped := latency.Budgets{MinSamples: budgets.MinSamples}
	for _, p := range budgets.Paths {
		if p.Name == "price_table" || p.Name == "submission_write" {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	samples := map[string][]int64{}
	for i := 0; i < budgets.MinSamples; i++ {
		start := time.Now()
		m.getJSON(t, base+"/prices", &table)
		samples["price_table"] = append(samples["price_table"], time.Since(start).Microseconds())
		start = time.Now()
		m.call(t, http.MethodPut, "/api/projects/"+project+"/submissions/"+sub+"/cells/comparison:appointment", []byte(`{"state":"priced","amount":"12000","note":"Concept and documentation appointment, source lump sum"}`), http.StatusOK, &table)
		samples["submission_write"] = append(samples["submission_write"], time.Since(start).Microseconds())
	}
	code, report := latency.Gate(scoped, samples, true)
	for name, values := range samples {
		p50, _ := latency.Percentile(values, .5)
		p90, _ := latency.Percentile(values, .9)
		t.Logf("%s p50=%d us p90=%d us", name, p50, p90)
	}
	if code != 0 {
		t.Fatalf("quoted comparison exceeded its existing budget: %s", report)
	}
}
