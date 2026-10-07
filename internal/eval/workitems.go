package eval

import (
	"fmt"
	"sort"

	"sitewise/internal/profile"
	"sitewise/internal/works"
)

// WorkKey holds semantic expectations, never generated record IDs or private
// source prose. Only an owner-reviewed key can gate extraction quality.
type WorkKey struct {
	SchemaVersion int            `yaml:"schema_version" json:"schema_version"`
	Project       string         `yaml:"project" json:"project"`
	Reviewed      bool           `yaml:"reviewed" json:"reviewed"`
	LabelSource   string         `yaml:"label_source" json:"label_source"`
	Documents     []string       `yaml:"documents" json:"documents"`
	Parts         []string       `yaml:"parts" json:"parts"`
	Items         []ExpectedWork `yaml:"work_items" json:"work_items"`
	Notes         []string       `yaml:"notes" json:"notes"`
}

type ExpectedWork struct {
	ID       string `yaml:"id" json:"id"`
	System   string `yaml:"system" json:"system"`
	Action   string `yaml:"action" json:"action"`
	Part     string `yaml:"part" json:"part"`
	Document string `yaml:"document" json:"document"`
	Page     int    `yaml:"page" json:"page"`
}

// WorkSnapshot is an offline export of canonical parts/work items. Version and
// corpus pins travel with results so a changed input is never a silent rerun.
type WorkSnapshot struct {
	Project           string            `json:"project"`
	Model             string            `json:"model"`
	QuestionVersion   string            `json:"question_version"`
	KnowledgeVersion  string            `json:"knowledge_version"`
	ThresholdsVersion string            `json:"thresholds_version"`
	AppBuild          string            `json:"app_build"`
	KeySHA256         string            `json:"key_sha256"`
	Corpus            map[string]string `json:"corpus"`
	Parts             []profile.Part    `json:"parts"`
	Items             []works.Item      `json:"items"`
}

type WorkScore struct {
	Project      string   `json:"project"`
	Reviewed     bool     `json:"reviewed"`
	TP           int      `json:"true_positive"`
	FP           int      `json:"false_positive"`
	FN           int      `json:"false_negative"`
	Precision    *float64 `json:"precision"`
	Recall       *float64 `json:"recall"`
	Missing      []string `json:"missing"`
	Extra        []string `json:"extra"`
	GateEligible bool     `json:"gate_eligible"`
	GatePassed   bool     `json:"gate_passed"`
}

// ScoreWorkItems uses one-to-one system/part/action matches (D-34). A duplicate
// is an extra item, and an abstention misses every required keyed item. Groups,
// retired rows and exclusions are not physical included work.
func ScoreWorkItems(key WorkKey, actual WorkSnapshot) (WorkScore, error) {
	out := WorkScore{Project: key.Project, Reviewed: key.Reviewed, Missing: []string{}, Extra: []string{}}
	if key.Project == "" || key.Project != actual.Project || key.SchemaVersion != 2 || len(key.Items) == 0 {
		return out, fmt.Errorf("nonempty version-2 work key and matching project required")
	}
	expected := map[[3]string]string{}
	ids := map[string]bool{}
	parts := map[string]bool{}
	docs := map[string]bool{}
	for _, p := range key.Parts {
		if p == "" || parts[p] {
			return out, fmt.Errorf("invalid key parts")
		}
		parts[p] = true
	}
	for _, d := range key.Documents {
		if d == "" || docs[d] {
			return out, fmt.Errorf("invalid key documents")
		}
		docs[d] = true
	}
	for _, item := range key.Items {
		semantic := [3]string{item.System, item.Part, item.Action}
		if item.ID == "" || ids[item.ID] || item.System == "" || !works.ValidAction(item.Action) || !parts[item.Part] || !docs[item.Document] || item.Page < 1 {
			return out, fmt.Errorf("invalid keyed work item %q", item.ID)
		}
		if expected[semantic] != "" {
			return out, fmt.Errorf("duplicate keyed work semantics")
		}
		expected[semantic] = item.ID
		ids[item.ID] = true
	}
	actualParts := map[string]string{}
	labels := map[string]bool{}
	for _, p := range actual.Parts {
		if p.ID == "" || p.Label == "" || actualParts[p.ID] != "" || labels[p.Label] {
			return out, fmt.Errorf("invalid snapshot parts")
		}
		actualParts[p.ID] = p.Label
		labels[p.Label] = true
	}
	for _, item := range actual.Items {
		if item.RetiredAt != nil || item.IsGroup || item.Inclusion == "excluded" {
			continue
		}
		if item.Inclusion != "included" || item.ID == "" || actualParts[item.PartID] == "" {
			return out, fmt.Errorf("invalid snapshot work item")
		}
		semantic := [3]string{item.SystemID, actualParts[item.PartID], item.Action}
		if _, ok := expected[semantic]; ok {
			out.TP++
			delete(expected, semantic)
		} else {
			out.FP++
			out.Extra = append(out.Extra, item.ID)
		}
	}
	for _, id := range expected {
		out.Missing = append(out.Missing, id)
	}
	sort.Strings(out.Missing)
	sort.Strings(out.Extra)
	out.FN = len(out.Missing)
	if out.TP+out.FP > 0 {
		v := float64(out.TP) / float64(out.TP+out.FP)
		out.Precision = &v
	}
	if out.TP+out.FN > 0 {
		v := float64(out.TP) / float64(out.TP+out.FN)
		out.Recall = &v
	}
	out.GateEligible = key.Reviewed
	out.GatePassed = out.GateEligible && out.Precision != nil && out.Recall != nil && *out.Precision >= .9 && *out.Recall >= .9
	return out, nil
}
