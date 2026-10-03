package profile

import (
	"fmt"
	"regexp"
	"strings"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
)

// QuestionVersion versions every profile question below. Bump it when
// wording or options change, so recorded answers are not mixed.
const QuestionVersion = "profile-1"

// PassageInfo is what header routing reads about a passage.
type PassageInfo struct {
	Kind    string
	Section string
	Ordinal int
}

var (
	headerKinds   = map[string]bool{"design_brief": true, "specification": true, "report": true, "contract": true, "commercial": true}
	headerSection = regexp.MustCompile(`(?i)description|outline|scope|introduction|project details|key development`)
)

const headerOrdinals = 12

// HeaderRouted reports whether a passage gets project header questions: an
// early or descriptive passage of a brief, specification, report, contract
// or quote. Code routes; Jev answers only where the header is likely stated
// (https://docs.typesafe.ai/patterns/intent-routing).
func HeaderRouted(p PassageInfo) bool {
	return headerKinds[p.Kind] && (p.Ordinal < headerOrdinals || headerSection.MatchString(p.Section))
}

const notStatedCriterion = "The passage does not state it."

// LabelQuestions are the profile questions asked in the label fan-out, which
// reads the passage before labels exist. They join that one request
// (https://docs.typesafe.ai/patterns/fan-out). Returned candidates go into
// the call state under "candidates".
func LabelQuestions(p PassageInfo, h Harvested, cat *knowledge.Catalog) (map[string]jev.Question, map[string][]Candidate) {
	qs := map[string]jev.Question{}
	state := map[string][]Candidate{}
	if cat == nil {
		return qs, state
	}
	for _, key := range h.Triggered {
		switch {
		case strings.HasPrefix(key, "det."):
			d, ok := cat.Determinant(strings.TrimPrefix(key, "det."))
			if !ok {
				continue
			}
			if d.Value == "multi_choice" {
				for _, c := range h.Candidates[key] {
					qs[key+"."+c.Norm] = optionPresence(d, c.Norm)
				}
			} else if q, ok := valueQuestion(d, h.Candidates[key]); ok {
				qs[key] = q
				if len(h.Candidates[key]) > 0 {
					state[key] = h.Candidates[key]
				}
			} else {
				continue
			}
			qs[key+".assertion"] = assertionQuestion(d.Label)
		case strings.HasPrefix(key, "fact."):
			f, ok := cat.ProjectFact(strings.TrimPrefix(key, "fact."))
			if !ok {
				continue
			}
			if q, ok := valueQuestion(f, h.Candidates[key]); ok {
				qs[key] = q
				if len(h.Candidates[key]) > 0 {
					state[key] = h.Candidates[key]
				}
			}
		case strings.HasPrefix(key, "hdr.scale."):
			label := strings.TrimPrefix(key, "hdr.scale.")
			for _, f := range cat.ScaleFields() {
				if f.Key == label {
					label = f.Label
				}
			}
			qs[key] = jev.Question{Type: jev.TypeChoice,
				Instructions: "Using `text` and `candidates`, which candidate does the passage state as this project's " +
					strings.ToLower(label) + "? Select none when absent or when it describes something other than this project.",
				Criteria: candidateCriteria(label, h.Candidates[key])}
			state[key] = h.Candidates[key]
		}
	}
	if HeaderRouted(p) {
		for id, q := range headerQuestions(cat.Taxonomy()) {
			qs[id] = q
		}
	}
	return qs, state
}

// EvidenceQuestions ask presence and provider for each live leaf system the
// passage was labelled with. They join the evidence fan-out.
func EvidenceQuestions(labels []string, cat *knowledge.Catalog) map[string]jev.Question {
	qs := map[string]jev.Question{}
	if cat == nil {
		return qs
	}
	for _, id := range labels {
		sys, ok := cat.System(id)
		if !ok || sys.Parent == "" || sys.Status == "deprecated" {
			continue
		}
		label := strings.TrimSpace(sys.Label)
		qs["sys."+id+".presence"] = jev.Question{Type: jev.TypeChoice,
			Instructions: "Using `text`, does the passage say whether the completed project will have " + label + "?",
			Criteria: map[string]string{
				"included":     "It says the project will have it: it is specified, required, priced, allowed for, or supplied by any party.",
				"not_included": "It says the project will not have it: not applicable, not required, or not part of the project.",
				"not_stated":   "It mentions it without saying whether the project will have it, or only excludes it from one party's price.",
			}}
		qs["sys."+id+".provider"] = jev.Question{Type: jev.TypeChoice,
			Instructions: "Using `text`, who does the passage say provides " + label + "?",
			Criteria: map[string]string{
				"contractor": "The builder, contractor or tenderer provides it as part of their works or price.",
				"owner":      "The owner, client or principal supplies or arranges it.",
				"others":     "An authority, developer, separate contractor or other named party provides it.",
				"not_stated": "The passage does not say who provides it.",
			}}
	}
	return qs
}

func assertionQuestion(label string) jev.Question {
	return jev.Question{Type: jev.TypeChoice,
		Instructions: "Using `text`, how does the passage present the " + label + "?",
		Criteria: map[string]string{
			"stated":     "It states the value as a fact about this project or site, such as an assessment result, a certificate or a measured value.",
			"required":   "It requires the value: a brief, consent, contract or specification says the building must have or achieve it.",
			"allowance":  "It treats the value as an assumption or a pricing allowance, for example 'we have allowed for' or 'assumed'.",
			"not_stated": "The passage does not give a value for it.",
		}}
}

// valueQuestion builds a determinant's or fact's own question: candidates
// plus none for pre-parsed values, its options plus not_stated for choices,
// and its explicit true/false/not_stated criteria for booleans.
func valueQuestion(d knowledge.Determinant, cands []Candidate) (jev.Question, bool) {
	if d.Question == nil || strings.TrimSpace(d.Question.Instructions) == "" {
		return jev.Question{}, false
	}
	q := jev.Question{Type: jev.TypeChoice, Instructions: strings.TrimSpace(d.Question.Instructions)}
	switch {
	case d.Extraction == "pre_parsed":
		if len(cands) == 0 {
			return jev.Question{}, false
		}
		q.Criteria = candidateCriteria(d.Label, cands)
	case len(d.Options) > 0:
		crit := map[string]string{"not_stated": notStatedCriterion}
		for _, o := range d.Options {
			crit[o.ID] = o.Describes
		}
		q.Criteria = crit
	default:
		crit, ok := d.Question.Criteria.(map[string]any)
		if !ok || len(crit) == 0 {
			return jev.Question{}, false
		}
		out := map[string]string{}
		for k, v := range crit {
			out[k] = strings.TrimSpace(fmt.Sprint(v))
		}
		q.Criteria = out
	}
	return q, true
}

func candidateCriteria(label string, cands []Candidate) map[string]string {
	crit := map[string]string{"none": "The passage does not state the " + strings.ToLower(label) + ", or no candidate is it."}
	for _, c := range cands {
		crit[c.ID] = fmt.Sprintf("%s (in: '%s')", c.Value, c.Context)
	}
	return crit
}

// optionPresence asks about one class at a time: a choice cannot return a
// set (SCHEMA.md, determinant evidence contract).
func optionPresence(d knowledge.Determinant, opt string) jev.Question {
	return jev.Question{Type: jev.TypeChoice,
		Instructions: fmt.Sprintf("Using `text`, does the passage state that this project's building or part is Class %s?", opt),
		Criteria: map[string]string{
			"stated_true":  fmt.Sprintf("It states that this project's building or part is Class %s.", opt),
			"stated_false": fmt.Sprintf("It states that this project's building or part is not Class %s.", opt),
			"not_stated":   fmt.Sprintf("It mentions Class %s for something else, such as a neighbouring building or a material grade, or does not say.", opt),
		}}
}

func headerQuestions(t knowledge.Taxonomy) map[string]jev.Question {
	qs := map[string]jev.Question{}
	classes := map[string]string{"not_stated": notStatedCriterion}
	subs := map[string]string{"not_stated": notStatedCriterion}
	for _, c := range t.BuildingClasses {
		classes[c.ID] = c.Label
		for _, s := range c.Subclasses {
			label := s.Label
			if s.NCCClass != "" && !strings.Contains(label, "Class") {
				label += " (NCC Class " + s.NCCClass + ")"
			}
			subs[s.ID] = c.Label + ": " + label
		}
	}
	ask := func(id, label string, crit map[string]string) {
		qs[id] = jev.Question{Type: jev.TypeChoice,
			Instructions: "Using `text` and `document`, which " + label + " does the passage state for this project?",
			Criteria:     crit}
	}
	ask("hdr.building_class", "building class", classes)
	ask("hdr.subclass", "building type", subs)
	works := map[string]string{"not_stated": notStatedCriterion}
	for _, w := range t.WorkTypes {
		works[w.ID] = w.Label
	}
	ask("hdr.work_type", "type of work (new build, refurbishment, extension, remediation or advisory)", works)
	for _, c := range t.Conditions {
		crit := map[string]string{"not_stated": notStatedCriterion}
		for _, o := range c.Options {
			crit[o.ID] = o.Label
		}
		ask("hdr.cond."+c.Key, strings.ToLower(c.Label), crit)
	}
	return qs
}
