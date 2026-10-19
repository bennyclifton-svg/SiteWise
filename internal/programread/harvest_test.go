package programread

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/xlsx"
)

func extract(t *testing.T, format string, body []byte) identity.Text {
	t.Helper()
	text, err := identity.Extract(context.Background(), format, bytes.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestHarvestXLSX(t *testing.T) {
	txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	var buf bytes.Buffer
	err := xlsx.WriteXLSX(&buf, []xlsx.Sheet{{Name: "Program", Rows: [][]xlsx.Cell{
		{txt("Smith Builders — construction program")},
		{},
		{txt("ID"), txt("Task Name"), txt("Duration"), txt("Start"), txt("Finish"), txt("Predecessors"), txt("Baseline Start")},
		{txt("1"), txt("Site establishment"), txt("5 days"), txt("2/03/2027"), txt("8/03/2027"), txt(""), txt("1/01/2027")},
		{txt("2"), txt("Excavate footings"), txt("1 wk?"), {Date: "2027-03-09"}, {Date: "2027-03-15"}, txt("1FS+2d"), txt("1/01/2027")},
		{txt("3"), txt("Pour slab"), txt("soon"), txt("next month"), txt(""), txt("2"), txt("")},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	h := HarvestText("Smith program.xlsx", extract(t, "xlsx", buf.Bytes()), 5)
	if len(h.Rows) != 3 {
		t.Fatalf("rows %+v", h.Rows)
	}
	r1, r2, r3 := h.Rows[0], h.Rows[1], h.Rows[2]
	if r1.ID != "r1" || r1.FileID != "1" || r1.Label != "Site establishment" || r1.StartRaw != "2/03/2027" || r1.Start != "2027-03-02" || r1.Finish != "2027-03-08" || r1.Duration == nil || r1.Duration.High != 5 {
		t.Fatalf("r1 %+v", r1)
	}
	// A sheet date is a serial number; code reads it as a date.
	if r2.Start != "2027-03-09" || r2.Finish != "2027-03-15" || r2.Duration == nil || r2.Duration.High != 5 {
		t.Fatalf("r2 %+v", r2)
	}
	if len(r2.PredLinks) != 1 || r2.PredLinks[0] != (PredecessorLink{Row: "r1", Type: "FS", Lag: 2, LagUnit: "working"}) {
		t.Fatalf("r2 links %+v", r2.PredLinks)
	}
	// A value that does not parse stays raw and unparsed.
	if r3.StartRaw != "next month" || r3.Start != "" || r3.Duration != nil || r3.DurationRaw != "soon" {
		t.Fatalf("r3 %+v", r3)
	}
	// 2/03 and 8/03 read either way: day/month is assumed and said.
	if strings.Join(h.Checks, "|") != "Date order assumed day/month in column Start|Date order assumed day/month in column Finish" {
		t.Fatalf("checks %q", h.Checks)
	}
}

func TestHarvestDayOrderDecidedPerColumn(t *testing.T) {
	txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	var buf bytes.Buffer
	if err := xlsx.WriteXLSX(&buf, []xlsx.Sheet{{Name: "P", Rows: [][]xlsx.Cell{
		{txt("Activity"), txt("Start"), txt("End")},
		{txt("Frame"), txt("3/04/2027"), txt("4/20/2027")},
		{txt("Roof"), txt("4/21/2027"), txt("5/02/2027")},
	}}}); err != nil {
		t.Fatal(err)
	}
	h := HarvestText("us.xlsx", extract(t, "xlsx", buf.Bytes()), 5)
	// 4/21 in Start and 4/20 in End can only be month/day.
	if h.Rows[0].Start != "2027-03-04" || h.Rows[1].Finish != "2027-05-02" || len(h.Checks) != 0 {
		t.Fatalf("%+v %q", h.Rows, h.Checks)
	}
}

func docxTable(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Program</w:t></w:r></w:p><w:tbl>`)
	for _, r := range rows {
		b.WriteString("<w:tr>")
		for _, c := range r {
			fmt.Fprintf(&b, `<w:tc><w:p><w:r><w:t>%s</w:t></w:r></w:p></w:tc>`, c)
		}
		b.WriteString("</w:tr>")
	}
	b.WriteString(`</w:tbl></w:body></w:document>`)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHarvestDOCXTable(t *testing.T) {
	body := docxTable(t, [][]string{
		{"Activity", "Start", "Finish", "Duration"},
		{"Demolition", "13/04/2027", "23/04/2027", "2 weeks"},
		{"Underpinning", "26 Apr 2027", "2027-05-07", "10 days"},
	})
	h := HarvestText("Programme.docx", extract(t, "docx", body), 5)
	if len(h.Rows) != 2 || len(h.Checks) != 0 {
		t.Fatalf("%+v %q", h.Rows, h.Checks)
	}
	if r := h.Rows[0]; r.Label != "Demolition" || r.Start != "2027-04-13" || r.Finish != "2027-04-23" || r.Duration.High != 10 {
		t.Fatalf("%+v", r)
	}
	if r := h.Rows[1]; r.Start != "2027-04-26" || r.Finish != "2027-05-07" || r.DurationRaw != "10 days" {
		t.Fatalf("%+v", r)
	}
}

// pdfLine lays one printed line out as positioned runs.
func pdfLine(y float64, cells ...string) []identity.Run {
	var out []identity.Run
	x := 40.0
	for _, c := range cells {
		out = append(out, identity.Run{Text: c, Source: identity.Source{Page: 1, X: x, Y: y, Width: float64(len(c)) * 5, Height: 8}})
		x += float64(len(c))*5 + 12
	}
	return out
}

func TestHarvestPDFLines(t *testing.T) {
	var runs []identity.Run
	runs = append(runs, pdfLine(800, "Construction Program — Rev B")...)
	runs = append(runs, pdfLine(780, "ID", "Task Name", "Duration", "Start", "Finish", "Predecessors")...)
	runs = append(runs, pdfLine(760, "11", "Pour slab", "5 days", "Mon 15/03/27", "Fri 19/03/27")...)
	runs = append(runs, pdfLine(745, "12", "Frame walls", "20 days", "Mon 22/03/27", "Fri 16/04/27", "11FS+2d")...)
	runs = append(runs, pdfLine(730, "Printed 1/10/2026")...)
	h := HarvestText("Program.pdf", identity.Text{Format: "pdf", TextLayer: true, Runs: runs}, 5)
	if len(h.Rows) != 3 {
		t.Fatalf("rows %+v", h.Rows)
	}
	slab, frame := h.Rows[0], h.Rows[1]
	if slab.FileID != "11" || slab.Label != "Pour slab" || slab.DurationRaw != "5 days" || slab.StartRaw != "Mon 15/03/27" || slab.Start != "2027-03-15" || slab.Finish != "2027-03-19" {
		t.Fatalf("slab %+v", slab)
	}
	if frame.PredecessorsRaw != "11FS+2d" || len(frame.PredLinks) != 1 || frame.PredLinks[0].Row != slab.ID {
		t.Fatalf("frame %+v", frame)
	}
	if h.Rows[2].Label != "Printed" {
		t.Fatalf("footer %+v", h.Rows[2])
	}
}
