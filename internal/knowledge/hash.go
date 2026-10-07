package knowledge

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
)

// Version identifies the bytes actually parsed at Load, not a later disk read.
// Paths are relative and slash-normalised so checkout location does not matter.
func (c *Catalog) Version() string { return c.version }

func (c *Catalog) hashLoaded() string {
	paths := make([]string, 0, len(c.loaded))
	for path := range c.loaded {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		body := c.loaded[path]
		fmt.Fprintf(h, "%d:%s%d:", len(path), path, len(body))
		h.Write(body)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *Catalog) remember(path string, body []byte) error {
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return err
	}
	c.loaded[filepath.ToSlash(rel)] = body
	return nil
}
