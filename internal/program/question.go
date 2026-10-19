package program

import (
	"fmt"
	"strings"

	"sitewise/internal/knowledge"
)

// The one question (plan 7.3): the single unknown Profile fact that would
// narrow the completion range most. Code chooses it by regenerating the
// program in memory for each possible answer; no Jev.

// MaxQuestionRegenerations caps the in-memory regenerations per request
// (plan 7.3 step 4). Candidates are scored in order (determinants, then
// conditions, then planning keys) until the next candidate's answers would
// not fit in what is left; the rest are not scored. Generation plus Compute
// on a 30-package program is well under a millisecond each, so 32 keeps the
// read inside its budget with room to spare.
const MaxQuestionRegenerations = 32

// Question kinds say which Profile path an answer is written through.
const (
	QuestionDeterminant = "determinant"
	QuestionWorkType    = "work_type"
	QuestionCondition   = "condition"
	QuestionPlanning    = "planning"
)

// Question is the strip's one question. Options are the labels shown;
// Values are the stored values in the same order. A boolean also shows
// "Not sure", which has no value: it dismisses the question until the
// Profile value changes. A number has no options: the strip asks for it.
type Question struct {
	Key           string   `json:"key"`
	Kind          string   `json:"kind"`
	Label         string   `json:"label"`
	Options       []string `json:"options"`
	Values        []string `json:"values"`
	Number        bool     `json:"number,omitempty"`
	NarrowsByDays int      `json:"narrows_by_days"`
}

// NotSure is the boolean option that dismisses the question.
const NotSure = "Not sure"

type candidate struct {
	q      Question
	values []string
}

// OneQuestion returns the question to ask, or nil when no unknown fact is
// referenced by the library. dismissed holds keys the user answered "Not
// sure" to since their Profile value last changed.
//
// When nothing narrows the range (every score is 0, as with a library that
// has no reviewed durations) the first unknown referenced fact is still
// offered with narrows_by_days 0: plan 13.1 offers one question on a
// project with almost no facts, and answering it can still change which
// stages and activities apply. With no referenced unknown at all (the empty
// library) there is nothing to ask and the strip shows only the start
// suggestion.
//
// narrows_by_days is in working days: current width minus the worst-case
// width after answering, where width is PF_high − PF_low of the known rows.
func OneQuestion(cat *knowledge.Catalog, in GenInput, dismissed map[string]bool) (*Question, error) {
	cands := referencedUnknowns(cat, in, dismissed)
	if len(cands) == 0 {
		return nil, nil
	}
	current, err := width(cat, in)
	if err != nil {
		return nil, err
	}
	best, bestScore, used := -1, 0, 0
	for i, c := range cands {
		if len(c.values) == 0 {
			continue // a number cannot be enumerated; it scores 0
		}
		if used+len(c.values) > MaxQuestionRegenerations {
			break
		}
		worst := 0
		for _, v := range c.values {
			used++
			w, err := width(cat, answer(in, c.q, v))
			if err != nil {
				return nil, err
			}
			worst = max(worst, w)
		}
		if score := current - worst; score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		best, bestScore = 0, 0
	}
	q := cands[best].q
	q.NarrowsByDays = bestScore
	return &q, nil
}

// width is PF_high − PF_low of a regenerated program, 0 when unknown.
func width(cat *knowledge.Catalog, in GenInput) (int, error) {
	p, err := Generate(cat, in)
	if err != nil {
		return 0, err
	}
	out := Compute(p.Input(Calendar{WorkingWeek: in.WorkingWeek}))
	if out.Finish.HighIndex == nil || out.Finish.LowIndex == nil {
		return 0, nil
	}
	return *out.Finish.HighIndex - *out.Finish.LowIndex, nil
}

// answer is a copy of in with one more Profile value.
func answer(in GenInput, q Question, v string) GenInput {
	out := in
	switch q.Kind {
	case QuestionWorkType:
		out.WorkTypes = []string{v}
	case QuestionCondition:
		out.Conditions = withValue(in.Conditions, q.Key, v)
	case QuestionPlanning:
		out.Planning = withValue(in.Planning, q.Key, v)
	default:
		out.Values = withValue(in.Values, q.Key, v)
	}
	return out
}

func withValue(m map[string]string, k, v string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for a, b := range m {
		out[a] = b
	}
	out[k] = v
	return out
}

// referencedUnknowns collects the unknown determinants, conditions and
// planning values that a skeleton, fragment or reviewed duration predicate
// or driver reads, in the order ties break: determinants in
// determinants.yaml order, then conditions in taxonomy order, then planning
// keys in registry order. Draft duration rows are never applied, so their
// predicates and drivers ask nothing. Numbers are asked only as drivers of
// reviewed rows; a numeric comparison in a predicate is not enumerated.
func referencedUnknowns(cat *knowledge.Catalog, in GenInput, dismissed map[string]bool) []candidate {
	dets, conds, plans := map[string]bool{}, map[string]bool{}, map[string]bool{}
	numberDets := map[string]bool{}
	read := func(p any) {
		for _, leaf := range leaves(p) {
			if d, ok := leaf["det"]; ok {
				dets[fmt.Sprint(d)] = true
			}
			if c, ok := leaf["cond"]; ok {
				conds[fmt.Sprint(c)] = true
			}
		}
	}
	for _, s := range cat.Skeletons() {
		read(s.AppliesWhen)
	}
	for _, f := range cat.Fragments() {
		read(f.AppliesWhen)
	}
	for _, id := range durationIDs(cat) {
		d, ok := cat.ReviewedDuration(id)
		if !ok {
			continue
		}
		read(d.AppliesWhen)
		if d.Driver != nil {
			if d.Driver.PlanningKey != "" {
				plans[d.Driver.PlanningKey] = true
			}
			if d.Driver.Determinant != "" {
				dets[d.Driver.Determinant] = true
				numberDets[d.Driver.Determinant] = true
			}
		}
	}

	var out []candidate
	for _, d := range cat.Determinants() {
		if !dets[d.ID] || dismissed[d.ID] {
			continue
		}
		if d.ID == "work_type" {
			if len(in.WorkTypes) > 0 {
				continue
			}
			c := candidate{q: Question{Key: d.ID, Kind: QuestionWorkType, Label: d.Label}}
			for _, w := range cat.Taxonomy().WorkTypes {
				c.values = append(c.values, w.ID)
				c.q.Options = append(c.q.Options, choiceLabel([]knowledge.Choice{w}, w.ID))
			}
			c.q.Values = c.values
			out = append(out, c)
			continue
		}
		if strings.TrimSpace(in.Values[d.ID]) != "" {
			continue
		}
		c := candidate{q: Question{Key: d.ID, Kind: QuestionDeterminant, Label: d.Label}}
		switch d.Value {
		case "boolean":
			c.values = []string{"true", "false"}
			c.q.Options = []string{"Yes", "No", NotSure}
		case "choice", "multi_choice":
			for _, o := range d.Options {
				// Option ids are the Profile's own values; `describes` is
				// Jev wording, not a label.
				c.values = append(c.values, o.ID)
				c.q.Options = append(c.q.Options, o.ID)
			}
		default:
			if !numberDets[d.ID] {
				continue
			}
			c.q.Number, c.q.Label = true, "Enter "+d.Label
		}
		c.q.Values = c.values
		out = append(out, c)
	}
	for _, cond := range cat.Taxonomy().Conditions {
		if !conds[cond.Key] || dismissed[cond.Key] || strings.TrimSpace(in.Conditions[cond.Key]) != "" {
			continue
		}
		c := candidate{q: Question{Key: cond.Key, Kind: QuestionCondition, Label: cond.Label}}
		for _, o := range cond.AllOptions() {
			c.values = append(c.values, o.ID)
			c.q.Options = append(c.q.Options, choiceLabel([]knowledge.Choice{o}, o.ID))
		}
		c.q.Values = c.values
		out = append(out, c)
	}
	for _, k := range cat.PlanningKeys() {
		if !plans[k.Key] || dismissed[k.Key] || strings.TrimSpace(in.Planning[k.Key]) != "" {
			continue
		}
		c := candidate{q: Question{Key: k.Key, Kind: QuestionPlanning, Label: k.Label}}
		switch k.Value {
		case "choice":
			for _, o := range k.Options {
				c.values = append(c.values, o.ID)
				c.q.Options = append(c.q.Options, o.Label)
			}
		case "boolean":
			c.values = []string{"true", "false"}
			c.q.Options = []string{"Yes", "No", NotSure}
		default:
			c.q.Number, c.q.Label = true, "Enter "+k.Label
		}
		c.q.Values = c.values
		out = append(out, c)
	}
	return out
}

// durationIDs lists every duration id the library references (activity
// durations and lags), in library order, without repeats.
func durationIDs(cat *knowledge.Catalog) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id *string) {
		if id != nil && !seen[*id] {
			seen[*id] = true
			ids = append(ids, *id)
		}
	}
	for _, s := range cat.Skeletons() {
		for _, l := range s.Links {
			add(l.Lag)
		}
	}
	for _, f := range cat.Fragments() {
		for _, a := range f.Activities {
			add(a.Duration)
		}
		for _, l := range f.Links {
			add(l.Lag)
		}
	}
	return ids
}
