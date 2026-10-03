package intake

import (
	"sort"
	"strings"
	"unicode/utf8"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

// choice is one option offered to Jev. ID is the answer token, Value is the
// literal stored when the option is applied, and Source is where it was found.
// Describe, when set, is the option's description in the request.
type choice struct {
	ID       string `json:"id"`
	Value    string `json:"value"`
	Source   string `json:"source"`
	Label    string `json:"-"`
	Describe any    `json:"-"`
}

type builtQuestion struct {
	Field        string
	Instructions any
	Options      []choice
}

// roleQuestion is a structured instruction: the role the chosen value plays
// and what it is not, so similar candidates are told apart.
// https://docs.typesafe.ai/primitives/advanced
type roleQuestion struct {
	Question string `json:"question"`
	NotFor   string `json:"not_for"`
}

// Identity questions name the role the value plays in a drawing register and
// pick among pre-found spans only. Code splits revisions from numbers and
// orders revisions; Jev is not asked to.
// https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
var (
	numberQuestion = roleQuestion{
		Question: "Which candidate is this sheet's own drawing or document number, as a drawing register or transmittal would list it?",
		NotFor:   "A project or job number shared by every sheet of the project, a referenced standard, another sheet's number, or a number with the revision appended when the plain number is also a candidate.",
	}
	revisionQuestion = roleQuestion{
		Question: "Which candidate is the revision or issue of this document?",
		NotFor:   "A pit, pump, page or grid reference, or any value that is not a revision.",
	}
	titleQuestion = roleQuestion{
		Question: "Which candidate reproduces the title printed on this sheet or report cover? Prefer the complete printed title, including its subtitle, over an abbreviated or renamed filename. Use a filename title only when no printed document title is available.",
		NotFor:   "A project, site or company name by itself, a caption such as Drawing Title, or a general note. A printed document title may include a project or site name as part of its subtitle.",
	}
	dateQuestion = roleQuestion{
		Question: "Which candidate is the stated issue date for this document's own revision in its title block or document control?",
		NotFor:   "A print timestamp, a referenced document's date, or a historical revision's date. Select the explicit issue-date value; do not compare or order dates. Choose none when the current issue date is not identifiable.",
	}
	noneOption = map[string]string{"what": "None of these is the requested value."}
)

type filingState struct {
	Filename   string   `json:"filename"`
	TitleBlock string   `json:"title_block,omitempty"`
	FirstPage  string   `json:"first_page,omitempty"`
	Authorship string   `json:"authorship_text,omitempty"`
	Headings   []string `json:"headings,omitempty"`
	Candidates []choice `json:"candidates"`
}

func filingStateOf(filename string, text identity.Text, harvested []Candidate) filingState {
	state := filingState{
		Filename:   filename,
		TitleBlock: clip(labeledText(harvested), 1000),
		FirstPage:  identityExcerpt(runText(text), 2400),
		Authorship: authorshipText(text),
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

// Preserve literal author cues even when drawing notes occupy both excerpt
// ends. These spans are evidence for Jev, never a discipline rule.
func authorshipText(text identity.Text) string {
	var lines []string
	for _, run := range text.Runs {
		s := strings.ToLower(run.Text)
		if strings.Contains(s, "©") || strings.Contains(s, "copyright") || strings.Contains(s, "prepared by") || strings.Contains(s, "pty ltd") || strings.Contains(s, "@") {
			lines = append(lines, strings.TrimSpace(run.Text))
		}
	}
	return identityExcerpt(strings.Join(lines, "\n"), 1200)
}

// The PDF's final text objects often hold the author and revision table.
// Keep both ends without growing state with every annotation on the sheet.
func identityExcerpt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max / 2
	for head > 0 && !utf8.RuneStart(s[head]) {
		head--
	}
	tail := len(s) - (max - head)
	for tail < len(s) && !utf8.RuneStart(s[tail]) {
		tail++
	}
	return s[:head] + "\n[identity text omitted]\n" + s[tail:]
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

func identityQuestion(field string, instructions any, harvested []Candidate) (builtQuestion, bool) {
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
			ID:       optionID(key),
			Value:    chosen.Display,
			Source:   chosen.Provenance.Origin,
			Label:    chosen.Display,
			Describe: map[string]string{"value": chosen.Display, "found": foundIn(groups[key])},
		})
	}
	opts = append(opts, choice{ID: choiceNone, Label: "none", Source: "choice", Describe: noneOption})
	return opts
}

// foundIn says where code found a value, in the order filename, labelled
// cell, other page text. It is provenance, not a judgement.
func foundIn(group []Candidate) string {
	var file, labeled, text, heading bool
	for _, c := range group {
		heading = heading || (c.Provenance.Origin == OriginText && c.Provenance.Heading)
		switch {
		case c.Provenance.Origin == OriginFilename:
			file = true
		case c.Provenance.Labeled:
			labeled = true
		default:
			text = true
		}
	}
	var parts []string
	if file {
		parts = append(parts, "filename")
	}
	if labeled {
		parts = append(parts, "next to a label on the page")
	}
	if heading {
		parts = append(parts, "printed cover heading")
	}
	if text {
		parts = append(parts, "page text")
	}
	return strings.Join(parts, "; ")
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
		criteria := make(map[string]any, len(q.Options))
		for _, opt := range q.Options {
			if opt.Describe != nil {
				criteria[opt.ID] = opt.Describe
				continue
			}
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
