package intake

import (
	"sort"
	"strings"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// choice is one option offered to Jev. ID is the answer token, Value is the
// literal stored when the option is applied, and Source is where it was found.
type choice struct {
	ID     string `json:"id"`
	Value  string `json:"value"`
	Source string `json:"source"`
	Label  string `json:"-"`
}

type builtQuestion struct {
	Field        string
	Instructions string
	Options      []choice
}

type filingState struct {
	Filename   string   `json:"filename"`
	TitleBlock string   `json:"title_block,omitempty"`
	FirstPage  string   `json:"first_page,omitempty"`
	Headings   []string `json:"headings,omitempty"`
	Candidates []choice `json:"candidates"`
}

func filingStateOf(filename string, text identity.Text, harvested []Candidate) filingState {
	state := filingState{
		Filename:   filename,
		TitleBlock: clip(labeledText(harvested), 1000),
		FirstPage:  clip(runText(text), 2000),
		Candidates: harvestedChoices(harvested),
	}
	for _, run := range text.Runs {
		if !run.Source.Heading {
			continue
		}
		line := strings.TrimSpace(run.Text)
		if line == "" {
			continue
		}
		state.Headings = append(state.Headings, clip(line, 200))
		if len(state.Headings) == 20 {
			break
		}
	}
	return state
}

func harvestedChoices(harvested []Candidate) []choice {
	out := make([]choice, 0, len(harvested))
	for _, c := range harvested {
		if c.Normalized == "" {
			continue
		}
		out = append(out, choice{
			ID:     optionID(c.Normalized),
			Value:  c.Display,
			Source: c.Provenance.Origin,
		})
		if len(out) == 64 {
			break
		}
	}
	return out
}

func labeledText(harvested []Candidate) string {
	var b strings.Builder
	for _, c := range harvested {
		if !c.Provenance.Labeled {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(c.Display)
	}
	return b.String()
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func optionID(normalized string) string {
	switch normalized {
	case "", choiceNone, choiceNew:
		return "candidate:" + normalized
	default:
		return normalized
	}
}

func identityQuestion(field, instructions string, harvested []Candidate) (builtQuestion, bool) {
	opts := identityOptions(harvested, field)
	if len(opts) < 2 {
		return builtQuestion{}, false
	}
	return builtQuestion{Field: field, Instructions: instructions, Options: opts}, true
}

func identityOptions(harvested []Candidate, field string) []choice {
	groups := map[string][]Candidate{}
	var order []string
	for _, c := range harvested {
		if c.Field != field || c.Normalized == "" {
			continue
		}
		if _, ok := groups[c.Normalized]; !ok {
			order = append(order, c.Normalized)
		}
		groups[c.Normalized] = append(groups[c.Normalized], c)
	}
	sort.Strings(order)
	opts := make([]choice, 0, len(order)+1)
	for _, key := range order {
		chosen := prefer(groups[key])
		opts = append(opts, choice{
			ID:     optionID(key),
			Value:  chosen.Display,
			Source: chosen.Provenance.Origin,
			Label:  chosen.Display,
		})
	}
	opts = append(opts, choice{ID: choiceNone, Label: "none", Source: "choice"})
	return opts
}

func catalogQuestion(field, instructions string, ids, labels []string) (builtQuestion, bool) {
	if len(ids) == 0 || len(ids) != len(labels) || len(ids) > jev.MaxChoiceOptions {
		return builtQuestion{}, false
	}
	opts := make([]choice, len(ids))
	for i := range ids {
		opts[i] = choice{ID: ids[i], Value: ids[i], Source: "catalog", Label: labels[i]}
	}
	return builtQuestion{Field: field, Instructions: instructions, Options: opts}, true
}

func supersessionQuestion(priors []store.NumberedDocument) (builtQuestion, bool) {
	if len(priors) == 0 || len(priors)+1 > jev.MaxChoiceOptions {
		return builtQuestion{}, false
	}
	sorted := append([]store.NumberedDocument(nil), priors...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	opts := make([]choice, 0, len(sorted)+1)
	for _, prior := range sorted {
		label := strings.TrimSpace(prior.Number + " " + prior.Revision)
		if label == "" {
			label = prior.ID
		}
		opts = append(opts, choice{ID: prior.ID, Value: prior.ID, Source: "project", Label: label})
	}
	opts = append(opts, choice{ID: choiceNew, Label: "new", Source: "choice"})
	return builtQuestion{
		Field:        FieldSupersedes,
		Instructions: "Which earlier filing of this document number does this one replace? Choose new if it does not replace one of the candidates. Do not order revisions or compare dates.",
		Options:      opts,
	}, true
}

func callFrom(state filingState, questions []builtQuestion) jev.Call {
	out := make(map[string]jev.Question, len(questions))
	for _, q := range questions {
		criteria := make(map[string]string, len(q.Options))
		for _, opt := range q.Options {
			label := opt.Label
			if label == "" {
				label = opt.ID
			}
			criteria[opt.ID] = label
		}
		out[q.Field] = jev.Question{
			Type:         jev.TypeChoice,
			Instructions: q.Instructions,
			Criteria:     criteria,
		}
	}
	return jev.Call{
		State:           state,
		Questions:       out,
		QuestionVersion: QuestionVersion,
		Priority:        jev.PriorityInteractive,
	}
}

func lookupChoice(q builtQuestion, id string) (choice, bool) {
	for _, opt := range q.Options {
		if opt.ID == id {
			return opt, true
		}
	}
	return choice{}, false
}
