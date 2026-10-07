package knowledge

import (
	"fmt"
	"strings"
)

type PackageBaseline struct {
	BuildingClasses []string `yaml:"building_classes"`
	WorkTypes       []string `yaml:"work_types"`
	Consultants     []string `yaml:"consultants"`
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
	if defaults.Version != 1 || (defaults.Status != "draft" && defaults.Status != "reviewed") {
		return fmt.Errorf("%s: invalid package defaults", path)
	}
	classes, types := map[string]bool{}, map[string]bool{}
	for _, c := range c.profile.taxonomy.BuildingClasses {
		classes[c.ID] = true
	}
	for _, w := range c.profile.taxonomy.WorkTypes {
		types[w.ID] = true
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
	for _, b := range defaults.Baselines {
		if len(b.BuildingClasses) == 0 || len(b.WorkTypes) == 0 || !namesValid(b.Consultants) {
			return fmt.Errorf("%s: empty baseline", path)
		}
		for _, id := range b.BuildingClasses {
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
	for _, a := range defaults.ComplexityAdditions {
		if strings.TrimSpace(a.Field) == "" || !namesValid(a.Values) || !namesValid(a.Consultants) {
			return fmt.Errorf("%s: invalid complexity addition", path)
		}
	}
	c.packageDefaults = defaults
	return nil
}
