// Package web holds the production SPA build embedded in the binary.
package web

import (
	"embed"
	"io/fs"
)

// dist is filled by `npm --prefix web run build`. The committed .gitkeep lets
// the Go build succeed before the first web build; the server then answers
// 503 for pages until the build exists.
//
//go:embed all:dist
var dist embed.FS

// Dist is the root of the Vite build.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
