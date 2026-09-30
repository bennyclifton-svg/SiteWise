package knowledge

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// StateUnknown means code will not treat the input as a determination.
	StateUnknown = "unknown"
	// StateDetermined is a table result from one consistent, reviewed scope.
	StateDetermined = "determined"
	// StateAddressed means a passage speaks to the question. It is coverage,
	// not a statement that the building complies.
	StateAddressed = "addressed"
	// StateNotAddressed means the supplied evidence explicitly does not
	// address the question.
	StateNotAddressed = "not_addressed"

	statusReviewed = "reviewed"
)

// FirePilotInterfaces are the first background pilot: hydraulic fire water,
// electrical supply, structural loading and smoke control.
var FirePilotInterfaces = []string{
	"if.fire-water-supplies-sprinklers",
	"if.essential-power-supplies-fire-water",
	"if.electrical-power-for-fire-systems",
	"if.fire-tanks-load-structure",
	"if.fire-detection-controls-smoke-plant",
	"if.electrical-power-for-smoke-control",
}

// Scope is the physical thing a fact belongs to. Facts from different scopes
// are not merged into one project-wide value.
type Scope struct {
	Level string
	Ref   string
}

func (s Scope) key() string {
	return s.Level + "\x00" + s.Ref
}

// Fact is one extracted determinant value and where it came from.
type Fact struct {
	Determinant string
	Value       string
	Scope       Scope
	DocumentID  string
	PassageID   string
}

// Derivation is the result of a table lookup. State is unknown unless every
// guard passes. Provenance keeps the facts that were considered, including
// the ones that disagree.
type Derivation struct {
	RuleID     string
	Gives      string
	State      string
	Value      string
	Reason     string
	Provenance []Fact
}

// Edge is one computed end of a canonical interface.
type Edge struct {
	InterfaceID string `json:"interface_id"`
	Type        string `json:"type"`
	From        string `json:"from"`
	To          string `json:"to"`
}

// Answer is evidence a passage did or did not address a question.
// A nil Addressed is unresolved and stays unknown.
type Answer struct {
	QuestionID string
	Addressed  *bool
	DocumentID string
	PassageID  string
}

// Provenance locates a coverage or conflict without becoming a compliance claim.
type Provenance struct {
	DocumentID string `json:"document_id,omitempty"`
	PassageID  string `json:"passage_id,omitempty"`
	Value      string `json:"value,omitempty"`
}

// QuestionCoverage is whether passages address one knowledge question.
type QuestionCoverage struct {
	ID         string       `json:"id"`
	State      string       `json:"state"`
	Provenance []Provenance `json:"provenance,omitempty"`
}

// Coverage is evidence for one interface. Draft stays true until the owner
// reviews the interface. There is no compliance flag on purpose.
type Coverage struct {
	InterfaceID string             `json:"interface_id"`
	Draft       bool               `json:"draft"`
	Questions   []QuestionCoverage `json:"questions"`
}

// Report is the fire pilot: applicable edges and evidence coverage.
type Report struct {
	Edges    []Edge     `json:"edges"`
	Coverage []Coverage `json:"coverage"`
}

// Edges expands every interface into from/to pairs. The interface record
// stays canonical; this is the graph code walks.
func (c *Catalog) Edges() []Edge {
	out := make([]Edge, 0)
	for _, iface := range c.Interfaces {
		for _, from := range iface.From {
			for _, to := range iface.To {
				out = append(out, Edge{
					InterfaceID: iface.ID,
					Type:        iface.Type,
					From:        from,
					To:          to,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].InterfaceID != out[j].InterfaceID {
			return out[i].InterfaceID < out[j].InterfaceID
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

// Applicable returns edges whose from and to systems are both present.
// A present descendant satisfies a broader endpoint.
func (c *Catalog) Applicable(present []string) []Edge {
	var out []Edge
	for _, edge := range c.Edges() {
		if c.presentCovers(present, edge.From) && c.presentCovers(present, edge.To) {
			out = append(out, edge)
		}
	}
	return out
}

func (c *Catalog) presentCovers(present []string, endpoint string) bool {
	for _, id := range present {
		if c.covers(id, endpoint) {
			return true
		}
	}
	return false
}

// Derive looks up ruleID from facts. Missing tables, pending derivations,
// unreviewed rules or tables, mixed scopes and contradictory values all
// return unknown. The facts are kept on Provenance either way.
func (c *Catalog) Derive(ruleID string, facts []Fact) Derivation {
	out := Derivation{RuleID: ruleID, State: StateUnknown, Provenance: cloneFacts(facts)}
	rule, ok := c.rules[ruleID]
	if !ok || rule.Derives == nil {
		out.Reason = "not_derived"
		return out
	}
	out.Gives = rule.Derives.Gives
	if rule.Derives.Pending {
		out.Reason = "pending"
		return out
	}
	if rule.Status != statusReviewed {
		out.Reason = "unreviewed"
		return out
	}
	table, ok := c.tables[rule.Derives.Table]
	if !ok {
		out.Reason = "missing_table"
		return out
	}
	if table.Status != statusReviewed || !table.Verified {
		out.Reason = "unreviewed"
		return out
	}
	values, reason := agree(rule.Derives.Inputs, facts)
	if reason != "" {
		out.Reason = reason
		return out
	}
	value, ok := lookup(table, rule.Derives.Gives, values)
	if !ok {
		out.Reason = "no_row"
		return out
	}
	out.State = StateDetermined
	out.Value = value
	out.Reason = ""
	return out
}

func agree(inputs []string, facts []Fact) (map[string]string, string) {
	relevant := make([]Fact, 0)
	for _, fact := range facts {
		for _, input := range inputs {
			if fact.Determinant == input {
				relevant = append(relevant, fact)
				break
			}
		}
	}
	if len(relevant) == 0 {
		return nil, "missing_evidence"
	}
	scopes := map[string]struct{}{}
	for _, fact := range relevant {
		scopes[fact.Scope.key()] = struct{}{}
	}
	if len(scopes) != 1 {
		return nil, "mixed_scope"
	}
	values := map[string]string{}
	for _, input := range inputs {
		var got string
		seen := false
		for _, fact := range relevant {
			if fact.Determinant != input {
				continue
			}
			value := strings.TrimSpace(fact.Value)
			if !seen {
				got = value
				seen = true
				continue
			}
			if value != got {
				return nil, "conflict"
			}
		}
		if !seen || got == "" {
			return nil, "missing_evidence"
		}
		values[input] = got
	}
	return values, ""
}

func lookup(table Table, gives string, values map[string]string) (string, bool) {
	var found string
	matched := false
	for _, row := range table.Rows {
		if !rowMatches(row, values) {
			continue
		}
		value, ok := rowScalar(row[gives])
		if !ok {
			return "", false
		}
		if matched && value != found {
			return "", false
		}
		found = value
		matched = true
	}
	return found, matched
}

func rowMatches(row map[string]any, values map[string]string) bool {
	for input, value := range values {
		switch input {
		case "ncc_class":
			if classes, ok := row["classes"]; ok {
				if !scalarListContains(classes, value) {
					return false
				}
				continue
			}
		case "rise_in_storeys":
			if _, ok := row["rise_min"]; ok {
				n, err := strconv.Atoi(value)
				if err != nil || n < asInt(row["rise_min"]) {
					return false
				}
				if max, ok := row["rise_max"]; ok && n > asInt(max) {
					return false
				}
				continue
			}
		}
		cell, ok := row[input]
		if !ok {
			return false
		}
		got, ok := rowScalar(cell)
		if !ok || got != value {
			return false
		}
	}
	return true
}

func scalarListContains(list any, value string) bool {
	switch items := list.(type) {
	case []any:
		for _, item := range items {
			got, ok := rowScalar(item)
			if ok && got == value {
				return true
			}
		}
	case []string:
		for _, item := range items {
			if item == value {
				return true
			}
		}
	}
	return false
}

func rowScalar(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, t != ""
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), true
		}
		return strconv.FormatFloat(t, 'f', -1, 64), true
	default:
		return "", false
	}
}

func asInt(v any) int {
	s, ok := rowScalar(v)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func cloneFacts(facts []Fact) []Fact {
	if len(facts) == 0 {
		return nil
	}
	out := make([]Fact, len(facts))
	copy(out, facts)
	return out
}

// FirePilot reports evidence coverage for the pilot interfaces that the
// present systems bring into the graph. Draft output never claims the
// building complies.
func (c *Catalog) FirePilot(present []string, answers []Answer) Report {
	applicable := map[string][]Edge{}
	for _, edge := range c.Applicable(present) {
		if !pilotInterface(edge.InterfaceID) {
			continue
		}
		applicable[edge.InterfaceID] = append(applicable[edge.InterfaceID], edge)
	}
	report := Report{}
	for _, id := range FirePilotInterfaces {
		edges := applicable[id]
		if len(edges) == 0 {
			continue
		}
		report.Edges = append(report.Edges, edges...)
		iface := c.interfaceByID(id)
		coverage := Coverage{
			InterfaceID: id,
			Draft:       iface.Status != statusReviewed,
		}
		for _, q := range iface.ResolvedWhen {
			if q.Type != "noul" || q.ID == "" {
				continue
			}
			coverage.Questions = append(coverage.Questions, coverQuestion(q.ID, answers))
		}
		report.Coverage = append(report.Coverage, coverage)
	}
	return report
}

func pilotInterface(id string) bool {
	for _, want := range FirePilotInterfaces {
		if id == want {
			return true
		}
	}
	return false
}

func (c *Catalog) interfaceByID(id string) Interface {
	for _, iface := range c.Interfaces {
		if iface.ID == id {
			return iface
		}
	}
	return Interface{}
}

func coverQuestion(id string, answers []Answer) QuestionCoverage {
	var hits []Answer
	for _, answer := range answers {
		if answer.QuestionID == id && answer.Addressed != nil {
			hits = append(hits, answer)
		}
	}
	out := QuestionCoverage{ID: id, State: StateUnknown}
	if len(hits) == 0 {
		return out
	}
	state := StateAddressed
	if !*hits[0].Addressed {
		state = StateNotAddressed
	}
	for _, hit := range hits[1:] {
		next := StateAddressed
		if !*hit.Addressed {
			next = StateNotAddressed
		}
		if next != state {
			out.Provenance = answerProvenance(hits)
			return out
		}
	}
	out.State = state
	out.Provenance = answerProvenance(hits)
	return out
}

func answerProvenance(answers []Answer) []Provenance {
	out := make([]Provenance, len(answers))
	for i, answer := range answers {
		value := "false"
		if answer.Addressed != nil && *answer.Addressed {
			value = "true"
		}
		out[i] = Provenance{
			DocumentID: answer.DocumentID,
			PassageID:  answer.PassageID,
			Value:      value,
		}
	}
	return out
}

// LabelQuestionID is the noul that asks whether a passage is about a top-level system.
func LabelQuestionID(systemID string) string {
	return "system." + systemID
}

// LeafQuestionID is the speculative choice of a child system, asked with the noul.
func LeafQuestionID(systemID string) string {
	return "leaf." + systemID
}

// NoneOption is the leaf choice for a passage that is not about one child system.
func NoneOption() string { return "none" }

// Describe returns criterion text, falling back to the label when a draft
// system has no describes yet.
func Describe(sys System, fallback string) string {
	if text := strings.TrimSpace(sys.Describes); text != "" {
		return text
	}
	if text := strings.TrimSpace(sys.Label); text != "" {
		return text
	}
	return fallback
}

// ExcludesText is the false criterion for a system noul.
func ExcludesText(sys System) string {
	if text := strings.TrimSpace(sys.Excludes); text != "" {
		return text
	}
	return fmt.Sprintf("The passage is not about %s.", Describe(sys, sys.ID))
}
