package knowledge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	intakevocab "sitewise/data/intake"
	"strings"
)

type CostElement struct {
	System    string            `yaml:"system" json:"system"`
	Code      string            `yaml:"code" json:"code"`
	Label     string            `yaml:"cost_label" json:"label"`
	Trade     string            `yaml:"trade"`
	Itemised  bool              `yaml:"itemised"`
	Overrides []CostOverride    `yaml:"overrides"`
	Items     []CostElementItem `yaml:"items"`
}
type CostOverride struct {
	Sub   string `yaml:"sub"`
	Trade string `yaml:"trade"`
}
type CostElementItem struct {
	Sub    string `yaml:"sub"`
	Label  string `yaml:"cost_label"`
	Costed bool   `yaml:"costed"`
	Why    string `yaml:"why"`
}
type CostStarter struct {
	ID          string `yaml:"id"`
	Label       string `yaml:"label"`
	Category    string `yaml:"category"`
	AppliesWhen any    `yaml:"applies_when"`
}

func (c *Catalog) CostElements() []CostElement { return c.costElements }
func (c *Catalog) CostStarters() []CostStarter { return c.costStarters }

func (c *Catalog) loadCostElements(root string) error {
	path := filepath.Join(root, "costs", "elements.yaml")
	if ok, e := exists(path); !ok {
		return e
	}
	var file struct {
		Version  int
		Status   string
		Elements []CostElement
		Sources  []map[string]string
	}
	if e := c.unmarshal(path, &file); e != nil {
		return e
	}
	if file.Version != 1 || (file.Status != "draft" && file.Status != "reviewed") || len(file.Sources) == 0 {
		return fmt.Errorf("invalid cost elements metadata")
	}
	// Use the existing intake vocabulary, not a second list of trade names.
	raw := intakevocab.Disciplines
	var e error
	var vocab struct{ Disciplines []struct{ ID string } }
	if e = json.Unmarshal(raw, &vocab); e != nil {
		return e
	}
	trades := map[string]bool{}
	for _, d := range vocab.Disciplines {
		trades[d.ID] = strings.HasPrefix(d.ID, "trade.")
	}
	seen, codes := map[string]bool{}, map[string]bool{}
	for n, el := range file.Elements {
		sys, ok := c.System(el.System)
		if !ok || sys.Parent != "" || seen[el.System] || el.Code == "" || codes[el.Code] || !trades[el.Trade] {
			return fmt.Errorf("invalid cost element %s", el.System)
		}
		seen[el.System] = true
		codes[el.Code] = true
		if el.Label == "" {
			file.Elements[n].Label = sys.Label
		}
		subSeen := map[string]bool{}
		for _, o := range el.Overrides {
			sub, ok := c.System(o.Sub)
			if !ok || sub.Parent != el.System || !trades[o.Trade] || subSeen[o.Sub] {
				return fmt.Errorf("invalid cost override %s", o.Sub)
			}
			subSeen[o.Sub] = true
		}
		subSeen = map[string]bool{}
		for _, i := range el.Items {
			sub, ok := c.System(i.Sub)
			if !ok || sub.Parent != el.System || subSeen[i.Sub] || (!i.Costed && i.Why == "") {
				return fmt.Errorf("invalid cost sub-item %s", i.Sub)
			}
			subSeen[i.Sub] = true
		}
	}
	for _, sys := range c.TopSystems() {
		if !seen[sys.ID] {
			return fmt.Errorf("missing cost element %s", sys.ID)
		}
	}
	c.costElements = file.Elements
	path = filepath.Join(root, "costs", "starter_lines.yaml")
	var starters struct {
		Version int
		Status  string
		Lines   []CostStarter
		Sources []map[string]string
	}
	if e = c.unmarshal(path, &starters); e != nil {
		return e
	}
	if starters.Version != 1 || (starters.Status != "draft" && starters.Status != "reviewed") || len(starters.Sources) == 0 {
		return fmt.Errorf("invalid cost starters metadata")
	}
	seen = map[string]bool{}
	for _, s := range starters.Lines {
		if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(s.ID) || seen[s.ID] || strings.TrimSpace(s.Label) == "" {
			return fmt.Errorf("invalid cost starter %s", s.ID)
		}
		seen[s.ID] = true
		switch s.Category {
		case "contingency", "escalation", "authority_fees", "exclusions", "other":
		default:
			return fmt.Errorf("invalid cost starter category %s", s.Category)
		}
		if e = c.validateCostStarter(s.AppliesWhen); e != nil {
			return e
		}

	}
	c.costStarters = starters.Lines
	return nil
}

// Starter rules deliberately support only unconditional labels and exact
// profile choices. They never infer a value from an unknown profile row.
func (c *Catalog) validateCostStarter(p any) error {
	if _, ok := p.(bool); ok {
		return nil
	}
	m, ok := asMap(p)
	if !ok || len(m) != 2 {
		return fmt.Errorf("invalid starter predicate")
	}
	field, ok := m["field"].(string)
	if !ok {
		return fmt.Errorf("starter predicate needs field")
	}
	known := false
	for _, cond := range c.profile.taxonomy.Conditions {
		known = known || "hdr.cond."+cond.Key == field
	}
	if !known {
		return fmt.Errorf("unknown starter field %s", field)
	}
	list := asList(m["in"])
	if len(list) == 0 {
		return fmt.Errorf("starter predicate needs choices")
	}
	for _, v := range list {
		if _, ok := v.(string); !ok {
			return fmt.Errorf("starter choices must be strings")
		}
	}
	return nil
}
func (c *Catalog) CostStarterHolds(p any, values map[string]string) Truth {
	if b, ok := p.(bool); ok {
		if b {
			return True
		}
		return False
	}
	m, ok := asMap(p)
	if !ok {
		return Unknown
	}
	value := values[fmt.Sprint(m["field"])]
	if value == "" {
		return Unknown
	}
	for _, v := range asList(m["in"]) {
		if value == fmt.Sprint(v) {
			return True
		}
	}
	return False
}

func (c *Catalog) CostStarterFields() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range c.costStarters {
		if m, ok := asMap(s.AppliesWhen); ok {
			if key, ok := m["field"].(string); ok && !seen[key] {
				out = append(out, key)
				seen[key] = true
			}
		}
	}
	return out
}
