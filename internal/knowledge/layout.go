package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile layout (profile/layout.yaml) is screen order only: the build order
// of the top-level systems and the entry each input is shown under. Nothing
// read by Jev or stored depends on it.

// Homes that are not a system.
const (
	HomeProject  = "project"
	HomePlanning = "planning"
)

type layoutData struct {
	order []string
	homes map[string]string
}

// BuildOrder lists the top-level system ids in the order a building is
// built. With no layout file it is the catalogue's own order.
func (c *Catalog) BuildOrder() []string {
	if len(c.layout.order) > 0 {
		return append([]string(nil), c.layout.order...)
	}
	var out []string
	for _, s := range c.TopSystems() {
		out = append(out, s.ID)
	}
	return out
}

// Home is the entry a profile input is shown under: HomeProject,
// HomePlanning or a top-level system id. An unlisted determinant goes under
// its first system; any other unlisted key has no home ("").
func (c *Catalog) Home(key string) string {
	if h, ok := c.layout.homes[key]; ok {
		return h
	}
	id, ok := strings.CutPrefix(key, "det.")
	if !ok {
		return ""
	}
	d, ok := c.Determinant(id)
	if !ok || len(d.Systems) == 0 {
		return HomeProject
	}
	top, _, _ := strings.Cut(d.Systems[0], ".")
	return top
}

func (c *Catalog) loadLayout(dir string) error {
	p := filepath.Join(dir, "layout.yaml")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	var file struct {
		Version    int                 `yaml:"version"`
		Status     string              `yaml:"status"`
		BuildOrder []string            `yaml:"build_order"`
		Homes      map[string][]string `yaml:"homes"`
	}
	if err := c.unmarshal(p, &file); err != nil {
		return err
	}
	if file.Version != 1 {
		return fmt.Errorf("%s: version %d", p, file.Version)
	}
	top := map[string]bool{}
	for _, s := range c.TopSystems() {
		top[s.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range file.BuildOrder {
		if !top[id] || seen[id] {
			return fmt.Errorf("%s: build_order: %q is not a top-level system or repeats", p, id)
		}
		seen[id] = true
	}
	if len(seen) != len(top) {
		return fmt.Errorf("%s: build_order lists %d of %d top-level systems", p, len(seen), len(top))
	}
	flags := map[string]bool{}
	if sk, ok := c.RFPSkeleton(); ok {
		for _, f := range sk.Flags {
			flags[f.Key] = true
		}
	}
	homes := map[string]string{}
	for home, keys := range file.Homes {
		if home != HomeProject && home != HomePlanning && !top[home] {
			return fmt.Errorf("%s: home %q is not project, planning or a top-level system", p, home)
		}
		for _, key := range keys {
			if _, dup := homes[key]; dup {
				return fmt.Errorf("%s: %s has two homes", p, key)
			}
			if err := c.checkHomeKey(key, flags); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			homes[key] = home
		}
	}
	c.layout = layoutData{order: file.BuildOrder, homes: homes}
	return nil
}

func (c *Catalog) checkHomeKey(key string, flags map[string]bool) error {
	if id, ok := strings.CutPrefix(key, "det."); ok {
		if _, found := c.Determinant(id); !found {
			return fmt.Errorf("%s: no such determinant", key)
		}
		return nil
	}
	if flag, ok := strings.CutPrefix(key, DetailPrefix+"flag."); ok {
		if !flags[flag] {
			return fmt.Errorf("%s: no such RFP flag", key)
		}
		return nil
	}
	return fmt.Errorf("%s: only det.<id> and proj.flag.<key> are placed", key)
}
