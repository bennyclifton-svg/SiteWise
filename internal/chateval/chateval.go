// Package chateval scores what the chat agent did against the command key
// (data/eval/chat/commands.json). It knows nothing about the model or the
// store: an Agent turns a case into an Outcome, and Score judges the Outcome.
// The same scorer serves a replayed transcript and a live run.
package chateval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Case is one typed message on one fixture project and what must be true
// after the turn.
type Case struct {
	ID      string `json:"id"`
	Source  string `json:"source"`  // worked | review | area | guard | question
	Group   string `json:"group"`   // the row of the score sheet this case counts under
	Fixture string `json:"fixture"` // house | apartments | refurb
	Message string `json:"message"`
	// Money marks a case that writes an amount, a date or a duration: one
	// wrong change in any run fails the case whatever the floor.
	Money bool   `json:"money_or_date,omitempty"`
	Note  string `json:"note,omitempty"`
	// Needs names slower parts of the fixture a case depends on, so only
	// those cases pay for them: "tenders" reads the fixture's tenders.
	Needs []string `json:"needs,omitempty"`
	// Before are earlier messages of the same conversation, sent first; the
	// case judges where the whole conversation leaves the project. Screen
	// is what the strip says is on screen when Message is sent.
	Before []string      `json:"before,omitempty"`
	Screen *Screen       `json:"screen,omitempty"`
	Expect []Expectation `json:"expect"`
}

// Screen is the screen context sent with a case's message, with rows named
// the way the fixture names them (see Rows), never by id: the runner turns
// them into this fixture's ids.
type Screen struct {
	View     string `json:"view,omitempty"` // costs | program | procurement | profile | specifications
	Open     *Named `json:"open,omitempty"`
	Selected *Rows  `json:"selected,omitempty"`
}

// Named is one row of a table, named as the fixture names it.
type Named struct {
	Kind string `json:"kind"` // document
	Name string `json:"name"`
}

// Rows are rows of one table, named as the fixture names them: a cost line
// by its label, a document by its title, a Program row by its label, a
// package by its discipline id.
type Rows struct {
	Table string   `json:"table"` // cost_lines | documents | program | procurement
	Rows  []string `json:"rows"`
}

// Expectation is one thing that must hold. Kinds:
//
//	state          the read of Area after the turn holds every Has line and no Lacks line
//	asks           the turn ends with a question and no change
//	refuses        the turn ends with no change and says why (Has words in the text)
//	prepared       a change waits for the PM's click; nothing was applied
//	cites          the answer quotes Document and no other document
//	no_change      nothing was applied or prepared
//	not_applied    nothing was applied; a change may wait for a click
//	no_own_figure  the answer carries no figure, date or clause of the model's own
//	states_search  the answer says what was searched (Go's coverage line)
//	no_verdict     the answer draws no conclusion on compliance or risk level
//	selects        the screen moved: Do on Table with exactly Rows (by name)
//	opens          a dialog opened with Fields (by name); nothing was done in it
//	no_model       Go answered the message; the model was not called
type Expectation struct {
	Kind  string   `json:"kind"`
	Area  string   `json:"area,omitempty"`
	Has   []string `json:"has,omitempty"`
	Lacks []string `json:"lacks,omitempty"`
	// Count, on a state expectation, is how many lines of the read must
	// hold every Has string ("five rows of ten days"), no more and no fewer.
	Count    int    `json:"count,omitempty"`
	Document string `json:"document,omitempty"`
	// On selects: the table, the move (select when empty; only, show,
	// clear or open) and the exact set of rows, by name, in any order.
	Table string   `json:"table,omitempty"`
	Do    string   `json:"do,omitempty"`
	Rows  []string `json:"rows,omitempty"`
	// On opens: the dialog and its fields by name (award: package is the
	// discipline id, column the firm's name).
	Dialog string            `json:"dialog,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

// View is one screen move of the turn, with rows named as the fixture
// names them. The runner names them; a recording carries the names.
type View struct {
	Do     string            `json:"do"`
	Table  string            `json:"table,omitempty"`
	Rows   []string          `json:"rows,omitempty"`
	Dialog string            `json:"dialog,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

// Outcome is what one turn did, as the scorer needs it.
type Outcome struct {
	Text      string            `json:"text"`
	Question  bool              `json:"question"`
	Changed   []string          `json:"changed"`         // areas whose read differs from before the turn
	Prepared  int               `json:"prepared"`        // changes waiting for a click
	After     map[string]string `json:"after"`           // area to its read after the turn, ids replaced by labels
	Cited     []string          `json:"cited"`           // titles of documents whose quotations checked out
	Unchecked int               `json:"unchecked"`       // figures, dates and clauses Go marked "check it"
	Coverage  bool              `json:"coverage"`        // Go's "what was searched" line is on the answer
	Verdicts  int               `json:"verdicts"`        // conclusions on risk or compliance Go marked
	Views     []View            `json:"views,omitempty"` // the turn's screen moves
	Rounds    int               `json:"rounds"`
	Millis    int64             `json:"millis"`
	InTokens  int64             `json:"in_tokens"`
	OutTokens int64             `json:"out_tokens"`
	CacheRead int64             `json:"cache_read_tokens"`
}

// Verdict classes. A failing verdict has exactly one class: the worst thing
// that went wrong, in the order below.
const (
	Passed          = "passed"
	WrongChange     = "wrong_change"
	MissedQuestion  = "missed_question"
	UncheckedFigure = "unchecked_figure"
	Failed          = "failed" // anything else: a missing change, a missing citation
)

type Verdict struct {
	Pass    bool     `json:"pass"`
	Class   string   `json:"class"`
	Reasons []string `json:"reasons,omitempty"`
}

var kinds = []string{"state", "asks", "refuses", "prepared", "cites", "no_change", "not_applied", "no_own_figure", "states_search", "no_verdict", "selects", "opens", "no_model"}

// verdictWords are conclusions the agent must not draw (review change 1 and
// cases 2, 4 and 21). A case may add its own with Lacks.
var verdictWords = []string{"low risk", "is compliant", "it complies", "does comply", "does not comply", "no asbestos", "there are no trees", "not risky"}

// Score judges one outcome against one case.
func Score(c Case, o Outcome) Verdict {
	rank := map[string]int{WrongChange: 4, MissedQuestion: 3, UncheckedFigure: 2, Failed: 1}
	v := Verdict{Pass: true, Class: Passed}
	fail := func(class, reason string) {
		v.Pass = false
		if rank[class] > rank[v.Class] {
			v.Class = class
		}
		v.Reasons = append(v.Reasons, reason)
	}
	// Any area that changed must be one a state expectation names.
	allowed := map[string]bool{}
	for _, e := range c.Expect {
		if e.Kind == "state" {
			allowed[e.Area] = true
		}
	}
	for _, area := range o.Changed {
		if !allowed[area] {
			fail(WrongChange, area+" changed and the case expects no change there")
		}
	}
	// A typographic apostrophe is the same word ("can’t").
	text := strings.ReplaceAll(strings.ToLower(o.Text), "’", "'")
	for _, e := range c.Expect {
		switch e.Kind {
		case "state":
			read := o.After[e.Area]
			changed := slices.Contains(o.Changed, e.Area)
			missing := Failed
			if changed {
				missing = WrongChange
			}
			if e.Count > 0 {
				if n := linesWithAll(read, e.Has); n != e.Count {
					fail(missing, fmt.Sprintf("%s has %d lines with %v, want %d", e.Area, n, e.Has, e.Count))
				}
				break
			}
			for _, line := range e.Has {
				if !strings.Contains(read, line) {
					fail(missing, e.Area+" lacks: "+line)
				}
			}
			for _, line := range e.Lacks {
				if strings.Contains(read, line) {
					// Still there because nothing was done is a miss; still
					// there after a change to the area is a wrong change.
					fail(missing, e.Area+" still has: "+line)
				}
			}
		case "asks":
			if !o.Question {
				fail(MissedQuestion, "no question was asked")
			}
			if o.Prepared > 0 {
				fail(WrongChange, "a change was prepared where a question was due")
			}
		case "refuses":
			if o.Prepared > 0 {
				fail(WrongChange, "a change was prepared where a refusal was due")
			}
			for _, w := range e.Has {
				if !strings.Contains(text, strings.ToLower(w)) {
					fail(Failed, "the refusal does not say: "+w)
				}
			}
		case "prepared":
			if o.Prepared == 0 {
				fail(Failed, "nothing was prepared")
			}
		case "no_change":
			if o.Prepared > 0 {
				fail(WrongChange, "a change was prepared")
			}
		case "cites":
			if !slices.Contains(o.Cited, e.Document) {
				fail(Failed, "does not quote "+e.Document)
			}
			for _, d := range o.Cited {
				if d != e.Document {
					fail(Failed, "quotes another document: "+d)
				}
			}
		case "no_own_figure":
			if o.Unchecked > 0 {
				fail(UncheckedFigure, fmt.Sprintf("%d figure(s) of the model's own", o.Unchecked))
			}
		case "states_search":
			if !o.Coverage {
				fail(Failed, "does not say what was searched")
			}
		case "no_verdict":
			if o.Verdicts > 0 {
				fail(Failed, fmt.Sprintf("draws %d conclusion(s) on risk or compliance", o.Verdicts))
			}
			for _, w := range append(slices.Clone(verdictWords), e.Lacks...) {
				w = strings.ToLower(w)
				for from := 0; ; {
					i := strings.Index(text[from:], w)
					if i < 0 {
						break
					}
					at := from + i
					from = at + len(w)
					if !declined(text, at) {
						fail(Failed, "draws a conclusion: "+w)
						break
					}
				}
			}
		case "selects":
			if !moved(o.Views, e) {
				fail(Failed, fmt.Sprintf("the screen did not %s %s %v; moves: %s", doOf(e), e.Table, e.Rows, movesSaid(o.Views)))
			}
		case "opens":
			if !opened(o.Views, e) {
				fail(Failed, fmt.Sprintf("%s did not open with %v; moves: %s", e.Dialog, e.Fields, movesSaid(o.Views)))
			}
		case "no_model":
			if o.Rounds > 0 {
				fail(Failed, fmt.Sprintf("the model was called (%d rounds); Go answers this", o.Rounds))
			}
		}
	}
	return v
}

func doOf(e Expectation) string {
	if e.Do == "" {
		return "select"
	}
	return e.Do
}

// moved reports whether one of the turn's moves is e's: the same move on
// the same table with exactly the same rows, in any order.
func moved(views []View, e Expectation) bool {
	for _, v := range views {
		if v.Dialog == "" && v.Do == doOf(e) && v.Table == e.Table && sameRows(v.Rows, e.Rows) {
			return true
		}
	}
	return false
}

// opened reports whether the dialog opened with every field e names.
func opened(views []View, e Expectation) bool {
	for _, v := range views {
		if v.Do != "open" || v.Dialog != e.Dialog {
			continue
		}
		all := true
		for k, want := range e.Fields {
			all = all && v.Fields[k] == want
		}
		if all {
			return true
		}
	}
	return false
}

// sameRows compares two lists of row names as multisets: a fixture may
// hold two rows of one name.
func sameRows(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	sort.Strings(x)
	sort.Strings(y)
	return slices.Equal(x, y)
}

func movesSaid(views []View) string {
	if len(views) == 0 {
		return "none"
	}
	var parts []string
	for _, v := range views {
		if v.Dialog != "" {
			parts = append(parts, fmt.Sprintf("open %s %v", v.Dialog, v.Fields))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s %v", v.Do, v.Table, v.Rows))
	}
	return strings.Join(parts, "; ")
}

// declined reports whether the words before position at decline to say
// what follows ("I can't say what is late"). It mirrors the server's guard.
func declined(lower string, at int) bool {
	before := lower[max(at-48, 0):at]
	if i := strings.LastIndexAny(before, ".!?\n"); i >= 0 {
		before = before[i+1:]
	}
	for _, d := range []string{"can't", "cannot", "can not", "don't", "doesn't", "does not", "do not", "not ", "never", "isn't", "won't", "whether", "rather than"} {
		if strings.Contains(before, d) {
			return true
		}
	}
	return false
}

func linesWithAll(read string, words []string) int {
	n := 0
	for _, line := range strings.Split(read, "\n") {
		all := true
		for _, w := range words {
			all = all && strings.Contains(line, w)
		}
		if all {
			n++
		}
	}
	return n
}

// Load reads and checks the command key.
func Load(path string) ([]Case, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file struct {
		Cases []Case `json:"cases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, c := range file.Cases {
		switch {
		case c.ID == "" || seen[c.ID]:
			return nil, fmt.Errorf("case id %q is empty or repeated", c.ID)
		case !slices.Contains([]string{"house", "apartments", "refurb"}, c.Fixture):
			return nil, fmt.Errorf("%s: unknown fixture %q", c.ID, c.Fixture)
		case strings.TrimSpace(c.Message) == "" || c.Group == "" || len(c.Expect) == 0:
			return nil, fmt.Errorf("%s: needs a message, a group and an expectation", c.ID)
		}
		seen[c.ID] = true
		for _, e := range c.Expect {
			if !slices.Contains(kinds, e.Kind) || (e.Kind == "state" && e.Area == "") || (e.Kind == "cites" && e.Document == "") ||
				(e.Kind == "selects" && !slices.Contains(Tables, e.Table)) || (e.Kind == "opens" && (e.Dialog == "" || len(e.Fields) == 0)) {
				return nil, fmt.Errorf("%s: bad expectation %+v", c.ID, e)
			}
		}
		if err := checkScreen(c); err != nil {
			return nil, fmt.Errorf("%s: %w", c.ID, err)
		}
	}
	return file.Cases, nil
}

// Tables are the tables a screen move or a selection names.
var Tables = []string{"cost_lines", "documents", "program", "procurement", "register", "messages"}

func checkScreen(c Case) error {
	for _, m := range c.Before {
		if strings.TrimSpace(m) == "" {
			return fmt.Errorf("an earlier message is empty")
		}
	}
	if c.Screen == nil {
		return nil
	}
	if s := c.Screen.Selected; s != nil && (!slices.Contains(Tables, s.Table) || len(s.Rows) == 0) {
		return fmt.Errorf("a selection names a table and its rows")
	}
	if o := c.Screen.Open; o != nil && (o.Kind != "document" || o.Name == "") {
		return fmt.Errorf("the open row is a document named by its title")
	}
	return nil
}

// Agent runs one case from a fresh fixture and reports what happened.
type Agent interface {
	Run(ctx context.Context, c Case) (Outcome, error)
}

type AgentFunc func(context.Context, Case) (Outcome, error)

func (f AgentFunc) Run(ctx context.Context, c Case) (Outcome, error) { return f(ctx, c) }

// Replay is the stand-in agent: it answers each case with its recording.
func Replay(recorded map[string]Outcome) Agent {
	return AgentFunc(func(_ context.Context, c Case) (Outcome, error) {
		o, ok := recorded[c.ID]
		if !ok {
			return Outcome{}, fmt.Errorf("%s: no recording", c.ID)
		}
		return o, nil
	})
}

// Row is one line of the score sheet: every run of every case in a group.
type Row struct {
	Runs            int   `json:"runs"`
	Passed          int   `json:"passed"`
	WrongChange     int   `json:"wrong_change"`
	MissedQuestion  int   `json:"missed_question"`
	UncheckedFigure int   `json:"unchecked_figure"`
	Failed          int   `json:"failed"`
	Errors          int   `json:"errors"`
	Prepared        int   `json:"ended_prepared"`
	Rounds          int   `json:"rounds"`
	Millis          []int `json:"millis"`
}

// Sheet is the result of a run of the key.
type Sheet struct {
	Groups map[string]Row `json:"groups"`
	// Failed lists money and date cases with a wrong change in any run.
	Failed []string            `json:"failed_money_or_date_cases"`
	Notes  map[string][]string `json:"notes"` // case id to its failure reasons
}

// Workers is how many runs RunAll does at once. One keeps timings clean.
var Workers = 1

// RunAll runs every case the given number of times and fills the sheet.
func RunAll(ctx context.Context, agent Agent, cases []Case, runs int) Sheet {
	// Every run is independent (a fresh fixture each), so they may run a
	// few at a time; the sheet is filled in case order afterwards.
	type result struct {
		o   Outcome
		err error
	}
	results := make([][]result, len(cases))
	workers := max(Workers, 1)
	slots := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for ci, c := range cases {
		results[ci] = make([]result, runs)
		for i := 0; i < runs; i++ {
			wg.Add(1)
			slots <- struct{}{}
			go func() {
				defer func() { <-slots; wg.Done() }()
				o, err := agent.Run(ctx, c)
				results[ci][i] = result{o, err}
			}()
		}
	}
	wg.Wait()
	sheet := Sheet{Groups: map[string]Row{}, Notes: map[string][]string{}}
	for ci, c := range cases {
		row := sheet.Groups[c.Group]
		for i := 0; i < runs; i++ {
			row.Runs++
			o, err := results[ci][i].o, results[ci][i].err
			if err != nil {
				row.Errors++
				sheet.Notes[c.ID] = append(sheet.Notes[c.ID], "error: "+err.Error())
				continue
			}
			row.Rounds += o.Rounds
			row.Millis = append(row.Millis, int(o.Millis))
			if o.Prepared > 0 {
				row.Prepared++
			}
			v := Score(c, o)
			switch v.Class {
			case Passed:
				row.Passed++
			case WrongChange:
				row.WrongChange++
				if c.Money && !slices.Contains(sheet.Failed, c.ID) {
					sheet.Failed = append(sheet.Failed, c.ID)
				}
			case MissedQuestion:
				row.MissedQuestion++
			case UncheckedFigure:
				row.UncheckedFigure++
			default:
				row.Failed++
			}
			if !v.Pass {
				sheet.Notes[c.ID] = append(sheet.Notes[c.ID], fmt.Sprintf("run %d %s: %s", i+1, v.Class, strings.Join(v.Reasons, "; ")))
			}
		}
		sheet.Groups[c.Group] = row
	}
	return sheet
}

// Percentile returns the p-th percentile (0 to 100) of the samples, or 0.
func Percentile(samples []int, p int) int {
	if len(samples) == 0 {
		return 0
	}
	s := slices.Clone(samples)
	sort.Ints(s)
	i := (len(s)*p + 99) / 100
	if i < 1 {
		i = 1
	}
	return s[i-1]
}
