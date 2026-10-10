package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Project details (profile/project_details.yaml) are facts about the project
// that the user can state once: the client, the LGA, the zone, the site
// address. Profile is their one home and every RFP reads them from there.
// A detail marked `extraction: pre_parsed` is also read from the documents
// the profile reads: code finds candidate values, Jev picks one, and the
// user's own word always wins. The rest are never asked of Jev.

// DetailPrefix starts the profile row key of a project detail. RFP flags
// with no determinant are stored as proj.flag.<skeleton flag key>.
const DetailPrefix = "proj."

// SitePrefix starts the profile row key of a site detail (address, lot).
// Their typed value lives on the site record; a reading is only shown when
// nothing has been typed.
const SitePrefix = "site."

// ExtractionPreParsed marks a detail whose candidates code finds in a passage
// (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
const ExtractionPreParsed = "pre_parsed"

// ProjectDetail is one registered detail. Value is text or yes_no.
type ProjectDetail struct {
	Key   string `yaml:"key" json:"key"`
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
	Scope string `yaml:"scope" json:"scope"`
	Help  string `yaml:"help" json:"help,omitempty"`
	// Extraction, Triggers and Asks say how documents are read for it. Empty
	// Extraction means the user is its only source.
	Extraction string   `yaml:"extraction" json:"-"`
	Triggers   []string `yaml:"triggers" json:"-"`
	Asks       string   `yaml:"asks" json:"-"`

	triggers []trigger
}

// TriggeredIn reports whether a trigger matches text already lowercased once.
func (d ProjectDetail) TriggeredIn(t Text) bool { return anyMatch(d.triggers, t) }

// FlagReading says how a document is read for one RFP flag that has no
// determinant (proj.flag.<key>): a keyword gate and a yes/no question.
type FlagReading struct {
	Key      string   `yaml:"key"`
	Triggers []string `yaml:"triggers"`
	Asks     string   `yaml:"asks"`

	triggers []trigger
}

// TriggeredIn reports whether a trigger matches text already lowercased once.
func (f FlagReading) TriggeredIn(t Text) bool { return anyMatch(f.triggers, t) }

type detailsData struct {
	list  []ProjectDetail
	byKey map[string]int
	site  []ProjectDetail
	flags []FlagReading
}

// ProjectDetails lists the registry in file order.
func (c *Catalog) ProjectDetails() []ProjectDetail { return c.details.list }

// SiteDetails lists the details stored on the site record (address, lot).
func (c *Catalog) SiteDetails() []ProjectDetail { return c.details.site }

// FlagReadings lists the flags read from documents.
func (c *Catalog) FlagReadings() []FlagReading { return c.details.flags }

// ProjectDetail returns a detail by key, without the "proj." prefix.
func (c *Catalog) ProjectDetail(key string) (ProjectDetail, bool) {
	i, ok := c.details.byKey[key]
	if !ok {
		return ProjectDetail{}, false
	}
	return c.details.list[i], true
}

// SiteDetail returns a site detail by key, without the "site." prefix.
func (c *Catalog) SiteDetail(key string) (ProjectDetail, bool) {
	for _, d := range c.details.site {
		if d.Key == key {
			return d, true
		}
	}
	return ProjectDetail{}, false
}

// FlagReading returns the reading of one flag, by flag key.
func (c *Catalog) FlagReading(key string) (FlagReading, bool) {
	for _, f := range c.details.flags {
		if f.Key == key {
			return f, true
		}
	}
	return FlagReading{}, false
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
		Version     int             `yaml:"version"`
		Details     []ProjectDetail `yaml:"details"`
		SiteDetails []ProjectDetail `yaml:"site_details"`
	}
	if err := c.unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	c.details = detailsData{byKey: map[string]int{}}
	for i := range file.Details {
		d := &file.Details[i]
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
		if err := d.compileReading(p); err != nil {
			return err
		}
		c.details.byKey[d.Key] = i
	}
	c.details.list = file.Details
	seen := map[string]bool{}
	for i := range file.SiteDetails {
		d := &file.SiteDetails[i]
		switch {
		case !planningKeyPattern.MatchString(d.Key) || seen[d.Key]:
			return fmt.Errorf("%s: site detail %q: lower-case words and underscores, once", p, d.Key)
		case d.Label == "" || d.Value != "text":
			return fmt.Errorf("%s: site detail %s needs a label and value text", p, d.Key)
		}
		seen[d.Key] = true
		d.Scope = ScopeSite
		if err := d.compileReading(p); err != nil {
			return err
		}
	}
	c.details.site = file.SiteDetails
	return c.loadFlagReadings(dir)
}

// compileReading checks a detail's reading settings and compiles its triggers.
func (d *ProjectDetail) compileReading(path string) error {
	if d.Extraction == "" {
		if len(d.Triggers) > 0 || d.Asks != "" {
			return fmt.Errorf("%s: detail %s has triggers or asks but no extraction", path, d.Key)
		}
		return nil
	}
	if d.Extraction != ExtractionPreParsed || d.Value != "text" || len(d.Triggers) == 0 || d.Asks == "" {
		return fmt.Errorf("%s: detail %s: a read detail is text, pre_parsed, with triggers and asks", path, d.Key)
	}
	res, err := compile(d.Triggers)
	if err != nil {
		return fmt.Errorf("%s: detail %s: %w", path, d.Key, err)
	}
	d.triggers = res
	return nil
}

func (c *Catalog) loadFlagReadings(dir string) error {
	p := filepath.Join(dir, "flag_readings.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version int           `yaml:"version"`
		Flags   []FlagReading `yaml:"flags"`
	}
	if err := c.unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	seen := map[string]bool{}
	for i := range file.Flags {
		f := &file.Flags[i]
		if !planningKeyPattern.MatchString(f.Key) || seen[f.Key] || len(f.Triggers) == 0 || f.Asks == "" {
			return fmt.Errorf("%s: flag reading %q: a once-only key with triggers and asks", p, f.Key)
		}
		seen[f.Key] = true
		res, err := compile(f.Triggers)
		if err != nil {
			return fmt.Errorf("%s: flag reading %s: %w", p, f.Key, err)
		}
		f.triggers = res
	}
	c.details.flags = file.Flags
	return nil
}
