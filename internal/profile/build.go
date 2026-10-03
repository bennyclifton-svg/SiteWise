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

// Build reconciles, then adds typical systems for the subclass and work type
// the profile now holds (user, green or amber) and reconciles again.
func Build(in Input, cat *knowledge.Catalog) []Row {
	in.Facts = readFacts(in.Facts, in.Read)
	rows := Reconcile(in, cat)
	sub, work := headerValue(rows, "hdr.subclass"), headerValue(rows, "hdr.work_type")
	if sub == "" || work == "" {
		return rows
	}
	if suggested := cat.Typical(sub, work); len(suggested) > 0 {
		in.Suggested = suggested
		rows = Reconcile(in, cat)
	}
	return rows
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

func headerValue(rows []Row, key string) string {
	for _, r := range rows {
		if r.Key == key && (r.Band == bandUser || r.Band == bandGreen || r.Band == bandAmber) {
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
