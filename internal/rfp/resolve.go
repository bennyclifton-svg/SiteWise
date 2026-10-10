package rfp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/knowledge"
)

// Stage states. An option is priced separately and not in the base fee; ask
// means a fact it depends on is unknown and the user must decide.
const (
	Include = "include"
	Option  = "option"
	Exclude = "exclude"
	Done    = "done"
	Ask     = "ask"
)

type tri int8

const (
	no tri = iota
	yes
	unk
)

// Setting is one stored user choice. Basis records the source fact the user
// overrode, so a later source change shows as a conflict instead of hiding.
type Setting struct {
	Value json.RawMessage `json:"value"`
	Basis json.RawMessage `json:"basis,omitempty"`
}

type Settings map[string]Setting

type Row struct {
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text"`
}

type StageResult struct {
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	Text           string   `json:"text"`
	Responsibility string   `json:"design_responsibility,omitempty"`
	FeeBasis       string   `json:"fee_basis"`
	ShownWhen      string   `json:"shown_when"`
	ByElement      bool     `json:"by_element,omitempty"`
	State          string   `json:"state"`
	Source         string   `json:"source"` // rule | boundary | default | user
	Why            string   `json:"why,omitempty"`
	Assumption     string   `json:"assumption,omitempty"`
	Locked         bool     `json:"locked"`
	Unknown        []string `json:"unknown,omitempty"`
}

type ElementResult struct {
	Key        string `json:"key"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Text       string `json:"text"`
	State      string `json:"state"`
	Source     string `json:"source"`
	Why        string `json:"why,omitempty"`
	Assumption string `json:"assumption,omitempty"`
}

type AllowanceResult struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Text   string `json:"text"`
	Value  string `json:"value"`
	Owner  string `json:"owner"`
	Source string `json:"source"` // default | user | none
}

type PositionResult struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Kind       string   `json:"kind"`
	Value      string   `json:"value"` // I | O | X | ask
	Party      string   `json:"party,omitempty"`
	Source     string   `json:"source"`
	Why        string   `json:"why,omitempty"`
	Assumption string   `json:"assumption,omitempty"`
	Unknown    []string `json:"unknown,omitempty"`
}

type FieldValue struct {
	Value  string `json:"value"`
	Source string `json:"source"` // default | user | none
}

type Weight struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Value  int    `json:"value"`
	Source string `json:"source"`
}

// Check is a Before-issue item. Blocking checks stop issue.
type Check struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Text     string   `json:"text"`
	Blocking bool     `json:"blocking"`
	Control  string   `json:"control,omitempty"`
	Changes  []string `json:"changes,omitempty"`
}

type Boundary struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Source    string `json:"source"` // suggested | user
	Confirmed bool   `json:"confirmed"`
}

type Resolved struct {
	Discipline    string                `json:"discipline"`
	Version       int                   `json:"version"`
	Status        string                `json:"status"`
	Title         string                `json:"title"`
	Heading       string                `json:"heading"`
	Scale         string                `json:"scale"`
	ScaleSource   string                `json:"scale_source"`
	ScaleWhy      string                `json:"scale_why"`
	Boundary      Boundary              `json:"boundary"`
	Facts         map[string]Fact       `json:"facts"`
	Parts         []Part                `json:"parts"`
	AllParts      []Part                `json:"all_parts"`
	Modules       map[string]string     `json:"modules"`
	Stages        []StageResult         `json:"stages"`
	Elements      []ElementResult       `json:"elements,omitempty"`
	Allowances    []AllowanceResult     `json:"allowances"`
	Positions     []PositionResult      `json:"positions"`
	ScopeSchedule []Row                 `json:"scope_schedule,omitempty"`
	Project       []Row                 `json:"project"`
	Grid          []GridColumn          `json:"grid"`
	Conditions    []Condition           `json:"conditions"`
	Appointment   []Row                 `json:"appointment"`
	Role          []Row                 `json:"role"`
	StageIntro    string                `json:"stage_intro"`
	ElementNote   string                `json:"element_intro,omitempty"`
	Terms         []Row                 `json:"terms"`
	Returnables   []Row                 `json:"returnables"`
	Fields        map[string]FieldValue `json:"fields"`
	Weights       []Weight              `json:"weights"`
	Budget        string                `json:"budget,omitempty"`
	Checks        []Check               `json:"checks"`
	Orphans       []string              `json:"orphans"`
	Durations     string                `json:"durations"`
	Legend        string                `json:"legend"`
	AllowNote     string                `json:"allowance_note"`
	Departures    string                `json:"departures"`
	EvalNote      string                `json:"evaluation_note"`
}

// GridColumn is one column of the project grid on page one; its cells are
// the knowledge.ProjectGrid rows that have something to print. Control is
// the editor control that changes the value ("" when Profile alone does).
type GridColumn struct {
	Title string     `json:"title"`
	Cells []GridCell `json:"cells"`
}

type GridCell struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Text    string `json:"text"`
	Control string `json:"control,omitempty"`
}

// Condition is one site condition printed on page one with its answer.
type Condition struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Answer  string `json:"answer"`
	Control string `json:"control"`
}

func (r Resolved) Stage(key string) StageResult {
	for _, s := range r.Stages {
		if s.Key == key {
			return s
		}
	}
	return StageResult{}
}

func (r Resolved) Position(key string) PositionResult {
	for _, p := range r.Positions {
		if p.Key == key {
			return p
		}
	}
	return PositionResult{}
}

func (r Resolved) Allowance(key string) AllowanceResult {
	for _, a := range r.Allowances {
		if a.Key == key {
			return a
		}
	}
	return AllowanceResult{}
}

func (r Resolved) Fact(key string) Fact { return r.Facts[key] }

func (r Resolved) Blocking() []Check {
	out := []Check{}
	for _, c := range r.Checks {
		if c.Blocking {
			out = append(out, c)
		}
	}
	return out
}

func (r Resolved) HasCheck(id string) bool {
	for _, c := range r.Checks {
		if c.ID == id {
			return true
		}
	}
	return false
}

type resolver struct {
	d        knowledge.RFPDiscipline
	sk       knowledge.RFPSkeleton
	drivers  DriverSet
	settings Settings
	out      Resolved
	// asks groups the rows that wait on each unknown fact.
	asks        map[string][]string
	askOrder    []string
	assumptions map[string][]string
	used        map[string]bool
	// skipped lists rows left out because their module's activation is not
	// established; they are reported, not silently dropped.
	skipped map[string][]string
}

// Resolve applies settings over the frozen drivers and the discipline rules.
// It is a pure function: the same inputs always give the same RFP.
func Resolve(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton, drivers DriverSet, settings Settings) Resolved {
	r := &resolver{d: d, sk: sk, drivers: drivers, settings: settings, asks: map[string][]string{}, assumptions: map[string][]string{}, used: map[string]bool{}, skipped: map[string][]string{}}
	r.out = Resolved{Discipline: d.ID, Version: d.Version, Status: d.Status, Title: d.Title, Heading: d.Heading, Modules: map[string]string{}, Fields: map[string]FieldValue{}, Checks: []Check{}, Orphans: []string{}, Durations: d.Durations, Legend: sk.Legend, AllowNote: sk.Allowances, Departures: sk.Departures, EvalNote: sk.Evaluate, StageIntro: d.StageIntro, ElementNote: d.ElementIntro}
	if d.AllowancesNote != "" {
		r.out.AllowNote = d.AllowancesNote
	}
	if d.EvaluationNote != "" {
		r.out.EvalNote = d.EvaluationNote
	}
	r.applyFacts()
	r.applyParts()
	r.resolveFields()
	r.resolveScale()
	r.resolveModules()
	r.resolveBoundary()
	r.resolveStages()
	r.resolveElements()
	r.resolveAllowances()
	r.resolvePositions()
	r.checkConsistency()
	r.resolveRows()
	r.resolveWeights()
	r.checkPathway()
	r.finishChecks()
	r.findOrphans()
	return r.out
}

// ---- facts and settings ----

func (r *resolver) setting(key string) (json.RawMessage, bool) {
	s, ok := r.settings[key]
	if ok {
		r.used[key] = true
	}
	return s.Value, ok
}

func settingString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func settingStrings(raw json.RawMessage) []string {
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	if s := settingString(raw); s != "" {
		return []string{s}
	}
	return nil
}

func (r *resolver) applyFacts() {
	r.out.Facts = map[string]Fact{}
	for k, v := range r.drivers.Facts {
		r.out.Facts[k] = v
	}
	if f := r.out.Facts["budget"]; f.State == Known {
		r.out.Budget = f.Values[0]
	}
}

func (r *resolver) fact(key string) Fact {
	if f, ok := r.out.Facts[key]; ok {
		return f
	}
	return Fact{State: Unknown, Source: "none"}
}

func (r *resolver) applyParts() {
	r.out.AllParts = r.drivers.Parts
	r.out.Parts = r.drivers.Parts
	if raw, ok := r.setting("parts:affected"); ok {
		ids := settingStrings(raw)
		chosen := []Part{}
		for _, p := range r.drivers.Parts {
			if contains(ids, p.ID) {
				chosen = append(chosen, p)
			}
		}
		if len(chosen) > 0 {
			r.out.Parts = chosen
		}
	}
}

// ---- three-valued evaluation ----

type outcome struct {
	t       tri
	why     string
	unknown []string
}

func (r *resolver) eval(w knowledge.RFPWhen) outcome {
	if len(w) == 0 {
		return outcome{t: yes}
	}
	res := outcome{t: no}
	for _, clause := range w {
		c := r.clause(clause)
		switch c.t {
		case yes:
			return c
		case unk:
			res.t = unk
			res.unknown = appendUnique(res.unknown, c.unknown...)
		}
	}
	return res
}

func (r *resolver) clause(clause map[string][]string) outcome {
	keys := make([]string, 0, len(clause))
	for k := range clause {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := outcome{t: yes}
	whys := []string{}
	for _, key := range keys {
		o := r.driver(key, clause[key])
		switch o.t {
		case no:
			return outcome{t: no}
		case unk:
			res.t = unk
			res.unknown = appendUnique(res.unknown, o.unknown...)
		default:
			if o.why != "" {
				whys = append(whys, o.why)
			}
		}
	}
	if res.t == yes {
		res.why = strings.Join(whys, ", ")
	}
	return res
}

var overFacts = map[string]string{"budget_over": "budget", "site_area_over": "site_area", "gfa_over": "gfa", "parking_over": "parking"}

func (r *resolver) driver(key string, values []string) outcome {
	switch key {
	case "class":
		if len(r.out.Parts) == 0 {
			return outcome{t: unk, unknown: []string{"class"}}
		}
		missing := false
		for _, p := range r.out.Parts {
			if len(p.Classes) == 0 {
				missing = true
			}
			for _, c := range p.Classes {
				if contains(values, c) {
					return outcome{t: yes, why: "Class " + c + ", " + p.Label}
				}
			}
		}
		if missing {
			return outcome{t: unk, unknown: []string{"class"}}
		}
		return outcome{t: no}
	case "flag":
		res := outcome{t: no}
		for _, v := range values {
			f := r.fact("flag." + v)
			switch {
			case f.State == Known && f.Values[0] == "yes":
				return outcome{t: yes, why: flagLabel(r.sk, v)}
			case f.State != Known:
				res.t = unk
				res.unknown = append(res.unknown, "flag."+v)
			}
		}
		return res
	case "stage_included":
		for _, v := range values {
			if r.stageState(v) == Include {
				return outcome{t: yes}
			}
		}
		return outcome{t: no}
	case "stage":
		for _, v := range values {
			switch r.stageState(v) {
			case Include, Option:
				return outcome{t: yes}
			}
		}
		return outcome{t: no}
	case "element":
		for _, v := range values {
			for _, e := range r.out.Elements {
				if e.Key == v && (e.State == Include || e.State == Option) {
					return outcome{t: yes}
				}
			}
		}
		return outcome{t: no}
	case "scale":
		if contains(values, r.out.Scale) {
			return outcome{t: yes, why: r.out.Scale}
		}
		return outcome{t: no}
	}
	if fact, ok := overFacts[key]; ok {
		n, known := number(r.fact(fact))
		if !known {
			return outcome{t: unk, unknown: []string{fact}}
		}
		for _, v := range values {
			limit, ok := number(Fact{Values: []string{v}, State: Known})
			if ok && n > limit {
				return outcome{t: yes, why: factLabel(r.sk, fact) + " over " + formatNumber(v)}
			}
		}
		return outcome{t: no}
	}
	f := r.fact(key)
	if f.State != Known {
		return outcome{t: unk, unknown: []string{key}}
	}
	for _, v := range f.Values {
		if contains(values, v) {
			return outcome{t: yes, why: factLabel(r.sk, key) + " " + valueLabel(r.sk, key, v)}
		}
	}
	return outcome{t: no}
}

func (r *resolver) stageState(key string) string {
	for _, s := range r.out.Stages {
		if s.Key == key {
			return s.State
		}
	}
	return ""
}

// ---- scale, modules, boundary ----

func (r *resolver) resolveScale() {
	if raw, ok := r.setting("scale"); ok {
		if v := settingString(raw); v == "simple" || v == "complex" {
			r.out.Scale, r.out.ScaleSource, r.out.ScaleWhy = v, "user", "Set by you"
			return
		}
	}
	if len(r.d.Scale.Complex) == 0 {
		r.out.Scale, r.out.ScaleSource, r.out.ScaleWhy = "simple", "rule", "No complex service-level triggers defined"
		return
	}
	o := r.eval(r.d.Scale.Complex)
	switch o.t {
	case yes:
		r.out.Scale, r.out.ScaleSource, r.out.ScaleWhy = "complex", "rule", o.why
	case no:
		r.out.Scale, r.out.ScaleSource, r.out.ScaleWhy = "simple", "rule", "No complex trigger applies"
	default:
		names := []string{}
		for _, u := range o.unknown {
			names = append(names, factLabel(r.sk, u))
		}
		r.out.Scale, r.out.ScaleSource, r.out.ScaleWhy = "simple", "assumed", "Assumed simple: not established — "+strings.Join(names, ", ")
		r.check(Check{ID: "scale", Label: "Service level assumed simple", Text: "These facts are not established: " + strings.Join(names, ", ") + ". Confirm them or set the service level.", Control: "scale"})
	}
}

func (r *resolver) resolveModules() {
	for _, m := range r.sk.Modules {
		o := r.eval(m.When)
		switch o.t {
		case yes:
			r.out.Modules[m.Key] = "on"
		case no:
			r.out.Modules[m.Key] = "off"
		default:
			r.out.Modules[m.Key] = "unknown"
		}
	}
}

// moduleGate reports whether a row's module lets it apply. When the module's
// activation is not established, its if_unknown decides: ask raises the
// decision, include keeps the row, and the default leaves the row out and
// reports it under Before issue.
func (r *resolver) moduleGate(module, row string) outcome {
	if module == "" {
		return outcome{t: yes}
	}
	switch r.out.Modules[module] {
	case "on":
		return outcome{t: yes}
	case "off":
		return outcome{t: no}
	}
	for _, m := range r.sk.Modules {
		if m.Key != module {
			continue
		}
		switch m.IfUnknown {
		case "ask":
			return r.eval(m.When)
		case "include":
			return outcome{t: yes}
		}
		r.skipped[module] = appendUnique(r.skipped[module], row)
	}
	return outcome{t: no}
}

func (r *resolver) resolveBoundary() {
	if raw, ok := r.setting("boundary:range"); ok {
		var b struct{ Start, End string }
		if json.Unmarshal(raw, &b) == nil && r.stageIndex(b.Start) >= 0 && r.stageIndex(b.End) >= r.stageIndex(b.Start) {
			r.out.Boundary = Boundary{Start: b.Start, End: b.End, Source: "user", Confirmed: true}
			return
		}
	}
	for _, b := range r.d.Boundary {
		if r.eval(b.When).t == yes {
			r.out.Boundary = Boundary{Start: b.Start, End: b.End, Source: "suggested"}
			break
		}
	}
	r.check(Check{ID: "boundary", Label: "Appointment not confirmed", Text: fmt.Sprintf("Suggested: %s to %s. Confirm the stages this appointment buys; stages after it that apply are priced as options.", r.stageName(r.out.Boundary.Start), r.stageName(r.out.Boundary.End)), Blocking: true, Control: "boundary:range"})
}

func (r *resolver) stageIndex(key string) int {
	for i, s := range r.d.Stages {
		if s.Key == key {
			return i
		}
	}
	return -1
}

func (r *resolver) stageName(key string) string {
	for _, s := range r.d.Stages {
		if s.Key == key {
			return s.Name
		}
	}
	return key
}

// ---- stages and elements ----

func (r *resolver) resolveStages() {
	start, end := r.stageIndex(r.out.Boundary.Start), r.stageIndex(r.out.Boundary.End)
	for i, st := range r.d.Stages {
		res := StageResult{Key: st.Key, Name: st.Name, Responsibility: st.Responsibility, FeeBasis: st.FeeBasis, ShownWhen: st.ShownWhen, ByElement: st.ByElement, Source: "rule"}
		gate := r.moduleGate(st.Module, st.Name)
		o := r.eval(st.When)
		if gate.t != yes {
			o = andOutcome(gate, o)
		}
		switch {
		case st.Follows != "":
			res.State, res.Why = r.stageState(st.Follows), "Follows "+r.stageName(st.Follows)
			if res.State == Ask {
				res.State = Option
			}
		case o.t == no:
			res.State, res.Why = Exclude, "Does not apply: "+st.ShownWhen
		case o.t == unk:
			res.Unknown = o.unknown
			if end >= 0 && i > end {
				// A later stage the client is not buying now is priced as an
				// option; it needs no decision before this issue.
				res.State, res.Source, res.Why = Option, "boundary", "After this appointment: priced as an option"
				res.Assumption = st.Assumption
				break
			}
			res.State, res.Assumption = r.unknownState(st.Suggest, st.IfUnknown, st.Assumption, st.Name, o.unknown, &res.Why)
			if res.State == Ask && (start >= 0 && i < start) {
				res.State = Exclude
			}
		default:
			res.Why = o.why
			if res.Why == "" {
				res.Why = st.ShownWhen
			}
			switch {
			case start >= 0 && i < start:
				res.State, res.Source, res.Why = Exclude, "boundary", "Before this appointment"
			case end >= 0 && i > end:
				res.State, res.Source, res.Why = Option, "boundary", "After this appointment: priced as an option"
			case st.Default == Option && (len(st.IncludeWhen) == 0 || r.eval(st.IncludeWhen).t != yes):
				res.State, res.Source = Option, "default"
			default:
				res.State = Include
			}
		}
		res.Locked = st.Follows != ""
		if raw, ok := r.setting("stage:" + st.Key); ok {
			if v := settingString(raw); contains([]string{Include, Option, Exclude, Done}, v) {
				res.State, res.Source, res.Assumption = v, "user", ""
			}
		}
		r.out.Stages = append(r.out.Stages, res)
	}
	// Deliverables are filled after every stage state is known, since a
	// deliverable may name an allowance whose condition tests a stage.
	for i, st := range r.d.Stages {
		r.out.Stages[i].Text = st.Deliverable
	}
}

// unknownState applies a row's if_unknown. A suggest rule that is false
// means the row does not apply; true or unknown raises the decision.
func (r *resolver) unknownState(suggest knowledge.RFPWhen, ifUnknown, assumption, name string, unknown []string, why *string) (string, string) {
	if len(suggest) > 0 {
		s := r.eval(suggest)
		if s.t == no {
			*why = "Not suggested for this project; set it if it applies"
			return Exclude, ""
		}
		if s.why != "" {
			*why = "Suggested: " + s.why + " (rule not verified)"
		}
		r.ask(unknown, name)
		return Ask, ""
	}
	switch ifUnknown {
	case "option":
		r.assume(unknown, name)
		return Option, assumption
	case "include":
		r.assume(unknown, name)
		return Include, assumption
	}
	r.ask(unknown, name)
	return Ask, ""
}

func (r *resolver) ask(unknown []string, name string) {
	for _, u := range unknown {
		if _, ok := r.asks[u]; !ok {
			r.askOrder = append(r.askOrder, u)
		}
		r.asks[u] = appendUnique(r.asks[u], name)
	}
}

func (r *resolver) assume(unknown []string, name string) {
	for _, u := range unknown {
		r.assumptions[u] = appendUnique(r.assumptions[u], name)
	}
}

func andOutcome(a, b outcome) outcome {
	if a.t == no || b.t == no {
		return outcome{t: no}
	}
	if a.t == unk || b.t == unk {
		return outcome{t: unk, unknown: appendUnique(append([]string(nil), a.unknown...), b.unknown...)}
	}
	return outcome{t: yes, why: strings.Trim(a.why+", "+b.why, ", ")}
}

func (r *resolver) resolveElements() {
	for _, e := range r.d.Elements {
		res := ElementResult{Key: e.Key, Code: e.Code, Name: e.Name, Text: e.Includes, Source: "rule"}
		o := r.eval(e.When)
		switch o.t {
		case yes:
			res.State, res.Why = Include, o.why
		case no:
			res.State, res.Why = Exclude, "Does not apply: "+e.ShownWhen
		default:
			res.State, res.Assumption = r.unknownState(nil, e.IfUnknown, e.Assumption, "Element "+e.Code+" "+e.Name, o.unknown, &res.Why)
		}
		if raw, ok := r.setting("element:" + e.Key); ok {
			if v := settingString(raw); contains([]string{Include, Option, Exclude}, v) {
				res.State, res.Source, res.Assumption = v, "user", ""
			}
		}
		r.out.Elements = append(r.out.Elements, res)
	}
}

// ---- allowances and positions ----

var countRE = regexp.MustCompile(`^[0-9]{1,3}$`)

func (r *resolver) resolveAllowances() {
	for _, a := range r.d.Allowances {
		if r.andModule(a.Module, a.Label, a.When).t == no {
			continue
		}
		res := AllowanceResult{Key: a.Key, Label: a.Label, Text: a.Text, Owner: a.Owner, Source: "none"}
		switch {
		case r.out.Scale == "complex" && a.ComplexDefault != "":
			res.Value, res.Source = a.ComplexDefault, "default"
		case a.Default != "":
			res.Value, res.Source = a.Default, "default"
		}
		if raw, ok := r.setting("allowance:" + a.Key); ok {
			if v := settingString(raw); countRE.MatchString(v) {
				res.Value, res.Source = v, "user"
			}
		}
		if res.Value == "" && a.Owner == "client" {
			// A count for a stage the client is buying now blocks issue; one for
			// a later option is asked for but leaves the firm to propose it.
			bought := r.boughtNow(a.When)
			text := "State the quantity every firm prices; a blank client quantity makes prices incomparable."
			if !bought {
				text = "Only an optional stage uses it. State it so options are priced on the same basis."
			}
			r.check(Check{ID: "allowance:" + a.Key, Label: a.Label + ": count not set", Text: text, Blocking: bought, Control: "allowance:" + a.Key})
		}
		r.out.Allowances = append(r.out.Allowances, res)
	}
}

// boughtNow reports whether a row gated on stages applies to a stage the
// client is buying in this appointment. Rows with no stage gate apply.
func (r *resolver) boughtNow(when knowledge.RFPWhen) bool {
	gated := false
	for _, clause := range when {
		for _, key := range clause["stage"] {
			gated = true
			if r.stageState(key) == Include {
				return true
			}
		}
	}
	return !gated
}

func (r *resolver) andModule(module, row string, when knowledge.RFPWhen) outcome {
	gate := r.moduleGate(module, row)
	if gate.t == no {
		return gate
	}
	return andOutcome(gate, r.eval(when))
}

func (r *resolver) resolvePositions() {
	for _, p := range r.d.Positions {
		res := PositionResult{Key: p.Key, Label: p.Label, Kind: p.Kind, Source: "rule"}
		row := r.andModule(p.Module, p.Label, p.When)
		var value string
		switch row.t {
		case no:
			value = "-"
		case unk:
			res.Unknown = row.unknown
			state, assumption := r.unknownState(p.Suggest, p.IfUnknown, p.Assumption, p.Label, row.unknown, &res.Why)
			value, res.Assumption = positionFor(state), assumption
		default:
			cell := p.Value
			if p.Kind == "service_level" {
				cell = p.Simple
				if r.out.Scale == "complex" {
					cell = p.Complex
				}
				res.Why = "Service level " + r.out.Scale
			}
			value = r.cell(*cell, p, &res)
		}
		if raw, ok := r.setting("position:" + p.Key); ok {
			if v := settingString(raw); contains([]string{"I", "O", "X"}, v) {
				value, res.Source, res.Assumption = v, "user", ""
			}
		}
		if value == "-" {
			continue
		}
		res.Value = value
		if value == "X" {
			res.Party = r.party(p)
		}
		r.out.Positions = append(r.out.Positions, res)
	}
}

func positionFor(state string) string {
	switch state {
	case Include:
		return "I"
	case Option:
		return "O"
	case Exclude:
		return "-"
	}
	return Ask
}

func (r *resolver) cell(c knowledge.RFPCell, p knowledge.RFPPosition, res *PositionResult) string {
	if len(c.When) == 0 {
		return c.Value
	}
	o := r.eval(c.When)
	switch o.t {
	case yes:
		res.Why = strings.TrimSpace(o.why)
		return c.Value
	case no:
		if c.Else == "" {
			return "-"
		}
		return c.Else
	}
	res.Unknown = o.unknown
	state, assumption := r.unknownState(p.Suggest, p.IfUnknown, p.Assumption, p.Label, o.unknown, &res.Why)
	res.Assumption = assumption
	return positionFor(state)
}

// party names who does an excluded item, with the status of that firm's
// package when the project has one. An engineer or consultant the project
// has no package for is printed as unallocated.
func (r *resolver) party(p knowledge.RFPPosition) string {
	party := p.Responsible
	if raw, ok := r.setting("responsible:" + p.Key); ok {
		if v := strings.TrimSpace(settingString(raw)); v != "" {
			party = v
		}
	}
	switch party {
	case "":
		// The user excluded an item the rules never exclude: someone must
		// still own it.
		r.check(Check{ID: "unallocated:" + p.Key, Label: "Unallocated: " + p.Label, Text: "You excluded this item. Name who does it.", Control: "responsible:" + p.Key})
		return "party not named"
	case "not_required":
		return "not required"
	case "client", "contractor", "superintendent", "architect":
		if party == "architect" && r.d.ID != "architecture" {
			if c, ok := r.consultantFor(party); ok {
				return "the architect, " + c.Status
			}
		}
		return "the " + party
	}
	if !needsFirm(party) {
		return party
	}
	if c, ok := r.consultantFor(party); ok {
		return party + ", " + c.Status
	}
	r.check(Check{ID: "unallocated:" + p.Key, Label: "Unallocated: " + p.Label, Text: "Excluded from this RFP, but the project has no " + party + " package. Add one or name who does it.", Control: "responsible:" + p.Key})
	return party + ", not yet engaged"
}

func needsFirm(party string) bool {
	if strings.Contains(party, "contractor") {
		return false
	}
	for _, w := range []string{"engineer", "consultant", "designer", "planner", "surveyor", "auditor", "locator"} {
		if strings.Contains(party, w) {
			return true
		}
	}
	return false
}

var partyWords = map[string][]string{
	"structural": {"structural", "structure"}, "civil": {"civil"}, "geotechnical": {"geotech", "geotechnical"},
	"hydraulic": {"hydraulic", "services"}, "town planner": {"planner", "planning"}, "quantity surveyor": {"cost", "qs", "quantity"},
	"traffic": {"traffic"}, "architect": {"architect"}, "façade": {"facade", "façade"}, "fire": {"fire"}, "access": {"access"},
}

func (r *resolver) consultantFor(party string) (Consultant, bool) {
	lower := strings.ReplaceAll(strings.ToLower(party), "-", " ")
	// These appointments used to share a taxonomy alias. A strategy author
	// cannot satisfy the separate system designer's responsibility.
	fireID := ""
	fireTitles := []string{}
	if strings.Contains(lower, "fire services") {
		fireID, fireTitles = "consultant.fire_services", []string{"fire services design", "fire services engineer", "wet fire design", "sprinkler design"}
	} else if strings.Contains(lower, "fire engineer") || strings.Contains(lower, "fire safety engineer") || strings.Contains(lower, "fire and access engineers") {
		fireID, fireTitles = "consultant.fire_engineer", []string{"fire engineer", "fire safety engineer"}
	}
	if fireID != "" {
		if strings.Contains(lower, " or ") {
			return Consultant{}, false // The project must settle the allocation.
		}
		for _, c := range r.drivers.Consultants {
			if c.DisciplineID != "" {
				if c.DisciplineID == fireID {
					return c, true
				}
				continue
			}
			title := strings.ReplaceAll(strings.ToLower(c.Title), "-", " ")
			if strings.Contains(title, "contractor") || strings.Contains(title, "trade") {
				continue
			}
			for _, phrase := range fireTitles {
				if strings.Contains(title, phrase) {
					return c, true
				}
			}
		}
		return Consultant{}, false
	}
	for key, words := range partyWords {
		if !strings.Contains(lower, key) {
			continue
		}
		for _, c := range r.drivers.Consultants {
			if key == "hydraulic" && (c.DisciplineID == "consultant.fire_services" || c.DisciplineID == "consultant.fire_engineer" || c.DisciplineID == "trade.fire_services") {
				continue
			}
			title := strings.ToLower(c.Title)
			for _, w := range words {
				if strings.Contains(title, w) {
					return c, true
				}
			}
		}
	}
	return Consultant{}, false
}

// checkConsistency refuses an item that a live stage includes while its own
// position excludes it: the RFP would both buy and exclude the same work.
func (r *resolver) checkConsistency() {
	pos := map[string]PositionResult{}
	for _, p := range r.out.Positions {
		pos[p.Key] = p
	}
	for i, st := range r.d.Stages {
		state := r.out.Stages[i].State
		if state != Include && state != Option {
			continue
		}
		for _, item := range st.Items {
			if p, ok := pos[item]; ok && p.Value == "X" {
				r.check(Check{ID: "conflict:" + item, Label: "Included and excluded at once: " + p.Label, Text: st.Name + " includes this item, but section 5 excludes it. Change the stage or the position.", Blocking: true, Control: "position:" + item})
			}
		}
	}
}

// ---- rows, fields, weights ----

func (r *resolver) resolveFields() {
	for _, f := range r.sk.Fields {
		v := FieldValue{Value: f.Default, Source: "default"}
		if d, ok := r.d.FieldDefaults[f.Key]; ok {
			v.Value = d
		}
		if v.Value == "" {
			v.Source = "none"
		}
		// A field with a profile home is read, never set here (D1).
		if f.Profile != "" {
			v = FieldValue{Value: r.drivers.Text[f.Key], Source: "profile"}
			if v.Value == "" {
				v.Source = "none"
			}
			r.out.Fields[f.Key] = v
			continue
		}
		if raw, ok := r.setting("field:" + f.Key); ok {
			if s := strings.TrimSpace(settingString(raw)); s != "" {
				v = FieldValue{Value: s, Source: "user"}
			}
		}
		r.out.Fields[f.Key] = v
	}
}

func (r *resolver) resolveRows() {
	keep := func(row knowledge.RFPRow) bool { return r.andModule(row.Module, row.Key, row.When).t == yes }
	fill := func(prefix string, row knowledge.RFPRow) Row {
		label := row.Label
		if label == "" {
			// An unlabelled row is the opening sentence of its section.
			label = map[string]string{"intro": "Project description"}[row.Key]
			if label == "" {
				label = strings.ReplaceAll(row.Key, "_", " ")
			}
		}
		return Row{Key: row.Key, Label: row.Label, Text: r.fill(row.Text, prefix+row.Key, label, row.Required)}
	}
	// Preserve the shared order when a discipline explicitly replaces a row.
	merge := func(shared, own []knowledge.RFPRow) []knowledge.RFPRow {
		replacements := map[string]knowledge.RFPRow{}
		for _, row := range own {
			if row.Replace && keep(row) {
				replacements[row.Key] = row
			}
		}
		rows := make([]knowledge.RFPRow, 0, len(shared)+len(own))
		for _, row := range shared {
			if replacement, ok := replacements[row.Key]; ok {
				row = replacement
			}
			rows = append(rows, row)
		}
		for _, row := range own {
			if !row.Replace {
				rows = append(rows, row)
			}
		}
		return rows
	}
	// Page one is shared: a discipline's replacement rewords a shared row or
	// a grid cell in place; its other project rows describe the appointment
	// and print in section 2.
	replacements := map[string]knowledge.RFPRow{}
	own := []knowledge.RFPRow{}
	for _, row := range r.d.ProjectRows {
		switch {
		case !keep(row):
		case row.Replace:
			replacements[row.Key] = row
		default:
			own = append(own, row)
		}
	}
	for _, row := range r.sk.ProjectRows {
		if replacement, ok := replacements[row.Key]; ok {
			row = replacement
		}
		if keep(row) && (row.Required || r.complete(row.Text)) {
			r.out.Project = append(r.out.Project, fill("project:", row))
		}
	}
	r.resolveGrid(replacements)
	r.resolveConditions()
	if len(r.out.Parts) < len(r.out.AllParts) {
		names := []string{}
		for _, p := range r.out.Parts {
			names = append(names, p.Label)
		}
		r.out.Appointment = append(r.out.Appointment, Row{Key: "covers", Text: "This appointment covers: " + strings.Join(names, ", ") + "."})
	}
	for _, row := range own {
		if row.Required || r.complete(row.Text) {
			r.out.Appointment = append(r.out.Appointment, fill("project:", row))
		}
	}
	for _, row := range r.d.Role {
		if keep(row) {
			r.out.Role = append(r.out.Role, fill("role:", row))
		}
	}
	for _, row := range r.d.ScopeSchedule {
		if keep(row) {
			r.out.ScopeSchedule = append(r.out.ScopeSchedule, fill("scope_schedule:", row))
		}
	}
	for _, row := range merge(r.sk.Terms, r.d.Terms) {
		if keep(row) && (row.Required || r.complete(row.Text)) {
			r.out.Terms = append(r.out.Terms, fill("terms:", row))
		}
	}
	for _, ret := range r.sk.Returnables {
		base := ret.Text
		for _, n := range r.d.ReturnableNotes {
			if n.Key == ret.Key && n.Replace && r.eval(n.When).t == yes {
				base = n.Text
			}
		}
		parts := []string{base}
		if extra := r.d.Returnables[ret.Key]; extra != "" {
			parts = append(parts, extra)
		}
		for _, n := range r.d.ReturnableNotes {
			if n.Key == ret.Key && !n.Replace && r.eval(n.When).t == yes {
				parts = append(parts, n.Text)
			}
		}
		// An optional sentence whose value is not set is left out rather than
		// printed with a blank.
		kept := []string{}
		for _, sentence := range strings.SplitAfter(strings.Join(parts, " "), ". ") {
			if r.complete(sentence) {
				kept = append(kept, sentence)
			}
		}
		r.out.Returnables = append(r.out.Returnables, Row{Key: ret.Key, Label: ret.Title, Text: strings.TrimSpace(r.fill(strings.Join(kept, ""), "returnables:"+ret.Key, ret.Title, false))})
	}
	for i := range r.out.Stages {
		s := &r.out.Stages[i]
		if s.State == Exclude {
			s.Text = r.fillQuiet(s.Text)
			continue
		}
		s.Text = r.fill(s.Text, "stage:"+s.Key, s.Name, false)
	}
	for i := range r.out.Allowances {
		a := &r.out.Allowances[i]
		n := a.Value
		if n == "" {
			n = "___"
			if a.Owner == "consultant" {
				n = "proposed by you in R2:"
			}
		}
		a.Text = strings.ReplaceAll(a.Text, "{n}", n)
	}
	// Missing issue details are one decision, not one per field; missing
	// project details are another, answered once in Profile.
	missing, first := []string{}, ""
	terms, termsFirst := []string{}, ""
	project, projectFirst := []string{}, ""
	for _, f := range r.sk.Fields {
		if !f.Required || r.out.Fields[f.Key].Value != "" {
			continue
		}
		if f.Profile != "" {
			project = append(project, f.Label)
			if projectFirst == "" {
				projectFirst = "fact:" + f.Key
			}
			continue
		}
		// Terms are their own decision, so its marker sits in section 6.
		if f.Group == "terms" {
			terms = append(terms, f.Label)
			if termsFirst == "" {
				termsFirst = "field:" + f.Key
			}
			continue
		}
		missing = append(missing, f.Label)
		if first == "" {
			first = "field:" + f.Key
		}
	}
	if len(missing) > 0 {
		r.check(Check{ID: "fields", Label: "Issue details not set", Text: strings.Join(missing, ", ") + ".", Blocking: true, Control: first, Changes: missing})
	}
	if len(terms) > 0 {
		r.check(Check{ID: "terms", Label: "Terms not set", Text: strings.Join(terms, ", ") + ".", Blocking: true, Control: termsFirst, Changes: terms})
	}
	if len(project) > 0 {
		r.check(Check{ID: "details", Label: "Project details not set", Text: strings.Join(project, ", ") + ". Set once in Profile; every RFP uses it.", Blocking: true, Control: projectFirst, Changes: project})
	}
}

var slotRE = regexp.MustCompile(`\{([a-z_]+(?::[a-z_0-9.]+)?)\}`)

// complete reports whether every slot in text has a value.
func (r *resolver) complete(text string) bool {
	for _, m := range slotRE.FindAllStringSubmatch(text, -1) {
		if _, ok := r.slot(m[1]); !ok {
			return false
		}
	}
	return true
}

// fill substitutes slots. An unfilled slot prints ___ and raises a check:
// blocking when the row is required, advisory otherwise.
func (r *resolver) fill(text, id, label string, required bool) string {
	missing := []string{}
	out := slotRE.ReplaceAllStringFunc(text, func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := r.slot(name)
		if !ok {
			missing = append(missing, slotLabel(r.sk, name))
			return "___"
		}
		return v
	})
	// A required field already has its own check; repeat only what it lacks.
	uncovered := []string{}
	for _, m := range missing {
		covered := false
		for _, f := range r.sk.Fields {
			covered = covered || (f.Required && f.Label == m)
		}
		if !covered {
			uncovered = append(uncovered, m)
		}
	}
	if len(uncovered) > 0 && required {
		r.check(Check{ID: "slot:" + id, Label: sentenceCase(label) + ": " + strings.Join(uncovered, ", ") + " not set", Text: "Needed before issue.", Blocking: true, Control: slotControl(uncovered, r.sk)})
	}
	return out
}

func (r *resolver) fillQuiet(text string) string {
	return slotRE.ReplaceAllStringFunc(text, func(m string) string {
		if v, ok := r.slot(m[1 : len(m)-1]); ok {
			return v
		}
		return "___"
	})
}

func (r *resolver) slot(name string) (string, bool) {
	if key, ok := strings.CutPrefix(name, "field:"); ok {
		v := r.out.Fields[key].Value
		return v, v != ""
	}
	if key, ok := strings.CutPrefix(name, "allowance:"); ok {
		for _, a := range r.out.Allowances {
			if a.Key == key && a.Value != "" {
				return a.Value, true
			}
		}
		for _, a := range r.d.Allowances {
			if a.Key == key && a.Default != "" {
				return a.Default, true
			}
		}
		return "", false
	}
	switch name {
	case "client":
		v := r.out.Fields["client"].Value
		return v, v != ""
	case "address":
		v := r.out.Fields["address"].Value
		if v == "" {
			v = r.drivers.Text["address"]
		}
		if v == "" {
			return "", false
		}
		if lga := r.out.Fields["lga"].Value; lga != "" {
			v += " (" + lga + ")"
		}
		return v, true
	case "work_phrase":
		f := r.fact("work_type")
		if f.State != Known {
			return "", false
		}
		phrase := map[string]string{"new": "build", "refurb": "refurbish and fit out", "extend": "extend", "remediation": "remediate", "advisory": "seek advice on"}[f.Values[0]]
		return phrase, phrase != ""
	case "parts_phrase":
		return r.partsPhrase()
	case "pathway":
		f := r.fact("pathway")
		if f.State != Known {
			return "Not yet decided", true
		}
		return valueLabel(r.sk, "pathway", f.Values[0]), true
	case "procurement", "procurement_route":
		f := r.fact("procurement")
		if f.State != Known {
			return "Not yet decided", true
		}
		v := valueLabel(r.sk, "procurement", f.Values[0])
		if n := r.fact("novation"); name == "procurement" && n.State == Known && n.Values[0] == "yes" {
			v += "; this appointment is novated to the contractor"
		}
		return v, true
	case "information":
		if v := r.out.Fields["information"].Value; v != "" {
			return v, true
		}
		if r.drivers.Documents > 0 {
			return fmt.Sprintf("%d current documents in the project register, listed in the issue record", r.drivers.Documents), true
		}
		return "", false
	case "consultants":
		names := []string{}
		for _, c := range r.drivers.Consultants {
			names = append(names, c.Title+" ("+c.Status+")")
		}
		if len(names) == 0 {
			return "none recorded yet", true
		}
		return strings.Join(names, ", "), true
	case "planning_report", "planning_report_long":
		f := r.fact("pathway")
		if f.State != Known {
			return "", false
		}
		short := map[string]string{"da": "Statement of Environmental Effects", "cdc": "complying development application", "ssd": "Environmental Impact Statement", "planning_permit": "permit application", "ref": "Review of Environmental Factors", "modification": "modification application"}
		long := map[string]string{"da": "Statement of Environmental Effects incorporating the specialist reports", "ssd": "Request for SEARs, then an Environmental Impact Statement incorporating the specialist reports"}
		v := short[f.Values[0]]
		if name == "planning_report_long" {
			v = long[f.Values[0]]
		}
		return v, v != ""
	}
	return "", false
}

func (r *resolver) partsPhrase() (string, bool) {
	// The opening sentence describes the project; section 2 says which parts
	// this appointment covers.
	names := []string{}
	for _, p := range r.out.AllParts {
		label := p.Label
		if p.Kind == "whole" {
			label = "the building"
			if c := r.fact("category"); c.State == Known {
				label = "a " + c.Values[0] + " building"
			}
		}
		if len(p.Classes) > 0 {
			label += " (Class " + strings.Join(p.Classes, ", ") + ")"
		}
		names = append(names, label)
	}
	if len(names) == 0 {
		return "", false
	}
	if len(names) == 1 {
		return names[0], true
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1], true
}

func evaluationWeights(d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton) []knowledge.RFPWeight {
	if len(d.Evaluation) > 0 {
		return d.Evaluation
	}
	return sk.Evaluation
}

func (r *resolver) resolveWeights() {
	total := 0
	for _, w := range evaluationWeights(r.d, r.sk) {
		res := Weight{Key: w.Key, Label: w.Label, Value: w.Default, Source: "default"}
		if raw, ok := r.setting("weight:" + w.Key); ok {
			var n int
			if json.Unmarshal(raw, &n) == nil && n >= 0 && n <= 100 {
				res.Value, res.Source = n, "user"
			}
		}
		total += res.Value
		r.out.Weights = append(r.out.Weights, res)
	}
	if total != 100 {
		r.check(Check{ID: "weights", Label: fmt.Sprintf("Evaluation weights total %d%%", total), Text: "They must total 100%.", Blocking: true, Control: "weight:price"})
	}
}

func (r *resolver) checkPathway() {
	f := r.fact("pathway")
	if f.State == Known && contains(r.d.UnsupportedPathways, f.Values[0]) {
		r.check(Check{ID: "pathway_unsupported", Label: "No reviewed " + strings.ToLower(r.d.Title) + " stages for " + valueLabel(r.sk, "pathway", f.Values[0]), Text: "This pathway's stage bundle has not been written and reviewed. It does not fall back to another pathway's wording.", Blocking: true, Control: "fact:pathway"})
	}
}

// finishChecks turns each unknown fact into one decision listing the rows
// that wait on it, and lists printed assumptions for review.
func (r *resolver) finishChecks() {
	for _, key := range []string{"gfa", "site_area", "parking", "storeys", "budget", "work_type", "category", "pathway", "procurement", "state"} {
		if f := r.fact(key); f.State == Conflict {
			r.check(Check{ID: "fact:" + key, Label: factLabel(r.sk, key) + ": sources disagree (" + strings.Join(f.Values, " vs ") + ")", Text: "Set the one figure every firm prices from.", Blocking: true, Control: "fact:" + key})
		}
	}
	for _, m := range r.sk.Modules {
		rows := r.skipped[m.Key]
		if len(rows) == 0 {
			continue
		}
		control := ""
		if o := r.eval(m.When); len(o.unknown) > 0 {
			control = controlFor(o.unknown[0])
		}
		r.check(Check{ID: "module:" + m.Key, Label: m.Label + ": not established", Text: "Left out until confirmed: " + strings.Join(rows, "; ") + ".", Control: control, Changes: rows})
	}
	for _, u := range r.askOrder {
		label := factLabel(r.sk, u)
		state := "not established"
		if r.fact(u).State == Conflict {
			state = "sources disagree (" + strings.Join(r.fact(u).Values, " vs ") + ")"
		}
		r.check(Check{ID: "fact:" + u, Label: label + ": " + state, Text: "Decide before issue. It changes: " + strings.Join(r.asks[u], "; ") + ".", Blocking: true, Control: controlFor(u), Changes: r.asks[u]})
	}
	keys := make([]string, 0, len(r.assumptions))
	for k := range r.assumptions {
		if _, asked := r.asks[k]; !asked {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, u := range keys {
		r.check(Check{ID: "assumed:" + u, Label: factLabel(r.sk, u) + ": not established", Text: "Priced on a printed assumption until confirmed: " + strings.Join(r.assumptions[u], "; ") + ".", Control: controlFor(u), Changes: r.assumptions[u]})
	}
}

func (r *resolver) findOrphans() {
	keys := make([]string, 0, len(r.settings))
	for k := range r.settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if r.used[k] || validKey(r.d, r.sk, r.drivers, k) {
			continue
		}
		r.out.Orphans = append(r.out.Orphans, k)
		r.check(Check{ID: "orphan:" + k, Label: "Setting no longer applies", Text: k + " is not in the current " + strings.ToLower(r.d.Title) + " rules. Remove it once you have checked it.", Control: k})
	}
}

func (r *resolver) check(c Check) {
	for _, old := range r.out.Checks {
		if old.ID == c.ID {
			return
		}
	}
	r.out.Checks = append(r.out.Checks, c)
	sort.SliceStable(r.out.Checks, func(i, j int) bool { return r.out.Checks[i].Blocking && !r.out.Checks[j].Blocking })
}

func controlFor(fact string) string {
	switch fact {
	case "class":
		return "parts:affected"
	}
	if strings.HasPrefix(fact, "stage:") {
		return fact
	}
	return "fact:" + fact
}

func slotControl(missing []string, sk knowledge.RFPSkeleton) string {
	if len(missing) == 0 {
		return ""
	}
	for _, f := range sk.Fields {
		if f.Label == missing[0] {
			if f.Profile != "" {
				return "fact:" + f.Key
			}
			return "field:" + f.Key
		}
	}
	switch missing[0] {
	case "Planning pathway":
		return "fact:pathway"
	case "Procurement":
		return "fact:procurement"
	case "Work type":
		return "fact:work_type"
	}
	return ""
}

func appendUnique(list []string, values ...string) []string {
	for _, v := range values {
		if !contains(list, v) {
			list = append(list, v)
		}
	}
	return list
}

func sentenceCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ---- page one ----

// resolveGrid fills the project grid from facts already resolved; a
// discipline's complete replacement rewords its cell.
func (r *resolver) resolveGrid(replacements map[string]knowledge.RFPRow) {
	used := usedDrivers(r.d, r.sk)
	for _, title := range knowledge.ProjectGridColumns {
		column := GridColumn{Title: title, Cells: []GridCell{}}
		for _, row := range knowledge.ProjectGrid {
			if row.Column != title || (row.Key == "regulated_design" && !used["regulated_design"]) {
				continue
			}
			text, ok := r.gridText(row.Key)
			if o, replaced := replacements[row.Key]; replaced && r.complete(o.Text) {
				text, ok = r.fill(o.Text, "project:"+o.Key, row.Label, o.Required), true
			}
			if !ok && !row.Essential {
				continue
			}
			if !ok {
				text = "___"
			}
			control := ""
			if FactHome(r.sk, r.out, row.Key).Kind != "" {
				control = "fact:" + row.Key
			}
			column.Cells = append(column.Cells, GridCell{Key: row.Key, Label: row.Label, Text: text, Control: control})
		}
		r.out.Grid = append(r.out.Grid, column)
	}
}

func (r *resolver) gridText(key string) (string, bool) {
	number := func(fact, unit string) (string, bool) {
		switch f := r.fact(fact); f.State {
		case Known:
			return formatNumber(f.Values[0]) + unit, true
		case Conflict:
			return "To be confirmed", true
		}
		return "", false
	}
	switch key {
	case "work_type":
		switch f := r.fact("work_type"); f.State {
		case Known:
			return valueLabel(r.sk, "work_type", f.Values[0]), true
		case Conflict:
			return "To be confirmed", true
		}
		return "", false
	case "parts":
		lines := []string{}
		for _, p := range r.out.AllParts {
			line := p.Label
			if len(p.Classes) > 0 {
				line += ": Class " + strings.Join(p.Classes, ", ")
			} else {
				line += ": class not established"
			}
			if p.GFA != "" {
				line += ", " + formatNumber(p.GFA) + " m² GFA"
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "; "), len(lines) > 0
	case "storeys", "parking":
		return number(key, "")
	case "gfa", "site_area":
		return number(key, " m²")
	case "address":
		v := r.out.Fields["address"].Value
		if v == "" {
			v = r.drivers.Text["address"]
		}
		return v, v != ""
	case "pathway", "procurement":
		return r.slot(key)
	case "regulated_design":
		return answer(r.fact("regulated_design")), true
	}
	v := r.out.Fields[key].Value
	return v, v != ""
}

// resolveConditions lists every site condition the discipline tests, and
// any other known to apply, each with its answer. A scope decision is the
// client's choice for this appointment, not a condition of the site.
func (r *resolver) resolveConditions() {
	used := usedDrivers(r.d, r.sk)
	for _, f := range r.sk.Flags {
		fact := r.fact("flag." + f.Key)
		applies := fact.State == Known && fact.Values[0] == "yes"
		if f.ScopeDecision || (!used["flag."+f.Key] && !applies) {
			continue
		}
		r.out.Conditions = append(r.out.Conditions, Condition{Key: f.Key, Label: f.Label, Answer: answer(fact), Control: "fact:flag." + f.Key})
	}
}

func answer(f Fact) string {
	switch {
	case f.State == Conflict:
		return "To be confirmed"
	case f.State == Known && f.Values[0] == "yes":
		return "Yes"
	case f.State == Known && f.Values[0] == "no":
		return "No"
	}
	return "Not known"
}
