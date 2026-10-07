package knowledge

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"regexp"
	"slices"
	"strings"
)

type ReportClauseRef struct {
	ID      string `yaml:"id" json:"id"`
	Version int    `yaml:"version" json:"version"`
}
type ReportSection struct {
	ID        string            `yaml:"id" json:"id"`
	Title     string            `yaml:"title" json:"title"`
	Essential bool              `yaml:"essential" json:"essential"`
	Clauses   []ReportClauseRef `yaml:"clauses" json:"clauses"`
}
type ReportTemplate struct {
	ID       string              `yaml:"id" json:"id"`
	Version  int                 `yaml:"version" json:"version"`
	Kind     string              `yaml:"kind" json:"kind"`
	Status   string              `yaml:"status" json:"status"`
	Sources  []map[string]string `yaml:"sources" json:"sources"`
	Sections []ReportSection     `yaml:"sections" json:"sections"`
}

func (s *ReportSection) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		ID        string            `yaml:"id"`
		Title     string            `yaml:"title"`
		Essential *bool             `yaml:"essential"`
		Clauses   []ReportClauseRef `yaml:"clauses"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	if raw.Essential == nil {
		return fmt.Errorf("report section %q requires essential boolean", raw.ID)
	}
	*s = ReportSection{ID: raw.ID, Title: raw.Title, Essential: *raw.Essential, Clauses: raw.Clauses}
	return nil
}

func (c *Catalog) ReportTemplate(id string, version int) (ReportTemplate, bool) {
	t, ok := c.reportTemplates[id]
	return t, ok && t.Version == version
}

// CurrentReportTemplate resolves the catalogue's current version for refresh.
func (c *Catalog) CurrentReportTemplate(id string) (ReportTemplate, bool) {
	t, ok := c.reportTemplates[id]
	return t, ok
}

func (c *Catalog) loadReportTemplates(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version   int              `yaml:"version"`
		Templates []ReportTemplate `yaml:"templates"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: unsupported template catalogue version", path)
	}
	c.reportTemplates = map[string]ReportTemplate{}
	idRE, sectionRE := regexp.MustCompile(`^tpl\.[a-z0-9-]+$`), regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, t := range file.Templates {
		if !idRE.MatchString(t.ID) || t.Version < 1 || !slices.Contains([]string{"rfp", "rft", "pmp"}, t.Kind) || !slices.Contains([]string{"draft", "reviewed"}, t.Status) || len(t.Sources) == 0 || len(t.Sections) == 0 {
			return fmt.Errorf("%s: invalid template %q", path, t.ID)
		}
		if _, ok := c.reportTemplates[t.ID]; ok {
			return fmt.Errorf("%s: duplicate template %q", path, t.ID)
		}
		seen := map[string]bool{}
		for _, section := range t.Sections {
			if !sectionRE.MatchString(section.ID) || strings.TrimSpace(section.Title) == "" || seen[section.ID] || section.Clauses == nil {
				return fmt.Errorf("%s: invalid/duplicate section %q", path, section.ID)
			}
			seen[section.ID] = true
			refs := map[string]bool{}
			for _, ref := range section.Clauses {
				clause, ok := c.Clause(ref.ID, ref.Version)
				if !ok || clause.Section != section.ID || !slices.Contains(clause.Outputs, t.Kind) || refs[ref.ID] {
					return fmt.Errorf("%s: incompatible clause %s v%d in %s", path, ref.ID, ref.Version, section.ID)
				}
				refs[ref.ID] = true
			}
		}
		c.reportTemplates[t.ID] = t
	}
	return nil
}
