package program

import "sitewise/internal/knowledge"

// The program no longer asks Profile questions (plan 7.3). Flood, bushfire
// and every other determinant, condition and planning value are answered on
// the Profile. The strip asks only for a start date, which is not a Question.

// MaxQuestionRegenerations remains so a caller that capped in-memory
// regenerations still compiles. The program does not regenerate to score
// a question.
const MaxQuestionRegenerations = 32

// Question kinds say which Profile path an answer is written through.
const (
	QuestionDeterminant = "determinant"
	QuestionWorkType    = "work_type"
	QuestionCondition   = "condition"
	QuestionPlanning    = "planning"
)

// Question is a Profile question the program used to ask. The program
// returns none. The answer route still accepts this shape.
type Question struct {
	Key           string   `json:"key"`
	Kind          string   `json:"kind"`
	Label         string   `json:"label"`
	Options       []string `json:"options"`
	Values        []string `json:"values"`
	Number        bool     `json:"number,omitempty"`
	NarrowsByDays int      `json:"narrows_by_days"`
}

// NotSure is the boolean option that used to dismiss a Profile question.
const NotSure = "Not sure"

// OneQuestion returns no question. Profile facts are answered on the
// Profile. A missing start date is offered separately and is not a Question.
func OneQuestion(_ *knowledge.Catalog, _ GenInput, _ map[string]bool) (*Question, error) {
	return nil, nil
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
