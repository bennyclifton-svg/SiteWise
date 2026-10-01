package intake

import (
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// Latency paths reported to an Observer. The names match bench/budgets.json.
const (
	PathIdentity = "identity_text_extraction"
	PathHarvest  = "candidate_harvesting"
	PathRules    = "deterministic_field_rules"
	PathJev      = "jev_admission_request"
	PathCommit   = "commit_sse_enqueue"
)

// Observer receives one duration per path per filing, including failed and
// degraded runs. It is called from concurrent filings and must not block.
type Observer func(path string, d time.Duration)

// Draft is one filing before it touches the database: the rule decisions, the
// single Jev fan-out for what is still open, and the answers applied to it.
// Service.File and the calibration tool share it, so the evaluation measures
// the production decision path rather than a copy.
type Draft struct {
	catalog   Catalog
	filename  string
	text      identity.Text
	harvested []Candidate
	by        map[string]Decision
	priors    []store.NumberedDocument
	questions []builtQuestion
}

// NewDraft applies the deterministic rules to harvested candidates. users are
// the user's own decisions; nothing in the draft replaces them.
func NewDraft(cat Catalog, filename string, text identity.Text, harvested []Candidate, users map[string]Decision) *Draft {
	by := make(map[string]Decision, len(users)+8)
	for field, d := range users {
		by[field] = d
	}
	for _, result := range Decide(harvested) {
		if _, user := by[result.Field]; user {
			continue
		}
		if result.Settled {
			by[result.Field] = ruleDecision(result, BandGreen)
			continue
		}
		if result.Rule == RuleMissing {
			by[result.Field] = Decision{Field: result.Field, Band: BandBlank, DecidedBy: DecidedByRule}
		}
	}
	for _, result := range settleVocabulary(cat, filename, text) {
		if _, taken := by[result.Field]; taken {
			continue
		}
		by[result.Field] = ruleDecision(result, BandGreen)
	}
	return &Draft{catalog: cat, filename: filename, text: text, harvested: harvested, by: by}
}

// Plan scopes supersession candidates to project documents sharing a
// harvested number, then builds the questions still open. self is this
// document's id. Too many candidates leaves supersession unresolved.
func (d *Draft) Plan(project []store.NumberedDocument, self string) {
	d.priors = seriesPriors(d.harvested, project, self)
	d.questions = d.open()
	if len(d.priors)+1 > jev.MaxChoiceOptions && d.by[FieldSupersedes].DecidedBy != DecidedByUser {
		d.by[FieldSupersedes] = Decision{Field: FieldSupersedes, Band: BandBlank, DecidedBy: DecidedByRule}
	}
}

// Call is the one fan-out for this filing, or false when nothing is open.
func (d *Draft) Call() (jev.Call, bool) {
	if len(d.questions) == 0 {
		return jev.Call{}, false
	}
	return callFrom(filingStateOf(d.filename, d.text, d.harvested), d.questions), true
}

// Apply records the answers. A non-nil err is the degraded path: every open
// question goes grey and the rule values stay. It reports whether it did.
func (d *Draft) Apply(result jev.Result, err error, thresholds Thresholds) (grey bool) {
	grey = err != nil
	for _, q := range d.questions {
		if d.by[q.Field].DecidedBy == DecidedByUser {
			continue
		}
		if grey {
			d.by[q.Field] = Decision{
				Field:           q.Field,
				Band:            BandGrey,
				DecidedBy:       DecidedByJev,
				QuestionVersion: QuestionVersion,
			}
			continue
		}
		ans, ok := result.Answers[q.Field]
		d.by[q.Field] = fromAnswer(q, ans, ok, thresholds)
	}
	return grey
}

// Choice is one validated Jev selection before any threshold. Value is the
// literal the option would store; it is empty for none and new.
type Choice struct {
	Field      string
	ID         string
	Value      string
	Options    int
	Confidence *float64
}

// Choices returns the raw selections in result for the draft's questions.
// Calibration needs them whether or not a cut-off would apply them.
func (d *Draft) Choices(result jev.Result) []Choice {
	var out []Choice
	for _, q := range d.questions {
		ans, ok := result.Answers[q.Field]
		if !ok || ans.Type != jev.TypeChoice {
			continue
		}
		opt, found := lookupChoice(q, ans.Choice)
		if !found {
			continue
		}
		out = append(out, Choice{
			Field:      q.Field,
			ID:         opt.ID,
			Value:      opt.Value,
			Options:    len(q.Options),
			Confidence: copyFloat(ans.Confidence),
		})
	}
	return out
}

// Link is the prior document to link as superseded, or empty. See priorToLink.
func (d *Draft) Link(links [][2]string, self string) string {
	return priorToLink(d.by, d.priors, d.by[FieldRevision].Value, self, links)
}

// Decisions lists every field in filing order. Identity fields nothing
// decided are blank rule decisions.
func (d *Draft) Decisions() []Decision {
	d.fillBlanks()
	out := make([]Decision, 0, len(filingFields))
	for _, field := range filingFields {
		if dec, ok := d.by[field]; ok {
			out = append(out, dec)
		}
	}
	return out
}

func (d *Draft) fillBlanks() {
	for _, field := range []string{FieldNumber, FieldRevision, FieldTitle, FieldDate} {
		if _, ok := d.by[field]; !ok {
			d.by[field] = Decision{Field: field, Band: BandBlank, DecidedBy: DecidedByRule}
		}
	}
}

var filingFields = []string{FieldKind, FieldDiscipline, FieldLifecycle, FieldNumber, FieldRevision, FieldTitle, FieldDate, FieldSupersedes}

func (d *Draft) open() []builtQuestion {
	var out []builtQuestion
	cat := d.catalog
	if _, ok := d.by[FieldKind]; !ok {
		if q, ok := catalogQuestion(FieldKind, "Which kind is this document?", kindIDs(cat), kindLabels(cat)); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldDiscipline]; !ok {
		if q, ok := catalogQuestion(FieldDiscipline, "Which discipline produced this document?", disciplineIDs(cat), disciplineLabels(cat)); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldLifecycle]; !ok {
		if q, ok := catalogQuestion(FieldLifecycle, "Which lifecycle area does this document belong to?", lifecycleIDs(cat), lifecycleLabels(cat)); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldNumber]; !ok {
		if q, ok := identityQuestion(FieldNumber, numberQuestion, d.harvested); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldRevision]; !ok {
		if q, ok := identityQuestion(FieldRevision, revisionQuestion, d.harvested); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldTitle]; !ok {
		if q, ok := identityQuestion(FieldTitle, titleQuestion, d.harvested); ok {
			out = append(out, q)
		}
	}
	if _, ok := d.by[FieldSupersedes]; !ok {
		if q, ok := supersessionQuestion(d.priors); ok {
			out = append(out, q)
		}
	}
	return out
}
