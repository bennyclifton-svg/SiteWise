package profile

import (
	"encoding/json"
	"errors"
	"os"
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
		strings.HasPrefix(id, "hdr.") || strings.HasPrefix(id, "sys.")
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
	rows := Reconcile(in, cat)
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
		rows = Reconcile(in, cat)
	}
	rows = withExistingSystems(rows, whole)
	rows = withScope(rows, whole, suggested)
	// Scope suggestions are added after reconciliation and need provenance too.
	Annotate(rows, cat)
	return rows
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
	return rows
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
