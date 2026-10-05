package eval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// choiceNew is the supersession answer for "replaces none of these".
const choiceNew = "new"

// Evaluator files each case through the production draft path: the same
// harvest, rules, question map and threshold bands as Service.File, with no
// database. Supersession candidates are the earlier cases of the same corpus
// and split, filed under their expected number and revision.
type Evaluator struct {
	Catalog    intake.Catalog
	Thresholds intake.Thresholds
	Asker      intake.Asker
	// Labeler, when set, is told which case the next Jev call belongs to.
	Labeler interface{ SetCase(string) }
	// Misses, when set, reports replay misses; any miss fails the run.
	Misses func() int
	// Deadline overrides the client deadline per call. Recording uses a long
	// one so accuracy is not truncated; the bench measures the real deadline.
	Deadline time.Duration
}

// CaseResult is one filed case. Truth includes the derived supersession label.
type CaseResult struct {
	ID        string
	Corpus    string
	Split     string
	NotFiled  string
	Asked     bool
	Grey      bool
	Decisions []intake.Decision
	Choices   []intake.Choice
	Link      string
	Truth     map[string]Label
	// Sheets is set when the application would split the case into drawing
	// sheets; its calls are part of the recording (AT-35).
	Sheets *SheetRun
}

// RunResult is every case result and the Jev call counts.
type RunResult struct {
	Cases  []CaseResult
	Calls  int
	Errors int
	Misses int
}

type filedPrior struct {
	doc  store.NumberedDocument
	auth bool
}

// Run files every case in order. A replay miss is an error: replay must
// reproduce the recorded run exactly.
func (e Evaluator) Run(ctx context.Context, set CaseSet) (*RunResult, error) {
	if e.Asker == nil {
		return nil, errors.New("evaluator has no Jev asker")
	}
	out := &RunResult{}
	priors := map[string][]filedPrior{}
	for _, c := range set.Cases {
		scope := c.Corpus + "/" + c.Split
		res, err := e.file(ctx, c, priors[scope], out)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.ID, err)
		}
		out.Cases = append(out.Cases, res)
		if n, ok := c.Labels[intake.FieldNumber]; ok {
			rev := c.Labels[intake.FieldRevision]
			priors[scope] = append(priors[scope], filedPrior{
				doc:  store.NumberedDocument{ID: c.ID, Number: n.Value, Revision: rev.Value},
				auth: n.Authoritative && rev.Authoritative,
			})
		}
	}
	if e.Misses != nil {
		out.Misses = e.Misses()
		if out.Misses > 0 {
			return out, fmt.Errorf("%w: %d requests; re-record with -live", ErrReplayMiss, out.Misses)
		}
	}
	return out, nil
}

func (e Evaluator) file(ctx context.Context, c Case, priors []filedPrior, run *RunResult) (CaseResult, error) {
	res := CaseResult{ID: c.ID, Corpus: c.Corpus, Split: c.Split, Truth: map[string]Label{}}
	for f, l := range c.Labels {
		res.Truth[f] = l
	}
	if l, ok := supersessionTruth(c, priors); ok {
		res.Truth[intake.FieldSupersedes] = l
	}
	text, reason, err := extract(ctx, c)
	if err != nil {
		return res, err
	}
	if reason != "" {
		res.NotFiled = reason
		return res, nil
	}
	harvested := intake.Harvest(c.Filename(), text)
	draft := intake.NewDraft(e.Catalog, c.Filename(), text, harvested, nil)
	docs := make([]store.NumberedDocument, len(priors))
	for i, p := range priors {
		docs[i] = p.doc
	}
	draft.Plan(docs, c.ID)
	if call, ok := draft.Call(); ok {
		if e.Deadline > 0 {
			call.Deadline = e.Deadline
		}
		if e.Labeler != nil {
			e.Labeler.SetCase(c.ID)
		}
		result, err := e.Asker.Ask(ctx, call)
		run.Calls++
		if err != nil {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			run.Errors++
		}
		res.Asked = true
		res.Choices = draft.Choices(result)
		res.Grey = draft.Apply(result, err, e.Thresholds)
	}
	res.Link = draft.Link(nil, c.ID)
	res.Decisions = draft.Decisions()
	if expands(c, text.PageCount, res.Decisions) {
		sheets, err := e.expand(ctx, c, text.PageCount, docs, run)
		if err != nil {
			return res, err
		}
		res.Sheets = sheets
	}
	return res, nil
}

func extract(ctx context.Context, c Case) (identity.Text, string, error) {
	f, err := os.Open(c.Path)
	if err != nil {
		return identity.Text{}, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return identity.Text{}, "", err
	}
	text, err := identity.Extract(ctx, c.Format, f, info.Size(), identity.DefaultLimits())
	switch {
	case errors.Is(err, identity.ErrTooLarge):
		return text, intake.ReasonTooLarge, nil
	case errors.Is(err, identity.ErrMalformed):
		return text, intake.ReasonUnreadable, nil
	case err != nil:
		return text, "", err
	case !text.TextLayer:
		return text, intake.ReasonNoText, nil
	}
	return text, "", nil
}

// supersessionTruth derives what the case should replace from the keys: the
// latest earlier revision of the same number, or new. It is unknown when a
// number or revision is unlabeled, or code cannot order the revisions.
func supersessionTruth(c Case, priors []filedPrior) (Label, bool) {
	num, ok := c.Labels[intake.FieldNumber]
	if !ok {
		return Label{}, false
	}
	want, ok := intake.Normalize(intake.FieldNumber, num.Value)
	if !ok {
		return Label{}, false
	}
	rev, hasRev := c.Labels[intake.FieldRevision]
	auth := num.Authoritative && rev.Authoritative
	var earlier []filedPrior
	same := 0
	for _, p := range priors {
		n, ok := intake.Normalize(intake.FieldNumber, p.doc.Number)
		if !ok || n != want {
			continue
		}
		same++
		auth = auth && p.auth
		if !hasRev || p.doc.Revision == "" {
			return Label{}, false
		}
		cmp, comparable := intake.Compare(p.doc.Revision, rev.Value)
		if !comparable {
			return Label{}, false
		}
		if cmp == -1 {
			earlier = append(earlier, p)
		}
	}
	if same == 0 {
		return Label{Value: choiceNew, Authoritative: num.Authoritative, Source: "derived"}, true
	}
	if len(earlier) == 0 {
		return Label{Value: choiceNew, Authoritative: auth, Source: "derived"}, true
	}
	latest := earlier[0]
	for _, p := range earlier[1:] {
		cmp, comparable := intake.Compare(latest.doc.Revision, p.doc.Revision)
		if !comparable {
			return Label{}, false
		}
		if cmp == -1 {
			latest = p
		}
	}
	return Label{Value: latest.doc.ID, Authoritative: auth, Source: "derived"}, true
}

// Outcomes is every scored field in split whose label is in the tier.
func (r *RunResult) Outcomes(split string, authoritative bool) []Outcome {
	var out []Outcome
	for _, c := range r.Cases {
		if c.Split != split {
			continue
		}
		decided := map[string]intake.Decision{}
		for _, d := range c.Decisions {
			decided[d.Field] = d
		}
		for field, label := range c.Truth {
			if label.Authoritative != authoritative {
				continue
			}
			d := decided[field]
			value := appliedValue(d)
			out = append(out, Outcome{
				Case:      c.ID,
				Field:     field,
				Expected:  label.Value,
				Value:     value,
				Band:      d.Band,
				DecidedBy: d.DecidedBy,
				Correct:   value != "" && SameValue(field, label.Value, value),
			})
		}
	}
	return out
}

// appliedValue is the stored value, with an applied "replaces nothing"
// supersession answer spelled new so it can be scored.
func appliedValue(d intake.Decision) string {
	if d.Value != "" {
		return d.Value
	}
	if d.Field == intake.FieldSupersedes && d.DecidedBy == intake.DecidedByJev && (d.Band == intake.BandGreen || d.Band == intake.BandAmber) {
		return choiceNew
	}
	return ""
}

// Answers is every raw Jev choice in split on an authoritative label.
func (r *RunResult) Answers(split string) []Answer {
	var out []Answer
	for _, c := range r.Cases {
		if c.Split != split {
			continue
		}
		for _, ch := range c.Choices {
			label, ok := c.Truth[ch.Field]
			if !ok || !label.Authoritative || ch.Confidence == nil {
				continue
			}
			value := ch.Value
			if ch.Field == intake.FieldSupersedes && ch.ID == choiceNew {
				value = choiceNew
			}
			out = append(out, Answer{
				Case:       c.ID,
				Field:      ch.Field,
				Options:    ch.Options,
				Confidence: *ch.Confidence,
				Correct:    value != "" && SameValue(ch.Field, label.Value, value),
			})
		}
	}
	return out
}

// SupersessionCounts are automatic links and how many were wrong. A link
// whose truth is unknown is unverified and counts against the gate.
type SupersessionCounts struct {
	Asked      int `json:"asked"`
	Scored     int `json:"scored"`
	Automatic  int `json:"automatic"`
	Incorrect  int `json:"incorrect"`
	Unverified int `json:"unverified"`
}

// Supersession counts links in split.
func (r *RunResult) Supersession(split string) SupersessionCounts {
	var s SupersessionCounts
	for _, c := range r.Cases {
		if c.Split != split {
			continue
		}
		truth, known := c.Truth[intake.FieldSupersedes]
		if known && truth.Authoritative {
			s.Scored++
		}
		for _, ch := range c.Choices {
			if ch.Field == intake.FieldSupersedes {
				s.Asked++
			}
		}
		if c.Link == "" {
			continue
		}
		s.Automatic++
		switch {
		case !known:
			s.Unverified++
		case truth.Value != c.Link:
			s.Incorrect++
		}
	}
	return s
}

// SameValue compares an expected label with a filed value the way intake
// compares harvested values. Dates compare as calendar days.
func SameValue(field, expected, got string) bool {
	switch field {
	case intake.FieldNumber, intake.FieldRevision, intake.FieldTitle:
		a, okA := intake.Normalize(field, expected)
		b, okB := intake.Normalize(field, got)
		if okA && okB {
			return a == b
		}
		return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(got))
	case intake.FieldDate:
		a, okA := parseDate(expected)
		b, okB := parseDate(got)
		if okA && okB {
			return a.Equal(b)
		}
		return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(got))
	default:
		return expected == got
	}
}

// Australian documents write the day first.
var dateLayouts = []string{
	"2006-01-02", "02/01/2006", "2/1/2006", "02.01.2006", "2.1.2006", "02-01-2006",
	"02/01/06", "2/1/06", "02.01.06", "2 January 2006", "2 Jan 2006", "02 Jan 2006", "02 January 2006",
	"2 Jan. 2006", "January 2, 2006",
}

var ordinalDate = regexp.MustCompile(`(?i)^(\d{1,2})(?:st|nd|rd|th) ([a-z]+),? (\d{4})$`)

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	s = ordinalDate.ReplaceAllString(s, "${1} ${2} ${3}")
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// interactiveDeadline is the production foreground budget; recorded answers
// slower than this would have been grey in production.
const interactiveDeadline = jev.InteractiveDeadline
