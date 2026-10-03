package intake

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
)

const (
	FieldKind       = "kind"
	FieldDiscipline = "discipline"
	FieldLifecycle  = "lifecycle"
	FieldSupersedes = "supersedes"

	BandGreen = "green"
	BandAmber = "amber"
	BandBlank = "blank"
	BandGrey  = "grey"

	DecidedByRule = "rule"
	DecidedByJev  = "jev"
	DecidedByUser = "user"

	// QuestionVersion is the intake question map this process asks.
	// Thresholds calibrated for another version do not apply.
	QuestionVersion = "intake-75"

	choiceNone = "none"
	choiceNew  = "new"
)

// Decision is one filed field. An empty Value was not applied. Confidence is
// set only when Jev returned a finite choice confidence.
type Decision struct {
	Field           string
	Value           string
	Band            string
	DecidedBy       string
	QuestionVersion string
	Confidence      *float64
}

// Threshold is the calibrated cut-off for one question, version and option
// shape. Options is the smallest option count it covers and MaxOptions the
// largest; a nil MaxOptions covers Options exactly. A nil cut-off is unknown
// and must not authorise an automatic action. N is the number of calibration
// answers behind the fit; it is evidence, not a setting.
type Threshold struct {
	Options    *int     `json:"options"`
	MaxOptions *int     `json:"max_options,omitempty"`
	Green      *float64 `json:"green"`
	Amber      *float64 `json:"amber"`
	N          int      `json:"n,omitempty"`
}

// Thresholds is the calibration file. Supersession is its own question. A
// question with no entry, or no entry covering an option count, is unknown.
type Thresholds struct {
	QuestionVersion string                 `json:"question_version"`
	Reconciliation  string                 `json:"reconciliation"`
	Fit             json.RawMessage        `json:"fit,omitempty"`
	Questions       map[string][]Threshold `json:"questions"`
}

// LoadThresholds reads calibration data. A calibrated amber-only band may
// suggest review values when validation does not support automatic green.
func LoadThresholds(dir string) (Thresholds, error) {
	var got Thresholds
	if err := readJSON(filepath.Join(dir, "thresholds.json"), &got); err != nil {
		return Thresholds{}, err
	}
	if got.QuestionVersion != QuestionVersion {
		return Thresholds{}, fmt.Errorf("threshold version %q", got.QuestionVersion)
	}
	if strings.TrimSpace(got.Reconciliation) == "" {
		return Thresholds{}, fmt.Errorf("threshold reconciliation is missing")
	}
	for field, list := range got.Questions {
		for _, q := range list {
			if err := validateThreshold(field, q); err != nil {
				return Thresholds{}, err
			}
		}
		if err := validateShapes(field, list); err != nil {
			return Thresholds{}, err
		}
	}
	return got, nil
}

func validateThreshold(field string, q Threshold) error {
	n := 0
	if q.Green != nil {
		n++
	}
	if q.Amber != nil {
		n++
	}
	if q.Options != nil {
		n++
	}
	if n == 0 && q.MaxOptions == nil {
		return nil
	}
	amberOnly := q.Green == nil && q.Amber != nil && q.Options != nil && q.N > 0
	if n != 3 && !amberOnly {
		return fmt.Errorf("%s threshold is partial", field)
	}
	if *q.Amber < 0 || *q.Amber > 1 || (q.Green != nil && (*q.Green < 0 || *q.Green > 1 || *q.Amber > *q.Green)) {
		return fmt.Errorf("%s threshold is out of range", field)
	}
	lo, hi := q.shape()
	if lo < 2 || hi > jev.MaxChoiceOptions || hi < lo {
		return fmt.Errorf("%s option shape", field)
	}
	return nil
}

// validateShapes rejects two cut-offs that cover the same option count.
func validateShapes(field string, list []Threshold) error {
	for i := range list {
		if list[i].Options == nil {
			continue
		}
		lo, hi := list[i].shape()
		for j := i + 1; j < len(list); j++ {
			if list[j].Options == nil {
				continue
			}
			lo2, hi2 := list[j].shape()
			if lo <= hi2 && lo2 <= hi {
				return fmt.Errorf("%s option shapes overlap", field)
			}
		}
	}
	return nil
}

func (q Threshold) shape() (lo, hi int) {
	lo = *q.Options
	hi = lo
	if q.MaxOptions != nil {
		hi = *q.MaxOptions
	}
	return lo, hi
}

// Band reports the colour for one answer. apply is false when the cut-off is
// unknown, no cut-off covers this option count, or the answer sits below
// amber. Choice confidence is not reused across questions or option shapes.
//
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/patterns/confidence-routing
func (t Thresholds) Band(field string, confidence *float64, options int) (band string, apply bool) {
	if t.QuestionVersion != QuestionVersion || confidence == nil {
		return BandBlank, false
	}
	for _, q := range t.Questions[field] {
		if q.Amber == nil || q.Options == nil {
			continue
		}
		if lo, hi := q.shape(); options < lo || options > hi {
			continue
		}
		switch {
		case q.Green != nil && *confidence >= *q.Green:
			return BandGreen, true
		case *confidence >= *q.Amber:
			return BandAmber, true
		default:
			return BandBlank, false
		}
	}
	return BandBlank, false
}

// Only identity assertions can settle vocabulary. A mention in body prose
// (BCA compliance, contract terms, a schedule reference) is not authorship or kind.
func settleVocabulary(cat Catalog, filename string, text identity.Text) []Result {
	var out []Result
	kind, kindKnown := onePhrase(vocabularyEvidence("", text, "Kind", kindPhrases(cat)), kindPhrases(cat))
	if kindKnown {
		id := kind
		out = append(out, Result{Field: FieldKind, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	if id, ok := onePhrase(vocabularyEvidence("", text, "Discipline", disciplinePhrases(cat)), disciplinePhrases(cat)); ok {
		out = append(out, Result{Field: FieldDiscipline, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	// Lifecycle is filing purpose, not the issue stamp. Technical drawings
	// remain Design even when issued for construction (owner decision).
	if (kindKnown && kind == "drawing") || hasDrawingIdentity(text) {
		out = append(out, Result{Field: FieldLifecycle, Settled: true, Rule: RuleUnique, Display: "design", Normalized: "design"})
	} else if id, ok := explicitLifecycle(cat, text); ok {
		out = append(out, Result{Field: FieldLifecycle, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	return out
}

func vocabularyEvidence(filename string, text identity.Text, caption string, phrases map[string][]string) string {
	lines := []string{filename}
	for _, run := range text.Runs {
		s := strings.TrimSpace(run.Text)
		for _, list := range phrases {
			for _, phrase := range list {
				if strings.EqualFold(s, caption+": "+phrase) || (caption == "Kind" && strings.EqualFold(s, "Drawing") && strings.EqualFold(phrase, "Drawing")) {
					lines = append(lines, s)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func explicitLifecycle(cat Catalog, text identity.Text) (string, bool) {
	var lines []string
	for _, run := range text.Runs {
		s := strings.TrimSpace(run.Text)
		for _, area := range cat.Lifecycle {
			if strings.EqualFold(s, "Lifecycle: "+area.Label) {
				lines = append(lines, s)
			}
		}
	}
	return onePhrase(strings.Join(lines, "\n"), lifecyclePhrases(cat))
}

func hasDrawingIdentity(text identity.Text) bool {
	for _, run := range text.Runs {
		s := strings.ToLower(strings.TrimSpace(run.Text))
		for _, caption := range []string{"drawing no", "drawing number", "dwg no", "drg no", "project number/drawing number"} {
			if strings.TrimRight(s, ":.") == caption || strings.HasPrefix(s, caption+" ") {
				return true
			}
		}
	}
	return false
}

func kindPhrases(cat Catalog) map[string][]string {
	out := make(map[string][]string, len(cat.Kinds))
	for _, k := range cat.Kinds {
		out[k.ID] = []string{k.Label}
	}
	return out
}

func disciplinePhrases(cat Catalog) map[string][]string {
	out := make(map[string][]string, len(cat.Disciplines))
	for _, d := range cat.Disciplines {
		phrases := make([]string, 0, 1+len(d.Aliases))
		phrases = append(phrases, d.Label)
		phrases = append(phrases, d.Aliases...)
		out[d.ID] = phrases
	}
	return out
}

func lifecyclePhrases(cat Catalog) map[string][]string {
	out := make(map[string][]string, len(cat.Lifecycle))
	for _, area := range cat.Lifecycle {
		out[area.ID] = []string{area.Label}
	}
	return out
}

func onePhrase(haystack string, phrases map[string][]string) (string, bool) {
	var hit string
	for id, list := range phrases {
		if !anyPhrase(haystack, list) {
			continue
		}
		if hit != "" {
			return "", false
		}
		hit = id
	}
	return hit, hit != ""
}

func anyPhrase(haystack string, phrases []string) bool {
	for _, phrase := range phrases {
		if containsPhrase(haystack, phrase) {
			return true
		}
	}
	return false
}

func containsPhrase(haystack, phrase string) bool {
	phrase = strings.TrimSpace(phrase)
	if len(phrase) < 2 {
		return false
	}
	h := strings.ToLower(haystack)
	p := strings.ToLower(phrase)
	from := 0
	for from < len(h) {
		i := strings.Index(h[from:], p)
		if i < 0 {
			return false
		}
		i += from
		end := i + len(p)
		before := i == 0 || !isTokenByte(h[i-1])
		after := end >= len(h) || !isTokenByte(h[end])
		if before && after {
			return true
		}
		from = i + 1
	}
	return false
}

func isTokenByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func runText(text identity.Text) string {
	var b strings.Builder
	for _, run := range text.Runs {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(run.Text)
	}
	return b.String()
}

func knownField(field string) bool {
	switch field {
	case FieldKind, FieldDiscipline, FieldLifecycle, FieldNumber, FieldRevision, FieldTitle, FieldDate, FieldSupersedes:
		return true
	default:
		return false
	}
}
