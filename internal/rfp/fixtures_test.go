package rfp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
	"sitewise/internal/knowledge"
)

// scenario is one selection answer key from data/eval/rfp/answer-keys.
type scenario struct {
	SchemaVersion int    `yaml:"schema_version"`
	Reviewed      bool   `yaml:"reviewed"`
	LabelSource   string `yaml:"label_source"`
	Name          string `yaml:"scenario"`
	Description   string `yaml:"description"`
	Facts         map[string]yaml.Node
	Flags         map[string]string `yaml:"flags"`
	Parts         []struct {
		ID      string   `yaml:"id"`
		Label   string   `yaml:"label"`
		Kind    string   `yaml:"kind"`
		Classes []string `yaml:"classes"`
		GFA     string   `yaml:"gfa"`
	} `yaml:"parts"`
	Address     string                               `yaml:"address"`
	Documents   int                                  `yaml:"documents"`
	Consultants []Consultant                         `yaml:"consultants"`
	Fields      map[string]string                    `yaml:"fields"`
	Settings    map[string]map[string]yaml.Node      `yaml:"settings"`
	Expect      map[string]expectation               `yaml:"expect"`
	Sources     []string                             `yaml:"sources"`
	Snapshot    bool                                 `yaml:"snapshot"`
	Extra       map[string]map[string]map[string]any `yaml:"-"`
}

type expectation struct {
	StagesInclude   []string          `yaml:"stages_include"`
	StagesOption    []string          `yaml:"stages_option"`
	StagesAbsent    []string          `yaml:"stages_absent"`
	StagesAsk       []string          `yaml:"stages_ask"`
	AllowPresent    []string          `yaml:"allowances_present"`
	AllowAbsent     []string          `yaml:"allowances_absent"`
	ElementsInclude []string          `yaml:"elements_include"`
	ElementsOption  []string          `yaml:"elements_option"`
	ElementsAbsent  []string          `yaml:"elements_absent"`
	Positions       map[string]string `yaml:"positions"`
	Party           map[string]string `yaml:"party"`
	Blocking        []string          `yaml:"blocking"`
	BlockingOnly    bool              `yaml:"blocking_only"`
	Checks          []string          `yaml:"checks"`
	NoChecks        []string          `yaml:"no_checks"`
	Scale           string            `yaml:"scale"`
	Role            []string          `yaml:"role"`
	RoleAbsent      []string          `yaml:"role_absent"`
}

const answerKeyDir = "../../data/eval/rfp/answer-keys"

var catalogOnce struct {
	sync.Once
	cat *knowledge.Catalog
	err error
}

func loadCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	catalogOnce.Do(func() { catalogOnce.cat, catalogOnce.err = knowledge.Load("../../knowledge") })
	if catalogOnce.err != nil {
		t.Fatal(catalogOnce.err)
	}
	return catalogOnce.cat
}

func loadScenarios(t *testing.T) []scenario {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(answerKeyDir, "*.yaml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no answer keys in %s: %v", answerKeyDir, err)
	}
	sort.Strings(paths)
	out := []scenario{}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var s scenario
		if err := yaml.Unmarshal(body, &s); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if s.Name == "" || s.SchemaVersion != 1 || s.Reviewed && s.LabelSource == "agent-draft" {
			t.Fatalf("%s: answer key needs a scenario name and schema_version 1; an agent draft cannot be reviewed", path)
		}
		out = append(out, s)
	}
	return out
}

// drivers builds the frozen driver set a scenario describes.
func (s scenario) drivers(t *testing.T, sk knowledge.RFPSkeleton, discipline knowledge.RFPDiscipline) DriverSet {
	t.Helper()
	d := DriverSet{Facts: map[string]Fact{}, Text: map[string]string{"address": s.Address}, Documents: s.Documents, Consultants: []Consultant{}}
	// The real driver set excludes the RFP's own package; fixtures list
	// consultants by title, so drop the one this discipline is.
	for _, c := range s.Consultants {
		own := false
		for _, alias := range discipline.TitleAliases {
			own = own || strings.Contains(strings.ToLower(c.Title), alias)
		}
		if !own {
			d.Consultants = append(d.Consultants, c)
		}
	}
	for key, node := range s.Facts {
		if node.Kind == yaml.ScalarNode {
			d.Facts[key] = Fact{Values: []string{node.Value}, State: Known, Source: "profile"}
			continue
		}
		var f struct {
			State  string   `yaml:"state"`
			Values []string `yaml:"values"`
		}
		if err := node.Decode(&f); err != nil {
			t.Fatalf("%s fact %s: %v", s.Name, key, err)
		}
		d.Facts[key] = Fact{Values: f.Values, State: f.State, Source: "profile"}
	}
	for key, v := range s.Flags {
		d.Facts["flag."+key] = Fact{Values: []string{v}, State: Known, Source: "profile"}
	}
	for _, p := range s.Parts {
		kind := p.Kind
		if kind == "" {
			kind = "building"
		}
		d.Parts = append(d.Parts, Part{ID: p.ID, Label: p.Label, Kind: kind, Classes: p.Classes, GFA: p.GFA})
	}
	deriveFlags(d.Facts, d.Parts)
	for _, f := range sk.Flags {
		if _, ok := d.Facts["flag."+f.Key]; !ok {
			d.Facts["flag."+f.Key] = Fact{State: Unknown, Source: "none"}
		}
	}
	return d
}

// settings merges the scenario's issue fields with one discipline's choices.
func (s scenario) settings(t *testing.T, discipline string, drivers DriverSet) Settings {
	t.Helper()
	out := Settings{}
	for key, v := range s.Fields {
		raw, _ := json.Marshal(v)
		out["field:"+key] = Setting{Value: raw}
	}
	for key, node := range s.Settings[discipline] {
		var v any
		if err := node.Decode(&v); err != nil {
			t.Fatalf("%s setting %s: %v", s.Name, key, err)
		}
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out[key] = Setting{Value: raw, Basis: SettingBasis(drivers, key)}
	}
	return out
}
