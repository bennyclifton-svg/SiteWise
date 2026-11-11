package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spaces are the Accommodation schedule's lines: the rooms and areas a
// building provides after the works, one catalogue for every building type
// (knowledge/profile/spaces.yaml, docs/plans/2026-11-11-accommodation-schedule-lines.md).

// SpacePrefix starts every accommodation key: space.<id>.<attribute>.
const SpacePrefix = "space."

// SpaceAttributes are the values read for one space type. provided is yes
// or no; count and capacity are whole numbers; area_sqm is square metres
// for the space (or its group); area_each_sqm is per room; level is text
// such as "Level 3"; mix lists rooms by size ("4x8,2x16": four for 8
// people, two for 16). need.<requirement> and next_to.<space> are yes or no.
var SpaceAttributes = []string{"provided", "count", "area_sqm", "area_each_sqm", "capacity", "level", "mix"}

// Space is one space type.
type Space struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
	// Line names one line of this space; several rooms are numbered.
	Line string `yaml:"line" json:"line"`
	// Kind is room (one line each) or setting (one line, its count in the
	// comment).
	Kind string `yaml:"kind" json:"kind"`
	// Unit is the plural a comment uses: "42 workstations".
	Unit    string `yaml:"unit" json:"unit"`
	Pattern string `yaml:"pattern" json:"-"`
	// CountFrom are the card's counts that give the number of lines.
	CountFrom []string `yaml:"count_from" json:"count_from,omitempty"`

	re *regexp.Regexp
}

// Read reports whether documents are read for this space; one without a
// pattern comes only from a typical list or a card count.
func (s Space) Read() bool { return s.re != nil }

// FindAll returns the [start, end) of every match of the space's name.
func (s Space) FindAll(text string) [][]int {
	if s.re == nil {
		return nil
	}
	return s.re.FindAllStringIndex(text, -1)
}

// SpaceRequirement is a phrase a line's comment may carry ("AV-ready").
type SpaceRequirement struct {
	ID      string `yaml:"id"`
	Label   string `yaml:"label"`
	Pattern string `yaml:"pattern"`

	re *regexp.Regexp
}

// FindAll returns the [start, end) of every match of the requirement's words.
func (r SpaceRequirement) FindAll(text string) [][]int { return r.re.FindAllStringIndex(text, -1) }

// TypicalSpace is one placeholder line of a building type's list. Line,
// when set, names the line for that building type ("Staff room").
type TypicalSpace struct {
	Space string `yaml:"space"`
	Line  string `yaml:"line"`
}

// UnmarshalYAML takes a bare space id or {space, line}.
func (t *TypicalSpace) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		t.Space = n.Value
		return nil
	}
	type plain TypicalSpace
	return n.Decode((*plain)(t))
}

type spaceSchedule struct {
	ID         string         `yaml:"id"`
	Subclasses []string       `yaml:"subclasses"`
	Classes    []string       `yaml:"classes"`
	Basis      string         `yaml:"basis"`
	Source     string         `yaml:"source"`
	Spaces     []TypicalSpace `yaml:"spaces"`
	NewBuild   []TypicalSpace `yaml:"new_build"`
}

// Spaces lists the catalogue in table order.
func (c *Catalog) Spaces() []Space { return c.profile.spaces }

// Space returns one space type by id.
func (c *Catalog) Space(id string) (Space, bool) {
	for _, s := range c.profile.spaces {
		if s.ID == id {
			return s, true
		}
	}
	return Space{}, false
}

// SpaceRequirements lists the comment phrases.
func (c *Catalog) SpaceRequirements() []SpaceRequirement { return c.profile.spaceNeeds }

// SpaceRequirement returns one comment phrase by id.
func (c *Catalog) SpaceRequirement(id string) (SpaceRequirement, bool) {
	for _, r := range c.profile.spaceNeeds {
		if r.ID == id {
			return r, true
		}
	}
	return SpaceRequirement{}, false
}

// SpaceKey splits "space.<id>.<attribute>" into its parts; the attribute of
// a requirement or adjacency keeps its second part ("need.av_ready",
// "next_to.dining"). ok is false for any other key, an unknown space,
// requirement or attribute, and a space next to itself.
func (c *Catalog) SpaceKey(key string) (id, attribute string, ok bool) {
	rest, found := strings.CutPrefix(key, SpacePrefix)
	if !found {
		return "", "", false
	}
	parts := strings.Split(rest, ".")
	if _, known := c.Space(parts[0]); !known {
		return "", "", false
	}
	switch {
	case len(parts) == 2 && slices.Contains(SpaceAttributes, parts[1]):
		return parts[0], parts[1], true
	case len(parts) == 3 && parts[1] == "need":
		_, known := c.SpaceRequirement(parts[2])
		return parts[0], "need." + parts[2], known
	case len(parts) == 3 && parts[1] == "next_to" && parts[2] != parts[0]:
		_, known := c.Space(parts[2])
		return parts[0], "next_to." + parts[2], known
	}
	return "", "", false
}

// TypicalSpaces is the placeholder list for a building: the lists naming
// its type, else those naming its class, merged in order with each space
// once. A new building also gets each list's base-building spaces.
func (c *Catalog) TypicalSpaces(subclass, class, work string) []TypicalSpace {
	var lists []spaceSchedule
	for _, s := range c.profile.schedules {
		if subclass != "" && slices.Contains(s.Subclasses, subclass) {
			lists = append(lists, s)
		}
	}
	if len(lists) == 0 {
		for _, s := range c.profile.schedules {
			if class != "" && slices.Contains(s.Classes, class) {
				lists = append(lists, s)
			}
		}
	}
	var out []TypicalSpace
	seen := map[string]bool{}
	add := func(ts []TypicalSpace) {
		for _, t := range ts {
			if !seen[t.Space] {
				seen[t.Space] = true
				out = append(out, t)
			}
		}
	}
	for _, s := range lists {
		add(s.Spaces)
	}
	if work == "new" {
		for _, s := range lists {
			add(s.NewBuild)
		}
	}
	return out
}

func (c *Catalog) loadSpaces(dir string) error {
	p := filepath.Join(dir, "spaces.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version      int                `yaml:"version"`
		Spaces       []Space            `yaml:"spaces"`
		Requirements []SpaceRequirement `yaml:"requirements"`
		Schedules    []spaceSchedule    `yaml:"schedules"`
	}
	if err := c.unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 2 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	scale := map[string]bool{}
	for _, f := range c.profile.scaleFields {
		scale["hdr.scale."+f.Key] = true
	}
	seen := map[string]bool{}
	for i := range file.Spaces {
		s := &file.Spaces[i]
		if !planningKeyPattern.MatchString(s.ID) || seen[s.ID] || s.Label == "" || s.Line == "" || s.Unit == "" ||
			(s.Kind != "room" && s.Kind != "setting") || (s.Pattern == "" && len(s.CountFrom) == 0) {
			return fmt.Errorf("%s: space %q: a once-only id of lower-case words, a label, a line, a unit, a kind (room or setting) and a pattern or a card count", p, s.ID)
		}
		seen[s.ID] = true
		for _, k := range s.CountFrom {
			if len(scale) > 0 && !scale[k] {
				return fmt.Errorf("%s: space %s: count_from %q is not a card count", p, s.ID, k)
			}
		}
		if s.Pattern == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + s.Pattern)
		if err != nil {
			return fmt.Errorf("%s: space %s: %w", p, s.ID, err)
		}
		s.re = re
	}
	needs := map[string]bool{}
	for i := range file.Requirements {
		r := &file.Requirements[i]
		if !planningKeyPattern.MatchString(r.ID) || needs[r.ID] || r.Label == "" || r.Pattern == "" {
			return fmt.Errorf("%s: requirement %q: a once-only id, a label and a pattern", p, r.ID)
		}
		needs[r.ID] = true
		re, err := regexp.Compile("(?i)" + r.Pattern)
		if err != nil {
			return fmt.Errorf("%s: requirement %s: %w", p, r.ID, err)
		}
		r.re = re
	}
	types, classes := map[string]bool{}, map[string]bool{}
	for _, bc := range c.profile.taxonomy.BuildingClasses {
		classes[bc.ID] = true
		for _, sc := range bc.Subclasses {
			types[sc.ID] = true
		}
	}
	for _, s := range file.Schedules {
		if s.ID == "" || len(s.Spaces) == 0 || (s.Basis != "guide" && s.Basis != "practice") || (s.Basis == "guide" && s.Source == "") {
			return fmt.Errorf("%s: schedule %q: an id, spaces, and a basis (guide with its source, or practice)", p, s.ID)
		}
		for _, t := range append(slices.Clone(s.Spaces), s.NewBuild...) {
			if !seen[t.Space] {
				return fmt.Errorf("%s: schedule %s: unknown space %q", p, s.ID, t.Space)
			}
		}
		for _, id := range s.Subclasses {
			if len(types) > 0 && !types[id] {
				return fmt.Errorf("%s: schedule %s: unknown building type %q", p, s.ID, id)
			}
		}
		for _, id := range s.Classes {
			if len(classes) > 0 && !classes[id] {
				return fmt.Errorf("%s: schedule %s: unknown building class %q", p, s.ID, id)
			}
		}
	}
	c.profile.spaces = file.Spaces
	c.profile.spaceNeeds = file.Requirements
	c.profile.schedules = file.Schedules
	return nil
}
