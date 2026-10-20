package programread

// Reading B of the Program plan (docs/plans/2026-10-18-program.md 10.3):
// the R7 Program returnable in a firm's proposal. SiteWise's own response
// form asks for Stage, Your weeks, Authority weeks and Inputs per package
// stage (internal/rfp/form.go, table r7). Code settles every row of an
// exact R7 table whose Stage cell is one of the package's stage labels and
// whose Your weeks cell is a plain number. For the stages still open, code
// harvests duration-like text near the stage label, and one background Jev
// fan-out picks which candidate the proposal states as the firm's own
// duration for that stage, or none. Code converts the chosen verbatim text
// to working days; Jev never converts, counts or compares.
//
// This is its own question map (StageWeeksQuestionVersion), never part of
// the proposal read's "proposal-1" map (plan 13.3 Q3).

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/program"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// StageWeeksQuestionVersion is the program_stage_weeks question map. Any
// change to its wording or option shape is a new version that needs a new
// calibration run (plan section 2, Jev rule 11).
const StageWeeksQuestionVersion = "proposal-program-1"

// StageWeeksFamily is the threshold family of the one question Reading B asks.
const StageWeeksFamily = "program_stage_weeks"

const (
	// nearChars is how far from a stage label a duration may sit to be a
	// candidate for that stage (plan 10.3 "Harvest for the rest").
	nearChars = 200
	// maxStageCandidates is the most candidates one stage offers.
	maxStageCandidates = 20
	// maxLabelHits bounds the label occurrences one stage contributes, so a
	// label repeated on every page cannot fill the excerpt alone.
	maxLabelHits = 12
	// maxScanBytes bounds the proposal text scanned for labels.
	maxScanBytes = 400_000
)

// Stage is one live package stage the R7 table answers. Label is the
// package stage's own text, used literally in the question.
type Stage struct {
	ID    string // package stage id; the question id
	Label string
	// TargetActivityID is the program's package activity for this package
	// stage; "" when the program has none.
	TargetActivityID string
	// CatalogueStageID is the program stage the package stage sits in, so
	// renaming or reordering never loses the evidence (plan 7.7).
	CatalogueStageID string
}

// StageWeeksInput is one proposal to read.
type StageWeeksInput struct {
	Filename string
	Text     identity.Text
	PerWeek  int
	Stages   []Stage
}

type candidate struct {
	ID    string // "c1", "c2", ...
	Value string // verbatim
}

type window struct{ from, to int }

// StageDraft is one proposal's R7 reading in progress.
type StageDraft struct {
	in      StageWeeksInput
	text    string // the proposal text, one printed line per line
	settled map[string]Reading
	cands   map[string][]candidate
	windows map[string][]window

	asked    map[string]jev.Question
	options  map[string]int
	notAsked []string
	grey     []string
}

// NewStageDraft settles R7 rows by rule and harvests candidates for the
// rest.
func NewStageDraft(in StageWeeksInput) *StageDraft {
	if in.PerWeek <= 0 {
		in.PerWeek = 5
	}
	d := &StageDraft{in: in, settled: map[string]Reading{}, cands: map[string][]candidate{}, windows: map[string][]window{}}
	d.rule()
	var b strings.Builder
	for _, l := range proposals.Lines(in.Text) {
		if b.Len()+len(l.Text)+1 > maxScanBytes {
			b.WriteString(clipBytes(l.Text, maxScanBytes-b.Len()))
			break
		}
		b.WriteString(l.Text)
		b.WriteByte('\n')
	}
	d.text = b.String()
	d.harvest()
	return d
}

// rule settles each row of an exact R7 table whose Stage cell is one of
// our stage labels and whose Your weeks cell is a plain number: code fast
// path, decided_by rule, no Jev (plan 10.3 "Rules first";
// https://docs.typesafe.ai/patterns/intent-routing). Only a DOCX or XLSX
// returnable carries the table as cells. A stage the table lists twice is
// left to the harvest.
func (d *StageDraft) rule() {
	if d.in.Text.Format != "docx" && d.in.Text.Format != "xlsx" {
		return
	}
	byLabel := map[string]Stage{}
	for _, s := range d.in.Stages {
		byLabel[cellText(s.Label)] = s
	}
	seen := map[string]int{}
	found := map[string]Reading{}
	for _, g := range cellGrids(d.in.Text) {
		for i, row := range g.rows {
			col, ok := r7Header(g.cells[row])
			if !ok {
				continue
			}
			for _, r := range g.rows[i+1:] {
				cells := g.cells[r]
				s, ok := byLabel[cellText(cells[col])]
				if !ok {
					continue
				}
				seen[s.ID]++
				weeks := cellText(cells[col+1])
				if !plainNumber.MatchString(weeks) {
					continue
				}
				dur, err := program.ParseDuration(weeks+"w", d.in.PerWeek)
				if err != nil {
					continue
				}
				found[s.ID] = d.reading(s, weeks, weeks+" weeks", dur, BandGreen, ByRule, nil, "")
			}
			break
		}
	}
	for id, r := range found {
		if seen[id] == 1 {
			d.settled[id] = r
		}
	}
}

var plainNumber = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

// r7Header reports the column where an exact R7 header starts in a row.
func r7Header(cells map[int]string) (int, bool) {
	for c := range cells {
		if cellText(cells[c]) != rfp.R7Columns[0] {
			continue
		}
		match := true
		for k, name := range rfp.R7Columns[1:] {
			if cellText(cells[c+k+1]) != name {
				match = false
				break
			}
		}
		if match {
			return c, true
		}
	}
	return 0, false
}

// clipBytes cuts s to at most n bytes on a character boundary.
func clipBytes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func cellText(s string) string { return strings.Join(strings.Fields(s), " ") }

type grid struct {
	rows  []int
	cells map[int]map[int]string
}

// cellGrids groups a workbook's or Word file's runs into one grid per
// sheet or table, rows in order.
func cellGrids(text identity.Text) []grid {
	type key struct {
		sheet string
		table int
	}
	var order []key
	grids := map[key]*grid{}
	for _, r := range text.Runs {
		src := r.Source
		if src.Row <= 0 || src.Col <= 0 {
			continue
		}
		k := key{src.Sheet, src.Table}
		g := grids[k]
		if g == nil {
			g = &grid{cells: map[int]map[int]string{}}
			grids[k] = g
			order = append(order, k)
		}
		if g.cells[src.Row] == nil {
			g.cells[src.Row] = map[int]string{}
			g.rows = append(g.rows, src.Row)
		}
		t := strings.TrimSpace(r.Text)
		if prev := g.cells[src.Row][src.Col]; prev != "" {
			t = prev + " " + t
		}
		g.cells[src.Row][src.Col] = t
	}
	out := make([]grid, 0, len(order))
	for _, k := range order {
		g := grids[k]
		sort.Ints(g.rows)
		out = append(out, *g)
	}
	return out
}

// durationToken is duration-like text: "6 weeks", "30 working days",
// "4–6 wks", "6-week". Months are not working days and are not collected.
var durationToken = regexp.MustCompile(`(?i)\b[0-9]+(?:\.[0-9]+)?(?:\s*(?:–|—|-|to)\s*[0-9]+(?:\.[0-9]+)?)?(?:\s*|-)(?:working\s+days?|business\s+days?|work\s+days?|days?|d|wks?|weeks?)\b`)

// harvest collects, for each open stage, the duration-like text within 200
// characters of each place the proposal names the stage, verbatim, at most
// 20 per stage (plan 10.3).
func (d *StageDraft) harvest() {
	tokens := durationToken.FindAllStringIndex(d.text, -1)
	for _, s := range d.in.Stages {
		if _, done := d.settled[s.ID]; done {
			continue
		}
		re := labelPattern(s.Label)
		if re == nil {
			continue
		}
		seen := map[string]bool{}
		var list []candidate
		for _, hit := range re.FindAllStringIndex(d.text, maxLabelHits) {
			w := window{back(d.text, hit[0], nearChars), fwd(d.text, hit[1], nearChars)}
			d.windows[s.ID] = append(d.windows[s.ID], w)
			for _, t := range tokens {
				if t[0] < w.from || t[1] > w.to || len(list) >= maxStageCandidates {
					continue
				}
				v := d.text[t[0]:t[1]]
				if seen[v] {
					continue
				}
				seen[v] = true
				list = append(list, candidate{ID: "c" + strconv.Itoa(len(list)+1), Value: v})
			}
		}
		if len(list) > 0 {
			d.cands[s.ID] = list
		}
	}
}

// labelPattern finds the stage label as words, any case, any spacing.
func labelPattern(label string) *regexp.Regexp {
	words := strings.Fields(label)
	if len(words) == 0 {
		return nil
	}
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)` + strings.Join(words, `\s+`))
}

// back and fwd move n characters (not bytes) from i.
func back(s string, i, n int) int {
	for ; n > 0 && i > 0; n-- {
		_, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
	}
	return i
}

func fwd(s string, i, n int) int {
	for ; n > 0 && i < len(s); n-- {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return i
}

// ParseStated converts a stated duration to working days in code: weeks
// are the program's working weeks, a range gives low and high. ok is
// false for text that is not a whole number of working days.
func ParseStated(value string, perWeek int) (program.Duration, bool) {
	t := strings.ToLower(cellText(value))
	t = statedDays.ReplaceAllString(t, "d")
	t = hyphenUnit.ReplaceAllString(t, "$1 $2")
	dur, err := program.ParseDuration(t, perWeek)
	return dur, err == nil
}

var (
	statedDays = regexp.MustCompile(`(?:working|business|work)\s+days?`)
	hyphenUnit = regexp.MustCompile(`([0-9])-(d\b|days?|wks?|weeks?)`)
)

func (d *StageDraft) reading(s Stage, value, stated string, dur program.Duration, band, by string, conf *float64, version string) Reading {
	return Reading{TargetID: s.TargetActivityID, CatalogueStageID: s.CatalogueStageID, Field: FieldDuration, Value: value,
		Parsed: map[string]any{"low": dur.Low, "high": dur.High, "stated": stated, "package_stage_id": s.ID, "stage_label": s.Label},
		Band:   band, DecidedBy: by, Confidence: conf, QuestionVersion: version}
}

// stageInstructions is the program_stage_weeks question for one stage. The
// stage label is the package's literal text, never generated. Jev is asked
// which candidate the proposal states, never to convert or compare it
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13 §2–3).
func stageInstructions(label string) string {
	return "Using `excerpt`, which candidate does the proposal state as the firm's own duration for the stage \"" + label + "\"? A duration stated for authority review, for a different stage, or for the whole commission is not it. Choose none when the proposal states no duration for this stage."
}

const stageNone = "The proposal states no duration of its own for this stage."

type valueCriterion struct {
	Value string `json:"value"`
}

type stageState struct {
	Excerpt string `json:"excerpt"`
}

// Call is the one fan-out for every open stage with candidates, or false
// when nothing is open. It shrinks until jev.CheckRequestSize passes: the
// excerpt 12,000 → 6,000 → 3,000 characters with 20 → 10 → 5 options per
// stage. `none` is never dropped (plan section 2, Jev rules 9 and 12).
//
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
func (d *StageDraft) Call() (jev.Call, bool) {
	var open []Stage
	for _, s := range d.in.Stages {
		if _, ok := d.cands[s.ID]; ok {
			open = append(open, s)
		}
	}
	if len(open) == 0 {
		return jev.Call{}, false
	}
	call := jev.Call{QuestionVersion: StageWeeksQuestionVersion, Priority: jev.PriorityBackground, Deadline: 6 * time.Second}
	for _, step := range []struct{ excerpt, k int }{{12000, 20}, {6000, 10}, {3000, 5}} {
		qs := map[string]jev.Question{}
		options := map[string]int{}
		var ws []window
		for _, s := range open {
			list := d.cands[s.ID]
			if len(list) > step.k {
				list = list[:step.k]
			}
			criteria := make(map[string]any, len(list)+1)
			for _, c := range list {
				criteria[c.ID] = valueCriterion{Value: c.Value}
			}
			criteria[choiceNone] = stageNone
			qs[s.ID] = jev.Question{Type: jev.TypeChoice, Instructions: stageInstructions(s.Label), Criteria: criteria}
			options[s.ID] = len(list) + 1
			ws = append(ws, d.windows[s.ID]...)
		}
		call.Questions, call.State = qs, stageState{Excerpt: excerptOf(d.text, ws, step.excerpt)}
		d.asked, d.options, d.notAsked = qs, options, nil
		if fits(call) {
			return call, true
		}
	}
	d.asked, d.options, d.notAsked = map[string]jev.Question{}, map[string]int{}, nil
	for _, s := range open {
		d.notAsked = append(d.notAsked, s.ID)
	}
	return jev.Call{}, false
}

// excerptOf is the text around the stage labels only, in document order,
// overlapping windows merged, at most max bytes (jev-1.13 §5).
func excerptOf(text string, ws []window, max int) string {
	sort.Slice(ws, func(i, j int) bool { return ws[i].from < ws[j].from })
	var merged []window
	for _, w := range ws {
		if n := len(merged); n > 0 && w.from <= merged[n-1].to {
			if w.to > merged[n-1].to {
				merged[n-1].to = w.to
			}
			continue
		}
		merged = append(merged, w)
	}
	var b strings.Builder
	for _, w := range merged {
		part := text[w.from:w.to]
		if b.Len() > 0 {
			part = "\n…\n" + part
		}
		if b.Len()+len(part) > max {
			break
		}
		b.WriteString(part)
	}
	return b.String()
}

// Apply records Jev's answers. A failed call leaves every asked stage grey
// for a background retry; an answer with no calibrated cut-off for its
// family, version and option count stays blank and writes nothing; none
// writes nothing.
//
//	https://docs.typesafe.ai/confidence
func (d *StageDraft) Apply(res jev.Result, err error, th Thresholds) (grey bool) {
	byID := map[string]Stage{}
	for _, s := range d.in.Stages {
		byID[s.ID] = s
	}
	ids := make([]string, 0, len(d.asked))
	for id := range d.asked {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err != nil {
			d.grey = append(d.grey, id)
			grey = true
			continue
		}
		a, ok := res.Answers[id]
		if !ok || a.Type != jev.TypeChoice || a.Choice == choiceNone {
			continue
		}
		value, ok := d.candidate(id, a.Choice)
		if !ok {
			continue
		}
		band, apply := th.Band(StageWeeksFamily, a.Confidence, d.options[id])
		if !apply {
			continue
		}
		dur, ok := ParseStated(value, d.in.PerWeek)
		if !ok {
			continue
		}
		d.settled[id] = d.reading(byID[id], value, value, dur, band, ByJev, a.Confidence, StageWeeksQuestionVersion)
	}
	return grey
}

func (d *StageDraft) candidate(stage, id string) (string, bool) {
	q, ok := d.asked[stage]
	if !ok {
		return "", false
	}
	c, ok := q.Criteria.(map[string]any)[id]
	if !ok {
		return "", false
	}
	v, ok := c.(valueCriterion)
	return v.Value, ok
}

// StageOutcome is one finished R7 reading.
type StageOutcome struct {
	// Readings are one duration per settled stage, in stage order.
	Readings []Reading
	// Grey are stages whose question failed; a background retry asks again.
	Grey []string
	// NotAsked are open stages left out because no request fits.
	NotAsked []string
	Asked    bool
}

// Finish returns the settled stages as readings.
func (d *StageDraft) Finish() StageOutcome {
	out := StageOutcome{Readings: []Reading{}, Grey: append([]string{}, d.grey...), NotAsked: append([]string{}, d.notAsked...)}
	for _, s := range d.in.Stages {
		if r, ok := d.settled[s.ID]; ok {
			out.Readings = append(out.Readings, r)
		}
	}
	return out
}

// Asked are the stage ids in the last Call, for the eval.
func (d *StageDraft) Asked() []string {
	out := make([]string, 0, len(d.asked))
	for id := range d.asked {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Options is an asked stage's option count, none included.
func (d *StageDraft) Options(stage string) int { return d.options[stage] }

// Candidates are an open stage's harvested values, verbatim, for the eval.
func (d *StageDraft) Candidates(stage string) []string {
	var out []string
	for _, c := range d.cands[stage] {
		out = append(out, c.Value)
	}
	return out
}

// Raw is Jev's own answer for an asked stage before any threshold: the
// verbatim value ("" for none), its confidence, and whether it answered.
func (d *StageDraft) Raw(res jev.Result, stage string) (string, *float64, bool) {
	if _, ok := d.asked[stage]; !ok {
		return "", nil, false
	}
	a, ok := res.Answers[stage]
	if !ok || a.Type != jev.TypeChoice {
		return "", nil, false
	}
	if a.Choice == choiceNone {
		return "", a.Confidence, true
	}
	v, ok := d.candidate(stage, a.Choice)
	return v, a.Confidence, ok
}

// ReadStageWeeks reads one proposal's R7 returnable: rules, at most one
// background Jev fan-out, then code.
func ReadStageWeeks(ctx context.Context, ask Asker, th Thresholds, in StageWeeksInput) (StageOutcome, error) {
	d := NewStageDraft(in)
	asked := false
	if call, ok := d.Call(); ok {
		asked = true
		res, err := ask.Ask(ctx, call)
		if err != nil && ctx.Err() != nil {
			return StageOutcome{}, ctx.Err()
		}
		d.Apply(res, err, th)
	}
	out := d.Finish()
	out.Asked = asked
	return out, nil
}
