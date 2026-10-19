// Package programread is Reading A of the Program plan
// (docs/plans/2026-10-18-program.md section 10.2): a program file dropped on
// Program is harvested by code into rows, code settles every target whose
// label matches a row exactly, and one background Jev fan-out picks, for
// each target still open, which shortlisted row is the same work, or none.
// Jev never sees a date, a duration or a number: code parses those, and a
// match only proposes the row's verbatim values as readings the user
// accepts. The structure copies internal/proposals/read.go.
package programread

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/program"
	"sitewise/internal/proposals"
)

// ExtractLimits reads far more of a program file than filing's identity
// region: every sheet row up to a large program, and every PDF page a
// printed Gantt usually runs to.
func ExtractLimits() identity.Limits {
	l := identity.DefaultLimits()
	l.MaxPages = 60
	l.ReadControlPage = false
	l.MaxRuns = 120000
	l.MaxSheets = 6
	l.MaxRows = 6000
	l.MaxCols = 40
	return l
}

// MaxRows bounds the rows one file contributes; a program larger than the
// program's own row limit cannot be matched usefully anyway.
const MaxRows = 5000

// Row is one row of the file. Every *Raw field is the cell copied verbatim;
// the parsed fields are code's reading of it, empty or nil when the value
// does not parse (it then stays raw and unparsed).
type Row struct {
	ID              string `json:"id"` // "r<n>", 1-based in file order
	FileID          string `json:"file_id,omitempty"`
	Label           string `json:"label"`
	StartRaw        string `json:"start_raw,omitempty"`
	FinishRaw       string `json:"finish_raw,omitempty"`
	DurationRaw     string `json:"duration_raw,omitempty"`
	PredecessorsRaw string `json:"predecessors_raw,omitempty"`
	Level           int    `json:"level,omitempty"` // outline level when the file states one

	Start        string            `json:"start,omitempty"`  // YYYY-MM-DD
	Finish       string            `json:"finish,omitempty"` // YYYY-MM-DD
	Duration     *program.Duration `json:"duration,omitempty"`
	Predecessors []program.PredRef `json:"-"`
	// PredLinks are the parsed predecessors that name another row of this
	// file, by row id.
	PredLinks []PredecessorLink `json:"pred_links,omitempty"`
}

// PredecessorLink is one parsed predecessor resolved to a row of this file.
type PredecessorLink struct {
	Row     string `json:"row"`
	Type    string `json:"type"`
	Lag     int    `json:"lag"`
	LagUnit string `json:"lag_unit"`
}

// Harvest is everything code read from one file.
type Harvest struct {
	Filename string   `json:"filename"`
	Rows     []Row    `json:"rows"`
	Checks   []string `json:"checks"`
}

// Header keywords (plan 10.2 harvest step 1).
type column int

const (
	colNone column = iota
	colID
	colLevel
	colLabel
	colStart
	colFinish
	colDuration
	colPred
)

// headerColumn classifies one header cell. Order matters: "Task ID" is an
// id, "Baseline Start" is not this program's start, and "Task Name" is
// the label.
func headerColumn(cell string) column {
	h := strings.ToLower(strings.Join(strings.Fields(cell), " "))
	h = strings.Trim(h, " :")
	switch {
	case h == "":
		return colNone
	case strings.Contains(h, "baseline") || strings.Contains(h, "actual") || strings.Contains(h, "early") || strings.Contains(h, "late"):
		return colNone
	case strings.Contains(h, "predecessor"):
		return colPred
	case strings.Contains(h, "successor"):
		return colNone
	case strings.Contains(h, "duration") || h == "dur" || h == "dur." || h == "days":
		return colDuration
	case strings.Contains(h, "start") || h == "from" || h == "begin":
		return colStart
	case strings.Contains(h, "finish") || h == "end" || strings.HasPrefix(h, "end ") || h == "to" || h == "completion" || h == "complete by":
		return colFinish
	case h == "id" || h == "#" || h == "no" || h == "no." || h == "task id" || h == "unique id" || h == "ref" || h == "item" || h == "line":
		return colID
	case strings.Contains(h, "outline") || h == "wbs" || h == "level":
		return colLevel
	case strings.Contains(h, "task") || strings.Contains(h, "activity") || strings.Contains(h, "name") || strings.Contains(h, "description"):
		return colLabel
	}
	return colNone
}

// headerOf maps a header row's columns, or ok false when the row is not a
// program header: it needs a label column and a start, finish or duration.
func headerOf(cells map[int]string) (map[column]int, map[column]string, bool) {
	cols, names := map[column]int{}, map[column]string{}
	keys := make([]int, 0, len(cells))
	for c := range cells {
		keys = append(keys, c)
	}
	sort.Ints(keys)
	for _, c := range keys {
		k := headerColumn(cells[c])
		if k == colNone {
			continue
		}
		if _, taken := cols[k]; taken {
			continue
		}
		cols[k], names[k] = c, strings.TrimSpace(cells[c])
	}
	_, label := cols[colLabel]
	_, start := cols[colStart]
	_, finish := cols[colFinish]
	_, dur := cols[colDuration]
	return cols, names, label && (start || finish || dur)
}

// rawRow is one harvested row before parsing: verbatim cells, and the
// header names of its date columns for the day-order check.
type rawRow struct {
	group                                          string
	fileID, label, start, finish, dur, pred, level string
}

// HarvestText reads one file's text into rows (plan 10.2 harvest steps 1–4).
// perWeek is the program's working days per week, for durations and lags
// written in weeks.
func HarvestText(filename string, text identity.Text, perWeek int) Harvest {
	h := Harvest{Filename: filename, Rows: []Row{}, Checks: []string{}}
	var raws []rawRow
	startNames, finishNames := map[string]string{}, map[string]string{}
	if text.Format == "pdf" || hasGeometry(text) {
		raws = pdfRows(text)
		startNames[""], finishNames[""] = "Start", "Finish"
	} else {
		raws = tableRows(text, startNames, finishNames)
	}
	if len(raws) > MaxRows {
		raws = raws[:MaxRows]
	}
	// Day-month order is decided per column (one group's Start column, one
	// group's Finish column), never per value.
	type colKey struct{ group, which string }
	values := map[colKey][]string{}
	for _, r := range raws {
		values[colKey{r.group, "start"}] = append(values[colKey{r.group, "start"}], r.start)
		values[colKey{r.group, "finish"}] = append(values[colKey{r.group, "finish"}], r.finish)
	}
	orders := map[colKey]program.DayOrder{}
	var groups []colKey
	for k := range values {
		groups = append(groups, k)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].group != groups[j].group {
			return groups[i].group < groups[j].group
		}
		return groups[i].which > groups[j].which // start before finish
	})
	seenCheck := map[string]bool{}
	for _, k := range groups {
		order, assumed := program.DecideDayOrder(values[k])
		orders[k] = order
		if !assumed {
			continue
		}
		name := startNames[k.group]
		if k.which == "finish" {
			name = finishNames[k.group]
		}
		msg := "Date order assumed day/month in column " + name
		if !seenCheck[msg] {
			seenCheck[msg] = true
			h.Checks = append(h.Checks, msg)
		}
	}
	byFileID := map[string]string{}
	for i, r := range raws {
		row := Row{ID: "r" + strconv.Itoa(i+1), FileID: r.fileID, Label: r.label, StartRaw: r.start, FinishRaw: r.finish, DurationRaw: r.dur, PredecessorsRaw: r.pred}
		row.Level = levelOf(r.level)
		row.Start = parseDate(r.start, orders[colKey{r.group, "start"}])
		row.Finish = parseDate(r.finish, orders[colKey{r.group, "finish"}])
		if d, err := program.ParseDuration(cleanDuration(r.dur), perWeek); err == nil && r.dur != "" {
			row.Duration = &d
		}
		if refs, err := program.ParsePredecessors(r.pred, perWeek); err == nil {
			row.Predecessors = refs
		}
		id := r.fileID
		if id == "" {
			id = strconv.Itoa(i + 1)
		}
		if _, dup := byFileID[id]; !dup {
			byFileID[id] = row.ID
		}
		h.Rows = append(h.Rows, row)
	}
	// Predecessors name the file's own task numbers; a file without an ID
	// column numbers tasks by row, as program tools do.
	for i := range h.Rows {
		for _, p := range h.Rows[i].Predecessors {
			if to, ok := byFileID[strconv.Itoa(p.Row)]; ok && to != h.Rows[i].ID {
				h.Rows[i].PredLinks = append(h.Rows[i].PredLinks, PredecessorLink{Row: to, Type: p.Type, Lag: p.Lag, LagUnit: p.LagUnit})
			}
		}
	}
	return h
}

func hasGeometry(text identity.Text) bool {
	for _, r := range text.Runs {
		if r.Source.Width != 0 || r.Source.X != 0 || r.Source.Y != 0 {
			return true
		}
	}
	return false
}

// tableRows reads workbook sheets and Word tables: each sheet or table is a
// group; its header row is found by keywords and every row below it with a
// label is a program row.
func tableRows(text identity.Text, startNames, finishNames map[string]string) []rawRow {
	type gkey struct {
		sheet string
		table int
	}
	var order []gkey
	rows := map[gkey]map[int]map[int]string{}
	for _, r := range text.Runs {
		src := r.Source
		if src.Row <= 0 || src.Col <= 0 {
			continue
		}
		k := gkey{src.Sheet, src.Table}
		if rows[k] == nil {
			rows[k] = map[int]map[int]string{}
			order = append(order, k)
		}
		if rows[k][src.Row] == nil {
			rows[k][src.Row] = map[int]string{}
		}
		t := strings.TrimSpace(r.Text)
		if prev := rows[k][src.Row][src.Col]; prev != "" {
			t = prev + " " + t
		}
		rows[k][src.Row][src.Col] = t
	}
	var out []rawRow
	for _, k := range order {
		group := k.sheet + "#" + strconv.Itoa(k.table)
		nums := make([]int, 0, len(rows[k]))
		for n := range rows[k] {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		var cols map[column]int
		header := -1
		for i, n := range nums {
			if i > 40 {
				break
			}
			if c, names, ok := headerOf(rows[k][n]); ok {
				cols, header = c, i
				startNames[group], finishNames[group] = nameOr(names[colStart], "Start"), nameOr(names[colFinish], "Finish")
				break
			}
		}
		if header < 0 {
			continue
		}
		cell := func(row map[int]string, c column) string {
			idx, ok := cols[c]
			if !ok {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		for _, n := range nums[header+1:] {
			row := rows[k][n]
			label := cell(row, colLabel)
			if label == "" {
				continue
			}
			out = append(out, rawRow{group: group, fileID: cell(row, colID), label: label,
				start: serialDate(cell(row, colStart)), finish: serialDate(cell(row, colFinish)),
				dur: cell(row, colDuration), pred: cell(row, colPred), level: cell(row, colLevel)})
		}
	}
	return out
}

func nameOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// dateToken finds a date-like token on a printed line: d/m/yyyy, d/m/yy,
// yyyy-mm-dd, or d Mon yyyy, with an optional weekday before it.
var dateToken = regexp.MustCompile(`(?i)(?:\b(?:mon|tue|tues|wed|thu|thur|thurs|fri|sat|sun)[a-z]*\.?,?\s+)?(?:\b\d{4}-\d{1,2}-\d{1,2}\b|\b\d{1,2}[/.-]\d{1,2}[/.-](?:\d{4}|\d{2})\b|\b\d{1,2}[ -](?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?,?[ -](?:\d{4}|\d{2})\b)`)

var (
	leadingID   = regexp.MustCompile(`^\s*(\d{1,5})\s+`)
	trailingDur = regexp.MustCompile(`(?i)\s+(\d+(?:\.\d+)?\s*(?:d|dy|dys|days?|w|wks?|weeks?|edays?)\??)\s*$`)
	headingOnly = regexp.MustCompile(`(?i)^(id|task name|task|name|activity|duration|start|finish|predecessors?|\s)+$`)
)

// pdfRows reads a printed program: every line with at least one date-like
// token is a row. The label is the words before the first date, less a
// leading task number and a trailing duration; the predecessor column is
// what follows the last date when it parses as one.
func pdfRows(text identity.Text) []rawRow {
	var out []rawRow
	for _, line := range proposals.Lines(text) {
		t := strings.Join(strings.Fields(line.Text), " ")
		locs := dateToken.FindAllStringIndex(t, -1)
		if len(locs) == 0 {
			continue
		}
		before := t[:locs[0][0]]
		r := rawRow{start: strings.TrimSpace(t[locs[0][0]:locs[0][1]])}
		last := locs[0]
		if len(locs) > 1 {
			r.finish = strings.TrimSpace(t[locs[1][0]:locs[1][1]])
			last = locs[1]
		}
		if m := leadingID.FindStringSubmatch(before); m != nil {
			r.fileID = m[1]
			before = before[len(m[0]):]
		}
		if m := trailingDur.FindStringSubmatchIndex(before); m != nil {
			r.dur = strings.TrimSpace(before[m[2]:m[3]])
			before = before[:m[0]]
		}
		r.label = strings.TrimSpace(before)
		if r.label == "" || headingOnly.MatchString(r.label) {
			continue
		}
		if after := strings.TrimSpace(t[last[1]:]); after != "" {
			if refs, err := program.ParsePredecessors(after, 5); err == nil && len(refs) > 0 {
				r.pred = after
			}
		}
		out = append(out, r)
	}
	return out
}

// serialDate turns a workbook date serial ("46447", "46447.333") into
// yyyy-mm-dd: a sheet stores dates as day numbers from 1899-12-30 and only
// its display format, which the text layer does not keep, makes them dates.
// Anything else is returned as it was.
func serialDate(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 20000 || v > 80000 {
		return s
	}
	day := int(math.Floor(v))
	return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, day).Format("2006-01-02")
}

func parseDate(raw string, order program.DayOrder) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	d, err := program.ParseDate(raw, order)
	if err != nil {
		return ""
	}
	return d
}

// cleanDuration drops the "?" a program tool prints on an estimated
// duration ("5 days?"); the number and unit are as written.
func cleanDuration(s string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "?"))
}

// levelOf reads an outline level ("2") or a WBS code ("1.2.3" is level 3).
func levelOf(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < 50 {
		return n
	}
	parts := strings.Split(strings.TrimSuffix(s, "."), ".")
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return 0
		}
	}
	return len(parts)
}
