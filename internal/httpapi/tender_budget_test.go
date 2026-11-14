package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sitewise/internal/latency"
	"sitewise/internal/tender"
	tsample "sitewise/internal/tender/sample"
)

// bigTender commits a 330-line reading (invented figures) to one submission:
// six packs of 54 lines and a subtotal each, spread over the package's rows.
func bigTender(t *testing.T, a *app, org, sub string, rows []string, seed int) {
	t.Helper()
	var lines []tender.RawLine
	var text []string
	var total int64
	for pack := 0; pack < 6; pack++ {
		group := fmt.Sprintf("Pack %d", pack+1)
		var sum int64
		for i := 0; i < 54; i++ {
			c := int64(10000+i*137+pack*11+seed*7) * 100
			sum += c
			amt := fmt.Sprintf("$%d.00", c/100)
			lines = append(lines, tender.RawLine{P: 1, Label: fmt.Sprintf("Trade line %d.%d", pack+1, i+1), Amt: amt, G: "ex", St: "priced", Kind: "base", Where: "sum", Group: group, Row: rows[i%len(rows)], Sure: true})
			text = append(text, amt)
		}
		amt := fmt.Sprintf("$%d.00", sum/100)
		lines = append(lines, tender.RawLine{P: 1, Label: "Sub total", Amt: amt, G: "ex", St: "priced", Kind: "subtotal", Where: "inside", Group: group, Sure: true})
		text = append(text, amt)
		total += sum
	}
	amt := fmt.Sprintf("$%d.00", total/100)
	lines = append(lines, tender.RawLine{P: 1, Label: "Tender sum", Amt: amt, G: "ex", St: "priced", Kind: "total", Where: "inside", Sure: true})
	text = append(text, amt)
	var trows []tender.Row
	for _, k := range rows {
		trows = append(trows, tender.Row{Key: k, Label: k})
	}
	r := tender.Reconcile(tender.RawReading{Basis: "fixed", GST: "ex", Lines: lines}, []tender.Page{{Number: 1, Text: strings.Join(text, "\n")}}, trows, nil)
	if r.TenderSum == nil || *r.TenderSum != total || r.Gap != 0 {
		t.Fatalf("the generated tender does not reconcile: sum %v gap %d", r.TenderSum, r.Gap)
	}
	if err := a.store.CommitTenderRead(context.Background(), org, sub, r, tender.PromptVersion, false, time.Now()); err != nil {
		t.Fatal(err)
	}
}

// fiveBigTenders is a package with five firms, each with a 330-line reading.
func fiveBigTenders(t *testing.T) (*app, *member, string, string, []string) {
	t.Helper()
	a, m, project, base, _ := tenderSetup(t, withTender(tsample.Model{}, tinyRender))
	m.modelSwitch(t, true, http.StatusOK)
	var table tenderTableBody
	m.getJSON(t, base+"/prices", &table)
	var rows []string
	for _, it := range table.Items {
		if it.Base {
			rows = append(rows, it.Key)
		}
	}
	var subs []string
	for firm := 1; firm <= 5; firm++ {
		body, _ := json.Marshal(map[string]any{"firm": map[string]string{"name": fmt.Sprintf("Invented Builder %02d Pty Ltd", firm)}})
		m.call(t, http.MethodPost, base+"/firms", body, http.StatusCreated, &table)
		sub := table.Columns[len(table.Columns)-1].Submission.ID
		bigTender(t, a, m.orgID, sub, rows, firm)
		subs = append(subs, sub)
	}
	return a, m, project, base, subs
}

func gateTender(t *testing.T, names []string, samples map[string][]int64) {
	t.Helper()
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	scoped := latency.Budgets{MinSamples: budgets.MinSamples}
	for _, p := range budgets.Paths {
		for _, n := range names {
			if p.Name == n {
				scoped.Paths = append(scoped.Paths, p)
			}
		}
	}
	if len(scoped.Paths) != len(names) {
		t.Fatalf("bench/budgets.json lacks one of %v", names)
	}
	code, report := latency.Gate(scoped, samples, true)
	for name, values := range samples {
		p50, _ := latency.Percentile(values, .5)
		p90, _ := latency.Percentile(values, .9)
		t.Logf("%s p50=%d us p90=%d us", name, p50, p90)
	}
	if code != 0 {
		t.Fatalf("tender budget broken: %s", report)
	}
}

// TestTenderTableFiveFirmsBudget holds price_table_tender: the price table
// of one package with five tender-read firms of 330 lines each.
func TestTenderTableFiveFirmsBudget(t *testing.T) {
	_, m, _, base, subs := fiveBigTenders(t)
	var table tenderTableBody
	m.getJSON(t, base+"/prices", &table)
	if len(table.Tender.Columns) != 5 || len(table.column(subs[0]).Lines) < 330 {
		t.Fatalf("%d tender columns", len(table.Tender.Columns))
	}
	samples := map[string][]int64{}
	for i := 0; i < 30; i++ {
		start := time.Now()
		raw := m.raw(t, http.MethodGet, base+"/prices")
		samples["price_table_tender"] = append(samples["price_table_tender"], time.Since(start).Microseconds())
		if len(raw) == 0 {
			t.Fatal("empty table")
		}
	}
	gateTender(t, []string{"price_table_tender"}, samples)
}
