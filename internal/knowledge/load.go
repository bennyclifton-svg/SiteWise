// Package knowledge loads the building model and evaluates it in code.
// Jev questions are data here; this package does not call Jev.
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// System is one node in the building hierarchy. A top-level system has no parent.
type System struct {
	ID        string `yaml:"id"`
	Parent    string `yaml:"parent"`
	Label     string `yaml:"label"`
	Describes string `yaml:"describes"`
	Excludes  string `yaml:"excludes"`
	Status    string `yaml:"status"`
	// ReplacedBy names the live system a deprecated one maps to.
	ReplacedBy string `yaml:"replaced_by"`
}

// Question is one Jev question stored in the knowledge files.
// Empty choice criteria are templates, not requests: code fills those later.
type Question struct {
	ID           string
	Type         string   `yaml:"type"`
	Instructions string   `yaml:"instructions"`
	Criteria     any      `yaml:"criteria"`
	RunsOn       []string `yaml:"runs_on"`
}

// Interface is a canonical relationship between systems. Edges are computed
// from From and To; they are not stored as a second copy of this record.
type Interface struct {
	ID           string     `yaml:"id"`
	Type         string     `yaml:"type"`
	From         []string   `yaml:"from"`
	To           []string   `yaml:"to"`
	Summary      string     `yaml:"summary"`
	Status       string     `yaml:"status"`
	AppliesWhen  any        `yaml:"applies_when,omitempty"`
	ResolvedWhen []Question `yaml:"resolved_when"`
}

// Rule is a code-evaluated requirement. Derives is nil when the rule does
// not look up a table.
type Rule struct {
	ID       string    `yaml:"id"`
	Status   string    `yaml:"status"`
	Derives  *Derives  `yaml:"derives"`
	Evidence *Question `yaml:"evidence"`
	// Systems the rule is about, and the predicate over determinants and
	// present systems that limits it (SCHEMA.md, Predicates). The scoped
	// profile reads both to decide which determinants are relevant.
	Systems     []string `yaml:"systems"`
	AppliesWhen any      `yaml:"applies_when"`
}

// Derives names a table lookup. Pending derivations stay unknown: the seed
// numbers are not a table, and a default must not stand in for one.
type Derives struct {
	Table   string   `yaml:"table"`
	Inputs  []string `yaml:"inputs"`
	Gives   string   `yaml:"gives"`
	Pending bool     `yaml:"pending"`
	Reason  string   `yaml:"reason"`
}

// Table is a verified transcription of an instrument table. Status stays
// draft until the owner reviews it, even when Verified is true.
type Table struct {
	ID       string           `yaml:"id"`
	Status   string           `yaml:"status"`
	Verified bool             `yaml:"verified"`
	Inputs   []string         `yaml:"inputs"`
	Outputs  []string         `yaml:"outputs"`
	Rows     []map[string]any `yaml:"rows"`
}

// Catalog is the building hierarchy, its interfaces and the tables code may
// apply. Draft records stay in the catalog; evaluation refuses to treat them
// as a determination.
type Catalog struct {
	root            string
	loaded          map[string][]byte
	version         string
	systems         map[string]System
	systemIDs       []string
	Interfaces      []Interface
	rules           map[string]Rule
	tables          map[string]Table
	evidence        []Question
	profile         profileData
	planning        planningData
	works           worksData
	clauses         map[string]Clause
	stages          []DeliveryStage
	packageDefaults PackageDefaults
	reportTemplates map[string]ReportTemplate
}

// Load reads knowledge/ (or a fixture with the same layout).
// gopkg.in/yaml.v3 is the parser: the files use anchors, folded scalars and
// nested maps, which is more than a line-oriented reader can take on safely.
func Load(root string) (*Catalog, error) {
	c := &Catalog{
		root: root, loaded: map[string][]byte{},
		systems: map[string]System{},
		rules:   map[string]Rule{},
		tables:  map[string]Table{},
	}
	if err := c.loadSystems(filepath.Join(root, "systems.yaml")); err != nil {
		return nil, err
	}
	clusters := filepath.Join(root, "clusters")
	entries, err := os.ReadDir(clusters)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(clusters, entry.Name())
		if err := c.loadOptionalSystems(filepath.Join(dir, "systems.yaml")); err != nil {
			return nil, err
		}
	}
	if err := c.checkSystems(); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(clusters, entry.Name())
		if err := c.loadRules(filepath.Join(dir, "rules.yaml")); err != nil {
			return nil, err
		}
		if err := c.loadInterfaces(filepath.Join(dir, "interfaces.yaml")); err != nil {
			return nil, err
		}
		if err := c.loadFailures(filepath.Join(dir, "failure_modes.yaml")); err != nil {
			return nil, err
		}
	}
	if err := c.loadTables(filepath.Join(root, "tables")); err != nil {
		return nil, err
	}
	if err := c.loadDeterminants(filepath.Join(root, "determinants.yaml")); err != nil {
		return nil, err
	}
	if err := c.loadProfile(filepath.Join(root, "profile")); err != nil {
		return nil, err
	}
	var clusterDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			clusterDirs = append(clusterDirs, filepath.Join(clusters, entry.Name()))
		}
	}
	if err := c.loadWorks(root, clusterDirs); err != nil {
		return nil, err
	}
	if err := c.loadClauses(root); err != nil {
		return nil, err
	}
	if err := c.loadDeliveryStages(filepath.Join(root, "works", "stages.yaml")); err != nil {
		return nil, err
	}
	if err := c.loadPackageDefaults(filepath.Join(root, "works", "package_defaults.yaml")); err != nil {
		return nil, err
	}
	if err := c.loadReportTemplates(filepath.Join(root, "reports", "templates.yaml")); err != nil {
		return nil, err
	}
	sort.Slice(c.evidence, func(i, j int) bool { return c.evidence[i].ID < c.evidence[j].ID })
	c.version = c.hashLoaded()
	c.loaded = nil
	return c, nil
}

func (c *Catalog) loadSystems(path string) error {
	var file struct {
		Version int      `yaml:"version"`
		Systems []System `yaml:"systems"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", path, file.Version)
	}
	for _, sys := range file.Systems {
		if err := c.addSystem(path, sys); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) loadOptionalSystems(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return c.loadSystems(path)
}

func (c *Catalog) addSystem(path string, sys System) error {
	sys.ID = strings.TrimSpace(sys.ID)
	if sys.ID == "" {
		return fmt.Errorf("%s: system id is required", path)
	}
	if _, ok := c.systems[sys.ID]; ok {
		return fmt.Errorf("%s: duplicate system %s", path, sys.ID)
	}
	c.systems[sys.ID] = sys
	c.systemIDs = append(c.systemIDs, sys.ID)
	return nil
}

func (c *Catalog) checkSystems() error {
	for _, id := range c.systemIDs {
		sys := c.systems[id]
		if sys.Parent == "" {
			continue
		}
		if sys.Parent == id {
			return fmt.Errorf("system %s is its own parent", id)
		}
		if _, ok := c.systems[sys.Parent]; !ok {
			return fmt.Errorf("system %s parent %s is not loaded", id, sys.Parent)
		}
		seen := map[string]struct{}{id: {}}
		cur := sys.Parent
		for cur != "" {
			if _, ok := seen[cur]; ok {
				return fmt.Errorf("system %s is in a cycle", id)
			}
			seen[cur] = struct{}{}
			parent, ok := c.systems[cur]
			if !ok {
				break
			}
			cur = parent.Parent
		}
	}
	return nil
}

func (c *Catalog) loadRules(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	var file struct {
		Version int    `yaml:"version"`
		Rules   []Rule `yaml:"rules"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", path, file.Version)
	}
	for _, rule := range file.Rules {
		if rule.ID == "" {
			return fmt.Errorf("%s: rule id is required", path)
		}
		if _, ok := c.rules[rule.ID]; ok {
			return fmt.Errorf("%s: duplicate rule %s", path, rule.ID)
		}
		c.rules[rule.ID] = rule
		if rule.Evidence != nil && rule.Evidence.Type == "noul" {
			q := *rule.Evidence
			q.ID = "rule:" + rule.ID
			c.addEvidence(q)
		}
	}
	return nil
}

func (c *Catalog) loadInterfaces(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	var file struct {
		Version    int         `yaml:"version"`
		Interfaces []Interface `yaml:"interfaces"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", path, file.Version)
	}
	seen := map[string]struct{}{}
	for _, existing := range c.Interfaces {
		seen[existing.ID] = struct{}{}
	}
	for _, iface := range file.Interfaces {
		if iface.ID == "" {
			return fmt.Errorf("%s: interface id is required", path)
		}
		if _, ok := seen[iface.ID]; ok {
			return fmt.Errorf("%s: duplicate interface %s", path, iface.ID)
		}
		seen[iface.ID] = struct{}{}
		for i := range iface.ResolvedWhen {
			q := iface.ResolvedWhen[i]
			q.ID = fmt.Sprintf("if:%s#%d", iface.ID, i)
			iface.ResolvedWhen[i] = q
			if q.Type == "noul" {
				c.addEvidence(q)
			}
		}
		c.Interfaces = append(c.Interfaces, iface)
	}
	sort.Slice(c.Interfaces, func(i, j int) bool { return c.Interfaces[i].ID < c.Interfaces[j].ID })
	return nil
}

func (c *Catalog) loadFailures(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	var file struct {
		Version int `yaml:"version"`
		Modes   []struct {
			ID       string   `yaml:"id"`
			Detector Question `yaml:"detector"`
		} `yaml:"failure_modes"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", path, file.Version)
	}
	for _, mode := range file.Modes {
		if mode.Detector.Type != "noul" || mode.ID == "" {
			continue
		}
		q := mode.Detector
		q.ID = "fm:" + mode.ID
		c.addEvidence(q)
	}
	return nil
}

func (c *Catalog) addEvidence(q Question) {
	if strings.TrimSpace(q.Instructions) == "" || len(q.RunsOn) == 0 {
		return
	}
	c.evidence = append(c.evidence, q)
}

func (c *Catalog) loadTables(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		var table Table
		raw := struct {
			Version int `yaml:"version"`
			Table   `yaml:",inline"`
		}{}
		if err := c.unmarshal(path, &raw); err != nil {
			return err
		}
		if raw.Version != 1 {
			return fmt.Errorf("%s: version %d", path, raw.Version)
		}
		table = raw.Table
		if table.ID == "" {
			return fmt.Errorf("%s: table id is required", path)
		}
		if table.ID+".yaml" != entry.Name() {
			return fmt.Errorf("%s: id %s does not match the filename", path, table.ID)
		}
		if _, ok := c.tables[table.ID]; ok {
			return fmt.Errorf("%s: duplicate table %s", path, table.ID)
		}
		c.tables[table.ID] = table
	}
	return nil
}

func (c *Catalog) unmarshal(path string, dest any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := c.remember(path, body); err != nil {
		return err
	}
	if err := yaml.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// System returns one system by id.
func (c *Catalog) System(id string) (System, bool) {
	sys, ok := c.systems[id]
	return sys, ok
}

// TopSystems returns systems that have no parent, ordered by id.
func (c *Catalog) TopSystems() []System {
	var out []System
	for _, id := range c.systemIDs {
		sys := c.systems[id]
		if sys.Parent == "" {
			out = append(out, sys)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Children returns the live direct children of parent, ordered by id.
// Deprecated systems are never offered to Jev.
func (c *Catalog) Children(parent string) []System {
	var out []System
	for _, id := range c.systemIDs {
		sys := c.systems[id]
		if sys.Parent == parent && sys.Status != statusDeprecated {
			out = append(out, sys)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Rule returns one rule by id.
func (c *Catalog) Rule(id string) (Rule, bool) {
	rule, ok := c.rules[id]
	return rule, ok
}

// Table returns one lookup table by id.
func (c *Catalog) Table(id string) (Table, bool) {
	table, ok := c.tables[id]
	return table, ok
}

// EvidenceQuestions returns the noul questions whose runs_on matches labels.
// A label matches a run target when it is that system or a descendant of it.
// One passage's questions are one fan-out; this does not split them.
func (c *Catalog) EvidenceQuestions(labels []string) []Question {
	var out []Question
	for _, q := range c.evidence {
		if c.runsOn(labels, q.RunsOn) {
			out = append(out, q)
		}
	}
	return out
}

func (c *Catalog) runsOn(labels, targets []string) bool {
	for _, label := range labels {
		for _, target := range targets {
			if c.covers(label, target) {
				return true
			}
		}
	}
	return false
}

// covers reports whether systemID is endpoint or a descendant of endpoint.
func (c *Catalog) covers(systemID, endpoint string) bool {
	seen := map[string]struct{}{}
	cur := systemID
	for cur != "" {
		if cur == endpoint {
			return true
		}
		if _, ok := seen[cur]; ok {
			return false
		}
		seen[cur] = struct{}{}
		sys, ok := c.systems[cur]
		if !ok {
			return false
		}
		cur = sys.Parent
	}
	return false
}
