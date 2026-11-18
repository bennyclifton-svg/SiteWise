package consent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"

	"sitewise/internal/delivery"
)

// The answer key (docs/plans/2026-11-17-consent-conditions.md, "Answer
// key"). It lives in data/eval/consent/private/, outside the public tree's
// history: one file per notice, per condition its number, page, gate,
// party, system and whether evidence is needed, and never the condition's
// words. An agent may draft it; only the owner marks it reviewed, and no
// pick shows green or is accepted until then.

// Key is one notice's labels.
type Key struct {
	// Document names the notice; Source is where its text or file is on the
	// owner's machine (read by cmd/consent-eval only).
	Document string `yaml:"document"`
	Source   string `yaml:"source"`
	Reviewed bool   `yaml:"reviewed"`
	// Modification says the notice modifies a consent; its conditions carry
	// a marker.
	Modification bool           `yaml:"modification"`
	Conditions   []KeyCondition `yaml:"conditions"`
	// Evidence maps later documents to the conditions they evidence
	// (Task 8): about 20 Petersham consultant documents.
	Evidence []KeyEvidence `yaml:"evidence"`
	file     string
}

// KeyCondition is one condition's labels.
type KeyCondition struct {
	Number   string `yaml:"number"`
	Page     int    `yaml:"page"`
	Stage    string `yaml:"stage"`
	Party    string `yaml:"party"`
	System   string `yaml:"system"`
	Evidence *bool  `yaml:"evidence"`
	Marker   string `yaml:"marker"` // deleted, changed or ""
}

// KeyEvidence is one later document and the conditions it evidences.
type KeyEvidence struct {
	Document   string   `yaml:"document"`
	Conditions []string `yaml:"conditions"`
}

// LoadKeys reads every *.yaml in dir. A missing directory is no key: the
// tests that need it skip.
func LoadKeys(dir string) ([]Key, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	sort.Strings(names)
	var out []Key
	for _, name := range names {
		body, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var k Key
		if err := yaml.Unmarshal(body, &k); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		k.file = name
		if err := k.check(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, k)
	}
	return out, nil
}

func (k Key) check() error {
	if k.Document == "" {
		return errors.New("document is needed")
	}
	seen := map[string]bool{}
	for _, c := range k.Conditions {
		switch {
		case c.Number == "" || seen[c.Number]:
			return fmt.Errorf("condition %q is missing or repeated", c.Number)
		case c.Stage != "" && !slices.Contains(delivery.Gates, c.Stage):
			return fmt.Errorf("condition %s: unknown stage %q", c.Number, c.Stage)
		case c.Party != "" && !KnownParty(c.Party):
			return fmt.Errorf("condition %s: unknown party %q", c.Number, c.Party)
		case c.Marker != "" && c.Marker != "deleted" && c.Marker != "changed":
			return fmt.Errorf("condition %s: marker is deleted or changed", c.Number)
		}
		seen[c.Number] = true
	}
	return nil
}

// SplitScore compares a split with a key: the split must match exactly,
// number for number and page for page, before any pick is scored.
func SplitScore(k Key, n Notice) (exact bool, problems []string) {
	got := map[string]Condition{}
	for _, c := range n.Conditions {
		got[c.Number] = c
	}
	want := map[string]bool{}
	for _, c := range k.Conditions {
		want[c.Number] = true
		g, ok := got[c.Number]
		switch {
		case !ok:
			problems = append(problems, "missed "+c.Number)
		case c.Page > 0 && g.Page != c.Page:
			problems = append(problems, fmt.Sprintf("%s on page %d, key says %d", c.Number, g.Page, c.Page))
		case c.Marker != g.Marker:
			problems = append(problems, fmt.Sprintf("%s marker %q, key says %q", c.Number, g.Marker, c.Marker))
		}
	}
	for _, c := range n.Conditions {
		if !want[c.Number] {
			problems = append(problems, "extra "+c.Number)
		}
	}
	return len(problems) == 0, problems
}

// PagesFile reads a notice's pages as cmd/consent-eval -pages wrote them:
// [{"page":1,"text":"…"}].
func PagesFile(path string) ([]Page, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Page int    `json:"page"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]Page, len(raw))
	for i, p := range raw {
		out[i] = Page{Page: p.Page, Text: p.Text}
	}
	return out, nil
}
