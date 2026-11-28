package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"sitewise/internal/reports"
)

// scheduleSections is a payment schedule as reports.ScheduleSheetSections
// saves it: a label and value grid, a lines table with a reason and a bold
// sub total, a payment list whose first four columns are words, and a row
// with room to sign.
func scheduleSections(lines int) []reports.Section {
	basis := func(v map[string]any) json.RawMessage {
		raw, _ := json.Marshal(v)
		return raw
	}
	doc := func(cols []string, rows ...[]string) *reports.Table {
		return &reports.Table{Columns: cols, Rows: rows, Display: "document"}
	}
	works := doc([]string{"Description", "Total let ($)", "(%)", "Completed ($)", "Less prior ($)", "This claim ($)"})
	bold, notes := []int{}, []int{}
	for i := 0; i < lines; i++ {
		works.Rows = append(works.Rows, []string{fmt.Sprintf("Line %d\nits note", i+1), "100,000.00", "90.0", "90,000.00", "60,000.00", "30,000.00"})
		if i == 0 {
			notes = append(notes, len(works.Rows))
			works.Rows = append(works.Rows, []string{"Reason: Footings to grid C not poured", "", "", "", "", ""})
		}
	}
	bold = append(bold, len(works.Rows))
	works.Rows = append(works.Rows, []string{"Sub total ex GST ($)", "150,000.00", "", "110,000.00", "60,000.00", "50,000.00"})
	bold = append(bold, len(works.Rows))
	works.Rows = append(works.Rows, []string{"Total claim inc retention (ex GST)", "", "", "", "", "49,900.00"})
	return []reports.Section{
		{ID: "details", Blocks: []reports.Block{
			{ID: "schedule:details", Basis: basis(map[string]any{"layout": "label_value"}), Table: doc([]string{"Parties", "Claim"},
				[]string{"Project: Harbour Street", "Claim no.: 02"}, []string{"Principal: Harbour Trust", "Period: OCT26"})},
			{ID: "schedule:act", Text: "This is a payment schedule."}}},
		{ID: "works", Title: "Contract summary: Works", Blocks: []reports.Block{
			{ID: "schedule:works", Basis: basis(map[string]any{"layout": "schedule", "left": 1, "merge": true, "widths": []float64{0, 76, 42, 76, 76, 76}, "bold": bold, "notes": notes}), Table: works}}},
		{ID: "payments", Title: "Payment list", Blocks: []reports.Block{
			{ID: "schedule:payments", Basis: basis(map[string]any{"layout": "schedule", "left": 4, "widths": []float64{0, 60, 0, 0, 80, 70, 84}, "bold": []int{1}}),
				Table: doc([]string{"Period", "Claim no.", "Invoice no.", "Invoice date", "Value ex GST ($)", "GST ($)", "Value inc GST ($)"},
					[]string{"OCT26", "02", "Awaiting invoice", "", "49,900.00", "4,990.00", "54,890.00"},
					[]string{"Total", "", "", "", "49,900.00", "4,990.00", "54,890.00"})}}},
		{ID: "approval", Title: "Approval", Blocks: []reports.Block{
			{ID: "schedule:approval", Basis: basis(map[string]any{"layout": "schedule", "left": 3, "tall": []int{0}}),
				Table: doc([]string{"Approved by", "Signature", "Date"}, []string{"A. Manager", "", "16 October 2026"})}}},
	}
}

// A schedule's figures are right aligned under right-aligned heads; a sub
// total is bold with a strong rule; a reason is one cell across the table.
func TestScheduleTablesAlignFigures(t *testing.T) {
	d := FromSections("Payment schedule 02", "Harbour Street · Main Works", "2026-10-16", scheduleSections(2))
	if len(d.Sections) != 4 || d.Sections[0].Title != "" {
		t.Fatalf("sections %+v", d.Sections)
	}
	// The details read as the Profile card does: a head over label and value.
	details := d.Sections[0].Blocks[0].Table
	if len(details.Columns) != 4 || len(details.Head) != 2 || details.Head[0].ColSpan != 2 || !details.Rows[0].Cells[0].Runs[0].Muted {
		t.Fatalf("details %+v", details)
	}
	works := d.Sections[1].Blocks[0].Table
	if len(works.Columns) != 6 || works.Columns[0].Fixed != 0 || works.Columns[1].Fixed != 76 || works.Columns[2].Fixed != 42 {
		t.Fatalf("columns %+v", works.Columns)
	}
	if works.Head[0].Align != Left || works.Head[1].Align != Right || works.Head[5].Align != Right {
		t.Fatalf("head %+v", works.Head)
	}
	line := works.Rows[0]
	if len(line.Cells) != 6 || line.Cells[0].Align != Left || line.Cells[1].Align != Right || line.Cells[5].Align != Right {
		t.Fatalf("line %+v", line)
	}
	// A line's note sits under its label, quieter.
	if runs := line.Cells[0].Runs; len(runs) != 2 || runs[0].Text != "Line 1" || runs[1].Text != "\nits note" || !runs[1].Muted {
		t.Fatalf("label runs %+v", line.Cells[0].Runs)
	}
	reason := works.Rows[1]
	if len(reason.Cells) != 1 || reason.Cells[0].ColSpan != 6 || !reason.Cells[0].Runs[0].Muted {
		t.Fatalf("reason %+v", reason)
	}
	total := works.Rows[len(works.Rows)-2]
	if !total.Strong || len(total.Cells) != 6 || !total.Cells[0].Runs[0].Bold || !total.Cells[1].Runs[0].Bold {
		t.Fatalf("sub total %+v", total)
	}
	// A label with nothing beside it runs across to its figure.
	if net := works.Rows[len(works.Rows)-1]; len(net.Cells) != 2 || net.Cells[0].ColSpan != 5 || net.Cells[1].Align != Right || net.Cells[1].Runs[0].Text != "49,900.00" {
		t.Fatalf("net %+v", net)
	}
	// The approval's empty signature cell is its own: nothing runs into it.
	if sign := d.Sections[3].Blocks[0].Table.Rows[0]; len(sign.Cells) != 3 {
		t.Fatalf("signature cells %+v", sign)
	}
	payments := d.Sections[2].Blocks[0].Table
	if payments.Rows[0].Cells[3].Align != Left || payments.Rows[0].Cells[4].Align != Right || payments.Head[4].Align != Right {
		t.Fatalf("payment list %+v", payments.Rows[0])
	}
	// Short tables stay whole; a long breakdown may run on.
	if !d.Sections[2].Blocks[0].Table.Together || !works.Together {
		t.Fatalf("short tables are kept together")
	}
	// The approval row leaves room to sign.
	if sign := d.Sections[3].Blocks[0].Table.Rows[0]; sign.MinHeight < 2*rowHeight {
		t.Fatalf("signature row %v", sign.MinHeight)
	}
	pdf, err := PDF(d)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("pdf %v", err)
	}
	if _, err := DOCX(d); err != nil {
		t.Fatalf("docx %v", err)
	}
}

// A long breakdown runs on to the next page; no row is cut to fit.
func TestScheduleOfEightyLinesRunsOn(t *testing.T) {
	d := FromSections("Payment schedule 02", "", "2026-10-16", scheduleSections(80))
	w, err := newPDFWriter()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.layout(d); err != nil {
		t.Fatal(err)
	}
	if len(w.pages) < 3 {
		t.Fatalf("eighty lines on %d pages", len(w.pages))
	}
	if d.Sections[1].Blocks[0].Table.Together {
		t.Fatal("eighty lines cannot be kept on one page")
	}
	if got := len(d.Sections[1].Blocks[0].Table.Rows); got != 83 {
		t.Fatalf("%d rows, want every line, its reason and the two totals", got)
	}
}
