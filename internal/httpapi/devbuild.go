package httpapi

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Development only: the running binary is a snapshot of the source tree.
// tools/dev.ps1 rebuilds the server on every start but rebuilds the embedded
// web UI only with -Build, so the status says which of the two is behind.

// serverSources are what serve compiles or loads at start.
var serverSources = []string{"cmd", "internal", "knowledge", "data/intake", "data/profile", "go.mod", "go.sum"}

type devBuildJSON struct {
	StartedAt     time.Time `json:"started_at"`
	ServerStale   bool      `json:"server_stale"`
	ServerChanged string    `json:"server_changed,omitempty"`
	WebStale      bool      `json:"web_stale"`
	WebChanged    string    `json:"web_changed,omitempty"`
}

func devBuild(root string, started time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, devBuildStatus(root, started))
	}
}

func devBuildStatus(root string, started time.Time) devBuildJSON {
	out := devBuildJSON{StartedAt: started}
	if path, at := newest(root, serverSources...); at.After(started) {
		out.ServerStale, out.ServerChanged = true, path
	}
	built := modTime(filepath.Join(root, "web", "dist", "index.html"))
	if path, at := newest(root, "web/src", "web/index.html", "web/package.json"); at.After(built) {
		out.WebStale, out.WebChanged = true, path
	}
	return out
}

// newest returns the most recently modified file under the given paths,
// relative to root with forward slashes. Missing paths are skipped.
func newest(root string, rels ...string) (string, time.Time) {
	var best string
	var bestAt time.Time
	for _, rel := range rels {
		_ = filepath.WalkDir(filepath.Join(root, rel), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil || !info.ModTime().After(bestAt) {
				return nil
			}
			bestAt = info.ModTime()
			if r, err := filepath.Rel(root, p); err == nil {
				best = filepath.ToSlash(r)
			}
			return nil
		})
	}
	return best, bestAt
}

// modTime is zero for a missing file, so a never-built UI reads as stale.
func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
