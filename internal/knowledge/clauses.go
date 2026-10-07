package knowledge

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Clause is approved reusable text, referenced by ID and version. WP-20 only
// validates references; authoring report catalogues belongs to WP-40.
type Clause struct {
	ID      string              `yaml:"id"`
	Version int                 `yaml:"version"`
	Text    string              `yaml:"text"`
	Status  string              `yaml:"status"`
	Outputs []string            `yaml:"outputs"`
	Section string              `yaml:"section"`
	Sources []map[string]string `yaml:"sources"`
}

func (c *Catalog) Clause(id string, version int) (Clause, bool) {
	clause, ok := c.clauses[id]
	return clause, ok && clause.Version == version
}
func (c *Catalog) Clauses() []Clause {
	out := []Clause{}
	for _, clause := range c.clauses {
		out = append(out, clause)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (c *Catalog) ApprovedClause(id string, version int) bool {
	clause, ok := c.clauses[id]
	return ok && clause.Version == version && clause.Status == "reviewed"
}

func (c *Catalog) loadClauses(root string) error {
	path := filepath.Join(root, "reports", "clauses.yaml")
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version int      `yaml:"version"`
		Clauses []Clause `yaml:"clauses"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: unsupported version", path)
	}
	c.clauses = map[string]Clause{}
	idRE := regexp.MustCompile(`^cl\.[a-z0-9-]+$`)
	for _, clause := range file.Clauses {
		if !idRE.MatchString(clause.ID) || clause.Version < 1 || strings.TrimSpace(clause.Text) == "" || strings.TrimSpace(clause.Section) == "" || len(clause.Outputs) == 0 || len(clause.Sources) == 0 || (clause.Status != "draft" && clause.Status != "reviewed") {
			return fmt.Errorf("%s: invalid clause %s", path, clause.ID)
		}
		for _, output := range clause.Outputs {
			if !slices.Contains([]string{"rfp", "rft", "pmp"}, output) {
				return fmt.Errorf("%s: invalid clause output %s", path, output)
			}
		}
		if _, exists := c.clauses[clause.ID]; exists {
			return fmt.Errorf("%s: duplicate clause %s", path, clause.ID)
		}
		c.clauses[clause.ID] = clause
	}
	return nil
}
