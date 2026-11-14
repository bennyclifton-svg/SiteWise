package httpapi

import (
	"bytes"
	"encoding/csv"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/store"
	"sitewise/internal/tender"
	"sitewise/internal/xlsx"
)

// tenderMode reports whether a package's table is shown as the two tender
// tables: at least one of its firms has been read by the tender pass.
func tenderMode(t store.PriceTable) bool {
	if t.Tender == nil {
		return false
	}
	for _, c := range t.Tender.Columns {
		if c.State == store.TenderRead {
			return true
		}
	}
	return false
}

// tenderSheets lays the two tender tables and every line out as a workbook:
// "As tendered", "Adds and subs" and "Lines". Every figure is a value
// computed in Go; there are no formulas, so the file says what the screen
// said on the day it was made.
func tenderSheets(t store.PriceTable) []xlsx.Sheet {
	bold := func(s string) xlsx.Cell { return xlsx.Cell{Text: s, Bold: true} }
	money := func(c *int64, isBold bool) xlsx.Cell {
		if c == nil {
			return xlsx.Cell{Text: "Unknown", Bold: isBold}
		}
		v := *c
		return xlsx.Cell{Cents: &v, Bold: isBold}
	}
	tcol := map[string]*store.TenderColumn{}
	for i := range t.Tender.Columns {
		tcol[t.Tender.Columns[i].SubmissionID] = &t.Tender.Columns[i]
	}
	name := func(c store.FirmColumn) string {
		if c.Firm != nil {
			return c.Firm.Name
		}
		return c.Submission.Filename
	}
	words := map[string]string{tender.StateIncluded: "Included", tender.StateExcluded: "Excluded", tender.StateNotPriced: "Not priced"}

	// 1. As tendered.
	first := xlsx.Sheet{Name: "As tendered", Widths: []float64{46}, TitleRow: 1}
	head := []xlsx.Cell{bold(t.Title + " — as tendered (ex GST)")}
	for _, c := range t.Columns {
		head = append(head, bold(name(c)))
		first.Widths = append(first.Widths, 20)
	}
	first.Rows = append(first.Rows, head)
	for _, row := range t.Tender.Rows {
		r := []xlsx.Cell{{Text: row.Label}}
		for _, c := range t.Columns {
			tc := tcol[c.Submission.ID]
			switch {
			case tc == nil:
				// Read the old way: its own cell, as the old export writes it.
				old, ok := c.Cells[row.Key]
				switch {
				case !ok || old.State == "":
					r = append(r, xlsx.Cell{Text: "—"})
				case old.State == tender.StatePriced && old.Cents != nil:
					r = append(r, money(old.Cents, false))
				default:
					r = append(r, xlsx.Cell{Text: words[old.State]})
				}
			case tc.State != store.TenderRead:
				r = append(r, xlsx.Cell{Text: "Prices not read"})
			default:
				cell, ok := tc.Cells[row.Key]
				switch {
				case !ok || cell.State == "":
					r = append(r, xlsx.Cell{Text: "—"})
				case cell.State == tender.StatePriced && cell.Cents != nil:
					r = append(r, money(cell.Cents, false))
				case cell.State == tender.StatePriced:
					r = append(r, xlsx.Cell{Text: "Amount to type"})
				default:
					r = append(r, xlsx.Cell{Text: words[cell.State]})
				}
			}
		}
		first.Rows = append(first.Rows, r)
	}
	sum := []xlsx.Cell{bold("Tender sum")}
	for _, c := range t.Columns {
		switch tc := tcol[c.Submission.ID]; {
		case tc == nil:
			sum = append(sum, money(c.Totals.BaseTotal, true))
		case tc.State != store.TenderRead:
			sum = append(sum, bold("Prices not read"))
		default:
			sum = append(sum, money(tc.Sum, true))
		}
	}
	first.Rows = append(first.Rows, sum)

	// 2. Adds and subs: an amount and its state for each firm.
	second := xlsx.Sheet{Name: "Adds and subs", Widths: []float64{46}, TitleRow: 1}
	head = []xlsx.Cell{bold("Addition or subtraction")}
	for _, c := range t.Columns {
		head = append(head, bold(name(c)), bold("Applied or suggested"))
		second.Widths = append(second.Widths, 18, 18)
	}
	second.Rows = append(second.Rows, head)
	for _, a := range t.Tender.Adjustments {
		r := []xlsx.Cell{{Text: a.Label}}
		for _, c := range t.Columns {
			m, ok := a.Amounts[c.Submission.ID]
			switch {
			case !ok:
				r = append(r, xlsx.Cell{}, xlsx.Cell{})
			case m.Cents == nil:
				r = append(r, xlsx.Cell{Text: "Not priced"}, xlsx.Cell{Text: m.State})
			default:
				r = append(r, money(m.Cents, false), xlsx.Cell{Text: m.State})
			}
		}
		second.Rows = append(second.Rows, r)
	}
	like := []xlsx.Cell{bold("Like for like (applied only)")}
	for _, c := range t.Columns {
		if tc := tcol[c.Submission.ID]; tc != nil && tc.State == store.TenderRead {
			like = append(like, money(tc.LikeForLike, true), xlsx.Cell{})
		} else {
			like = append(like, xlsx.Cell{}, xlsx.Cell{})
		}
	}
	second.Rows = append(second.Rows, like)

	// 3. Every line each firm printed.
	rowLabel := map[string]string{"": "Not broken down"}
	for _, row := range t.Tender.Rows {
		rowLabel[row.Key] = row.Label
	}
	for _, it := range t.Items {
		if _, ok := rowLabel[it.Key]; !ok {
			rowLabel[it.Key] = it.Label
		}
	}
	lines := xlsx.Sheet{Name: "Lines", Widths: []float64{26, 30, 46, 16, 16, 10, 8, 10, 60}, TitleRow: 1}
	lines.Rows = append(lines.Rows, []xlsx.Cell{bold("Firm"), bold("Row"), bold("The firm's wording"), bold("Printed"), bold("Amount ex GST"), bold("GST"), bold("Page"), bold("Counted"), bold("Notes")})
	for _, c := range t.Columns {
		tc := tcol[c.Submission.ID]
		if tc == nil || tc.State != store.TenderRead {
			continue
		}
		for _, l := range tc.Lines {
			counted := "No"
			if l.Counted {
				counted = "Yes"
			}
			var notes []string
			for _, f := range l.Flags {
				notes = append(notes, f.Text)
			}
			if l.Note != "" {
				notes = append(notes, l.Note)
			}
			amount := xlsx.Cell{Text: words[l.State]}
			if l.Cents != nil {
				amount = money(l.Cents, false)
			} else if l.State == tender.StatePriced {
				amount = xlsx.Cell{Text: "Amount to type"}
			}
			page := xlsx.Cell{}
			if l.Page > 0 {
				p := float64(l.Page)
				page = xlsx.Cell{Number: &p}
			}
			lines.Rows = append(lines.Rows, []xlsx.Cell{{Text: name(c)}, {Text: rowLabel[l.Row]}, {Text: l.Label}, {Text: l.Printed}, amount, {Text: l.GST}, page, {Text: counted}, {Text: strings.Join(notes, " ")}})
		}
	}
	return []xlsx.Sheet{first, second, lines}
}

// sheetsCSV writes sheets one after the other as CSV, a blank line between
// them. Money is an exact decimal string.
func sheetsCSV(sheets []xlsx.Sheet) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for i, s := range sheets {
		if i > 0 {
			if err := w.Write([]string{""}); err != nil {
				return nil, err
			}
		}
		for _, row := range s.Rows {
			rec := make([]string, len(row))
			for j, c := range row {
				switch {
				case c.Cents != nil:
					rec[j] = invoices.FormatCents(*c.Cents)
				default:
					rec[j] = c.Text
				}
			}
			if err := w.Write(rec); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
