package intake

import (
	"context"
	"errors"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// Asker is the one System One call a filing is allowed to make.
type Asker interface {
	Ask(context.Context, jev.Call) (jev.Result, error)
}

// Service files one document. Rules run first. Anything still unresolved is
// one Jev fan-out, then a single commit.
//
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/concepts/how-to-build-with-system-one
type Service struct {
	store      *store.Store
	jev        Asker
	catalog    Catalog
	thresholds Thresholds
	observer   Observer
}

// Filed is the durable result of one intake pass.
type Filed struct {
	Status       string
	Number       string
	Revision     string
	SupersedesID string
	Decisions    []Decision
}

// NewService binds the store, the Jev client and the calibration data.
func NewService(db *store.Store, ask Asker, cat Catalog, thresholds Thresholds) (*Service, error) {
	if db == nil || ask == nil {
		return nil, errors.New("filing service is not configured")
	}
	if thresholds.QuestionVersion != QuestionVersion {
		return nil, errors.New("threshold question version")
	}
	return &Service{store: db, jev: ask, catalog: cat, thresholds: thresholds}, nil
}

// File harvests identity, asks Jev once when a field is still open, and commits
// the decisions. A timeout keeps rule values and marks the unanswered fields grey.
func (s *Service) File(ctx context.Context, orgID, documentID string, text identity.Text) (Filed, error) {
	return s.file(ctx, orgID, documentID, text, false)
}

func (s *Service) file(ctx context.Context, orgID, documentID string, text identity.Text, missingOnly bool) (Filed, error) {
	doc, err := s.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return Filed{}, err
	}
	if doc.Status == store.StatusNotFiled {
		return Filed{}, errors.New("document is not filed")
	}
	if missingOnly && (doc.Status != store.StatusFiled || !store.OCRDetailsActive(doc.Reason)) {
		return Filed{}, store.ErrNotFound
	}
	if doc.Status != store.StatusPending && !missingOnly {
		return s.load(ctx, orgID, documentID)
	}

	existing, err := s.store.DocumentDecisions(ctx, orgID, documentID)
	if err != nil {
		return Filed{}, err
	}
	versions := make(map[string]int64, len(existing))
	users := map[string]Decision{}
	for _, row := range existing {
		versions[row.Field] = row.Version
		if row.DecidedBy == DecidedByUser || (missingOnly && row.Value != "") {
			users[row.Field] = decisionFromStored(row)
		}
	}

	start := time.Now()
	harvested := Harvest(doc.Filename, text)
	s.observe(PathHarvest, time.Since(start))
	start = time.Now()
	draft := NewDraft(s.catalog, doc.Filename, text, harvested, users)
	s.observe(PathRules, time.Since(start))

	docs, err := s.store.ProjectDocuments(ctx, orgID, doc.ProjectID)
	if err != nil {
		return Filed{}, err
	}
	draft.Plan(docs, documentID)
	if text.OCR {
		draft.Plan(nil, documentID)
	}

	grey := false
	if call, ok := draft.Call(); ok {
		start = time.Now()
		if text.OCR {
			call.Priority = jev.PriorityBackground
		}
		result, err := s.jev.Ask(ctx, call)
		s.observe(PathJev, time.Since(start))
		if err != nil && ctx.Err() != nil {
			return Filed{}, ctx.Err()
		}
		grey = draft.Apply(result, err, s.thresholds)
	}

	links, err := s.store.OrgSupersessions(ctx, orgID)
	if err != nil {
		return Filed{}, err
	}
	priorID := draft.Link(links, documentID)
	draft.fillBlanks()
	writes := decisionWrites(draft.by, versions)
	if text.OCR {
		priorID, grey = "", false // OCR cannot establish supersession or a later green retry.
		filtered := writes[:0]
		for _, d := range writes {
			if d.Field == FieldSupersedes {
				continue
			}
			if d.Band == BandGreen {
				d.Band = BandAmber
			}
			d.QuestionVersion = "ocr-tesseract-v1+" + QuestionVersion
			filtered = append(filtered, d)
		}
		writes = filtered
	}
	start = time.Now()
	if missingOnly {
		out, err := s.store.CommitMissingDetails(ctx, orgID, documentID, writes)
		s.observe(PathCommit, time.Since(start))
		return filedFrom(out), err
	}
	out, err := s.store.CommitFiling(ctx, orgID, documentID, store.CommitFiling{
		OCR:       text.OCR,
		Decisions: writes,
		PDFPages:  text.PageCount,
		PriorID:   priorID,
		RetryJev:  grey,
	})
	s.observe(PathCommit, time.Since(start))
	if err != nil {
		return Filed{}, err
	}
	return filedFrom(out), nil
}

// Observe reports per-path latency to o. Set it before the first filing.
func (s *Service) Observe(o Observer) {
	s.observer = o
}

func (s *Service) observe(path string, d time.Duration) {
	if s.observer != nil {
		s.observer(path, d)
	}
}

// Correct stores a user value. An in-flight filing that answers the same field
// keeps this value.
func (s *Service) Correct(ctx context.Context, orgID, documentID, field, value string) error {
	if !knownField(field) {
		return errors.New("unknown field")
	}
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return errors.New("correction is too long")
	}
	return s.store.CorrectDecision(ctx, orgID, documentID, field, value)
}

func (s *Service) load(ctx context.Context, orgID, documentID string) (Filed, error) {
	doc, err := s.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return Filed{}, err
	}
	rows, err := s.store.DocumentDecisions(ctx, orgID, documentID)
	if err != nil {
		return Filed{}, err
	}
	out := Filed{
		Status:       doc.Status,
		Number:       doc.Number,
		Revision:     doc.Revision,
		SupersedesID: doc.SupersedesID,
		Decisions:    make([]Decision, len(rows)),
	}
	for i, row := range rows {
		out.Decisions[i] = decisionFromStored(row)
	}
	return out, nil
}

func ruleDecision(result Result, band string) Decision {
	return Decision{
		Field:     result.Field,
		Value:     result.Display,
		Band:      band,
		DecidedBy: DecidedByRule,
	}
}

func fromAnswer(q builtQuestion, ans jev.Answer, ok bool, thresholds Thresholds) Decision {
	d := Decision{
		Field:           q.Field,
		Band:            BandBlank,
		DecidedBy:       DecidedByJev,
		QuestionVersion: QuestionVersion,
	}
	if !ok || ans.Type != jev.TypeChoice {
		return d
	}
	opt, found := lookupChoice(q, ans.Choice)
	if !found {
		return d
	}
	band, apply := thresholds.Band(q.Field, ans.Confidence, len(q.Options))
	d.Confidence = copyFloat(ans.Confidence)
	if !apply {
		return d
	}
	d.Band = band
	d.Value = opt.Value
	return d
}

// priorToLink returns a prior document only when supersession is in its green
// band, the selected number matches that prior, and code can place this
// revision strictly later in the same series. Any other case keeps both
// documents and does not write the link.
func priorToLink(by map[string]Decision, priors []store.NumberedDocument, revision, self string, links [][2]string) string {
	d, ok := by[FieldSupersedes]
	if !ok || d.Band != BandGreen || d.Value == "" || d.DecidedBy != DecidedByJev {
		return ""
	}
	refuse := func() string {
		d.Band = BandBlank
		by[FieldSupersedes] = d
		return ""
	}
	if d.Value == self || linkWouldCycle(links, self, d.Value) {
		return refuse()
	}
	number := by[FieldNumber]
	if number.Value == "" || number.Band == BandBlank || number.Band == BandGrey {
		return refuse()
	}
	var prior *store.NumberedDocument
	for i := range priors {
		if priors[i].ID == d.Value {
			prior = &priors[i]
			break
		}
	}
	if prior == nil || normalizeNumber(prior.Number) != normalizeNumber(number.Value) {
		return refuse()
	}
	cmp, comparable := Compare(prior.Revision, revision)
	if !comparable || cmp != -1 {
		return refuse()
	}
	return prior.ID
}

func linkWouldCycle(links [][2]string, documentID, priorID string) bool {
	if documentID == "" || priorID == "" || documentID == priorID {
		return true
	}
	next := make(map[string]string, len(links))
	for _, link := range links {
		if link[0] == documentID {
			return true
		}
		next[link[0]] = link[1]
	}
	seen := map[string]struct{}{}
	cur := priorID
	for {
		if cur == documentID {
			return true
		}
		if _, ok := seen[cur]; ok {
			return true
		}
		seen[cur] = struct{}{}
		step, ok := next[cur]
		if !ok {
			return false
		}
		cur = step
	}
}

func seriesPriors(harvested []Candidate, docs []store.NumberedDocument, self string) []store.NumberedDocument {
	want := map[string]struct{}{}
	for _, c := range harvested {
		if c.Field == FieldNumber && c.Normalized != "" {
			want[c.Normalized] = struct{}{}
		}
	}
	if len(want) == 0 {
		return nil
	}
	var out []store.NumberedDocument
	for _, doc := range docs {
		if doc.ID == self || doc.Number == "" {
			continue
		}
		if _, ok := want[normalizeNumber(doc.Number)]; ok {
			out = append(out, doc)
		}
	}
	return out
}

func decisionWrites(by map[string]Decision, versions map[string]int64) []store.DecisionWrite {
	fields := []string{FieldKind, FieldDiscipline, FieldLifecycle, FieldNumber, FieldRevision, FieldTitle, FieldDate, FieldSupersedes}
	out := make([]store.DecisionWrite, 0, len(fields))
	for _, field := range fields {
		d, ok := by[field]
		if !ok || d.DecidedBy == DecidedByUser {
			continue
		}
		out = append(out, store.DecisionWrite{
			Field:           d.Field,
			Value:           d.Value,
			Band:            d.Band,
			DecidedBy:       d.DecidedBy,
			QuestionVersion: d.QuestionVersion,
			Confidence:      copyFloat(d.Confidence),
			ExpectedVersion: versions[field],
		})
	}
	return out
}

func decisionFromStored(row store.StoredDecision) Decision {
	return Decision{
		Field:           row.Field,
		Value:           row.Value,
		Band:            row.Band,
		DecidedBy:       row.DecidedBy,
		QuestionVersion: row.QuestionVersion,
		Confidence:      copyFloat(row.Confidence),
	}
}

func filedFrom(out store.FilingOutcome) Filed {
	filed := Filed{
		Status:       out.Status,
		Number:       out.Number,
		Revision:     out.Revision,
		SupersedesID: out.SupersedesID,
		Decisions:    make([]Decision, len(out.Decisions)),
	}
	for i, row := range out.Decisions {
		filed.Decisions[i] = decisionFromStored(row)
	}
	return filed
}

func copyFloat(v *float64) *float64 {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

func kindIDs(cat Catalog) []string {
	out := make([]string, len(cat.Kinds))
	for i, k := range cat.Kinds {
		out[i] = k.ID
	}
	return out
}

func kindLabels(cat Catalog) []string {
	out := make([]string, len(cat.Kinds))
	for i, k := range cat.Kinds {
		out[i] = k.Label
	}
	return out
}

func disciplineIDs(cat Catalog) []string {
	out := make([]string, len(cat.Disciplines))
	for i, d := range cat.Disciplines {
		out[i] = d.ID
	}
	return out
}

func disciplineLabels(cat Catalog) []string {
	out := make([]string, len(cat.Disciplines))
	for i, d := range cat.Disciplines {
		out[i] = d.Label
	}
	return out
}

func lifecycleIDs(cat Catalog) []string {
	out := make([]string, len(cat.Lifecycle))
	for i, area := range cat.Lifecycle {
		out[i] = area.ID
	}
	return out
}

func lifecycleLabels(cat Catalog) []string {
	out := make([]string, len(cat.Lifecycle))
	for i, area := range cat.Lifecycle {
		out[i] = area.Label
	}
	return out
}
