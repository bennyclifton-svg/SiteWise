package httpapi_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"sitewise/internal/llm"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

type rowReplyJSON struct {
	Draft store.ReportDraft `json:"draft"`
	Row   string            `json:"row"`
}

func draftTable(t *testing.T, d *store.ReportDraft, id string) *reports.Table {
	t.Helper()
	for _, s := range d.Sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID == id {
				return b.Table
			}
		}
	}
	t.Fatalf("no table %s", id)
	return nil
}

func reportView(t *testing.T, m *member, report string) store.ReportView {
	t.Helper()
	var v store.ReportView
	m.getJSON(t, "/api/reports/"+report, &v)
	return v
}

// The row routes (project report plan, section 7): each is the PM's
// session, the report's org, and the draft version the screen saw.
func TestReportRowRoutes(t *testing.T) {
	a, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "ok"} })
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	base := "/api/reports/" + report
	v := reportView(t, m, report)

	var out rowReplyJSON
	m.call(t, http.MethodPost, base+"/rows", []byte(`{"table":"approvals","cells":{"0":"Sydney Water","1":"Section 73"},"version":`+itoa(v.Draft.Version)+`}`), http.StatusOK, &out)
	if out.Row == "" {
		t.Fatal("no row id")
	}
	if got := strings.Join(draftTable(t, &out.Draft, "approvals").Rows[len(draftTable(t, &out.Draft, "approvals").Rows)-1], "|"); !strings.HasPrefix(got, "Sydney Water|Section 73") {
		t.Fatalf("new row %q", got)
	}
	m.call(t, http.MethodPatch, base+"/rows/"+out.Row, []byte(`{"cells":{"1":"Section 73 certificate"},"version":`+itoa(out.Draft.Version)+`}`), http.StatusOK, &out)
	// Refusals: an unknown table, a column out of range, a stale version, a bad id.
	for name, c := range map[string]struct {
		method, path, body string
		want               int
	}{
		"unknown table":    {http.MethodPost, base + "/rows", `{"table":"nope","cells":{"0":"x"}}`, http.StatusNotFound},
		"column":           {http.MethodPatch, base + "/rows/" + out.Row, `{"cells":{"7":"x"}}`, http.StatusUnprocessableEntity},
		"stale version":    {http.MethodPatch, base + "/rows/" + out.Row, `{"cells":{"1":"x"},"version":1}`, http.StatusConflict},
		"bad row id":       {http.MethodPatch, base + "/rows/nope", `{"cells":{"1":"x"}}`, http.StatusNotFound},
		"no table or row":  {http.MethodPost, base + "/rows", `{"cells":{"0":"x"}}`, http.StatusBadRequest},
		"period backwards": {http.MethodPut, base + "/period", `{"from":"2026-11-10","to":"2026-11-01"}`, http.StatusUnprocessableEntity},
	} {
		if got := m.status(t, c.method, c.path, []byte(c.body)); got != c.want {
			t.Fatalf("%s = %d, want %d", name, got, c.want)
		}
	}
	// Another org cannot reach any of it.
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodPost, base+"/rows", []byte(`{"table":"approvals","cells":{"0":"x"}}`)); got != http.StatusNotFound {
		t.Fatalf("foreign row = %d", got)
	}
	if got := other.status(t, http.MethodDelete, base+"/rows/"+out.Row, []byte(`{}`)); got != http.StatusNotFound {
		t.Fatalf("foreign delete = %d", got)
	}
	// The period: typed, and the cover follows it.
	var d store.ReportDraft
	m.call(t, http.MethodPut, base+"/period", []byte(`{"from":"","to":"2026-11-28","label":"To 28 November 2026"}`), http.StatusOK, &d)
	if d.PeriodLabel == nil || *d.PeriodLabel != "To 28 November 2026" {
		t.Fatalf("period %v", d.PeriodLabel)
	}
	// Delete the PM's row.
	m.call(t, http.MethodDelete, base+"/rows/"+out.Row, []byte(`{}`), http.StatusOK, &out)
	for _, row := range draftTable(t, &out.Draft, "approvals").Rows {
		if row[0] == "Sydney Water" {
			t.Fatal("deleted row still there")
		}
	}
}

// From chat (project report plan, section 9): a cell, a row, a hidden
// row, a move and a give-back apply at once with Undo; the period waits for
// the PM's click; the read names each table and row key.
func TestReportRowOperations(t *testing.T) {
	_, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "ok"} })
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	read := m.plainRead(t, project, "report")
	for _, want := range []string{"Authority approvals (table approvals;", "[ph:approvals:0] Energy distributor | Nil to report", "(placeholder)", "07 Site activity: [part site]"} {
		if !strings.Contains(read, want) {
			t.Fatalf("read lacks %q:\n%s", want, read)
		}
	}
	set := m.ops(t, project, "reports", http.StatusOK, op("reports.set_cell", map[string]any{"table": "site", "row": "Safety and induction", "column": "Summary", "text": "Inductions held for all new starters."}))
	if set.ChangeID == "" || set.Undo.Kind != "change" {
		t.Fatalf("set_cell %+v", set)
	}
	cell := func() string {
		v := reportView(t, m, report)
		for i, row := range draftTable(t, v.Draft, "site").Rows {
			if strings.HasPrefix(row[0], "Safety") {
				return draftTable(t, v.Draft, "site").Rows[i][1]
			}
		}
		return ""
	}
	if got := cell(); got != "Inductions held for all new starters." {
		t.Fatalf("cell %q", got)
	}
	m.undo(t, project, set.ChangeID, http.StatusOK)
	if got := cell(); got != "Nil to report" {
		t.Fatalf("after undo %q", got)
	}
	added := m.ops(t, project, "reports", http.StatusOK, op("reports.add_row", map[string]any{"table": "insurances", "cells": []any{"Professional indemnity", "Held by the architect"}}))
	if added.Undo.Kind != "change" {
		t.Fatalf("add_row %+v", added)
	}
	m.ops(t, project, "reports", http.StatusOK, op("reports.remove_row", map[string]any{"table": "insurances", "row": "Workers compensation"}))
	m.ops(t, project, "reports", http.StatusOK, op("reports.move_row", map[string]any{"table": "insurances", "row": "Professional indemnity", "direction": "up"}))
	v := reportView(t, m, report)
	ins := draftTable(t, v.Draft, "insurances")
	var firsts []string
	for _, row := range ins.Rows {
		firsts = append(firsts, row[0])
	}
	if got := strings.Join(firsts, ","); got != "Contract works insurance,Public liability,Professional indemnity,Security or bank guarantee" || ins.Hidden != 1 {
		t.Fatalf("insurances %q hidden %d", got, ins.Hidden)
	}
	m.ops(t, project, "reports", http.StatusOK, op("reports.restore_row", map[string]any{"table": "insurances", "row": "ph:insurances:2"}))
	if h := draftTable(t, reportView(t, m, report).Draft, "insurances").Hidden; h != 0 {
		t.Fatalf("restored, still %d hidden", h)
	}
	// Refusals in words: an unknown table, a row that is not there.
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.set_cell", map[string]any{"table": "nope", "row": "x", "column": "0", "text": "x"})); r.Field != "table" || !strings.Contains(r.Message, "approvals") {
		t.Fatalf("unknown table %+v", r)
	}
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.set_cell", map[string]any{"table": "site", "row": "Nothing here", "column": "0", "text": "x"})); r.Field != "row" {
		t.Fatalf("unknown row %+v", r)
	}
	// The period is the PM's to confirm.
	if r := m.ops(t, project, "reports", http.StatusUnprocessableEntity, op("reports.period", map[string]any{"to": "2026-11-28"})); !r.NeedsClick {
		t.Fatalf("period from ops %+v", r)
	}
}

// A cell drafted in chat carrying a figure nobody gave is refused.
func TestChatRefusesAReportCellWithItsOwnFigure(t *testing.T) {
	_, m, project, model := chatApp(t, func(n int, conv llm.Conversation) llm.Turn {
		if n == 1 {
			return applyCall("reports", true, op("reports.set_cell", map[string]any{"table": "site", "row": "Safety and induction", "column": "Summary", "text": "42 inductions held."}))
		}
		if last := string(conv.History[len(conv.History)-1]); !strings.Contains(last, "Say the figure") {
			t.Errorf("the refusal was not fed back: %s", last)
		}
		return llm.Turn{Text: "How many inductions were held?"}
	})
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	m.chat(t, project, "", "Fill in safety and induction")
	if model.calls() != 2 {
		t.Fatalf("%d calls", model.calls())
	}
	if read := m.plainRead(t, project, "report"); strings.Contains(read, "42 inductions") {
		t.Fatal("the model's own figure was written")
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
