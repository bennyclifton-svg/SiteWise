package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Project details (profile/project_details.yaml) are facts the user states
// once for the project: the client, the LGA, the zone. No document reading
// fills them and Jev is never asked; Profile is their one home, and every
// RFP reads them from there.

// DetailPrefix starts the profile row key of a project detail. RFP flags
// with no determinant are stored as proj.flag.<skeleton flag key>.
const DetailPrefix = "proj."

// ProjectDetail is one registered detail. Value is text or yes_no.
type ProjectDetail struct {
	Key   string `yaml:"key" json:"key"`
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
	Scope string `yaml:"scope" json:"scope"`
	Help  string `yaml:"help" json:"help,omitempty"`
}

type detailsData struct {
	list  []ProjectDetail
	byKey map[string]int
}

// ProjectDetails lists the registry in file order.
func (c *Catalog) ProjectDetails() []ProjectDetail { return c.details.list }

// ProjectDetail returns a detail by key, without the "proj." prefix.
func (c *Catalog) ProjectDetail(key string) (ProjectDetail, bool) {
	i, ok := c.details.byKey[key]
	if !ok {
		return ProjectDetail{}, false
	}
	return c.details.list[i], true
}

// Check returns "" when v is a valid value of the detail.
func (d ProjectDetail) Check(v string) string {
	if d.Value == "yes_no" && v != "yes" && v != "no" {
		return d.Key + " must be yes or no"
	}
	return ""
}

func (c *Catalog) loadProjectDetails(dir string) error {
	p := filepath.Join(dir, "project_details.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version int             `yaml:"version"`
		Details []ProjectDetail `yaml:"details"`
	}
	if err := c.unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	c.details = detailsData{byKey: map[string]int{}}
	for i, d := range file.Details {
		switch {
		case !planningKeyPattern.MatchString(d.Key) || d.Key == "flag":
			return fmt.Errorf("%s: project detail %q: lower-case words and underscores", p, d.Key)
		case d.Label == "":
			return fmt.Errorf("%s: project detail %s needs a label", p, d.Key)
		case !slices.Contains([]string{"text", "yes_no"}, d.Value):
			return fmt.Errorf("%s: project detail %s: value %q", p, d.Key, d.Value)
		case d.Scope != ScopeSite && d.Scope != ScopeProject:
			return fmt.Errorf("%s: project detail %s: scope %q", p, d.Key, d.Scope)
		}
		if _, dup := c.details.byKey[d.Key]; dup {
			return fmt.Errorf("%s: duplicate project detail %s", p, d.Key)
		}
		c.details.byKey[d.Key] = i
	}
	c.details.list = file.Details
	return nil
}
