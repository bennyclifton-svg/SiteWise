package program

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/xlsx"
)

// Export (plan docs/plans/2026-10-18-program.md section 8, WP-P11): the
// program as a table in the grid's current view. Pure formatting over the
// engine's input and output, so the same rows feed Excel and CSV and the
// tests need no database. No Jev.

// The two grid views (plan 7.5). Collapsed shows the stages only.
const (
	ViewExpanded  = "expanded"
	ViewCollapsed = "collapsed"
)

// ExportHeader is the table's first row.
var ExportHeader = []string{"ID", "Tier", "Name", "Start", "Finish", "Duration (working days)", "Predecessors", "Float (working days)", "Critical", "Origin"}

// ExportRow is one exported row. ID is the row number the grid shows: the
// 1-based position in grid order of the exported view, and what the
// Predecessors column refers to. Unknown values are blank: nil numbers, ""
// text.
type ExportRow struct {
	ID       int
	Tier     string
	Name     string
	Start    string // YYYY-MM-DD when Dated, else "Week N, day M"
	Finish   string // "" for a zero-length row, which shows its start only
	Dated    bool
	Duration *int
	// Predecessors is the grid's typed format ("12", "12SS+2d", "7FS+3cd"),
	// comma-separated, in row order.
	Predecessors string
	Float        *int
	Critical     string // "Yes", "No", or "" when unknown
	Origin       string
}

var tierWords = map[string]string{KindStage: "Stage", KindActivity: "Activity", KindMilestone: "Milestone", KindHoldPoint: "Hold point"}

// ExportRows builds the table. out is Compute(in); origin maps an activity
// id to its stored origin (library, user, document).
//
// Grid order is stages by ordinal, each followed by its activities by
// ordinal, ties by id: the order the grid shows and the store's outline.
// The collapsed view exports the stages only, and a link is exported only
// when both of its ends are exported rows. Links between activities of
// collapsed stages are omitted rather than lifted to their stages: a lifted
// link would state a relationship (type and lag) that the program does not
// hold, and re-importing it would add links that were never there.
func ExportRows(in Input, out Output, origin map[string]string, view string) []ExportRow {
	collapsed := view == ViewCollapsed
	order := gridOrder(in.Activities, collapsed)
	rowOf := make(map[string]int, len(order))
	for n, i := range order {
		rowOf[in.Activities[i].ID] = n + 1
	}

	preds := map[string][]PredRef{}
	for _, l := range in.Links {
		p, okP := rowOf[l.Pred]
		_, okS := rowOf[l.Succ]
		if !okP || !okS {
			continue
		}
		ref := PredRef{Row: p, Type: strings.ToUpper(l.Type), Lag: l.Lag, LagUnit: l.LagUnit}
		if ref.Type == "" {
			ref.Type = LinkFS
		}
		if ref.LagUnit == "" {
			ref.LagUnit = LagWorking
		}
		preds[l.Succ] = append(preds[l.Succ], ref)
	}

	// Compute returns no times when the program has a loop; every row is
	// then unknown rather than guessed.
	timed := len(out.Activities) == len(in.Activities) && out.Cycle == nil
	cal := NewCalendar(in.Calendar)
	rows := make([]ExportRow, 0, len(order))
	for n, i := range order {
		a := in.Activities[i]
		kind := a.Kind
		if kind == "" {
			kind = KindActivity
		}
		r := ExportRow{ID: n + 1, Tier: tierWords[kind], Name: a.Label, Dated: !cal.Relative(), Origin: originWord(origin[a.ID])}
		if timed && !out.Activities[i].Unknown {
			t := out.Activities[i]
			d, tf := t.EF-t.ES, t.TF
			r.Duration, r.Float = &d, &tf
			r.Critical = "No"
			if t.Critical {
				r.Critical = "Yes"
			}
			if cal.Relative() {
				r.Start = cal.DayLabel(t.ES)
				if d > 0 {
					r.Finish = cal.DayLabel(t.EF - 1)
				}
			} else {
				r.Start, r.Finish = t.Start, t.Finish
			}
		}
		refs := preds[a.ID]
		sort.Slice(refs, func(x, y int) bool { return refs[x].Row < refs[y].Row })
		parts := make([]string, len(refs))
		for k, ref := range refs {
			parts[k] = FormatPredecessor(ref)
		}
		r.Predecessors = strings.Join(parts, ", ")
		rows = append(rows, r)
	}
	return rows
}

// gridOrder returns indices into acts in grid order; collapsed keeps the
// stages only.
func gridOrder(acts []Activity, collapsed bool) []int {
	var stages []int
	kids := map[string][]int{}
	for i, a := range acts {
		if a.ParentID == "" {
			stages = append(stages, i)
		} else if !collapsed {
			kids[a.ParentID] = append(kids[a.ParentID], i)
		}
	}
	byOrdinal := func(s []int) {
		sort.SliceStable(s, func(x, y int) bool {
			a, b := acts[s[x]], acts[s[y]]
			if a.Ordinal != b.Ordinal {
				return a.Ordinal < b.Ordinal
			}
			return a.ID < b.ID
		})
	}
	byOrdinal(stages)
	out := make([]int, 0, len(acts))
	for _, s := range stages {
		out = append(out, s)
		k := kids[acts[s].ID]
		byOrdinal(k)
		out = append(out, k...)
	}
	return out
}

func originWord(o string) string {
	if o == "" {
		return ""
	}
	return strings.ToUpper(o[:1]) + strings.ReplaceAll(o[1:], "_", " ")
}

// ExportSheet is the Excel sheet: a bold header row that repeats on each
// printed page, then one row per ExportRow. Numbers are numbers and dates
// are dates, so the workbook can be reworked rather than retyped.
func ExportSheet(rows []ExportRow) xlsx.Sheet {
	s := xlsx.Sheet{Name: "Program", Widths: []float64{6, 11, 44, 13, 13, 12, 22, 11, 9, 11}, TitleRow: 1}
	head := make([]xlsx.Cell, len(ExportHeader))
	for i, h := range ExportHeader {
		head[i] = xlsx.Cell{Text: h, Bold: true}
	}
	s.Rows = make([][]xlsx.Cell, 0, len(rows)+1)
	s.Rows = append(s.Rows, head)
	num := func(v *int) xlsx.Cell {
		if v == nil {
			return xlsx.Cell{}
		}
		f := float64(*v)
		return xlsx.Cell{Number: &f}
	}
	day := func(r ExportRow, v string) xlsx.Cell {
		if r.Dated {
			return xlsx.Cell{Date: v}
		}
		return xlsx.Cell{Text: v}
	}
	for _, r := range rows {
		id := float64(r.ID)
		s.Rows = append(s.Rows, []xlsx.Cell{{Number: &id}, {Text: r.Tier}, {Text: r.Name, Bold: r.Tier == "Stage"}, day(r, r.Start), day(r, r.Finish),
			num(r.Duration), {Text: r.Predecessors}, num(r.Float), {Text: r.Critical}, {Text: r.Origin}})
	}
	return s
}

// WriteExportCSV writes the header and rows. Dates stay YYYY-MM-DD; a name
// that a spreadsheet would read as a formula is prefixed with a quote.
func WriteExportCSV(w io.Writer, rows []ExportRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(ExportHeader); err != nil {
		return err
	}
	num := func(v *int) string {
		if v == nil {
			return ""
		}
		return strconv.Itoa(*v)
	}
	for _, r := range rows {
		if err := cw.Write([]string{strconv.Itoa(r.ID), r.Tier, csvSafe(r.Name), r.Start, r.Finish, num(r.Duration), r.Predecessors, num(r.Float), r.Critical, r.Origin}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe stops a spreadsheet from running a cell as a formula (CSV
// injection), as the invoice export does.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
