package knowledge

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
)

// Key scope (SCHEMA.md "Key scope", owner decision D-04): whether a profile
// key is stored against the site, which outlasts projects, or the project.
// The first matching entry of profile/key_scope.yaml wins.

const (
	ScopeSite    = "site"
	ScopeProject = "project"
)

type keyScopeEntry struct {
	Match string `yaml:"match"`
	Scope string `yaml:"scope"`
}

// KeyScope returns "site" or "project" for a profile key, or "" when the
// registry is absent or no entry matches.
func (c *Catalog) KeyScope(key string) string {
	for _, e := range c.profile.keyScope {
		if ok, _ := path.Match(e.Match, key); ok {
			return e.Scope
		}
	}
	return ""
}

// loadKeyScope reads the registry and refuses a catalogue key it does not
// classify, so no value is stored against the wrong owner. A missing file
// leaves every key unclassified (fixtures and older layouts).
func (c *Catalog) loadKeyScope(dir string) error {
	p := filepath.Join(dir, "key_scope.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version  int             `yaml:"version"`
		Families []keyScopeEntry `yaml:"families"`
	}
	if err := unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	for _, e := range file.Families {
		if e.Scope != ScopeSite && e.Scope != ScopeProject {
			return fmt.Errorf("%s: %s has scope %q", p, e.Match, e.Scope)
		}
		if _, err := path.Match(e.Match, ""); err != nil {
			return fmt.Errorf("%s: bad pattern %q: %w", p, e.Match, err)
		}
	}
	c.profile.keyScope = file.Families
	var missing []string
	for _, k := range c.profileKeys() {
		if c.KeyScope(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s: no scope for %d keys, e.g. %v", p, len(missing), missing[:min(len(missing), 5)])
	}
	return nil
}

// profileKeys lists every key the profile can store: header, scale,
// condition, determinant, fact, system and scope keys.
func (c *Catalog) profileKeys() []string {
	keys := []string{"hdr.building_class", "hdr.subclass", "hdr.work_type"}
	for _, f := range c.profile.scaleFields {
		keys = append(keys, "hdr.scale."+f.Key)
	}
	for _, cond := range c.profile.taxonomy.Conditions {
		keys = append(keys, "hdr.cond."+cond.Key)
	}
	for _, d := range c.profile.determinants {
		keys = append(keys, "det."+d.ID)
	}
	for _, f := range c.profile.facts {
		keys = append(keys, "fact."+f.ID)
	}
	for _, id := range c.systemIDs {
		for _, suffix := range []string{"presence", "provider", "note", "action", "condition", "existing"} {
			keys = append(keys, "sys."+id+"."+suffix)
		}
		keys = append(keys, "scope."+id)
	}
	return keys
}
