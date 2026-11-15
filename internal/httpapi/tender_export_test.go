package httpapi_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	psample "sitewise/internal/proposals/sample"
	tsample "sitewise/internal/tender/sample"
)

// Every firm and figure in these tests is invented.

func csvCents(t *testing.T, v string) int64 {
	t.Helper()
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		t.Fatalf("%q is not an amount", v)
	}
	return int64(f*100 + 0.5)
}

// The export, the comparison report and the board for a package in tender
// mode: both tables, applied amounts only in "Like for like", and sign-off
// held until every column is checked.
func TestTenderExportReportAndBoard(t *testing.T) {
	_, m, project, base, labels := tenderSetup(t, withTender(tsample.Model{}, tinyRender))

	// Before any tender is read the export is the old one-sheet workbook.
	old := m.raw(t, http.MethodGet, base+"/prices/export")
	if !bytes.HasPrefix(old, []byte("PK")) {
		t.Fatal("the old export is not a workbook")
	}

	m.modelSwitch(t, true, http.StatusOK)
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels, psample.Line{Label: "Zebra crossing study", Cents: 50000})
	table := m.waitTender(t, base, sub, "read")
	sum := money(t, table.column(sub).TenderSum)
	ops := base + "/tender/ops"

	// One applied row and one typed-but-different firm's nothing: add a row and apply an amount.
	var edited tenderTableBody
	m.call(t, http.MethodPost, ops, opsBody(table.Tender.Version, map[string]any{"op": "add_adjustment", "label": "Site shed"}), http.StatusOK, &edited)
	var shed string
	for _, a := range edited.Tender.Adjustments {
		if a.Label == "Site shed" {
			shed = a.ID
		}
	}
	m.call(t, http.MethodPost, ops, opsBody(edited.Tender.Version, map[string]any{"op": "set_adjustment_amount", "adjustment": shed, "submission": sub, "amount": "-250"}), http.StatusOK, &edited)

	// CSV: the two tables. The first column of figures adds to the tender sum.
	resp := m.do(t, http.MethodGet, base+"/prices/export?format=csv", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") {
		t.Fatalf("csv: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var cells, tenderSum, likeForLike int64
	section := 0
	for _, rec := range records {
		switch {
		case rec[0] == "Addition or subtraction":
			section++ // the second table's head
		case section == 0 && rec[0] == "Tender sum":
			tenderSum = csvCents(t, rec[1])
		case section == 0 && !strings.Contains(rec[0], "as tendered") && len(rec) > 1 && strings.ContainsAny(rec[1], "0123456789"):
			cells += csvCents(t, rec[1])
		case section == 1 && strings.HasPrefix(rec[0], "Like for like"):
			likeForLike = csvCents(t, rec[1])
		case section == 1 && rec[0] == "Site shed":
			if rec[1] != "-250.00" || rec[2] != "applied" {
				t.Fatalf("adds and subs row %v", rec)
			}
		}
	}
	if section != 1 || tenderSum != int64(sum*100+0.5) || cells != tenderSum || likeForLike != tenderSum-25000 {
		t.Fatalf("sections %d, cells %d, tender sum %d, like for like %d", section+1, cells, tenderSum, likeForLike)
	}
	if strings.Contains(string(raw), "=") {
		t.Fatal("the export carries a formula")
	}
	// The workbook has three sheets now.
	book := m.raw(t, http.MethodGet, base+"/prices/export")
	if !bytes.HasPrefix(book, []byte("PK")) || !bytes.Contains(book, []byte("sheet3.xml")) || len(book) == len(old) {
		t.Fatalf("workbook of %d bytes", len(book))
	}

	// The report carries the figures and is signed off without a column check.
	var report struct {
		Table tenderTableBody `json:"table"`
	}
	m.getJSON(t, base+"/report", &report)
	if c := report.Table.column(sub); c == nil || money(t, c.LikeForLike) != sum-250 {
		t.Fatalf("report table %+v", c)
	}
	recommend, _ := json.Marshal(map[string]any{"text": "Appoint the invented firm.", "sign_off": true})
	if code, words := m.text(t, http.MethodPut, base+"/report", recommend); code != http.StatusOK {
		t.Fatalf("sign-off of an unchecked column: %d %q", code, words)
	}

	// The board card shows the like-for-like total and never says "to check".
	var board struct {
		Packages map[string][]struct {
			SubmissionID string  `json:"submission_id"`
			BaseTotal    *string `json:"base_total"`
			LikeForLike  bool    `json:"like_for_like"`
			ToCheck      bool    `json:"to_check"`
			Lowest       bool    `json:"lowest"`
		} `json:"packages"`
	}
	m.getJSON(t, "/api/projects/"+project+"/procurement/board", &board)
	found := false
	for _, cards := range board.Packages {
		for _, c := range cards {
			if c.SubmissionID == sub {
				found = true
				if !c.LikeForLike || c.ToCheck || money(t, c.BaseTotal) != sum-250 || !c.Lowest {
					t.Fatalf("card %+v", c)
				}
			}
		}
	}
	if !found {
		t.Fatal("no board card for the tender")
	}
}
