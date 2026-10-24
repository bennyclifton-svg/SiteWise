// Package jobs runs the background stages that follow a filing.
// Each stage is one leased job. Labeling and evidence are each one Jev
// fan-out per passage; they are not extra round trips on the filing path.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/programread"
	"sitewise/internal/store"
)

const (
	questionVersion = "knowledge.v2+" + profile.QuestionVersion
	maxPassageRunes = 2000
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
	Context    string
	Page       int
	Kind       string
	Discipline string
	Title      string
	Read       string // the document's reading setting; header routing reads it
	Labels     []string
	Parts      []profile.Part
	PartLabel  string // applied location from the label stage
}

// Worker leases background jobs for one org at a time. MinNoul is an explicit
// band for accepting a label or an evidence noul. Zero accepts nothing, so a
// missing calibration cannot become an automatic action.
type Worker struct {
	Visual  func(context.Context, store.ClaimedJob) error
	OCR     func(context.Context, string, string) error
	Store   *store.Store
	Ask     Asker
	Text    func(ctx context.Context, orgID, documentID string) (string, error)
	Source  func(context.Context, string, string) (store.DocumentSource, error)
	Catalog *knowledge.Catalog
	Lease   time.Duration
	Backoff time.Duration
	MinNoul float64
	// Since limits the worker to jobs created at or after it; zero works
	// every job, including a backlog.
	Since time.Time
	// Kinds separates text extraction from slower profile reading. Empty
	// preserves the all-stage worker used by tests and command-line tools.
	Kinds []string
	// Profile holds the profile's per-shape floors. Empty floors apply
	// nothing: readings are stored and rows stay blank.
	Profile profile.Thresholds
	// Reading decides which documents' facts the profile uses. The zero
	// policy uses every document.
	Reading profile.ReadPolicy
	// ProgramThresholds band hold-point release answers (plan 10.4). No
	// cut-off proposes nothing.
	ProgramThresholds programread.Thresholds
	profileOnce       sync.Once
	profileConfigured *store.Store
}

// Once leases and runs one full-text, label or evidence job.
func (w *Worker) Once(ctx context.Context, orgID string) error {
	lease := w.Lease
	if lease <= 0 {
		lease = 30 * time.Second
	}
	kinds := w.Kinds
	if len(kinds) == 0 {
		kinds = []string{store.JobKindFullText, store.JobKindLabel, store.JobKindEvidence, store.JobKindSpecification}
	}
	job, err := w.Store.ClaimJobSince(ctx, orgID, lease, kinds, w.Since)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(lease / 3)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if err := w.Store.RenewBackgroundLease(runCtx, job, lease.Seconds()); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	runErr := w.Perform(runCtx, job)
	cancel()
	<-stopped
	if err := runErr; err != nil {
		_ = w.Store.FailJob(ctx, job.OrgID, job.ID, job.Token, err.Error(), w.retryAfter(err), store.EventWrite{
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

// retryAfter spaces retries. An open Jev circuit fails every call until its
// cooldown passes, so an immediate retry would only spend the attempt budget.
func (w *Worker) retryAfter(err error) time.Duration {
	d := w.backoff()
	if errors.Is(err, jev.ErrCircuitOpen) && d < jev.BreakerCooldown {
		return jev.BreakerCooldown
	}
	return d
}

// Perform runs the stage without completing the lease. A second delivery of
// the same stage replaces passages and labels instead of appending them.
func (w *Worker) Perform(ctx context.Context, job store.ClaimedJob) error {
	switch job.Kind {
	case store.JobKindVisualIdentity:
		if w.Visual == nil {
			return errors.New("visual reader unavailable")
		}
		return w.Visual(ctx, job)
	case store.JobKindSpecification:
		return w.specification(ctx, job)
	case store.JobKindOCR:
		if w.OCR == nil {
			return errors.New("OCR worker unavailable")
		}
		return w.OCR(ctx, job.OrgID, job.DocumentID)
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
	if w.Source != nil {
		src, err := w.Source(ctx, job.OrgID, job.DocumentID)
		if err != nil {
			return err
		}
		return w.Store.ReplaceSource(ctx, job.OrgID, job.DocumentID, src)
	}
	if w.Text == nil {
		return errors.New("document text is unavailable")
	}
	text, err := w.Text(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return err
	}
	// Reading (label, then evidence) waits for the user's "Update project
	// profile" (store.RequestProfileRead), so filing and profiling stay apart.
	src := store.DocumentSource{Source: []store.SourcePage{{Text: text, Location: "Document"}}}
	for _, body := range SplitPassages(text) {
		src.Units = append(src.Units, store.SourceUnit{Body: body})
	}
	return w.Store.ReplaceSource(ctx, job.OrgID, job.DocumentID, src)
}

func (w *Worker) label(ctx context.Context, job store.ClaimedJob) error {
	if w.Catalog == nil {
		return errors.New("knowledge is not loaded")
	}
	passages, err := w.passages(ctx, job)
	if err != nil {
		return err
	}
	factSets := make([][]store.StoredFact, len(passages))
	err = eachPassage(ctx, len(passages), func(ctx context.Context, i int) error {
		passage := passages[i]
		if Boilerplate(passage.Passage) {
			return nil
		}
		call, cands := labelCall(w.Catalog, passage.Passage)
		if len(call.Questions) == 0 {
			return nil
		}
		result, err := w.cachedAsk(ctx, job, passage.ID, "label", call)
		if err != nil {
			return err
		}
		labels := AcceptLabels(w.Catalog, result, w.MinNoul)
		if err := w.Store.SetPassageSystems(ctx, job.OrgID, passage.ID, labels); err != nil {
			return err
		}
		readings := profile.Readings(result, call.Questions, cands, passage.Text)
		_, partLabel := w.Profile.AppliedLocation(result.Answers["source.scope"], passage.Parts)
		factSets[i] = storedFacts(passage.ID, readings, partLabel)
		return w.Store.SetSourceReading(ctx, job.OrgID, passage.ID, sourceReading(result, labels, readings, passage.Parts, w.Profile))
	})
	if err != nil {
		return err
	}
	var facts []store.StoredFact
	for _, set := range factSets {
		facts = append(facts, set...)
	}

	if err := w.profileStore().ReplaceDocumentFacts(ctx, job.OrgID, job.DocumentID, []string{"det.", "fact.", "hdr.", knowledge.DetailPrefix, knowledge.SitePrefix}, profile.QuestionVersion, facts); err != nil {
		return err
	}
	if err := w.Store.SetReadRevision(ctx, job.OrgID, job.DocumentID, profile.ReadRevision); err != nil {
		return err
	}
	// Header and compliance readings show now; systems follow evidence.
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
	factSets := make([][]store.StoredFact, len(passages))
	holds, err := w.releaseTargets(ctx, job, passages)
	if err != nil {
		return err
	}
	releases := make([][]store.HoldPointReading, len(passages))
	err = eachPassage(ctx, len(passages), func(ctx context.Context, i int) error {
		passage := passages[i]
		if passage.SkipEvidence {
			return nil
		}
		labels, err := w.Store.PassageSystems(ctx, job.OrgID, passage.ID)
		if err != nil {
			return err
		}
		passage.Labels = labels
		// Reuse the label-stage location decision only if its complete current
		// question fingerprint still matches. This is a cache read, never a
		// second model call, and avoids reinterpreting an old option after a rename.
		labelRequest, _ := labelCall(w.Catalog, passage.Passage)
		labelFingerprint, err := passageFingerprint(labelRequest)
		if err != nil {
			return err
		}
		located, found, err := w.Store.CachedPassageCall(ctx, job.OrgID, passage.ID, "label", labelFingerprint)
		if err != nil {
			return err
		}
		if found {
			_, passage.PartLabel = w.Profile.AppliedLocation(located.Answers["source.scope"], passage.Parts)
			// Evidence enriches the persisted labels before the profile commits.
			// Retrying from those outputs would change this request and miss its
			// cache. Reuse the matching label-stage input, like its location above.
			passage.Labels = AcceptLabels(w.Catalog, located, w.MinNoul)
		}
		call, ok := EvidenceCall(w.Catalog, passage.Passage)
		if !ok {
			return nil
		}
		// A program hold point's release noul joins this one fan-out; its
		// answer is split off before anything below reads the result.
		asked, target, joined := WithRelease(w.Catalog, call, passage.Passage, holds)
		result, err := w.cachedAsk(ctx, job, passage.ID, "evidence", asked)
		if err != nil {
			return err
		}
		if joined {
			var answer jev.Answer
			var answered bool
			result, answer, answered = splitRelease(result, knowledge.ReleaseQuestionID(target.InterfaceID))
			if r, ok := w.releaseReading(target, passage.ID, answer); answered && ok {
				releases[i] = append(releases[i], r)
			}
		}
		if detail, ok := DetailCall(w.Catalog, passage.Passage, DetailLeaves(w.Catalog, passage.Passage, result, w.MinNoul)); ok {
			more, err := w.cachedAsk(ctx, job, passage.ID, "evidence_detail", detail)
			if err != nil {
				return err
			}
			call, result = mergeCall(call, detail, result, more)
		}
		if err := w.Store.SetPassageEvidence(ctx, job.OrgID, passage.ID, evidenceStates(call.Questions, result, w.MinNoul)); err != nil {
			return err
		}
		readings := profile.Readings(result, call.Questions, nil, passage.Text)
		labels = append(labels, AcceptLabels(w.Catalog, result, w.MinNoul)...)
		for _, r := range readings {
			accepted := r.Confidence != nil && *r.Confidence >= 0.6
			if strings.HasSuffix(r.QuestionID, ".action") {
				accepted = w.Profile.ActionApplied(r)
			}
			if strings.HasPrefix(r.QuestionID, "sys.") && accepted {
				id := strings.TrimPrefix(r.QuestionID, "sys.")
				if at := strings.LastIndex(id, "."); at > 0 {
					labels = append(labels, id[:at])
				}
			}
		}
		if err := w.Store.SetPassageSystems(ctx, job.OrgID, passage.ID, labels); err != nil {
			return err
		}
		factSets[i] = storedFacts(passage.ID, readings, passage.PartLabel)
		var keys []string
		unresolved := append([]string{}, result.Unresolved...)
		for _, r := range readings {
			keys = append(keys, r.QuestionID)
			if r.Confidence == nil || *r.Confidence < 0.6 || (strings.HasPrefix(r.QuestionID, "sig.") && *r.Confidence < .9) || (strings.HasSuffix(r.QuestionID, ".action") && !w.Profile.ActionApplied(r)) {
				unresolved = append(unresolved, r.QuestionID)
			}
		}
		return w.Store.MergeSourceEvidence(ctx, job.OrgID, passage.ID, keys, unresolved)
	})
	if err != nil {
		return err
	}
	var facts []store.StoredFact
	for _, set := range factSets {
		facts = append(facts, set...)
	}

	if err := w.profileStore().ReplaceDocumentFacts(ctx, job.OrgID, job.DocumentID, []string{"sys.", "sig."}, profile.QuestionVersion, facts); err != nil {
		return err
	}
	if len(holds) > 0 {
		var recs []store.HoldPointReading
		for _, set := range releases {
			recs = append(recs, set...)
		}
		return w.Store.ReplaceHoldPointReadings(ctx, job.OrgID, job.DocumentID, recs)
	}
	return nil
}

// profileStore commits evidence and the code-only projection together.
func (w *Worker) profileStore() *store.Store {
	// Worker configuration is fixed before processing jobs. Reuse its immutable
	// evaluator rather than compiling the full catalogue for every document.
	w.profileOnce.Do(func() {
		cat, thresholds, reading := w.Catalog, w.Profile, w.Reading
		w.profileConfigured = w.Store.WithProfile(store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion,
			ThresholdsVersion: thresholds.Version, ReadKinds: reading.Kinds(), Compute: func(s store.ProfileSnapshot) []profile.Row {
				return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: thresholds, Read: reading}, cat)
			}})
	})
	return w.profileConfigured
}

func storedFacts(passageID string, readings []profile.Reading, partLabel string) []store.StoredFact {
	out := make([]store.StoredFact, 0, len(readings))
	for _, r := range readings {
		out = append(out, store.StoredFact{PassageID: passageID, QuestionID: r.QuestionID, Value: r.Value, Unit: r.Unit,
			Basis: r.Basis, Excerpt: r.Excerpt, Confidence: r.Confidence, DecidedBy: "jev", PartLabel: partLabel})
	}
	return out
}

func (w *Worker) passages(ctx context.Context, job store.ClaimedJob) ([]storedPassage, error) {
	parts, err := w.Store.DocumentParts(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
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
	read, err := w.Store.DocumentReadSetting(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
	out := make([]storedPassage, len(rows))
	section := ""
	metadata, err := w.Store.SourceUnits(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		// Passages carry the nearest heading above them; header routing reads it.
		if h := sectionOf(row.Body); h != "" {
			section = h
		}
		meta := metadata[row.ID]
		if meta.Section != "" {
			section = meta.Section
		}
		out[i] = storedPassage{
			ID:           row.ID,
			SkipEvidence: meta.Category == "reference" || meta.Category == "background",
			Passage: Passage{
				Parts:   parts,
				Ordinal: int(row.Ordinal),
				Text:    row.Body,
				Section: section,
				Context: meta.Context, Page: meta.Page,
				Kind:       kind,
				Discipline: discipline,
				Title:      title,
				Read:       read,
			},
		}
	}
	return out, nil
}

type storedPassage struct {
	SkipEvidence bool
	ID           string
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
	for _, paragraph := range strings.Split(text, "\n\n") {
		for _, part := range splitAddressee(strings.TrimSpace(paragraph)) {
			out = appendCut(out, part)
		}
	}
	return out
}

// addresseeLine opens a To / Client / Project / Site block, with or without
// Markdown bold around the label.
var addresseeLine = regexp.MustCompile(`(?i)^[*_\s]*(?:to|attn\.?|attention|client|principal|owner|project|site|prepared\s+for)[*_\s]*:`)

// splitAddressee starts a new passage where an addressee block follows other
// lines, so a letterhead's office address and the project's site address
// are read apart. A paragraph that begins with labels stays whole.
func splitAddressee(part string) []string {
	lines := strings.Split(part, "\n")
	for i := 1; i < len(lines); i++ {
		if addresseeLine.MatchString(lines[i]) && !addresseeLine.MatchString(lines[i-1]) {
			head := strings.TrimSpace(strings.Join(lines[:i], "\n"))
			return append([]string{head}, splitAddressee(strings.TrimSpace(strings.Join(lines[i:], "\n")))...)
		}
	}
	return []string{part}
}

// appendCut adds one passage. A very long paragraph is cut so one passage
// cannot dominate the index.
func appendCut(out []string, part string) []string {
	for part != "" {
		if utf8.RuneCountInString(part) <= maxPassageRunes {
			return append(out, part)
		}
		cut := cutRunes(part, maxPassageRunes)
		out = append(out, strings.TrimSpace(part[:cut]))
		part = strings.TrimSpace(part[cut:])
	}
	return out
}

var (
	signatureBlank = regexp.MustCompile(`_{6,}`)
	signatureWords = regexp.MustCompile(`(?i)\b(?:accepted|signed|signature|sign|date|name)\b`)
	billingHeading = regexp.MustCompile(`(?i)^[#*\s]*(?:billing|invoicing|payment terms|payment)\b`)
)

// Boilerplate reports a passage nothing in the building's profile can come
// from: a signature or acceptance block, or a fee proposal's billing terms.
// It is not sent to Jev. A contract's payment clause is read: it can state
// provisional sums, notice periods and retention.
func Boilerplate(p Passage) bool {
	if signatureBlank.MatchString(p.Text) && signatureWords.MatchString(p.Text) && len(strings.Fields(p.Text)) <= 30 {
		return true
	}
	return p.Kind == "commercial" && (billingHeading.MatchString(p.Text) || billingHeading.MatchString(p.Section))
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

// LabelCall asks independent system membership and source classification in
// one fan-out. Multiple leaves can describe the same passage.
func LabelCall(cat *knowledge.Catalog, passage Passage) jev.Call {
	call, _ := labelCall(cat, passage)
	return call
}

// labelCall also returns the profile candidates offered in the call, so the
// answers can be mapped back to verbatim values.
func labelCall(cat *knowledge.Catalog, passage Passage) (jev.Call, map[string][]profile.Candidate) {
	passage.Labels = nil
	passage.PartLabel = ""
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
			Instructions: "Using `excerpt`, is any part of this clause about " + name + "? Several systems may be present together.",
			Criteria: map[string]string{
				"true":  knowledge.Describe(sys, sys.ID),
				"false": "None of this category is discussed. Exclusions alone: " + knowledge.ExcludesText(sys) + " Mention of another system alongside this category does not make the answer false.",
			},
		}
	}

	for id, q := range sourceQuestions(passage.Parts) {
		questions[id] = q
	}
	// Profile questions read the same passage, so they join this request
	// instead of adding a round trip (https://docs.typesafe.ai/patterns/fan-out).
	extra, candidates := profile.LabelQuestions(passage.Info(), profile.HarvestOwn(passageExcerpt(passage), passage.Text, cat), cat)
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
	return profile.PassageInfo{Kind: p.Kind, Section: p.Section, Ordinal: p.Ordinal, Text: p.Text, Read: p.Read}
}

// EvidenceCall is one fan-out of the knowledge nouls that match the passage
// labels. It is a different state from labeling and is not asked in the
// foreground filing request.
func EvidenceCall(cat *knowledge.Catalog, passage Passage) (jev.Call, bool) {
	if cat == nil {
		return jev.Call{}, false
	}
	// Literal service names route speculative questions; they do not decide
	// inclusion. A low family score must not hide an explicit exclusion.
	for _, route := range explicitEvidenceRoutes {
		if route.re.MatchString(passageExcerpt(passage)) {
			passage.Labels = append(passage.Labels, route.family)
		}
	}
	if len(passage.Labels) == 0 {
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
			Criteria:     noulCriteria(q.Criteria),
		}
	}
	// Ask all leaves of every relevant family together. A single-choice or
	// narrow first-pass label must not hide a second service in the clause.
	leafIDs := []string{}
	seen := map[string]bool{}
	for _, id := range passage.Labels {
		sys, ok := cat.System(id)
		if !ok {
			continue
		}
		parent := sys.ID
		if sys.Parent != "" {
			parent = sys.Parent
		}
		for _, child := range cat.Children(parent) {
			if !seen[child.ID] {
				seen[child.ID] = true
				leafIDs = append(leafIDs, child.ID)
			}
		}
	}
	for _, id := range leafIDs {
		child, _ := cat.System(id)
		questions[knowledge.LabelQuestionID(id)] = jev.Question{Type: jev.TypeNoul, Instructions: "Using `excerpt`, does this clause concern any of " + child.Label + "? A clause that says it is excluded, not used or by others still concerns it; use `section` to identify the subject of a short clause.", Criteria: map[string]string{"true": knowledge.Describe(child, id) + " Includes a clause that excludes it or says it is not used.", "false": "None of this category is discussed, included or excluded. Other systems can be discussed alongside it without making the answer false."}}
	}
	// Signals follow the labelled passage runs_on contract; no extra model call.
	for _, q := range cat.SignalQuestions(passage.Labels) {
		questions[q.ID] = jev.Question{Type: jev.TypeNoul, Instructions: q.Instructions, Criteria: noulCriteria(q.Criteria)}
	}
	// Leaf detail (presence, provider, action) waits for DetailCall. Asking it
	// for every leaf of several families exceeded the request budget.
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

// DetailCall asks presence, provider and action only for the leaves the
// first evidence pass found, so a clause spanning several families stays
// within the request budget without dropping any family. It is a second,
// background fan-out per passage; accuracy outranks reading time here.
func DetailCall(cat *knowledge.Catalog, passage Passage, leaves []string) (jev.Call, bool) {
	questions := profile.EvidenceQuestions(leaves, cat)
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

// DetailLeaves are the leaves the passage was already labelled with plus
// those the first evidence pass accepted, once each, in order.
func DetailLeaves(cat *knowledge.Catalog, passage Passage, first jev.Result, minNoul float64) []string {
	var out []string
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, passage.Labels...), AcceptLabels(cat, first, minNoul)...) {
		if sys, ok := cat.System(id); ok && sys.Parent != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// mergeCall joins both evidence passes so the passage's readings, states and
// unresolved keys are recorded exactly as one call's would be.
func mergeCall(first, detail jev.Call, a, b jev.Result) (jev.Call, jev.Result) {
	questions := make(map[string]jev.Question, len(first.Questions)+len(detail.Questions))
	answers := make(map[string]jev.Answer, len(a.Answers)+len(b.Answers))
	for id, q := range first.Questions {
		questions[id] = q
	}
	for id, q := range detail.Questions {
		questions[id] = q
	}
	for id, x := range a.Answers {
		answers[id] = x
	}
	for id, x := range b.Answers {
		answers[id] = x
	}
	first.Questions = questions
	a.Answers = answers
	a.Unresolved = append(append([]string{}, a.Unresolved...), b.Unresolved...)
	return first, a
}

var explicitEvidenceRoutes = []struct {
	family string
	re     *regexp.Regexp
}{
	{"hydraulic", regexp.MustCompile(`(?i)\brain\s?water\b|\bnon[ -]?potable\b|\bhot water\b|\bnatural gas\b`)},
	{"site", regexp.MustCompile(`(?i)\b(?:recessed|loading|on-grade) docks?\b|\bloading bay\b`)},
	{"mechanical", regexp.MustCompile(`(?i)\b(?:battery|MHE) charging\b`)},
}

const knowledgeNoul = "noul"

// noulCriteria gives a noul's criteria string keys. YAML reads `true:` and
// `false:` as boolean keys, which cannot be encoded as a JSON object, so the
// client would reject the whole call (https://docs.typesafe.ai/api).
func noulCriteria(c any) any {
	switch m := c.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[fmt.Sprint(k)] = strings.TrimSpace(fmt.Sprint(v))
		}
		return out
	case map[bool]any:
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[fmt.Sprint(k)] = strings.TrimSpace(fmt.Sprint(v))
		}
		return out
	}
	return c
}

func passageState(p Passage) map[string]any {
	return map[string]any{
		"document": map[string]string{
			"kind":       p.Kind,
			"discipline": p.Discipline,
			"title":      p.Title,
		},
		"section":         p.Section,
		"excerpt":         passageExcerpt(p),
		"context":         p.Context,
		"page":            p.Page,
		"system_families": p.Labels,
		"text":            p.Text,
		"site_parts":      profile.OrderedLocationParts(p.Parts),
		"applied_part":    p.PartLabel,
	}
}

func passageExcerpt(p Passage) string {
	return strings.TrimSpace(p.Section + "\n" + p.Context + "\n" + p.Text)
}

// AcceptLabels keeps independently supported leaves and their parents.
// A low parent answer cannot veto a specific leaf. Zero accepts nothing.
func AcceptLabels(cat *knowledge.Catalog, result jev.Result, minNoul float64) []string {
	if cat == nil || minNoul <= 0 {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, sys := range cat.TopSystems() {
		a := result.Answers[knowledge.LabelQuestionID(sys.ID)]
		if a.Type == jev.TypeNoul && a.Noul >= minNoul {
			add(sys.ID)
		}
	}
	for _, sys := range cat.Leaves() {
		a := result.Answers[knowledge.LabelQuestionID(sys.ID)]
		if a.Type == jev.TypeNoul && a.Noul >= minNoul {
			add(sys.Parent)
			add(sys.ID)
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
		floor := minNoul
		if strings.HasPrefix(id, "sig:") {
			floor = .9
		}
		answer, ok := result.Answers[id]
		if ok && answer.Type == jev.TypeNoul && answer.Noul >= floor {
			out[id] = knowledge.StateAddressed
		}
	}
	return out
}

// Eight background requests leave the client's foreground reserve untouched.
// Checkpoints make cancellation and process restarts cheap to resume.
// A failure stops dispatch but lets calls in flight finish: cancelling them
// would discard paid answers, and a cancelled half-open probe would keep the
// Jev circuit open on every retry.
func eachPassage(ctx context.Context, n int, fn func(context.Context, int) error) error {
	stop, cancel := context.WithCancel(ctx)
	defer cancel()
	queue := make(chan int)
	var wg sync.WaitGroup
	var once sync.Once
	var first error
	for j := 0; j < 8; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				if stop.Err() != nil {
					return
				}
				if err := fn(ctx, i); err != nil {
					once.Do(func() { first = err; cancel() })
					return
				}
			}
		}()
	}
dispatch:
	for i := 0; i < n; i++ {
		select {
		case <-stop.Done():
			break dispatch
		case queue <- i:
		}
	}
	close(queue)
	wg.Wait()
	if first != nil {
		return first
	}
	return ctx.Err()
}
