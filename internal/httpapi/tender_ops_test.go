package httpapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	psample "sitewise/internal/proposals/sample"
	tsample "sitewise/internal/tender/sample"
)

// Every firm and figure in these tests is invented.

func opsBody(version int64, ops ...map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{"version": version, "ops": ops})
	return b
}

func (m *member) text(t *testing.T, method, path string, body []byte) (int, string) {
	t.Helper()
	resp := m.do(t, method, path, body)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(raw))
}

func TestTenderOpsRouteAndUndo(t *testing.T) {
	a, m, project, base, labels := tenderSetup(t, withTender(tsample.Model{}, tinyRender))
	m.modelSwitch(t, true, http.StatusOK)
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels, psample.Line{Label: "Zebra crossing study", Cents: 50000})
	table := m.waitTender(t, base, sub, "read")
	col := table.column(sub)
	var line, from, to string
	for _, l := range col.Lines {
		if l.Counted && l.Row != "" && line == "" {
			line, from = l.ID, l.Row
		}
	}
	for _, r := range table.Tender.Rows {
		if !r.System && r.Key != from && to == "" {
			to = r.Key
		}
	}
	ops := base + "/tender/ops"

	// A move answers with the table and the change that can be undone.
	var moved tenderTableBody
	m.call(t, http.MethodPost, ops, opsBody(table.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": to}), http.StatusOK, &moved)
	if moved.Change == "" || moved.Tender.Version <= table.Tender.Version {
		t.Fatalf("change %q version %d", moved.Change, moved.Tender.Version)
	}
	rowOf := func(b tenderTableBody) string {
		for _, l := range b.column(sub).Lines {
			if l.ID == line {
				return l.Row
			}
		}
		return "?"
	}
	if rowOf(moved) != to || money(t, moved.column(sub).TenderSum) != money(t, col.TenderSum) {
		t.Fatalf("row %q, sum %v", rowOf(moved), moved.column(sub).TenderSum)
	}

	// A stale version, an invalid operation, a signed-out caller, another org.
	if code, words := m.text(t, http.MethodPost, ops, opsBody(table.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": from})); code != http.StatusConflict || !strings.Contains(words, "This table changed") {
		t.Fatalf("stale: %d %q", code, words)
	}
	if code, words := m.text(t, http.MethodPost, ops, opsBody(moved.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": "stage:nowhere"})); code != http.StatusUnprocessableEntity || !strings.HasPrefix(words, "operation 1: ") {
		t.Fatalf("invalid: %d %q", code, words)
	}
	anon := &member{app: a, client: &http.Client{}}
	if got := anon.status(t, http.MethodPost, ops, opsBody(moved.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": from})); got != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", got)
	}
	other := a.member(t, newUUID(t))
	if got := other.status(t, http.MethodPost, ops, opsBody(moved.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": from})); got != http.StatusNotFound {
		t.Fatalf("another org: %d", got)
	}
	undo := "/api/projects/" + project + "/changes/" + moved.Change + "/undo"
	if got := other.status(t, http.MethodPost, undo, nil); got != http.StatusNotFound {
		t.Fatalf("another org's undo: %d", got)
	}

	// Undo through the change log's route puts the line back.
	var done struct {
		Undone bool     `json:"undone"`
		Areas  []string `json:"areas"`
	}
	m.call(t, http.MethodPost, undo, nil, http.StatusOK, &done)
	if !done.Undone || len(done.Areas) != 1 || done.Areas[0] != "procurement" {
		t.Fatalf("%+v", done)
	}
	var after tenderTableBody
	m.getJSON(t, base+"/prices", &after)
	if rowOf(after) != from {
		t.Fatalf("after undo the line is on %q", rowOf(after))
	}
	if got := m.status(t, http.MethodPost, undo, nil); got != http.StatusConflict {
		t.Fatalf("second undo: %d", got)
	}
	if got := m.status(t, http.MethodPost, "/api/projects/"+project+"/changes/not-a-uuid/undo", nil); got != http.StatusNotFound {
		t.Fatalf("malformed change id: %d", got)
	}

	// An undo over a later edit is refused, in words.
	var first, second tenderTableBody
	m.call(t, http.MethodPost, ops, opsBody(after.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": to}), http.StatusOK, &first)
	m.call(t, http.MethodPost, ops, opsBody(first.Tender.Version, map[string]any{"op": "move_line", "line": line, "row": ""}), http.StatusOK, &second)
	var refused struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/changes/"+first.Change+"/undo", nil, http.StatusUnprocessableEntity, &refused)
	if refused.Error != "changed_since" || refused.Message != "Can't undo: changed since" {
		t.Fatalf("%+v", refused)
	}

	// A column check carries no change to undo.
	var checked tenderTableBody
	m.call(t, http.MethodPost, ops, opsBody(second.Tender.Version, map[string]any{"op": "check_column", "submission": sub, "checked": true}), http.StatusOK, &checked)
	if checked.Change != "" || !checked.column(sub).Checked {
		t.Fatalf("change %q checked %v", checked.Change, checked.column(sub).Checked)
	}
}

// TestTenderEditBudget holds tender_edit and change_undo: a line move and
// its undo on a package of five firms with 330 lines each, with the
// recomputed table returned.
func TestTenderEditBudget(t *testing.T) {
	_, m, project, base, subs := fiveBigTenders(t)
	var table tenderTableBody
	m.getJSON(t, base+"/prices", &table)
	var rows []string
	for _, r := range table.Tender.Rows {
		if !r.System {
			rows = append(rows, r.Key)
		}
	}
	samples := map[string][]int64{}
	for i := 0; i < 24; i++ {
		col := table.column(subs[i%len(subs)])
		line := col.Lines[(i*13)%300]
		to := rows[0]
		if line.Row == to {
			to = rows[1]
		}
		start := time.Now()
		m.call(t, http.MethodPost, base+"/tender/ops", opsBody(table.Tender.Version, map[string]any{"op": "move_line", "line": line.ID, "row": to}), http.StatusOK, &table)
		samples["tender_edit"] = append(samples["tender_edit"], time.Since(start).Microseconds())
		if table.Change == "" {
			t.Fatal("the move returned no change")
		}
		start = time.Now()
		if got := m.status(t, http.MethodPost, fmt.Sprintf("/api/projects/%s/changes/%s/undo", project, table.Change), nil); got != http.StatusOK {
			t.Fatalf("undo: %d", got)
		}
		samples["change_undo"] = append(samples["change_undo"], time.Since(start).Microseconds())
		m.getJSON(t, base+"/prices", &table)
	}
	gateTender(t, []string{"tender_edit", "change_undo"}, samples)
}
