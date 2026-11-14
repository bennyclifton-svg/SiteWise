package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// benchTender is a 330-line tender (invented figures): six packs of 54
// lines and a subtotal each, spread over the package's rows, and a total.
func benchTender(rows []string, seed int) (tender.RawReading, []tender.Page, []tender.Row) {
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
	return tender.RawReading{Basis: "fixed", GST: "ex", Lines: lines}, []tender.Page{{Number: 1, Text: strings.Join(text, "\n")}}, trows
}

type tenderBenchTable struct {
	Change string `json:"change"`
	Items  []struct {
		Key  string `json:"key"`
		Base bool   `json:"base"`
	} `json:"items"`
	Columns []struct {
		Submission struct {
			ID string `json:"id"`
		} `json:"submission"`
	} `json:"columns"`
	Tender struct {
		Version int64 `json:"version"`
		Columns []struct {
			SubmissionID string `json:"submission_id"`
			Lines        []struct {
				ID      string `json:"id"`
				Row     string `json:"row"`
				Counted bool   `json:"counted"`
			} `json:"lines"`
		} `json:"columns"`
	} `json:"tender"`
}

// tenderTables times the tender comparison at its full width
// (docs/design/2026-11-14-tender-comparison.md, section 12): one package,
// five firms of 330 lines each. tender_reconcile is the Go work after the
// model answers; price_table_tender the table read; tender_edit a line
// move and its undo. No model is called: the readings are committed as the
// background reader would commit them.
func (a *apiClient) tenderTables(ctx context.Context, st *store.Store, n int) error {
	project, err := a.createProject(ctx, "Tender comparison benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project
	raw, err := a.call(ctx, http.MethodPost, base+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	var pkg procurement.Package
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return err
	}
	prices := base + "/procurement/packages/" + pkg.ID + "/prices"
	var t tenderBenchTable
	if raw, err = a.call(ctx, http.MethodGet, prices, nil, http.StatusOK); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		return err
	}
	var rows []string
	for _, it := range t.Items {
		if it.Base {
			rows = append(rows, it.Key)
		}
	}
	if len(rows) == 0 {
		return errors.New("tender benchmark: the package has no fee rows")
	}
	for firm := 1; firm <= 5; firm++ {
		body, _ := json.Marshal(map[string]any{"firm": map[string]string{"name": fmt.Sprintf("Tender Firm %02d Pty Ltd", firm)}})
		raw, err := a.call(ctx, http.MethodPost, base+"/procurement/packages/"+pkg.ID+"/firms", body, http.StatusCreated)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &t); err != nil || len(t.Columns) != firm {
			return fmt.Errorf("tender benchmark: firm %d not added: %v", firm, err)
		}
		reading, pages, trows := benchTender(rows, firm)
		start := time.Now()
		r := tender.Reconcile(reading, pages, trows, nil)
		tender.Cells(r, trows)
		a.samples.Add("tender_reconcile", time.Since(start))
		if r.TenderSum == nil || r.Gap != 0 {
			return errors.New("tender benchmark: the generated tender does not reconcile")
		}
		if err := st.CommitTenderRead(ctx, benchOrg, t.Columns[firm-1].Submission.ID, r, tender.PromptVersion, false, time.Now()); err != nil {
			return err
		}
	}
	// The reconcile budget wants its own sample count, not five.
	reading, pages, trows := benchTender(rows, 9)
	for i := 0; i < n; i++ {
		start := time.Now()
		tender.Cells(tender.Reconcile(reading, pages, trows, nil), trows)
		a.samples.Add("tender_reconcile", time.Since(start))
	}
	ops := base + "/procurement/packages/" + pkg.ID + "/tender/ops"
	for i := 0; i < n; i++ {
		raw, err := a.timed(ctx, "price_table_tender", http.MethodGet, prices, nil, http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}
		if len(t.Tender.Columns) != 5 || len(t.Tender.Columns[0].Lines) < 330 {
			return errors.New("tender benchmark: the table lacks the five readings")
		}
		// Move one counted line to the next row, then undo it.
		col := t.Tender.Columns[i%5]
		line := col.Lines[i%300]
		to := rows[0]
		if line.Row == to {
			to = rows[1%len(rows)]
		}
		body, _ := json.Marshal(map[string]any{"version": t.Tender.Version, "ops": []map[string]any{{"op": "move_line", "line": line.ID, "row": to}}})
		raw, err = a.timed(ctx, "tender_edit", http.MethodPost, ops, body, http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &t); err != nil || t.Change == "" {
			return fmt.Errorf("tender benchmark: the move returned no change to undo: %v", err)
		}
		if _, err := a.timed(ctx, "tender_edit", http.MethodPost, base+"/changes/"+t.Change+"/undo", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}
