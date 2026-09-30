package intake

import (
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
	QuestionVersion = "intake-1"

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

// Threshold is the calibrated cut-off for one question, version and option count.
// A nil cut-off is unknown and must not authorise an automatic action.
type Threshold struct {
	Green   *float64 `json:"green"`
	Amber   *float64 `json:"amber"`
	Options *int     `json:"options"`
}

// Thresholds is the calibration file. Supersession is its own question.
type Thresholds struct {
	QuestionVersion string               `json:"question_version"`
	Reconciliation  string               `json:"reconciliation"`
	Questions       map[string]Threshold `json:"questions"`
}

// LoadThresholds reads calibration data. Unknown cut-offs are valid; a partial
// cut-off is not, because a lone number would become a production default.
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
	for field, q := range got.Questions {
		if err := validateThreshold(field, q); err != nil {
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
	if n == 0 {
		return nil
	}
	if n != 3 {
		return fmt.Errorf("%s threshold is partial", field)
	}
	if *q.Green < 0 || *q.Green > 1 || *q.Amber < 0 || *q.Amber > 1 || *q.Amber > *q.Green {
		return fmt.Errorf("%s threshold is out of range", field)
	}
	if *q.Options < 2 || *q.Options > jev.MaxChoiceOptions {
		return fmt.Errorf("%s option shape", field)
	}
	return nil
}

// Band reports the colour for one answer. apply is false when the cut-off is
// unknown, belongs to a different option count, or the answer sits below amber.
// Choice confidence is not reused across questions or option shapes.
//
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/patterns/confidence-routing
func (t Thresholds) Band(field string, confidence *float64, options int) (band string, apply bool) {
	if t.QuestionVersion != QuestionVersion || confidence == nil {
		return BandBlank, false
	}
	q, ok := t.Questions[field]
	if !ok || q.Green == nil || q.Amber == nil || q.Options == nil || *q.Options != options {
		return BandBlank, false
	}
	switch {
	case *confidence >= *q.Green:
		return BandGreen, true
	case *confidence >= *q.Amber:
		return BandAmber, true
	default:
		return BandBlank, false
	}
}

// settleVocabulary closes kind, discipline and lifecycle when the identity
// text names exactly one catalog entry. Zero or several stay unresolved.
func settleVocabulary(cat Catalog, filename string, text identity.Text) []Result {
	haystack := filename + "\n" + runText(text)
	var out []Result
	if id, ok := onePhrase(haystack, kindPhrases(cat)); ok {
		out = append(out, Result{Field: FieldKind, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	if id, ok := onePhrase(haystack, disciplinePhrases(cat)); ok {
		out = append(out, Result{Field: FieldDiscipline, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	if id, ok := onePhrase(haystack, lifecyclePhrases(cat)); ok {
		out = append(out, Result{Field: FieldLifecycle, Settled: true, Rule: RuleUnique, Display: id, Normalized: id})
	}
	return out
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
