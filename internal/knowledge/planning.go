package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Planning keys (SCHEMA.md "Planning keys"): the registry of keys a
// planning value (an assumption or a calculated value) may use. A planning
// value is shown on the profile as row "plan.<key>".

// PlanningPrefix starts the profile row key of a planning value.
const PlanningPrefix = "plan."

var planningKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// PlanningOption is one allowed value of a choice planning key.
type PlanningOption struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

// PlanningKey is one registered planning key.
type PlanningKey struct {
	Key     string           `yaml:"key"`
	Label   string           `yaml:"label"`
	Value   string           `yaml:"value"` // integer, number, boolean, choice or text
	Unit    string           `yaml:"unit"`
	Scope   string           `yaml:"scope"`
	Options []PlanningOption `yaml:"options"`
	// Favourable are values code or an assumption may never record:
	// structural adequacy, ground conditions and compliance are not defaulted
	// as favourable (L276). Only a person's stated word may say so.
	Favourable []string `yaml:"favourable"`
	// StartingValue is offered as a starting assumption only once the owner
	// has reviewed the registry (NW-REQ-185); a draft registry offers none.
	StartingValue string `yaml:"starting_value"`
}

// IsFavourable reports whether v is a value no assumption may take.
func (k PlanningKey) IsFavourable(v string) bool { return slices.Contains(k.Favourable, v) }

type planningData struct {
	status string
	keys   []PlanningKey
	byKey  map[string]int
}

// PlanningKey returns the registered key (without the "plan." prefix).
func (c *Catalog) PlanningKey(key string) (PlanningKey, bool) {
	i, ok := c.planning.byKey[key]
	if !ok {
		return PlanningKey{}, false
	}
	return c.planning.keys[i], true
}

// PlanningKeys lists the registry in file order.
func (c *Catalog) PlanningKeys() []PlanningKey { return c.planning.keys }

// StartingValues are the starting assumptions code may propose. None until
// the owner marks the registry reviewed: suggestions read only approved
// libraries (NW-REQ-185), and none exist yet.
func (c *Catalog) StartingValues() []PlanningKey {
	if c.planning.status != "reviewed" {
		return nil
	}
	var out []PlanningKey
	for _, k := range c.planning.keys {
		if k.StartingValue != "" {
			out = append(out, k)
		}
	}
	return out
}

// CheckPlanningValue checks a value against its key's type and options.
// "" means valid.
func (k PlanningKey) CheckPlanningValue(v string) string {
	switch k.Value {
	case "integer":
		if !integerPattern.MatchString(v) {
			return k.Key + " must be a whole number"
		}
	case "number":
		if !numberPattern.MatchString(v) {
			return k.Key + " must be a number"
		}
	case "boolean":
		if v != "true" && v != "false" {
			return k.Key + " must be true or false"
		}
	case "choice":
		for _, o := range k.Options {
			if o.ID == v {
				return ""
			}
		}
		return "value is not an option for " + k.Key
	}
	return ""
}

var (
	integerPattern = regexp.MustCompile(`^-?[0-9]+$`)
	numberPattern  = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
)

// loadPlanningKeys reads profile/planning_keys.yaml when present. Money
// totals belong to the cost plan (L329), so the registry refuses them.
func (c *Catalog) loadPlanningKeys(dir string) error {
	p := filepath.Join(dir, "planning_keys.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version int           `yaml:"version"`
		Status  string        `yaml:"status"`
		Keys    []PlanningKey `yaml:"keys"`
	}
	if err := unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	c.planning = planningData{status: file.Status, byKey: map[string]int{}}
	for i, k := range file.Keys {
		if err := k.validate(); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if _, dup := c.planning.byKey[k.Key]; dup {
			return fmt.Errorf("%s: duplicate key %s", p, k.Key)
		}
		c.planning.byKey[k.Key] = i
	}
	c.planning.keys = file.Keys
	return nil
}

func (k PlanningKey) validate() error {
	switch {
	case !planningKeyPattern.MatchString(k.Key) || strings.HasPrefix(k.Key, "cost"):
		return fmt.Errorf("planning key %q: lower-case words and underscores, and no cost keys (money totals belong to the cost plan)", k.Key)
	case k.Label == "":
		return fmt.Errorf("planning key %s needs a label", k.Key)
	case !slices.Contains([]string{"integer", "number", "boolean", "choice", "text"}, k.Value):
		return fmt.Errorf("planning key %s: value %q (money totals belong to the cost plan)", k.Key, k.Value)
	case k.Scope != ScopeSite && k.Scope != ScopeProject:
		return fmt.Errorf("planning key %s: scope %q", k.Key, k.Scope)
	case k.Value == "choice" && len(k.Options) == 0:
		return fmt.Errorf("planning key %s: a choice needs options", k.Key)
	}
	for _, v := range k.Favourable {
		if msg := k.CheckPlanningValue(v); msg != "" {
			return fmt.Errorf("planning key %s: favourable %q: %s", k.Key, v, msg)
		}
	}
	if k.StartingValue != "" {
		if msg := k.CheckPlanningValue(k.StartingValue); msg != "" || k.IsFavourable(k.StartingValue) {
			return fmt.Errorf("planning key %s: starting value %q is invalid or favourable", k.Key, k.StartingValue)
		}
	}
	return nil
}
