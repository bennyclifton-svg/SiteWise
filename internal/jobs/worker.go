// Package jobs runs the background stages that follow a filing.
// Each stage is one leased job. Labeling and evidence are each one Jev
// fan-out per passage; they are not extra round trips on the filing path.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

const (
	questionVersion = "knowledge.v2+" + profile.QuestionVersion
	maxPassageRunes = 2000
	maxPassages     = 500
)

// Asker is the background Jev client. Calls from this package use background
// priority, which cannot take the interactive reserve.
type Asker interface {
	Ask(ctx context.Context, call jev.Call) (jev.Result, error)
}

// Passage is the state one fan-out reads. Labels are empty until the label
// stage has accepted them.
type Passage struct {
	Ordinal    int
	Text       string
	Section    string
	Kind       string
	Discipline string
	Title      string
	Labels     []string
}

// Worker leases background jobs for one org at a time. MinNoul is an explicit
// band for accepting a label or an evidence noul. Zero accepts nothing, so a
// missing calibration cannot become an automatic action.
type Worker struct {
	Store   *store.Store
	Ask     Asker
	Text    func(ctx context.Context, orgID, documentID string) (string, error)
	Catalog *knowledge.Catalog
	Lease   time.Duration
	Backoff time.Duration
	MinNoul float64
	// Profile holds the profile's per-shape floors. Empty floors apply
	// nothing: readings are stored and rows stay blank.
	Profile profile.Thresholds
}

// Once leases and runs one full-text, label or evidence job.
func (w *Worker) Once(ctx context.Context, orgID string) error {
	lease := w.Lease
	if lease <= 0 {
		lease = 30 * time.Second
	}
	job, err := w.Store.ClaimJob(ctx, orgID, lease, []string{
		store.JobKindFullText,
		store.JobKindLabel,
		store.JobKindEvidence,
	})
	if err != nil {
		return err
	}
	if err := w.Perform(ctx, job); err != nil {
		_ = w.Store.FailJob(ctx, job.OrgID, job.ID, job.Token, err.Error(), w.backoff(), store.EventWrite{
			Kind:       "job",
			DocumentID: job.DocumentID,
			Payload:    jobPayload(job.Kind, store.JobStatusFailed),
		})
		return err
	}
	return w.Store.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{
		Kind:       "job",
		DocumentID: job.DocumentID,
		Payload:    jobPayload(job.Kind, store.JobStatusDone),
	})
}

func (w *Worker) backoff() time.Duration {
	if w.Backoff < 0 {
		return 0
	}
	return w.Backoff
}

// Perform runs the stage without completing the lease. A second delivery of
// the same stage replaces passages and labels instead of appending them.
func (w *Worker) Perform(ctx context.Context, job store.ClaimedJob) error {
	switch job.Kind {
	case store.JobKindFullText:
		return w.fullText(ctx, job)
	case store.JobKindLabel:
		return w.label(ctx, job)
	case store.JobKindEvidence:
		return w.evidence(ctx, job)
	default:
		return errors.New("unknown background stage")
	}
}

func (w *Worker) fullText(ctx context.Context, job store.ClaimedJob) error {
	if w.Text == nil {
		return errors.New("document text is unavailable")
	}
	text, err := w.Text(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return err
	}
	if err := w.Store.ReplacePassages(ctx, job.OrgID, job.DocumentID, SplitPassages(text)); err != nil {
		return err
	}
	return w.Store.EnqueueStage(ctx, job.OrgID, job.DocumentID, store.JobKindLabel)
}

func (w *Worker) label(ctx context.Context, job store.ClaimedJob) error {
	if w.Catalog == nil {
		return errors.New("knowledge is not loaded")
	}
	passages, err := w.passages(ctx, job)
	if err != nil {
		return err
	}
	var facts []store.StoredFact
	for _, passage := range passages {
		call, cands := labelCall(w.Catalog, passage.Passage)
		if len(call.Questions) == 0 {
			continue
		}
		result, err := w.ask(ctx, call)
		if err != nil {
			return err
		}
		labels := AcceptLabels(w.Catalog, result, w.MinNoul)
		if err := w.Store.SetPassageSystems(ctx, job.OrgID, passage.ID, labels); err != nil {
			return err
		}
		facts = append(facts, storedFacts(passage.ID, profile.Readings(result, call.Questions, cands, passage.Text))...)
	}
	if err := w.Store.ReplaceDocumentFacts(ctx, job.OrgID, job.DocumentID, []string{"det.", "fact.", "hdr."}, profile.QuestionVersion, facts); err != nil {
		return err
	}
	return w.Store.EnqueueStage(ctx, job.OrgID, job.DocumentID, store.JobKindEvidence)
}

func (w *Worker) evidence(ctx context.Context, job store.ClaimedJob) error {
	if w.Catalog == nil {
		return errors.New("knowledge is not loaded")
	}
	passages, err := w.passages(ctx, job)
	if err != nil {
		return err
	}
	var facts []store.StoredFact
	for i, passage := range passages {
		labels, err := w.Store.PassageSystems(ctx, job.OrgID, passage.ID)
		if err != nil {
			return err
		}
		passages[i].Labels = labels
		call, ok := EvidenceCall(w.Catalog, passages[i].Passage)
		if !ok {
			continue
		}
		result, err := w.ask(ctx, call)
		if err != nil {
			return err
		}
		if err := w.Store.SetPassageEvidence(ctx, job.OrgID, passage.ID, evidenceStates(call.Questions, result, w.MinNoul)); err != nil {
			return err
		}
		facts = append(facts, storedFacts(passage.ID, profile.Readings(result, call.Questions, nil, passage.Text))...)
	}
	if err := w.Store.ReplaceDocumentFacts(ctx, job.OrgID, job.DocumentID, []string{"sys."}, profile.QuestionVersion, facts); err != nil {
		return err
	}
	return w.rebuildProfile(ctx, job.OrgID, job.DocumentID)
}

// rebuildProfile reconciles the document's project in code after its
// evidence stage. It never calls Jev.
func (w *Worker) rebuildProfile(ctx context.Context, orgID, documentID string) error {
	doc, err := w.Store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return err
	}
	return w.Store.RebuildProfile(ctx, orgID, doc.ProjectID, w.Profile.Version, func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Thresholds: w.Profile}, w.Catalog)
	})
}

func storedFacts(passageID string, readings []profile.Reading) []store.StoredFact {
	out := make([]store.StoredFact, 0, len(readings))
	for _, r := range readings {
		out = append(out, store.StoredFact{PassageID: passageID, QuestionID: r.QuestionID, Value: r.Value, Unit: r.Unit,
			Basis: r.Basis, Excerpt: r.Excerpt, Confidence: r.Confidence, DecidedBy: "jev"})
	}
	return out
}

func (w *Worker) passages(ctx context.Context, job store.ClaimedJob) ([]storedPassage, error) {
	rows, err := w.Store.DocumentPassages(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
	decisions, err := w.Store.DocumentDecisions(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
	kind := decisionValue(decisions, "kind")
	discipline := decisionValue(decisions, "discipline")
	title := decisionValue(decisions, "title")
	out := make([]storedPassage, len(rows))
	section := ""
	for i, row := range rows {
		// Passages carry the nearest heading above them; header routing reads it.
		if h := sectionOf(row.Body); h != "" {
			section = h
		}
		out[i] = storedPassage{
			ID: row.ID,
			Passage: Passage{
				Ordinal:    int(row.Ordinal),
				Text:       row.Body,
				Section:    section,
				Kind:       kind,
				Discipline: discipline,
				Title:      title,
			},
		}
	}
	return out, nil
}

type storedPassage struct {
	ID string
	Passage
}

func (w *Worker) ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	if w.Ask == nil {
		return jev.Result{}, errors.New("jev client is unavailable")
	}
	call.Priority = jev.PriorityBackground
	return w.Ask.Ask(ctx, call)
}

func decisionValue(list []store.StoredDecision, field string) string {
	for _, d := range list {
		if d.Field == field {
			return d.Value
		}
	}
	return ""
}

func jobPayload(kind, status string) string {
	body, _ := json.Marshal(map[string]string{"kind": kind, "status": status})
	return string(body)
}

// SplitPassages breaks full text into indexable passages. Blank lines start a
// new passage. A very long paragraph is cut so one passage cannot dominate
// the index.
func SplitPassages(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	for _, part := range strings.Split(text, "\n\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		for part != "" && len(out) < maxPassages {
			if utf8.RuneCountInString(part) <= maxPassageRunes {
				out = append(out, part)
				break
			}
			cut := cutRunes(part, maxPassageRunes)
			out = append(out, strings.TrimSpace(part[:cut]))
			part = strings.TrimSpace(part[cut:])
		}
		if len(out) >= maxPassages {
			break
		}
	}
	return out
}

func cutRunes(s string, n int) int {
	count := 0
	lastSpace := 0
	for i := range s {
		if count == n {
			if lastSpace > 0 {
				return lastSpace
			}
			return i
		}
		if s[i] == ' ' || s[i] == '\n' {
			lastSpace = i
		}
		count++
	}
	return len(s)
}

// BackgroundCalls is the Jev work for these passages: one label fan-out per
// passage, then one evidence fan-out per passage that has labels. It does not
// emit an interactive call.
func BackgroundCalls(cat *knowledge.Catalog, passages []Passage) []jev.Call {
	var out []jev.Call
	for _, passage := range passages {
		call := LabelCall(cat, passage)
		if len(call.Questions) > 0 {
			out = append(out, call)
		}
		evidence, ok := EvidenceCall(cat, passage)
		if ok {
			out = append(out, evidence)
		}
	}
	return out
}

// LabelCall puts every top-level system noul and every speculative leaf
// choice into one request. Code decides which leaf answers to keep after the
// response; the choices are not a second round trip.
func LabelCall(cat *knowledge.Catalog, passage Passage) jev.Call {
	call, _ := labelCall(cat, passage)
	return call
}

// labelCall also returns the profile candidates offered in the call, so the
// answers can be mapped back to verbatim values.
func labelCall(cat *knowledge.Catalog, passage Passage) (jev.Call, map[string][]profile.Candidate) {
	questions := map[string]jev.Question{}
	if cat == nil {
		return jev.Call{}, nil
	}
	for _, sys := range cat.TopSystems() {
		name := strings.TrimSpace(sys.Label)
		if name == "" {
			name = sys.ID
		}
		questions[knowledge.LabelQuestionID(sys.ID)] = jev.Question{
			Type:         jev.TypeNoul,
			Instructions: "Using `text`, is this passage about " + name + "?",
			Criteria: map[string]string{
				"true":  knowledge.Describe(sys, sys.ID),
				"false": knowledge.ExcludesText(sys),
			},
		}
		children := cat.Children(sys.ID)
		if len(children) == 0 || len(children) > jev.MaxChoiceOptions-1 {
			continue
		}
		criteria := map[string]string{
			knowledge.NoneOption(): "The passage is not about a specific part of this system.",
		}
		for _, child := range children {
			criteria[child.ID] = knowledge.Describe(child, child.ID)
		}
		questions[knowledge.LeafQuestionID(sys.ID)] = jev.Question{
			Type:         jev.TypeChoice,
			Instructions: "Using `text`, which part of " + name + " is this passage about?",
			Criteria:     criteria,
		}
	}
	// Profile questions read the same passage, so they join this request
	// instead of adding a round trip (https://docs.typesafe.ai/patterns/fan-out).
	extra, candidates := profile.LabelQuestions(passage.Info(), profile.Harvest(passage.Text, cat), cat)
	for id, q := range extra {
		questions[id] = q
	}
	state := passageState(passage)
	if len(candidates) > 0 {
		state["candidates"] = candidates
	}
	return jev.Call{
		State:           state,
		Questions:       questions,
		Priority:        jev.PriorityBackground,
		QuestionVersion: questionVersion,
	}, candidates
}

// Info is what profile header routing reads about a passage.
func (p Passage) Info() profile.PassageInfo {
	return profile.PassageInfo{Kind: p.Kind, Section: p.Section, Ordinal: p.Ordinal}
}

// EvidenceCall is one fan-out of the knowledge nouls that match the passage
// labels. It is a different state from labeling and is not asked in the
// foreground filing request.
func EvidenceCall(cat *knowledge.Catalog, passage Passage) (jev.Call, bool) {
	if cat == nil || len(passage.Labels) == 0 {
		return jev.Call{}, false
	}
	questions := map[string]jev.Question{}
	for _, q := range cat.EvidenceQuestions(passage.Labels) {
		if q.Type != knowledgeNoul || strings.TrimSpace(q.Instructions) == "" {
			continue
		}
		questions[q.ID] = jev.Question{
			Type:         jev.TypeNoul,
			Instructions: q.Instructions,
			Criteria:     q.Criteria,
		}
	}
	for id, q := range profile.EvidenceQuestions(passage.Labels, cat) {
		questions[id] = q
	}
	if len(questions) == 0 {
		return jev.Call{}, false
	}
	return jev.Call{
		State:           passageState(passage),
		Questions:       questions,
		Priority:        jev.PriorityBackground,
		QuestionVersion: questionVersion,
	}, true
}

const knowledgeNoul = "noul"

func passageState(p Passage) map[string]any {
	return map[string]any{
		"document": map[string]string{
			"kind":       p.Kind,
			"discipline": p.Discipline,
			"title":      p.Title,
		},
		"section": p.Section,
		"text":    p.Text,
	}
}

// AcceptLabels keeps systems whose noul meets minNoul, plus a child choice
// when that parent was accepted. minNoul of zero accepts nothing.
func AcceptLabels(cat *knowledge.Catalog, result jev.Result, minNoul float64) []string {
	if cat == nil || minNoul <= 0 {
		return nil
	}
	var out []string
	for _, sys := range cat.TopSystems() {
		answer, ok := result.Answers[knowledge.LabelQuestionID(sys.ID)]
		if !ok || answer.Type != jev.TypeNoul || answer.Noul < minNoul {
			continue
		}
		out = append(out, sys.ID)
		leaf, ok := result.Answers[knowledge.LeafQuestionID(sys.ID)]
		if !ok || leaf.Type != jev.TypeChoice || leaf.Choice == "" || leaf.Choice == knowledge.NoneOption() {
			continue
		}
		child, ok := cat.System(leaf.Choice)
		if ok && child.Parent == sys.ID {
			out = append(out, child.ID)
		}
	}
	return out
}

func evidenceStates(questions map[string]jev.Question, result jev.Result, minNoul float64) map[string]string {
	out := make(map[string]string, len(questions))
	for id, question := range questions {
		out[id] = knowledge.StateUnknown
		if question.Type != jev.TypeNoul || minNoul <= 0 {
			continue
		}
		answer, ok := result.Answers[id]
		if ok && answer.Type == jev.TypeNoul && answer.Noul >= minNoul {
			out[id] = knowledge.StateAddressed
		}
	}
	return out
}
