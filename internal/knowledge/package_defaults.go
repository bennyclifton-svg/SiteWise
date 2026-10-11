package knowledge

import (
	"encoding/json"
	"fmt"
	"strings"

	intakevocab "sitewise/data/intake"
)

// PackageBaseline suggests consultants for a work type. No BuildingClasses
// means any class, less ExceptBuildingClasses.
type PackageBaseline struct {
	BuildingClasses       []string `yaml:"building_classes"`
	ExceptBuildingClasses []string `yaml:"except_building_classes"`
	WorkTypes             []string `yaml:"work_types"`
	Consultants           []string `yaml:"consultants"`
}

// PackageScopeDefault suggests consultants when a listed system, or one
// below it, is in the project's scope of works.
type PackageScopeDefault struct {
	Systems     []string `yaml:"systems"`
	Consultants []string `yaml:"consultants"`
}
type PackageComplexityDefault struct {
	Field       string   `yaml:"field"`
	Values      []string `yaml:"values"`
	Consultants []string `yaml:"consultants"`
}
type PackageDefaults struct {
	Version             int                        `yaml:"version"`
	Status              string                     `yaml:"status"`
	Baselines           []PackageBaseline          `yaml:"baselines"`
	ScopeAdditions      []PackageScopeDefault      `yaml:"scope_additions"`
	ComplexityAdditions []PackageComplexityDefault `yaml:"complexity_additions"`
}

func (c *Catalog) PackageDefaults() PackageDefaults { return c.packageDefaults }

func (c *Catalog) loadPackageDefaults(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var defaults PackageDefaults
	if err := c.unmarshal(path, &defaults); err != nil {
		return err
	}
	if defaults.Version != 2 || (defaults.Status != "draft" && defaults.Status != "reviewed") {
		return fmt.Errorf("%s: invalid package defaults", path)
	}
	classes, types := map[string]bool{}, map[string]bool{}
	for _, c := range c.profile.taxonomy.BuildingClasses {
		classes[c.ID] = true
	}
	for _, w := range c.profile.taxonomy.WorkTypes {
		types[w.ID] = true
	}
	// A suggestion must name a discipline the Procurement table can make an
	// RFP for, so names are checked against the shipped discipline list.
	var vocab struct{ Disciplines []struct{ ID, Label string } }
	if err := json.Unmarshal(intakevocab.Disciplines, &vocab); err != nil {
		return err
	}
	consultants := map[string]bool{}
	for _, d := range vocab.Disciplines {
		if strings.HasPrefix(d.ID, "consultant.") {
			consultants[d.Label] = true
		}
	}
	namesValid := func(names []string) bool {
		if len(names) == 0 {
			return false
		}
		for _, s := range names {
			if strings.TrimSpace(s) == "" || len(s) > 200 {
				return false
			}
		}
		return true
	}
	disciplinesValid := func(names []string) error {
		if !namesValid(names) {
			return fmt.Errorf("%s: empty consultant list", path)
		}
		for _, n := range names {
			if !consultants[n] {
				return fmt.Errorf("%s: consultant %q is not a discipline label", path, n)
			}
		}
		return nil
	}
	for _, b := range defaults.Baselines {
		if len(b.WorkTypes) == 0 {
			return fmt.Errorf("%s: empty baseline", path)
		}
		if err := disciplinesValid(b.Consultants); err != nil {
			return err
		}
		for _, id := range append(append([]string{}, b.BuildingClasses...), b.ExceptBuildingClasses...) {
			if !classes[id] {
				return fmt.Errorf("%s: unknown building class %q", path, id)
			}
		}
		for _, id := range b.WorkTypes {
			if !types[id] {
				return fmt.Errorf("%s: unknown work type %q", path, id)
			}
		}
	}
	for _, a := range defaults.ScopeAdditions {
		if len(a.Systems) == 0 {
			return fmt.Errorf("%s: scope addition without systems", path)
		}
		for _, id := range a.Systems {
			if _, ok := c.System(id); !ok {
				return fmt.Errorf("%s: unknown system %q", path, id)
			}
		}
		if err := disciplinesValid(a.Consultants); err != nil {
			return err
		}
	}
	for _, a := range defaults.ComplexityAdditions {
		if strings.TrimSpace(a.Field) == "" || !namesValid(a.Values) {
			return fmt.Errorf("%s: invalid complexity addition", path)
		}
		if err := disciplinesValid(a.Consultants); err != nil {
			return err
		}
	}
	c.packageDefaults = defaults
	return nil
}
