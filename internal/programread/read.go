package programread

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"sitewise/internal/jev"
)

// Bands and deciders, as in filing and the proposal read.
const (
	BandGreen = "green"
	BandAmber = "amber"
	BandBlank = "blank"
	BandGrey  = "grey"

	ByRule = "rule"
	ByJev  = "jev"
)

// Reading fields written for a matched row (plan 10.2 "Apply").
const (
	FieldStart    = "start"
	FieldFinish   = "finish"
	FieldDuration = "duration"
)

// Target tiers, in the order plan 10.2 asks them and drops them when a
// request is too large: stages, then package activities, milestones and
// hold points, then fragment activities.
const (
	TierStage    = 0
	TierPackage  = 1
	TierFragment = 2
)

const (
	choiceNone = "none"
	// shortlistSize is the most rows offered for one target (plan 10.2
	// "Shortlist"); the request shrinks it to 20 then 10 when too large.
	shortlistSize = 40
	// maxRequestBytes keeps one program call well inside the provider's
	// limit, as proposals.maxRequestBytes does; labels are words, so the
	// generic preflight is the binding check and this only caps a huge file.
	maxRequestBytes = 120_000
)

// Target is one live program row a file row may match. Label and
// StageLabel are the program's own text, used literally in the question.
type Target struct {
	ID         string
	Label      string
	Kind       string // stage, activity, milestone or hold_point
	StageLabel string // the parent stage's label; "" for a stage
	// CatalogueStageID is set on a stage: evidence for a stage is stored
	// with it so renaming or reordering never loses it (plan 7.7).
	CatalogueStageID string
	Tier             int
}

// Match is one target settled to one file row.
type Match struct {
	TargetID        string
	RowID           string
	Band            string
	DecidedBy       string
	Confidence      *float64
	QuestionVersion string
}

// Reading is one proposed value for a target, ready to store as a
// program_readings row. Value is the file's text verbatim; Parsed is code's
// reading of it.
type Reading struct {
	TargetID         string         `json:"target_activity_id"`
	CatalogueStageID string         `json:"catalogue_stage_id,omitempty"`
	RowID            string         `json:"row"`
	Field            string         `json:"field"`
	Value            string         `json:"value"`
	Parsed           map[string]any `json:"parsed"`
	Band             string         `json:"band"`
	DecidedBy        string         `json:"decided_by"`
	Confidence       *float64       `json:"confidence,omitempty"`
	QuestionVersion  string         `json:"question_version,omitempty"`
}

// Asker is the one System One call a program read may make.
type Asker interface {
	Ask(context.Context, jev.Call) (jev.Result, error)
}

type scored struct {
	row   string
	score float64
}

// Draft is one program file reading in progress.
type Draft struct {
	h       Harvest
	targets []Target
	rows    map[string]*Row
	norm    map[string]string // row id → normalised label

	settled   map[string]Match    // target id → match
	used      map[string]bool     // row ids settled by rule
	shortlist map[string][]scored // open target id → ranked rows

	// Set by Call: the questions sent, their option counts, and the open
	// targets left out of the request (they show "Not matched").
	asked    map[string]jev.Question
	options  map[string]int
	notAsked []string
	grey     []string
}

// NewDraft settles what rules can and shortlists the rest. targets are
// the program's live rows in program order.
func NewDraft(h Harvest, targets []Target) *Draft {
	d := &Draft{h: h, rows: map[string]*Row{}, norm: map[string]string{}, settled: map[string]Match{}, used: map[string]bool{}, shortlist: map[string][]scored{}}
	d.targets = append([]Target(nil), targets...)
	sort.SliceStable(d.targets, func(i, j int) bool { return d.targets[i].Tier < d.targets[j].Tier })
	for i := range d.h.Rows {
		r := &d.h.Rows[i]
		d.rows[r.ID] = r
		d.norm[r.ID] = Normalise(r.Label)
	}
	d.rule()
	d.shortlistAll()
	return d
}

// rule settles a target whose normalised label equals exactly one file
// row's, when that row equals no other target: code fast path, no Jev
// (plan 10.2 "Rules first"; https://docs.typesafe.ai/patterns/intent-routing).
// Each file row settles at most one target.
func (d *Draft) rule() {
	rowsByNorm := map[string][]string{}
	for _, r := range d.h.Rows {
		if n := d.norm[r.ID]; n != "" {
			rowsByNorm[n] = append(rowsByNorm[n], r.ID)
		}
	}
	targetsByNorm := map[string]int{}
	for _, t := range d.targets {
		targetsByNorm[Normalise(t.Label)]++
	}
	for _, t := range d.targets {
		n := Normalise(t.Label)
		if n == "" || targetsByNorm[n] != 1 || len(rowsByNorm[n]) != 1 {
			continue
		}
		row := rowsByNorm[n][0]
		if d.used[row] {
			continue
		}
		d.used[row] = true
		d.settled[t.ID] = Match{TargetID: t.ID, RowID: row, Band: BandGreen, DecidedBy: ByRule}
	}
}

// shortlistAll ranks unsettled rows for each unsettled target by trigram
// similarity of normalised labels, keeping the top 40 above zero (plan 10.2
// "Shortlist"). A target with an empty shortlist gets no question.
func (d *Draft) shortlistAll() {
	grams := map[string]map[string]bool{}
	for _, r := range d.h.Rows {
		if !d.used[r.ID] && d.norm[r.ID] != "" {
			grams[r.ID] = trigrams(d.norm[r.ID])
		}
	}
	for _, t := range d.targets {
		if _, done := d.settled[t.ID]; done {
			continue
		}
		tg := trigrams(Normalise(t.Label))
		var list []scored
		for _, r := range d.h.Rows {
			g, ok := grams[r.ID]
			if !ok {
				continue
			}
			if s := jaccard(tg, g); s > 0 {
				list = append(list, scored{r.ID, s})
			}
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
		if len(list) > shortlistSize {
			list = list[:shortlistSize]
		}
		if len(list) > 0 {
			d.shortlist[t.ID] = list
		}
	}
}

// boundary is the guidance every program_row question carries (plan 10.2).
const boundary = "A heading or summary row that only groups other rows is not the same work. A row for a different trade or a different stage of the same trade is not the same work."

// instructions is the program_row question for one target. The target and
// stage labels are the program's literal text, never generated. Jev is
// asked only which row is the same work: never about a date or a duration
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13 §2–3).
func instructions(t Target) string {
	if t.Kind == "stage" {
		return "Using `rows`, which row of this program is the same work as the stage \"" + t.Label + "\"? Choose none when no row is that work. " + boundary
	}
	return "Using `rows`, which row of this program is the same work as the activity \"" + t.Label + "\" (stage: " + t.StageLabel + ")? Choose none when no row is that work. " + boundary
}

func noneText(t Target) string {
	if t.Kind == "stage" {
		return "No row in this program is the same work as this stage."
	}
	return "No row in this program is the same work as this activity."
}

type rowCriterion struct {
	Row string `json:"row"`
}

type stateRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Level int    `json:"level,omitempty"`
}

type state struct {
	Filename string     `json:"filename"`
	Rows     []stateRow `json:"rows"`
}

// question is one target's choice over its top k shortlisted rows plus
// none. Criteria show the verbatim row label only.
func (d *Draft) question(t Target, k int) (jev.Question, []string) {
	list := d.shortlist[t.ID]
	if len(list) > k {
		list = list[:k]
	}
	criteria := make(map[string]any, len(list)+1)
	ids := make([]string, 0, len(list))
	for _, s := range list {
		criteria[s.row] = rowCriterion{Row: d.rows[s.row].Label}
		ids = append(ids, s.row)
	}
	criteria[choiceNone] = noneText(t)
	return jev.Question{Type: jev.TypeChoice, Instructions: instructions(t), Criteria: criteria}, ids
}

// Call is the one fan-out for every open target, or false when nothing is
// open. It shrinks until jev.CheckRequestSize passes: 40 → 20 → 10 options
// per target, then fragment activity targets are dropped, then package
// activity targets. `none` is never dropped. Targets left out are recorded
// and show "Not matched" (plan 10.2 "Call").
//
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
func (d *Draft) Call() (jev.Call, bool) {
	var open []Target
	for _, t := range d.targets {
		if _, ok := d.shortlist[t.ID]; ok {
			open = append(open, t)
		}
	}
	if len(open) == 0 {
		return jev.Call{}, false
	}
	call := jev.Call{QuestionVersion: QuestionVersion, Priority: jev.PriorityBackground, Deadline: 6 * time.Second}
	steps := []struct{ k, maxTier int }{{40, TierFragment}, {20, TierFragment}, {10, TierFragment}, {10, TierPackage}, {10, TierStage}}
	for i, step := range steps {
		qs := map[string]jev.Question{}
		options := map[string]int{}
		inState := map[string]bool{}
		var left []string
		for _, t := range open {
			if t.Tier > step.maxTier {
				left = append(left, t.ID)
				continue
			}
			q, ids := d.question(t, step.k)
			qs[t.ID] = q
			options[t.ID] = len(ids) + 1
			for _, id := range ids {
				inState[id] = true
			}
		}
		st := state{Filename: d.h.Filename, Rows: []stateRow{}}
		for _, r := range d.h.Rows {
			if inState[r.ID] {
				st.Rows = append(st.Rows, stateRow{ID: r.ID, Label: r.Label, Level: r.Level})
			}
		}
		call.Questions, call.State = qs, st
		d.asked, d.options, d.notAsked = qs, options, left
		if len(qs) > 0 && fits(call) {
			return call, true
		}
		if i == len(steps)-1 {
			break
		}
	}
	// Nothing fits even with stages alone: no target is asked, and every
	// one shows "Not matched" rather than a guess.
	d.asked, d.options, d.notAsked = map[string]jev.Question{}, map[string]int{}, nil
	for _, t := range open {
		d.notAsked = append(d.notAsked, t.ID)
	}
	return jev.Call{}, false
}

func fits(call jev.Call) bool {
	s, err := jev.CheckRequestSize(call)
	return err == nil && s.Bytes <= maxRequestBytes
}

// Apply records Jev's answers. A failed call leaves every asked target
// grey for a background retry; an answer below every calibrated cut-off,
// or with no cut-off for its option count, stays blank and writes nothing.
//
//	https://docs.typesafe.ai/confidence
func (d *Draft) Apply(res jev.Result, err error, th Thresholds) (grey bool) {
	ids := make([]string, 0, len(d.asked))
	for id := range d.asked {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	picked := map[string][]string{}
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
		if _, isRow := d.rows[a.Choice]; !isRow || d.used[a.Choice] {
			continue
		}
		band, apply := th.Band(Family, a.Confidence, d.options[id])
		if !apply {
			continue
		}
		d.settled[id] = Match{TargetID: id, RowID: a.Choice, Band: band, DecidedBy: ByJev, Confidence: a.Confidence, QuestionVersion: QuestionVersion}
		picked[a.Choice] = append(picked[a.Choice], id)
	}
	// Two targets answered with the same row are each a judgement the user
	// should see: neither is shown green.
	for _, targets := range picked {
		if len(targets) < 2 {
			continue
		}
		for _, id := range targets {
			m := d.settled[id]
			m.Band = BandAmber
			d.settled[id] = m
		}
	}
	return grey
}

// Outcome is one finished read.
type Outcome struct {
	Readings []Reading
	// NotMatched are targets no row was offered for: an empty shortlist, or
	// left out of the request by the shrink steps.
	NotMatched []string
	// Grey are targets whose question failed; a background retry asks again.
	Grey   []string
	Rows   []Row
	Checks []string
	Asked  bool
}

// Finish turns matches into readings: start, finish and duration for each
// matched target, each with the row's verbatim text and code's parse.
func (d *Draft) Finish() Outcome {
	out := Outcome{Rows: d.h.Rows, Checks: d.h.Checks, NotMatched: []string{}, Grey: append([]string{}, d.grey...), Readings: []Reading{}}
	for _, t := range d.targets {
		m, ok := d.settled[t.ID]
		if !ok {
			if _, open := d.shortlist[t.ID]; !open || contains(d.notAsked, t.ID) {
				out.NotMatched = append(out.NotMatched, t.ID)
			}
			continue
		}
		r := d.rows[m.RowID]
		stage := ""
		if t.Kind == "stage" {
			stage = t.CatalogueStageID
		}
		add := func(field, value string, parsed map[string]any) {
			if strings.TrimSpace(value) == "" {
				return
			}
			parsed["row"], parsed["row_label"] = r.ID, r.Label
			out.Readings = append(out.Readings, Reading{TargetID: t.ID, CatalogueStageID: stage, RowID: r.ID, Field: field, Value: value, Parsed: parsed,
				Band: m.Band, DecidedBy: m.DecidedBy, Confidence: m.Confidence, QuestionVersion: m.QuestionVersion})
		}
		add(FieldStart, r.StartRaw, dateParsed(r.Start))
		add(FieldFinish, r.FinishRaw, dateParsed(r.Finish))
		dur := map[string]any{}
		if r.Duration != nil {
			dur["low"], dur["high"] = r.Duration.Low, r.Duration.High
		}
		add(FieldDuration, r.DurationRaw, dur)
	}
	return out
}

func dateParsed(d string) map[string]any {
	if d == "" {
		return map[string]any{}
	}
	return map[string]any{"date": d}
}

// Matches are the settled targets, for the eval and tests.
func (d *Draft) Matches() map[string]Match {
	out := make(map[string]Match, len(d.settled))
	for k, v := range d.settled {
		out[k] = v
	}
	return out
}

// Asked are the target ids in the last Call, for the eval.
func (d *Draft) Asked() []string {
	out := make([]string, 0, len(d.asked))
	for id := range d.asked {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Options is an asked target's option count, none included.
func (d *Draft) Options(target string) int { return d.options[target] }

// Raw is Jev's own answer for an asked target before any threshold: the row
// id ("" for none), its confidence, and whether it answered. For the eval.
func (d *Draft) Raw(res jev.Result, target string) (string, *float64, bool) {
	if _, ok := d.asked[target]; !ok {
		return "", nil, false
	}
	a, ok := res.Answers[target]
	if !ok || a.Type != jev.TypeChoice {
		return "", nil, false
	}
	if a.Choice == choiceNone {
		return "", a.Confidence, true
	}
	return a.Choice, a.Confidence, true
}

// Read reads one harvested file: rules, at most one Jev fan-out, then code.
func Read(ctx context.Context, ask Asker, th Thresholds, h Harvest, targets []Target) (Outcome, error) {
	d := NewDraft(h, targets)
	asked := false
	if call, ok := d.Call(); ok {
		asked = true
		res, err := ask.Ask(ctx, call)
		if err != nil && ctx.Err() != nil {
			return Outcome{}, ctx.Err()
		}
		d.Apply(res, err, th)
	}
	out := d.Finish()
	out.Asked = asked
	return out, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

var (
	// stageWords is "stage 3", "stage 2a": a stage number says where a row
	// sits, not what work it is.
	stageWords = regexp.MustCompile(`\bstage\s*[0-9]+[a-z]?\b`)
	// outlineNumber is a leading "1.", "2.3", "4)" numbering.
	outlineNumber = regexp.MustCompile(`^\s*[0-9]+(?:\.[0-9]+)*[.)]?\s+`)
	nonWord       = regexp.MustCompile(`[^a-z0-9]+`)
)

// Synonyms maps a normalised word to the word it is matched as. The plan
// names "the existing construction synonym list used by search"; search in
// this repo is PostgreSQL 'english' full text with no thesaurus, so no such
// list exists and none is invented here. Words map to themselves until the
// knowledge owner authors one.
var Synonyms = map[string]string{}

// Normalise is a label reduced for matching (plan 10.2 "Rules first"):
// lower case, stage numbers and leading outline numbers stripped,
// punctuation removed, spaces collapsed, synonyms applied.
func Normalise(label string) string {
	s := strings.ToLower(label)
	s = outlineNumber.ReplaceAllString(s, "")
	s = stageWords.ReplaceAllString(s, " ")
	s = nonWord.ReplaceAllString(s, " ")
	words := strings.Fields(s)
	for i, w := range words {
		if to, ok := Synonyms[w]; ok {
			words[i] = to
		}
	}
	return strings.Join(words, " ")
}

func trigrams(s string) map[string]bool {
	out := map[string]bool{}
	if s == "" {
		return out
	}
	p := "  " + s + " "
	r := []rune(p)
	for i := 0; i+3 <= len(r); i++ {
		out[string(r[i:i+3])] = true
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for g := range a {
		if b[g] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}
