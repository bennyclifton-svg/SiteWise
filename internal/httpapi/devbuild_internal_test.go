package httpapi

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDevBuildStatusSpotsChangesSinceStart(t *testing.T) {
	root := t.TempDir()
	write := func(rel string, at time.Time) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Now().Add(-time.Hour)
	before, after := started.Add(-time.Minute), started.Add(time.Minute)
	write("internal/store/store.go", before)
	write("web/src/App.tsx", before)
	write("web/dist/index.html", before.Add(time.Second))

	got := devBuildStatus(root, started)
	if got.ServerStale || got.WebStale {
		t.Fatalf("fresh build reported stale: %+v", got)
	}

	write("knowledge/systems.md", after)
	write("web/src/Profile.tsx", after)
	got = devBuildStatus(root, started)
	if !got.ServerStale || got.ServerChanged != "knowledge/systems.md" {
		t.Fatalf("server change missed: %+v", got)
	}
	if !got.WebStale || got.WebChanged != "web/src/Profile.tsx" {
		t.Fatalf("web change missed: %+v", got)
	}
}
