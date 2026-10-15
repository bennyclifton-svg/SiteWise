package invoices

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"
)

// Row is one ledger row as the Invoices tab shows it: one per allocation.
// A split invoice's GST is shared pro rata to the cent, with the last share
// taking the remainder, so the rows add up to the invoice exactly.
type Row struct {
	InvoiceID string `json:"invoice_id"`
	Share     int    `json:"share"`
	Shares    int    `json:"shares"`
	LineID    string `json:"line_id"`
	Ex        *int64 `json:"-"`
	GST       *int64 `json:"-"`
	Total     *int64 `json:"-"`
}

// Rows splits each invoice into its allocation rows.
func Rows(inv Invoice) []Row {
	shares := Shares(inv)
	if len(shares) <= 1 {
		r := Row{InvoiceID: inv.ID, Shares: 1, Ex: inv.Ex, GST: inv.GST, Total: inv.Total}
		if len(shares) == 1 {
			r.LineID = shares[0].LineID
		}
		return []Row{r}
	}
	out := make([]Row, len(shares))
	var gstLeft int64
	if inv.GST != nil {
		gstLeft = *inv.GST
	}
	for i, a := range shares {
		r := Row{InvoiceID: inv.ID, Share: i, Shares: len(shares), LineID: a.LineID, Ex: a.Amount}
		if inv.GST != nil && inv.Ex != nil && a.Amount != nil {
			g := gstLeft
			if i < len(shares)-1 && *inv.Ex != 0 {
				g = roundDiv(*inv.GST**a.Amount, *inv.Ex)
			}
			gstLeft -= g
			t := *a.Amount + g
			r.GST, r.Total = &g, &t
		}
		out[i] = r
	}
	return out
}

func roundDiv(a, b int64) int64 {
	if b < 0 {
		a, b = -a, -b
	}
	if a >= 0 {
		return (a + b/2) / b
	}
	return -((-a + b/2) / b)
}

// Filter is what the Invoices tab is showing. Empty fields match everything;
// Line "unallocated" matches rows with no line.
type Filter struct {
	Supplier string
	Month    string // YYYY-MM
	Line     string
	Status   string
}

// String describes the filter for an export's header block.
func (f Filter) String(supplier, line func(string) string) string {
	var parts []string
	if f.Supplier != "" {
		parts = append(parts, "supplier "+supplier(f.Supplier))
	}
	if f.Month != "" {
		if t, err := time.Parse("2006-01", f.Month); err == nil {
			parts = append(parts, "month "+t.Format("January 2006"))
		}
	}
	if f.Line != "" {
		parts = append(parts, "activity line "+line(f.Line))
	}
	if f.Status != "" {
		parts = append(parts, "status "+strings.ReplaceAll(f.Status, "_", " "))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "; ")
}

// Match reports whether a row of an invoice with the shown status passes.
func (f Filter) Match(inv Invoice, row Row, status string) bool {
	if f.Supplier != "" && inv.PackageID != f.Supplier {
		return false
	}
	if f.Month != "" && !strings.HasPrefix(inv.Period, f.Month) {
		return false
	}
	if f.Line == "unallocated" && row.LineID != "" || f.Line != "" && f.Line != "unallocated" && row.LineID != f.Line {
		return false
	}
	return f.Status == "" || f.Status == status
}

// ExportRow is one Invoices sheet row with its names resolved.
type ExportRow struct {
	Number, Supplier, Date, Period, Line, Status, Filename string
	Ex, GST, Total                                         *int64
}

// ExportMeta is the header block on each sheet.
type ExportMeta struct {
	Project  string
	Exported time.Time
	Filters  string
}

var statusWords = map[string]string{StatusFiled: "Filed", StatusNeedsReview: "Needs review", StatusNotRead: "Not read", StatusVoid: "Void", StatusSuperseded: "Superseded", StatusReading: "Reading"}

// StatusWord is a status as the ledger prints it.
func StatusWord(s string) string {
	if w, ok := statusWords[s]; ok {
		return w
	}
	return s
}

func header(meta ExportMeta, filters string) [][]Cell {
	return [][]Cell{
		{{Text: "Project", Bold: true}, {Text: meta.Project}},
		{{Text: "Exported", Bold: true}, {Text: meta.Exported.In(Sydney).Format("2 January 2006 15:04")}},
		{{Text: "Filters", Bold: true}, {Text: filters}},
		{},
	}
}

func moneyCell(p *int64) Cell {
	if p == nil {
		return Cell{}
	}
	v := *p
	return Cell{Cents: &v}
}

// Workbook is the export: the Cost plan sheet (every line, Unallocated and a
// SUM row) and the Invoices sheet (the filtered rows).
func Workbook(meta ExportMeta, s Summary, rows []ExportRow) []Sheet {
	cost := Sheet{Name: "Cost plan", Widths: []float64{44, 18, 16, 16, 16, 10}, Rows: header(meta, "none (the whole cost plan)")}
	cost.Rows = append(cost.Rows, []Cell{{Text: "Activity line", Bold: true}, {Text: "Category", Bold: true}, {Text: "Budget (ex GST)", Bold: true}, {Text: "Invoiced to date (ex GST)", Bold: true}, {Text: "Remaining", Bold: true}, {Text: "% spent", Bold: true}})
	cost.TitleRow = len(cost.Rows)
	first := len(cost.Rows) + 1
	for _, l := range s.Lines {
		label := l.Label
		if l.Archived {
			label += " (removed from the cost plan)"
		}
		inv := l.Invoiced
		row := []Cell{{Text: label}, {Text: l.Category}, moneyCell(l.Budget), {Cents: &inv}, moneyCell(l.Remaining), {}}
		if l.Percent != "" {
			row[5] = percentCell(l.Percent)
		}
		cost.Rows = append(cost.Rows, row)
	}
	un := s.Unallocated.Invoiced
	cost.Rows = append(cost.Rows, []Cell{{Text: "Unallocated"}, {}, {}, {Cents: &un}, {}, {}})
	last := len(cost.Rows)
	total := s.Total
	budget := s.Budget
	sum := func(col string) string { return "SUM(" + col + itoa(first) + ":" + col + itoa(last) + ")" }
	totalRow := []Cell{{Text: "Total", Bold: true}, {}, {Formula: sum("C"), Cents: &budget, Bold: true}, {Formula: sum("D"), Cents: &total, Bold: true}, {}, {}}
	if s.Remaining != nil {
		r := *s.Remaining
		totalRow[4] = Cell{Formula: "C" + itoa(last+1) + "-D" + itoa(last+1), Cents: &r, Bold: true}
	}
	if s.Percent != "" {
		totalRow[5] = percentCell(s.Percent)
	}
	cost.Rows = append(cost.Rows, totalRow)

	invs := Sheet{Name: "Invoices", Widths: []float64{18, 28, 13, 12, 36, 14, 14, 14, 14, 44}, Rows: header(meta, meta.Filters)}
	invs.Rows = append(invs.Rows, []Cell{{Text: "Invoice no.", Bold: true}, {Text: "Supplier", Bold: true}, {Text: "Invoice date", Bold: true}, {Text: "Month", Bold: true}, {Text: "Activity line", Bold: true}, {Text: "Ex GST (AUD)", Bold: true}, {Text: "GST (AUD)", Bold: true}, {Text: "Total inc GST (AUD)", Bold: true}, {Text: "Status", Bold: true}, {Text: "Source file", Bold: true}})
	invs.TitleRow = len(invs.Rows)
	for _, r := range rows {
		invs.Rows = append(invs.Rows, []Cell{{Text: r.Number}, {Text: r.Supplier}, {Date: r.Date}, {Month: r.Period}, {Text: r.Line}, moneyCell(r.Ex), moneyCell(r.GST), moneyCell(r.Total), {Text: r.Status}, {Text: r.Filename}})
	}
	return []Sheet{cost, invs}
}

func percentCell(p string) Cell {
	var f float64
	for i, part := range strings.SplitN(strings.TrimPrefix(p, "-"), ".", 2) {
		n := 0.0
		for _, r := range part {
			n = n*10 + float64(r-'0')
		}
		if i == 0 {
			f = n
		} else {
			f += n / 10
		}
	}
	if strings.HasPrefix(p, "-") {
		f = -f
	}
	return Cell{Number: &f}
}

func itoa(n int) string { return strconv.Itoa(n) }

// WriteCSV writes one sheet as CSV: the header block, then the table.
// Money is a plain decimal so a spreadsheet reads it as a number.
func WriteCSV(w io.Writer, s Sheet) error {
	cw := csv.NewWriter(w)
	for _, row := range s.Rows {
		rec := make([]string, len(row))
		for i, c := range row {
			switch {
			case c.Cents != nil:
				rec[i] = FormatCents(*c.Cents)
			case c.Number != nil:
				rec[i] = strings.TrimRight(strings.TrimRight(formatFloat(*c.Number), "0"), ".")
			case c.Date != "":
				rec[i] = c.Date
			case c.Month != "":
				rec[i] = c.Month[:7]
			default:
				rec[i] = csvSafe(c.Text)
			}
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func formatFloat(f float64) string {
	tenths := int64(f*10 + 0.5)
	if f < 0 {
		tenths = int64(f*10 - 0.5)
	}
	sign := ""
	if tenths < 0 {
		sign, tenths = "-", -tenths
	}
	return sign + strconv.FormatInt(tenths/10, 10) + "." + strconv.FormatInt(tenths%10, 10)
}

// csvSafe stops a supplier-controlled text from running as a spreadsheet
// formula when the CSV is opened.
// https://owasp.org/www-community/attacks/CSV_Injection
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
