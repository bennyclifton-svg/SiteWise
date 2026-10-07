package profile

import (
	"sort"
	"strings"

	"sitewise/internal/knowledge"
)

// Fact is one stored reading of one question in one passage.
type Fact struct {
	ID, QuestionVersion string
	QuestionID          string
	Value               string
	Unit, Basis         string
	PartLabel           string
	Excerpt             string
	Confidence          *float64
	DecidedBy           string // jev | rule
	DocumentID          string
	PassageID           string
	DocumentKind        string
	// ReadSetting is the document's profile reading setting (auto, read, skip).
	ReadSetting string
	Superseded  bool
	// Where the reading came from, kept as a snapshot because passage ids are
	// replaced when a document is reprocessed (plan §4.1, SourceRef).
	FileSHA256     string
	Filename       string
	DocumentNumber string
	Revision       string
	Page           int
	Location       string
	Section        string
	StartOffset    int
	EndOffset      int
}

// Part is a building, part, storey, compartment or tenancy facts belong to.
type Part struct{ ID, Label, Kind, NCCClass string }

// UserValue is the user's word for one key on one part. It is final.
type UserValue struct {
	Scope, ReviewStatus string
	PartID, Key         string
	Value               *string // nil means cleared by the user, or unknown (State)
	Note                string
	// State is set, cleared or unknown; empty reads as set or cleared from Value.
	State string
	// Origin is user or assumption; Meaning is stated, requirement, allowance
	// or forecast (plan §4.1). Empty means user and stated.
	Origin, Meaning string
	Version         int64
}

// PlanningValue is an assumption or a calculated value on a registered
// planning key (plan §4.3), shown as row "plan.<key>". It is never evidence
// and never feeds a derivation (D-06).
type PlanningValue struct {
	Scope       string
	PartID, Key string // Key without the "plan." prefix
	// State is set or unknown; Value is nil when unknown.
	State               string
	Value               *string
	RangeLow, RangeHigh *float64
	Unit                string
	Origin              string // user, assumption or calculation
	ReviewStatus        string // proposed, accepted_for_planning or superseded
	Meaning             string
	Rationale           string
	Limitations         string
	Version             int64
}

// Source is where a row's evidence came from.
// It is a snapshot (SourceRef, plan §4.1): the document's hash, revision and
// location survive reprocessing, which replaces passage ids.
type Source struct {
	DocumentID     string   `json:"document_id"`
	PassageID      string   `json:"passage_id,omitempty"`
	Excerpt        string   `json:"excerpt,omitempty"`
	Confidence     *float64 `json:"confidence,omitempty"`
	FileSHA256     string   `json:"file_sha256,omitempty"`
	Filename       string   `json:"filename,omitempty"`
	DocumentNumber string   `json:"document_number,omitempty"`
	Revision       string   `json:"revision,omitempty"`
	Page           int      `json:"page,omitempty"`
	Location       string   `json:"location,omitempty"`
	Section        string   `json:"section,omitempty"`
	StartOffset    int      `json:"start_offset,omitempty"`
	EndOffset      int      `json:"end_offset,omitempty"`
}

// Alternative is one competing value in a conflict, with its support.
type Alternative struct {
	Value   string   `json:"value"`
	Sources []Source `json:"sources"`
}

// Derived records a code table lookup and why it is unknown when it is.
type Derived struct {
	Rule       string               `json:"rule"`
	State      string               `json:"state"`
	Reason     string               `json:"reason,omitempty"`
	Provenance DerivationProvenance `json:"provenance"`
}

// DerivationProvenance snapshots the inputs actually considered by a rule.
// Ref identifies a part/key in this profile, including an intermediate result.
type DerivationProvenance struct {
	Inputs []DerivationInput `json:"inputs"`
}

type DerivationInput struct {
	Ref          string `json:"ref"`
	Origin       string `json:"origin"`
	ReviewStatus string `json:"review_status"`
}

// Row is one reconciled profile value.
type Row struct {
	PartID, Key, Value string
	Band               string // green|amber|red|blank|suggested|user|planning|unchecked
	Assertion          string
	Note               string
	Sources            []Source
	Alternatives       []Alternative
	Tenders            string // "", consistent, differ
	Derived            *Derived
	// Provenance (plan §4.1), set by Annotate: the key's owner (site or
	// project), origin, review status, meaning and value state. UserVersion
	// is the user value's version, for optimistic edits.
	Scope        string
	Origin       string
	ReviewStatus string
	Meaning      string
	ValueState   string
	UserVersion  int64
}

// Thresholds are per question shape. A missing amber floor applies nothing;
// a nil green withholds green (https://docs.typesafe.ai/patterns/confidence-routing).
type Thresholds struct {
	LabelMinNoul float64
	Amber        map[string]float64
	Green        map[string]*float64
	Approved     bool
	Version      string
}

// Input is everything one project's reconciliation reads.
type Input struct {
	Parts      []Part
	Facts      []Fact
	User       []UserValue
	Planning   []PlanningValue // live values only
	Thresholds Thresholds
	Suggested  []string // leaf ids typical for the chosen subclass and work type
	// Read drops facts from documents the profile does not read. The zero
	// policy keeps every fact.
	Read ReadPolicy
}

const (
	maxNote      = 120
	maxSources   = 10
	kindTender   = "commercial"
	partWhole    = "whole"
	bandUser     = "user"
	bandGreen    = "green"
	bandAmber    = "amber"
	bandRed      = "red"
	bandBlank    = "blank"
	bandSuggest  = "suggested"
	bandPlanning = "planning"
	valIncluded  = "included"
	assertSuffix = ".assertion"
)

// Reconcile turns readings into rows. It is pure: no I/O, no Jev, no clock.
// Rules: a user value is final; superseded documents are ignored; quotes and
// tenders are one source together because they read the same package; two
// values on one part conflict; allowances and requirements are shown but
// never derived from; a suggestion never outranks evidence.
func Reconcile(in Input, cat *knowledge.Catalog) []Row {
	parts := partIndex(in.Parts)
	assertions := map[string][]Fact{}
	grouped := map[[2]string][]Fact{}
	for _, f := range in.Facts {
		if f.Superseded {
			continue
		}
		key, value, ok := normalise(f.QuestionID, f.Value, cat)
		if !ok {
			continue
		}
		f.Value = value
		if strings.HasSuffix(key, assertSuffix) {
			base := strings.TrimSuffix(key, assertSuffix)
			assertions[f.DocumentID+"|"+f.PassageID+"|"+base] = append(assertions[f.DocumentID+"|"+f.PassageID+"|"+base], f)
			continue
		}
		part := parts.whole
		if id, ok := parts.byLabel[f.PartLabel]; ok && f.PartLabel != "" {
			part = id
		}
		grouped[[2]string{part, key}] = append(grouped[[2]string{part, key}], f)
	}

	rows := map[[2]string]*Row{}
	for k, facts := range grouped {
		r := evidenceRow(k[0], k[1], facts, assertions, in.Thresholds, cat)
		rows[k] = &r
	}
	for _, u := range in.User {
		k := [2]string{u.PartID, u.Key}
		r, ok := rows[k]
		if !ok {
			r = &Row{PartID: u.PartID, Key: u.Key}
			rows[k] = r
		}
		r.Band, r.Value, r.Note, r.Assertion = bandUser, "", u.Note, ""
		if u.Value != nil {
			r.Value = *u.Value
		}
		r.Origin, r.Meaning, r.ValueState, r.UserVersion = orDefault(u.Origin, OriginUser), orDefault(u.Meaning, MeaningStated), userState(u), u.Version
		r.ReviewStatus = orDefault(u.ReviewStatus, ReviewAccepted)
	}
	// A planning value is shown as itself, an assumption or a calculation;
	// the user's stated word on the same key wins (existing precedence).
	for _, p := range in.Planning {
		k := [2]string{p.PartID, knowledge.PlanningPrefix + p.Key}
		if r, ok := rows[k]; ok && r.Band == bandUser {
			continue
		}
		r := &Row{PartID: k[0], Key: k[1], Band: bandPlanning, Note: cut(p.Rationale, maxNote)}
		if p.Value != nil {
			r.Value = *p.Value
		}
		// UserVersion stays 0: the planning version belongs to the planning
		// endpoint, and a profile edit of this key checks the user value's.
		r.Origin, r.ReviewStatus, r.Meaning, r.ValueState = p.Origin, p.ReviewStatus, p.Meaning, p.State
		rows[k] = r
	}
	for _, leaf := range in.Suggested {
		k := [2]string{parts.whole, "sys." + cat.Resolve(leaf) + ".presence"}
		if r, ok := rows[k]; ok && r.Band != bandBlank {
			continue
		}
		r := rows[k]
		if r == nil {
			r = &Row{PartID: k[0], Key: k[1]}
			rows[k] = r
		}
		r.Band, r.Value = bandSuggest, valIncluded
	}
	derive(rows, parts, cat)

	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	// Scope comes from the registry, never from which project supplied the
	// document. The store persists these rows against the part's site.
	Annotate(out, cat)
	sort.Slice(out, func(i, j int) bool {
		if out[i].PartID != out[j].PartID {
			return out[i].PartID < out[j].PartID
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// normalise maps a question id and answer to a row key and value. Silence
// ("not_stated" or a candidate sentinel) is not evidence; enumerated "none" is.
// A per-option class question
// (det.ncc_class.7b answered stated_true) becomes the value 7b of det.ncc_class.
func normalise(qid, value string, cat *knowledge.Catalog) (string, string, bool) {
	if value == "" || value == "not_stated" {
		return "", "", false
	}
	if value == "none" {
		if !enumeratedNone(qid, cat) {
			return "", "", false
		}
	}
	switch {
	case strings.HasPrefix(qid, "sys."):
		rest := strings.TrimPrefix(qid, "sys.")
		dot := strings.LastIndex(rest, ".")
		if dot < 0 {
			return "", "", false
		}
		return "sys." + cat.Resolve(rest[:dot]) + rest[dot:], value, true
	case strings.HasPrefix(qid, "det."):
		rest := strings.TrimPrefix(qid, "det.")
		if strings.HasSuffix(rest, assertSuffix) {
			return "det." + cat.Resolve(strings.TrimSuffix(rest, assertSuffix)) + assertSuffix, value, true
		}
		if id, opt, ok := strings.Cut(rest, "."); ok {
			if d, found := cat.Determinant(id); found && d.Value == "multi_choice" {
				if value != "stated_true" {
					return "", "", false
				}
				return "det." + id, opt, true
			}
		}
		return "det." + cat.Resolve(rest), value, true
	}
	return qid, value, true
}

func enumeratedNone(qid string, cat *knowledge.Catalog) bool {
	var d knowledge.Determinant
	if strings.HasPrefix(qid, "det.") {
		d, _ = cat.Determinant(strings.TrimPrefix(qid, "det."))
	}
	if strings.HasPrefix(qid, "fact.") {
		d, _ = cat.ProjectFact(strings.TrimPrefix(qid, "fact."))
	}
	for _, o := range d.Options {
		if o.ID == "none" {
			return true
		}
	}
	for _, c := range cat.Taxonomy().Conditions {
		if qid == "hdr.cond."+c.Key {
			for _, o := range c.Options {
				if o.ID == "none" {
					return true
				}
			}
		}
	}
	return false
}

func shapeOf(key string) string {
	switch {
	case strings.HasSuffix(key, ".presence"):
		return "presence"
	case strings.HasSuffix(key, ".provider"):
		return "provider"
	case strings.HasSuffix(key, ".action"):
		return "action"
	case strings.HasSuffix(key, assertSuffix):
		return "assertion"
	case strings.HasPrefix(key, "hdr."):
		return "header"
	}
	return "determinant"
}

func applied(f Fact, shape string, th Thresholds) bool {
	if f.DecidedBy == "rule" {
		return true
	}
	floor, ok := th.Amber[shape]
	return ok && f.Confidence != nil && *f.Confidence >= floor
}

func evidenceRow(part, key string, facts []Fact, assertions map[string][]Fact, th Thresholds, cat *knowledge.Catalog) Row {
	r := Row{PartID: part, Key: key, Band: bandBlank, Origin: OriginDocument, ReviewStatus: ReviewProposed}
	shape := shapeOf(key)
	byValue := map[string][]Fact{}
	var order []string
	for _, f := range facts {
		if len(r.Sources) < maxSources {
			r.Sources = append(r.Sources, source(f))
		}
		if !applied(f, shape, th) {
			continue
		}
		if _, seen := byValue[f.Value]; !seen {
			order = append(order, f.Value)
		}
		byValue[f.Value] = append(byValue[f.Value], f)
	}
	if len(order) == 0 {
		if shape == "action" && len(r.Sources) > 0 {
			r.Note = "Needs mapping — action has not passed its own confidence threshold"
		}
		return r
	}
	multi := false
	if strings.HasPrefix(key, "det.") {
		d, _ := cat.Determinant(strings.TrimPrefix(key, "det."))
		multi = d.Value == "multi_choice"
	}
	allTenders := true
	for _, v := range order {
		for _, f := range byValue[v] {
			allTenders = allTenders && f.DocumentKind == kindTender
		}
	}
	switch {
	case len(order) == 1 || multi:
		sort.Strings(order)
		r.Value = strings.Join(order, ",")
		r.Band = bandAmber
		var supporting []Fact
		for _, v := range order {
			supporting = append(supporting, byValue[v]...)
		}
		if isGreen(supporting, shape, th) {
			r.Band = bandGreen
		}
		if allTenders && distinctDocs(supporting) > 1 {
			r.Tenders = "consistent"
		}
		r.Assertion = assertionOf(supporting, assertions, key, th)
		if shape == "presence" && r.Value == valIncluded {
			r.Note = cut(supporting[0].Excerpt, maxNote)
		}
		if shape == "action" && r.Value == "several" {
			r.Note = "Needs mapping — several actions; choose or split the work"
		}
	default:
		r.Band = bandRed
		sort.SliceStable(order, func(i, j int) bool {
			si, sj := support(byValue[order[i]]), support(byValue[order[j]])
			if si != sj {
				return si > sj
			}
			return order[i] < order[j]
		})
		for _, v := range order {
			alt := Alternative{Value: v}
			for _, f := range byValue[v] {
				alt.Sources = append(alt.Sources, source(f))
			}
			r.Alternatives = append(r.Alternatives, alt)
		}
		if allTenders {
			r.Tenders = "differ"
		}
	}
	return r
}

func isGreen(facts []Fact, shape string, th Thresholds) bool {
	// Location application is provisional even if the value itself is certain.
	// A person's explicit value may override this later; evidence cannot.
	for _, f := range facts {
		if f.PartLabel != "" {
			return false
		}
	}
	for _, f := range facts {
		if f.DecidedBy == "rule" {
			return true
		}
	}
	g := th.Green[shape]
	if g == nil {
		return false
	}
	for _, f := range facts {
		if f.Confidence == nil || *f.Confidence < *g {
			return false
		}
	}
	return true
}

// support counts independent documents: all quotes and tenders count once.
func support(facts []Fact) int {
	seen := map[string]bool{}
	for _, f := range facts {
		k := f.DocumentID
		if f.DocumentKind == kindTender {
			k = kindTender
		}
		seen[k] = true
	}
	return len(seen)
}

func distinctDocs(facts []Fact) int {
	seen := map[string]bool{}
	for _, f := range facts {
		seen[f.DocumentID] = true
	}
	return len(seen)
}

// assertionOf is the assertion every supporting passage agrees on, or "".
func assertionOf(facts []Fact, assertions map[string][]Fact, key string, th Thresholds) string {
	got := ""
	for _, f := range facts {
		for _, a := range assertions[f.DocumentID+"|"+f.PassageID+"|"+key] {
			if !applied(a, "assertion", th) {
				continue
			}
			if got != "" && got != a.Value {
				return ""
			}
			got = a.Value
		}
	}
	return got
}

func source(f Fact) Source {
	return Source{DocumentID: f.DocumentID, PassageID: f.PassageID, Excerpt: cut(f.Excerpt, maxNote), Confidence: f.Confidence,
		FileSHA256: f.FileSHA256, Filename: f.Filename, DocumentNumber: f.DocumentNumber, Revision: f.Revision,
		Page: f.Page, Location: f.Location, Section: f.Section, StartOffset: f.StartOffset, EndOffset: f.EndOffset}
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

type partsIdx struct {
	whole   string
	byLabel map[string]string
	ids     []string
}

func partIndex(parts []Part) partsIdx {
	idx := partsIdx{byLabel: map[string]string{}}
	for _, p := range parts {
		idx.byLabel[p.Label] = p.ID
		idx.ids = append(idx.ids, p.ID)
		if p.Kind == partWhole && idx.whole == "" {
			idx.whole = p.ID
		}
	}
	return idx
}

// derive runs table lookups from eligible stated facts. Planning-only results
// can feed later lookups (Type of Construction, then limits), but their
// unverified status must follow the whole chain.
func derive(rows map[[2]string]*Row, parts partsIdx, cat *knowledge.Catalog) {
	var derived []knowledge.Determinant
	for _, d := range cat.ProfileDeterminants() {
		if d.Derived && d.By != "" {
			derived = append(derived, d)
		}
	}
	if len(derived) == 0 {
		return
	}
	for _, part := range parts.ids {
		if part != parts.whole && !hasClass(rows, part) {
			continue
		}
		for pass := 0; pass < 2; pass++ {
			for _, d := range derived {
				facts := usableFacts(rows, part)
				res := cat.Derive(d.By, facts)
				k := [2]string{part, "det." + d.ID}
				if old, ok := rows[k]; ok && old.Band == bandUser {
					continue // the user's word is final, even over a table lookup
				}
				r := &Row{PartID: part, Key: k[1], Band: bandBlank, Origin: OriginCalculation,
					Derived: &Derived{Rule: d.By, State: res.State, Reason: res.Reason}}
				rule, _ := cat.Rule(d.By)
				verified := true
				if rule.Derives != nil {
					for _, input := range rule.Derives.Inputs {
						inputKey := "det." + input
						if source := rows[[2]string{part, inputKey}]; source != nil {
							r.Derived.Provenance.Inputs = append(r.Derived.Provenance.Inputs, DerivationInput{
								Ref: part + "/" + inputKey, Origin: source.Origin, ReviewStatus: orDefault(source.ReviewStatus, ReviewAccepted),
							})
							verified = verified && source.ReviewStatus == ReviewVerified
						} else {
							verified = false
						}
					}
				}
				if res.State == knowledge.StateDetermined {
					r.Value, r.Band = res.Value, bandGreen
					r.ReviewStatus = ReviewVerified
					if !verified {
						r.Band, r.ReviewStatus = bandAmber, ReviewAccepted
						r.Note = "Planning only — inputs not verified"
					}
				}
				rows[k] = r
			}
		}
	}
}

func hasClass(rows map[[2]string]*Row, part string) bool {
	r, ok := rows[[2]string{part, "det.ncc_class"}]
	return ok && r.Value != ""
}

func usableFacts(rows map[[2]string]*Row, part string) []knowledge.Fact {
	var out []knowledge.Fact
	for k, r := range rows {
		if k[0] != part || !strings.HasPrefix(k[1], "det.") || r.Value == "" {
			continue
		}
		if r.ValueState == StateUnknown || r.ValueState == StateCleared || r.ReviewStatus == ReviewSupersede ||
			r.Origin == OriginAssumption || (r.Meaning != "" && r.Meaning != MeaningStated) {
			continue
		}
		usable := eligibleUser(r) || ((r.Band == bandGreen || r.Band == bandAmber) && r.Derived != nil && r.Derived.State == knowledge.StateDetermined) ||
			((r.Band == bandGreen || r.Band == bandAmber) && r.Assertion == "stated")
		if !usable {
			continue
		}
		out = append(out, knowledge.Fact{Determinant: strings.TrimPrefix(k[1], "det."), Value: r.Value,
			Scope: knowledge.Scope{Level: "part", Ref: part}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Determinant < out[j].Determinant })
	return out
}
