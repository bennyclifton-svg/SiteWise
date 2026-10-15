package invoices

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSplitRowsShareGSTToTheCent(t *testing.T) {
	inv := readInvoice("1", 100000, 10001)
	inv.Allocations = []Allocation{{LineID: "A", Amount: i64(33333)}, {LineID: "B", Amount: i64(33333)}, {LineID: "C", Amount: i64(33334)}}
	rows := Rows(inv)
	var ex, gst, total int64
	for _, r := range rows {
		ex += *r.Ex
		gst += *r.GST
		total += *r.Total
	}
	if len(rows) != 3 || ex != 100000 || gst != 10001 || total != 110001 {
		t.Fatalf("rows %d ex %d gst %d total %d", len(rows), ex, gst, total)
	}
}

func TestWorkbookHasNumbersAndASumRow(t *testing.T) {
	s := Summarize([]BudgetLine{{ID: "L1", Label: "Architect", Budget: i64(1000000)}}, []Invoice{readInvoice("1", 400000, 40000)}, "ex_tax")
	sheets := Workbook(ExportMeta{Project: "Petersham <&>", Exported: time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC), Filters: "none"},
		s, []ExportRow{{Number: "=HYPERLINK(\"x\")", Supplier: "Architect", Date: "2026-10-01", Period: "2026-10-01", Line: "Architect", Ex: i64(400000), GST: i64(40000), Total: i64(440000), Status: "Filed", Filename: "a.pdf"}})
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, sheets); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{}
	for _, f := range z.File {
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		body[f.Name] = string(b)
	}
	cost := body["xl/worksheets/sheet1.xml"]
	if !strings.Contains(cost, `<f>SUM(D6:D7)</f><v>4000.00</v>`) || !strings.Contains(cost, `<v>10000.00</v>`) {
		t.Fatalf("cost sheet: %s", cost)
	}
	if !strings.Contains(body["xl/worksheets/sheet1.xml"], "Petersham &lt;&amp;&gt;") {
		t.Fatal("project name not escaped")
	}
	if !strings.Contains(body["xl/workbook.xml"], `_xlnm.Print_Titles`) {
		t.Fatal("no repeated header row for print")
	}
	var csvOut bytes.Buffer
	if err := WriteCSV(&csvOut, sheets[1]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csvOut.String(), `'=HYPERLINK`) || !strings.Contains(csvOut.String(), ",4000.00,400.00,4400.00,") {
		t.Fatalf("csv: %s", csvOut.String())
	}
	if out := os.Getenv("SITEWISE_XLSX_OUT"); out != "" {
		_ = os.WriteFile(out, buf.Bytes(), 0o644)
	}
}
