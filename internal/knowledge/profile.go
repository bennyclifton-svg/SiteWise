package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

const statusDeprecated = "deprecated"

// Option is one allowed value of a choice determinant or fact.
type Option struct {
	ID        string `yaml:"id"`
	Describes string `yaml:"describes"`
}

// StatedIn names the kind of document that usually states a value, so the
// profile can say what to upload when nothing states it.
type StatedIn struct {
	Kind       string `yaml:"kind"`
	Discipline string `yaml:"discipline"`
	Label      string `yaml:"label"`
}

// Determinant is a fact that decides which rules apply. Project facts share
// this shape. Triggers gate its question: code asks Jev only when one matches
// (https://docs.typesafe.ai/patterns/intent-routing).
type Determinant struct {
	ID              string     `yaml:"id"`
	Label           string     `yaml:"label"`
	Value           string     `yaml:"value"`
	Unit            string     `yaml:"unit"`
	Extraction      string     `yaml:"extraction"`
	Options         []Option   `yaml:"options"`
	Derived         bool       `yaml:"derived"`
	By              string     `yaml:"by"`
	TriggerPatterns []string   `yaml:"triggers"`
	StatedIn        []StatedIn `yaml:"stated_in"`
	ProfileGroup    string     `yaml:"profile_group"`
	Question        *Question  `yaml:"question"`
	Status          string     `yaml:"status"`
	ReplacedBy      string     `yaml:"replaced_by"`

	triggers []*regexp.Regexp
}

// Triggered reports whether any trigger matches text.
func (d Determinant) Triggered(text string) bool {
	return anyMatch(d.triggers, text)
}

// Deprecated reports whether the record is kept only for its id.
func (d Determinant) Deprecated() bool { return d.Status == statusDeprecated }

// ScaleField is one size measure of a subclass (storeys, GFA, units).
type ScaleField struct {
	Key             string   `yaml:"key"`
	Label           string   `yaml:"label"`
	Type            string   `yaml:"type"`
	Unit            string   `yaml:"unit"`
	BasisRequired   bool     `yaml:"basis_required"`
	TriggerPatterns []string `yaml:"triggers"`

	triggers []*regexp.Regexp
}

// Triggered reports whether any trigger matches text.
func (f ScaleField) Triggered(text string) bool { return anyMatch(f.triggers, text) }

// Subclass is one building type with its NCC class hint and scale fields.
type Subclass struct {
	ID          string       `yaml:"id"`
	Label       string       `yaml:"label"`
	NCCClass    string       `yaml:"ncc_class"`
	ScaleFields []ScaleField `yaml:"scale_fields"`
}

// BuildingClass groups subclasses (residential, industrial and so on).
type BuildingClass struct {
	ID         string     `yaml:"id"`
	Label      string     `yaml:"label"`
	WorkTypes  []string   `yaml:"work_types"`
	Subclasses []Subclass `yaml:"subclasses"`
}

// Choice is an id and its display label.
type Choice struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

// Condition is one project condition from the Clerk header (planning,
// procurement, access and so on).
type Condition struct {
	Key       string   `yaml:"key"`
	Label     string   `yaml:"label"`
	AppliesTo []string `yaml:"applies_to"`
	Options   []Choice `yaml:"options"`
}

// Taxonomy is the project header vocabulary copied from Clerk data.
type Taxonomy struct {
	WorkTypes       []Choice        `yaml:"work_types"`
	BuildingClasses []BuildingClass `yaml:"building_classes"`
	Conditions      []Condition     `yaml:"conditions"`
}

// Subclass finds a subclass by id across classes.
func (t Taxonomy) Subclass(id string) (Subclass, bool) {
	for _, c := range t.BuildingClasses {
		for _, s := range c.Subclasses {
			if s.ID == id {
				return s, true
			}
		}
	}
	return Subclass{}, false
}

// ScaleFields returns every distinct scale field across subclasses, by key.
func (t Taxonomy) ScaleFields() []ScaleField {
	seen := map[string]bool{}
	var out []ScaleField
	for _, c := range t.BuildingClasses {
		for _, s := range c.Subclasses {
			for _, f := range s.ScaleFields {
				if !seen[f.Key] {
					seen[f.Key] = true
					out = append(out, f)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

type profileData struct {
	determinants []Determinant
	facts        []Determinant
	taxonomy     Taxonomy
	typical      map[string][]string
}

func (c *Catalog) loadDeterminants(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version      int           `yaml:"version"`
		Determinants []Determinant `yaml:"determinants"`
	}
	if err := unmarshal(path, &file); err != nil {
		return err
	}
	for i := range file.Determinants {
		if err := compileTriggers(&file.Determinants[i]); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	c.profile.determinants = file.Determinants
	return nil
}

func (c *Catalog) loadProfile(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var tax struct {
		Taxonomy `yaml:",inline"`
	}
	if err := unmarshal(filepath.Join(dir, "taxonomy.yaml"), &tax); err != nil {
		return err
	}
	for ci := range tax.BuildingClasses {
		for si := range tax.BuildingClasses[ci].Subclasses {
			fields := tax.BuildingClasses[ci].Subclasses[si].ScaleFields
			for fi := range fields {
				res, err := compile(fields[fi].TriggerPatterns)
				if err != nil {
					return fmt.Errorf("taxonomy scale field %s: %w", fields[fi].Key, err)
				}
				fields[fi].triggers = res
			}
		}
	}
	c.profile.taxonomy = tax.Taxonomy

	var facts struct {
		Facts []Determinant `yaml:"facts"`
	}
	if err := unmarshal(filepath.Join(dir, "project_facts.yaml"), &facts); err != nil {
		return err
	}
	for i := range facts.Facts {
		if err := compileTriggers(&facts.Facts[i]); err != nil {
			return err
		}
	}
	c.profile.facts = facts.Facts

	var typical struct {
		Typical []struct {
			Subclass string   `yaml:"subclass"`
			WorkType string   `yaml:"work_type"`
			Systems  []string `yaml:"systems"`
		} `yaml:"typical"`
	}
	if err := unmarshal(filepath.Join(dir, "typical_systems.yaml"), &typical); err != nil {
		return err
	}
	c.profile.typical = map[string][]string{}
	for _, t := range typical.Typical {
		for _, id := range t.Systems {
			if _, ok := c.systems[id]; !ok {
				return fmt.Errorf("typical systems %s/%s: unknown system %s", t.Subclass, t.WorkType, id)
			}
		}
		c.profile.typical[t.Subclass+"/"+t.WorkType] = t.Systems
	}
	return nil
}

func compileTriggers(d *Determinant) error {
	res, err := compile(d.TriggerPatterns)
	if err != nil {
		return fmt.Errorf("%s: %w", d.ID, err)
	}
	d.triggers = res
	return nil
}

// compile uses Go's RE2 syntax, case-insensitive, as SCHEMA.md documents.
func compile(patterns []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("trigger %q: %w", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func anyMatch(res []*regexp.Regexp, text string) bool {
	for _, re := range res {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// Determinants returns every determinant in file order.
func (c *Catalog) Determinants() []Determinant { return c.profile.determinants }

// Determinant returns one determinant by id.
func (c *Catalog) Determinant(id string) (Determinant, bool) {
	for _, d := range c.profile.determinants {
		if d.ID == id {
			return d, true
		}
	}
	return Determinant{}, false
}

// ProfileDeterminants returns live determinants shown in the profile.
func (c *Catalog) ProfileDeterminants() []Determinant {
	var out []Determinant
	for _, d := range c.profile.determinants {
		if d.ProfileGroup != "" && !d.Deprecated() {
			out = append(out, d)
		}
	}
	return out
}

// ProjectFacts returns the project facts (consent, contract and so on).
func (c *Catalog) ProjectFacts() []Determinant { return c.profile.facts }

// ProjectFact returns one project fact by id.
func (c *Catalog) ProjectFact(id string) (Determinant, bool) {
	for _, f := range c.profile.facts {
		if f.ID == id {
			return f, true
		}
	}
	return Determinant{}, false
}

// Taxonomy returns the project header vocabulary.
func (c *Catalog) Taxonomy() Taxonomy { return c.profile.taxonomy }

// Typical returns suggested leaf systems for a subclass and work type, or
// nil. Suggestions are never evidence.
func (c *Catalog) Typical(subclass, workType string) []string {
	return c.profile.typical[subclass+"/"+workType]
}

// Resolve maps a deprecated system or determinant id to its replacement, so
// stored facts under an old id show under the new one.
func (c *Catalog) Resolve(id string) string {
	for i := 0; i < 8; i++ {
		if sys, ok := c.systems[id]; ok && sys.Status == statusDeprecated && sys.ReplacedBy != "" {
			id = sys.ReplacedBy
			continue
		}
		if d, ok := c.Determinant(id); ok && d.Deprecated() && d.ReplacedBy != "" {
			id = d.ReplacedBy
			continue
		}
		return id
	}
	return id
}

// Leaves returns live child systems in id order.
func (c *Catalog) Leaves() []System {
	var out []System
	for _, id := range c.systemIDs {
		sys := c.systems[id]
		if sys.Parent != "" && sys.Status != statusDeprecated {
			out = append(out, sys)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
