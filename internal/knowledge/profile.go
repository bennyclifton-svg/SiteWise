package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"sort"
	"strings"
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
	// Systems makes the determinant relevant when one of them is in a
	// project's scope, besides any rule that reads it.
	Systems    []string  `yaml:"systems"`
	Question   *Question `yaml:"question"`
	Status     string    `yaml:"status"`
	ReplacedBy string    `yaml:"replaced_by"`

	triggers []trigger
}

// Triggered reports whether any trigger matches text.
func (d Determinant) Triggered(text string) bool { return d.TriggeredIn(NewText(text)) }

// TriggeredIn is Triggered over text already lowercased once.
func (d Determinant) TriggeredIn(t Text) bool { return anyMatch(d.triggers, t) }

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

	triggers []trigger
}

// Triggered reports whether any trigger matches text.
func (f ScaleField) Triggered(text string) bool { return f.TriggeredIn(NewText(text)) }

// TriggeredIn is Triggered over text already lowercased once.
func (f ScaleField) TriggeredIn(t Text) bool { return anyMatch(f.triggers, t) }

// Text is a passage with its lowercase form, computed once per passage.
type Text struct{ Raw, Lower string }

// NewText lowercases text for trigger prefiltering.
func NewText(s string) Text { return Text{Raw: s, Lower: strings.ToLower(s)} }

// trigger is a compiled pattern plus a literal every match must contain.
// Case-insensitive patterns lose Go's literal prefix scan, so a substring
// check on the lowercase text rules most passages out before the regexp runs.
type trigger struct {
	re      *regexp.Regexp
	literal string
}

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
	// DisplayLabel is shown to people when set. Label stays the wording Jev
	// reads, so a screen name change never changes a recorded question.
	DisplayLabel string `yaml:"display_label"`
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
	scaleFields  []ScaleField
	scope        scopeDefaults
	live         []Determinant
	keyScope     []keyScopeEntry
}

// Preset is a named set of leaf systems the scope picker can apply at once.
type Preset struct {
	ID      string   `yaml:"id"`
	Label   string   `yaml:"label"`
	Systems []string `yaml:"systems"`
}

// scopeDefaults is knowledge/profile/scope_defaults.yaml.
type scopeDefaults struct {
	AlwaysShown    []string `yaml:"always_shown"`
	EmptyWorkTypes []string `yaml:"empty_work_types"`
	Presets        []Preset `yaml:"presets"`
	Classes        []struct {
		Class    string   `yaml:"class"`
		WorkType string   `yaml:"work_type"`
		Systems  []string `yaml:"systems"`
	} `yaml:"classes"`
	Categories []struct {
		Category string   `yaml:"category"`
		Systems  []string `yaml:"systems"`
	} `yaml:"categories"`
}

func (c *Catalog) loadDeterminants(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version      int           `yaml:"version"`
		Determinants []Determinant `yaml:"determinants"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	for i := range file.Determinants {
		if err := compileTriggers(&file.Determinants[i]); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	c.profile.determinants = file.Determinants
	for _, d := range file.Determinants {
		if d.ProfileGroup != "" && !d.Deprecated() {
			c.profile.live = append(c.profile.live, d)
		}
	}
	return nil
}

func (c *Catalog) loadProfile(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var tax struct {
		Taxonomy `yaml:",inline"`
	}
	if err := c.unmarshal(filepath.Join(dir, "taxonomy.yaml"), &tax); err != nil {
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
	c.profile.scaleFields = tax.Taxonomy.ScaleFields()

	var facts struct {
		Facts []Determinant `yaml:"facts"`
	}
	if err := c.unmarshal(filepath.Join(dir, "project_facts.yaml"), &facts); err != nil {
		return err
	}
	for i := range facts.Facts {
		if err := compileTriggers(&facts.Facts[i]); err != nil {
			return err
		}
	}
	c.profile.facts = facts.Facts

	var scope scopeDefaults
	if err := c.unmarshal(filepath.Join(dir, "scope_defaults.yaml"), &scope); err != nil {
		return err
	}
	check := func(where string, ids []string) error {
		for _, id := range ids {
			if _, ok := c.systems[id]; !ok {
				return fmt.Errorf("scope defaults %s: unknown system %s", where, id)
			}
		}
		return nil
	}
	for _, p := range scope.Presets {
		if err := check("preset "+p.ID, p.Systems); err != nil {
			return err
		}
	}
	for _, e := range scope.Classes {
		if err := check(e.Class+"/"+e.WorkType, e.Systems); err != nil {
			return err
		}
	}
	for _, e := range scope.Categories {
		if err := check(e.Category, e.Systems); err != nil {
			return err
		}
	}
	c.profile.scope = scope
	if err := c.loadPlanningKeys(dir); err != nil {
		return err
	}
	return c.loadKeyScope(dir)
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
func compile(patterns []string) ([]trigger, error) {
	out := make([]trigger, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("trigger %q: %w", p, err)
		}
		out = append(out, trigger{re: re, literal: requiredLiteral(p)})
	}
	return out, nil
}

// requiredLiteral returns the longest literal of three or more runes that
// every match of a top-level concatenation must contain, lowercased; "" when
// there is none (for example a top-level alternation).
func requiredLiteral(pattern string) string {
	re, err := syntax.Parse(pattern, syntax.Perl|syntax.FoldCase)
	if err != nil {
		return ""
	}
	re = re.Simplify()
	for re.Op == syntax.OpCapture {
		re = re.Sub[0]
	}
	best := ""
	consider := func(r *syntax.Regexp) {
		if r.Op == syntax.OpLiteral && len(r.Rune) >= 3 && len(r.Rune) > len([]rune(best)) {
			best = strings.ToLower(string(r.Rune))
		}
	}
	if re.Op == syntax.OpConcat {
		for _, sub := range re.Sub {
			consider(sub)
		}
	} else {
		consider(re)
	}
	return best
}

func anyMatch(ts []trigger, t Text) bool {
	for _, tr := range ts {
		if tr.literal != "" && !strings.Contains(t.Lower, tr.literal) {
			continue
		}
		if tr.re.MatchString(t.Raw) {
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
func (c *Catalog) ProfileDeterminants() []Determinant { return c.profile.live }

// ScaleFields returns the distinct scale fields across subclasses, computed once at load.
func (c *Catalog) ScaleFields() []ScaleField { return c.profile.scaleFields }

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

// ScopeDefaults returns the leaf systems a project of this building category,
// building class (taxonomy subclass) and work type starts with, or nil.
// Refurbishment, remediation and advisory start empty. New build and
// extension use the class's list for the work type, else its new-build list,
// else its category's list; the category is taken from the class when not
// given. Defaults are suggestions, never evidence.
func (c *Catalog) ScopeDefaults(category, class, workType string) []string {
	if workType == "" {
		return nil
	}
	for _, w := range c.profile.scope.EmptyWorkTypes {
		if w == workType {
			return nil
		}
	}
	for _, want := range []string{workType, "new"} {
		for _, e := range c.profile.scope.Classes {
			if class != "" && e.Class == class && e.WorkType == want {
				return e.Systems
			}
		}
	}
	if class != "" {
		for _, bc := range c.profile.taxonomy.BuildingClasses {
			for _, s := range bc.Subclasses {
				if s.ID == class {
					category = bc.ID
				}
			}
		}
	}
	for _, e := range c.profile.scope.Categories {
		if category != "" && e.Category == category {
			return e.Systems
		}
	}
	return nil
}

// Presets are the scope picker's one-click system sets.
func (c *Catalog) Presets() []Preset { return c.profile.scope.Presets }

// Preset returns one preset by id.
func (c *Catalog) Preset(id string) (Preset, bool) {
	for _, p := range c.profile.scope.Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// AlwaysShown are the profile determinants every scope shows.
func (c *Catalog) AlwaysShown() []string { return c.profile.scope.AlwaysShown }

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
