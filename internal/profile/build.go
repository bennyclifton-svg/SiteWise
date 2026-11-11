package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
)

// Reading is one profile answer ready to store as a fact.
type Reading struct {
	QuestionID string
	Value      string
	Unit       string
	Basis      string
	Excerpt    string
	Confidence *float64
}

// IsProfileQuestion reports whether a question id belongs to the profile.
func IsProfileQuestion(id string) bool {
	return strings.HasPrefix(id, "det.") || strings.HasPrefix(id, "fact.") ||
		strings.HasPrefix(id, "hdr.") || strings.HasPrefix(id, "sys.") || strings.HasPrefix(id, "sig:") ||
		strings.HasPrefix(id, knowledge.DetailPrefix) || strings.HasPrefix(id, knowledge.SitePrefix) ||
		strings.HasPrefix(id, knowledge.SpacePrefix)
}

// Readings turns one call's profile answers into readings. A pre-parsed pick
// stores the candidate's normalised value with its verbatim context as the
// excerpt; candidate silence and not_stated are not stored. Code copies text; Jev
// never writes it (https://docs.typesafe.ai/model-jaggedness/jev-1.13).
func Readings(result jev.Result, questions map[string]jev.Question, cands map[string][]Candidate, passage string) []Reading {
	excerpt := cut(strings.Join(strings.Fields(passage), " "), maxNote)
	var out []Reading
	for id := range questions {
		if !IsProfileQuestion(id) {
			continue
		}
		a, ok := result.Answers[id]
		if strings.HasPrefix(id, "sig:") {
			if ok && a.Type == jev.TypeNoul && a.Noul >= 0 && a.Noul <= 1 {
				value, probability := "false", 1-a.Noul
				if a.Noul >= .5 {
					value, probability = "true", a.Noul
				}
				out = append(out, Reading{QuestionID: strings.TrimPrefix(id, "sig:"), Value: value, Excerpt: excerpt, Confidence: &probability, Basis: "noul_probability"})
			}
			continue
		}
		if !ok || a.Type != jev.TypeChoice || a.Choice == "" || a.Choice == "not_stated" {
			continue
		}
		// "none" is an extraction sentinel only for candidate questions. It is
		// also a real enumerated value, for example no fuel gas supply.
		if _, candidateQuestion := cands[id]; candidateQuestion && a.Choice == "none" {
			continue
		}
		if criteria, ok := questions[id].Criteria.(map[string]string); ok {
			if _, offered := criteria[a.Choice]; !offered {
				continue
			}
		}
		r := Reading{QuestionID: id, Value: a.Choice, Excerpt: excerpt, Confidence: a.Confidence}
		if strings.HasSuffix(id, ".presence") && r.Value == "included_by_others" {
			r.Value = "included"
		}
		if list, ok := cands[id]; ok {
			c, found := pick(list, a.Choice)
			if !found {
				continue // an option we did not offer is not evidence
			}
			r.Value, r.Unit, r.Basis, r.Excerpt = c.Norm, c.Unit, c.Basis, cut(c.Context, maxNote)
			if r.Value == "" {
				r.Value = c.Value
			}
		}
		out = append(out, r)
	}
	return out
}

func pick(list []Candidate, id string) (Candidate, bool) {
	for _, c := range list {
		if c.ID == id {
			return c, true
		}
	}
	return Candidate{}, false
}

// Build reconciles, then works out the project's scope of works: the
// defaults for its building category, class and work type, systems a read
// document includes, and the user's choices, which are final. Defaults the
// user has not removed are suggested in the checklist. Scope rows are
// "scope.<leaf>" on the whole project: value in or out; band suggested for a
// default, the evidence band for a document, user for the user.
func Build(in Input, cat *knowledge.Catalog) []Row {
	in.Facts = readFacts(in.Facts, in.Read)
	var implied map[string]string
	in.Facts, implied = splitImplied(in.Facts)
	// Defaults depend only on headers. Derivations produce determinant rows,
	// so reconciling every source twice cannot change these header choices.
	rows := Reconcile(scopeHeaders(in), cat)
	whole := wholePart(in.Parts)
	category, class, work := headerValue(rows, whole, "hdr.building_class"), headerValue(rows, whole, "hdr.subclass"), headerValue(rows, whole, "hdr.work_type")
	defaults := cat.ScopeDefaults(category, class, work)
	removed := map[string]bool{}
	for _, u := range in.User {
		if leaf, ok := strings.CutPrefix(u.Key, scopePrefix); ok && u.Value != nil && *u.Value == scopeOut {
			removed[leaf] = true
		}
	}
	var suggested []string
	for _, leaf := range defaults {
		if !removed[leaf] {
			suggested = append(suggested, leaf)
		}
	}
	if len(suggested) > 0 {
		in.Suggested = suggested
	}
	rows = Reconcile(in, cat)
	rows = withoutInapplicableConditions(rows, category, class, cat)
	rows = withExistingSystems(rows, whole)
	rows = withScope(rows, whole, suggested)
	rows = withImpliedScope(rows, whole, implied)
	rows = withProposedGFA(rows, whole)
	rows = withAddressState(rows, whole)
	// Scope suggestions are added after reconciliation and need provenance too.
	Annotate(rows, cat)
	return rows
}

func scopeHeaders(in Input) Input {
	out := Input{Parts: in.Parts, Thresholds: in.Thresholds}
	for _, f := range in.Facts {
		if strings.HasPrefix(f.QuestionID, "hdr.") {
			out.Facts = append(out.Facts, f)
		}
	}
	for _, u := range in.User {
		if strings.HasPrefix(u.Key, "hdr.") {
			out.User = append(out.User, u)
		}
	}
	return out
}

const (
	scopePrefix = "scope."
	scopeIn     = "in"
	scopeOut    = "out"
)

// withScope adds scope rows for defaults and for systems included by
// evidence or by the user's checklist answer. A user scope row (made by
// Reconcile from the user's value) is never replaced; when the user removed
// a system a document includes, its note says so.
func withScope(rows []Row, whole string, defaults []string) []Row {
	if whole == "" {
		return rows
	}
	work := workContextFor(rows, whole)
	byKey := map[string]int{}
	for i, r := range rows {
		if r.PartID == whole {
			byKey[r.Key] = i
		}
	}
	add := func(leaf, band string) {
		key := scopePrefix + leaf
		if i, ok := byKey[key]; ok {
			if rows[i].Band != bandUser && band != bandSuggest {
				rows[i].Band = band
			}
			return
		}
		byKey[key] = len(rows)
		rows = append(rows, Row{PartID: whole, Key: key, Value: scopeIn, Band: band})
	}
	for _, leaf := range defaults {
		add(leaf, bandSuggest)
	}
	for _, r := range rows {
		if work.existingPresence(r) || work.severalActions(r) {
			continue
		}
		leaf, ok := strings.CutPrefix(r.Key, "sys.")
		if !ok || r.PartID != whole || !strings.HasSuffix(leaf, ".presence") || r.Value != valIncluded {
			continue
		}
		leaf = strings.TrimSuffix(leaf, ".presence")
		switch r.Band {
		case bandAmber, bandGreen, bandUser:
		default:
			continue
		}
		if i, ok := byKey[scopePrefix+leaf]; ok && rows[i].Band == bandUser && rows[i].Value == scopeOut {
			if r.Band != bandUser {
				rows[i].Note = cut("A document says it is included: "+firstExcerpt(r), maxNote)
			}
			continue
		}
		add(leaf, r.Band)
	}
	// An applied works action names a system the works touch: a stair to
	// relocate, a bathroom to upgrade. It puts the system in scope when its
	// presence answer fell short, unless the documents state it is not
	// included. Retain and investigate do no work on the system.
	notIncluded := map[string]bool{}
	for _, r := range rows {
		if leaf, ok := strings.CutPrefix(r.Key, "sys."); ok && r.PartID == whole && strings.HasSuffix(leaf, ".presence") && r.Value == "not_included" && appliedWorkRow(r) {
			notIncluded[strings.TrimSuffix(leaf, ".presence")] = true
		}
	}
	ctx := workContextFor(rows, whole)
	for _, r := range rows {
		leaf, ok := strings.CutPrefix(r.Key, "sys.")
		if !ok || r.PartID != whole || !strings.HasSuffix(leaf, ".action") || !appliedWorkRow(r) || !scopeAction[r.Value] || ctx.oldBuildingAction(r) {
			continue
		}
		leaf = strings.TrimSuffix(leaf, ".action")
		if notIncluded[leaf] {
			continue
		}
		if i, ok := byKey[scopePrefix+leaf]; ok && rows[i].Band == bandUser && rows[i].Value == scopeOut {
			if r.Band != bandUser && rows[i].Note == "" {
				rows[i].Note = cut("A document says the works "+r.Value+" it: "+firstExcerpt(r), maxNote)
			}
			continue
		}
		add(leaf, r.Band)
	}
	return rows
}

// withImpliedScope suggests the systems the documents' own words imply
// ("ensuite" implies wet-area waterproofing). It never overrides a row:
// the user's choice and document evidence stand, and a default keeps its band.
func withImpliedScope(rows []Row, whole string, implied map[string]string) []Row {
	if whole == "" || len(implied) == 0 {
		return rows
	}
	have := map[string]bool{}
	for _, r := range rows {
		if r.PartID == whole {
			have[r.Key] = true
		}
	}
	leaves := make([]string, 0, len(implied))
	for leaf := range implied {
		leaves = append(leaves, leaf)
	}
	sort.Strings(leaves)
	for _, leaf := range leaves {
		if have[scopePrefix+leaf] {
			continue
		}
		rows = append(rows, Row{PartID: whole, Key: scopePrefix + leaf, Value: scopeIn, Band: bandSuggest,
			Note: cut("Implied by the documents' words: \""+implied[leaf]+"\"", maxNote)})
	}
	return rows
}

// withProposedGFA computes the building's GFA after the works as existing
// plus added, when both are applied and no proposed GFA is stated or typed.
// One unknown input leaves it unknown.
func withProposedGFA(rows []Row, whole string) []Row {
	const proposed, existing, added = "hdr.scale.gfa_sqm", "hdr.scale.existing_gfa_sqm", "hdr.scale.added_gfa_sqm"
	at := map[string]int{}
	for i, r := range rows {
		if r.PartID == whole {
			at[r.Key] = i
		}
	}
	if i, ok := at[proposed]; ok && strings.TrimSpace(rows[i].Value) != "" {
		return rows
	}
	ei, eok := at[existing]
	ai, aok := at[added]
	if !eok || !aok || !appliedWorkRow(rows[ei]) || !appliedWorkRow(rows[ai]) {
		return rows
	}
	e, err1 := strconv.ParseFloat(strings.ReplaceAll(rows[ei].Value, ",", ""), 64)
	a, err2 := strconv.ParseFloat(strings.ReplaceAll(rows[ai].Value, ",", ""), 64)
	if err1 != nil || err2 != nil || e < 0 || a < 0 {
		return rows
	}
	// A sum, even of two typed values, is a calculation: amber with its
	// derivation, never the user's own word.
	r := Row{PartID: whole, Key: proposed, Value: strconv.FormatFloat(e+a, 'f', -1, 64), Band: bandAmber,
		Note: fmt.Sprintf("Existing %s m² plus %s m² added by the works.", rows[ei].Value, rows[ai].Value),
		Derived: &Derived{Rule: "proposed GFA = existing GFA + added GFA", State: knowledge.StateDetermined,
			Provenance: DerivationProvenance{Inputs: []DerivationInput{
				{Ref: whole + "/" + existing, Origin: rows[ei].Origin, ReviewStatus: rows[ei].ReviewStatus},
				{Ref: whole + "/" + added, Origin: rows[ai].Origin, ReviewStatus: rows[ai].ReviewStatus}}}}}
	if i, ok := at[proposed]; ok {
		rows[i] = r
		return rows
	}
	return append(rows, r)
}

// scopeAction are the works actions that do work on a system.
var scopeAction = map[string]bool{"new": true, "replace": true, "upgrade": true, "alter": true, "repair": true, "remove": true}

// withoutInapplicableConditions drops a read header condition whose
// taxonomy applies_to leaves out the project's building category, so a
// house is not given an industrial hazardous process from asbestos
// passages (knock-down D5, docs/plans/2026-11-06-knockdown-two-buildings.md).
// The user's value is kept; an unknown category keeps every row.
func withoutInapplicableConditions(rows []Row, category, class string, cat *knowledge.Catalog) []Row {
	tax := cat.Taxonomy()
	if category == "" {
		for _, bc := range tax.BuildingClasses {
			for _, s := range bc.Subclasses {
				if class != "" && s.ID == class {
					category = bc.ID
				}
			}
		}
	}
	if category == "" {
		return rows
	}
	limited := map[string]bool{}
	for _, c := range tax.Conditions {
		if len(c.AppliesTo) > 0 && !slices.Contains(c.AppliesTo, category) {
			limited["hdr.cond."+c.Key] = true
		}
	}
	out := rows[:0]
	for _, r := range rows {
		if limited[r.Key] && r.Band != bandUser {
			continue
		}
		out = append(out, r)
	}
	return out
}

func firstExcerpt(r Row) string {
	for _, s := range r.Sources {
		if s.Excerpt != "" {
			return s.Excerpt
		}
	}
	return r.Note
}

func wholePart(parts []Part) string {
	for _, p := range parts {
		if p.Kind == partWhole {
			return p.ID
		}
	}
	return ""
}

// readFacts keeps the facts of documents the policy reads. A document turned
// off leaves the profile at the next rebuild, which is code only; its stored
// readings stay, so turning it back on costs no Jev call.
func readFacts(facts []Fact, p ReadPolicy) []Fact {
	if !p.Loaded() {
		return facts
	}
	out := make([]Fact, 0, len(facts))
	for _, f := range facts {
		if p.Reads(f.DocumentKind, f.ReadSetting) {
			out = append(out, f)
		}
	}
	return out
}

func headerValue(rows []Row, part, key string) string {
	for _, r := range rows {
		if r.PartID == part && r.Key == key && appliedWorkRow(r) {
			return r.Value
		}
	}
	return ""
}

type thresholdsFile struct {
	QuestionVersion string  `json:"question_version"`
	Status          string  `json:"status"`
	Approved        bool    `json:"approved_by_owner"`
	LabelMinNoul    float64 `json:"label_min_noul"`
	Shapes          map[string]struct {
		Amber *float64 `json:"amber"`
		Green *float64 `json:"green"`
	} `json:"shapes"`
}

// LoadThresholds reads data/profile/thresholds.json. A missing file returns
// empty thresholds, which apply nothing, and os.ErrNotExist.
func LoadThresholds(path string) (Thresholds, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Thresholds{Amber: map[string]float64{}, Green: map[string]*float64{}}, err
	}
	var f thresholdsFile
	if err := json.Unmarshal(body, &f); err != nil {
		return Thresholds{}, err
	}
	if f.QuestionVersion == "" {
		return Thresholds{}, errors.New("profile thresholds need a question_version")
	}
	t := Thresholds{LabelMinNoul: f.LabelMinNoul, Approved: f.Approved, Version: f.QuestionVersion,
		Amber: map[string]float64{}, Green: map[string]*float64{}}
	for shape, v := range f.Shapes {
		if v.Amber != nil {
			if *v.Amber <= 0 || *v.Amber > 1 {
				return Thresholds{}, errors.New("profile amber floor must be in (0, 1]: " + shape)
			}
			t.Amber[shape] = *v.Amber
		}
		t.Green[shape] = v.Green
	}
	return t, nil
}
